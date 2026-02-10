# ADR-003: Validate-Before-Mutate with Atomic Writes for Operation Atomicity

## Status

Accepted

## Context

CON-DI-004 requires that when any command fails — **for any reason** — no persistent state files are modified. The system must remain in the exact same state as before the failed command was invoked. This applies to: the serial counter file, the CRL number file, `index.json`, certificate files in `certs/`, `ca.key`, `ca.crt`, and `ca.crl`.

This is an atomicity guarantee without a database or transaction mechanism. The experiment uses flat files for storage (Assumption A-4, no database). Go's standard library provides no file-system transaction support. The question is: how do we ensure that a failed operation leaves no trace in persistent state?

There are two categories of failure:

1. **Logical failures**: Invalid input, CSR signature verification failure, unsupported key algorithm, certificate not found, already revoked, CA not initialized. These are detectable before any file writes.
2. **I/O failures**: Disk full, permission denied, filesystem error during the mutate phase itself. These occur during file writes and cannot be prevented by pre-validation alone.

This matters most for `ca sign`, which modifies three files on success: the certificate file (`certs/<serial>.pem`), the serial counter (`serial`), and the index (`index.json`). If the certificate is written but the index write fails, the system has an orphaned certificate file and a stale index — inconsistent state that violates CON-DI-004.

## Decision

Use a **two-layer atomicity strategy**:

### Layer 1: Validate-Before-Mutate

Every command function performs all validation checks before making any state changes. The function is structured in two phases:

1. **Validate phase**: Check all preconditions — CA initialization, input parsing, CSR signature, key algorithm, serial existence, revocation status. If any check fails, return an error immediately. No files have been touched.
2. **Mutate phase**: Only reached if all validations pass.

This pattern is applied consistently across all state-modifying operations: `InitCA`, `SignCSR`, `RevokeCert`, and `GenerateCRL`.

### Layer 2: Atomic File Writes with Ordered Cleanup

For the mutate phase, two mechanisms protect against I/O failures:

**WriteFileAtomic**: A helper function that writes data to a temporary file in the same directory as the target, then atomically renames it over the target using `os.Rename`. On POSIX systems, `os.Rename` within the same filesystem is atomic — the target file either has the old content or the new content, never a partial write. This protects individual file writes against corruption from I/O failures mid-write.

```go
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error
// 1. Create temp file in same directory as path (same filesystem for atomic rename).
// 2. Write data to temp file.
// 3. Sync temp file.
// 4. Rename temp file over path (atomic on POSIX same-filesystem).
// 5. If any step fails: remove temp file, return error.
```

**Ordered cleanup for multi-file mutations**: Operations that modify multiple files (e.g., `SignCSR` writes cert, serial, and index) use ordered writes with cleanup on failure:

- `SignCSR` mutate phase:
  1. Write cert to `certs/<serial>.pem` (new file — cleanup target).
  2. Write serial counter via `WriteFileAtomic`. If fails: `RemoveIfExists(certPath)`, return error.
  3. Write index via `WriteFileAtomic`. If fails: `RemoveIfExists(certPath)`, `WriteCounter(oldSerial)` to restore, return error.

- `GenerateCRL` mutate phase:
  1. Write CRL via `WriteFileAtomic` (overwrites previous CRL atomically).
  2. Write CRL number counter via `WriteFileAtomic`. If fails: restore previous CRL content or remove CRL, return error.

- `RevokeCert` mutate phase: Single `WriteFileAtomic` call for `index.json` — atomic by itself.

- `InitCA` mutate phase: Creates a new data directory. If any write fails after directory creation, remove the entire data directory and return error.

**RemoveIfExists**: A helper that removes a file if it exists, returning nil if the file does not exist. Used in cleanup paths.

```go
func RemoveIfExists(path string) error
// Removes the file at path if it exists. Returns nil if file does not exist.
```

## Alternatives Considered

- **Validate-before-mutate only (no I/O failure handling)**: Perform all validation before writes, but accept that I/O failures during the mutate phase can leave inconsistent state. This was the initial approach and is simpler — no cleanup logic, no atomic writes. However, it explicitly violates CON-DI-004 for I/O failure scenarios. The contract states that failed operations shall not modify state "for any reason," not just for validation failures. Rejected because it does not satisfy CON-DI-004's absolute requirement.

- **Write-then-rollback**: Perform operations optimistically and roll back on failure. This is the standard approach with database transactions (BEGIN/COMMIT/ROLLBACK). Without a database, rollback means deleting files and restoring previous file contents. This is conceptually what Layer 2 does, but framed as "ordered cleanup" rather than a general rollback mechanism. A general rollback system would require snapshotting all modified files before mutation, which adds more code and complexity than the targeted cleanup approach. Rejected in favor of the simpler ordered-cleanup pattern.

- **Temporary files with atomic rename for ALL files**: Write all outputs to temporary files first, then atomically rename them all into place only after all operations succeed. This provides the strongest atomicity guarantee but is more complex for multi-file operations (need to track multiple temp files and rename them in sequence). The ordered-cleanup approach provides equivalent protection for the experiment's mutation patterns with less code. Rejected as over-engineering.

- **In-memory state with single flush**: Load all state into memory, perform operations in memory, and flush everything to disk in one pass at the end. This would work but changes the storage model significantly — every operation would need to load and save all state, even for operations that only touch one file. Rejected because it is an unnecessary architectural change.

## Consequences

### Positive

- Simple and deterministic: the code reads linearly — all checks first, then all writes with cleanup.
- Satisfies CON-DI-004 for both logical failures (Layer 1) and I/O failures (Layer 2).
- `WriteFileAtomic` prevents partial/corrupt files from I/O failures mid-write.
- Ordered cleanup ensures that if a later write fails, earlier writes are rolled back.
- Easy to audit: reviewing the validate phase confirms no side effects before it completes; reviewing the mutate phase confirms cleanup paths exist for each write.
- The pattern naturally matches Go's error-handling idiom: check error, clean up, return early.

### Negative

- The cleanup logic adds code to each multi-file mutation. `SignCSR` has two cleanup branches (serial failure, index failure) and `GenerateCRL` has one. This is more code than the validate-only approach.
- **Residual risk**: If a cleanup operation itself fails (e.g., `RemoveIfExists` fails after `WriteFileAtomic` for serial fails), the system may still be left in an inconsistent state. This requires two sequential I/O failures on the same filesystem, which is extremely unlikely in a single-operator experiment. This narrow residual risk is accepted.
- The validate phase may duplicate some I/O — for example, `SignCSR` loads the CA key and cert during validation (to verify they exist) and then uses them during mutation. In practice, the values are loaded once and reused, so there is no actual duplication.

### Neutral

- `WriteFileAtomic` uses `os.Rename` which is atomic on POSIX same-filesystem renames. Since temp files are created in the same directory as the target, this guarantee holds. Cross-filesystem atomicity is not needed.
- The cleanup paths are tested implicitly by the behavioral validation script through error scenarios.

## References

- CON-DI-004: "Failed operations shall not modify state"
- CON-DI-007: "Certificate–index correspondence" (protected by ordered cleanup in SignCSR)
- CON-SC-003: "CSR Validation Gate — both checks shall pass before any certificate is created"
- REQ-ER-001, REQ-ER-003, REQ-ER-004, REQ-ER-005, REQ-ER-006, REQ-ER-008 from SPEC.md (all error conditions requiring no state change)
- Assumption A-2: Single-machine, single-operator scenario (no concurrent access)
