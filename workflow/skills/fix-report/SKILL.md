---
name: fix-report
description: Defines the format, structure, and quality standards for fix plans and FIX_REPORT.md documents. Used by the fixer agent in Phase 7a to diagnose validation failures, propose targeted fixes, and document what was changed and why.
---

# Fix Report Skill

## Step-by-Step Instructions

1. Read VALIDATION_REPORT.md §2 to identify all failed scenarios.
2. For each failure, diagnose the **root cause** — not the symptom. A root cause is the specific code, logic, or data issue that produces the wrong behavior.
3. Group related failures by root cause — multiple scenario failures often share one underlying issue.
4. Read CONTRACTS.md to understand what boundaries the fix must respect.
5. Read relevant ADRs to understand which design decisions are intentional and must not be "fixed."
6. Produce a **Fix Plan** — one entry per root cause, describing: what failed, why, proposed fix, constraints.
7. **STOP. Present the Fix Plan to the user for approval.** This is a hard gate.
8. After approval, implement fixes. One commit per root cause.
9. Write FIX_REPORT.md following the Output Template below.

## Examples

**Good root cause diagnosis:**
"Scenario §6.3 (revoke certificate) fails because `revoke_certificate()` in `ca_engine.py` adds the revoked serial to the in-memory list but does not persist it to `storage.json`. When the CRL is regenerated, it reads from storage and the revoked certificate is missing. Root cause: missing persistence call after revocation state change."

**Bad root cause diagnosis:**
"The revocation test fails." (This is the symptom, not the cause.)

**Good fix plan entry:**
```
Root Cause 1: Revocation state not persisted
  Failures: §6.3, §6.5
  Diagnosis: revoke_certificate() updates in-memory state but does not call storage.save()
  Proposed Fix: Add storage.save() call after updating revocation list in ca_engine.py:revoke_certificate()
  Constraints:
    - Must not violate CON-DAT-03 (revocation is immediate and permanent)
    - Must respect ADR-003 (storage format is JSON, not database)
  Impact: 2 lines changed in ca_engine.py
```

## Common Edge Cases

- A failure seems like a spec issue, not a code issue. Document this in the fix plan and let the user decide. Do not silently change spec behavior.
- A fix would violate a contract. The contract wins. Find a different fix that respects the contract.
- A fix would contradict an ADR. The ADR wins unless the user explicitly approves superseding it.
- Multiple root causes interact. Fix them independently and document the interaction.

## Output Template

### Fix Plan (presented to user BEFORE implementation)

```markdown
# Fix Plan — [Project Name]

## Failures Analyzed
[Total count and summary from VALIDATION_REPORT.md]

## Root Causes

### Root Cause 1: [Short Title]
- **Failing scenarios:** §6.N, §6.N
- **Diagnosis:** [What's wrong and why]
- **Proposed fix:** [Specific change — file, function, what changes]
- **Constraints:** [Contracts and ADRs that constrain this fix]
- **Impact:** [Scope of change — files and lines affected]

(Continue for all root causes.)

## Fix Order
[Recommended order of fixes, noting any dependencies between them.]

## Risk Assessment
[What could go wrong with these fixes. What to watch for in re-validation.]
```

### FIX_REPORT.md (written AFTER implementation)

```markdown
# FIX_REPORT.md — [Project Name]

## §1 — Fix Summary

| # | Root Cause | Scenarios Fixed | Commit | Status |
|---|-----------|----------------|--------|--------|
| 1 | [title] | §6.N, §6.N | [SHA] | FIXED |

## §2 — Detailed Changes

### Root Cause 1: [Title]

**Diagnosis:** [What was wrong]
**Fix:** [What was changed — specific files, functions, lines]
**Contracts respected:** CON-XX
**ADRs respected:** ADR-NNN

```diff
[Key diff snippet showing the fix]
```

(Continue for all root causes.)

## §3 — Unfixed Issues

[Any failures that were NOT addressed, with reasoning. If none: "All identified failures have been addressed."]

## §4 — Post-Fix State

| Metric | Value |
|--------|-------|
| Root causes identified | N |
| Root causes fixed | N |
| Scenarios expected to pass after fix | N/M |
| Files modified | N |
| Total commits | N |

## §5 — Re-Validation Readiness

[Confirmation that the codebase is ready for Phase 7b re-validation. Any caveats or things to watch for.]
```

## Quality Checklist

- [ ] Every failure from VALIDATION_REPORT.md is traced to a root cause
- [ ] Root causes are actual causes, not symptom descriptions
- [ ] Related failures are grouped under shared root causes
- [ ] Fix plan was presented to and approved by the user before implementation
- [ ] Every fix respects all contracts from CONTRACTS.md
- [ ] No fix contradicts an ADR without explicit user approval
- [ ] One commit per root cause in the fix branch
- [ ] FIX_REPORT.md §2 shows specific code changes for each fix
- [ ] Unfixed issues (if any) are documented with reasoning in §3
- [ ] §4 post-fix state gives accurate counts
- [ ] No empty sections or placeholder text
