# DESIGN.md — Certificate Authority Service

## §1 — Technology Stack

| Choice | Selection | Justification |
|--------|-----------|---------------|
| Language | Go 1.21+ | Go's `crypto/x509` package provides complete X.509v3 certificate creation, CSR parsing, and CRL generation in the standard library — zero external dependencies required. RESEARCH.md §3.1 evaluated Go, Python, Rust, and Java; Go was the top candidate with "zero dependencies for certificates, CSRs, and CRLs." SPEC.md D-1 confirms this choice. Go produces a single static binary (`go build`), requires zero configuration, and naturally supports CLI interfaces via the `flag` package. |
| Build tool | `go build` | Single command produces a static binary. No Makefile, no build configuration. Zero-config per philosophy. |

No external dependencies. Standard library only.

**Packages used from stdlib:**

| Package | Purpose |
|---------|---------|
| `crypto/x509` | Certificate creation (`CreateCertificate`), CSR parsing (`ParseCertificateRequest`, `CreateCertificateRequest`), CRL generation (`CreateRevocationList`) |
| `crypto/ecdsa` | ECDSA P-256 key generation |
| `crypto/rsa` | RSA 2048 key generation |
| `crypto/rand` | Cryptographically secure random number generation (CSPRNG) |
| `crypto/elliptic` | P-256 curve parameter |
| `crypto/sha1` | Subject Key Identifier computation (RFC 5280 method 1) |
| `encoding/pem` | PEM encoding and decoding of all artifacts |
| `encoding/json` | Certificate index (index.json) serialization |
| `encoding/asn1` | ASN.1 marshaling for certificate extensions |
| `math/big` | Serial number handling (big.Int for X.509 serial numbers) |
| `flag` | CLI flag parsing per subcommand |
| `net` | IP address parsing for SAN entries |
| `os` | File I/O, environment variables, exit codes |
| `fmt` | Formatted output to stdout/stderr |
| `time` | Timestamp generation (system clock per CON-DI-014) |
| `strings` | String manipulation for DN/SAN parsing |
| `strconv` | Hex string ↔ integer conversion for serial numbers |
| `path/filepath` | File path construction |

## §2 — File Structure

```
Certificate_Authority_Service/
├── go.mod         ← Go module definition (module ca-service)
├── main.go        ← Entry point: CLI subcommand dispatch, flag parsing, output formatting
├── ca.go          ← Core CA operations: initialize CA, sign CSRs, revoke certificates, list certificates
├── store.go       ← File-based persistence: read/write certs, keys, serial counter, CRL counter, index
├── crl.go         ← CRL generation: build and sign X.509 CRL v2 from revocation data
├── verify.go      ← Certificate verification: signature check, expiry check, revocation check against CRL
├── request.go     ← CSR generation utility: generate key pair and PKCS#10 CSR for testing
├── dn.go          ← Distinguished Name parsing (string ↔ pkix.Name) and SAN string parsing
└── validate.sh    ← Behavioral validation: exercises full lifecycle and error scenarios
```

**9 files total.** All Go source files are in the project root in a single `main` package. The `artifacts/` and `workflow/` directories are pre-existing project infrastructure and not part of the build.

| File | Purpose | Key Contents | Dependencies |
|------|---------|-------------|-------------|
| `go.mod` | Go module definition | Module path `ca-service`, Go version `1.21` | None |
| `main.go` | CLI entry point and subcommand dispatch | `main()`, `resolveDataDir()`, `runInit()`, `runSign()`, `runRevoke()`, `runCRL()`, `runList()`, `runVerify()`, `runRequest()`, `printUsage()` | ca.go, store.go, crl.go, verify.go, request.go, dn.go |
| `ca.go` | Core CA business logic | `InitCA()`, `SignCSR()`, `RevokeCert()`, `ListCerts()`, `InitResult`, `SignResult`, `CertInfo`, `ReasonCodes` map, `ValidReasons` slice, `generateKeyPair()` | store.go, dn.go |
| `store.go` | File-based state persistence | `IndexEntry` struct, `InitDataDir()`, `IsInitialized()`, `SavePrivateKey()`, `LoadPrivateKey()`, `SaveCertPEM()`, `LoadCertificate()`, `ReadCounter()`, `WriteCounter()`, `LoadIndex()`, `SaveIndex()`, `FormatSerial()`, `FormatSerialBig()` | None (leaf dependency) |
| `crl.go` | CRL construction and signing | `GenerateCRL()`, `CRLResult` struct, `ReasonNames` map | store.go |
| `verify.go` | Certificate verification pipeline | `VerifyCert()`, `VerifyResult` struct | store.go |
| `request.go` | CSR and key pair generation | `GenerateCSR()`, `RequestResult` struct | dn.go, store.go (SavePrivateKey reuse) |
| `dn.go` | DN and SAN string parsing | `ParseDN()`, `FormatDN()`, `ParseSANs()`, `AlgoDisplayName()` | None (leaf dependency) |
| `validate.sh` | Behavioral validation script | Shell functions for lifecycle testing, `check()` helper, `check_stdout_contains()`, `check_file_exists()` | Compiled `ca` binary |

## §3 — Component Design

### §3.1 — DN/SAN Parser (`dn.go`)

**Responsibility:** Parse Distinguished Name strings into Go's `pkix.Name` struct and SAN specification strings into typed DNS name and IP address lists.

**Public Interface:**
```go
func ParseDN(dn string) (pkix.Name, error)
// Parses "CN=My Root CA,O=My Org,C=US" into pkix.Name.
// Supported attributes: CN, O, OU, L, ST, C.
// Splits on ',', then splits each part on first '='.
// Returns error if attribute type is unknown or value is empty.

func FormatDN(name pkix.Name) string
// Formats pkix.Name back to "CN=...,O=...,C=..." string.
// Outputs fields in order: CN, O, OU, L, ST, C. Skips empty fields.

func ParseSANs(sanList string) (dnsNames []string, ips []net.IP, err error)
// Parses "DNS:example.com,DNS:www.example.com,IP:10.0.0.1".
// Splits on ',', checks prefix DNS: or IP:.
// Returns error if prefix is not DNS: or IP:, or if IP fails net.ParseIP.

func AlgoDisplayName(keyAlgo string) string
// Maps "ecdsa-p256" → "ECDSA P-256", "rsa-2048" → "RSA 2048".
```

**Contracts Enforced:** CON-BD-001 (subject DN validation), CON-BD-019 (subject and SAN validation for request), CON-BD-021 (SAN format validation)
**Requirements Served:** REQ-CL-001, REQ-CL-007

**Data Flow:**
- Receives: Raw DN string from CLI `--subject` flag, raw SAN string from `--san` flag
- Produces: Typed `pkix.Name` consumed by `ca.go:InitCA` and `request.go:GenerateCSR`; typed SAN lists consumed by `request.go:GenerateCSR`

**Mock Boundary:** None. This is real implementation.

---

### §3.2 — Storage Layer (`store.go`)

**Responsibility:** Provide file-based persistence for all CA state: private keys, certificates, serial/CRL counters, and the certificate index.

