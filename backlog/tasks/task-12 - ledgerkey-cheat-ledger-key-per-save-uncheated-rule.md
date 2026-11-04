---
id: TASK-12
title: "ledgerkey: cheat ledger key per save (uncheated rule)"
status: In Progress
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-F"
dependencies: []
priority: low
ordinal: 12000
card: "ledgerkey"
owns:
  - "internal/cheat/ledger.go"
  - "internal/cheat/ledgerkey.go"
  - "internal/cheat/ledgerkey_test.go"
  - "cmd/d2cheat/main.go"
exclusive: []
branch: "agent/ledgerkey"
agent: "snaplib"
result: "Offline unit tests only; no live run needed for the key itself."
live: ""
merge_hash: ""
overlap: "Overlap (declared): card memrules (agent/memrules, unmerged) edits internal/cheat/ledger.go (adds `Run` after `Backup`, separate hunk) and cmd/d2cheat/main.go (`runstart`, `--since-run`, separate hunks); merge order is free, ledgerkey was built on main. Overlap (declared): card snaplib writes the setup-copy entries."
source: "openspec-final:openspec/changes/ledgerkey"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The cheat ledger is keyed by character name, so five copies of one save (or one name in two prefixes) share one identity and one cheat poisons all of them (docs/research/character-factory.md 4.5; Palli exists in two prefixes with a shared count). The baseline copies and the memrules 'writes after start' count need per-prefix identity.

What changes:
- internal/cheat/ledger.go: optional `char_id`, `Filter.Key`, `Ledger.CheatedKey`, `Summary.ByKey`; bookkeeping kinds (setup-copy, runstart) never count as cheats.
- internal/cheat/ledgerkey.go: `Entry.Key()` (CharID, else instance/name, else name), `KindSetupCopy`, `CharKey`.
- cmd/d2cheat: `ledger --key INST/NAME --inst I`, key column.

Next step: merge check, then close

Gates (from the card):
- go test -p 1 ./internal/cheat/...; go vet ./cmd/d2cheat; changecheck -cli 0 warnings; leakcheck -mode fail 0

Source: git tag openspec-final, openspec/changes/ledgerkey/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 1.1 key, filter, CheatedKey, summary (internal/cheat/ledgerkey_test.go, legacy and renamed cases)
- [x] #2 1.2 d2cheat ledger flags
- [ ] #3 1.3 private tools (soak24.sh preflight, ledgercheck.sh) key by instance/name (coordinator)
<!-- AC:END -->
