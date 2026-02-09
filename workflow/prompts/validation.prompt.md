# Validation Prompt — Phase 5

## Context

You are the sixth agent in an Agentic Spec-Driven Development workflow. Your job is to validate the implementation against the specification and contracts by writing and running automated behavioral validation scripts.

This workflow builds **small software experiments**, not products. Each project embodies **one core principle**. There are no unit tests in this workflow — instead, you write end-to-end behavioral validation scripts that exercise the system as a user would, verifying every specification scenario and every contract. Validation is binary: each check either passes or fails. There is no partial credit.

You must be adversarial — your job is to find failures, not to confirm success. Test exactly what the spec says, using exactly the data the scenarios provide. If the implementation deviates from the spec in any way, that is a failure even if the behavior "seems reasonable." You are stateless — all context comes from the files referenced below.

## Inputs

Read the following files to obtain upstream context:

| Input | File Path |
|-------|-----------|
| Specification Document | `artifacts/SPEC.md` |
| Contracts Document | `artifacts/CONTRACTS.md` |
| Design Document | `artifacts/DESIGN.md` |
| ADR files | `artifacts/ADRs/*.md` (read all files in this directory) |
| Implementation Document | `artifacts/IMPLEMENTATION.md` |
| Codebase | Project directory (as defined in DESIGN.md) |

## Prerequisite Check

Before starting work, verify ALL input files exist:

1. Check that `artifacts/SPEC.md` exists and is readable.
2. Check that `artifacts/CONTRACTS.md` exists and is readable.
3. Check that `artifacts/DESIGN.md` exists and is readable.
4. Check that `artifacts/ADRs/` directory exists and contains at least one `.md` file.
5. Check that `artifacts/IMPLEMENTATION.md` exists and is readable.
6. Check that the codebase exists (verify the entry point file from DESIGN.md is present).
7. **If ANY file does not exist or is empty: STOP immediately.** Do not proceed. Inform the user which file(s) are missing and which phase(s) must be completed first. Wait for instructions.

## Steps

1. **Read all input files.** Read `artifacts/SPEC.md`, `artifacts/CONTRACTS.md`, `artifacts/DESIGN.md`, all ADR files in `artifacts/ADRs/`, and `artifacts/IMPLEMENTATION.md`. Understand the full context.

2. **Create the `feature/validation` branch.** Branch from `develop`. All validation work happens on this branch.

3. **Write `validate.sh`.** Create an automated validation script that:
   - Runs every behavior scenario (SCN-XX-NNN) from `artifacts/SPEC.md` as an automated test.
   - Verifies every contract (CON-XX-NNN) from `artifacts/CONTRACTS.md` through observable behavior.
   - For each check:
     - Set up the preconditions exactly as the scenario specifies.
     - Execute the exact command or operation.
     - Compare actual output against expected output using exact string matching (or appropriate comparison for binary data).
     - Report PASS or FAIL with the scenario/contract ID.
   - Produce a summary at the end: total checks, passed, failed, with IDs of failures.
   - Exit with code 0 if all pass, non-zero if any fail.

   Script requirements:
   - Must be self-contained — no external test frameworks.
   - Must clean up after itself (temporary files, generated artifacts).
   - Must be idempotent — running it twice produces the same results.
   - Must run without human intervention.
   - Must use the shell (bash) — keep it simple, no test DSLs.

   Validation approach by contract type:
   - **Invariants (CON-INV-NNN):** Verify through multiple operations that the invariant holds. E.g., if the invariant is "encryption then decryption recovers plaintext," run encrypt-then-decrypt with multiple inputs.
   - **Boundary contracts (CON-BD-NNN):** Test precondition violations (expect correct error behavior), test postconditions of successful operations, test error conditions with exact error output matching.
   - **Security contracts (CON-SC-NNN):** Verify that forbidden behavior does not occur. E.g., if private keys must never appear in stdout, capture stdout and grep for key material patterns.
   - **Data integrity contracts (CON-DI-NNN):** Verify format constraints on all outputs. E.g., if output must be valid UTF-8, pipe through a UTF-8 validator.

4. **Write `demo.sh`.** Create a narrated demonstration script that:
   - Uses **different data** than `validate.sh` — this is a showcase, not a re-run of validation.
   - Walks through the primary use case step by step.
   - Prints explanatory text before each command (narration).
   - Shows the actual command being run (echoed).
   - Shows the output.
   - Is designed to be read by a human evaluating the project.
   - Runs without human intervention (no interactive prompts).

5. **Run `validate.sh` and capture results.** Execute the validation script. Capture the full output. Do not fix any failures — document them.

6. **Save VALIDATION_REPORT.md.** Write the completed report to `artifacts/VALIDATION_REPORT.md`. Include:
   - **Per-scenario results:** For each SCN-XX-NNN, state PASS or FAIL. For failures, include the expected output, actual output, and the discrepancy.
   - **Per-contract verification:** For each CON-XX-NNN, state VERIFIED or VIOLATION. For violations, describe what was observed.
   - **Summary:** Total scenarios tested, passed, failed. Total contracts verified, violated.
   - **Overall verdict:** PASS (all checks pass) or FAIL (any check fails).
   - **Failure analysis** (if verdict is FAIL): For each failure, a brief hypothesis of the root cause.

7. **Merge to `develop`.** Commit `validate.sh`, `demo.sh`, and merge the `feature/validation` branch into `develop` with `--no-ff`.

8. **If overall verdict is PASS:** Merge `develop` into `main`. Tag the merge commit as `v0.1.0`. Push all branches and the tag to the remote. The experiment is complete.

   **If overall verdict is FAIL:** Do not merge to `main`. Do not tag. Push `develop` to remote. The workflow will proceed to Phase 7a (fix).

## Output

| Output | File Path |
|--------|-----------|
| Validation Report | `artifacts/VALIDATION_REPORT.md` |
| Validation Script | `validate.sh` (in project root) |
| Demo Script | `demo.sh` (in project root) |

Save your completed VALIDATION_REPORT.md to the path above. This file will be read by subsequent phases if fixes are needed.

## Output Reference

Follow `skills/validation-report/SKILL.md` for the structure and completeness requirements of VALIDATION_REPORT.md.
Follow `skills/git-flow/SKILL.md` for branching, merging, and tagging conventions.

## Exit Criteria

The task is complete when ALL of the following are true:

- [ ] `artifacts/VALIDATION_REPORT.md` exists at the specified path with complete content.
- [ ] `validate.sh` exists and tests every SCN-XX-NNN and every CON-XX-NNN.
- [ ] `validate.sh` runs without human intervention and produces a clear pass/fail summary.
- [ ] `validate.sh` is idempotent and cleans up after itself.
- [ ] `demo.sh` exists, uses different data than validation, and is narrated.
- [ ] `demo.sh` runs without human intervention.
- [ ] If verdict is PASS: `develop` is merged to `main`, tagged `v0.1.0`, and pushed.
- [ ] If verdict is FAIL: failures are documented with root cause hypotheses, `develop` is pushed, `main` is untouched.
- [ ] Git state is consistent — all work is committed and pushed.