**Public Interface:**
```go
type IndexEntry struct {
    Serial           string `json:"serial"`
    Subject          string `json:"subject"`
    NotBefore        string `json:"not_before"`
    NotAfter         string `json:"not_after"`
    Status           string `json:"status"`
    RevokedAt        string `json:"revoked_at"`
    RevocationReason string `json:"revocation_reason"`
}

func InitDataDir(dataDir string) error
// Creates data directory, certs/ subdirectory, serial file ("02"),
// crlnumber file ("01"), and index.json ("[]").

func IsInitialized(dataDir string) bool
// Returns true if both ca.key and ca.crt exist in dataDir.

func SavePrivateKey(path string, key crypto.PrivateKey) error
// Marshals key to PKCS#8 DER, then PEM-encodes with "PRIVATE KEY" header.

func LoadPrivateKey(path string) (crypto.PrivateKey, error)
// Reads PEM file, decodes, parses PKCS#8.

func SaveCertPEM(path string, certDER []byte) error
// PEM-encodes DER bytes with "CERTIFICATE" header and writes to path.

func LoadCertificate(path string) (*x509.Certificate, error)
// Reads PEM file, decodes, parses X.509.

func ReadCounter(path string) (int64, error)
// Reads hex string from file, parses to int64.

func WriteCounter(path string, value int64) error
// Formats int64 as lowercase hex (zero-padded to 2 digits), writes to file.

func LoadIndex(dataDir string) ([]IndexEntry, error)
// Reads and parses index.json from dataDir.

func SaveIndex(dataDir string, entries []IndexEntry) error
// Serializes entries to JSON with indentation, writes to index.json.

func SaveCRLPEM(path string, crlDER []byte) error
// PEM-encodes DER bytes with "X509 CRL" header and writes to path.

func LoadCRL(path string) (*x509.RevocationList, error)
// Reads PEM file, decodes, parses CRL.

func FormatSerial(n int64) string
// Returns lowercase hex string zero-padded to at least 2 digits.
// E.g., 1 → "01", 10 → "0a", 255 → "ff".

func FormatSerialBig(n *big.Int) string
// Same as FormatSerial but for *big.Int values from parsed certificates.
```

**Private helpers (ADR-006 — atomic file replacement):**
```go
func writeFileAtomic(path string, data []byte, perm os.FileMode) error
// 1. Write data to path + ".tmp" with specified permissions.
// 2. Rename path + ".tmp" → path (atomic on POSIX via rename(2)).
// 3. On write failure: remove .tmp file, return error.
// 4. On rename failure: remove .tmp file, return error.
// All public write functions (SaveCertPEM, WriteCounter, SaveIndex,
// SaveCRLPEM, SavePrivateKey) use this internally.

func cleanupTempFiles(paths []string)
// Best-effort removal of .tmp files. Called on stage-sub-phase failure
// to ensure no temporary artifacts are left in the data directory.
// Ignores removal errors.
```

**Contracts Enforced:** CON-DI-001 (PEM encoding), CON-DI-002 (hex format), CON-DI-003 (RFC 3339 timestamps — via IndexEntry schema), CON-DI-004 (atomicity — via writeFileAtomic, ADR-006), CON-DI-005 (index schema), CON-DI-007 (cert–index correspondence — file paths), CON-DI-008 (serial counter consistency), CON-DI-009 (CRL number consistency)
**Requirements Served:** REQ-DT-001, REQ-DT-005, REQ-DT-006, REQ-DT-007

**Data Flow:**
- Receives: Typed Go objects (keys, certificates, index entries, counter values) from ca.go, crl.go, verify.go, request.go
- Produces: Persisted files on disk; loaded Go objects from disk to callers

**Mock Boundary:** None. This is real implementation. File-based storage is the simplest persistence mechanism; no database is mocked.

---

### §3.3 — CA Operations (`ca.go`)

**Responsibility:** Perform the core CA operations: initialize the root CA, sign CSRs into end-entity certificates, revoke certificates, and list issued certificates.

**Public Interface:**
```go
type InitResult struct {
    Subject   string
    Algorithm string
    Serial    string
    NotAfter  time.Time
    CertPath  string
    KeyPath   string
}

type SignResult struct {
    Serial   string
    Subject  string
    NotAfter time.Time
    CertPath string
}

type CertInfo struct {
    Serial   string
    Subject  string
    NotAfter time.Time
    Status   string // "active", "revoked", or "expired"
}

// ReasonCodes maps reason code strings to RFC 5280 CRL reason code integers.
var ReasonCodes = map[string]int{
    "unspecified":          0,
    "keyCompromise":        1,
    "affiliationChanged":  3,
    "superseded":          4,
    "cessationOfOperation": 5,
}

// ValidReasons is the ordered list of accepted reason code strings.
var ValidReasons = []string{
    "unspecified", "keyCompromise", "affiliationChanged",
    "superseded", "cessationOfOperation",
}

func InitCA(dataDir string, subject pkix.Name, keyAlgo string, validityDays int) (*InitResult, error)
// VALIDATE PHASE:
// 1. Check IsInitialized → error if true (CON-BD-003).
// MUTATE PHASE:
// 2. Generate key pair (ECDSA P-256 or RSA 2048) using crypto/rand (CON-SC-002).
// 3. Build X.509v3 template: serial 01, subject=issuer, validity, extensions per CON-DI-011.
// 4. Self-sign with x509.CreateCertificate (CON-INV-006, CON-INV-008).
// 5. Create data directory and certs/ subdirectory.
// STAGE SUB-PHASE (ADR-006):
// 6. Stage ca.key → ca.key.tmp, ca.crt → ca.crt.tmp, serial → serial.tmp,
//    crlnumber → crlnumber.tmp, index.json → index.json.tmp.
//    If any stage write fails: cleanupTempFiles, remove only the certs/
//    subdirectory if it was created by this init attempt, return error.
//    The data directory itself is NOT removed — it may contain unrelated
//    user files if --data-dir pointed to a pre-existing directory.
// COMMIT SUB-PHASE (ADR-006):
// 7. Rename in order: serial, crlnumber, index.json, ca.key, ca.crt.
//    (Support files first; initialization markers ca.key/ca.crt last
//    so IsInitialized remains false until all required state is in place.)
// 8. Return InitResult.

func SignCSR(dataDir string, csrPEM []byte, csrPath string, validityDays int) (*SignResult, error)
// VALIDATE PHASE (all before any mutation — CON-DI-004, CON-SC-003):
// 1. Check IsInitialized → error if false.
// 2. PEM-decode csrPEM → error "failed to parse CSR from <csrPath>" if fails.
// 3. x509.ParseCertificateRequest → error "failed to parse CSR from <csrPath>" if fails.
// 4. csr.CheckSignature() → error "CSR signature verification failed" if fails.
// 5. Check key algorithm (ECDSA P-256 or RSA 2048) → error "unsupported key algorithm" if fails.
// MUTATE PHASE:
// 6. LoadPrivateKey, LoadCertificate (CA key and cert).
// 7. ReadCounter (serial).
// 8. Build X.509v3 template: serial from counter, issuer=CA DN, subject=CSR DN,
//    validity, extensions per CON-DI-012.
// 9. x509.CreateCertificate (CON-INV-005, CON-INV-008).
// STAGE SUB-PHASE (ADR-006 — all writes to .tmp files):
// 10. Stage serial counter (serial + 1) → serial.tmp.
// 11. Stage cert PEM → certs/<serial>.pem.tmp.
// 12. Stage updated index → index.json.tmp.
//     If any stage write fails: cleanupTempFiles, return error.
// COMMIT SUB-PHASE (ADR-006 — rename .tmp → final in defined order):
// 13. Rename serial.tmp → serial.         (prevents serial reuse)
// 14. Rename certs/<serial>.pem.tmp → certs/<serial>.pem.
// 15. Rename index.json.tmp → index.json. (commit point)
// 16. Return SignResult.

func RevokeCert(dataDir string, serialHex string, reason string) error
// VALIDATE PHASE (CON-DI-004):
// 1. Check IsInitialized → error if false.
// 2. LoadIndex.
// 3. Find entry by serial → error "not found" if missing (CON-BD-009).
// 4. Check status → error "already revoked" if revoked (CON-INV-003).
// MUTATE PHASE:
// 5. Set entry status="revoked", revoked_at=time.Now().UTC().Format(RFC3339),
//    revocation_reason=reason.
// 6. SaveIndex (atomic via writeFileAtomic — ADR-006; single file, no staging needed).

func ListCerts(dataDir string) ([]CertInfo, error)
// 1. Check IsInitialized → error if false.
// 2. LoadIndex.
// 3. For each entry, compute display status:
//    - "revoked" if status=="revoked"
//    - "expired" if not_after < now and status!="revoked"
//    - "active" otherwise
// 4. Return []CertInfo.
// Read-only — no state modification (CON-BD-014).
```

