# CONTRACTS.md — Certificate Authority Service

## 1. Overview

This document codifies the enforceable invariants and hard constraints derived from SPEC.md for the Certificate Authority Service experiment. These contracts are implementation-agnostic — they describe what must always be true, never how it is achieved. An implementation agent must obey every contract in this document. A validation agent must verify every contract in this document.

**Core Principle:** How a Certificate Authority issues, signs, revokes, and manages X.509 digital certificates through the certificate lifecycle, using a chain-of-trust model with CRL-based revocation.

---

## 2. System-Wide Invariants (CON-INV)

These properties must hold across the entire system at all times, not just within a single operation.

---

### CON-INV-001: Serial Number Uniqueness

Every certificate issued by the CA (including the root CA certificate) SHALL have a unique serial number. No two certificates SHALL ever share a serial number.

**Traces to:** REQ-CP-004

---

### CON-INV-002: Serial Number Monotonicity

Serial numbers SHALL be assigned in strictly monotonically increasing order. The root CA certificate receives serial `01`. Each subsequent certificate receives the next value from the serial counter. The serial counter file SHALL always contain the next serial number to be assigned, never one that has already been used.

**Traces to:** REQ-CP-004, REQ-DT-005

---

### CON-INV-003: Certificate State Irreversibility

A certificate's status transition is one-directional: `active` → `revoked`. Once a certificate's status is set to `revoked`, it SHALL never return to `active`. No other status values are permitted in the persistent index. Attempting to revoke an already-revoked certificate is an error.

**Traces to:** REQ-CP-005, REQ-ER-004

---

### CON-INV-004: CA Initialization Prerequisite

The CA private key (`ca.key`) and CA certificate (`ca.crt`) SHALL exist in the data directory before any operation that reads CA state, signs artifacts, verifies certificates, or modifies the certificate index. Only `ca init` and `ca request` may execute without an initialized CA. All other commands SHALL fail with an error if the CA is not initialized.

**Traces to:** REQ-ER-002

---

### CON-INV-005: Chain of Trust Integrity

Every issued end-entity certificate SHALL be signed by the CA's private key. Every CRL SHALL be signed by the CA's private key. The Authority Key Identifier extension on every issued end-entity certificate and every CRL SHALL match the CA certificate's Subject Key Identifier value. This chain-of-trust linkage SHALL be verifiable.

**Traces to:** REQ-CP-003, REQ-CP-006, REQ-DT-003, REQ-DT-004

---

### CON-INV-006: Root CA Self-Signed Identity

The root CA certificate's issuer Distinguished Name SHALL equal its subject Distinguished Name. The root CA certificate SHALL be verifiable using its own public key (self-signed).

**Traces to:** REQ-CP-001

---

### CON-INV-007: CRL Number Monotonicity

CRL numbers SHALL be monotonically increasing starting from `1`. Each newly generated CRL SHALL have a CRL number strictly greater than every previously generated CRL. The CRL number counter file SHALL always contain the next CRL number to be assigned.

**Traces to:** REQ-CP-006

---

### CON-INV-008: SHA-256 Signature Algorithm

All cryptographic signatures produced by the system — on the root CA certificate, on issued end-entity certificates, and on CRLs — SHALL use SHA-256 as the hash algorithm.

**Traces to:** REQ-CP-001, REQ-CP-003, REQ-CP-006

---

### CON-INV-009: Index Contains Only End-Entity Certificates

The certificate index (`index.json`) SHALL contain entries exclusively for issued end-entity certificates. The root CA certificate SHALL NOT appear in the index.

**Traces to:** REQ-DT-006

---

### CON-INV-010: Supported Key Algorithms Only

The system SHALL only generate or accept keys using ECDSA P-256 or RSA 2048. The CA's own key pair SHALL use one of these two algorithms. CSRs containing any other key algorithm SHALL be rejected.

**Traces to:** REQ-CP-001, REQ-ER-006, REQ-MK-004

---

### CON-INV-011: No Identity Verification

The system SHALL NOT perform domain validation, organization validation, extended validation, or any other form of identity verification on CSRs. A CSR that passes self-signature verification and key algorithm validation SHALL be accepted for signing without additional checks.

**Traces to:** REQ-MK-001, REQ-MK-004

---

## 3. Boundary Contracts (CON-BD)

Per-interface contracts defining preconditions, postconditions, and error conditions for each CLI command and cross-cutting concerns.

---

### 3.1 `ca init`

#### CON-BD-001: `ca init` Preconditions

