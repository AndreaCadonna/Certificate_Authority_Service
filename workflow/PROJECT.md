# Project Scope

## Project Idea

**Certificate Authority (CA) Service**

## Domain

Public Key Infrastructure (PKI), certificate lifecycle management, and revocation strategies.

## Key Areas to Explore

- Certificate Authority fundamentals — how a CA issues, signs, and manages digital certificates
- Certificate lifecycle — generation, signing, renewal, expiration, revocation
- Revocation strategies — Certificate Revocation Lists (CRL) vs Online Certificate Status Protocol (OCSP), their tradeoffs, and when each is appropriate

## Research Directives

The research phase must specifically address:

1. **Programming language evaluation** — Which languages are best suited for implementing a CA service experiment? Evaluate based on: standard library cryptography support, ASN.1/X.509 handling, ease of certificate manipulation, community maturity for PKI work.
2. **Design pattern analysis** — Which architectural and design patterns are most appropriate for a CA service? Consider: certificate chain of trust modeling, storage patterns for certificate state, strategy patterns for revocation mechanisms (CRL vs OCSP), command patterns for lifecycle operations.
