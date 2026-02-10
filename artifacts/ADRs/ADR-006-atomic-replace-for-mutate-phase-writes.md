# ADR-006: Atomic Replace for Mutate-Phase Writes

## Status

Accepted — Amends ADR-003

## Context

ADR-003 introduced the validate-before-mutate pattern to satisfy CON-DI-004 ("Failed operations shall not modify state"). That pattern prevents state corruption for all validation-detectable failures: bad input, failed CSR signature checks, unsupported key algorithms, and other precondition violations. However, ADR-003 explicitly accepted a limitation:

> Does not protect against failures during the mutate phase itself (e.g., disk full when writing the certificate file). If the serial counter is incremented but the index write fails, state becomes inconsistent. This is accepted as a known limitation for a single-operator experiment.

During review, this was identified as a conflict with CON-DI-004, which states that when any command fails **for any reason** — including I/O errors — no persistent state files shall be modified. The contract's enumeration ends with "or any other error", and I/O failures during the mutate phase are covered by that clause. The consequence of the current design is that an operational I/O failure (disk full, permission revoked, filesystem error) can produce:

- **Orphaned certificate files**: `certs/<serial>.pem` written but `index.json` never updated.
- **Serial/index inconsistency**: Serial counter advanced but no corresponding index entry.
- **CRL number gaps**: CRL file written but `crlnumber` not incremented, or vice versa.

These are not theoretical: a disk-full condition after the first write in a multi-file mutation is a realistic operational failure, especially on constrained systems. The gap between the contract and the implementation must be closed.

## Decision

Introduce **atomic file replacement** using the temp-file-then-rename pattern for all persistent writes, and define a **stage-then-commit** protocol for multi-file mutations.

### Mechanism: `writeFileAtomic`

Add a private helper to `store.go`:

```go
func writeFileAtomic(path string, data []byte, perm os.FileMode) error
```

This function:

1. Writes `data` to `path + ".tmp"` with the specified permissions.
2. Calls `os.Rename(path + ".tmp", path)`.
3. If the write (step 1) fails, removes the `.tmp` file and returns the error.
4. If the rename (step 2) fails, removes the `.tmp` file and returns the error.

On POSIX systems, `rename(2)` is atomic when source and target are on the same filesystem. Since the data directory is a single directory tree, this is guaranteed. All public write functions — `SaveCertPEM`, `WriteCounter`, `SaveIndex`, `SaveCRLPEM`, `SavePrivateKey` — use `writeFileAtomic` internally. This makes every individual file write atomic: the final path either contains the old content or the new content, never a partial write.

### Protocol: Stage-then-Commit for Multi-File Mutations

For operations that modify multiple files, the mutate phase is split into two sub-phases:

**Stage sub-phase**: Prepare all output data in memory. Write each output to its `.tmp` path via the staging variant. If any `.tmp` write fails, remove all `.tmp` files created so far and return an error. At this point no final paths have been touched — the system state is unchanged.

**Commit sub-phase**: Rename each `.tmp` file to its final path in a defined order. Since the staging sub-phase proved that all data could be written successfully (sufficient disk space, correct permissions), rename failures are confined to catastrophic filesystem conditions.

The commit order for each operation is chosen to minimize the severity of a partial commit (process crash between renames):

**SignCSR commit order:**

| Step | Rename | Rationale |
|------|--------|-----------|
| 1 | `serial.tmp` → `serial` | Advances counter first to prevent serial reuse (CON-INV-001) |
| 2 | `certs/<serial>.pem.tmp` → `certs/<serial>.pem` | Places artifact; serial already reserved |
| 3 | `index.json.tmp` → `index.json` | Records the entry last; this is the commit point |

Partial-commit outcomes (crash between renames):
- After step 1 only: Serial gap — harmless, the number is simply unused. No orphaned files.
- After steps 1–2: Certificate file exists but is not in the index. The serial is consumed. An operator can detect this by comparing `certs/` directory contents against `index.json` entries. No violation of serial uniqueness.
- After all 3: Fully consistent.

**GenerateCRL commit order:**

| Step | Rename | Rationale |
|------|--------|-----------|
| 1 | `ca.crl.tmp` → `ca.crl` | CRL is updated first; a stale CRL number is less harmful than a stale CRL |
| 2 | `crlnumber.tmp` → `crlnumber` | Advances counter after CRL is in place |

**RevokeCert**: Single file (`index.json`) — `writeFileAtomic` handles it completely.

**InitCA commit order:**

