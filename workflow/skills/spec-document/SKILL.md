---
name: spec-document
description: Defines the format, structure, and quality standards for SPEC.md documents. Used by the spec-writer agent in Phase 1 to transform RESEARCH.md into a precise, testable specification with requirements, interfaces, behavior scenarios, and traceability.
---

# Spec Document Skill

## Step-by-Step Instructions

1. Read RESEARCH.md completely. Internalize the core principle, scope, and domain concepts.
2. Extract functional requirements from the research. Each requirement must be independently testable.
3. Assign requirement identifiers using the format `REQ-XX-NNN` where XX is a category code and NNN is sequential (e.g., REQ-CA-001, REQ-CRL-001).
4. Define all interfaces: CLI commands, function signatures, data formats (input and output).
5. Write behavior scenarios in Given/When/Then format using concrete sample data — never abstract placeholders.
6. Cover both success paths and failure/error paths for each interface.
7. Build the traceability matrix mapping every requirement to at least one scenario.
8. Verify completeness: every in-scope item from RESEARCH.md §5.1 must have at least one requirement.
9. Write SPEC.md following the Output Template below.

## Examples

**Good requirement:**
`REQ-CA-001`: The system shall generate a self-signed root CA certificate with a configurable subject name, key algorithm (RSA-2048 or RSA-4096), and validity period.

**Bad requirement:**
"The system should handle certificates properly." (Not testable, not specific.)

**Good scenario:**
```
Scenario: Generate root CA certificate
  Given no existing CA state
  When the user runs `ca init --subject "CN=MyRootCA" --key-algo RSA-2048 --validity-days 3650`
  Then a root CA certificate is created with subject "CN=MyRootCA"
  And the certificate is self-signed (issuer equals subject)
  And the key algorithm is RSA-2048
  And the certificate validity is 3650 days from the current date
  And the certificate has the CA:TRUE basic constraint
```

**Bad scenario:**
"When you create a certificate, it should work correctly." (No concrete data, no observable outcome.)

## Common Edge Cases

- A requirement seems testable but has no observable output. Rewrite it so the behavior produces something checkable (exit code, file content, CLI output).
- Two requirements overlap significantly. Merge them or clarify the boundary.
- A scenario requires state from a previous scenario. Make dependencies explicit in the Given clause.
- The research identified something as in-scope but it's hard to specify. This might indicate a scope problem — consider whether it truly serves the core principle.

## Output Template

```markdown
# SPEC.md — [Project Name]

## §1 — Core Principle

[Carried forward from RESEARCH.md §1. Restated for self-containment.]

## §2 — Scope Summary

[Carried forward from RESEARCH.md §5. Restated for self-containment. What's in, what's out, what's mocked.]

## §3 — Requirements

### §3.1 — [Category Name]
[Requirements in this category, each with REQ-XX-NNN identifier.]

### §3.2 — [Category Name]
[Requirements in this category.]

(Continue for all requirement categories.)

## §4 — Domain Model

[Key entities, their attributes, and relationships. This is not a database schema — it's the conceptual model that requirements operate on.]

## §5 — Interface Contracts

### §5.1 — CLI Interface
[Every command, its arguments, its output format, its exit codes.]

### §5.2 — Data Formats
[Input and output data structures. Concrete examples in the format the system will use (JSON, PEM, etc.).]

## §6 — Behavior Scenarios

### §6.1 — [Feature/Flow Name]
[Scenarios in Given/When/Then format with concrete data.]

### §6.2 — [Feature/Flow Name]
[Scenarios in Given/When/Then format with concrete data.]

(Continue for all features/flows. Include success and failure scenarios.)

## §7 — Error Catalog

[Every expected error condition. For each: trigger condition, error message/code, system behavior after the error.]

## §8 — Assumptions

[Carried forward from RESEARCH.md §6, refined. These are facts the spec treats as true.]

## §9 — Traceability Matrix

| Requirement | Scenarios | Interface |
|-------------|-----------|-----------|
| REQ-XX-NNN  | §6.N      | §5.N      |

[Every requirement must appear. Every requirement must map to at least one scenario.]
```

## Quality Checklist

- [ ] Core principle is restated in §1 (document is self-contained)
- [ ] Every requirement in §3 has a REQ-XX-NNN identifier
- [ ] Every requirement is independently testable (has observable outcome)
- [ ] No ambiguous language ("should", "properly", "correctly", "as expected")
- [ ] §4 domain model covers all entities referenced in requirements
- [ ] §5 interfaces specify exact CLI syntax, arguments, and output format
- [ ] §5 data formats include concrete examples (not just schemas)
- [ ] Every scenario in §6 uses concrete sample data
- [ ] Every scenario has explicit Given (state), When (action), Then (outcome)
- [ ] Both success and failure paths are covered for each interface
- [ ] §7 error catalog covers every error condition mentioned in scenarios
- [ ] §9 traceability matrix has no unmapped requirements
- [ ] Every in-scope item from RESEARCH.md has at least one requirement
- [ ] No empty sections or placeholder text
