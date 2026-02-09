---
name: implementation-document
description: Defines the format, structure, and quality standards for IMPLEMENTATION.md documents. Used by the developer agent in Phase 4 to document the implementation process, setup instructions, scenario verification results, and deviations from the design.
---

# Implementation Document Skill

## Step-by-Step Instructions

1. As you implement the system following DESIGN.md, document the process in IMPLEMENTATION.md.
2. Write setup instructions that work from a clean environment — clone, install, run. Copy-paste ready.
3. Document any deviations from DESIGN.md with rationale.
4. Record the result of manually running each behavior scenario from SPEC.md §6.
5. List all files created with a one-line description of each.
6. Document the Git history summary (branches created, merge sequence).
7. Write IMPLEMENTATION.md following the Output Template below.

## Examples

**Good setup instructions:**
```bash
git clone <url>
cd project-name
python -m venv venv
source venv/bin/activate  # or venv\Scripts\activate on Windows
pip install -r requirements.txt  # if applicable
python main.py --help
```

**Bad setup instructions:**
"Install the dependencies and run it." (Not copy-paste ready, not specific.)

**Good deviation documentation:**
"DESIGN.md §3.2 specified a `parse_csr` function returning a dict. During implementation, returning a dataclass (`CSRData`) proved clearer for type hints and downstream consumers. This is a structural change, not a behavioral one — all contracts are still satisfied."

## Common Edge Cases

- A design step turns out to be unnecessary. Document why it was skipped, don't silently omit it.
- A scenario partially passes — some assertions pass, others fail. Document the exact state. Don't round up to "pass."
- The setup requires platform-specific steps. Document all platforms mentioned in DESIGN.md.

## Output Template

```markdown
# IMPLEMENTATION.md — [Project Name]

## §1 — Setup Instructions

[Copy-paste ready. From clean environment to running system. Every command listed.]

### §1.1 — Prerequisites
[Language runtime version, system dependencies if any.]

### §1.2 — Installation
[Step-by-step commands.]

### §1.3 — Verification
[One command to verify the installation works. Expected output.]

## §2 — File Inventory

| File | Description | Requirements | Contracts |
|------|-------------|-------------|-----------|
| `filename.ext` | [one-line description] | REQ-XX-NNN | CON-XX |

## §3 — Deviations from Design

| Design Reference | Deviation | Rationale |
|-----------------|-----------|-----------|
| DESIGN.md §X.X  | [what changed] | [why] |

[If no deviations: "None. Implementation matches DESIGN.md exactly."]

## §4 — Git History

[Branch creation and merge sequence. Key commits listed.]

```
main ← initial commit
  └── develop
        ├── feature/xxx ← [description] → merged to develop
        ├── feature/yyy ← [description] → merged to develop
        └── ...
```

## §5 — Dependencies

[List of all dependencies with versions and justification. Or: "No external dependencies."]

## §6 — Scenario Verification

[Result of manually running each scenario from SPEC.md §6.]

| Scenario | Status | Notes |
|----------|--------|-------|
| §6.1 — [name] | PASS/FAIL | [brief observation] |

## §7 — Known Issues

[Any issues discovered during implementation. If none: "None identified."]
```

## Quality Checklist

- [ ] Setup instructions work from a clean environment (tested)
- [ ] Every file in the codebase is listed in §2 with requirement/contract mapping
- [ ] All deviations from DESIGN.md are documented with rationale in §3
- [ ] Git history in §4 shows the branch/merge sequence
- [ ] Every scenario from SPEC.md §6 has a verification result in §6
- [ ] No scenario is marked PASS unless it fully passes
- [ ] Dependencies (if any) are listed with versions and justification
- [ ] Known issues are documented honestly, not hidden
- [ ] No empty sections or placeholder text
