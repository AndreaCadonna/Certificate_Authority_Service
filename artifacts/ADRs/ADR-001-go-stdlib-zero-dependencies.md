# ADR-001: Go Standard Library for All X.509 Operations

## Status

Accepted

## Context

The Certificate Authority Service experiment requires X.509v3 certificate creation, CSR parsing, CRL generation, RSA/ECDSA key generation, and PEM encoding. RESEARCH.md §3.1 evaluated four candidate languages: Go, Python, Rust, and Java/Kotlin.

The workflow philosophy mandates boring tech, standard library first, zero-config execution, and minimal dependencies. The experiment's core principle is CA certificate lifecycle management — the language choice must optimize for X.509 domain coverage, not for developer velocity or ecosystem breadth.

Two viable candidates emerged: Go (zero dependencies) and Python + `cryptography` library (one dependency). Both can implement the experiment fully. The decision affects build process, dependency management, binary distribution, and whether the experiment starts with a `pip install` step.

## Decision

Use Go 1.21+ with zero external dependencies for all X.509 operations. The Go standard library's `crypto/x509` package provides `CreateCertificate`, `ParseCertificateRequest`, `CreateCertificateRequest`, `CreateRevocationList`, and full RSA/ECDSA key generation — every operation the experiment requires.

## Alternatives Considered

- **Python + `cryptography` library**: Python's `ssl` and `hashlib` modules cannot generate certificates, CSRs, or CRLs. The `cryptography` library (pyca/cryptography) covers all CA operations with a builder-pattern API. It is the de facto standard for Python PKI work. However, it is an external dependency with a compiled Rust backend. Installation requires `pip install cryptography`, which downloads and compiles native extensions. This violates the "standard library first" and "zero-config" philosophy constraints. Python would produce approximately 500-1000 LOC (vs Go's 800-1500) but requires a dependency management step that Go avoids entirely. RESEARCH.md rated Python as "strong second candidate" specifically because of this dependency requirement.

- **Rust**: Standard library has no cryptographic primitives. Would require `rcgen` + `x509-parser` + additional crates. Not "boring tech" for most teams. Steeper learning curve would spend innovation tokens on the language rather than on CA mechanics. RESEARCH.md verdict: "Not recommended for this experiment."

- **Java/Kotlin**: Standard library cannot easily generate X.509v3 certificates with extensions. Requires Bouncy Castle (~10MB dependency). JVM startup overhead, verbose code, and 15-25 file requirement conflict with the small experiment philosophy. RESEARCH.md verdict: "Not recommended. Overkill."

## Consequences

### Positive

- Zero external dependencies. No `go.sum`, no version pinning, no supply chain risk.
- `go build` produces a single static binary. No runtime dependencies, no virtual environments, no interpreters.
- Zero-config execution: `go build -o ca .` is the entire build process.
- Go's `crypto/x509` package is battle-tested — production CAs (Smallstep step-ca, parts of Let's Encrypt's Boulder) are built on it.
- RESEARCH.md estimated 800-1500 LOC, which is within the experiment's scope.

### Negative

- Go is more verbose than Python for equivalent operations. The experiment will have more boilerplate (error handling, type assertions) than a Python equivalent.
- Go's template-based certificate API requires understanding struct fields rather than Python's builder-pattern API which is more discoverable.
- Developers more familiar with Python may find the Go implementation harder to follow.

### Neutral

- Go 1.21+ is required on the development machine (Assumption A-1 in SPEC.md).
- The choice of Go does not affect the CA mechanics being demonstrated — the same X.509 operations happen regardless of language.

## References

- REQ-CP-001, REQ-CP-002, REQ-CP-003, REQ-CP-006 from SPEC.md (X.509 operations requiring library support)
- SPEC.md D-1: "Language: Go — Zero external dependencies for X.509 operations."
- RESEARCH.md §3.1: Go evaluation and Python evaluation
- CON-SC-002: Cryptographically secure key generation (Go's `crypto/rand` satisfies this)
