# RESEARCH.md — Certificate Authority Service

## 1. Core Principle

**How a Certificate Authority issues, signs, revokes, and manages X.509 digital certificates through the certificate lifecycle, using a chain-of-trust model with Certificate Revocation List (CRL)-based revocation.**

This experiment exists to explore the mechanics of PKI certificate lifecycle management from the CA's perspective — not to build a production CA, but to understand the exact operations a CA performs when it creates a root keypair, signs end-entity certificates from CSRs, tracks certificate state, and publishes revocation information.

---

## 2. Domain Research

### 2.1 Key Concepts and Terminology

| Term | Definition |
|------|-----------|
| **Certificate Authority (CA)** | An entity that issues and signs digital certificates, vouching for the binding between a public key and an identity. |
| **X.509** | The ITU-T standard (profiled for the Internet by RFC 5280) defining the format of public key certificates, CRLs, and attribute certificates. Current version: v3. |
| **Public Key Infrastructure (PKI)** | The set of roles, policies, hardware, software, and procedures needed to create, manage, distribute, use, store, and revoke digital certificates. |
| **Certificate Signing Request (CSR)** | A PKCS#10 (RFC 2986) message sent by a certificate applicant to a CA, containing the applicant's public key, identity information, and a self-signature proving possession of the corresponding private key. |
| **Chain of Trust** | A hierarchy of certificates: Root CA (self-signed) → Intermediate CA (optional) → End-Entity certificate. Validation walks the chain upward until a trusted root is reached. |
| **Distinguished Name (DN)** | An X.500 name identifying a certificate subject or issuer (e.g., `CN=example.com, O=Example Inc, C=US`). |
| **PEM** | Privacy-Enhanced Mail encoding — Base64-encoded DER data wrapped in `-----BEGIN/END-----` headers. The standard text format for certificates and keys. |
| **DER** | Distinguished Encoding Rules — the canonical binary encoding of ASN.1 structures. Certificates are DER-encoded before PEM-wrapping. |
| **ASN.1** | Abstract Syntax Notation One — the schema language used to define X.509 certificate structures. Each element is encoded as a Tag-Length-Value (TLV) triplet. |
| **Serial Number** | A unique non-negative integer (max 20 bytes) assigned by the CA to each certificate it issues. Must be unique per CA for the CA's lifetime. |
| **Certificate Revocation List (CRL)** | A signed list published by a CA containing the serial numbers of certificates that have been revoked before their expiration date. Defined in RFC 5280 (X.509 CRL v2). |
| **Online Certificate Status Protocol (OCSP)** | A protocol (RFC 6960) for real-time certificate status checking. A client sends a certificate serial number; the responder replies with `good`, `revoked`, or `unknown`. |
| **Basic Constraints** | An X.509v3 extension that indicates whether a certificate is a CA certificate (`cA=TRUE`) and optionally limits path length. Must be marked critical on CA certificates. |
| **Key Usage** | An X.509v3 extension defining permitted cryptographic operations for the key (e.g., `digitalSignature`, `keyCertSign`, `cRLSign`). Always marked critical. |
| **Subject Alternative Name (SAN)** | An X.509v3 extension listing additional identities (DNS names, IP addresses, email addresses, URIs) bound to the certificate's public key. |

### 2.2 How a Certificate Authority Works

#### Certificate Structure (X.509v3)

An X.509v3 certificate contains these fields:

1. **Version**: 3 (indicated as integer 2, zero-indexed)
2. **Serial Number**: Unique per-CA identifier
3. **Signature Algorithm**: Algorithm the CA used to sign (e.g., SHA-256 with RSA)
4. **Issuer**: DN of the signing CA
5. **Validity**: `notBefore` and `notAfter` timestamps (UTC)
6. **Subject**: DN of the certificate holder
7. **Subject Public Key Info**: Public key + algorithm identifier
8. **Extensions** (v3 only): Basic Constraints, Key Usage, SAN, Authority Key Identifier, Subject Key Identifier, CRL Distribution Points, etc.
9. **Signature**: The CA's digital signature over the DER-encoded certificate body

