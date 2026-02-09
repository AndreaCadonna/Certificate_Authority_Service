# Project Philosophy

How we think about, scope, and build software experiments using agentic workflows.

This document defines the nature, constraints, and goals of projects built with this system. Give it to the agent alongside the workflow files and the Builder's Guide so it can make autonomous decisions about scope, technology, architecture, and testing without asking you.

---

## What We're Building

These projects are **experiments**. They exist to test whether a concept works, to learn how something is built, and to demonstrate a core principle in running code. They are not products, not prototypes, not MVPs. They are the software equivalent of a lab experiment: controlled, focused, and designed to answer a specific question.

The question is always some variation of: **"Can this concept be built, and does it work as expected?"**

---

## The Core Principle Rule

Every project has one **core principle** — the fundamental mechanism, algorithm, or pattern that makes it interesting. Everything else is scaffolding.

Before writing a single line of code, the agent must identify and articulate this core principle. It should be expressible in one or two sentences. If you can't say what the core principle is, the research phase isn't done.

**Examples:**
- A search engine experiment → core principle is **inverted index construction and query matching**
- A blockchain experiment → core principle is **hash-linked blocks with proof-of-work consensus**
- A rate limiter experiment → core principle is **token bucket algorithm with sliding window tracking**

The core principle determines what gets built and what gets cut. Every feature must directly serve demonstrating this principle. If a feature doesn't contribute to understanding or showcasing the core mechanism, it's out of scope regardless of how important it would be in a real system.

---

## Scope Philosophy

### Default to cutting, not adding

The natural instinct of both humans and agents is to expand scope. Resist it. A working demonstration of one concept is worth more than a half-built attempt at five.

### Treat big ideas as small projects

Even if the project idea describes a system that would be massive in the real world (a database engine, an operating system scheduler, a recommendation system), we treat it as a pet project. We're not building the real thing. We're building the smallest possible version that demonstrates why the real thing works.

### What "small" means concretely

- **Single-session implementation.** The coding phase should be completable in one focused agent session. If the design calls for more than 15-20 files, scope is too broad.
- **One core feature set.** Not multiple independent features. One cohesive mechanism with 3-5 testable behaviors.
- **Minutes to understand, not hours.** Someone reading the codebase should grasp the structure and purpose within 10-15 minutes.

### The scope test

Ask: "If I remove this feature, does the core principle still demonstrate correctly?"
- Yes → remove it
- No → keep it

---

## What We Build vs. What We Skip

### Always in scope

- The core mechanism/algorithm
- Enough data structures to represent the domain entities
- A CLI or programmatic interface to exercise the system
- Concrete behavior scenarios with real-looking sample data
- Automated validation that proves the system works
- A narrated demo that shows the system working

### Always out of scope

- User authentication and authorization
- Production-grade error handling (we handle the expected cases, not every edge case)
- Deployment, containerization, cloud configuration
- UI/frontend (unless the UI IS the core principle)
- Performance optimization
- Logging frameworks, monitoring, telemetry
- Database migrations or schema versioning
- API versioning
- Rate limiting (unless it IS the core principle)
- Internationalization/localization
- Configuration management systems

### Mock aggressively

Anything that isn't the core principle but is needed for the system to function should be mocked or stubbed:

- External APIs → hardcoded responses or local mock servers
- Databases → in-memory data structures or JSON files
- File systems → local temp directories with sample data
- Network calls → deterministic fake responses
- Time-dependent behavior → fixed timestamps or controllable clocks
- User input in complex formats → pre-built sample files

The mock should be realistic enough to demonstrate the behavior but simple enough to not become a project of its own.

---

## Technology Philosophy

### Boring is better

Choose mature, well-documented, widely-known technology. No bleeding-edge tools, niche frameworks, or experimental libraries. The experiment is the software concept, not the technology stack.

### Minimize everything

- **Dependencies:** Standard library first. Every external dependency must justify its existence. If the standard library can do it in 10 more lines of code, use the standard library.
- **Frameworks:** Avoid them unless the core principle requires one. A plain script is better than a framework for an experiment.
- **Abstraction layers:** Don't build them. Direct function calls, explicit data passing, no middleware.
- **Configuration:** Zero-config. One command to install, one command to run. No environment variables, no config files, no Docker (unless containerization IS the experiment).

### Architecture for clarity, not scalability

