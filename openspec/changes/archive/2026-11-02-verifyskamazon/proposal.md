## Why

Independent verification of the first-pass Amazon trace (docs/research/skamazon-trace.md, card skamazon): own disassembly reading, own model and own emulator harness, counted cases, a verdict per claim and per ledger line.

## What Changes

- docs/research/verifyskamazon-trace.md: verdicts, own counts (calc 52,300; pierce 99,092; roll 20,000), one wrong cell, the pierce counter finding, bot findings, drill recipe, matrix text.
- tools/emu/vam_*: own calc, pierce, roll, pierce-bit and vacuity harnesses (worker host).
- No ledger edit (the rows under review are not on this base yet; repair text is in the trace).

## Impact

Docs and emulator tooling only. No Go, spec, policy, ADR or glossary change.
