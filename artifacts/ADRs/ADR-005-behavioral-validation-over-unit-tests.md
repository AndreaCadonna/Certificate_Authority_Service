# ADR-005: Behavioral Validation Script Over Go Unit Tests

## Status

Accepted

## Context

The experiment needs validation to confirm that the CA lifecycle works correctly. Go's standard testing approach uses the `testing` package with `*_test.go` files, run via `go test`. The workflow philosophy explicitly mandates "behavioral validation scripts instead" of unit tests (Assumption A-10 in SPEC.md).

The question is how to validate the experiment: Go unit tests (idiomatic for Go projects), a bash script that exercises the compiled binary (behavioral), or both.

The experiment's value is in the observable behavior of the compiled CLI tool — the commands it accepts, the files it produces, the exit codes it returns, and the output it prints. The internal function boundaries are implementation details that may change.

## Decision

Use a single bash script (`validate.sh`) that builds the `ca` binary, exercises all CLI commands in sequence, and validates exit codes, stdout/stderr content, and file existence/content. No Go unit tests are written.

The script implements helper functions (`check`, `check_stdout_contains`, `check_stderr_contains`, `check_file_exists`, `check_file_starts_with`) and runs through test groups corresponding to SPEC.md §5 scenarios (SCN-CP-001 through SCN-MK-004 and SCN-ER-001 through SCN-ER-008).

## Alternatives Considered

- **Go `testing` package with unit tests**: Write `ca_test.go`, `store_test.go`, `verify_test.go`, etc. with `func Test*` functions. This is the idiomatic Go approach and would test internal functions directly (e.g., `TestParseDN`, `TestSignCSR`). However, the workflow philosophy explicitly rejects unit tests in favor of behavioral validation. Unit tests test implementation details (function signatures, internal error types) rather than observable behavior. When the implementation changes (e.g., refactoring `SignCSR` into smaller functions), unit tests break even if behavior is unchanged. A behavioral script tests the contract — the CLI interface — which is stable. Rejected per workflow philosophy.

- **Go `testing` package with integration tests**: Write Go test files that invoke the compiled binary via `os/exec.Command`. This is a hybrid approach — Go test infrastructure but behavioral testing style. It would allow using Go's `testing.T` assertions and parallel test execution. However, it requires a build step before testing (compile the binary, then run tests that invoke it), adds `_test.go` files that inflate the file count, and gains little over a bash script for sequential CLI validation. Rejected because it adds files and complexity without meaningful benefit.

- **Both bash script and Go unit tests**: Write the bash script for behavioral validation AND Go unit tests for internal functions. This provides maximum coverage but doubles the validation effort, adds test files to the project, and conflicts with the workflow philosophy's "no unit tests" directive. Rejected as over-engineering.

## Consequences

### Positive

- Tests the actual user-facing interface: CLI commands, exit codes, stdout/stderr output, and file artifacts.
- Changes to internal function signatures or module structure do not break the validation script.
- The script is readable by anyone — no Go testing knowledge required.
- A single file (`validate.sh`) covers all scenarios. No test file proliferation.
- Consistent with the workflow philosophy of behavioral validation over unit tests.

### Negative

- Bash scripts are less precise than Go assertions. Pattern matching with `grep` can produce false positives if the search pattern is too broad.
- No parallel test execution. Bash scripts run sequentially, making the validation slower than parallelized Go tests. (Acceptable for an experiment with <30 test cases.)
- Error reporting is less detailed than Go's `testing.T` — the script reports PASS/FAIL with a description but not a stack trace or diff.
- Platform dependency: the script requires bash and standard Unix utilities (`mktemp`, `grep`, `sed`). Windows users need WSL or Git Bash. (Acceptable per Assumption A-11 — Go is cross-platform, but validation runs on Unix-like systems.)

### Neutral

- The validation script must be maintained alongside the CLI implementation. When output format changes, the script's expected patterns must be updated. This is the same maintenance burden as unit tests — the validation target is just different (stdout strings vs. function return values).

## References

- Assumption A-10 from SPEC.md: "Behavioral validation scripts replace unit tests."
- SPEC.md §5: All behavior scenarios (SCN-CP-*, SCN-CL-*, SCN-DT-*, SCN-ER-*, SCN-MK-*)
- Workflow philosophy: "no unit tests (behavioral validation scripts instead)"
