# ADR-003: Validate-Before-Mutate for Operation Atomicity

## Status

Accepted — Amended by ADR-006

## Context

CON-DI-004 requires that when any command fails, no persistent state files are modified. The system must remain in the exact same state as before the failed command was invoked. This applies to: the serial counter file, the CRL number file, `index.json`, certificate files in `certs/`, `ca.key`, `ca.crt`, and `ca.crl`.

This is an atomicity guarantee without a database or transaction mechanism. The experiment uses flat files for storage (Assumption A-4, no database). Go's standard library provides no file-system transaction support. The question is: how do we ensure that a failed operation (e.g., invalid CSR signature) leaves no trace in persistent state?

This matters most for `ca sign`, which modifies three files on success: the certificate file (`certs/<serial>.pem`), the serial counter (`serial`), and the index (`index.json`). If CSR validation fails after the serial counter has been incremented, the counter is now wrong and a serial number has been wasted.

## Decision

Use a **validate-before-mutate** pattern: every command function performs all validation checks before making any state changes. The function is structured in two phases:

1. **Validate phase**: Check all preconditions — CA initialization, input parsing, CSR signature, key algorithm, serial existence, revocation status. If any check fails, return an error immediately. No files have been touched.
2. **Mutate phase**: Only reached if all validations pass. Write files in sequence: certificate, serial counter, index.

This pattern is applied consistently across all state-modifying operations: `InitCA`, `SignCSR`, `RevokeCert`, and `GenerateCRL`.

## Alternatives Considered

- **Write-then-rollback**: Perform operations optimistically and roll back on failure. This is the standard approach with database transactions (BEGIN/COMMIT/ROLLBACK). Without a database, rollback means deleting files and restoring previous file contents. This is complex, error-prone (what if the rollback itself fails?), and adds significant code for a scenario that validate-before-mutate avoids entirely. Rejected because it adds unnecessary complexity.

- **Temporary files with atomic rename**: Write all outputs to temporary files first, then atomically rename them into place only after all operations succeed. This is more robust against crashes mid-operation (power failure between writes) but adds complexity. For a single-operator experiment with no concurrent access (Assumption A-2), the crash-safety benefit is negligible. The validate-before-mutate pattern already prevents the common failure modes (bad input, validation errors). Originally rejected as over-engineering for this experiment. **Update**: This gap was identified during review as conflicting with CON-DI-004's unqualified atomicity guarantee. ADR-006 introduces atomic file replacement (temp + rename) for all mutate-phase writes to close this gap.

- **In-memory state with single flush**: Load all state into memory, perform operations in memory, and flush everything to disk in one pass at the end. This would work but changes the storage model significantly — every operation would need to load and save all state, even for operations that only touch one file. Rejected because it is an unnecessary architectural change.

## Consequences

### Positive

- Simple and deterministic: the code reads linearly — all checks first, then all writes.
- No rollback logic needed. If validation fails, the function returns before any file I/O.
- Satisfies CON-DI-004 for all validation-detectable failures (CSR parse, signature, key algo, serial lookup, double revocation, re-initialization).
- Easy to audit: reviewing the validate phase confirms that no side effects occur before it completes.

### Negative

- Does not protect against failures during the mutate phase itself (e.g., disk full when writing the certificate file). If the serial counter is incremented but the index write fails, state becomes inconsistent. **This limitation is addressed by ADR-006**, which introduces atomic file replacement (temp-file-then-rename) and a stage-then-commit protocol for multi-file mutations. The residual risk is confined to process crashes between renames in the commit sub-phase — a window of microseconds rather than the full mutate phase.
- The validate phase may duplicate some I/O — for example, `SignCSR` loads the CA key and cert during validation (to verify they exist) and then uses them during mutation. In practice, the values are loaded once and reused, so there is no actual duplication.

### Neutral

- The pattern naturally matches Go's error-handling idiom: check error, return early. The code reads naturally.

## References

- ADR-006: Atomic Replace for Mutate-Phase Writes (amends this ADR)
- CON-DI-004: "Failed operations shall not modify state"
- CON-SC-003: "CSR Validation Gate — both checks shall pass before any certificate is created"
- REQ-ER-001, REQ-ER-003, REQ-ER-004, REQ-ER-005, REQ-ER-006, REQ-ER-008 from SPEC.md (all error conditions requiring no state change)
- Assumption A-2: Single-machine, single-operator scenario (no concurrent access)