- The `--subject` flag SHALL be provided and contain a non-empty Distinguished Name string.
- The `--key-algorithm` flag, if provided, SHALL be exactly `ecdsa-p256` or `rsa-2048`.
- The `--validity` flag, if provided, SHALL be a positive integer (number of days).
- The data directory SHALL NOT already contain a `ca.key` or `ca.crt` file.

**Traces to:** REQ-CL-001, REQ-ER-005

---

#### CON-BD-002: `ca init` Postconditions

On success (exit code `0`), the following SHALL all be true:

- `ca.key` SHALL exist in the data directory containing a PEM-encoded PKCS#8 private key (ECDSA P-256 or RSA 2048 as specified).
- `ca.crt` SHALL exist in the data directory containing a PEM-encoded self-signed X.509v3 root CA certificate with serial `01`.
- `serial` SHALL exist in the data directory containing `02`.
- `crlnumber` SHALL exist in the data directory containing `01`.
- `index.json` SHALL exist in the data directory containing `[]`.
- A `certs/` subdirectory SHALL exist in the data directory.
- Stdout SHALL contain a summary including subject, algorithm, serial, not-after date, and file paths.
- Stdout SHALL contain: `Warning: CA private key is stored unencrypted at <data-dir>/ca.key. Protect this file.`

**Traces to:** REQ-CP-001, REQ-DT-007, REQ-MK-002, REQ-MK-005

---

#### CON-BD-003: `ca init` Error Conditions

- If the data directory already contains `ca.key` or `ca.crt`: print `Error: CA already initialized at <data-dir>` to stderr, exit code `1`. Existing files SHALL NOT be overwritten or modified.
- If `--subject` is missing: exit code `2`.
- If `--key-algorithm` is provided but not `ecdsa-p256` or `rsa-2048`: exit code `2`.
- If `--validity` is provided but not a positive integer: exit code `2`.

**Traces to:** REQ-ER-005, REQ-CL-001, REQ-CL-009

---

### 3.2 `ca sign`

#### CON-BD-004: `ca sign` Preconditions

- The CA SHALL be initialized (CON-INV-004).
- The `<csr-file>` positional argument SHALL be provided and point to an existing file.
- The file SHALL contain a valid PEM-encoded PKCS#10 CSR (parseable).
- The CSR's self-signature SHALL be valid.
- The CSR's public key algorithm SHALL be ECDSA P-256 or RSA 2048.
- The `--validity` flag, if provided, SHALL be a positive integer (number of days).
- No identity verification is performed on the CSR (CON-INV-011).

**Traces to:** REQ-CL-002, REQ-CP-002, REQ-ER-001, REQ-ER-006, REQ-ER-008, REQ-MK-004

---

#### CON-BD-005: `ca sign` Postconditions

On success (exit code `0`), the following SHALL all be true:

- A new PEM-encoded X.509v3 certificate file SHALL exist at `<data-dir>/certs/<serial>.pem`, where `<serial>` is the hex serial number assigned.
- The certificate SHALL have: issuer equal to the CA's DN, subject equal to the CSR's subject, the CSR's public key, and extensions per CON-DI-012.
- The serial counter file SHALL have been incremented by 1.
- `index.json` SHALL contain a new entry with the assigned serial, the CSR's subject, validity timestamps, status `active`, and empty revocation fields.
- Stdout SHALL contain a summary including serial, subject, not-after date, and certificate file path.

**Traces to:** REQ-CP-003, REQ-CP-004, REQ-DT-003, REQ-DT-006, REQ-MK-005

---

#### CON-BD-006: `ca sign` Error Conditions

- If CA not initialized: `Error: CA not initialized. Run 'ca init' first.` to stderr, exit code `1`. No state changes.
- If the file cannot be parsed as a PEM-encoded CSR: `Error: failed to parse CSR from <file>` to stderr, exit code `1`. No state changes.
- If the CSR self-signature is invalid: `Error: CSR signature verification failed` to stderr, exit code `1`. No state changes.
- If the CSR key algorithm is not ECDSA P-256 or RSA 2048: `Error: unsupported key algorithm in CSR. Supported: ECDSA P-256, RSA 2048` to stderr, exit code `1`. No state changes.
- If `<csr-file>` positional argument is missing: exit code `2`.

**Traces to:** REQ-ER-001, REQ-ER-002, REQ-ER-006, REQ-ER-008, REQ-CL-009

---

### 3.3 `ca revoke`

#### CON-BD-007: `ca revoke` Preconditions

