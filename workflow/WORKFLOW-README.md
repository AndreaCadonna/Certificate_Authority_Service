# Agentic Spec-Driven Development Workflow

## Overview

A sequential pipeline for building software entirely through AI agents. Each phase produces artifacts that feed the next. Each phase can be executed by an independent agent with no shared memory.

The core pipeline is 7 phases (0–6). Phase 7 is a conditional fix cycle that runs only when validation finds failures.

The pipeline is built on three foundational layers that give agents maximum context with minimum ambiguity:

- **Specs** define what the system does (behavior and flows).
- **Contracts** define what must always be true (invariants and boundaries).
- **ADRs** define why decisions were made (rationale and tradeoffs).

Together, these layers form a traceability chain from project idea to validated codebase.

---

## The Pipeline

```
You (project idea)
 │
 ▼
Phase 0: Research ──────────→ RESEARCH.md
 │
 ▼ (you confirm scope)
Phase 1: Specification ─────→ SPEC.md
 │
 ▼ (you confirm spec)
Phase 2: Contracts ─────────→ CONTRACTS.md
 │
 ▼ (you confirm contracts)
Phase 3: Design ────────────→ DESIGN.md + ADRs/
 │
 ▼ (you confirm design)
Phase 4: Implementation ────→ Codebase + IMPLEMENTATION.md + Git repo
 │
 ▼
Phase 5: Validation ────────→ validate.sh + demo.sh + VALIDATION_REPORT.md
 │
 ├── ✅ All pass ──→ Done (main merge + tag)
 │
 └── ❌ Failures found
      │
      ▼
     Phase 7a: Fix ──→ Fix Plan (you approve) ──→ Fixed codebase + FIX_REPORT.md
      │
      ▼
     Phase 7b: Re-Validation ──→ Updated VALIDATION_REPORT.md + Final git state
      │
      ├── ✅ All pass ──→ Done (main merge + tag)
      └── ❌ Remaining ──→ You decide: accept partial / return to Phase 1-3 / another fix cycle
```

---

## The Three Foundational Layers

The pipeline is powered by three artifact layers that operate at different levels of abstraction. Understanding their separation of concerns is critical.

### Specs — "What are we building and how does it behave?"

Specs live at the behavioral and feature level. They describe workflows, user stories, API shapes, expected outcomes, success and failure cases. Specs are the **what and why** — oriented toward product intent and system design.

A spec says: *"Users can transfer money between accounts. Here is the flow, the inputs, the outputs, the error cases."*

### Contracts — "What must always be true?"

Contracts live at the function, module, and interface level. They define preconditions, postconditions, and invariants. Contracts are the **guarantees and constraints** — oriented toward correctness and defensive boundaries.

A contract says: *"No matter what, a transferred amount must be positive, the sender must have sufficient balance, and total money in the system must remain constant."*

Contracts are not requirements and not design. They are the constraints that sit between the two. Requirements say what to build. Contracts say what must never break while building it.

### ADRs — "Why did we make this choice?"

Architecture Decision Records capture the reasoning behind non-obvious design decisions. They record what was chosen, what was rejected, what tradeoffs were accepted, and why. ADRs are the **context and rationale** — oriented toward preventing future agents or developers from undoing intentional decisions.

An ADR says: *"We chose JWTs for access tokens because resource servers need to validate without calling back to the auth server. We accepted the tradeoff that revocation requires short token lifetimes."*

### How the Layers Work Together

```
        ┌─────────────┐
        │    SPECS     │  ← What are we building and how does it behave?
        ├─────────────┤
        │  CONTRACTS   │  ← What must always be true?
        ├─────────────┤
        │    ADRs      │  ← Why did we make these choices?
        └─────────────┘
```

When an agent implements a feature, it reads the spec to understand the flow, reads the contracts to know the hard boundaries, and reads the ADRs to understand why the design looks the way it does. Without all three, the agent builds something that sort of works but makes silent, dangerous assumptions.

