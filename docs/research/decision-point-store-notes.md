# Decision-point store: build notes (TASK-19, offline)

Package `internal/decide`. Built to [decision-point-store-spec.md](decision-point-store-spec.md); stdlib only, no game, no wiring. Tags: V = proven by a test in the package, U = my design choice.

## Files
- `catalogue.json` (embedded): 23 points, fields and bands (life edges 25/35/55/70), class table, risk tables, defaults, urgency table, config (N 2/3/5, fade 45 s, max skips 1). Reviewable data, no per-point Go.
- `catalogue.go`, `context.go`: load + validation, banding, context key `point|k=v,...` (sorted; act, diff, hc always in).
- `store.go`: Ask/Answer/Confirm/Tick/Revoke/RevokeAll/Sacrifice/Rebirth/Rules/Replay, `Clock`, `ManualClock`.
- `persist.go`: `Open(dir, profile, ...)`, file `decide-<profile>.json`, `FileVersion` 1, atomic write with `.bak`.
- `script.go`: `RunScript` drives a store from an answer script on virtual time.
- `rulefile.go`, `default_rules.json`: rule file format and the T0-T2 file from spec 5.3.

## API summary
`Shipped()`, `cat.Build(point, Obs, stage, UrgencyProvider) -> Context`, `store.Ask(ctx) -> Decision{Answer, Source rule|default|default-locked, Thought}` (never waits), `Answer(id, answerID|"notsure")`, `Confirm(id, yes)`, `Tick()` (fades), `Revoke(key)`, `RevokeAll()`, `Rules() []RuleView`, `Replay(cat, profile, log) -> canonical rules bytes`, `Stats()` (rules learned, answers given, thoughts faded).

## Behaviour proven (V, `go test ./internal/decide/`)
Streak N by class; reset on a different answer; new key is a new streak; fade and late answer neither count nor reset; notsure neither; risky confirm (no resets, fade leaves unlearned, yes learns); revoke one/all, player only (`RevokedBy` is always "player"); sacrifice skip; tier lock; irreversible defaults are do-nothing answers (catalogue refused otherwise); rebirth keeps rules, streaks, log, quality and fades the dead hero's thoughts; two builds (two classes, class numbers only in the class table, no class name in data or Go); unknown point, class, field, value, answer return errors naming them; 5 answer profiles (cautious, bold, inconsistent, 2 seeded holdouts) x 2 classes; 25 random scripts with player actions; replay equals live bytes; persistence round trip, refused newer or corrupt file.

## Stubbed for stage 2
`UrgencyProvider` interface and `stage 2` in `Build` (urgency replaces life on the three urgency points; old life-band rules keep their own keys and are not migrated, no journal line yet). The urgency table in the catalogue is data only, the store does not evaluate it (combat card). Journal/incident lines, the HTTP routes and the watch screen are not built. Plan-keyed points: no point uses a plan; an unknown class errors, an unknown plan has no field yet (U).

## Choices (U)
Skips are banked per point (not per key). A late answer is logged as a fade. A confirm thought shares the key's single open slot, so no ask thought opens while a confirm is pending. Defaults may read any key field, not only kit and hc (needed for gold rows of spec 5.2). Band edge value goes to the upper band. The enum lists for packtype, runkind, boss, slot, potiontier are placeholders.

## Rule-file gaps (ADR 0016 gate stays open)
Test `TestRuleFileExpressesTodaysDefaults` logs coverage: fully covered 9 points, partly 4, not covered 10. Gaps (also in `default_rules.json`): no "keep fighting" answer for life bands above 55 and "leave the game" is not an answer; rejuvenation vs portal needs the potion held; lethal-pack list is a placeholder; absolute clvl floors vs relative level gap; no per-area retreat counter knob; no knob for leave.safe, shrine, stash-full, gamble, potion tier, run-time ratio; restock gold edges 100/500 vs floors 300/60; pack percent vs rows; rare/unique depend on pickit priorities. Rows marked `differs` show where a knob-derived answer disagrees with the spec's default (retreat.drink at 25-35 and below 25, restock half belt, repair above 30, failstreak in hardcore).