#### CA Signing Operation

1. Applicant generates a key pair and creates a CSR (PKCS#10) containing their public key, DN, and a self-signature.
2. CA receives the CSR and validates the self-signature (proof of key possession).
3. CA constructs a new X.509v3 certificate with the applicant's public key and identity, the CA's issuer DN, a unique serial number, a validity period, and appropriate extensions.
4. CA signs the certificate using the CA's private key.
5. Signed certificate is returned to the applicant.

#### Chain of Trust

```
Root CA (self-signed, offline, cA=TRUE, pathLen=1)
  └── Intermediate CA (signed by Root, cA=TRUE, pathLen=0)
        └── End-Entity Certificate (signed by Intermediate, cA=FALSE)
```

- **Root CA**: Self-signed. Its private key is the most sensitive asset in the entire PKI. In production, stored offline in HSMs.
- **Intermediate CA**: Signed by the root. Performs day-to-day signing. Limits blast radius if compromised.
- **End-Entity**: The certificate used by servers, clients, or users. Cannot sign other certificates.

Validation walks from end-entity upward: verify each certificate's signature against the issuer's public key until a trusted root is reached.

#### Certificate Lifecycle States

```
[Generate Key Pair] → [Create CSR] → [CA Signs → ACTIVE]
                                            │
                                    ┌───────┴───────┐
                                    ▼               ▼
                               [EXPIRED]       [REVOKED]
```

- **Active**: Issued and within validity period, not revoked.
- **Expired**: `notAfter` timestamp has passed. No CA action needed.
- **Revoked**: CA has explicitly invalidated the certificate before expiration. Added to CRL.

#### Revocation via CRL

A CRL is a signed document containing:
- **Version**: v2
- **Issuer**: The CA's DN
- **thisUpdate**: When this CRL was issued
- **nextUpdate**: When the next CRL will be issued
- **Revoked Certificates**: List of (serial number, revocation date, optional reason code)
- **Extensions**: Authority Key Identifier, CRL Number, etc.
- **CA Signature**: Over the entire structure

Clients download the CRL and check whether a certificate's serial number appears in the list. CRLs are cached and refreshed on the `nextUpdate` schedule.

**CRL vs OCSP tradeoffs** (for context — OCSP is out of scope for this experiment):

| Aspect | CRL | OCSP |
|--------|-----|------|
| Timeliness | Periodic (batch) | Near-real-time |
| Bandwidth | Large download, infrequent | Small per-request |
| Offline operation | Yes (cached) | No (network required) |
| Privacy | Good (batch) | Poor (reveals which cert was checked) |
| Industry trend (2025+) | Required | Now optional (Let's Encrypt ended OCSP May 2025) |

### 2.3 Relevant Standards and RFCs

| Standard | Description |
|----------|-------------|
| **RFC 5280** | Internet X.509 PKI — Certificate and CRL Profile. The primary specification for X.509v3 certificates and v2 CRLs. |
| **RFC 2986** | PKCS#10 v1.7 — Certificate Signing Request syntax. |
| **RFC 6960** | OCSP — Online Certificate Status Protocol (out of scope but referenced for completeness). |
| **RFC 8017** | PKCS#1 v2.2 — RSA Cryptography Specifications. |
| **RFC 5958** | Asymmetric Key Packages (PKCS#8 successor). Private key information syntax. |
| **RFC 4158** | Certification Path Building — algorithms for constructing certificate chains. |
| **PKCS#12** | Personal Information Exchange Syntax — bundling private keys and certificates. |

### 2.4 Domain-Specific Gotchas

1. **Serial number collisions**: Every certificate issued by a CA must have a unique serial number (max 20 bytes). Reuse breaks revocation. Use a monotonic counter or cryptographic randomness.

2. **Time handling**: All certificate timestamps must be UTC. Clock skew between CA and clients causes validation failures. The `notBefore` field is often backdated by a few minutes to tolerate minor skew.

3. **Extension criticality**: If a certificate extension is marked `critical`, any system that does not recognize or cannot process it **must** reject the certificate. Mark extensions non-critical unless there is a specific reason.

4. **Basic Constraints on CA certs**: Omitting or mis-setting `BasicConstraints(cA=TRUE)` on a CA certificate will cause chain validation to fail in compliant implementations.

5. **Key Usage alignment**: A CA certificate must have `keyCertSign` in Key Usage to sign certificates, and `cRLSign` to sign CRLs. Missing bits cause validation failures.

6. **Self-signed vs. self-issued**: A self-signed certificate has the same subject and issuer AND is signed by its own key. A self-issued certificate merely has matching subject/issuer names. Root CA certificates are self-signed.

7. **CSR signature verification**: Always verify the CSR's self-signature before issuing. Skipping this removes proof-of-possession.

8. **Weak algorithms**: SHA-1 and MD5 are broken for certificate signing. Use SHA-256 or SHA-384 minimum.

---

## 3. Implementation Approaches

### 3.1 Candidate Languages

#### Go

**Standard library support**: Go's `crypto/x509` package is the most complete standard-library PKI toolkit of any mainstream language.

| Capability | stdlib support | Package |
|------------|---------------|---------|
| X.509v3 certificate generation | Yes | `crypto/x509.CreateCertificate()` |
| CSR creation and parsing | Yes | `crypto/x509.CreateCertificateRequest()` |
| CRL generation | Yes | `crypto/x509.CreateRevocationList()` |
| RSA/ECDSA/Ed25519 key generation | Yes | `crypto/rsa`, `crypto/ecdsa`, `crypto/ed25519` |
| PEM encoding/decoding | Yes | `encoding/pem` |
| ASN.1 handling | Yes | `encoding/asn1` |
| OCSP | Semi-external | `golang.org/x/crypto/ocsp` (not stdlib) |

**Ecosystem maturity**: Excellent. Production CAs are built in Go: [step-ca](https://github.com/smallstep/certificates) (Smallstep), parts of Let's Encrypt's Boulder. Google's internal PKI tooling is in Go.

**Alignment with boring tech**: Strong. Go is stable, well-understood, produces static binaries, has zero-config tooling (`go build`), and requires zero external dependencies for this experiment's core operations.

**Verdict**: **Top candidate.** Zero dependencies for certificates, CSRs, and CRLs. Single static binary. Template-based API is well-documented. ~800-1500 LOC for a complete CA experiment.

#### Python

**Standard library support**: Python's `ssl` and `hashlib` modules can validate certificates but **cannot generate them**. Certificate creation requires an external library.

| Capability | stdlib support | Requires |
|------------|---------------|----------|
| X.509v3 certificate generation | No | `cryptography` library |
| CSR creation and parsing | No | `cryptography` library |
| CRL generation | No | `cryptography` library |
| RSA/ECDSA key generation | No | `cryptography` library |
| PEM encoding/decoding | No | `cryptography` library |
| OCSP | No | `cryptography` library |

**Key library — `cryptography` (pyca/cryptography)**:
- Comprehensive X.509 support: `CertificateBuilder`, `CertificateSigningRequestBuilder`, `CertificateRevocationListBuilder`, `OCSPResponseBuilder`.
- Rust-backed implementation (fast, memory-safe).
- Active development, well-documented, widely used.
- Single dependency covers all CA operations.

**Ecosystem maturity**: Very high. The `cryptography` library is the de facto standard for Python PKI work.

**Alignment with boring tech**: Good. Python is boring tech. The `cryptography` library is mature and well-maintained. However, it is an external dependency with a compiled (Rust) backend.

**Verdict**: **Strong second candidate.** Fastest prototyping speed. Single dependency. ~500-1000 LOC. The tradeoff is the required external dependency.

#### Rust (evaluated, not recommended)

- Standard library has **no cryptographic primitives** by design.
- Requires `rcgen` (certificate generation) + `x509-parser` (parsing) + `ocsp` crate.
- Growing ecosystem but less battle-tested for PKI than Go or Python.
- Steeper learning curve; the experiment would spend "innovation tokens" on the language rather than on CA mechanics.
- Not "boring tech" for most teams.

**Verdict**: Not recommended for this experiment.

#### Java/Kotlin (evaluated, not recommended)

- Standard library (`java.security`) can parse and validate but not easily generate X.509v3 certificates with extensions.
- Requires Bouncy Castle (~10MB dependency), which is comprehensive but heavy.
- JVM startup overhead, verbose code, and tooling complexity conflict with the "small experiment" philosophy.
- Would require 15-25 files to be idiomatic.

**Verdict**: Not recommended for this experiment. Overkill.

### 3.2 Design Patterns

#### Pattern 1: Layered Architecture (CLI / Service / Storage)

**Structure**: Three layers with clear boundaries:
- **CLI Layer**: Parses commands, formats output, handles user interaction.
- **Service Layer**: Encapsulates CA operations (init, sign, revoke, list, generate CRL).
- **Storage Layer**: Persists certificates, keys, CRL, and serial state.

**Tradeoff**: Clean separation of concerns and testability vs. marginal overhead for a small project. The overhead is minimal (3-4 files define the layers) and the benefits are immediate — the CA logic can be exercised without CLI scaffolding, and storage can be swapped without touching core logic.

**Verdict**: **Recommended.** Even for 5-15 files, this pattern pays for itself. It mirrors how production CAs (step-ca, EJBCA) are structured.

#### Pattern 2: Repository Pattern for Certificate Storage

**Structure**: An abstract storage interface with a file-based implementation:
- `Store(cert)` — persist a certificate
- `Get(serial)` — retrieve by serial number
- `List()` — enumerate all certificates
- `Revoke(serial, reason)` — mark a certificate as revoked
- `GetRevoked()` — return all revoked entries (for CRL generation)

**Implementation**: File-based storage with certificates as PEM files (`certs/<serial>.pem`), a serial counter file, and a revocation index file.

**Tradeoff**: Adds one interface + one implementation file. Enables testing with an in-memory implementation and makes the storage mechanism explicit rather than scattered through service code.

**Verdict**: **Recommended.** Persistence is inherently required. The repository abstraction costs ~1 extra file but makes the storage contract explicit.

#### Pattern 3: Pipeline Pattern for CSR Processing (Functional Form)

**Structure**: CSR-to-certificate processing as a sequence of discrete steps:
1. Parse and validate CSR format
2. Verify CSR self-signature (proof of possession)
3. Check policy (key algorithm, key size, extensions)
4. Construct certificate from CSR data + CA policy
5. Sign certificate with CA key
6. Assign serial number and store

**Tradeoff**: Makes the signing workflow explicit and each step independently testable vs. a single monolithic `SignCSR()` function that is simpler but harder to debug and extend.

**Verdict**: **Recommended in lightweight form.** Implement as sequential function calls within the service layer, not as a full pipeline framework. Adds clarity without adding abstraction overhead.

### 3.3 Key Libraries

| Language | Library | Purpose | Notes |
|----------|---------|---------|-------|
| Go | (none needed) | — | Standard library is sufficient |
| Python | `cryptography` >=43.0 | All X.509 operations | Single dependency, Rust-backed |
| Go (optional) | `golang.org/x/crypto/ocsp` | OCSP support | Only if OCSP is in scope (it is not) |

---

## 4. Scope Boundaries

### In Scope

These are the concrete deliverables for the experiment:

1. **Root CA initialization** — Generate an RSA or ECDSA key pair and self-signed root CA certificate with proper extensions (Basic Constraints, Key Usage, Subject Key Identifier).
2. **CSR signing** — Accept a PEM-encoded CSR, validate it, and issue a signed end-entity certificate with configurable validity and extensions.
3. **Certificate storage** — Persist issued certificates as PEM files with a serial number index and metadata (subject, validity, status).
4. **Certificate revocation** — Mark a certificate as revoked by serial number with a reason code. Maintain revocation state.
5. **CRL generation** — Produce a signed X.509 CRL v2 containing all revoked certificates, with `thisUpdate`/`nextUpdate` timestamps.
6. **Certificate listing** — List all issued certificates with their status (active, revoked, expired).
7. **CLI interface** — All operations exposed as CLI commands (e.g., `ca init`, `ca sign`, `ca revoke`, `ca crl`, `ca list`).
8. **Behavioral validation** — Scripts that exercise the full lifecycle: init → sign → verify → revoke → generate CRL → verify revocation.

### Out of Scope

These items are explicitly excluded:

- **OCSP responder** — Requires a running HTTP server and a second protocol. Would introduce a second core principle (network service design). CRL is sufficient to demonstrate revocation.
- **Intermediate CA** — Adds chain-building complexity without teaching new CA mechanics. Root-only is sufficient.
- **Certificate renewal** — Mechanically identical to issuance (sign a new cert with a new serial). Not a distinct operation worth implementing separately.
- **Key escrow or key recovery** — Production concern, not relevant to understanding CA mechanics.
- **HSM integration** — Hardware dependency. Keys stored as PEM files on disk.
- **ACME protocol** — Entire separate protocol (RFC 8555) for automated certificate issuance. Would dominate the experiment.
- **Web UI or REST API** — CLI-only. Adding HTTP would introduce a second core principle (API design).
- **Database storage** — File-based storage is sufficient. SQLite/PostgreSQL would add a dependency without teaching CA concepts.
- **Certificate Transparency (CT) logs** — Production concern requiring external infrastructure.
- **Multi-user / access control** — Single-operator CLI. No authentication or authorization.
- **Cross-signing or bridge CAs** — Advanced PKI topology beyond the experiment's scope.

### Mocked / Simplified

These items are needed for the experiment to function but are simplified because they are not the core principle:

1. **Identity verification** — Real CAs verify the applicant's identity (domain validation, organization validation, extended validation). This experiment trusts the CSR at face value. The CA signs whatever CSR it receives.
2. **Key protection** — Real CAs store keys in HSMs or encrypted vaults. This experiment stores the CA private key as a PEM file on disk (with a file-permission warning).
3. **CRL distribution** — Real CAs publish CRLs to HTTP endpoints. This experiment writes the CRL to a local file. Distribution is manual.
4. **Policy engine** — Real CAs enforce issuance policies (allowed key sizes, name constraints, maximum validity). This experiment applies minimal policy: verify CSR signature, check key type, set a default validity period.
5. **Audit logging** — Real CAs maintain tamper-evident audit logs. This experiment prints operations to stdout.
6. **Clock/time source** — Real CAs use trusted time sources (NTP, hardware clocks). This experiment uses the system clock.

---

## 5. Assumptions

- **A-1**: The user's development machine has Go 1.21+ installed, or the user is willing to install it. (Go is the recommended language; its stdlib eliminates all external dependencies for this experiment.)

- **A-2**: The experiment targets a single-machine, single-operator scenario. There is no multi-user access, network service, or concurrent operation.

- **A-3**: RSA 2048-bit or ECDSA P-256 keys are acceptable for the experiment. Production CAs may require RSA 4096 or P-384, but the mechanics are identical — key size is a parameter, not a principle.

- **A-4**: The file system is the persistence layer. Certificates, keys, the serial counter, and the revocation index are stored as files in a local directory. No database is needed.

- **A-5**: The experiment does not need to interoperate with real-world TLS clients or browsers. Certificates will be validated using the experiment's own verification logic or standard CLI tools (`openssl verify`).

- **A-6**: SHA-256 is the hash algorithm for all signatures. SHA-1 and MD5 are excluded.

- **A-7**: The CA issues end-entity certificates only (no intermediate CAs). The trust hierarchy is: Root CA → End-Entity.

- **A-8**: CRL-based revocation is sufficient to demonstrate revocation mechanics. OCSP is not needed.

- **A-9**: The user is comfortable with a CLI-only interface. No GUI or web interface is expected.

- **A-10**: Behavioral validation scripts (testing that the full lifecycle works end-to-end) replace unit tests, consistent with the workflow philosophy.

- **A-11**: The experiment runs on Windows, macOS, or Linux. Go's standard library and file-based storage are cross-platform.

---

## 6. Open Questions

- **Q-1: Go or Python?**
  Go requires zero external dependencies and produces a single binary. Python with the `cryptography` library is faster to prototype and more concise. **Why it matters**: This decision affects the project structure, build process, and whether a dependency management step is needed. Go is recommended but the user may have a preference.

- **Q-2: RSA or ECDSA for the CA key pair?**
  RSA 2048 is the most widely understood and easiest to debug (`openssl` defaults to RSA). ECDSA P-256 is more modern, produces smaller certificates, and is faster. **Why it matters**: The choice affects key generation code and signature algorithm identifiers throughout the experiment. Both are straightforward to implement.

- **Q-3: Should the experiment support signing CSRs generated by external tools (e.g., `openssl req`)?**
  If yes, the CSR parser must handle real-world PKCS#10 files. If no, the experiment can generate CSRs internally using its own key generation, simplifying the code. **Why it matters**: External CSR support makes the experiment more realistic and testable with standard tools, but adds parsing edge cases.

- **Q-4: What certificate validity period should be the default?**
  Options: 365 days (standard for TLS), 90 days (Let's Encrypt style), or configurable via CLI flag. **Why it matters**: Affects the default in the signing code and how expiration is demonstrated in validation scripts. A configurable default with a sensible fallback (e.g., 365 days) is recommended.

- **Q-5: Should the CA private key be encrypted at rest?**
  Encrypting the CA key with a passphrase adds realism but complicates automation (every sign operation requires a passphrase prompt or environment variable). Storing it unencrypted is simpler and sufficient for an experiment. **Why it matters**: Affects the CA initialization flow and whether the CLI needs passphrase input.

- **Q-6: How detailed should the CRL reason codes be?**
  RFC 5280 defines reason codes: `unspecified`, `keyCompromise`, `cACompromise`, `affiliationChanged`, `superseded`, `cessationOfOperation`, `certificateHold`. The experiment could support all of these or just `unspecified`. **Why it matters**: Supporting multiple reason codes adds a CLI parameter and storage field but teaches more about the CRL specification.

---

## 7. References

### Standards and RFCs
- RFC 5280 — Internet X.509 PKI Certificate and CRL Profile: https://datatracker.ietf.org/doc/html/rfc5280
- RFC 2986 — PKCS#10 Certification Request Syntax v1.7: https://datatracker.ietf.org/doc/html/rfc2986
- RFC 6960 — OCSP: https://datatracker.ietf.org/doc/html/rfc6960
- RFC 8017 — PKCS#1 RSA Cryptography Specifications v2.2: https://tools.ietf.org/html/rfc8017
- RFC 5958 — Asymmetric Key Packages: https://datatracker.ietf.org/doc/html/rfc5958
- RFC 4158 — Certification Path Building: https://datatracker.ietf.org/doc/html/rfc4158

### Language Documentation
- Go `crypto/x509` package: https://pkg.go.dev/crypto/x509
- Python `cryptography` library: https://cryptography.io/en/latest/x509/
- Go CA tutorial: https://shaneutt.com/blog/golang-ca-and-signed-cert-go/

### Production CA References
- Smallstep step-ca: https://github.com/smallstep/certificates
- Let's Encrypt OCSP sunset: https://letsencrypt.org/2024/12/05/ending-ocsp