**Internal helper:**
```go
func generateKeyPair(keyAlgo string) (crypto.PrivateKey, error)
// Generates ECDSA P-256 or RSA 2048 key pair using crypto/rand.
```

**Contracts Enforced:** CON-INV-001 (serial uniqueness via monotonic counter), CON-INV-002 (serial monotonicity), CON-INV-003 (state irreversibility), CON-INV-004 (init prerequisite check), CON-INV-005 (chain of trust — sign with CA key), CON-INV-006 (self-signed root), CON-INV-008 (SHA-256), CON-INV-009 (index end-entity only), CON-INV-010 (supported key algos), CON-INV-011 (no identity verification), CON-BD-001 through CON-BD-009, CON-SC-002 (CSPRNG), CON-SC-003 (CSR validation gate), CON-DI-004 (atomicity via validate-before-mutate + atomic replace per ADR-003/ADR-006), CON-DI-005 (index schema), CON-DI-007 (cert–index correspondence), CON-DI-010 (X.509v3), CON-DI-011 (root extensions), CON-DI-012 (end-entity extensions)
**Requirements Served:** REQ-CP-001 through REQ-CP-005, REQ-CP-008, REQ-DT-002, REQ-DT-003, REQ-DT-007, REQ-ER-001 through REQ-ER-006, REQ-ER-008, REQ-MK-001, REQ-MK-004

**Data Flow:**
- Receives: Parsed arguments from `main.go`, parsed `pkix.Name` from `dn.go`
- Produces: Persisted CA state via `store.go`, result structs consumed by `main.go` for output

**Mock Boundary:** Identity verification is mocked by omission — `SignCSR` does not verify domain ownership or organization identity. This is deliberate per REQ-MK-001 and CON-INV-011. Policy enforcement is simplified to CSR signature + key algorithm check only (REQ-MK-004).

---

### §3.4 — CSR Generation (`request.go`)

**Responsibility:** Generate key pairs and PKCS#10 CSRs for the `ca request` utility command.

**Public Interface:**
```go
type RequestResult struct {
    Subject   string
    Algorithm string
    KeyPath   string
    CSRPath   string
}

func GenerateCSR(subject pkix.Name, dnsNames []string, ips []net.IP, keyAlgo string, outKeyPath string, outCSRPath string) (*RequestResult, error)
// 1. Generate key pair (ECDSA P-256 or RSA 2048) using crypto/rand (CON-SC-002).
// 2. Build x509.CertificateRequest template: subject, DNSNames, IPAddresses.
// 3. x509.CreateCertificateRequest to produce self-signed CSR.
// 4. SavePrivateKey to outKeyPath (PKCS#8 PEM — CON-DI-001).
// 5. PEM-encode CSR with "CERTIFICATE REQUEST" header, write to outCSRPath.
// 6. Return RequestResult.
```

