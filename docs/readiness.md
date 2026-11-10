# Readiness model: is this hero ready for that area

> status: slice A of the friends beta (cards TASK-217 area table, TASK-218 model, TASK-220 meter feed in part). Read-only. No gate, retreat or boss row reads it yet (TASK-219, TASK-222 later). Design source: [research/player-power.md](research/player-power.md) Part 3.
> Tags: V verified in this tree or by test; U unverified (judgment numbers, formulas); S secondary.

## What exists

- `internal/readiness/areatable.go`: `BuildAreaTable(db)` derives, per area and difficulty, the monster level (V: levels.txt MonLvl Ex columns, same source as `Runner.areaMLvl`), the elements some regular spawn is immune to (V: `data.Monster.Immune`, resist 100 or more), the elements an elite modifier could make a spawn immune to (rules from [difficulty.md](difficulty.md) section 4: elemental enchant +75, spectral hit +40 on poison), the affix names that can spawn (Magic Resistant not in Normal), and the largest regular spawn life, single hit and attack rating. The life, hit and attack rating scaling by monlvl.txt percent columns is U (the reading of the columns, no live measurement). The table is generated at run time from the install tables; the game data is never committed (docs/decisions.md). Offline tests use a tiny fixture DB; `TestRealAnchors` checks Blood Moor 1/36/67 against difficulty.md section 2.1 when an install is at `/tmp/d2mpq` (skipped otherwise; NOT run in this slice, no install on the build box: U).
- `internal/readiness/checks.json` (embedded): check rows as data. Per row: `id`, `kind`, `threat`, parameters per difficulty, optional per-class overrides. Per class: its damage types (U). Kinds (Go knows kinds, never classes or skills): `level_gap`, `resist_floor`, `life_vs_hit`, `hit_chance`, `time_to_kill`, `immunity_cover`, `kit`, `durability`. All numbers are U judgment, to be calibrated against recorded deaths (card ready-calibrate).
- `Evaluator.Evaluate(hero, area)` returns one pass/warn/fail/skip per row with the numbers in the reason, and a light: any fail red, any warn amber, else green. `skip` means the input is not measured (defence, damage per second, durability) and does not colour the light.
- Unknown class, difficulty or area, an unknown kind, a missing parameter: an error naming it. Classes today: sorceress, paladin (add a class = add data).
- `GET /api/readiness[?area=Name&difficulty=hell]` (new endpoint, existing ones unchanged, [ui-design-system.md](ui-design-system.md) section 0). Without `area` it answers for the area the hero stands in. Mounted when the game tables are loaded.
- The risk meter's `ResistGap` (always 0 before) now comes from the resist row (points short of the warn line); 0 when no evaluator is wired.

## The open question: does Player.Resist include the difficulty penalty

Not settled (TASK-216 needs a live read; none was done here). The evaluator takes it as an explicit input, `Hero.Basis`: `ResistRaw` (the numbers are before the penalty; the penalty is applied here and raw values cap at 75 first) or `ResistWithPenalty` (the numbers are as shown). The default is `ResistRaw`, which is what `combat.PlayerResists.Effective` assumes today. Tests cover both cases; the choice for the bot is one constant (`readiness.DefaultResistBasis`). U until TASK-216.

## Not wired (honest list)

Defence, damage per second and gear durability are not read by the bot yet, so those three rows skip live. Gates (`areaRulesWhy`, `BossGate`, route guards), retreat, the whyline templates, boss rows and the strictness policy are cards TASK-219, 221, 222, 223. The meter shows its existing rows plus the live resist gap; making every check a meter row is the rest of TASK-220.

## Live checklist (not run)

1. With a Nightmare character: `curl -s localhost:<viewer port>/api/readiness` and compare the resist line with the character sheet; settles `Basis` (TASK-216).
2. Walk into a cave with known immunities; check the immunity row names them.
