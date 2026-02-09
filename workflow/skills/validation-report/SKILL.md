---
name: validation-report
description: Defines the format, structure, and quality standards for VALIDATION_REPORT.md, validate.sh, and demo.sh. Used by the QA engineer agent in Phase 5 and 7b to validate the implementation against specs and contracts.
---

# Validation Report Skill

## Step-by-Step Instructions

1. Read SPEC.md §6 (scenarios), CONTRACTS.md (all sections), and IMPLEMENTATION.md §6 (scenario results).
2. Write `validate.sh` — an automated script that runs every behavior scenario end-to-end and every contract check. Use the template in `scripts/validate-template.sh` as a starting point.
3. Write `demo.sh` — a narrated walkthrough that tells the story of the system. Use DIFFERENT data than validation to prove the system generalizes.
4. Run `validate.sh` and capture all output.
5. For each scenario: record pass/fail, actual output vs expected output.
6. For each contract: record whether it's enforced (tested through scenarios that would violate it).
7. Write VALIDATION_REPORT.md following the Output Template below.
8. Determine the overall verdict: ALL PASS or FAILURES FOUND.

## Examples

**Good validation scenario (in validate.sh):**
```bash
echo "--- Scenario: REQ-CA-001 — Generate root CA certificate ---"
OUTPUT=$(python main.py init --subject "CN=TestRootCA" --key-algo RSA-2048 --validity-days 365)
# Check exit code
if [ $? -ne 0 ]; then echo "FAIL: Non-zero exit code"; FAILURES=$((FAILURES+1)); fi
# Check certificate was created
if [ ! -f ca/root.crt ]; then echo "FAIL: Certificate file not created"; FAILURES=$((FAILURES+1)); fi
# Check subject
SUBJECT=$(openssl x509 -in ca/root.crt -noout -subject 2>/dev/null)
echo "$SUBJECT" | grep -q "CN=TestRootCA" && echo "PASS: Subject matches" || { echo "FAIL: Subject mismatch"; FAILURES=$((FAILURES+1)); }
```

**Good demo narration (in demo.sh):**
```bash
echo ""
echo "=== Certificate Authority Service — Demo ==="
echo ""
echo "This demo walks through the complete lifecycle of a certificate"
echo "authority: creating a root CA, signing certificates, and revoking"
echo "them using both CRL and OCSP mechanisms."
echo ""
echo "--- Step 1: Initialize the Root CA ---"
echo "We create a new root Certificate Authority with a 10-year validity."
echo ""
python main.py init --subject "CN=DemoRootCA,O=Demo Corp" --validity-days 3650
echo ""
echo "The root CA is now ready. Let's inspect it:"
```

## Common Edge Cases

- A scenario requires cleanup between runs. The validate script must handle its own setup and teardown — assume nothing about prior state.
- A scenario depends on another scenario's output. Order matters in validate.sh — run dependent scenarios in sequence, but document the dependency.
- validate.sh needs to run on different platforms. Use portable shell constructs. Note platform requirements.
- A contract can't be directly tested (it's about what must NOT happen). Test it by attempting the violation and confirming the system rejects it.

## Output Template

```markdown
# VALIDATION_REPORT.md — [Project Name]

## §1 — Validation Environment

| Property | Value |
|----------|-------|
| Date | [YYYY-MM-DD] |
| Platform | [OS, runtime version] |
| Commit | [Git SHA] |
| Branch | [branch name] |

## §2 — Scenario Results

| # | Scenario | Spec Reference | Status | Notes |
|---|----------|---------------|--------|-------|
| 1 | [name] | §6.N / REQ-XX-NNN | PASS/FAIL | [brief note] |

### Failed Scenarios (if any)

#### Scenario N: [Name]
- **Expected:** [what should happen]
- **Actual:** [what happened]
- **Possible cause:** [initial diagnosis]

## §3 — Contract Verification

| Contract | Verification Method | Status | Notes |
|----------|-------------------|--------|-------|
| CON-XX | [how it was tested] | PASS/FAIL | [brief note] |

## §4 — Demo Status

| Property | Value |
|----------|-------|
| demo.sh runs without error | YES/NO |
| Uses different data than validation | YES/NO |
| Output is narrated | YES/NO |
| Core principle is demonstrated | YES/NO |

## §5 — Summary

| Metric | Count |
|--------|-------|
| Total scenarios | N |
| Passed | N |
| Failed | N |
| Contracts verified | N/M |

## §6 — Verdict

**[ALL PASS / FAILURES FOUND]**

[If ALL PASS: "All behavior scenarios pass. All contracts are verified. The system is ready for main merge and v0.1.0 tag."]

[If FAILURES FOUND: "N scenarios failed. See §2 for details. Proceeding to Phase 7 (fix cycle) is recommended."]
```

## Quality Checklist

- [ ] validate.sh runs from a clean state with zero human intervention
- [ ] validate.sh tests every scenario from SPEC.md §6
- [ ] validate.sh tests contract enforcement (attempts violations, confirms rejection)
- [ ] validate.sh outputs clear PASS/FAIL per scenario
- [ ] validate.sh returns non-zero exit code if any scenario fails
- [ ] demo.sh uses different sample data than validate.sh
- [ ] demo.sh has narrated output explaining each step
- [ ] demo.sh demonstrates the core principle end-to-end
- [ ] demo.sh is runnable by someone who has never seen the project
- [ ] VALIDATION_REPORT.md §2 covers every scenario
- [ ] VALIDATION_REPORT.md §3 covers every contract
- [ ] VALIDATION_REPORT.md §6 has a clear verdict
- [ ] Failed scenarios have expected vs actual comparison
- [ ] No empty sections or placeholder text
