---
name: research-document
description: Defines the format, structure, and quality standards for RESEARCH.md documents. Used by the researcher agent in Phase 0 to produce domain research that gives downstream agents complete context.
---

# Research Document Skill

## Step-by-Step Instructions

1. Read the project idea provided by the user.
2. Read PHILOSOPHY.md to understand scope constraints, technology philosophy, and what "small experiment" means.
3. Identify and articulate the **core principle** in 1-2 sentences. This is the single mechanism that makes the project interesting.
4. Research the domain thoroughly — explain every concept a newcomer would need to understand.
5. Define all domain-specific terms on first use.
6. Evaluate candidate programming languages — compare standard library support, ecosystem maturity, and fit for the core principle.
7. Evaluate candidate design patterns — which patterns naturally model the domain, and why.
8. Define scope: what's in (serves the core principle) and what's out (everything else).
9. List all assumptions explicitly.
10. Surface open questions — things the user needs to confirm before specification begins.
11. Write RESEARCH.md following the Output Template below.

## Examples

**Good core principle statement:**
"The core principle is X.509 certificate chain validation — how a relying party traverses a chain of certificates from leaf to root CA, verifying signatures and constraints at each step."

**Bad core principle statement:**
"The core principle is building a CA." (Too vague. What about a CA? What mechanism?)

**Good scope boundary:**
"In scope: certificate generation, signing, chain building, CRL generation, OCSP response serving. Out of scope: HSM integration, LDAP directory publishing, web UI, production key management."

## Common Edge Cases

- The project idea is too broad. Apply the scope test: "If I remove this, does the core principle still demonstrate?" If yes, cut it.
- Multiple core principles are competing. Pick one. The experiment demonstrates one thing well, not three things poorly.
- The user mentions technology preferences. Note them but still evaluate alternatives — the research must be neutral.
- Domain terminology overlaps with common English words. Define every term explicitly even if it seems obvious.

## Output Template

```markdown
# RESEARCH.md — [Project Name]

## §1 — Core Principle

[1-2 sentence statement of the fundamental mechanism this experiment demonstrates.]

## §2 — Domain Background

[Comprehensive explanation of the domain. Every concept a newcomer needs. All terms defined on first use. Written so someone with zero domain knowledge can understand what they're building and why.]

### §2.1 — Key Concepts
[Concept-by-concept breakdown with definitions and relationships.]

### §2.2 — How It Works
[End-to-end explanation of the mechanism. Walk through the core principle step by step.]

## §3 — Technology Evaluation

### §3.1 — Programming Language Candidates
[For each candidate: name, relevant standard library capabilities, ecosystem maturity for this domain, pros, cons, verdict.]

### §3.2 — Design Pattern Candidates
[For each candidate pattern: name, how it maps to the domain, pros, cons, applicability to the core principle.]

### §3.3 — Recommendation
[Recommended language and patterns with brief justification tied to the core principle and philosophy constraints.]

## §4 — Prior Art and References

[Relevant specifications (RFCs, standards), reference implementations, educational resources. Cited specifically, not vaguely.]

## §5 — Scope Definition

### §5.1 — In Scope
[Bulleted list. Each item directly serves demonstrating the core principle.]

### §5.2 — Out of Scope
[Bulleted list with brief rationale for each exclusion.]

### §5.3 — Mock Boundaries
[What will be mocked/stubbed and how. Per philosophy: mock everything that isn't the core principle.]

## §6 — Assumptions

[Numbered list of assumptions. Each one is something the specification will treat as true without further validation.]

## §7 — Open Questions

[Numbered list of questions for the user. Things that need human confirmation before proceeding to specification.]
```

## Quality Checklist

- [ ] Core principle is stated in §1 in 1-2 sentences
- [ ] Every domain term is defined on first use in §2
- [ ] §2.2 walks through the core mechanism end-to-end
- [ ] At least 2 programming languages are evaluated in §3.1 with pros/cons
- [ ] At least 2 design patterns are evaluated in §3.2 with pros/cons
- [ ] §3.3 gives a clear recommendation with justification
- [ ] Prior art references are specific (RFC numbers, URLs, names) not vague
- [ ] Every in-scope item in §5.1 passes the scope test
- [ ] Every out-of-scope item in §5.2 has a rationale
- [ ] Mock boundaries in §5.3 are concrete (what is mocked and how)
- [ ] Assumptions in §6 are falsifiable statements
- [ ] Open questions in §7 are genuine decisions the user must make
- [ ] Document is self-contained — readable with no prior domain knowledge
- [ ] No empty sections or placeholder text
