## Why

P18 paid hire is BLOCKED: no drill or harness verb hires for gold (docs/merclive.md design D8.h; hirefix deferred B3 for the same reason).

## What Changes

Implement drill D8.h per the design in docs/merclive.md: hirer window, gold set as SETUP before the run start, paid hire, merc present with the right class code in the dump, gold delta read against the quoted price; offline tests with the existing fakes.

## Impact

Offline only (fakes and docs), nothing verified live.

## Overlap (declared)

m1-liveproof owns cmd/d2drill/** and internal/drill/**; merchire edits them and does not list them in owns (like fixmanamodel): three new files (internal/drill/merchire.go, internal/drill/merchire_test.go, cmd/d2drill/hire_lab.go) and a one-line registration in internal/drill/dgame.go (D8 now includes D8h). No shared function is changed.
