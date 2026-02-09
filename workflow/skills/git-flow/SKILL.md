---
name: git-flow
description: Defines the Git branching strategy, commit conventions, and merge procedures for the workflow. Used by architect, developer, QA engineer, and fixer agents to maintain consistent Git history with requirement traceability.
---

# Git Flow Skill

## Step-by-Step Instructions

### Phase 4 — Implementation

1. Initialize a Git repository on `main` branch.
2. Create first commit: project structure, configuration files, IMPLEMENTATION.md stub.
3. Create `develop` branch from `main`.
4. For each implementation step in DESIGN.md §6:
   a. Create a `feature/[descriptive-name]` branch from `develop`.
   b. Implement the step.
   c. Commit with a message referencing requirements (see Commit Format below).
   d. Merge the feature branch into `develop` (no fast-forward: `--no-ff`).
   e. Delete the feature branch.
5. After all features are merged, push `develop` to remote.

### Phase 5 — Validation

1. Create `feature/validation` branch from `develop`.
2. Add validate.sh, demo.sh, and VALIDATION_REPORT.md.
3. Commit and merge into `develop`.
4. If ALL scenarios pass:
   a. Merge `develop` into `main` (no fast-forward).
   b. Tag `main` as `v0.1.0`.
   c. Push `main` and tags to remote.
5. If failures exist: stop. Proceed to Phase 7.

### Phase 7a — Fix

1. Create `fix/validation-fixes` branch from `develop`.
2. Implement fixes — **one commit per root cause**.
3. Merge into `develop`.

### Phase 7b — Re-Validation

1. Create `feature/re-validation` branch from `develop`.
2. Update validate.sh if needed, re-run, update VALIDATION_REPORT.md.
3. Commit and merge into `develop`.
4. If ALL scenarios pass:
   a. Merge `develop` into `main` (no fast-forward).
   b. Tag `main` as `v0.1.0`.
   c. Push `main` and tags to remote.

## Commit Format

```
[REQ-XX-NNN] Short imperative description

Longer description if needed. Explain what and why,
not how (the code shows how).

References:
- REQ-XX-NNN: [requirement title]
- CON-XX: [contract title] (if enforced in this commit)
```

**Examples:**

```
[REQ-CA-001] Implement root CA initialization

Create ca_engine.py with init_ca() function that generates a
self-signed root certificate with configurable subject and key algorithm.

References:
- REQ-CA-001: Generate self-signed root CA certificate
- CON-INV-01: CA certificate must have CA:TRUE basic constraint
```

```
[REQ-CRL-001] Add CRL generation

Implement crl_generator.py with generate_crl() that produces a signed
CRL containing all revoked certificate serial numbers.

References:
- REQ-CRL-001: Generate Certificate Revocation List
- CON-DAT-03: Revoked certificates must appear in next CRL
```

For fix commits:
```
[FIX] Root cause: revocation state not persisted

Add storage.save() call after revocation list update in ca_engine.py.
Fixes scenarios §6.3 and §6.5 from VALIDATION_REPORT.md.

References:
- CON-DAT-03: Revocation is immediate and permanent
```

## Branch Naming

| Phase | Pattern | Example |
|-------|---------|---------|
| Phase 4 | `feature/[descriptive-name]` | `feature/ca-initialization` |
| Phase 5 | `feature/validation` | `feature/validation` |
| Phase 7a | `fix/validation-fixes` | `fix/validation-fixes` |
| Phase 7b | `feature/re-validation` | `feature/re-validation` |

## Common Edge Cases

- A feature branch has multiple commits. Squash to one commit per feature branch before merging, unless the commits represent genuinely distinct logical steps.
- A merge conflict occurs. Resolve it in the merge commit. Document the resolution in the commit message.
- The remote is not yet created (Phase 4 start). Initialize git locally. Push to remote after the user provides the URL.

## Quality Checklist

- [ ] Repository has `main` and `develop` branches
- [ ] Feature branches follow naming convention
- [ ] Every merge to develop uses `--no-ff`
- [ ] Commit messages reference requirement IDs (REQ-XX-NNN)
- [ ] Fix commits reference the scenarios they fix
- [ ] No direct commits to `main` (only merges from develop)
- [ ] Tag `v0.1.0` exists on main only when all validations pass
- [ ] Git history is linear and readable
