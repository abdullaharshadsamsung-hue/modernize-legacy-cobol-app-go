# Migration Notes

**Model self-assessed confidence:** 93% (not a measured result)  
**Build at time of writing:** passed  
**Target unit tests:** passed  
**Behavior compared with the original:** inconclusive

The final recommendation is in the pull request description.

---

## What was migrated

- `data.cob` → `internal/store/store.go` (89% confidence)
- `operations.cob` → `internal/account/service.go` (85% confidence) ⚠️ needs review
- `main.cob` → `internal/cli/menu.go` (82% confidence) ⚠️ needs review
## Observer agent findings

The Observer agent monitored the migration and identified these patterns:

- **After 3 modules:** The target expert replaces COBOL's implicit runtime semantics (truncation, silent fall-through, rightmost-digit ACCEPT conversion) with idiomatic Go error handling or naive parsing, so behavior diverges on edge cases.

## Files requiring manual review

These files were migrated but scored below the confidence threshold.
Review them carefully before merging.

### `operations.cob`
Confidence: 85%
Issues:
  - [warning] Run returns ErrUnknownOperation instead of returning nil; no output is produced, but callers that treat errors as failures would change behavior.

### `main.cob`
Confidence: 82%
Issues:
  - [warning] ParseChoice uses the first character of the trimmed input. GnuCOBOL's PIC 9 ACCEPT keeps the rightmost digit, so '12' gives 2 there and 1 here. Input like '1a' is also handled differently.