- **Spec without contracts**: The agent builds the right feature but introduces subtle bugs — negative transfers, race conditions, broken invariants. You catch them late.
- **Contracts without specs**: The code is locally correct but doesn't compose into the right product behavior. Every function is safe but the system does the wrong thing.
- **Spec and contracts without ADRs**: A future agent "simplifies" something that was intentionally complex, undoing a security decision or architectural tradeoff.

---

## Architecture: Agents, Prompts, and Skills

This workflow separates execution concerns into three primitives:

| Primitive | What It Is | What It Contains | Analogy |
|-----------|-----------|-----------------|---------|
| **Agent** | A persistent identity | Role, mission, behavioral rules, decision framework, anti-patterns | WHO does the work |
| **Prompt** | A one-shot task | Context, inputs, steps, output reference, exit criteria | WHAT to do right now |
| **Skill** | Reusable knowledge | Conventions, formats, templates, quality checklists | HOW to do it well |

### How They Compose at Runtime

```
┌─────────────────────────────────────────────┐
│ SYSTEM PROMPT / TOP OF CONTEXT              │
│                                             │
│  1. [agent].agent.md    (identity/behavior) │
│  2. [skill-1].skill.md  (domain knowledge)  │
│  3. [skill-2].skill.md  (domain knowledge)  │
│                                             │
├─────────────────────────────────────────────┤
│ USER MESSAGE                                │
│                                             │
│  4. [phase].prompt.md   (task + inputs)     │
│     └── pasted upstream artifacts           │
│                                             │
└─────────────────────────────────────────────┘
```

---

## File Inventory

