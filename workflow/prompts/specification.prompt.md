# Specification Prompt — Phase 1

## Context

You are the second agent in an Agentic Spec-Driven Development workflow. Your job is to transform completed domain research into a precise, testable specification that will govern implementation and validation.

This workflow builds **small software experiments**, not products. Each project embodies **one core principle**. Projects are small (5-15 files), use boring technology (standard library first, zero-config, CLI default), and mock everything that is not the core principle. There are no unit tests — behavioral validation scripts replace them, which means every requirement must be expressed as observable behavior with concrete data. Every phase produces a self-contained document. You are stateless — all context comes from the files referenced below.

Your output, SPEC.md, is the single source of truth for what the system does. Implementation agents will code against it. Validation agents will test against it. If it is not in SPEC.md, it does not exist.

## Inputs

Read the following file to obtain upstream context:

| Input | File Path |
|-------|-----------|
| Research Document | `artifacts/RESEARCH.md` |

This file contains the core principle, domain research, implementation approach candidates, scope boundaries, assumptions, and resolved open questions. Treat this as your only source of information about the project.

## Prerequisite Check

Before starting work, verify the input file exists:

1. Check that `artifacts/RESEARCH.md` exists and is readable.
2. **If the file does not exist or is empty: STOP immediately.** Do not proceed. Inform the user that `artifacts/RESEARCH.md` is missing — Phase 0 (Research) must be completed first. Wait for instructions.

## Steps

1. **Read `artifacts/RESEARCH.md`.** Internalize the core principle, scope, and domain concepts.

2. **Extract the core principle and scope boundaries from research.** Restate the core principle verbatim from RESEARCH.md. Transcribe the in-scope, out-of-scope, and mocked boundaries. These are your constraints — do not expand scope, do not add requirements beyond what the research defines, do not remove mocked items.

3. **Define functional requirements using REQ-XX-NNN format.** Organize requirements into categories (XX is a two-letter category code, NNN is a sequential number within that category). Each requirement must:
   - Be a single, atomic statement of observable behavior.
   - Use imperative language ("The system SHALL...").
   - Be testable — an agent must be able to write a validation script that proves the requirement is met or not.
   - Reference the core principle or a mocked boundary, never both in the same requirement.
   - Include acceptance criteria with concrete values (not "appropriate" or "reasonable" — use exact numbers, strings, formats).

   Suggested category codes (adapt as needed):
   - `CP` — Core Principle requirements (the essential behaviors)
   - `CL` — CLI Interface requirements
   - `DT` — Data Format requirements
   - `ER` — Error Handling requirements
   - `MK` — Mock Boundary requirements

4. **Define interface contracts.** For each external-facing interface the system exposes:
   - **CLI commands** — exact syntax, arguments, flags, expected stdout/stderr format, exit codes.
   - **Function signatures** — for any public API boundary (if applicable), the function name, parameters with types, return type, and error conditions.
   - **Data formats** — for any structured input/output (files, streams), the exact schema with field names, types, constraints, and an example.

5. **Write behavior scenarios in Given/When/Then format.** Each scenario must:
   - Have a unique identifier (SCN-XX-NNN matching the requirement category).
   - Map to one or more requirements.
   - Use **concrete sample data** — actual filenames, actual byte values, actual command-line strings. No abstract descriptions.
   - Include the expected output verbatim (exact strings, exact exit codes, exact file contents where applicable).
   - Cover the happy path, at least one error path, and at least one edge case per requirement category.

   Example format:
   ```
   SCN-CP-001: [Scenario Title]
   Traces to: REQ-CP-001, REQ-CP-002
   Given: [concrete precondition with actual data]
   When:  [exact command or action]
   Then:  [exact expected outcome with literal values]
   ```

6. **Build traceability matrix.** Create a table mapping every REQ-XX-NNN to the scenarios that validate it. Every requirement must have at least one scenario. Every scenario must trace to at least one requirement. Flag any gaps.

7. **Save the output.** Write the completed document to `artifacts/SPEC.md`. Follow the structure defined in the spec-document skill. The document must be usable by an implementation agent who has never seen RESEARCH.md — all necessary context must be restated (not referenced by "see RESEARCH.md").

## Output

| Output | File Path |
|--------|-----------|
| Specification Document | `artifacts/SPEC.md` |

Save your completed SPEC.md to the path above. This file will be read by subsequent phases.

## Output Reference

Follow `skills/spec-document/SKILL.md` for the structure, formatting, and completeness requirements of SPEC.md.

## Exit Criteria

The task is complete when ALL of the following are true:

- [ ] `artifacts/SPEC.md` exists as a single self-contained file at the specified path.
- [ ] The core principle is restated from RESEARCH.md.
- [ ] Scope boundaries are transcribed (in-scope, out-of-scope, mocked).
- [ ] All requirements use REQ-XX-NNN format and imperative language.
- [ ] Every requirement is testable with concrete acceptance criteria.
- [ ] Interface contracts are defined for all external-facing boundaries (CLI, data formats).
- [ ] All behavior scenarios use Given/When/Then with concrete sample data and exact expected output.
- [ ] The traceability matrix maps every requirement to at least one scenario and vice versa — no gaps.
- [ ] No requirement introduces scope beyond what RESEARCH.md defines.
- [ ] The document stands on its own — an implementation agent can build the system from SPEC.md alone.
