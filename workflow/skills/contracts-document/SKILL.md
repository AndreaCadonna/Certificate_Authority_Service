---
name: contracts-document
description: Defines the format, structure, and quality standards for CONTRACTS.md documents. Used by the contract-writer agent in Phase 2 to extract invariants, boundary rules, security constraints, and data integrity contracts from SPEC.md.
---

# Contracts Document Skill

## Step-by-Step Instructions

1. Read SPEC.md completely. Understand every requirement, interface, and scenario.
2. Identify **system invariants** — global truths that must never be violated across the entire system, at any time, regardless of the operation being performed.
3. Identify **boundary contracts** — per-interface preconditions (what must be true before), postconditions (what must be true after), and error conditions (what happens when preconditions are violated).
4. Identify **security contracts** — non-negotiable security boundaries that no implementation decision can weaken. These are hard constraints, not guidelines.
5. Identify **data integrity contracts** — rules governing data consistency, format constraints, state transitions, and lifecycle rules.
6. Assign identifiers using the format: CON-INV-XX (invariants), CON-BND-XX (boundary), CON-SEC-XX (security), CON-DAT-XX (data integrity).
7. Map every contract back to one or more requirements (REQ-XX-NNN) from SPEC.md.
8. Verify: no contract invents a new requirement. Every contract formalizes a truth already implied by the spec.
9. Write CONTRACTS.md following the Output Template below.

## Examples

**Good system invariant:**
`CON-INV-01`: A certificate's issuer field must always match the subject field of its signing CA certificate. (Traces to: REQ-CA-003, REQ-CA-005)

**Bad system invariant:**
"Certificates should be valid." (Not specific. What property? What constitutes validity?)

**Good boundary contract:**
```
CON-BND-03: sign_certificate(csr, ca_cert, ca_key)
  Preconditions:
    - csr is a valid PKCS#10 certificate signing request
    - ca_cert has the CA:TRUE basic constraint
    - ca_key corresponds to ca_cert's public key
  Postconditions:
    - Returned certificate's issuer equals ca_cert's subject
    - Returned certificate's signature is verifiable using ca_cert's public key
    - Returned certificate's serial number is unique within the CA's issued set
  Error conditions:
    - If ca_cert lacks CA:TRUE → reject with error, no certificate produced
    - If ca_key doesn't match ca_cert → reject with error, no certificate produced
  Traces to: REQ-CA-003, REQ-CA-004
```

**Good security contract:**
`CON-SEC-01`: Private keys must never appear in CLI output, log messages, or any non-key-file output. (Traces to: REQ-SEC-001)

## Common Edge Cases

- An invariant seems obvious. Include it anyway — what's obvious to a domain expert is not obvious to an implementation agent. Explicit is better than implicit.
- A security constraint is implied but never stated in the spec. Flag it and include it — security contracts formalize implied safety, they don't need explicit spec statements.
- Two contracts overlap. Check if they're truly the same constraint at different levels (invariant vs boundary). If so, keep both — they serve different validation purposes.
- A contract seems to require a specific implementation. Rewrite it in terms of observable behavior, not implementation mechanism.

## Output Template

```markdown
# CONTRACTS.md — [Project Name]

## §1 — System Invariants

[Global truths that must hold at all times, across all operations. Each with CON-INV-XX identifier and requirement traceability.]

| ID | Invariant | Traces To |
|----|-----------|-----------|
| CON-INV-01 | [Statement] | REQ-XX-NNN |

### CON-INV-01: [Short Title]
[Detailed description of the invariant. What it means. Why it must hold. What breaks if it's violated.]

(Continue for all invariants.)

## §2 — Boundary Contracts

[Per-interface preconditions, postconditions, and error conditions. Each with CON-BND-XX identifier.]

### CON-BND-01: [Function/Command Name]
- **Preconditions:** [What must be true before execution]
- **Postconditions:** [What must be true after execution]
- **Error conditions:** [What happens when preconditions are violated]
- **Traces to:** REQ-XX-NNN

(Continue for all interfaces.)

## §3 — Security Contracts

[Non-negotiable security boundaries. Each with CON-SEC-XX identifier.]

| ID | Contract | Traces To |
|----|----------|-----------|
| CON-SEC-01 | [Statement] | REQ-XX-NNN |

### CON-SEC-01: [Short Title]
[Detailed description. What boundary it enforces. What attack or failure it prevents.]

(Continue for all security contracts.)

## §4 — Data Integrity Contracts

[Data consistency, format, lifecycle, and state transition rules. Each with CON-DAT-XX identifier.]

| ID | Contract | Traces To |
|----|----------|-----------|
| CON-DAT-01 | [Statement] | REQ-XX-NNN |

### CON-DAT-01: [Short Title]
[Detailed description. What data rule it enforces. What inconsistency it prevents.]

(Continue for all data integrity contracts.)

## §5 — Traceability

### §5.1 — Contract → Requirement Mapping
| Contract | Requirements |
|----------|-------------|
| CON-XX   | REQ-XX-NNN  |

### §5.2 — Requirement → Contract Mapping
| Requirement | Contracts |
|-------------|-----------|
| REQ-XX-NNN  | CON-XX    |

[Every contract must trace to at least one requirement. Flag any requirement that has no contract — this may indicate a gap or may be acceptable if the requirement has no invariant properties.]
```

## Quality Checklist

- [ ] Every contract has a CON-XX identifier with correct category prefix
- [ ] Every contract traces to at least one REQ-XX-NNN from SPEC.md
- [ ] No contract invents a requirement not present in SPEC.md
- [ ] System invariants (§1) describe properties that hold at ALL times, not just during specific operations
- [ ] Boundary contracts (§2) have explicit preconditions, postconditions, AND error conditions
- [ ] Security contracts (§3) are hard constraints, not guidelines or recommendations
- [ ] Data integrity contracts (§4) cover format, consistency, and lifecycle rules
- [ ] Contracts are implementation-agnostic — they describe observable behavior, not mechanism
- [ ] §5 traceability is bidirectional (contract→requirement AND requirement→contract)
- [ ] Requirements without contracts are flagged and justified
- [ ] No empty sections or placeholder text
- [ ] Language is precise — no "should", "properly", "correctly"
