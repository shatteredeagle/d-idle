## Why

P18 paid hire is BLOCKED: no drill or harness verb hires for gold (docs/merclive.md design D8.h; hirefix deferred B3 for the same reason).

## What Changes

Implement drill D8.h per the design in docs/merclive.md: hirer window, gold set as SETUP before the run start, paid hire, merc present with the right class code in the dump, gold delta read against the quoted price; offline tests with the existing fakes.

## Impact

Offline only (fakes and docs), nothing verified live.
