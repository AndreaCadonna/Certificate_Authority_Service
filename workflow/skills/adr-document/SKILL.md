---
name: adr-document
description: Defines the format, structure, numbering, and quality standards for Architecture Decision Records (ADRs). Used by the architect agent in Phase 3 to document non-obvious design decisions with full rationale, alternatives, and tradeoffs.
---

# ADR Document Skill

## Step-by-Step Instructions

1. During design, identify every non-obvious decision — technology choices, pattern selections, intentional exclusions, security decisions, deviations from standards.
2. For each decision, create an ADR file in `ADRs/` using the naming format `ADR-NNN-short-descriptive-title.md`.
3. Number sequentially starting from 001.
4. Use the template from `references/adr-template.md`.
5. Fill every section — no empty sections. If "Neutral consequences" is truly empty, write "None identified."
6. Reference specific REQ-XX-NNN and CON-XX identifiers in the References section.
7. Be specific in Alternatives Considered — name the alternatives and explain why they were rejected with concrete reasoning, not vague preferences.
8. Status should be "Accepted" for new ADRs. Use "Superseded by ADR-XXX" only when revising a previous decision.

## When to Create an ADR

Create an ADR when:
- A technology or framework is chosen over alternatives
- A design pattern is adopted that has meaningful tradeoffs
- A security decision constrains future implementation
- A standard or specification is intentionally deviated from
- A feature or capability is intentionally excluded
- A decision might be questioned or reversed without understanding the reasoning

Do NOT create an ADR for:
- Obvious choices with no meaningful alternatives
- Decisions that are trivially reversible with no impact
- Implementation details that don't affect architecture

## Examples

**Good ADR title:** `ADR-001-python-cryptography-library-over-stdlib.md`
**Bad ADR title:** `ADR-001-tech-choice.md` (Too vague)

**Good Alternatives Considered:**
"We considered using Python's built-in `ssl` module. It provides certificate loading and TLS context management but lacks X.509 certificate generation, CSR creation, and CRL building capabilities. These are core to the experiment, making `ssl` alone insufficient."

**Bad Alternatives Considered:**
"We considered other options but they weren't as good." (No substance)

## Common Edge Cases

- A decision seems obvious to you but wouldn't be obvious to a developer unfamiliar with the domain. Write the ADR — the audience is a future agent with no context.
- Two decisions are closely related. Write separate ADRs — each should be independently understandable.
- A decision is forced by a contract constraint. Still write the ADR — document that the decision was constrained, not freely chosen.

## Output Template

See `references/adr-template.md` for the exact template.

## Quality Checklist

- [ ] File naming follows `ADR-NNN-short-descriptive-title.md` format
- [ ] Title is specific and descriptive
- [ ] Status is set (Accepted / Superseded by ADR-XXX / Deprecated)
- [ ] Context explains the situation and problem clearly
- [ ] Decision is specific — names exact technologies, patterns, or choices
- [ ] Alternatives Considered lists real alternatives with concrete rejection reasons
- [ ] Positive consequences describe what the decision enables
- [ ] Negative consequences honestly describe accepted tradeoffs
- [ ] References link to specific REQ-XX-NNN and/or CON-XX identifiers
- [ ] No empty sections (write "None identified" if truly empty)
- [ ] ADR is self-contained — readable without other ADRs
