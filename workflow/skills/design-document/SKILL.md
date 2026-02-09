---
name: design-document
description: Defines the format, structure, and quality standards for DESIGN.md documents. Used by the architect agent in Phase 3 to design the technical solution constrained by specs and contracts.
---

# Design Document Skill

## Step-by-Step Instructions

1. Read RESEARCH.md (especially §3 technology evaluation), SPEC.md (all sections), and CONTRACTS.md (all sections).
2. Choose the technology stack — language, libraries (if any), tools. Justify against philosophy constraints: boring tech, standard library first, minimal dependencies, zero-config.
3. Define the file structure. Keep it flat: 5-15 files, max 2 levels of directory nesting. Name files by what they do (not generic names like service.py or handler.py).
4. Design each component: its responsibility, its public interface (function signatures), its data flow connections to other components.
5. Map each component to the requirements and contracts it satisfies.
6. Check every design decision against CONTRACTS.md — no component may violate any contract.
7. For every non-obvious decision, create an ADR (see adr-document skill).
8. Define the implementation plan as an ordered sequence of steps. Each step should be a discrete, committable unit of work.
9. Build the requirement + contract coverage matrix — every REQ and CON must be covered by at least one component.
10. Write DESIGN.md following the Output Template below.

## Examples

**Good file structure:**
```
ca_service/
├── main.py              ← Entry point: CLI parsing and dispatch
├── ca_engine.py         ← Core CA operations: init, sign, revoke
├── certificate.py       ← Certificate generation and parsing
├── crl_generator.py     ← CRL creation and management
├── ocsp_responder.py    ← OCSP response generation
├── storage.py           ← Certificate and key storage (in-memory/JSON)
├── crypto_utils.py      ← Cryptographic primitives wrapper
└── sample_data/         ← Pre-built sample CSRs and certificates
    └── ...
```

**Bad file structure:**
```
src/services/ca/handlers/certificate_handler.py  ← Too nested, generic name
src/services/ca/middleware/auth_middleware.py      ← Middleware is out of scope
```

**Good implementation step:**
"Step 3: Implement certificate signing (ca_engine.py: sign_certificate). Create feature/certificate-signing branch. Implement the sign_certificate function that takes a CSR and CA credentials, produces a signed X.509 certificate. Enforce CON-INV-01 (issuer matches CA subject) and CON-BND-03 (preconditions). Commit with message referencing REQ-CA-003."

## Common Edge Cases

- The recommended language from research has a critical gap. Document it as an ADR and either work around it or reconsider.
- A component needs to satisfy conflicting contracts. This indicates a design problem — resolve it, don't paper over it.
- The implementation plan has more than 8-10 steps. The scope is likely too broad. Revisit and consolidate.
- A contract can't be enforced by any single component. It may need to be enforced at multiple points — document this explicitly.

## Output Template

```markdown
# DESIGN.md — [Project Name]

## §1 — Technology Stack

| Choice | Selection | Justification |
|--------|-----------|---------------|
| Language | [Name + version] | [Why, tied to philosophy and core principle] |
| Library 1 | [Name] | [Why it's needed, why stdlib isn't sufficient] |
| ... | ... | ... |

[If no external libraries: "No external dependencies. Standard library only."]

## §2 — File Structure

[Tree diagram of all files with one-line descriptions. Include the entry point.]

```
project/
├── file1.ext  ← [description]
├── file2.ext  ← [description]
└── ...
```

## §3 — Component Design

### §3.1 — [Component Name] (`filename.ext`)

**Responsibility:** [What this component does — one sentence]

**Public Interface:**
```
function_name(param: Type, ...) → ReturnType
```

**Contracts Enforced:** CON-XX, CON-XX
**Requirements Served:** REQ-XX-NNN, REQ-XX-NNN

**Data Flow:**
- Receives: [what, from where]
- Produces: [what, consumed by whom]

(Continue for all components.)

## §4 — Data Flow Diagram

[ASCII diagram showing how data moves through the system from input to output.]

## §5 — Error Handling Strategy

[How errors are handled. Map to SPEC.md §7 error catalog. Keep it simple — handle expected cases per the spec, not every edge case.]

## §6 — Implementation Plan

[Ordered steps. Each step is a committable unit of work with a clear deliverable.]

| Step | Branch | Description | Files | Requirements | Contracts |
|------|--------|-------------|-------|-------------|-----------|
| 1 | feature/xxx | [what to build] | [files] | REQ-XX-NNN | CON-XX |
| 2 | feature/xxx | [what to build] | [files] | REQ-XX-NNN | CON-XX |

## §7 — Mock Strategy

[What is mocked, how, and why. Per philosophy: mock everything that isn't the core principle.]

## §8 — ADR Summary

[List of all ADRs produced during this phase with one-line descriptions.]

| ADR | Title | Decision |
|-----|-------|----------|
| ADR-001 | [title] | [one-line summary of the choice] |

## §9 — Requirement and Contract Coverage

### §9.1 — Requirement Coverage
| Requirement | Component(s) | Implementation Step |
|-------------|-------------|-------------------|
| REQ-XX-NNN  | §3.N        | Step N            |

### §9.2 — Contract Coverage
| Contract | Component(s) | Enforcement Mechanism |
|----------|-------------|----------------------|
| CON-XX   | §3.N        | [assertion/validation/check] |

[Every REQ and CON must appear. Flag any gaps.]
```

## Quality Checklist

- [ ] Technology stack justified against philosophy constraints (boring, minimal, zero-config)
- [ ] File structure is flat (5-15 files, max 2 directory levels)
- [ ] Files named by function, not by pattern (no "service", "handler", "manager")
- [ ] Every component has explicit public interface with function signatures
- [ ] Every component maps to requirements and contracts
- [ ] No component violates any contract from CONTRACTS.md
- [ ] Data flow diagram shows complete input-to-output pipeline
- [ ] Implementation plan is ordered and each step is a committable unit
- [ ] Implementation plan steps map to feature branches per git-flow
- [ ] ADRs exist for all non-obvious decisions
- [ ] §9 coverage matrix has no unmapped requirements or contracts
- [ ] Mock strategy covers everything out of scope per philosophy
- [ ] No empty sections or placeholder text