- The CA SHALL be initialized (CON-INV-004).
- The `<serial>` positional argument SHALL be provided as a hexadecimal string.
- A certificate with the given serial SHALL exist in the certificate index.
- The certificate SHALL NOT already have status `revoked`.
- The `--reason` flag, if provided, SHALL be one of: `unspecified`, `keyCompromise`, `affiliationChanged`, `superseded`, `cessationOfOperation`.

**Traces to:** REQ-CL-003, REQ-CP-005, REQ-ER-003, REQ-ER-004

---

#### CON-BD-008: `ca revoke` Postconditions

On success (exit code `0`), the following SHALL all be true:

- The index entry for the given serial SHALL have status `revoked`.
- The index entry SHALL have a non-empty `revoked_at` field containing an RFC 3339 UTC timestamp.
- The index entry SHALL have `revocation_reason` set to the specified reason code (default `unspecified`).
- Stdout SHALL contain a summary including the serial number and reason code.

**Traces to:** REQ-CP-005, REQ-DT-006, REQ-MK-005

---

#### CON-BD-009: `ca revoke` Error Conditions

- If CA not initialized: `Error: CA not initialized. Run 'ca init' first.` to stderr, exit code `1`. No state changes.
- If the serial is not found in the index: `Error: certificate with serial <serial> not found` to stderr, exit code `1`. No state changes.
- If the serial is already revoked: `Error: certificate with serial <serial> is already revoked` to stderr, exit code `1`. No state changes.
- If `<serial>` positional argument is missing: exit code `2`.
- If `--reason` is provided but not a valid reason code: exit code `2`.

**Traces to:** REQ-ER-002, REQ-ER-003, REQ-ER-004, REQ-CL-009

---

### 3.4 `ca crl`

#### CON-BD-010: `ca crl` Preconditions

- The CA SHALL be initialized (CON-INV-004).
- The `--next-update` flag, if provided, SHALL be a positive integer (number of hours).

**Traces to:** REQ-CL-004, REQ-ER-002

---

#### CON-BD-011: `ca crl` Postconditions

On success (exit code `0`), the following SHALL all be true:

- `ca.crl` SHALL exist in the data directory as a PEM-encoded X.509 CRL v2 with structure per CON-DI-013.
- The CRL SHALL contain revocation entries for all certificates with status `revoked` in `index.json`, and no others (CON-DI-006).
- `thisUpdate` SHALL be the current time (UTC). `nextUpdate` SHALL be `thisUpdate` plus the configured hours (default 24).
- The CRL number counter file SHALL have been incremented by 1.
- The CRL is written to a local file only. No HTTP distribution endpoint is provided.
- Stdout SHALL contain a summary including thisUpdate, nextUpdate, CRL number, revoked certificate count, and CRL file path.

**Traces to:** REQ-CP-006, REQ-DT-004, REQ-MK-003, REQ-MK-005

---

#### CON-BD-012: `ca crl` Error Conditions

- If CA not initialized: `Error: CA not initialized. Run 'ca init' first.` to stderr, exit code `1`.
- If `--next-update` is provided but not a positive integer: exit code `2`.

**Traces to:** REQ-ER-002, REQ-CL-009

---

### 3.5 `ca list`

#### CON-BD-013: `ca list` Preconditions

- The CA SHALL be initialized (CON-INV-004).

**Traces to:** REQ-CL-005, REQ-ER-002

---

#### CON-BD-014: `ca list` Postconditions

On success (exit code `0`), the following SHALL all be true:

- If end-entity certificates exist in the index: stdout SHALL contain a table with columns SERIAL, STATUS, NOT AFTER, SUBJECT for each certificate.
- Display status SHALL be computed dynamically as:
  - `revoked` if the certificate's index status is `revoked`.
  - `expired` if the certificate's `notAfter` is before the current time and the certificate is not revoked.
  - `active` if the certificate's `notAfter` is at or after the current time and the certificate is not revoked.
- If no end-entity certificates exist in the index: stdout SHALL contain `No certificates issued.`
- This command SHALL NOT modify any persistent state.

**Traces to:** REQ-CP-008, REQ-CL-005

---

#### CON-BD-015: `ca list` Error Conditions

- If CA not initialized: `Error: CA not initialized. Run 'ca init' first.` to stderr, exit code `1`.

**Traces to:** REQ-ER-002

---

### 3.6 `ca verify`

#### CON-BD-016: `ca verify` Preconditions