**Contracts Enforced:** CON-BD-019 (preconditions — validated in main.go), CON-BD-020 (postconditions — PKCS#8 key, valid CSR), CON-SC-002 (CSPRNG), CON-DI-001 (PEM encoding)
**Requirements Served:** REQ-CL-007, REQ-DT-001

**Data Flow:**
- Receives: Parsed `pkix.Name` and SAN lists from `main.go`/`dn.go`, key algorithm string
- Produces: Key file and CSR file on disk; `RequestResult` struct for `main.go` output

**Mock Boundary:** None. This is a standalone utility that generates real cryptographic artifacts. It does not require an initialized CA (CON-BD-019).

---

### §3.5 — CRL Generation (`crl.go`)

**Responsibility:** Generate a signed X.509 CRL v2 containing all revoked certificates from the index.

**Public Interface:**
```go
type CRLResult struct {
    ThisUpdate   time.Time
    NextUpdate   time.Time
    CRLNumber    int64
    RevokedCount int
    CRLPath      string
}

// ReasonNames maps RFC 5280 reason code integers back to display strings.
var ReasonNames = map[int]string{
    0: "unspecified",
    1: "keyCompromise",
    3: "affiliationChanged",
    4: "superseded",
    5: "cessationOfOperation",
}

func GenerateCRL(dataDir string, nextUpdateHours int) (*CRLResult, error)
// 1. Check IsInitialized → error if false (CON-INV-004).
// 2. LoadPrivateKey, LoadCertificate (CA key and cert).
// 3. LoadIndex, filter entries where status=="revoked" (CON-DI-006).
// 4. ReadCounter for crlnumber.
// 5. Build x509.RevocationList template:
//    - RevokedCertificateEntries: serial, revocation time, reason code for each revoked entry.
//    - ThisUpdate: time.Now().UTC()
//    - NextUpdate: thisUpdate + nextUpdateHours hours
//    - Number: current CRL number (big.Int)
//    - Extensions: AuthorityKeyIdentifier matching CA's SKI
// 6. x509.CreateRevocationList (signed with CA key, SHA-256 — CON-INV-005, CON-INV-008).
// STAGE SUB-PHASE (ADR-006):
// 7. Stage CRL PEM → ca.crl.tmp.
// 8. Stage crlnumber counter (crlnumber + 1) → crlnumber.tmp.
//    If any stage write fails: cleanupTempFiles, return error.
// COMMIT SUB-PHASE (ADR-006):
// 9. Rename ca.crl.tmp → ca.crl.       (CRL updated first)
// 10. Rename crlnumber.tmp → crlnumber. (counter advanced)
// 11. Return CRLResult.
```

**Contracts Enforced:** CON-INV-004 (init prerequisite), CON-INV-005 (chain of trust — CRL signed by CA), CON-INV-007 (CRL number monotonicity), CON-INV-008 (SHA-256), CON-BD-010 through CON-BD-012, CON-DI-006 (CRL–index consistency), CON-DI-009 (CRL number consistency), CON-DI-013 (CRL structure), CON-DI-014 (system clock)
**Requirements Served:** REQ-CP-006, REQ-DT-004, REQ-MK-003

**Data Flow:**
- Receives: Data directory path and next-update hours from `main.go`
- Produces: CRL file on disk (`ca.crl`), updated CRL counter; `CRLResult` for `main.go` output

**Mock Boundary:** CRL distribution is mocked — the CRL is written to a local file only. No HTTP endpoint is provided (REQ-MK-003).

---

### §3.6 — Verification (`verify.go`)

**Responsibility:** Verify a certificate's signature against the CA, check its validity period, and check revocation status against the CRL.

**Public Interface:**
```go
type VerifyResult struct {
    Valid     bool
    Subject   string
    Serial    string
    Issuer    string
    NotBefore time.Time
    NotAfter  time.Time
    SigOK     bool
    SigErr    string   // empty if SigOK is true
    ExpiryOK  bool
    RevStatus string   // "OK (not revoked)", "REVOKED (reason: X, date: Y)", or "NOT CHECKED (no CRL available)"
}

func VerifyCert(dataDir string, certPEM []byte, certPath string) (*VerifyResult, error)
// 1. Check IsInitialized → error if false (CON-INV-004).
// 2. PEM-decode and parse the certificate.
// 3. LoadCertificate for CA cert.
// 4. Populate result with subject, serial (FormatSerialBig), issuer, timestamps.
// 5. Check signature: cert.CheckSignatureFrom(caCert).
//    - If fails: result.SigOK=false, result.Valid=false, return early (CON-BD-017).
// 6. Check expiry: time.Now() within [notBefore, notAfter].
//    - result.ExpiryOK = true/false.
// 7. Check revocation: if ca.crl exists, load and parse it.
//    - Iterate CRL RevokedCertificateEntries, compare serial numbers.
//    - If found: result.RevStatus = "REVOKED (reason: X, date: Y)".
//    - If not found: result.RevStatus = "OK (not revoked)".
//    - If no CRL file: result.RevStatus = "NOT CHECKED (no CRL available)".
// 8. result.Valid = sigOK && expiryOK && (not revoked).
//    "NOT CHECKED" does NOT cause failure (CON-BD-017).
// 9. Return result. Read-only — no state modification.
```

**Contracts Enforced:** CON-INV-004 (init prerequisite), CON-BD-016 through CON-BD-018, CON-DI-014 (system clock for expiry check)
**Requirements Served:** REQ-CP-007, REQ-ER-007

**Data Flow:**
- Receives: Certificate PEM bytes and data directory path from `main.go`
- Produces: `VerifyResult` struct consumed by `main.go` for output formatting

**Mock Boundary:** None. This is real verification logic.

---

### §3.7 — CLI Dispatch (`main.go`)

**Responsibility:** Parse command-line arguments, resolve the data directory, dispatch to the appropriate command handler, format output to stdout/stderr, and manage exit codes.

**Public Interface:**
```go
func main()
// Reads os.Args, extracts subcommand, delegates to run* functions.
// Unknown command → stderr error, exit 2.
// No subcommand → print usage, exit 2.
```

**Internal functions:**
```go
func resolveDataDir(flagValue string) string
// Implements CON-BD-022: --data-dir flag > CA_DATA_DIR env > "./ca-data".

func runInit(args []string) int
// Parse flags (--subject, --key-algorithm, --validity, --data-dir).
// Validate: --subject required, --key-algorithm must be ecdsa-p256|rsa-2048,
// --validity must be positive.
// Call ParseDN, then InitCA. Format output. Return exit code.

func runSign(args []string) int
// Parse flags (positional csr-file, --validity, --data-dir).
// Read CSR file bytes. Call SignCSR. Format output. Return exit code.

func runRevoke(args []string) int
// Parse flags (positional serial, --reason, --data-dir).
// Validate: --reason must be in ValidReasons.
// Call RevokeCert. Format output. Return exit code.

func runCRL(args []string) int
// Parse flags (--next-update, --data-dir).
// Call GenerateCRL. Format output. Return exit code.

func runList(args []string) int
// Parse flags (--data-dir).
// Call ListCerts. Format table or "No certificates issued." Return exit code.

func runVerify(args []string) int
// Parse flags (positional cert-file, --data-dir).
// Read cert file bytes. Call VerifyCert. Format verification report. Return exit code.

func runRequest(args []string) int
// Parse flags (--subject, --san, --key-algorithm, --out-key, --out-csr).
// Call ParseDN, ParseSANs (if --san provided), then GenerateCSR.
// Format output. Return exit code.

func printUsage()
// Print available subcommands to stderr.
```

**Contracts Enforced:** CON-BD-022 (data dir resolution), CON-BD-023 (exit codes), CON-SC-001 (never print key material — only paths), all CON-BD precondition contracts (flag validation)
**Requirements Served:** REQ-CL-001 through REQ-CL-009, REQ-MK-002 (warning message), REQ-MK-005 (stdout summaries)

**Data Flow:**
- Receives: Raw `os.Args` from user
- Produces: Formatted output to stdout/stderr; delegates to all other components

**Mock Boundary:** Audit logging is mocked — stdout output serves as the audit log (REQ-MK-005).

---

### §3.8 — Behavioral Validation (`validate.sh`)

**Responsibility:** Exercise the full CA lifecycle and error scenarios, verifying exit codes, stdout content, and file existence.

**Key contents:**
- `check()` — Run a command, verify exit code, capture stdout/stderr
- `check_stdout_contains()` — Verify stdout contains a pattern
- `check_stderr_contains()` — Verify stderr contains a pattern
- `check_file_exists()` — Verify a file exists
- `check_file_contains()` — Verify a file starts with a given string
- Test groups for: full lifecycle (SCN-CP-001), initialization (SCN-CP-002), CSR rejection (SCN-CP-003), serial increment (SCN-CP-004), revocation (SCN-CP-005), CRL generation (SCN-CP-006), verify revoked (SCN-CP-007), verify without CRL (SCN-CP-008), error scenarios (SCN-ER-001 through SCN-ER-008)

**Requirements Served:** All REQ-* (validation), all SCN-* scenarios from SPEC.md §5
**Mock Boundary:** None. This exercises the real system.

## §4 — Data Flow Diagram

### Primary Use Case: Full Certificate Lifecycle

```
User (shell)
 │
 ▼
main.go ─── resolveDataDir() ──► data directory path
 │
 ├── "init" ─► runInit()
 │              │
 │              ├── dn.go:ParseDN(--subject)
 │              │
 │              └── ca.go:InitCA()
 │                   ├── generateKeyPair() ◄── crypto/rand (CSPRNG)
 │                   ├── x509.CreateCertificate() (self-signed root)
 │                   └── store.go: InitDataDir, SavePrivateKey, SaveCertPEM
 │                        └── writes: ca.key, ca.crt, serial("02"),
 │                                    crlnumber("01"), index.json("[]"), certs/
 │
 ├── "request" ─► runRequest()
 │                 │
 │                 ├── dn.go:ParseDN(--subject), ParseSANs(--san)
 │                 │
 │                 └── request.go:GenerateCSR()
 │                      ├── generateKeyPair() ◄── crypto/rand
 │                      ├── x509.CreateCertificateRequest()
 │                      └── writes: <out-key>, <out-csr>
 │
 ├── "sign" ──► runSign()
 │               │
 │               ├── reads: <csr-file>
 │               │
 │               └── ca.go:SignCSR()
 │                    ├── VALIDATE: PEM decode, parse CSR, check signature, check key algo
 │                    ├── store.go: LoadPrivateKey, LoadCertificate, ReadCounter
 │                    ├── x509.CreateCertificate() (signed by CA key)
 │                    └── store.go: SaveCertPEM, WriteCounter, SaveIndex
 │                         └── writes: certs/<serial>.pem, serial, index.json
 │
 ├── "revoke" ► runRevoke()
 │               │
 │               └── ca.go:RevokeCert()
 │                    ├── VALIDATE: IsInitialized, LoadIndex, find serial, check status
 │                    └── store.go: SaveIndex
 │                         └── writes: index.json (status→"revoked")
 │
 ├── "crl" ───► runCRL()
 │               │
 │               └── crl.go:GenerateCRL()
 │                    ├── store.go: LoadPrivateKey, LoadCertificate, LoadIndex, ReadCounter
 │                    ├── filter revoked entries
 │                    ├── x509.CreateRevocationList() (signed by CA key)
 │                    └── store.go: SaveCRLPEM, WriteCounter
 │                         └── writes: ca.crl, crlnumber
 │
 ├── "list" ──► runList()
 │               │
 │               └── ca.go:ListCerts()
 │                    └── store.go: LoadIndex (read-only)
 │
 └── "verify" ► runVerify()
                 │
                 ├── reads: <cert-file>
                 │
                 └── verify.go:VerifyCert()
                      ├── store.go: LoadCertificate (CA cert)
                      ├── cert.CheckSignatureFrom(caCert)
                      ├── time.Now() check against validity
                      └── store.go: LoadCRL (if exists)
                           └── check serial against CRL entries
```

## §5 — Error Handling Strategy

Error handling follows the **validate-before-mutate** pattern (see ADR-003). All validation occurs before any state is modified. If validation fails, the system returns an error and exits without touching persistent state (CON-DI-004).

**Error propagation model:**

1. **Core functions** (`ca.go`, `crl.go`, `verify.go`, `request.go`) return `error` values with the exact error messages specified in SPEC.md §3.4. Error messages include the `Error: ` prefix as required by the spec.
2. **`main.go`** receives these errors, prints them to stderr with `fmt.Fprintln(os.Stderr, err)`, and returns the appropriate exit code.
3. **Usage errors** (exit code 2) are detected and handled entirely in `main.go` during flag parsing, before any core function is called.
4. **Operational errors** (exit code 1) come from core functions and are printed to stderr by `main.go`.

**Error catalog mapping to SPEC.md §3.4:**

| Error | Source Component | Exit Code | Stderr Message |
|-------|-----------------|-----------|----------------|
| REQ-ER-001 | ca.go:SignCSR | 1 | `Error: CSR signature verification failed` |
| REQ-ER-002 | ca.go, crl.go, verify.go | 1 | `Error: CA not initialized. Run 'ca init' first.` |
| REQ-ER-003 | ca.go:RevokeCert | 1 | `Error: certificate with serial <serial> not found` |
| REQ-ER-004 | ca.go:RevokeCert | 1 | `Error: certificate with serial <serial> is already revoked` |
| REQ-ER-005 | ca.go:InitCA | 1 | `Error: CA already initialized at <data-dir>` |
| REQ-ER-006 | ca.go:SignCSR | 1 | `Error: unsupported key algorithm in CSR. Supported: ECDSA P-256, RSA 2048` |
| REQ-ER-007 | verify.go:VerifyCert | 1 | (stdout: `Signature: FAILED`) |
| REQ-ER-008 | ca.go:SignCSR | 1 | `Error: failed to parse CSR from <file>` |
| Usage error | main.go | 2 | Contextual message (e.g., `Error: --subject is required`) |
| Unknown cmd | main.go | 2 | `Error: unknown command "<cmd>"` |

**Special case — `ca verify`:** The verify command always produces structured output to **stdout** (even for INVALID results). The exit code is 1 for INVALID, 0 for VALID. No errors go to stderr unless the command cannot execute at all (e.g., CA not initialized, cert file unreadable).

## §6 — Implementation Plan

| Step | Branch | Description | Files | Requirements | Contracts |
|------|--------|-------------|-------|-------------|-----------|
| 1 | feature/foundation | Create Go module and implement DN/SAN parsing. Create `go.mod` with `module ca-service` and `go 1.21`. Implement `ParseDN(dn string) (pkix.Name, error)`: split on `,`, trim spaces, split each part on first `=`, map attribute types (CN→CommonName, O→Organization, OU→OrganizationalUnit, L→Locality, ST→Province, C→Country), return error for unknown attributes or empty values. Implement `FormatDN(name pkix.Name) string`: output fields in order CN, O, OU, L, ST, C, skip empty, join with `,`. Implement `ParseSANs(sanList string) ([]string, []net.IP, error)`: split on `,`, check `DNS:` or `IP:` prefix, parse IPs with `net.ParseIP`, error on invalid prefix. Implement `AlgoDisplayName(keyAlgo string) string`: map `ecdsa-p256`→`ECDSA P-256`, `rsa-2048`→`RSA 2048`. | go.mod, dn.go | REQ-CL-001 (subject parsing), REQ-CL-007 (SAN parsing) | CON-BD-001 (subject validation), CON-BD-019 (request subject/SAN validation), CON-BD-021 (SAN format validation) |
| 2 | feature/storage | Implement file-based persistence layer. Define `IndexEntry` struct with JSON tags per SPEC.md §4.2.1. Implement `InitDataDir`: create data dir + `certs/` subdir with `os.MkdirAll`, write `serial` file containing `"02"`, write `crlnumber` file containing `"01"`, write `index.json` containing `"[]"`. Implement `IsInitialized`: check `filepath.Join(dataDir, "ca.key")` and `filepath.Join(dataDir, "ca.crt")` exist via `os.Stat`. Implement `SavePrivateKey`: marshal with `x509.MarshalPKCS8PrivateKey`, PEM encode with type `"PRIVATE KEY"`, write with `os.WriteFile` mode `0600`. Implement `LoadPrivateKey`: read file, `pem.Decode`, `x509.ParsePKCS8PrivateKey`. Implement `SaveCertPEM`: PEM encode DER with type `"CERTIFICATE"`, write mode `0644`. Implement `LoadCertificate`: read, PEM decode, `x509.ParseCertificate`. Implement `SaveCRLPEM`: PEM encode DER with type `"X509 CRL"`, write mode `0644`. Implement `LoadCRL`: read, PEM decode, `x509.ParseRevocationList`. Implement `ReadCounter`: `os.ReadFile`, `strings.TrimSpace`, `strconv.ParseInt(s, 16, 64)`. Implement `WriteCounter`: `fmt.Sprintf("%02x", value)`, `os.WriteFile` mode `0644`. Implement `LoadIndex`: `os.ReadFile`, `json.Unmarshal`. Implement `SaveIndex`: `json.MarshalIndent(entries, "", "  ")`, `os.WriteFile` mode `0644`. Implement `FormatSerial(n int64) string`: `fmt.Sprintf("%02x", n)`. Implement `FormatSerialBig(n *big.Int) string`: `strings.ToLower(n.Text(16))`, left-pad with `"0"` if length < 2. | store.go | REQ-DT-001, REQ-DT-005, REQ-DT-006, REQ-DT-007 | CON-DI-001, CON-DI-002, CON-DI-003, CON-DI-005, CON-DI-007, CON-DI-008, CON-DI-009 |
| 3 | feature/ca-operations | Implement core CA operations. Define `InitResult`, `SignResult`, `CertInfo` structs. Define `ReasonCodes` map and `ValidReasons` slice. Implement `generateKeyPair(keyAlgo string) (crypto.PrivateKey, error)`: if `ecdsa-p256`, use `ecdsa.GenerateKey(elliptic.P256(), crypto.Reader)`; if `rsa-2048`, use `rsa.GenerateKey(crypto.Reader, 2048)`. **InitCA**: (a) check `IsInitialized`, return `fmt.Errorf("Error: CA already initialized at %s", dataDir)` if true; (b) call `generateKeyPair`; (c) compute Subject Key Identifier as SHA-1 of marshaled public key bytes; (d) build `x509.Certificate` template with `SerialNumber: big.NewInt(1)`, `Subject: subject`, `Issuer: subject` (implicit for self-signed), `NotBefore: time.Now().UTC()`, `NotAfter: notBefore.Add(validityDays * 24h)`, `KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign`, `BasicConstraintsValid: true`, `IsCA: true`, `SubjectKeyId: ski`; (e) `x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)`; (f) `InitDataDir`, `SavePrivateKey` to `ca.key`, `SaveCertPEM` to `ca.crt`. **SignCSR**: (a) check `IsInitialized`; (b) PEM decode `csrPEM`, error if nil block; (c) `x509.ParseCertificateRequest`, error if fails; (d) `csr.CheckSignature()`, error if fails; (e) check key: for `*ecdsa.PublicKey` verify curve is P-256, for `*rsa.PublicKey` verify `key.N.BitLen() == 2048`, else error; (f) load CA key+cert; (g) read serial counter; (h) compute SKI of CSR's public key; (i) build end-entity `x509.Certificate` template: `SerialNumber: big.NewInt(serial)`, subject from CSR, `NotBefore/NotAfter`, `KeyUsage: x509.KeyUsageDigitalSignature` (add `x509.KeyUsageKeyEncipherment` if RSA), `BasicConstraintsValid: true`, `IsCA: false`, `SubjectKeyId: subjectSKI`, `AuthorityKeyId: caCert.SubjectKeyId`; copy `DNSNames`, `IPAddresses`, `EmailAddresses` from CSR; (j) `x509.CreateCertificate(rand.Reader, template, caCert, csrPubKey, caPrivKey)`; (k) save cert to `certs/<serial>.pem`, increment serial, append index entry, save index. **RevokeCert**: (a) check `IsInitialized`; (b) load index; (c) find entry; (d) check not revoked; (e) update status, revoked_at, reason; (f) save index. **ListCerts**: (a) check `IsInitialized`; (b) load index; (c) compute display status using `time.Now().UTC()` and entry data. | ca.go | REQ-CP-001, REQ-CP-002, REQ-CP-003, REQ-CP-004, REQ-CP-005, REQ-CP-008, REQ-DT-002, REQ-DT-003, REQ-DT-007, REQ-ER-001 through REQ-ER-006, REQ-ER-008, REQ-MK-001, REQ-MK-004 | CON-INV-001 through CON-INV-011, CON-BD-001 through CON-BD-009, CON-SC-002, CON-SC-003, CON-DI-004, CON-DI-005, CON-DI-007, CON-DI-008, CON-DI-010, CON-DI-011, CON-DI-012 |
| 4 | feature/csr-generation | Implement CSR generation utility. Define `RequestResult` struct. Implement `GenerateCSR(subject pkix.Name, dnsNames []string, ips []net.IP, keyAlgo string, outKeyPath string, outCSRPath string) (*RequestResult, error)`: (a) call `generateKeyPair`; (b) build `x509.CertificateRequest` template with Subject, DNSNames, IPAddresses; (c) `x509.CreateCertificateRequest(rand.Reader, template, privateKey)`; (d) `SavePrivateKey` to `outKeyPath`; (e) PEM encode CSR DER with type `"CERTIFICATE REQUEST"`, write to `outCSRPath`; (f) return `RequestResult` with formatted subject and algorithm. | request.go | REQ-CL-007, REQ-DT-001 | CON-BD-020, CON-SC-002, CON-DI-001 |
| 5 | feature/crl-generation | Implement CRL generation. Define `CRLResult` struct. Define `ReasonNames` map (int→string reverse lookup). Implement `GenerateCRL(dataDir string, nextUpdateHours int) (*CRLResult, error)`: (a) check `IsInitialized`; (b) load CA key and cert; (c) load index, filter for `status=="revoked"`; (d) read CRL number counter; (e) build `[]x509.RevocationListEntry`: for each revoked entry, parse `revoked_at` timestamp, look up `ReasonCodes[revocation_reason]` for the ASN.1 reason code, create entry with `SerialNumber: big.NewInt(serial)`, `RevocationTime: revokedAt`, `ReasonCode: reasonCode`; (f) build `x509.RevocationList` template: `ThisUpdate: time.Now().UTC()`, `NextUpdate: thisUpdate.Add(nextUpdateHours * time.Hour)`, `Number: big.NewInt(crlNumber)`, `RevokedCertificateEntries: entries`; add AuthorityKeyIdentifier as `ExtraExtensions` entry using `asn1.Marshal`; (g) `x509.CreateRevocationList(rand.Reader, template, caCert, caPrivKey)`; (h) `SaveCRLPEM` to `ca.crl`; (i) `WriteCounter` for crlnumber+1; (j) return `CRLResult`. | crl.go | REQ-CP-006, REQ-DT-004, REQ-MK-003 | CON-INV-004, CON-INV-005, CON-INV-007, CON-INV-008, CON-BD-010, CON-BD-011, CON-BD-012, CON-DI-006, CON-DI-009, CON-DI-013, CON-DI-014 |
| 6 | feature/verification | Implement certificate verification. Define `VerifyResult` struct. Implement `VerifyCert(dataDir string, certPEM []byte, certPath string) (*VerifyResult, error)`: (a) check `IsInitialized`; (b) PEM decode and `x509.ParseCertificate` — return error if fails; (c) load CA cert; (d) populate result fields: `Subject: FormatDN(cert.Subject)`, `Serial: FormatSerialBig(cert.SerialNumber)`, `Issuer: FormatDN(cert.Issuer)`, `NotBefore`, `NotAfter`; (e) check signature: `cert.CheckSignatureFrom(caCert)` — if error, set `SigOK=false`, `SigErr=err.Error()`, `Valid=false`, return (no further checks needed per CON-BD-017); (f) check expiry: `now := time.Now().UTC()`, `ExpiryOK = !now.Before(cert.NotBefore) && !now.After(cert.NotAfter)`; (g) check revocation: check if `filepath.Join(dataDir, "ca.crl")` exists via `os.Stat`; if exists, `LoadCRL`, iterate `RevokedCertificateEntries`, compare `entry.SerialNumber` with `cert.SerialNumber` using `.Cmp()`, if match set `RevStatus = "REVOKED (reason: <name>, date: <RFC3339>)"` using `ReasonNames` from `crl.go`, if no match set `"OK (not revoked)"`; if CRL doesn't exist, set `"NOT CHECKED (no CRL available)"`; (h) compute `Valid = SigOK && ExpiryOK && !isRevoked`; (i) return result. | verify.go | REQ-CP-007, REQ-ER-007 | CON-INV-004, CON-BD-016, CON-BD-017, CON-BD-018, CON-DI-014 |
| 7 | feature/cli-dispatch | Implement CLI entry point. Implement `main()`: extract `os.Args[1]` as subcommand, switch on `init/sign/revoke/crl/list/verify/request`, call corresponding `run*` function, `os.Exit` with returned code; no args or unknown command → print usage to stderr, exit 2. Implement `resolveDataDir(flagValue string) string`: check flagValue, then `os.Getenv("CA_DATA_DIR")`, then `"./ca-data"`. Implement each `run*` function: create `flag.NewFlagSet(name, flag.ContinueOnError)`, define flags per SPEC.md §4.1, set `fs.SetOutput(io.Discard)` to suppress default error output, parse `fs.Parse(args)`, validate required flags (exit 2 if missing), validate flag values (exit 2 if invalid), call core function, format output per SPEC.md §4.1 templates, return exit code. **runInit**: print success summary + warning per §4.1.1. **runSign**: read CSR file (`os.ReadFile`), call `SignCSR`, print summary per §4.1.2. **runRevoke**: call `RevokeCert`, print summary per §4.1.3. **runCRL**: call `GenerateCRL`, print summary per §4.1.4. **runList**: call `ListCerts`, if empty print `"No certificates issued."`, else print table header `"SERIAL  STATUS   NOT AFTER             SUBJECT"` and rows per §4.1.5. **runVerify**: read cert file, call `VerifyCert`, format verification report per §4.1.6 — first line is `"Certificate verification: VALID"` or `"INVALID"`, then indented fields; return 0 for VALID, 1 for INVALID. **runRequest**: call `ParseDN`, `ParseSANs` (if --san), `GenerateCSR`, print summary per §4.1.7. Implement `printUsage()`: print `"Usage: ca <command> [flags]"` and list commands. | main.go | REQ-CL-001 through REQ-CL-009, REQ-MK-002, REQ-MK-005 | CON-BD-001, CON-BD-003, CON-BD-019, CON-BD-021, CON-BD-022, CON-BD-023, CON-SC-001 |
| 8 | feature/validation | Implement behavioral validation script. Create `validate.sh` with `#!/usr/bin/env bash`, `set -euo pipefail`. Implement helper functions: `check(desc, expected_exit, cmd...)` captures stdout and stderr to temp files, compares exit code; `check_stdout_contains(desc, pattern)` greps stdout; `check_stderr_contains(desc, pattern)` greps stderr; `check_file_exists(desc, path)` checks file existence; `check_file_starts_with(desc, path, prefix)` checks PEM headers. Create temp directory with `mktemp -d`, trap cleanup. Build binary: `go build -o "$TMPDIR/ca" .`. Test groups: (1) SCN-CP-001 full lifecycle (init→request→sign→verify→revoke→crl→verify); (2) SCN-CP-002 root CA structure validation (check all files, PEM headers); (3) SCN-CP-003 tampered CSR rejection (generate valid CSR, corrupt signature bytes with `sed`, attempt sign); (4) SCN-CP-004 serial increment (sign two CSRs, check serials 02/03, check serial file "04"); (5) SCN-ER-001 through SCN-ER-008 error scenarios; (6) SCN-CL-010 through SCN-CL-013 usage error scenarios. Print summary: total, passed, failed. Exit 0 if all pass, 1 if any fail. | validate.sh | All REQ-* (validation coverage) | All CON-* (validation coverage) |

## §7 — Mock Strategy

| Aspect | Real or Mocked | How | Why |
|--------|---------------|-----|-----|
| **X.509 certificate generation** | Real | Go's `crypto/x509.CreateCertificate` | Core principle — this IS the experiment |
| **CSR parsing and validation** | Real | Go's `crypto/x509.ParseCertificateRequest` + `CheckSignature` | Core principle — CSR validation is central to CA mechanics |
| **CRL generation** | Real | Go's `crypto/x509.CreateRevocationList` | Core principle — CRL-based revocation |
| **Key generation** | Real | Go's `crypto/ecdsa.GenerateKey` + `crypto/rsa.GenerateKey` with `crypto/rand.Reader` | Core principle — CSPRNG required |
| **Certificate verification** | Real | Go's `x509.Certificate.CheckSignatureFrom` | Core principle — chain-of-trust verification |
| **File-based storage** | Real | Direct file I/O via `os.ReadFile`/`os.WriteFile` | Simplest persistence; not the core principle but essential |
| **Identity verification** | Mocked by omission | No code exists to verify domain/org identity | Not the core principle (REQ-MK-001) |
| **Key protection** | Mocked | Keys stored as unencrypted PEM files | Not the core principle (REQ-MK-002); warning printed |
| **CRL distribution** | Mocked | CRL written to local file only, no HTTP endpoint | Not the core principle (REQ-MK-003) |
| **Policy engine** | Simplified | Only CSR signature + key algorithm check | Not the core principle (REQ-MK-004) |
| **Audit logging** | Simplified | Stdout summaries serve as audit log | Not the core principle (REQ-MK-005) |
| **Clock/time source** | Simplified | `time.Now()` — system clock directly | Not the core principle (REQ-MK-006, CON-DI-014) |

## §8 — ADR Summary

| ADR | Title | Decision |
|-----|-------|----------|
| [ADR-001](ADRs/ADR-001-go-stdlib-zero-dependencies.md) | Go Standard Library for All X.509 Operations | Use Go with zero external dependencies; reject Python + `cryptography` library |
| [ADR-002](ADRs/ADR-002-custom-cli-dispatch.md) | Custom CLI Subcommand Dispatch | Use Go's `flag` package with manual subcommand switch; reject cobra and urfave/cli |
| [ADR-003](ADRs/ADR-003-validate-before-mutate-atomicity.md) | Validate-Before-Mutate for Operation Atomicity | Perform all validation before any state mutation; reject write-then-rollback |
| [ADR-004](ADRs/ADR-004-manual-dn-string-parsing.md) | Manual Distinguished Name String Parsing | Implement custom DN parser via string splitting; reject LDAP libraries |
| [ADR-005](ADRs/ADR-005-behavioral-validation-over-unit-tests.md) | Behavioral Validation Script Over Go Unit Tests | Use a bash validation script exercising the compiled binary; reject Go `testing` package |

## §9 — Requirement and Contract Coverage

### §9.1 — Requirement Coverage

| Requirement | Component(s) | File(s) | Implementation Step |
|-------------|-------------|---------|-------------------|
| REQ-CP-001 | CA Operations (InitCA) | ca.go | Step 3 |
| REQ-CP-002 | CA Operations (SignCSR validate phase) | ca.go | Step 3 |
| REQ-CP-003 | CA Operations (SignCSR mutate phase) | ca.go | Step 3 |
| REQ-CP-004 | CA Operations + Storage (serial counter) | ca.go, store.go | Steps 2, 3 |
| REQ-CP-005 | CA Operations (RevokeCert) | ca.go | Step 3 |
| REQ-CP-006 | CRL Generation | crl.go | Step 5 |
| REQ-CP-007 | Verification | verify.go | Step 6 |
| REQ-CP-008 | CA Operations (ListCerts) | ca.go | Step 3 |
| REQ-CL-001 | CLI Dispatch (runInit) | main.go | Step 7 |
| REQ-CL-002 | CLI Dispatch (runSign) | main.go | Step 7 |
| REQ-CL-003 | CLI Dispatch (runRevoke) | main.go | Step 7 |
| REQ-CL-004 | CLI Dispatch (runCRL) | main.go | Step 7 |
| REQ-CL-005 | CLI Dispatch (runList) | main.go | Step 7 |
| REQ-CL-006 | CLI Dispatch (runVerify) | main.go | Step 7 |
| REQ-CL-007 | CLI Dispatch (runRequest) + CSR Generation | main.go, request.go | Steps 4, 7 |
| REQ-CL-008 | CLI Dispatch (resolveDataDir) | main.go | Step 7 |
| REQ-CL-009 | CLI Dispatch (all run* functions) | main.go | Step 7 |
| REQ-DT-001 | Storage (PEM encoding) + CSR Generation | store.go, request.go | Steps 2, 4 |
| REQ-DT-002 | CA Operations (InitCA root cert extensions) | ca.go | Step 3 |
| REQ-DT-003 | CA Operations (SignCSR end-entity extensions) | ca.go | Step 3 |
| REQ-DT-004 | CRL Generation (CRL structure) | crl.go | Step 5 |
| REQ-DT-005 | Storage (serial hex format) | store.go | Step 2 |
| REQ-DT-006 | Storage (IndexEntry schema) | store.go | Step 2 |
| REQ-DT-007 | Storage (InitDataDir) + CA Operations | store.go, ca.go | Steps 2, 3 |
| REQ-ER-001 | CA Operations (SignCSR validate phase) | ca.go | Step 3 |
| REQ-ER-002 | CA Operations (IsInitialized checks) | ca.go, crl.go, verify.go | Steps 3, 5, 6 |
| REQ-ER-003 | CA Operations (RevokeCert) | ca.go | Step 3 |
| REQ-ER-004 | CA Operations (RevokeCert) | ca.go | Step 3 |
| REQ-ER-005 | CA Operations (InitCA) | ca.go | Step 3 |
| REQ-ER-006 | CA Operations (SignCSR validate phase) | ca.go | Step 3 |
| REQ-ER-007 | Verification (VerifyCert) | verify.go | Step 6 |
| REQ-ER-008 | CA Operations (SignCSR PEM/parse phase) | ca.go | Step 3 |
| REQ-MK-001 | CA Operations (SignCSR — no identity check) | ca.go | Step 3 |
| REQ-MK-002 | CA Operations (InitCA) + CLI Dispatch (runInit output) | ca.go, main.go | Steps 3, 7 |
| REQ-MK-003 | CRL Generation (local file only) | crl.go | Step 5 |
| REQ-MK-004 | CA Operations (SignCSR — minimal policy) | ca.go | Step 3 |
| REQ-MK-005 | CLI Dispatch (stdout summaries) | main.go | Step 7 |
| REQ-MK-006 | All time-using components | ca.go, crl.go, verify.go | Steps 3, 5, 6 |

### §9.2 — Contract Coverage

| Contract | Component(s) | File(s) | Enforcement Mechanism |
|----------|-------------|---------|----------------------|
| CON-INV-001 | CA Operations | ca.go | Monotonic serial counter read-then-increment in SignCSR; root cert hardcoded to serial 01 |
| CON-INV-002 | CA Operations + Storage | ca.go, store.go | Serial counter file always contains next value; ReadCounter/WriteCounter enforce sequential allocation |
| CON-INV-003 | CA Operations | ca.go | RevokeCert checks `entry.Status != "revoked"` before mutation; returns error if already revoked |
| CON-INV-004 | CA Operations, CRL, Verification | ca.go, crl.go, verify.go | Every mutating/reading function calls `IsInitialized()` first; returns specific error message if false |
| CON-INV-005 | CA Operations + CRL | ca.go, crl.go | SignCSR uses `x509.CreateCertificate` with CA key as signer; GenerateCRL uses `x509.CreateRevocationList` with CA key; AuthorityKeyId set to CA's SKI |
| CON-INV-006 | CA Operations | ca.go | InitCA passes same template as both template and parent to `x509.CreateCertificate`, producing a self-signed certificate |
| CON-INV-007 | CRL Generation + Storage | crl.go, store.go | CRL number read from counter file, used in CRL, counter incremented after CRL written |
| CON-INV-008 | CA Operations + CRL | ca.go, crl.go | Go's `x509.CreateCertificate` and `x509.CreateRevocationList` default to SHA-256 with ECDSA/RSA; `SignatureAlgorithm` field left as default (auto-detected from key type) |
| CON-INV-009 | CA Operations | ca.go | InitCA does not add root cert to index; SignCSR only adds end-entity entries to index |
| CON-INV-010 | CA Operations | ca.go | InitCA only accepts `ecdsa-p256` or `rsa-2048` for generateKeyPair; SignCSR validates CSR public key is P-256 ECDSA or 2048-bit RSA |
| CON-INV-011 | CA Operations | ca.go | SignCSR performs no identity checks — only signature and key algorithm validation per CON-SC-003 |
| CON-BD-001 | CLI Dispatch | main.go | runInit validates --subject required, --key-algorithm in {ecdsa-p256, rsa-2048}, --validity > 0 before calling InitCA |
| CON-BD-002 | CA Operations + CLI | ca.go, main.go | InitCA creates all required files; runInit formats output per spec including warning |
| CON-BD-003 | CA Operations + CLI | ca.go, main.go | InitCA returns error if IsInitialized; runInit returns exit 2 for flag errors |
| CON-BD-004 | CA Operations + CLI | ca.go, main.go | SignCSR validates CSR in order: parse, signature, key algo; runSign validates flags |
| CON-BD-005 | CA Operations + Storage | ca.go, store.go | SignCSR writes cert file, increments serial, appends index entry on success |
| CON-BD-006 | CA Operations + CLI | ca.go, main.go | SignCSR returns specific error messages; runSign returns exit 1 for operational errors, 2 for usage |
| CON-BD-007 | CA Operations + CLI | ca.go, main.go | RevokeCert validates serial exists and is not already revoked; runRevoke validates flags |
| CON-BD-008 | CA Operations | ca.go | RevokeCert sets status, revoked_at (RFC 3339), and revocation_reason on success |
| CON-BD-009 | CA Operations + CLI | ca.go, main.go | RevokeCert returns specific error messages; runRevoke returns appropriate exit codes |
| CON-BD-010 | CRL Generation + CLI | crl.go, main.go | GenerateCRL checks IsInitialized; runCRL validates --next-update > 0 |
| CON-BD-011 | CRL Generation + Storage | crl.go, store.go | GenerateCRL writes CRL with all revoked entries, updates crlnumber; output includes summary |
| CON-BD-012 | CRL Generation + CLI | crl.go, main.go | GenerateCRL returns init error; runCRL returns exit 2 for invalid flags |
| CON-BD-013 | CA Operations + CLI | ca.go, main.go | ListCerts checks IsInitialized; runList validates flags |
| CON-BD-014 | CA Operations + CLI | ca.go, main.go | ListCerts computes display status dynamically; runList formats table; read-only operation |
| CON-BD-015 | CA Operations + CLI | ca.go, main.go | ListCerts returns init error; runList returns exit 1 |
| CON-BD-016 | Verification + CLI | verify.go, main.go | VerifyCert checks IsInitialized; runVerify validates cert-file argument |
| CON-BD-017 | Verification + CLI | verify.go, main.go | VerifyCert performs 3 checks in order (sig, expiry, revocation); runVerify formats report; early return if sig fails |
| CON-BD-018 | Verification + CLI | verify.go, main.go | VerifyCert returns init error; sig failure produces "FAILED" in output with exit 1 |
| CON-BD-019 | CLI Dispatch | main.go | runRequest validates --subject, --out-key, --out-csr required; --key-algorithm valid; no CA required |
| CON-BD-020 | CSR Generation + CLI | request.go, main.go | GenerateCSR writes PKCS#8 key and valid CSR; runRequest formats output |
| CON-BD-021 | CLI Dispatch | main.go | runRequest validates --san format via ParseSANs; returns exit 2 if invalid |
| CON-BD-022 | CLI Dispatch | main.go | resolveDataDir implements precedence: --data-dir flag > CA_DATA_DIR env > "./ca-data" |
| CON-BD-023 | CLI Dispatch | main.go | All run* functions return int (0, 1, or 2); main() calls os.Exit with returned value |
| CON-SC-001 | CLI Dispatch + All components | main.go, ca.go, request.go | Output formatting in main.go only prints file paths for keys, never key content |
| CON-SC-002 | CA Operations + CSR Generation | ca.go, request.go | generateKeyPair uses crypto/rand.Reader (OS CSPRNG) |
| CON-SC-003 | CA Operations | ca.go | SignCSR validate phase: (1) CheckSignature, (2) key algo check — both must pass before any file write or counter increment |
| CON-DI-001 | Storage + CSR Generation | store.go, request.go | SavePrivateKey uses "PRIVATE KEY" header; SaveCertPEM uses "CERTIFICATE"; SaveCRLPEM uses "X509 CRL"; request.go uses "CERTIFICATE REQUEST" |
| CON-DI-002 | Storage | store.go | FormatSerial/FormatSerialBig produce lowercase hex zero-padded to 2 digits; WriteCounter uses same format |
| CON-DI-003 | CA Operations | ca.go | IndexEntry timestamps formatted with `time.Time.UTC().Format(time.RFC3339)` which produces `Z`-suffix |
| CON-DI-004 | CA Operations + CRL + Storage | ca.go, crl.go, store.go | Validate-before-mutate pattern (ADR-003) + atomic file replacement with stage-then-commit protocol (ADR-006): all checks complete before any writes; all mutate-phase writes use writeFileAtomic; multi-file mutations stage to .tmp then rename in defined commit order |
| CON-DI-005 | Storage | store.go | IndexEntry struct defines exactly 7 fields with correct JSON tags; SaveIndex serializes complete entries |
| CON-DI-006 | CRL Generation | crl.go | GenerateCRL filters index for `status=="revoked"`, builds CRL entries from exactly that set |
| CON-DI-007 | CA Operations + Storage | ca.go, store.go | SignCSR stages cert file AND index entry to .tmp files, then commits via rename in sequence (ADR-006); staging failure leaves no artifacts |
| CON-DI-008 | CA Operations + Storage | ca.go, store.go | InitCA writes serial "02" after assigning "01" to root; SignCSR increments after each issuance |
| CON-DI-009 | CRL Generation + Storage | crl.go, store.go | GenerateCRL increments crlnumber after CRL written; init writes "01" |
| CON-DI-010 | CA Operations | ca.go | Go's x509.Certificate template version defaults to v3 when extensions are present |
| CON-DI-011 | CA Operations | ca.go | InitCA template: IsCA=true, BasicConstraintsValid=true, KeyUsage=CertSign+CRLSign, SubjectKeyId=SHA1(pubkey) |
| CON-DI-012 | CA Operations | ca.go | SignCSR template: IsCA=false, BasicConstraintsValid=true, KeyUsage=DigitalSignature(+KeyEncipherment if RSA), AuthorityKeyId=CA.SKI, SubjectKeyId=SHA1(subjectPubkey), SANs from CSR |
| CON-DI-013 | CRL Generation | crl.go | RevocationList template: issuer from CA cert (implicit), ThisUpdate, NextUpdate, Number, entries with serial+time+reason, AuthorityKeyId, signed by CA |
| CON-DI-014 | All time-using components | ca.go, crl.go, verify.go | All use `time.Now().UTC()` for timestamps; no external time source |
