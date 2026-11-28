# Assassin gap list against TASK-947 (card cls2, 2026-11-28)

> status: current offline audit. Tags V (verified: a test or a code read this run), S (secondary: an SME doc of this repo), U (unverified). Nothing here ran in the lab; every scenario below is validated only (`go run ./cmd/d2scenario validate`: ok; `smoke`: 0 blocked). Lab setups are CHEATED stand-ins: mechanics only, never evidence about balance. Inputs: fable review 8 (`docs/design/fable-review-2026-11-28/08-builds-as-data.md`), SME review `docs/probes/sme-other-classes-review-2026-11-27.md`.

## Found and fixed as data (this branch, V by tests)

| Gap | Fix | Test |
|---|---|---|
| `act2plan.json` called the Assassin a caster while `classtraits.json` says `caster: false` (review 8 C5) | archetype `melee`, the melee slices (S/U); `levelplan-normal.json` carries the melee row set (review-only). The SME asked for caster life floors for a trapper: not applied, the trait is per class and the runtime enforces the melee floors either way | `internal/plandata` `TestPlanArchetypeFollowsClassTraits` |
| No merc aura row | `mercAura` Defiance / Holy Freeze / Defiance (S: `docs/sme/class-assassin.md` section 5: a tank in Normal and Hell, the slowing aura in Nightmare; `docs/classes/assassin.md` section 5) | `internal/d2` `TestMercAuraHireableInItsDifficulty` |
| `internal/decide` answered "unknown class" | class rows: mana band 30/60 (S: mana is the binding constraint, class-assassin section 3), zone lead 1 (S/U), stance knobs = the ranged values (U) | `internal/decide` `TestDecideAnswersForAmazonAndAssassinBuilds` (fire blast and sentry tables) |
| Capability rows `Inferno Sentry` and `Royal Strike` had no level or prerequisite | level 24 and 30 from the skills.txt extract in `internal/power/skilldamage.json` (V), prerequisites from the SME table through the display names Wake of Inferno and Phoenix Strike (S); `docs/sme/capability-gaps.md` | `internal/plandata` `TestCapabilityReqLevelMatchesSkillExtract` |
| Death Sentry's corpse and the Necromancer corpse spells: nothing tested the shared reservation from the trap side | no data change (the `traps` ladder and the f8 key are fix8's) | `internal/combat` `TestDeathSentrySharesTheCorpseReservation`: a reserved corpse is skipped, the corpse under a new sentry is reserved; `TestCombatKeySelectsTheProfilePerTable`: both tables on one profile, every sentry they name is on its ladder |
| One scenario named the class (`combat.traps-stand`) | `tools/provingground/scenarios/amazon-assassin.json`: six Assassin rows | `d2scenario validate`, `smoke`, `setup-golden` |

## Acceptance criteria

| AC | State | What is left |
|---|---|---|
| 1 traits, plan table, combat profile data-only, two-build test | Done offline (V): traits with merc aura, 2 tables, one profile with the trap ladder as data; tests run on both tables | There is no martial-arts table or profile: the trap and martial split is one build today. `Dragon Flight` has no capability row (U, capability-gaps.md) and `combat/rules.go` names it wrongly as a fire fallback (audit row) |
| 2 leveling clvl 1 to 30 | Needs live drill | `assassin.level-1-30` (fresh hero, no_writes, level 30, at least one `combat trap` line, deaths max 6 U). The trap planner is UNVERIFIED LIVE (`internal/combat/trap.go` header) |
| 3 allocation locked, spends every point | Offline: capability levels equal the extract for every damage skill (V); the rows of both tables above the per-level cap are listed in `data/build-cap-ratchet.json` (U on Game.exe) | Needs live drill: `points.assassin-fireblast-normal30-lands-at-30`, `points.assassin-sentry-endgame90-lands-at-30` |
| 4 kit and economy, no 3 h death spiral | Offline: `manaNeed` 1, soak belt two life and two mana. SME change 3 (`mana_heavy` for the sentry table: 20 mana per trap, five per type) is not applied: it changes the belt of a table nobody ran | Needs live drill: `assassin.soak-3h-kit`; then decide `mana_heavy` from the mana-out count |
| 5 quest drills GREEN per act | Needs live drill; no rows written | clone `necro-acts.json` with a trapper stand-in kit after `combat.traps-stand` and the soak are GREEN |
| 6 fresh uncheated live Normal run | Needs live drill | `assassin.earned-act2`; record N and the real clvl. The Blood Moor exit refusal is fixed and verified live (TASK-184) |
| 7 known blockers closed | Open | whether sentries appear in the snapshot as friendly units (U, trap.go header: the cap of five and the cover check depend on it); Death Sentry at clvl 30+ only, so no Normal 1 to 30 run exercises the corpse path (`combat.traps-stand` does, CHEATED setup). Needs fork: nothing known; a read verb for standing traps would be the fix if the snapshot does not show them |

## Exact commands (lab, night window, from a labrun-synced tree; none run here)

```
tools/provingground/run.sh -id 'points.assassin-' -class Assassin -seed 11 -parallel 1
tools/provingground/run.sh -id assassin.soak-3h-kit -class Assassin -seed 11 -parallel 1
tools/provingground/run.sh -id assassin.level-1-30 -class Assassin -seed 11 -parallel 1
tools/provingground/run.sh -id assassin.earned-act2 -class Assassin -seed 11 -parallel 1
```
Order: the two table rows first (10 virtual minutes each; they prove the table, the profile and the keys resolve in the game), then the soak, then the two fresh-hero rows. The `-id` prefix form is U for this runner (same caveat as `docs/classes/necromancer-gap-2026-11-27.md`).

## Proposed, not done
- A martial-arts table and profile (charge-up and finisher grammar) is new combat logic, not data.
- Buff order Fade before Burst of Speed (SME change 2, group semantics unsure) and the clvl 12 to 23 re-row of the sentry table (SME change 4, confidence low) are left for a run to judge.
- ADR/decision text: none proposed.