- The CA SHALL be initialized (CON-INV-004).
- The `<cert-file>` positional argument SHALL be provided and point to an existing PEM-encoded certificate file.

**Traces to:** REQ-CL-006, REQ-ER-002

---

#### CON-BD-017: `ca verify` Postconditions

The command SHALL perform three checks in order:

1. **Signature validation**: verify the certificate's signature against the CA certificate's public key. Report `Signature: OK` or `Signature: FAILED`.
2. **Validity period check**: verify the current time is within the certificate's `notBefore`–`notAfter` range. Report `Expiry: OK` or `Expiry: EXPIRED`.
3. **Revocation check**: if `ca.crl` exists in the data directory, check the certificate's serial number against the CRL. Report `Revocation: OK (not revoked)` or `Revocation: REVOKED (reason: <reason>, date: <timestamp>)`. If no CRL exists, report `Revocation: NOT CHECKED (no CRL available)`.

- If all checks pass: stdout includes `Certificate verification: VALID`, exit code `0`.
- If any check fails: stdout includes `Certificate verification: INVALID`, exit code `1`.
- If signature fails, subsequent checks need not be reported.
- If no CRL is available, the revocation check does not cause failure.
- This command SHALL NOT modify any persistent state.

**Traces to:** REQ-CP-007, REQ-CL-006

---

#### CON-BD-018: `ca verify` Error Conditions

- If CA not initialized: `Error: CA not initialized. Run 'ca init' first.` to stderr, exit code `1`.
- If the certificate was not signed by this CA: output includes `Signature: FAILED`, exit code `1`.
- If `<cert-file>` positional argument is missing: exit code `2`.

**Traces to:** REQ-ER-002, REQ-ER-007, REQ-CL-009

---

### 3.7 `ca request`

#### CON-BD-019: `ca request` Preconditions

- The `--subject` flag SHALL be provided and contain a non-empty Distinguished Name string.
- The `--out-key` flag SHALL be provided with a file path.
- The `--out-csr` flag SHALL be provided with a file path.
- The `--key-algorithm` flag, if provided, SHALL be exactly `ecdsa-p256` or `rsa-2048`.
- The `--san` flag, if provided, SHALL contain a comma-separated list of values in the format `DNS:<name>` or `IP:<address>`.
- This command does NOT require an initialized CA.

**Traces to:** REQ-CL-007

---

#### CON-BD-020: `ca request` Postconditions

On success (exit code `0`), the following SHALL all be true:

