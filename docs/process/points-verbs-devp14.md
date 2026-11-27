# Fork additions wanted for stat and skill points (devp14), and what the bot does without them

> status: proposal for the fork-rebase routine (card agencyp0, 2026-11-19). Tags: V verified in the source of the fork checkout, S secondary, U unverified. Nothing here was built in the fork; the bot side (`docs/points.md`) works with the verbs that exist today.

## What exists (V: fork checkout, devp13 line)

- `cmd stat <stat> [n]` and `cmd skillpt <index>` in `apps/d2d/devctl/verbs.hpp` (lines 110 and 111) send `cmd::StatPoint{stat, n}` and `cmd::SkillPoint{index}`; `components/game/world/commands.cpp` applies them with `rules::spend_stat_points` (stat 0..3, count clipped to the unspent points) and `rules::learn_skill` after `rules::can_learn` (max level, clvl >= reqlevel, prerequisites). No lab flag.
- `dump player,stats,skills` carries `stat_pts`, `skill_pts`, `stats.base`, `stats.bonus`, `stats.raw` and `skills.base_levels` (S: `internal/d2dsrc/testdata/dump_head_late.json`).

## Wanted, in order of value (all behind no flag: they are plain reads and protocol-faithful commands)

1. `caps` tokens `points.stat` and `points.skill` (one line each in the `caps` list). Today the bot probes by the first "unknown"/"usage" answer (`cmd/d2bot/pointswire.go`); a token makes the capability check exact and the sheet can say "this d2d cannot spend points" before the first tap.
2. A refusal reason on `cmd skillpt` and `cmd stat`: today both answer `ok` even when the world ignores the command (no unspent point, cannot learn). The bot reads the points back and reports `not_taken`; a reply like `err cannot-learn reqlevel=18` or `ok spent` would turn that into a reason in words.
3. `dump player` to carry `class_base` (the four class starting attributes) so the sheet need not read charstats on the bot side (the bot has charstats already: low value).
4. For P1 (not this card): an Akara respec verb (`cmd respec`) and its count per difficulty in the dump, needed by the respec flow (design review 04 section 3.7; U whether Akara's respec works on d2d).

## Not wanted

A fork-side per-level skill cap. d2d's `can_learn` matches the decompiled game.exe icon check and has no such rule; the bot enforces nothing (the knob `resource.skillCapPerLevel` was retired 2026-11-27).

## Rebase routine note

When the routine rebases: re-run `go test ./internal/d2dctl -run 'BaseAttrs|PointVerbs'` (recorded replies) and, on a lab d2d, the live checklist 1 to 3 of `docs/points.md` section 7.
