# Archived live rigs

One-off live drivers whose results are recorded in docs. They carry `//go:build rigs`, so `go build ./...`, `go vet ./...` and CI skip them.
Build or vet one with `go build -tags rigs ./tools/rigs/NAME`. `tools/live.sh AGENT NAME ...` finds them here automatically (adds `-tags rigs`).
They may have bit-rotted against current internal APIs; fix on demand.

- `act2live`: Act 2 chain tool (`state units presets go hunt kill obj cubemix exitto ...`), see docs/act2.md.
- `classsmoke`: report-only class smoke runs (Paladin, Necromancer, Amazon), see docs/classes/smoke-*.md.
- `combatclass`: class-agnostic combat verification (`bind setleft auras farm`), see docs/combat.md.
- `d2helpersoak`: helper-death soak, see docs/leakhunt.md.
- `druidlive`: Druid honest play driver, see docs/summon.md.
- `friendlive`: read-only IsFriendly dump, see docs/friendly.md.
- `levelerlive`: leveler live proofs (`tree` etc.), see docs/leveler.md.
- `necrolive`: Necromancer summon driver (`buff census farm`), see docs/summon.md.
- `sandboxlive`: managed launch of one instance at the main menu inside the Wine mount sandbox, attach/helper/namespace checks, see docs/sandbox.md.