Skills follow the [Agent Skills open standard](https://agentskills.io/specification). Each skill is a directory containing a `SKILL.md` file with YAML frontmatter and optional `scripts/`, `references/`, and `assets/` subdirectories. This makes skills portable across any compatible agent platform (Claude Code, Codex, Cursor, GitHub Copilot, and others).

```
workflow/
├── WORKFLOW-README.md
│
├── agents/
│   ├── researcher.agent.md                   ← Phase 0
│   ├── spec-writer.agent.md                  ← Phase 1
│   ├── contract-writer.agent.md              ← Phase 2
│   ├── architect.agent.md                    ← Phase 3
│   ├── developer.agent.md                    ← Phase 4
│   ├── qa-engineer.agent.md                  ← Phase 5 + 7b
│   └── fixer.agent.md                        ← Phase 7a
│
├── prompts/
│   ├── research.prompt.md                    ← Phase 0
│   ├── specification.prompt.md               ← Phase 1
│   ├── contracts.prompt.md                   ← Phase 2
│   ├── design.prompt.md                      ← Phase 3
│   ├── implementation.prompt.md              ← Phase 4
│   ├── validation.prompt.md                  ← Phase 5
│   ├── fix.prompt.md                         ← Phase 7a
│   └── re-validation.prompt.md               ← Phase 7b
│
└── skills/
    ├── research-document/
    │   └── SKILL.md
    ├── spec-document/
    │   └── SKILL.md
    ├── contracts-document/
    │   └── SKILL.md
    ├── design-document/
    │   └── SKILL.md
    ├── adr-document/
    │   ├── SKILL.md
    │   └── references/
    │       └── adr-template.md
    ├── implementation-document/
    │   └── SKILL.md
    ├── validation-report/
    │   ├── SKILL.md
    │   └── scripts/
    │       └── validate-template.sh
    ├── fix-report/
    │   └── SKILL.md
    ├── git-flow/
    │   └── SKILL.md
    └── code-quality/
        └── SKILL.md
```

---

## Assembly Guide — What to Load Per Phase

| Phase | Agent | Skills | Prompt |
|-------|-------|--------|--------|
| 0 — Research | `researcher.agent.md` | `research-document/` | `research.prompt.md` |
| 1 — Specification | `spec-writer.agent.md` | `spec-document/` | `specification.prompt.md` |
| 2 — Contracts | `contract-writer.agent.md` | `contracts-document/` | `contracts.prompt.md` |
| 3 — Design | `architect.agent.md` | `design-document/` + `adr-document/` + `git-flow/` | `design.prompt.md` |
| 4 — Implementation | `developer.agent.md` | `code-quality/` + `git-flow/` + `implementation-document/` | `implementation.prompt.md` |
| 5 — Validation | `qa-engineer.agent.md` | `validation-report/` + `git-flow/` | `validation.prompt.md` |
| 7a — Fix | `fixer.agent.md` | `fix-report/` + `code-quality/` + `git-flow/` | `fix.prompt.md` |
| 7b — Re-Validation | `qa-engineer.agent.md` | `validation-report/` + `git-flow/` | `re-validation.prompt.md` |

### Step-by-Step: Running a Phase

1. **Assemble context**: Paste the agent file, then the skill file(s), as system prompt or at the top of context.
2. **Fill in the prompt**: Replace placeholders with your project idea (Phase 0) or upstream artifacts (Phase 1+).
3. **Send the prompt** as the user message.
4. **Review the output** against the skill's quality checklist.
5. **Gate check**: Confirm before proceeding to next phase (see Your Role below).

---

## Artifact Flow

```
Phase 0 Input:  Project idea (from you)
Phase 0 Output: RESEARCH.md

Phase 1 Input:  RESEARCH.md
Phase 1 Output: SPEC.md

Phase 2 Input:  SPEC.md
Phase 2 Output: CONTRACTS.md

Phase 3 Input:  RESEARCH.md + SPEC.md + CONTRACTS.md
Phase 3 Output: DESIGN.md + ADRs/

Phase 4 Input:  RESEARCH.md + SPEC.md + CONTRACTS.md + DESIGN.md + ADRs/ + Remote URL
Phase 4 Output: Codebase + IMPLEMENTATION.md + Git repo

Phase 5 Input:  SPEC.md + CONTRACTS.md + DESIGN.md + ADRs/ + IMPLEMENTATION.md + Codebase
Phase 5 Output: validate.sh + demo.sh + VALIDATION_REPORT.md + Git state

(If validation has failures)

Phase 7a Input:  SPEC.md + CONTRACTS.md + ADRs/ + DESIGN.md + IMPLEMENTATION.md + VALIDATION_REPORT.md + Codebase
Phase 7a Output: Fix Plan (user gate) → Fixed codebase + FIX_REPORT.md

Phase 7b Input:  SPEC.md + CONTRACTS.md + VALIDATION_REPORT.md (original) + FIX_REPORT.md + Fixed codebase
Phase 7b Output: Updated VALIDATION_REPORT.md + Final git state
```

---

## CONTRACTS.md — Structure

CONTRACTS.md is derived from SPEC.md during Phase 2. It formalizes the invariants and boundary rules that cut across all specifications. The contract-writer agent reads the spec and extracts the non-negotiable truths that must hold regardless of implementation.

### Sections

**§1 — System Invariants**: Global truths that must never be violated across the entire system. These are properties that hold at all times, not just within a single operation.

**§2 — Boundary Contracts**: Per-interface preconditions and postconditions. Each contract traces back to one or more requirements (REQ-XX-NNN) from SPEC.md. Structured as:
- Function or endpoint name
- Preconditions (what must be true before execution)
- Postconditions (what must be true after execution)
- Error conditions (what happens when preconditions are violated)

**§3 — Security Contracts**: Non-negotiable security boundaries that no implementation decision can weaken. These are hard constraints, not guidelines.

**§4 — Data Integrity Contracts**: Rules governing data consistency, storage, and lifecycle. Includes constraints on data formats, retention, and state transitions.

**§5 — Traceability**: Every contract (CON-XX) maps back to one or more requirements (REQ-XX-NNN) from SPEC.md. This ensures no contract exists without justification and no critical requirement lacks a contract.

### Contract Identifier Format

Contracts use the format `CON-XX` where XX is a sequential number. Categories can be prefixed for clarity:

- `CON-INV-XX` — System invariants
- `CON-BND-XX` — Boundary contracts
- `CON-SEC-XX` — Security contracts
- `CON-DAT-XX` — Data integrity contracts

---

## ADRs/ — Structure

ADRs are produced during Phase 3 (Design) as a natural byproduct of architectural decisions. Every non-obvious choice the architect agent makes becomes a numbered ADR document.

### File Naming

```
ADRs/
├── ADR-001-[short-descriptive-title].md
├── ADR-002-[short-descriptive-title].md
├── ADR-003-[short-descriptive-title].md
└── ...
```

### ADR Template

Each ADR follows this structure:

```
# ADR-NNN: [Title]

## Status
Accepted | Superseded by ADR-XXX | Deprecated

## Context
What situation, requirement, or constraint forced this decision?
What problem are we solving?

## Decision
What did we choose? Be specific.

## Alternatives Considered
What other options were evaluated and why were they rejected?

## Consequences

### Positive
What does this decision enable?

### Negative
What tradeoffs did we accept?
What becomes harder because of this choice?

### Neutral
What changes but is neither good nor bad?

## References
- REQ-XX-NNN from SPEC.md
- CON-XX from CONTRACTS.md (if applicable)
- External references (RFCs, security advisories, etc.)
```

### When to Create an ADR

The architect agent should create an ADR whenever:

- A technology or framework is chosen over alternatives.
- A design pattern is adopted that has meaningful tradeoffs.
- A security decision is made that constrains future implementation.
- A standard or specification is intentionally deviated from.
- A feature or capability is intentionally excluded.
- A decision is made that future developers might question or reverse without understanding the reasoning.

### When NOT to Create an ADR

- Obvious choices with no meaningful alternatives.
- Decisions that are trivially reversible with no impact.
- Implementation details that don't affect architecture.

---

## How Each Agent Uses the Three Layers

### Spec-Writer (Phase 1)
Reads RESEARCH.md. Produces SPEC.md with requirements, interface definitions, behavior scenarios, and traceability matrix. This is the behavioral blueprint.

### Contract-Writer (Phase 2)
Reads SPEC.md. Derives CONTRACTS.md by extracting invariants, boundary rules, and security constraints from the specification. Does not invent new requirements — only formalizes the non-negotiable truths implied by the spec.

### Architect (Phase 3)
Reads RESEARCH.md + SPEC.md + CONTRACTS.md. Produces DESIGN.md and ADRs/. Every architectural decision is checked against contracts to ensure the design doesn't violate them. Non-obvious decisions are captured as ADRs with full rationale.

### Developer (Phase 4)
Reads all upstream artifacts. Implements the system. Uses contracts to write assertions, validation logic, and defensive checks in code. Uses ADRs to understand why the design looks the way it does, avoiding "simplifications" that would undo intentional decisions.

### QA Engineer (Phase 5)
Reads SPEC.md + CONTRACTS.md + ADRs/. Validates that every contract is enforced in the codebase. Contract violations become test failures. ADRs provide context for understanding whether observed behavior is intentional or a bug.

### Fixer (Phase 7a)
Reads CONTRACTS.md + ADRs/ + VALIDATION_REPORT.md. This is where contracts and ADRs provide the most value. The fixer agent has the least context of any agent in the pipeline. Contracts tell it what boundaries the fix must respect. ADRs tell it which design decisions are intentional and must not be "fixed."

---

## Git Flow

```
Phase 4 creates:
  main ← initial commit
  develop ← branched from main
  feature/* ← one per task, merged into develop

Phase 5 adds:
  feature/validation ← scripts + report, merged into develop
  (if all pass) develop → main ← final merge + v0.1.0 tag

Phase 7a adds (if validation had failures):
  fix/validation-fixes ← one commit per root cause, merged into develop

Phase 7b adds:
  feature/re-validation ← updated report, merged into develop
  (if all pass) develop → main ← final merge + v0.1.0 tag
```

**Your manual step**: Before Phase 4, create the remote repository and provide the URL.

---

## Your Role at Each Gate

| After Phase | Your Action | Time |
|-------------|-------------|------|
| 0 — Research | Confirm scope (§5), answer open questions (§7) | 5 min |
| 1 — Specification | Review traceability matrix (§9), spot-check scenarios (§6) | 10 min |
| 2 — Contracts | Review invariants (§1), check security contracts (§3), verify traceability (§5) | 5 min |
| 3 — Design | Glance at tech stack (§1), skim implementation plan (§6), review ADR titles and decisions | 5 min |
| 4 — Implementation | Check scenario results (§6 of IMPLEMENTATION.md) | 2 min |
| 5 — Validation | Read the verdict (§6 of VALIDATION_REPORT.md). If failures → proceed to Phase 7 | 2 min |
| 7a — Fix (plan) | **Review and approve the Fix Plan** before execution. This is a hard gate. | 5 min |
| 7a — Fix (done) | Check FIX_REPORT.md §4 (post-fix state) | 2 min |
| 7b — Re-Validation | Read updated verdict. Decide: accept / return to Phase 1-3 / another cycle | 2 min |

---

## Traceability Chain

```
Project Idea
 → RESEARCH.md §5.1 (Scope Items)
   → SPEC.md §3 (Requirements: REQ-XX-NNN)
     → SPEC.md §5 (Interface Contracts)
       → SPEC.md §6 (Behavior Scenarios)
         → SPEC.md §9 (Traceability Matrix)
           → CONTRACTS.md §1-4 (Invariants + Boundaries + Security + Data: CON-XX)
             → CONTRACTS.md §5 (Contract ↔ Requirement Traceability)
               → ADRs/ (Decisions: ADR-NNN, referencing REQ + CON)
                 → DESIGN.md §3 (Components, constrained by contracts)
                   → DESIGN.md §6 (Implementation Steps)
                     → DESIGN.md §9 (Requirement + Contract Coverage)
                       → Codebase (working code, enforcing contracts)
                         → Git history (commit references to requirements + contracts)
                           → validate.sh (automated proof)
                             → VALIDATION_REPORT.md §3 (Requirement + Contract Coverage)
                               → VALIDATION_REPORT.md §6 (Verdict)
```

---

## Scaling

| Scenario | Adaptation |
|----------|-----------|
| **Large project** | Run all phases fully |
| **Small experiment** | Combine Phase 0+1 into one session, combine Phase 3+4 into one session. Contracts can be a section within SPEC.md rather than a separate document |
| **Adding a feature** | Start at Phase 1 — update SPEC.md, then run Phases 2-5 as deltas |
| **Fixing a bug** | Start at Phase 5 — re-run validation, then Phase 7a-7b to fix |
| **Validation failures** | Run Phase 7a (fix) → 7b (re-validate). Single pass. If still failing, decide: accept partial, fix upstream (Phase 1-3), or run another 7a→7b cycle |
| **Different tech stack** | Only Phase 3 changes — rewrite DESIGN.md + new ADRs, re-run Phases 4-5. Contracts are unchanged because they are implementation-agnostic |
| **Revisiting a decision** | Supersede the relevant ADR with a new one. Update DESIGN.md accordingly |
| **New skill needed** | Create a skill directory in `skills/` following the Agent Skills open standard, reference it in the relevant prompt |
| **Reuse agent for different task** | Same agent, different prompt, potentially different skills |

---

## Adding New Skills

Skills follow the [Agent Skills open standard](https://agentskills.io/specification), an open format maintained by Anthropic and adopted across the industry (Claude Code, OpenAI Codex, Cursor, GitHub Copilot, and others). This makes skills portable — write once, use with any compatible agent.

### Directory Structure

A skill is a directory containing at minimum a `SKILL.md` file:

```
skill-name/
├── SKILL.md              ← Required (frontmatter + instructions)
├── scripts/              ← Optional: executable code agents can run
├── references/           ← Optional: additional docs loaded on demand
└── assets/               ← Optional: templates, data files, schemas
```

### SKILL.md Format

The `SKILL.md` file must contain YAML frontmatter followed by Markdown content:

```markdown
---
name: skill-name
description: |
  What this skill does and when agents should use it.
  Include keywords that help agents match the skill to relevant tasks.
license: Apache-2.0
metadata:
  author: your-name-or-org
  version: "1.0"
---

# Skill Name

## Step-by-Step Instructions
[Imperative steps with explicit inputs and outputs]

## Examples
[Examples of inputs and expected outputs]

## Common Edge Cases
[Known pitfalls and how to handle them]

## Output Template (if applicable)
[Exact structure/skeleton of what gets produced]

## Quality Checklist
[Self-review criteria — checkbox format]
```

### Frontmatter Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Max 64 chars. Lowercase letters, numbers, hyphens only. Must match the parent directory name. |
| `description` | Yes | Max 1024 chars. Describes what the skill does and when to use it. Include trigger keywords. |
| `license` | No | License name or reference to a bundled license file. |
| `compatibility` | No | Max 500 chars. Environment requirements (system packages, network access, etc.). |
| `metadata` | No | Arbitrary key-value pairs for additional metadata (author, version, etc.). |
| `allowed-tools` | No | Space-delimited list of pre-approved tools the skill may use. Experimental. |

### Progressive Disclosure

Skills are structured for efficient use of context:

1. **Metadata** (~100 tokens): The `name` and `description` fields are loaded at startup for all skills.
2. **Instructions** (<5000 tokens recommended): The full `SKILL.md` body is loaded when the skill is activated.
3. **Resources** (as needed): Files in `scripts/`, `references/`, and `assets/` are loaded only when required.

Keep your main `SKILL.md` under 500 lines. Move detailed reference material to separate files in `references/`.

### Naming Conventions

- Directory name must match the `name` field in frontmatter.
- Use lowercase letters, numbers, and hyphens only.
- No consecutive hyphens (`--`), no leading or trailing hyphens.

Valid: `research-document`, `git-flow`, `code-quality`
Invalid: `Research-Document`, `-git-flow`, `code--quality`

### Adding a Skill to the Workflow

1. Create a new directory in `skills/` following the naming conventions.
2. Create a `SKILL.md` with valid frontmatter and instructions.
3. Optionally add `scripts/`, `references/`, or `assets/` subdirectories.
4. Reference the skill in the relevant `prompt.md` under the Skills section.
5. Add the skill to the Assembly Guide table for the appropriate phase(s).

---

## Adding New Agents

Agents follow a standard structure:

```markdown
# [Agent Name] — Agent

## Role
[One sentence: who this agent is]

## Mission
[What this agent is responsible for producing]

## Behavioral Rules
[How this agent should approach its work — numbered for reference]

## Decision Framework
[How this agent should make choices when facing ambiguity]

## Anti-Patterns
[What this agent should never do — explicit failure modes to avoid]

## Inputs
[What this agent receives and from which upstream phases]

## Outputs
[What this agent produces and where it goes next]
```

---

## Design Principles

1. **Agents are stateless.** No agent retains memory between sessions. All context comes from artifacts.
2. **Artifacts are the memory.** Every decision, requirement, constraint, and rationale is captured in a document that persists across phases.
3. **Three layers reduce ambiguity.** Specs provide direction. Contracts provide safety. ADRs provide context. Together, they give any agent enough signal to act autonomously with high confidence.
4. **You are the gatekeeper.** You review, confirm, and approve. You don't author. The agents produce, you verify.
5. **Traceability is non-negotiable.** Every requirement traces to a contract. Every contract traces to a test. Every decision traces to a rationale. If you can't trace it, it doesn't exist.
6. **Contracts are implementation-agnostic.** They survive technology changes, refactors, and redesigns. A contract that says "authorization codes are single-use" is true regardless of whether you use JWTs, opaque tokens, PostgreSQL, or Redis.
7. **ADRs prevent regression.** The most dangerous changes are the ones that undo intentional decisions. ADRs make the cost of reversal visible before it happens.
