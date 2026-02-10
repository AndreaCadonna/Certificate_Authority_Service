#!/usr/bin/env bash
set -euo pipefail

# Behavioral validation script for Certificate Authority Service
# Exercises full lifecycle and error scenarios per SPEC.md §5

PASS=0
FAIL=0
TOTAL=0
STDOUT_FILE=""
STDERR_FILE=""

# Helper: run command, capture output, check exit code
check() {
    local desc="$1"
    local expected_exit="$2"
    shift 2
    TOTAL=$((TOTAL + 1))

    STDOUT_FILE=$(mktemp)
    STDERR_FILE=$(mktemp)

    local actual_exit=0
    "$@" >"$STDOUT_FILE" 2>"$STDERR_FILE" || actual_exit=$?

    if [ "$actual_exit" -eq "$expected_exit" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $desc (exit $actual_exit)"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $desc (expected exit $expected_exit, got $actual_exit)"
        echo "    stdout: $(cat "$STDOUT_FILE")"
        echo "    stderr: $(cat "$STDERR_FILE")"
    fi
}

# Helper: check stdout contains pattern
check_stdout_contains() {
    local desc="$1"
    local pattern="$2"
    TOTAL=$((TOTAL + 1))

    if grep -q "$pattern" "$STDOUT_FILE" 2>/dev/null; then
        PASS=$((PASS + 1))
        echo "  PASS: $desc"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $desc (stdout does not contain '$pattern')"
        echo "    stdout: $(cat "$STDOUT_FILE")"
    fi
}

# Helper: check stderr contains pattern
check_stderr_contains() {
    local desc="$1"
    local pattern="$2"
    TOTAL=$((TOTAL + 1))

    if grep -q "$pattern" "$STDERR_FILE" 2>/dev/null; then
        PASS=$((PASS + 1))
        echo "  PASS: $desc"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $desc (stderr does not contain '$pattern')"
        echo "    stderr: $(cat "$STDERR_FILE")"
    fi
}

# Helper: check file exists
check_file_exists() {
    local desc="$1"
    local path="$2"
    TOTAL=$((TOTAL + 1))

    if [ -f "$path" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $desc"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $desc (file not found: $path)"
    fi
}

# Helper: check file starts with prefix
check_file_starts_with() {
    local desc="$1"
    local path="$2"
    local prefix="$3"
    TOTAL=$((TOTAL + 1))

    if head -1 "$path" 2>/dev/null | grep -q "^$prefix"; then
        PASS=$((PASS + 1))
        echo "  PASS: $desc"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $desc (file does not start with '$prefix')"
    fi
}

# Helper: check file contains pattern
check_file_contains() {
    local desc="$1"
    local path="$2"
    local pattern="$3"
    TOTAL=$((TOTAL + 1))

    if grep -q "$pattern" "$path" 2>/dev/null; then
        PASS=$((PASS + 1))
        echo "  PASS: $desc"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $desc (file does not contain '$pattern')"
    fi
}

# Setup
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WORKDIR=$(mktemp -d)
CA="$WORKDIR/ca"

echo "Building ca binary..."
cd "$SCRIPT_DIR"
go build -o "$CA" .
echo "Binary built at $CA"
echo ""

# Cleanup on exit
cleanup() {
    rm -rf "$WORKDIR"
}
trap cleanup EXIT

# ============================================================================
# SCN-CP-001: Full lifecycle
# ============================================================================
echo "=== SCN-CP-001: Full lifecycle ==="
D="$WORKDIR/cp001"
mkdir -p "$D"

check "Step 1: ca init" 0 \
    "$CA" init --subject "CN=Test Root CA,O=Test Org,C=US" --data-dir "$D"
check_stdout_contains "init output: CA initialized successfully" "CA initialized successfully."

check "Step 2: ca request" 0 \
    "$CA" request --subject "CN=leaf.example.com" --san "DNS:leaf.example.com" \
    --out-key "$WORKDIR/leaf.key" --out-csr "$WORKDIR/leaf.csr"
check_file_exists "leaf.key exists" "$WORKDIR/leaf.key"
check_file_exists "leaf.csr exists" "$WORKDIR/leaf.csr"

check "Step 3: ca sign" 0 \
    "$CA" sign --data-dir "$D" "$WORKDIR/leaf.csr"
check_stdout_contains "sign output: Serial 02" "Serial:      02"

check "Step 4: ca verify (valid)" 0 \
    "$CA" verify --data-dir "$D" "$D/certs/02.pem"
check_stdout_contains "verify output: VALID" "Certificate verification: VALID"

check "Step 5: ca revoke" 0 \
    "$CA" revoke --data-dir "$D" --reason keyCompromise 02
check_stdout_contains "revoke output: success" "Certificate revoked successfully."

check "Step 6: ca crl" 0 \
    "$CA" crl --data-dir "$D"
check_stdout_contains "crl output: 1 revoked" "Revoked certificates: 1"

check "Step 7: ca verify (revoked)" 1 \
    "$CA" verify --data-dir "$D" "$D/certs/02.pem"
check_stdout_contains "verify output: INVALID" "Certificate verification: INVALID"
check_stdout_contains "verify output: REVOKED" "Revocation: REVOKED (reason: keyCompromise"
echo ""

# ============================================================================
# SCN-CP-002: Root CA initialization
# ============================================================================
echo "=== SCN-CP-002: Root CA initialization ==="
D="$WORKDIR/cp002"
mkdir -p "$D"

check "ca init with ecdsa-p256" 0 \
    "$CA" init --subject "CN=Test Root CA,O=Test Org,C=US" --key-algorithm ecdsa-p256 --validity 3650 --data-dir "$D"
check_file_exists "ca.key exists" "$D/ca.key"
check_file_exists "ca.crt exists" "$D/ca.crt"
check_file_starts_with "ca.key is PEM PRIVATE KEY" "$D/ca.key" "-----BEGIN PRIVATE KEY-----"
check_file_starts_with "ca.crt is PEM CERTIFICATE" "$D/ca.crt" "-----BEGIN CERTIFICATE-----"
check_file_contains "serial contains 02" "$D/serial" "^02$"
check_file_contains "crlnumber contains 01" "$D/crlnumber" "^01$"
check_file_contains "index.json is empty array" "$D/index.json" '^\[\]$'
check_stdout_contains "warning about unencrypted key" "Warning: CA private key is stored unencrypted"
echo ""

# ============================================================================
# SCN-CP-004: Serial number increment
# ============================================================================
echo "=== SCN-CP-004: Serial number increment ==="
D="$WORKDIR/cp004"
mkdir -p "$D"

"$CA" init --subject "CN=Serial Test CA" --data-dir "$D" >/dev/null 2>&1

# Generate two CSRs
"$CA" request --subject "CN=first.example.com" --out-key "$WORKDIR/first.key" --out-csr "$WORKDIR/first.csr" >/dev/null 2>&1
"$CA" request --subject "CN=second.example.com" --out-key "$WORKDIR/second.key" --out-csr "$WORKDIR/second.csr" >/dev/null 2>&1

check "sign first CSR (serial 02)" 0 \
    "$CA" sign --data-dir "$D" "$WORKDIR/first.csr"
check_stdout_contains "first cert serial 02" "Serial:      02"
check_file_exists "certs/02.pem exists" "$D/certs/02.pem"

check "sign second CSR (serial 03)" 0 \
    "$CA" sign --data-dir "$D" "$WORKDIR/second.csr"
check_stdout_contains "second cert serial 03" "Serial:      03"
check_file_exists "certs/03.pem exists" "$D/certs/03.pem"
check_file_contains "serial file contains 04" "$D/serial" "^04$"
echo ""

# ============================================================================
# SCN-CP-005: Revocation records timestamp and reason
# ============================================================================
echo "=== SCN-CP-005: Revocation records timestamp and reason ==="
D="$WORKDIR/cp005"
mkdir -p "$D"

"$CA" init --subject "CN=Revoke Test CA" --data-dir "$D" >/dev/null 2>&1
"$CA" request --subject "CN=revoke.test" --out-key "$WORKDIR/rev.key" --out-csr "$WORKDIR/rev.csr" >/dev/null 2>&1
"$CA" sign --data-dir "$D" "$WORKDIR/rev.csr" >/dev/null 2>&1

check "revoke with superseded reason" 0 \
    "$CA" revoke --data-dir "$D" --reason superseded 02
check_stdout_contains "revoke: success" "Certificate revoked successfully."
check_stdout_contains "revoke: serial 02" "Serial: 02"
check_stdout_contains "revoke: reason superseded" "Reason: superseded"
check_file_contains "index: status revoked" "$D/index.json" '"status": "revoked"'
check_file_contains "index: reason superseded" "$D/index.json" '"revocation_reason": "superseded"'
check_file_contains "index: revoked_at non-empty" "$D/index.json" '"revoked_at": "20'
echo ""

# ============================================================================
# SCN-CP-006: CRL contains exactly the revoked certificates
# ============================================================================
echo "=== SCN-CP-006: CRL contains exactly revoked certificates ==="
D="$WORKDIR/cp006"
mkdir -p "$D"

"$CA" init --subject "CN=CRL Test CA" --data-dir "$D" >/dev/null 2>&1

# Issue 3 certs
for i in 1 2 3; do
    "$CA" request --subject "CN=crl-test-$i.example.com" --out-key "$WORKDIR/crl$i.key" --out-csr "$WORKDIR/crl$i.csr" >/dev/null 2>&1
    "$CA" sign --data-dir "$D" "$WORKDIR/crl$i.csr" >/dev/null 2>&1
done

# Revoke 02 and 04 (serials for certs 1 and 3)
"$CA" revoke --data-dir "$D" --reason keyCompromise 02 >/dev/null 2>&1
"$CA" revoke --data-dir "$D" --reason cessationOfOperation 04 >/dev/null 2>&1

check "generate CRL with 48h next-update" 0 \
    "$CA" crl --data-dir "$D" --next-update 48
check_stdout_contains "CRL: 2 revoked" "Revoked certificates: 2"
check_stdout_contains "CRL Number 1" "CRL Number:           1"
check_file_exists "ca.crl exists" "$D/ca.crl"
check_file_starts_with "ca.crl is PEM" "$D/ca.crl" "-----BEGIN X509 CRL-----"
check_file_contains "crlnumber incremented to 02" "$D/crlnumber" "^02$"
echo ""

# ============================================================================
# SCN-CP-007: Verify detects revoked certificate
# ============================================================================
echo "=== SCN-CP-007: Verify detects revoked certificate ==="
D="$WORKDIR/cp007"
mkdir -p "$D"

"$CA" init --subject "CN=Verify Test CA" --data-dir "$D" >/dev/null 2>&1
"$CA" request --subject "CN=verify.test" --out-key "$WORKDIR/v7.key" --out-csr "$WORKDIR/v7.csr" >/dev/null 2>&1
"$CA" sign --data-dir "$D" "$WORKDIR/v7.csr" >/dev/null 2>&1
"$CA" revoke --data-dir "$D" --reason keyCompromise 02 >/dev/null 2>&1
"$CA" crl --data-dir "$D" >/dev/null 2>&1

check "verify revoked cert" 1 \
    "$CA" verify --data-dir "$D" "$D/certs/02.pem"
check_stdout_contains "verify: INVALID" "Certificate verification: INVALID"
check_stdout_contains "verify: Signature OK" "Signature:  OK"
check_stdout_contains "verify: Expiry OK" "Expiry:     OK"
check_stdout_contains "verify: REVOKED" "Revocation: REVOKED (reason: keyCompromise"
echo ""

# ============================================================================
# SCN-CP-008: Verify without CRL
# ============================================================================
echo "=== SCN-CP-008: Verify without CRL ==="
D="$WORKDIR/cp008"
mkdir -p "$D"

"$CA" init --subject "CN=No CRL CA" --data-dir "$D" >/dev/null 2>&1
"$CA" request --subject "CN=nocrl.test" --out-key "$WORKDIR/v8.key" --out-csr "$WORKDIR/v8.csr" >/dev/null 2>&1
"$CA" sign --data-dir "$D" "$WORKDIR/v8.csr" >/dev/null 2>&1

check "verify without CRL" 0 \
    "$CA" verify --data-dir "$D" "$D/certs/02.pem"
check_stdout_contains "verify: VALID" "Certificate verification: VALID"
check_stdout_contains "verify: NOT CHECKED" "Revocation: NOT CHECKED (no CRL available)"
echo ""

# ============================================================================
# SCN-CL-001: ca init with defaults
# ============================================================================
echo "=== SCN-CL-001: ca init defaults (ECDSA P-256) ==="
D="$WORKDIR/cl001"
mkdir -p "$D"

check "init with defaults" 0 \
    "$CA" init --subject "CN=My CA" --data-dir "$D"
check_stdout_contains "algorithm ECDSA P-256" "Algorithm:   ECDSA P-256"
check_stdout_contains "subject CN=My CA" "Subject:     CN=My CA"
echo ""

# ============================================================================
# SCN-CL-002: ca init with RSA 2048
# ============================================================================
echo "=== SCN-CL-002: ca init RSA 2048 ==="
D="$WORKDIR/cl002"
mkdir -p "$D"

check "init RSA 2048" 0 \
    "$CA" init --subject "CN=RSA CA" --key-algorithm rsa-2048 --data-dir "$D"
check_stdout_contains "algorithm RSA 2048" "Algorithm:   RSA 2048"
echo ""

# ============================================================================
# SCN-CL-003: ca sign with custom validity
# ============================================================================
echo "=== SCN-CL-003: ca sign custom validity ==="
D="$WORKDIR/cl003"
mkdir -p "$D"

"$CA" init --subject "CN=Sign Test CA,O=Test,C=US" --data-dir "$D" >/dev/null 2>&1
"$CA" request --subject "CN=server.example.com" --san "DNS:server.example.com" \
    --out-key "$WORKDIR/srv.key" --out-csr "$WORKDIR/srv.csr" >/dev/null 2>&1

check "sign with 180 day validity" 0 \
    "$CA" sign --data-dir "$D" --validity 180 "$WORKDIR/srv.csr"
check_stdout_contains "sign: success" "Certificate issued successfully."
check_stdout_contains "sign: subject" "Subject:     CN=server.example.com"
echo ""

# ============================================================================
# SCN-CL-004: ca revoke default reason
# ============================================================================
echo "=== SCN-CL-004: ca revoke default reason ==="
D="$WORKDIR/cl004"
mkdir -p "$D"

"$CA" init --subject "CN=Default Reason CA" --data-dir "$D" >/dev/null 2>&1
"$CA" request --subject "CN=def.test" --out-key "$WORKDIR/def.key" --out-csr "$WORKDIR/def.csr" >/dev/null 2>&1
"$CA" sign --data-dir "$D" "$WORKDIR/def.csr" >/dev/null 2>&1

check "revoke with default reason" 0 \
    "$CA" revoke --data-dir "$D" 02
check_stdout_contains "reason: unspecified" "Reason: unspecified"
check_file_contains "index: reason unspecified" "$D/index.json" '"revocation_reason": "unspecified"'
echo ""

# ============================================================================
# SCN-CL-005 & SCN-CL-006: ca list
# ============================================================================
echo "=== SCN-CL-005: ca list with certificates ==="
D="$WORKDIR/cl005"
mkdir -p "$D"

"$CA" init --subject "CN=List Test CA" --data-dir "$D" >/dev/null 2>&1
"$CA" request --subject "CN=alpha.com" --out-key "$WORKDIR/alpha.key" --out-csr "$WORKDIR/alpha.csr" >/dev/null 2>&1
"$CA" request --subject "CN=beta.com" --out-key "$WORKDIR/beta.key" --out-csr "$WORKDIR/beta.csr" >/dev/null 2>&1
"$CA" sign --data-dir "$D" "$WORKDIR/alpha.csr" >/dev/null 2>&1
"$CA" sign --data-dir "$D" "$WORKDIR/beta.csr" >/dev/null 2>&1
"$CA" revoke --data-dir "$D" 03 >/dev/null 2>&1

check "list with certs" 0 \
    "$CA" list --data-dir "$D"
check_stdout_contains "list: header" "SERIAL"
check_stdout_contains "list: 02 active" "02.*active"
check_stdout_contains "list: 03 revoked" "03.*revoked"

echo ""
echo "=== SCN-CL-006: ca list with no certificates ==="
D2="$WORKDIR/cl006"
mkdir -p "$D2"
"$CA" init --subject "CN=Empty List CA" --data-dir "$D2" >/dev/null 2>&1

check "list with no certs" 0 \
    "$CA" list --data-dir "$D2"
check_stdout_contains "list: no certs message" "No certificates issued."
echo ""

# ============================================================================
# SCN-CL-007: ca request with SANs
# ============================================================================
echo "=== SCN-CL-007: ca request with SANs ==="

check "request with SANs" 0 \
    "$CA" request --subject "CN=web.example.com,O=Web Corp,C=US" \
    --san "DNS:web.example.com,DNS:www.web.example.com,IP:10.0.0.1" \
    --out-key "$WORKDIR/web.key" --out-csr "$WORKDIR/web.csr"
check_stdout_contains "request: subject" "Subject:   CN=web.example.com,O=Web Corp,C=US"
check_file_exists "web.key exists" "$WORKDIR/web.key"
check_file_exists "web.csr exists" "$WORKDIR/web.csr"
check_file_starts_with "web.key is PEM" "$WORKDIR/web.key" "-----BEGIN PRIVATE KEY-----"
check_file_starts_with "web.csr is PEM" "$WORKDIR/web.csr" "-----BEGIN CERTIFICATE REQUEST-----"
echo ""

# ============================================================================
# SCN-CL-010: Exit code 2 for missing required flag
# ============================================================================
echo "=== SCN-CL-010: Missing required flag ==="

check "init without --subject" 2 \
    "$CA" init
check_stderr_contains "error about missing subject" "subject"
echo ""

# ============================================================================
# SCN-CL-011: Exit code 2 for invalid flag value
# ============================================================================
echo "=== SCN-CL-011: Invalid flag value ==="

check "init with invalid key-algorithm" 2 \
    "$CA" init --subject "CN=Test" --key-algorithm ed25519
check_stderr_contains "error about invalid key algorithm" "invalid key algorithm"
echo ""

# ============================================================================
# SCN-CL-012: Exit code 2 for unknown command
# ============================================================================
echo "=== SCN-CL-012: Unknown command ==="

check "unknown command 'renew'" 2 \
    "$CA" renew
check_stderr_contains "error about unknown command" "unknown command"
echo ""

# ============================================================================
# SCN-CL-013: Exit code 2 for invalid SAN format
# ============================================================================
echo "=== SCN-CL-013: Invalid SAN format ==="

check "request with invalid SAN" 2 \
    "$CA" request --subject "CN=test.com" --san "INVALID:test.com" \
    --out-key "$WORKDIR/t.key" --out-csr "$WORKDIR/t.csr"
check_stderr_contains "error about invalid SAN" "invalid SAN"
echo ""

# ============================================================================
# SCN-ER-002: Commands fail when CA not initialized
# ============================================================================
echo "=== SCN-ER-002: CA not initialized errors ==="
D="$WORKDIR/er002"
mkdir -p "$D"

check "sign without init" 1 \
    "$CA" sign --data-dir "$D" "$WORKDIR/leaf.csr"
check_stderr_contains "sign: not initialized" "CA not initialized"

check "revoke without init" 1 \
    "$CA" revoke --data-dir "$D" 02
check_stderr_contains "revoke: not initialized" "CA not initialized"

check "list without init" 1 \
    "$CA" list --data-dir "$D"
check_stderr_contains "list: not initialized" "CA not initialized"

check "crl without init" 1 \
    "$CA" crl --data-dir "$D"
check_stderr_contains "crl: not initialized" "CA not initialized"

check "verify without init" 1 \
    "$CA" verify --data-dir "$D" "$WORKDIR/leaf.csr"
check_stderr_contains "verify: not initialized" "CA not initialized"
echo ""

# ============================================================================
# SCN-ER-003: Reject revocation of non-existent serial
# ============================================================================
echo "=== SCN-ER-003: Revoke non-existent serial ==="
D="$WORKDIR/er003"
mkdir -p "$D"
"$CA" init --subject "CN=Error Test CA" --data-dir "$D" >/dev/null 2>&1

check "revoke non-existent serial ff" 1 \
    "$CA" revoke --data-dir "$D" ff
check_stderr_contains "error: serial ff not found" "certificate with serial ff not found"
echo ""

# ============================================================================
# SCN-ER-004: Reject double revocation
# ============================================================================
echo "=== SCN-ER-004: Double revocation ==="
D="$WORKDIR/er004"
mkdir -p "$D"
"$CA" init --subject "CN=Double Rev CA" --data-dir "$D" >/dev/null 2>&1
"$CA" request --subject "CN=double.test" --out-key "$WORKDIR/dbl.key" --out-csr "$WORKDIR/dbl.csr" >/dev/null 2>&1
"$CA" sign --data-dir "$D" "$WORKDIR/dbl.csr" >/dev/null 2>&1
"$CA" revoke --data-dir "$D" 02 >/dev/null 2>&1

check "double revoke serial 02" 1 \
    "$CA" revoke --data-dir "$D" 02
check_stderr_contains "error: already revoked" "certificate with serial 02 is already revoked"
echo ""

# ============================================================================
# SCN-ER-005: Reject re-initialization
# ============================================================================
echo "=== SCN-ER-005: Re-initialization ==="
D="$WORKDIR/er005"
mkdir -p "$D"
"$CA" init --subject "CN=First CA" --data-dir "$D" >/dev/null 2>&1

check "re-init fails" 1 \
    "$CA" init --subject "CN=Another CA" --data-dir "$D"
check_stderr_contains "error: already initialized" "CA already initialized"
echo ""

# ============================================================================
# SCN-ER-006: Reject CSR with unsupported key algorithm
# ============================================================================
echo "=== SCN-ER-006: Unsupported key algorithm in CSR ==="
D="$WORKDIR/er006"
mkdir -p "$D"
"$CA" init --subject "CN=Algo Test CA" --data-dir "$D" >/dev/null 2>&1

# Generate a CSR with RSA 1024 using openssl (if available)
if command -v openssl >/dev/null 2>&1; then
    openssl genrsa -out "$WORKDIR/weak.key" 1024 2>/dev/null || true
    if [ -f "$WORKDIR/weak.key" ]; then
        openssl req -new -key "$WORKDIR/weak.key" -subj "/CN=weak" -out "$WORKDIR/weak.csr" 2>/dev/null || true
    fi
    if [ -f "$WORKDIR/weak.csr" ]; then
        check "sign CSR with RSA 1024" 1 \
            "$CA" sign --data-dir "$D" "$WORKDIR/weak.csr"
        check_stderr_contains "error: unsupported key" "unsupported key algorithm"
    else
        echo "  SKIP: Could not generate RSA 1024 CSR with openssl"
    fi
else
    echo "  SKIP: openssl not available for RSA 1024 test"
fi
echo ""

# ============================================================================
# SCN-ER-007: Verify rejects certificate not signed by this CA
# ============================================================================
echo "=== SCN-ER-007: Foreign certificate ==="
D="$WORKDIR/er007"
D2="$WORKDIR/er007b"
mkdir -p "$D" "$D2"

# Create two independent CAs
"$CA" init --subject "CN=CA One" --data-dir "$D" >/dev/null 2>&1
"$CA" init --subject "CN=CA Two" --data-dir "$D2" >/dev/null 2>&1

# Issue cert from CA Two
"$CA" request --subject "CN=foreign.test" --out-key "$WORKDIR/foreign.key" --out-csr "$WORKDIR/foreign.csr" >/dev/null 2>&1
"$CA" sign --data-dir "$D2" "$WORKDIR/foreign.csr" >/dev/null 2>&1

# Verify with CA One (should fail)
check "verify foreign cert" 1 \
    "$CA" verify --data-dir "$D" "$D2/certs/02.pem"
check_stdout_contains "verify: INVALID" "Certificate verification: INVALID"
check_stdout_contains "verify: Signature FAILED" "Signature:  FAILED"
echo ""

# ============================================================================
# SCN-ER-008: Reject non-PEM input
# ============================================================================
echo "=== SCN-ER-008: Non-PEM CSR ==="
D="$WORKDIR/er008"
mkdir -p "$D"
"$CA" init --subject "CN=Parse Test CA" --data-dir "$D" >/dev/null 2>&1

echo "This is not a CSR" > "$WORKDIR/garbage.csr"
check "sign garbage file" 1 \
    "$CA" sign --data-dir "$D" "$WORKDIR/garbage.csr"
check_stderr_contains "error: failed to parse" "failed to parse CSR"
echo ""

# ============================================================================
# SCN-DT-005: PEM encoding for all artifacts
# ============================================================================
echo "=== SCN-DT-005: PEM encoding validation ==="
D="$WORKDIR/dt005"
mkdir -p "$D"

"$CA" init --subject "CN=PEM Test CA" --data-dir "$D" >/dev/null 2>&1
"$CA" request --subject "CN=pem.test" --out-key "$WORKDIR/pemtest.key" --out-csr "$WORKDIR/pemtest.csr" >/dev/null 2>&1
"$CA" sign --data-dir "$D" "$WORKDIR/pemtest.csr" >/dev/null 2>&1
"$CA" revoke --data-dir "$D" 02 >/dev/null 2>&1
"$CA" crl --data-dir "$D" >/dev/null 2>&1

check_file_starts_with "ca.key: PRIVATE KEY" "$D/ca.key" "-----BEGIN PRIVATE KEY-----"
check_file_starts_with "ca.crt: CERTIFICATE" "$D/ca.crt" "-----BEGIN CERTIFICATE-----"
check_file_starts_with "test.key: PRIVATE KEY" "$WORKDIR/pemtest.key" "-----BEGIN PRIVATE KEY-----"
check_file_starts_with "test.csr: CERTIFICATE REQUEST" "$WORKDIR/pemtest.csr" "-----BEGIN CERTIFICATE REQUEST-----"
check_file_starts_with "02.pem: CERTIFICATE" "$D/certs/02.pem" "-----BEGIN CERTIFICATE-----"
check_file_starts_with "ca.crl: X509 CRL" "$D/ca.crl" "-----BEGIN X509 CRL-----"
echo ""

# ============================================================================
# SCN-MK-002: CA key stored unencrypted with warning
# ============================================================================
echo "=== SCN-MK-002: Key stored unencrypted ==="
D="$WORKDIR/mk002"
mkdir -p "$D"

check "init prints warning" 0 \
    "$CA" init --subject "CN=Unprotected CA" --data-dir "$D"
check_stdout_contains "warning: unencrypted key" "Warning: CA private key is stored unencrypted"
check_file_starts_with "key is not encrypted" "$D/ca.key" "-----BEGIN PRIVATE KEY-----"
echo ""

# ============================================================================
# Summary
# ============================================================================
echo ""
echo "========================================"
echo "  TOTAL:  $TOTAL"
echo "  PASSED: $PASS"
echo "  FAILED: $FAIL"
echo "========================================"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
exit 0
