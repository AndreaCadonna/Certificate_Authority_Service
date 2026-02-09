---
name: code-quality
description: Defines code standards, documentation conventions, and quality expectations for implementation. Used by the developer and fixer agents to write clean, consistent, well-documented code that traces back to specifications.
---

# Code Quality Skill

## Step-by-Step Instructions

1. Follow the chosen language's standard conventions (PEP 8 for Python, gofmt for Go, etc.).
2. Name files by what they do, not by architectural role. Use `certificate.py`, not `service.py`.
3. Name functions and variables descriptively. Prefer clarity over brevity.
4. Add a file-level docstring/comment to every file linking it to spec requirements.
5. Add inline comments only where the logic is non-obvious — explain WHY, not WHAT.
6. Enforce contracts from CONTRACTS.md as runtime assertions or validation checks in code.
7. Handle errors defined in SPEC.md §7 — expected error cases only, not every possible failure.
8. No dead code, no debug artifacts, no TODO comments, no commented-out code.
9. Prefer explicit data passing over global state. Functions receive inputs and return outputs.
10. Keep functions short and focused. If a function does two things, split it.

## Examples

**Good file-level comment:**
```python
"""
Certificate generation and parsing utilities.

Handles X.509 certificate creation, CSR parsing, and certificate
chain validation. Implements requirements REQ-CA-001 through REQ-CA-005.

Enforces contracts:
  CON-INV-01: Issuer field must match signing CA's subject
  CON-INV-02: Certificate serial numbers are unique within the CA
"""
```

**Bad file-level comment:**
```python
# certificate.py - handles certificates
```

**Good inline comment (explains WHY):**
```python
# RFC 5280 §4.1.2.2: Serial numbers must be positive integers.
# We use 20 bytes (160 bits) to minimize collision probability
# without exceeding the maximum of 20 octets.
serial = x509.random_serial_number()
```

**Bad inline comment (explains WHAT):**
```python
# Generate a random serial number
serial = x509.random_serial_number()
```

**Good contract enforcement:**
```python
def sign_certificate(csr, ca_cert, ca_key):
    # CON-BND-03: CA certificate must have CA:TRUE basic constraint
    if not ca_cert.extensions.get_extension_for_class(BasicConstraints).value.ca:
        raise ValueError("Signing certificate is not a CA (CON-BND-03)")

    # CON-BND-03: Signing key must correspond to CA certificate
    if ca_key.public_key() != ca_cert.public_key():
        raise ValueError("Key does not match CA certificate (CON-BND-03)")
```

**Good error handling (expected case from spec):**
```python
def revoke_certificate(serial_number, ca_state):
    if serial_number not in ca_state.issued_certificates:
        # SPEC.md §7: Error E-REV-001 — unknown serial number
        return {"error": "E-REV-001", "message": f"Certificate {serial_number} not found"}
```

## Common Edge Cases

- A function seems to need a comment but the logic is straightforward. Don't add the comment — clear code is its own documentation.
- A contract check adds overhead. Add it anyway. Correctness over performance for experiments.
- An error from the spec seems unlikely. Handle it anyway if it's in the spec. Skip it only if it's not in the spec.
- The language has multiple ways to do something. Choose the most idiomatic, most widely-known approach. No clever tricks.

## Quality Checklist

- [ ] Code follows the language's standard style conventions
- [ ] Files are named by function (no generic names)
- [ ] Every file has a file-level comment linking to requirements
- [ ] Inline comments explain WHY, not WHAT
- [ ] No dead code, no debug prints, no TODO comments, no commented-out code
- [ ] Contracts from CONTRACTS.md are enforced as runtime checks
- [ ] Contract violations raise clear error messages referencing the contract ID
- [ ] Error handling covers cases from SPEC.md §7
- [ ] Functions receive inputs and return outputs (no hidden global state)
- [ ] Error messages are clear and actionable
- [ ] No unnecessary abstractions or indirection
- [ ] Code is readable by someone unfamiliar with the project
