# Certificate Authority Service

A command-line Certificate Authority (CA) that manages X.509 digital certificates through their full lifecycle: initialization, CSR signing, revocation, CRL generation, verification, and listing.

Built in Go with **zero external dependencies** — uses only the Go standard library for all cryptographic operations.

## Features

- **Root CA initialization** with ECDSA P-256 (default) or RSA 2048 key pairs
- **CSR signing** — accepts any valid PEM-encoded PKCS#10 CSR
- **Certificate revocation** with reason codes (unspecified, keyCompromise, affiliationChanged, superseded, cessationOfOperation)
- **CRL generation** — X.509 CRL v2 with configurable next-update period
- **Certificate verification** — signature, expiry, and CRL revocation checks
- **Certificate listing** with dynamic status (active, revoked, expired)
- **CSR generation** utility for creating key pairs and certificate signing requests

## Prerequisites

- Go 1.21+

## Build

```bash
go build -o ca .
```

On Windows this produces `ca.exe`.

## Usage

```
ca <command> [options]

Commands:
  init      Initialize a new Certificate Authority
  sign      Sign a CSR to issue a certificate
  revoke    Revoke a certificate by serial number
  crl       Generate a Certificate Revocation List
  list      List all issued certificates
  verify    Verify a certificate against the CA
  request   Generate a new key pair and CSR
```

### Initialize a CA

```bash
ca init --subject "CN=My Root CA,O=My Org" [--key-algorithm ecdsa-p256] [--validity 3650]
```

### Generate a CSR

```bash
ca request --subject "CN=example.com,O=My Org" --out-key server.key --out-csr server.csr [--san "DNS:example.com,DNS:www.example.com,IP:192.168.1.1"]
```

### Sign a CSR

```bash
ca sign [--validity 365] server.csr
```

### List certificates

```bash
ca list
```

### Revoke a certificate

```bash
ca revoke [--reason keyCompromise] 02
```

### Generate a CRL

```bash
ca crl [--next-update 168]
```

### Verify a certificate

```bash
ca verify certs/02.crt
```

### Data directory

All CA data is stored in `./ca-data/` by default. Override with:

- `--data-dir <path>` flag (highest priority)
- `CA_DATA_DIR` environment variable
- Default: `./ca-data`

## Data Layout

```
ca-data/
  ca.key          # CA private key (PKCS#8 PEM, unencrypted)
  ca.crt          # CA root certificate (PEM)
  ca.crl          # Certificate Revocation List (PEM)
  serial          # Next serial number (hex)
  crlnumber       # Next CRL number (hex)
  index.json      # Certificate index (JSON array)
  certs/
    02.crt        # Issued certificates by serial number
    03.crt
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Operational error (invalid CSR, certificate not found, etc.) |
| 2 | Usage error (missing required flags, unknown command) |

## Validation

A behavioral validation script exercises the full certificate lifecycle:

```bash
bash validate.sh
```

This runs 114 checks covering all commands, error scenarios, and edge cases.

## Design

This project was built following a spec-driven development workflow. Design documents are in the `artifacts/` directory:

- `RESEARCH.md` — Background research on CA mechanics and technology choices
- `SPEC.md` — 34 functional requirements and 7 CLI command specifications
- `CONTRACTS.md` — 51 runtime contracts (invariants, boundary, security, data integrity)
- `DESIGN.md` — Component architecture and 8-step implementation plan
- `ADRs/` — 6 Architecture Decision Records
- `IMPLEMENTATION.md` — Implementation status, deviations, and validation results

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go | Standard library covers X.509, CSR, CRL, PEM, key generation. Zero dependencies. |
| CLI framework | `flag` package | No external CLI libraries. Manual subcommand dispatch. |
| Atomicity | Validate-before-mutate + atomic file writes | All mutations are validated first. File writes use temp-file-then-rename. |
| Testing | Behavioral validation script | Tests the compiled binary end-to-end rather than individual functions. |
| Key algorithms | ECDSA P-256 / RSA 2048 | Modern defaults with SHA-256 signatures. |
| Storage | File system + JSON index | Simple, inspectable, no database dependency. |

## Limitations

This is a **learning experiment**, not a production CA:

- CA private key is stored unencrypted on disk
- No identity verification — the CA signs any valid CSR
- CRL is a local file, not served over HTTP
- Single operator, no concurrency
- No OCSP, no intermediate CAs, no certificate renewal
