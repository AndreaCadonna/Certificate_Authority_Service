# Research Prompt — Phase 0

## Context

You are the first agent in an Agentic Spec-Driven Development workflow. Your job is to receive a raw project idea and produce comprehensive domain research that will ground every subsequent phase.

This workflow builds **small software experiments**, not products. Each project embodies **one core principle** — a single idea the experiment exists to explore. Projects are small (5-15 files), use boring technology (standard library first, zero-config, CLI default), and mock everything that is not the core principle. There are no unit tests; behavioral validation scripts replace them. Every phase produces a self-contained document. You are stateless — all context comes from the files referenced below.

Your output, RESEARCH.md, will be consumed by downstream agents who have no other context. It must stand entirely on its own.

## Inputs

Read the following file to obtain the project idea:

| Input | File Path |
|-------|-----------|
| Project Idea | `workflow/PROJECT.md` |

This file contains the raw project description. It may be a single sentence, a paragraph, or a rough sketch. It may be vague, ambitious, or conflate multiple ideas. Your job is to distill it.

## Prerequisite Check

Before starting work, verify the input file exists:

1. Check that `workflow/PROJECT.md` exists and is readable.
2. **If the file does not exist or is empty: STOP immediately.** Do not proceed. Inform the user that the required input file is missing and wait for instructions.

## Steps

1. **Read `workflow/PROJECT.md`** to obtain the project idea.

2. **Identify the core principle.** Read the project idea carefully. Determine the single concept, technique, or mechanism the experiment exists to explore. If the idea contains multiple principles, choose the most fundamental one and push the rest to out-of-scope. State the core principle in one sentence.

3. **Research the domain.** Investigate the problem space around the core principle:
   - Define key concepts and terminology that someone implementing this would need to know.
   - Explain how the underlying mechanism works at a level sufficient for implementation.
   - Identify relevant standards, protocols, or specifications (with version numbers where applicable).
   - Note any domain-specific constraints or gotchas that affect implementation.

4. **Survey implementation approaches.** Evaluate concrete options for building this experiment:
   - **Candidate languages** — For each candidate, evaluate: standard library support for the core principle's domain (e.g., crypto, networking, file I/O), ecosystem maturity, availability of zero-dependency or minimal-dependency solutions, and alignment with the "boring tech" philosophy.
   - **Design patterns** — Identify 2-3 architectural patterns that could structure the implementation (e.g., pipeline, event-driven, layered). For each, state the tradeoff.
   - **Key libraries** — If the standard library is insufficient for any candidate language, identify the most minimal and well-maintained library option. Prefer single-purpose libraries over frameworks.

5. **Define scope boundaries.** Apply the workflow philosophy to draw hard lines:
   - **In scope** — What the experiment will actually build. This must be achievable in a single implementation session with 5-15 files.
   - **Out of scope** — What the experiment explicitly will not build, even if related. Include anything that would require a second core principle, external services (unless mocked), persistent infrastructure, or UI beyond CLI.
   - **Mocked** — What the experiment will fake. Anything that is not the core principle but is needed for the experiment to function should be mocked.

6. **List assumptions.** State every assumption you are making — about the user's environment, about the domain, about what "success" looks like. Number each assumption (A-1, A-2, ...) so downstream agents can reference them.

7. **Surface open questions.** List anything you cannot determine from the project idea alone that the user needs to answer before specification can begin. Frame each question with why it matters (what decision it blocks). Number each question (Q-1, Q-2, ...).

8. **Save the output.** Write the completed document to `artifacts/RESEARCH.md`. Create the `artifacts/` directory if it does not exist. Follow the structure defined in the research-document skill. Every section must be present. Do not leave placeholders or TODOs.

## Output

| Output | File Path |
|--------|-----------|
| Research Document | `artifacts/RESEARCH.md` |

Save your completed RESEARCH.md to the path above. This file will be read by subsequent phases.

## Output Reference

Follow `skills/research-document/SKILL.md` for the structure, formatting, and completeness requirements of RESEARCH.md.

## Exit Criteria

The task is complete when ALL of the following are true:

- [ ] `artifacts/RESEARCH.md` exists as a single self-contained file at the specified path.
- [ ] The core principle is stated in one sentence.
- [ ] Domain research covers key concepts, terminology, and how the mechanism works.
- [ ] At least two candidate languages are evaluated with standard library support analysis.
- [ ] At least two design patterns are identified with tradeoffs.
- [ ] Scope boundaries are defined: in-scope, out-of-scope, and mocked items are listed.
- [ ] Assumptions are numbered and listed.
- [ ] Open questions are numbered, each with a rationale for why it matters.
- [ ] The document stands on its own — a reader with no other context can understand the project.
