# Amazon gap list against TASK-941 (card cls2, 2026-11-28)

> status: current offline audit. Tags V (verified: a test or a code read this run), S (secondary: an SME doc of this repo), U (unverified). Nothing here ran in the lab; every scenario below is validated only (`go run ./cmd/d2scenario validate`: ok; `smoke`: 0 blocked). Lab setups are CHEATED stand-ins: mechanics only, never evidence about balance. Inputs: fable review 8 (`docs/design/fable-review-2026-11-28/08-builds-as-data.md`), SME review `docs/probes/sme-other-classes-review-2026-11-27.md`.

## Found and fixed as data (this branch, V by tests)

| Gap | Fix | Test |
|---|---|---|
| `act2plan.json` called the Amazon a caster (Sorceress life gates 90 to 150) while `classtraits.json` says `caster: false` and the admission view derives its archetype from the trait (review 8 C5) | archetype `melee`, the melee slices (entries 20 to 24, life gates 160 to 220, S/U); `levelplan-normal.json` carries the melee row set (review-only, `active: false`) | `internal/plandata` `TestPlanArchetypeFollowsClassTraits` (every class) |
| No merc aura row: the aura was the default row | `mercAura` Defiance / Might / Defiance (S: `docs/classes/amazon.md` section 5, `docs/sme/class-amazon.md` section 5; Might is Nightmare-only, V in `docs/sme/merc-auras.md`) | `internal/d2` `TestMercAuraHireableInItsDifficulty` |
| `internal/decide` answered "unknown class" at every entry point (review 8 section 4) | class rows: mana band 25/50 (S low edge: swap to Fire Arrow under 25 percent, class-amazon section 3; high edge U), zone lead 1 (S: fragile), stance knobs = the ranged (caster) values (U) | `internal/decide` `TestDecideAnswersForAmazonAndAssassinBuilds` (bow, javelin, multishot tables) |
| The ammo path was a Go switch over three table names (review 8 B6): a new Amazon table got no restock | `"ammo": "bow"` or `"javelin"` key in the three tables, read by `leveler.TableAmmo`; an unknown path is a parse error | `internal/leveler` `TestTableAmmoIsData` |
| Nothing checked that a table lands on the profile of its weapon line | no data change needed (bow `amazon`, `amazon/multishot`, `amazon/javelin` already keyed by fix8) | `internal/combat` `TestCombatKeySelectsTheProfilePerTable`: three tables, three profiles, every main attack in a band |
| Zero scenarios named the class | `tools/provingground/scenarios/amazon-assassin.json`: six Amazon rows | `d2scenario validate`, `smoke`, `setup-golden` (hero resolves for both tables) |

## Acceptance criteria

| AC | State | What is left |
|---|---|---|
| 1 traits, plan table, combat profile data-only, two-build test | Done offline (V): traits with merc aura, 3 tables with class, combat and ammo keys, 3 profiles; tests run on bow, javelin and multishot. No class or skill literal added in Go; one removed (the ammo switch) | `combat/gearcheck.go` and `combat/rules.go` still carry older Amazon skill literals (audit rows, not this card) |
| 2 leveling clvl 1 to 30 | Needs live drill | `amazon.level-1-30` (fresh hero, no_writes, level 30, deaths max 6 U). The bow table needs a bow and arrows bought at the first town visit (start gear is a javelin stack; equip click fixed and verified live by TASK-182) |
| 3 allocation locked, spends every point | Offline: capability rows have level and prerequisites (fix8, S) and the level now equals the skills.txt extract for every damage skill (V, `TestCapabilityReqLevelMatchesSkillExtract`); the rows above the per-level cap are listed in `data/build-cap-ratchet.json` (U on Game.exe) | Needs live drill: `points.amazon-bow-normal30-lands-at-30`, `points.amazon-javelin-normal30-lands-at-30` |
| 4 kit and economy, no 3 h death spiral | Offline: melee-archetype kit rows, `manaNeed` 1, soak belt three life and one mana | Needs live drill: `amazon.soak-3h-kit`. Javelin quantity: whether a javelin skill consumes the stack is U (SME change 4), so the javelin restock trigger in `chores-defaults.json` is not added; TASK-204 (no javelins and no gold: idle swinging) is Deferred |
| 5 quest drills GREEN per act | Needs live drill; no rows written | clone `necro-acts.json` with a ranged stand-in kit once `amazon.soak-3h-kit` shows the class fights; the drill matrix rows take `-class Amazon` |
| 6 fresh uncheated live Normal run | Needs live drill | `amazon.earned-act2` (goal `campaign:normal_act2`); record N and the real clvl |
| 7 known blockers closed | Open | javelin quantity read (live, one probe); Valkyrie is a buff of every Amazon profile and no normal30 table buys it (inert, SME); Act 2 life gates 160 to 220 bind on Vitality for a hero with 96 life from levels at clvl 24 (S): the earned run tells whether she is admitted at all. Nothing needs the fork today |

## Exact commands (lab, night window, from a labrun-synced tree; none run here)

```
tools/provingground/run.sh -id 'points.amazon-' -class Amazon -seed 11 -parallel 1
tools/provingground/run.sh -id amazon.soak-3h-kit -class Amazon -seed 11 -parallel 1
tools/provingground/run.sh -id amazon.level-1-30 -class Amazon -seed 11 -parallel 1
tools/provingground/run.sh -id amazon.earned-act2 -class Amazon -seed 11 -parallel 1
```
Order: the two table rows first (10 virtual minutes each; they prove the table, the profile and the keys resolve in the game), then the soak, then the two fresh-hero rows. The `-id` prefix form is U for this runner (same caveat as `docs/classes/necromancer-gap-2026-11-27.md`).

## Proposed, not done
- A third admission archetype (`ranged`) with its own floors: today a bow Amazon is judged by melee floors. Needs a column in `admission.json` and one Go read of a trait; a design decision, not data.
- Per-build class traits (review 8 C7, C8): merc aura and caster are build facts; this card kept them per class.
- ADR/decision text: none proposed.