- **Flat file structures.** 5-15 files, rarely more than 2 levels of directory nesting.
- **One entry point.** One file that reads like a script: parse input, do the work, produce output.
- **Explicit data flow.** Follow a function call and you can trace the entire pipeline. No magic, no dependency injection, no event buses.
- **Components named by what they do.** `tokenizer.py`, `index_builder.py`, `query_engine.py` — not `service.py`, `handler.py`, `manager.py`.

### CLI is the default interface

Unless the spec says otherwise, the system is a command-line tool. CLIs are easy to build, easy to test, easy to automate, and easy to demonstrate. They don't need a browser, a GUI framework, or a server.

Design for automation: predictable output format (JSON preferred for structured data), non-interactive execution, meaningful exit codes.

---

## Testing Philosophy

### No unit tests

We don't write unit tests. These are experiments, and the overhead of a test framework, test organization, mocks within tests, and test maintenance is not justified.

### Behavioral validation instead

We validate by running the system end-to-end against concrete scenarios. Each scenario comes directly from the specification:

- **Given** a specific starting state and input
- **When** the system processes it
- **Then** the output matches what the spec predicted

This is captured in an automated validation script (`validate.sh` or equivalent) that:
- Sets up the environment from scratch
- Runs each scenario
- Compares actual output to expected output
- Reports pass/fail per scenario
- Requires zero human intervention

### Validation is not testing — it's proof

The validation script is evidence that the experiment works. It answers the question: "Does this code do what the spec says it should do?" Every behavior in the spec must have at least one validation scenario. If a behavior can't be validated automatically, the spec needs to be rewritten to make it observable.

### Demo is not validation — it's storytelling

Separately from validation, a demo script (`demo.sh` or equivalent) walks through the system's capabilities with narrated output. It uses different data than validation (to prove the system generalizes), and it tells a story: "Here's the problem. Here's how this system solves it. Watch."

The demo should be runnable by someone who has never seen the project. It should make the core principle click.

---

## Knowledge and Learning Philosophy

### The user may know nothing

The person starting these experiments may have little to no domain knowledge about the concept being built. The research phase exists to close this gap. The research document should be self-contained enough that someone unfamiliar with the domain can read it and understand what they're building and why.

### Each phase is self-contained

An agent picking up any phase's output must be able to execute the next phase with no additional context. This means:

- Research documents define every domain term
- Specifications include concrete examples, not just abstract behaviors
- Designs include exact function signatures and file structures
- Implementation documents include copy-paste-ready setup instructions

No "you'll figure it out" hand-waving between phases.

### The agent's reasoning is visible

When an agent makes a decision (scoping a feature out, choosing a technology, simplifying a design), the reasoning is documented in the artifact. Assumptions are listed explicitly. Deviations from upstream documents are logged with rationale. Open questions surface gaps instead of hiding them.

---

## Quality Bar

### For code

- It runs from a clean environment using only the documented setup instructions
- It follows the chosen language's standard conventions
- Comments explain WHY, not WHAT
- File-level comments trace back to spec requirements
- No dead code, no debug artifacts, no TODO comments
- Error messages are clear and actionable for the cases the spec defines

### For documents

- Every section is populated — no empty placeholders
- Every term is defined on first use
- Every behavior has a concrete example with real data
- Every decision is justified (even if briefly)
- Structure follows the skill template exactly — no added, removed, or renamed sections

### For the experiment as a whole

The ultimate quality test is a single question: **Could a stranger clone this repo, run the setup commands, execute the demo, and understand what the core principle is and that it works?**

If yes, the experiment succeeds. If no, something needs to be clearer, simpler, or better documented.

---

## Summary of Constraints

| Constraint | Rule |
|-----------|------|
| **Scope** | One core principle per project. Cut everything that doesn't serve it. |
| **Size** | Single-session implementation. 5-15 files. 3-5 testable behaviors. |
| **Technology** | Boring, minimal, zero-config. Standard library first. |
| **Architecture** | Flat, explicit, one entry point. CLI default. |
| **Dependencies** | Justify every one. Prefer none. |
| **External systems** | Mock everything that isn't the core principle. |
| **Testing** | Behavioral validation scripts, not unit tests. |
| **Deployment** | None. Local execution only. |
| **Interface** | CLI unless the spec says otherwise. |
| **Documentation** | Self-contained. Every phase readable without prior context. |
| **Code style** | Language conventions. Comments explain why. Requirements traced. |
| **User knowledge** | Assume none. Research and documents fill the gap. |
