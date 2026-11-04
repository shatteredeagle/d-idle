## Why

SKLIGHT finding B1 (docs/research/sk-lightning-trace.md): the bot uses one `combat.castGapMs` (650 ms) between casts, but Lightning and Chain Lightning take 19 ticks = 760 ms in game.exe, so the next command lands mid-cast. docs/fixcastgap.md has the evidence per skill and backend.

## What Changes

- internal/combat/castgap.go: per-skill cast length table (model validated against the game.exe oracle values) and `castGapFor`; `Fighter.castReady` uses the gap of the last cast's skill.
- Knobs `combat.castPerSkill` (1; 0 = flat override), `combat.castMarginMs` (90), `combat.castFCR` (0); `castGapMs` stays the floor and the fallback. docs/knobs.md regenerated.
- Skill choice, ranges and all other waits unchanged.

## Impact

Offline only. Spec combat-loop names `castGapMs` (default at least 560): the default stays 650, so no delta. Live proof pending (recipe in docs/fixcastgap.md).

Overlap (declared): internal/combat/fight.go is owned by the open card fixbuff, internal/combat/fighter.go by the archived card fixportalmelee, internal/combat/sim_test.go and internal/env/tuning_combat.go by archived cards; this card edits them (`Fighter.D2D` set in internal/d2dcombat/rig.go and cmd/d2bot/behaviors.go, owned by no open card; one line in castReady, one field and one assignment in `cast`, test helpers, three knobs) and does not list them in owns. docs/knobs.md is generated and not owned by an open card.