| Step | Rename | Rationale |
|------|--------|-----------|
| 1 | `serial.tmp` → `serial` | Counter/index files first — these are not checked by `IsInitialized` |
| 2 | `crlnumber.tmp` → `crlnumber` | Counter file |
| 3 | `index.json.tmp` → `index.json` | Index file |
| 4 | `ca.key.tmp` → `ca.key` | Key committed before cert (cert references key) |
| 5 | `ca.crt.tmp` → `ca.crt` | Certificate last — this is the final initialization marker |

Partial-commit outcomes (crash between renames):
- After steps 1–3 only: Counter/index files exist but `ca.key` and `ca.crt` are absent. `IsInitialized` returns false. A retry of `ca init` will overwrite these orphaned files via the normal staging process (`.tmp` → rename replaces existing files atomically). No user intervention required.
- After step 4 only (key present, cert absent): `IsInitialized` returns false (requires both files). Same recovery as above.
- After all 5: Fully consistent and initialized.

Note: `IsInitialized` returns true only when both `ca.key` and `ca.crt` exist. By committing these marker files last (steps 4–5), a crash at any earlier point leaves the system in a non-initialized state, allowing `ca init` to be re-run safely. The support files (serial, crlnumber, index.json) are guaranteed to be in place before the system becomes visible as initialized.

### Cleanup Helper

Add a private helper for batch cleanup:

```go
func cleanupTempFiles(paths []string)
```

This removes all `.tmp` files in the list, ignoring errors (best-effort). It is called on stage-sub-phase failure to ensure no `.tmp` artifacts are left behind.

**InitCA cleanup scope**: On staging failure, `InitCA` calls `cleanupTempFiles` for all staged `.tmp` paths and removes the `certs/` subdirectory if it was created by this init attempt. The data directory itself is **not** removed — the `--data-dir` flag may point to a pre-existing directory containing unrelated files, and a transient I/O error (disk full, permission failure) must not cause destructive removal of user data. Only artifacts created by the current init attempt are cleaned up.

## Alternatives Considered

- **Accept the limitation and update CON-DI-004**: Scope the atomicity guarantee to validation failures only, explicitly excluding I/O errors during mutation. This was rejected because CON-DI-004 traces to six requirements (REQ-ER-001, -003, -004, -005, -006, -008) that use unqualified language about failure handling. Weakening the contract would cascade through the requirements traceability and undermine the specification's integrity.

- **Write-ahead log (WAL)**: Record intended mutations to a log file before executing them; on recovery, replay or roll back from the log. This provides true transactional semantics and crash recovery. Rejected because it introduces significant complexity (log format, replay logic, compaction) that exceeds the experiment's scope. The temp-file-then-rename approach eliminates the common failure modes without a WAL's overhead.

- **In-memory state with single flush**: Load all state, mutate in memory, serialize everything to a single composite file, and write it atomically. Rejected because it changes the storage model fundamentally and couples all state into one file, conflicting with the current per-artifact file layout.

- **SQLite or embedded database**: Provides ACID transactions natively. Rejected per ADR-001's zero-dependency constraint.

## Consequences

### Positive

- Closes the gap between ADR-003's known limitation and CON-DI-004's guarantee. I/O failures during staging leave the system state unchanged.
- Each individual file write is atomic — no partial PEM files or truncated JSON on disk.
- The staging sub-phase serves as a pre-flight check for disk space and permissions before any final paths are modified.
- Minimal implementation cost: one private helper function (~15 lines) plus write-order discipline in the four mutating operations.
- No new dependencies; uses only `os.WriteFile`, `os.Rename`, and `os.Remove` from the standard library.

### Negative

- Temporary files (`.tmp` suffix) briefly occupy additional disk space during the staging sub-phase. For this experiment's data sizes (small PEM files, small JSON index), this is negligible.
- A process crash between renames in the commit sub-phase can still produce inconsistent state. This window is extremely narrow (microseconds between renames) compared to the previous design's window (entire mutate phase including cryptographic operations and multiple writes). The defined commit order ensures the least harmful inconsistency for each operation.
- Adds ordering constraints to the mutate phase that implementers must follow. The commit-order tables above serve as the specification.

### Neutral

- The `.tmp` file convention is widely used (package managers, editors, databases) and is immediately recognizable to operators inspecting the data directory.
- A stale `.tmp` file in the data directory after a crash signals that the previous operation did not complete. This can be leveraged for future diagnostics if needed.

## References

- ADR-003: Validate-Before-Mutate for Operation Atomicity (amended by this ADR)
- CON-DI-004: "Atomicity — Failed Operations Shall Not Modify State"
- CON-DI-007: "Certificate–Index Correspondence"
- CON-INV-001: "Serial Number Uniqueness"
- CON-INV-007: "CRL Number Monotonicity"
- POSIX `rename(2)`: Atomic replacement when source and target are on the same filesystem
- Assumption A-2: Single-machine, single-operator scenario
- Assumption A-4: File-based persistence, no database