- The file at `--out-key` SHALL contain a PEM-encoded PKCS#8 private key (`-----BEGIN PRIVATE KEY-----`).
- The file at `--out-csr` SHALL contain a PEM-encoded PKCS#10 CSR (`-----BEGIN CERTIFICATE REQUEST-----`).
- The CSR SHALL have a valid self-signature (verifiable with the CSR's own public key).
- The CSR's subject SHALL match the `--subject` value.
- If `--san` was provided, the CSR SHALL contain the specified Subject Alternative Names.
- Stdout SHALL contain a summary including subject, algorithm, key path, and CSR path.

**Traces to:** REQ-CL-007, REQ-DT-001, REQ-MK-005

---

#### CON-BD-021: `ca request` Error Conditions

- If `--subject`, `--out-key`, or `--out-csr` is missing: exit code `2`.
- If `--key-algorithm` is provided but not `ecdsa-p256` or `rsa-2048`: exit code `2`.
- If `--san` is provided but contains values not in the format `DNS:<name>` or `IP:<address>`: exit code `2`.

**Traces to:** REQ-CL-007, REQ-CL-009

---

### 3.8 Cross-Cutting Boundary Contracts

#### CON-BD-022: Data Directory Resolution

Every command that accepts `--data-dir` SHALL resolve the CA data directory as follows, in order of precedence:

1. `--data-dir` flag value, if provided.
2. `CA_DATA_DIR` environment variable, if set.
3. `./ca-data` as the default.

The `--data-dir` flag SHALL always take precedence over the `CA_DATA_DIR` environment variable.

**Traces to:** REQ-CL-008

---

#### CON-BD-023: Exit Code Semantics

All commands SHALL use exactly these exit codes:

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Operational error (invalid CSR, revoked certificate, verification failure, CA not initialized, etc.) |
| `2` | Usage error (missing required flags, invalid flag values, unknown commands) |

No other exit codes SHALL be used.

**Traces to:** REQ-CL-009

---

## 4. Security Contracts (CON-SC)

Non-negotiable security boundaries relevant to the core principle of CA operations.

---

### CON-SC-001: Private Key Material Never in Output

The CA private key material (the actual key bytes or PEM-encoded key content) SHALL never appear in stdout, stderr, or any log output. Only the file path to the key file SHALL be printed. This applies to CA keys and any keys generated by `ca request`.

**Traces to:** REQ-MK-002, REQ-MK-005

---

### CON-SC-002: Cryptographically Secure Key Generation

All key pairs generated by the system (CA key pair via `ca init`, request key pair via `ca request`) SHALL use a cryptographically secure random number generator (CSPRNG) provided by the runtime or operating system. Deterministic or predictable random sources SHALL NOT be used for key generation.

**Traces to:** REQ-CP-001, REQ-CL-007

---

### CON-SC-003: CSR Validation Gate

No certificate SHALL be issued without first completing both of the following checks, in any order:

1. The CSR's self-signature SHALL be verified as valid.
2. The CSR's public key algorithm SHALL be confirmed as ECDSA P-256 or RSA 2048.

Both checks SHALL pass before any certificate is created, any file is written to `certs/`, any serial number is consumed, or any entry is added to `index.json`. If either check fails, the system SHALL remain in the exact state it was in before the command was invoked.

**Traces to:** REQ-CP-002, REQ-MK-004, REQ-ER-001, REQ-ER-006

---

## 5. Data Integrity Contracts (CON-DI)

Contracts governing the consistency, format, and lifecycle of all data.

---

### CON-DI-001: PEM Encoding for All Artifacts

All cryptographic artifacts SHALL use PEM encoding with the following headers:

| Artifact | PEM Header |
|----------|-----------|
| Private keys | `-----BEGIN PRIVATE KEY-----` (PKCS#8, unencrypted) |
| Certificates | `-----BEGIN CERTIFICATE-----` |
| CSRs | `-----BEGIN CERTIFICATE REQUEST-----` |
| CRLs | `-----BEGIN X509 CRL-----` |

No other PEM header types SHALL be used. Encrypted private key headers (`-----BEGIN ENCRYPTED PRIVATE KEY-----`) SHALL NOT be used.

**Traces to:** REQ-DT-001

---

### CON-DI-002: Serial Number Hexadecimal Format

Serial numbers SHALL be stored and displayed as lowercase hexadecimal strings, zero-padded to at least 2 digits. Examples: `01`, `02`, `0a`, `0b`, `ff`. Uppercase hex digits SHALL NOT be used in storage or display.

**Traces to:** REQ-DT-005

---

### CON-DI-003: Timestamps in RFC 3339 UTC

All timestamps stored in `index.json` (`not_before`, `not_after`, `revoked_at`) SHALL be in RFC 3339 UTC format, ending with `Z`. Example: `2026-02-09T12:00:00Z`. Non-UTC timezones or non-RFC-3339 formats SHALL NOT be used.

**Traces to:** REQ-DT-006

---

### CON-DI-004: Atomicity — Failed Operations Shall Not Modify State

When any command fails for any reason — CSR parse failure, CSR signature failure, unsupported key algorithm, serial not found, double revocation, re-initialization attempt, or any other error — no persistent state files SHALL be modified. Specifically:

- `serial` SHALL remain unchanged.
- `crlnumber` SHALL remain unchanged.
- `index.json` SHALL remain unchanged.
- No files SHALL be created in `certs/`.
- `ca.key` and `ca.crt` SHALL remain unchanged.
- `ca.crl` SHALL remain unchanged.

The system SHALL be in the exact same state as before the failed command was invoked.

**Implementation strategy:** Two complementary mechanisms enforce this contract:

1. **Validate-before-mutate** (ADR-003): All precondition checks execute before any state modification. Validation failures return errors without touching any files.
2. **Atomic file replacement** (ADR-006): All mutate-phase writes use temp-file-then-rename (`writeFileAtomic`). Multi-file mutations stage all outputs to `.tmp` files first; if any staging write fails, all `.tmp` files are removed and no final paths are modified. Renames proceed in a defined commit order that minimizes inconsistency in the unlikely event of a process crash between renames.

**Residual risk:** A process crash (e.g., SIGKILL, power loss) occurring between individual `rename(2)` calls in the commit sub-phase of a multi-file mutation can produce partially committed state. This window is on the order of microseconds and is inherent to flat-file storage without a write-ahead log. The commit order for each operation is defined in ADR-006 to ensure the least harmful partial state.

**Traces to:** REQ-ER-001, REQ-ER-003, REQ-ER-004, REQ-ER-005, REQ-ER-006, REQ-ER-008

---

### CON-DI-005: Index Schema Completeness

Every entry in `index.json` SHALL have the following seven fields:

| Field | Type | Constraints |
|-------|------|-------------|
| `serial` | string | Lowercase hex, zero-padded to at least 2 digits |
| `subject` | string | Certificate subject DN |
| `not_before` | string | RFC 3339 UTC timestamp |
| `not_after` | string | RFC 3339 UTC timestamp |
| `status` | string | Exactly `"active"` or `"revoked"` |
| `revoked_at` | string | RFC 3339 UTC timestamp if revoked, `""` if active |
| `revocation_reason` | string | Reason code string if revoked, `""` if active |

When status is `active`: `revoked_at` and `revocation_reason` SHALL both be `""`.
When status is `revoked`: `revoked_at` and `revocation_reason` SHALL both be non-empty.

**Traces to:** REQ-DT-006

---

### CON-DI-006: CRL–Index Consistency

A generated CRL SHALL contain the serial numbers and revocation dates of all certificates with status `revoked` in `index.json` at the time of CRL generation. No revoked certificate SHALL be omitted from the CRL. No non-revoked certificate SHALL appear in the CRL.

**Traces to:** REQ-CP-006, REQ-DT-004

---

### CON-DI-007: Certificate–Index Correspondence

For every entry in `index.json` with serial `S`, a certificate file `<data-dir>/certs/<S>.pem` SHALL exist. Conversely, for every file in `<data-dir>/certs/`, a corresponding entry SHALL exist in `index.json`.

**Traces to:** REQ-DT-006, REQ-DT-007

---

### CON-DI-008: Serial Counter Consistency

The serial counter file SHALL always contain the next serial number to be assigned, as a lowercase hex string zero-padded to at least 2 digits. After initialization (root certificate serial `01` assigned), the file SHALL contain `02`. After `N` end-entity certificates have been issued, the file SHALL contain the hex representation of `N + 2`.

**Traces to:** REQ-CP-004, REQ-DT-005

---

### CON-DI-009: CRL Number Counter Consistency

The CRL number counter file SHALL always contain the next CRL number to be assigned, as a lowercase hex string zero-padded to at least 2 digits. After initialization, the file SHALL contain `01`. After `M` CRLs have been generated, the file SHALL contain the hex representation of `M + 1`.

**Traces to:** REQ-CP-006, REQ-DT-004

---

### CON-DI-010: X.509 Version 3 for All Certificates

All certificates produced by the system — both the root CA certificate and all issued end-entity certificates — SHALL be X.509 version 3.

**Traces to:** REQ-CP-001, REQ-CP-003

---

### CON-DI-011: Root CA Certificate Extensions

The root CA certificate SHALL have the following extensions:

| Extension | Critical | Value |
|-----------|----------|-------|
| Basic Constraints | Yes | `cA=TRUE` |
| Key Usage | Yes | `keyCertSign`, `cRLSign` |
| Subject Key Identifier | No | SHA-1 hash of the CA's public key (RFC 5280 method 1) |

**Traces to:** REQ-DT-002

---

### CON-DI-012: End-Entity Certificate Extensions

Issued end-entity certificates SHALL have the following extensions:

| Extension | Critical | Value |
|-----------|----------|-------|
| Basic Constraints | Yes | `cA=FALSE` |
| Key Usage | Yes | `digitalSignature` (always); additionally `keyEncipherment` if the subject key is RSA |
| Subject Alternative Name | No | Copied from the CSR's SAN extension if present; omitted entirely if the CSR has no SAN |
| Authority Key Identifier | No | The CA's Subject Key Identifier value |
| Subject Key Identifier | No | SHA-1 hash of the subject's public key |

**Traces to:** REQ-DT-003

---

### CON-DI-013: CRL Structure

The CRL SHALL be X.509 CRL version 2 with the following structure:

| Field | Value |
|-------|-------|
| Issuer | The CA's Distinguished Name |
| This Update | Time of CRL generation (UTC) |
| Next Update | `thisUpdate` + configured hours (default 24) |
| Revoked Certificates | List of entries, each with: serial number (hex), revocation date (UTC), reason code |
| Authority Key Identifier (non-critical) | The CA's Subject Key Identifier value |
| CRL Number (non-critical) | Monotonically increasing integer, starting at `1` |
| Signature | Signed by the CA's private key using SHA-256 |

**Traces to:** REQ-DT-004

---

### CON-DI-014: System Clock for Timestamps

All timestamps generated by the system (certificate validity periods, revocation timestamps, CRL thisUpdate/nextUpdate) SHALL use the system clock as the time source. No external time synchronization or trusted time source is required or expected.

**Traces to:** REQ-MK-006

---

## 6. Traceability

### 6.1 Contract → Requirement Mapping

| Contract | Requirement(s) |
|----------|----------------|
| CON-INV-001 | REQ-CP-004 |
| CON-INV-002 | REQ-CP-004, REQ-DT-005 |
| CON-INV-003 | REQ-CP-005, REQ-ER-004 |
| CON-INV-004 | REQ-ER-002 |
| CON-INV-005 | REQ-CP-003, REQ-CP-006, REQ-DT-003, REQ-DT-004 |
| CON-INV-006 | REQ-CP-001 |
| CON-INV-007 | REQ-CP-006 |
| CON-INV-008 | REQ-CP-001, REQ-CP-003, REQ-CP-006 |
| CON-INV-009 | REQ-DT-006 |
| CON-INV-010 | REQ-CP-001, REQ-ER-006, REQ-MK-004 |
| CON-INV-011 | REQ-MK-001, REQ-MK-004 |
| CON-BD-001 | REQ-CL-001, REQ-ER-005 |
| CON-BD-002 | REQ-CP-001, REQ-DT-007, REQ-MK-002, REQ-MK-005 |
| CON-BD-003 | REQ-ER-005, REQ-CL-001, REQ-CL-009 |
| CON-BD-004 | REQ-CL-002, REQ-CP-002, REQ-ER-001, REQ-ER-006, REQ-ER-008, REQ-MK-004 |
| CON-BD-005 | REQ-CP-003, REQ-CP-004, REQ-DT-003, REQ-DT-006, REQ-MK-005 |
| CON-BD-006 | REQ-ER-001, REQ-ER-002, REQ-ER-006, REQ-ER-008, REQ-CL-009 |
| CON-BD-007 | REQ-CL-003, REQ-CP-005, REQ-ER-003, REQ-ER-004 |
| CON-BD-008 | REQ-CP-005, REQ-DT-006, REQ-MK-005 |
| CON-BD-009 | REQ-ER-002, REQ-ER-003, REQ-ER-004, REQ-CL-009 |
| CON-BD-010 | REQ-CL-004, REQ-ER-002 |
| CON-BD-011 | REQ-CP-006, REQ-DT-004, REQ-MK-003, REQ-MK-005 |
| CON-BD-012 | REQ-ER-002, REQ-CL-009 |
| CON-BD-013 | REQ-CL-005, REQ-ER-002 |
| CON-BD-014 | REQ-CP-008, REQ-CL-005 |
| CON-BD-015 | REQ-ER-002 |
| CON-BD-016 | REQ-CL-006, REQ-ER-002 |
| CON-BD-017 | REQ-CP-007, REQ-CL-006 |
| CON-BD-018 | REQ-ER-002, REQ-ER-007, REQ-CL-009 |
| CON-BD-019 | REQ-CL-007 |
| CON-BD-020 | REQ-CL-007, REQ-DT-001, REQ-MK-005 |
| CON-BD-021 | REQ-CL-007, REQ-CL-009 |
| CON-BD-022 | REQ-CL-008 |
| CON-BD-023 | REQ-CL-009 |
| CON-SC-001 | REQ-MK-002, REQ-MK-005 |
| CON-SC-002 | REQ-CP-001, REQ-CL-007 |
| CON-SC-003 | REQ-CP-002, REQ-MK-004, REQ-ER-001, REQ-ER-006 |
| CON-DI-001 | REQ-DT-001 |
| CON-DI-002 | REQ-DT-005 |
| CON-DI-003 | REQ-DT-006 |
| CON-DI-004 | REQ-ER-001, REQ-ER-003, REQ-ER-004, REQ-ER-005, REQ-ER-006, REQ-ER-008 |
| CON-DI-005 | REQ-DT-006 |
| CON-DI-006 | REQ-CP-006, REQ-DT-004 |
| CON-DI-007 | REQ-DT-006, REQ-DT-007 |
| CON-DI-008 | REQ-CP-004, REQ-DT-005 |
| CON-DI-009 | REQ-CP-006, REQ-DT-004 |
| CON-DI-010 | REQ-CP-001, REQ-CP-003 |
| CON-DI-011 | REQ-DT-002 |
| CON-DI-012 | REQ-DT-003 |
| CON-DI-013 | REQ-DT-004 |
| CON-DI-014 | REQ-MK-006 |

### 6.2 Requirement → Contract Mapping

| Requirement | Contract(s) |
|-------------|-------------|
| REQ-CP-001 | CON-INV-006, CON-INV-008, CON-INV-010, CON-BD-002, CON-SC-002, CON-DI-010 |
| REQ-CP-002 | CON-BD-004, CON-SC-003 |
| REQ-CP-003 | CON-INV-005, CON-INV-008, CON-BD-005, CON-DI-010 |
| REQ-CP-004 | CON-INV-001, CON-INV-002, CON-BD-005, CON-DI-008 |
| REQ-CP-005 | CON-INV-003, CON-BD-007, CON-BD-008 |
| REQ-CP-006 | CON-INV-005, CON-INV-007, CON-INV-008, CON-BD-011, CON-DI-006, CON-DI-009, CON-DI-013 |
| REQ-CP-007 | CON-BD-017 |
| REQ-CP-008 | CON-BD-014 |
| REQ-CL-001 | CON-BD-001, CON-BD-003 |
| REQ-CL-002 | CON-BD-004 |
| REQ-CL-003 | CON-BD-007 |
| REQ-CL-004 | CON-BD-010 |
| REQ-CL-005 | CON-BD-013, CON-BD-014 |
| REQ-CL-006 | CON-BD-016, CON-BD-017 |
| REQ-CL-007 | CON-BD-019, CON-BD-020, CON-BD-021, CON-SC-002 |
| REQ-CL-008 | CON-BD-022 |
| REQ-CL-009 | CON-BD-003, CON-BD-006, CON-BD-009, CON-BD-012, CON-BD-018, CON-BD-021, CON-BD-023 |
| REQ-DT-001 | CON-BD-020, CON-DI-001 |
| REQ-DT-002 | CON-DI-011 |
| REQ-DT-003 | CON-INV-005, CON-BD-005, CON-DI-012 |
| REQ-DT-004 | CON-INV-005, CON-BD-011, CON-DI-006, CON-DI-009, CON-DI-013 |
| REQ-DT-005 | CON-INV-002, CON-DI-002, CON-DI-008 |
| REQ-DT-006 | CON-INV-009, CON-BD-005, CON-BD-008, CON-DI-003, CON-DI-005, CON-DI-007 |
| REQ-DT-007 | CON-BD-002, CON-DI-007 |
| REQ-ER-001 | CON-BD-004, CON-BD-006, CON-SC-003, CON-DI-004 |
| REQ-ER-002 | CON-INV-004, CON-BD-006, CON-BD-009, CON-BD-010, CON-BD-012, CON-BD-013, CON-BD-015, CON-BD-016, CON-BD-018 |
| REQ-ER-003 | CON-BD-007, CON-BD-009, CON-DI-004 |
| REQ-ER-004 | CON-INV-003, CON-BD-007, CON-BD-009, CON-DI-004 |
| REQ-ER-005 | CON-BD-001, CON-BD-003, CON-DI-004 |
| REQ-ER-006 | CON-INV-010, CON-BD-004, CON-BD-006, CON-SC-003, CON-DI-004 |
| REQ-ER-007 | CON-BD-018 |
| REQ-ER-008 | CON-BD-004, CON-BD-006, CON-DI-004 |
| REQ-MK-001 | CON-INV-011 |
| REQ-MK-002 | CON-BD-002, CON-SC-001 |
| REQ-MK-003 | CON-BD-011 |
| REQ-MK-004 | CON-INV-010, CON-INV-011, CON-BD-004, CON-SC-003 |
| REQ-MK-005 | CON-BD-002, CON-BD-005, CON-BD-008, CON-BD-011, CON-BD-020, CON-SC-001 |
| REQ-MK-006 | CON-DI-014 |

### 6.3 Coverage Gaps

Every contract traces to at least one requirement. Every requirement is covered by at least one contract.

---

## 7. Summary

| Category | Count | ID Range |
|----------|-------|----------|
| System-Wide Invariants | 11 | CON-INV-001 – CON-INV-011 |
| Boundary Contracts | 23 | CON-BD-001 – CON-BD-023 |
| Security Contracts | 3 | CON-SC-001 – CON-SC-003 |
| Data Integrity Contracts | 14 | CON-DI-001 – CON-DI-014 |
| **Total** | **51** | |
