# Fable review 8 of 8: builds as data (2026-11-28)

> status: review, decides nothing. Read-only pass over `origin/main` at `415b11f25`, "today" = the commits after `75e4c2f4e` (2026-11-26 23:54 EST). No Go changed, no game. Tags: V = read in the repo or run here (said how), S = reviewer knowledge of classic 1.14d, U = not verified.

## 1. Verdict in plain words

- **Go side: clean today.** The 12,893 non-test Go lines added today contain no new `d2.<Class>` comparison, no new `"sorceress"`-style string switch and two skill-name literals (both merc auras). The rule "builds are data" was respected by today's code.
- **Data side: not consistent.** The rule moved the risk into JSON, and nothing checks the JSON as a whole. The capability table has no level or prerequisite facts for 53 Amazon and Assassin skills, 12 plan tables contain rows the per-level skill cap would refuse, the Assassin's main attacks from clvl 12 are in no attack band, and the Act 2 plan calls Amazon and Assassin casters while the class traits say they are not.
- **Playable end to end today: Paladin and Sorceress only**, with the Necromancer summoner close. Druid, Barbarian, Amazon and Assassin have tables and profiles but no decide rows (every decide call returns "unknown class"), at most three scenarios each, and copied plan rows nobody judged.

## 2. Task 1: new literals and fallbacks in today's Go

Method (V): `git diff -U0 75e4c2f4e..HEAD -- 'internal/**/*.go' 'cmd/**/*.go'` without tests, added lines only, comment lines dropped; grep for `d2.<Class>`, quoted class names, `Class ==`, quoted names of every skill in `classdata/skills.json`, level comparisons against a number.

| # | sev | file:line | finding | tag |
|---|---|---|---|---|
| A1 | medium | `internal/task/act2.go:448` | new today: `aura := "Defiance"` as the Go default of the hire step. It has no difficulty, so an actor without `MercAuraChooser` hires Defiance in Nightmare where the window has Thorns, Holy Freeze and Might (`docs/sme/merc-auras.md` table). | V |
| A2 | medium | `internal/town/merc.go:177-181` | `HirePolicy` got the class parameter today but kept the default ladder in Go (`"Defiance"`, `"Might"`, floors 400 / 8000 / 15000). Six of seven classes have no `mercAura` row, so for them the aura is still a Go literal. | V |
| A3 | low | `cmd/d2bot/mercwire.go:1043` | new today: probe offer `Aura: "Cold Arrow"` for the log line. Text only, but it is a skill literal the rule forbids. | V |
| A4 | low | `internal/combat/druid.go:239`, `internal/combat/necro.go:415`, `internal/combat/minions.go:361` | today's lines extend class-named drivers (`isDruid(p) && mode != 2`, `isNecro`, `CountAssassinPets`). The minion driver is data (`minions` block), the switch that picks the driver is still a class check. | V |

Unknown-class paths, checked by reading (V): `d2.TraitsOf`, `town.HirePolicy`, `diffready.Check`, `supply.NeedFor`, `decide` (`context.go:58,82`, `policy.go:230`, `firsts.go:361`, `proposezone.go:220`), `scenario/hero.Resolve` and `advisor.Advise` (empty class) all return an error naming the class. None of today's changes removed an error.

Older silent fallbacks still in the tree (V by `git blame`, 2026-11-01 or older; not in the audit's 39 rows unless noted):

| # | sev | file:line | finding |
|---|---|---|---|
| B1 | high | `internal/combat/classdata.go:568` | `SelectProfile` ends in `return ProfileFor(d2.Sorceress, "")`. Logged once, then the hero fights with the Sorceress profile. This is the exact thing the rule forbids. |
| B2 | high | `cmd/d2bot/class.go:70` | on `cp.Err` the fighter "keeps its default" and the bot goes on to fight; line 27 of the same file says the caller must not fight on a guess. |
| B3 | medium | `cmd/d2bot/class.go:37`, `internal/combat/classdata.go:428` | every class starts with combat id `blizzard`, and a class default profile answers to `blizzard` and `lightning`. A table without a `combat` key therefore resolves silently: 11 of 31 tables have no key (three Paladin leveling tables, both Assassin tables, Amazon bow, Barbarian melee, Druid fire, the Sorceress endgame table, two legacy). |
| B4 | medium | `internal/advisor/advisor.go:153` | only the empty class is an error; a misspelt class silently gets the class-free rows. |
| B5 | low | `internal/d2/classtraits.go:74,78` | `Caster()` reads false and `ManaNeed()` reads 1 for an unknown class. |
| B6 | low | `cmd/d2bot/class.go:88-95` | `ammoPath` is a switch over three Amazon table names; a new Amazon table gets no ammo path. |
| B7 | low | baseline | 62 quoted skill names in 24 non-test Go files (V, same grep over the whole tree). Largest: `combat/rules.go` 13, `combat/druid.go` 8, `combat/castgap.go` 5, `merc/policy.go` 4. `combat/trap.go:32` (`trapLadder`), `combat/assassin.go:70`, `combat/druid.go:107` and `combat/gearcheck.go:71` (`dualWield`) are not in `docs/research/build-flexibility-audit.md`. |

## 3. Task 2: data layer consistency

### 3.1 The check (described; run here as a throwaway script, not committed)

Inputs: every `internal/plandata/build-*.json`, `classdata/profiles.json`, `classdata/skills.json`, `data/capabilities/capabilities.json`, and the extracted `skills.txt` rows (`gear.json` in the tables cache, 210 class skills with class, `req_level`, `req`, id).

Per table: (1) `class` present; (2) `(class, combat)` resolves to exactly one profile; (3) every row skill is a skill of that class; (4) every prerequisite has a point before the row; (5) points in a skill never exceed `clvl - req_level + 1` and 20; (6) level points spent by a row never exceed `clvl - 1`, quest points 12; (7) each `hotkeys` and `main_attack` skill is trained by the table, and not before its `from_clvl`; (8) each `main_attack` skill is in a band of the profile, not only in its layout; (9) each band `requires` and each buff group is trained by at least one table that uses the profile. Per profile: every name is a skill of the class, has a `skills.json` row, band attacks and auras have a key. Per capability row: class, `reqLevel`, `reqSkills` equal skills.txt.

Recommended home: one test in `internal/plandata` (or `cmd/d2bot` next to `TestMainAttackSkillsHaveAKey`) that runs on the worker host where the tables cache exists, failing on (1) to (4) and (7) to (9), logging (5) until the question in 3.2 C2 is answered.

Current Go state (V): `tools/agentrun.sh fr8 -- go test -p 1 ./internal/leveler ./internal/plandata ./internal/combat/... ./internal/d2 ./internal/diffready ./internal/charms ./internal/advisor` all `ok`, exit 0. So none of the findings below is caught by a test.

### 3.2 Findings

| # | sev | where | finding | tag |
|---|---|---|---|---|
| C1 | high | `data/capabilities/capabilities.json` (for example `Magic Arrow` at 4619) | 53 skills have no `reqLevel` and no `reqSkills`: 26 Amazon, 27 Assassin (plus the misnamed `Fire Golem` row at 4323; skills.txt says `FireGolem`). Seven skills are absent: Critical Strike, Dodge, Avoid, Evade, Claw Mastery, Weapon Block, Dragon Flight; six of them are bought by shipped tables. The rows say why: "class is S, no skills.txt on the build box". Any consumer that asks the capability table whether an Amazon or Assassin skill is legal gets zero. The other 150 rows match skills.txt exactly. | V |
| C2 | high | 12 tables, 119 rows; worst `build-paladin_hammerdin_endgame90.json:229` | rows above the per-level cap. The Hammerdin table spends 11 banked points on Blessed Hammer at clvl 18 where the cap allows 1; also `paladin_hammerdin_guide` 16 rows, `assassin_sentry_endgame90` 20, `assassin_fireblast_normal30` 16, `necromancer_poison_nova_sme` 14, `druid_tornado_endgame90` 11, `amazon_multishot_endgame90` 9, `amazon_javelin_normal30` 8, `barbarian_frenzy_guide` 6, `barbarian_warcry_endgame90` 5, `druid_fire_guide` 1, `paladin_hammerdin_zeal_endgame90` 1. `docs/points.md:34` records that d2d does not enforce the cap and that Game.exe is U; `leveler.DBRule` (`autoalloc.go:28`) does not model it. | V rows; S that retail enforces it (required level of a skill rises by one per point; this is why kolbot's AutoSkill has the test) |
| C3 | high | `profiles.json:1033` (assassin) | the only band attacks with `Fire Trauma` and `Attack`. Wake of Fire, Charged Bolt Sentry and Lightning Sentry are `main_attack` of both Assassin tables from clvl 12 and 24 and sit only in the layout; they reach the game only through the Go trap driver (`combat/trap.go`, header: "no Assassin ever laid a trap"). `TestMainAttackSkillsHaveAKey` passes because a key alone satisfies it. Same shape: `Corpse Explosion` (summoner tables) and `Raven` (tornado). | V |
| C4 | high | `cmd/d2bot/necroplay_test.go:32,57` | the test the brief names is weaker than its name. It skips the 14 tables with no `main_attack`, among them all three Paladin leveling tables (the live hero's), every `*_guide` and `*_sme` table and the Sorceress default; it accepts "in a band" without a key; it asserts table counts for Necromancer and Sorceress only. | V |
| C5 | medium | `internal/plandata/act2plan.json:552,679` | Amazon and Assassin carry `"archetype": "caster"` with the Sorceress life gates (90 to 150); `internal/d2/classtraits.json` says `caster: false` for both and `bot/admission.go:45-56` derives melee from the trait. Two sources, opposite answers. The life gates are minimums written for a 150-life Sorceress, so for these classes they never bind. | V |
| C6 | medium | `act2plan.json:171` (barbarian), `profiles.json` barbarian `melee` blocks | Barbarian rows are Paladin copies: same life gates, and `manaReserve: 4`, `surround: 4` in all three Barbarian profiles. SME change 3 and 7 of `docs/probes/sme-other-classes-review-2026-11-27.md` (warcry surround 6, lower life floors) are not applied. | V |
| C7 | medium | `internal/d2/classtraits.json:5`, `act2plan.json:21` | merc aura has three homes: `classtraits.mercAura` (Necromancer only), the Go default (A1, A2) and `act2plan.merc.aura` (`melee`/`caster`, read by nobody per `docs/design/merc-aura-by-archetype.md:3`). It is keyed by class, yet it is a build fact: a bone Necromancer does not want Might for an army it does not have. | V; S for the last clause |
| C8 | medium | `classtraits.json` `manaNeed` vs table `mana_heavy` | the Hammerdin and the War Cry Barbarian flag `mana_heavy: true`, their class trait is `manaNeed` 1 and 0. `resource.Input.manaNeed` raises it only outside Normal (`manaheavy.go:65`); `bot/kitgate.go:42`, `bot/routeguard.go:64`, `combat/hard.go:59`, `combat/rules.go:104` and the supply rows (`manaNeedAtLeast: 2`, fed by `town/planshadow.go:84`) read the class value. A Normal Hammerdin gets no mana kit line. | V |
| C9 | medium | `internal/bot/diffready.go:124` | `KeySkillLevel: -1` always, so `key_skill_min` (set for six classes) is never checked; it lands in "unread" and does not block. The rows are per class, not per build ("key skill" of which Paladin?). Hell rows are Nightmare life times 1.4 and can still print "ready for hell (U)". | V |
| C10 | medium | table names | seven tables have a file name and an inner `build` that differ (`paladin-normal30-v2` vs `paladin_normal30_v2`, the three lightning tables, `primary-blizzard`). `act2plan.json` and `levelplan-normal.json` name the Paladin table by the inner form, `leveler.buildFiles` by the file form, `resource.loadManaHeavy` keys by the inner form and is asked with the file form. Works today by luck (none of the seven is mana-heavy). | V |
| C11 | low | 25 hotkey rows, 22 main_attack rows | `from_clvl` earlier than the table trains the skill (Hammerdin: Concentration key from 18, trained 32; Blizzard endgame: Blizzard from 24, trained 28; summoner endgame: Corpse Explosion main attack from 6, trained 16). | V |
| C12 | low | `fallback-lightning`, `primary-blizzard` | no `class` key: still "no class means Sorceress" (audit row 22). | V |
| C13 | low | `internal/advisor/advisor.json` | rows are keyed by class, not build: Leaf (fire staff) for every Sorceress, Steel, Strength and Honor for a Hammerdin. Rune lists and base types look right (S). `internal/charms/charms.json:11-12` lists Annihilus and Hellfire Torch, which an offline 1.14d character cannot obtain (S). | V / S |
| C14 | low | `tools/provingground/scenarios/combatdrills.json` | eight scenarios set `pre.skills` by numeric id (151, 54, 74, 112, 59, 261, 276). All seven ids belong to the matrix class (V against skills.txt), but the rule says names, and each grants a skill without its prerequisites (declared CHEATED). | V |

Clean (V): no row skill of a wrong class, no missing prerequisite in a no-respec table, no level-point overspend, no duplicate layout key, every `trigger.build` in the 263 scenarios names an existing table.

## 4. Task 3: who can play end to end today

Columns: L leveling table, E endgame table, P own combat profile reached by a `combat` key, D decide rows (`catalogue.json`, `proposezone.json`), LP levelplan row, A2 act2plan row, M merc aura row, Sc scenarios naming the class (of 263), SME verdict 2026-11-27. All V by reading the files; "playable" is the reviewer's judgment (S).

| class (user order) | L | E | P | D | LP | A2 | M | Sc | SME | playable today | blockers |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Necromancer | summoner, bonespear | yes | summoner, bonespear, poisonnova | yes | Sorceress copy, inactive | Sorceress copy | yes | 60 | summoner sensible; bonespear two defects (applied) | nearly: summoner in Normal | Lower Resist, Decrepify, Weaken have no key in the summoner layout; policy rows are caster kite values (`kiteLifePct` 85); no Necromancer seed set; C2 for poison nova |
| Druid | fire | tornado | fire, tornado | **no** | Sorceress copy, inactive | Sorceress copy | no | 3 | fire sensible; tornado two unreachable skills | no | decide errors on the class; C2 (tornado 11 rows); Grizzly and Dire Wolf have no key; 3 scenarios |
| Sorceress | cold, fire, lightning | blizzard | four | yes | own, inactive | own | no | 148 | cold sensible, two defects | yes (Normal Act 1 proven) | endgame table has no `combat` key and resolves through the `blizzard` default (B3); `Energy Shield` buff group has no key |
| Barbarian | melee | warcry | default, frenzy, warcry | **no** | melee rows, inactive | Paladin copy | no | 3 | melee "depends"; warcry was NOT sensible (kind fixed today, U live) | no | decide errors; Frenzy needs two weapons (U); Paladin melee numbers (C6); leveling table attacks have no key in table or profile |
| Amazon | bow, javelin | multishot | default, javelin, multishot | **no** | Sorceress copy, inactive | Sorceress copy as caster (C5) | no | **0** | bow NOT sensible 18+ (band order applied today) | no | C1 (no capability facts); decide errors; zero scenarios; javelin ammo trigger missing; `ammoPath` Go switch (B6) |
| Assassin | fireblast | sentry | one default | **no** | Sorceress copy, inactive | Sorceress copy as caster (C5) | no | 1 | plan sensible; Death Sentry key added today | no | C1; C3 (sentries in no band, trap driver never ran live); C2 (36 rows); decide errors |
| Paladin (live hero) | normal30-v2, zealot-lite | hammerdin, hammerdin_zeal | default, hammerdin, hammerdin_zeal | yes | own, **active** | own | no (Go default) | 198 | hammerdin NOT sensible clvl 3-17; zeal table wrong filler (partly applied) | yes | C2 at clvl 18 if it ever runs on Game.exe; C8 (no mana kit in Normal for the Hammerdin); C4 (its leveling tables are outside the key test) |

## 5. Task 4: ten fixes, in order

1. **Capability facts for Amazon and Assassin.** `data/capabilities/capabilities.json` (rows such as 4619): generate `reqLevel` and `reqSkills` for the 53 rows from skills.txt, add the seven missing skills, rename `Fire Golem` (4323). Add the equality test of 3.1 on the worker host.
2. **Decide the per-level cap once.** One live read on Game.exe (a banked point into a skill at `req_level` + 0 with one point already in it). If refused: add the term to `internal/leveler/autoalloc.go:28` and re-row the 12 tables, starting at `build-paladin_hammerdin_endgame90.json:229`. If accepted: delete the U in `docs/points.md:34`.
3. **Remove the Sorceress fallback.** `internal/combat/classdata.go:568`: return the error to the caller; `cmd/d2bot/class.go:70`: do not fight when `cp.Err` is set (park with the reason).
4. **Require a `combat` key.** `cmd/d2bot/class.go:37` and `internal/combat/classdata.go:428`: drop the `blizzard`/`lightning` aliasing for class defaults; add `"combat"` to the 11 tables without one and `"class"` to the two legacy tables.
5. **Make the key test mean what it says.** `cmd/d2bot/necroplay_test.go:32,57`: fail on a table without `main_attack`, require band membership (and a key where the driver needs one), assert at least one table per class.
6. **Assassin bands.** `internal/combat/classdata/profiles.json:1033`: bands at 12 and 24 with the sentries, or a `traps` block that the trap driver reads instead of `trapLadder` (`internal/combat/trap.go:32`).
7. **One archetype source.** `internal/plandata/act2plan.json:552,679`: drop `archetype` from the file and derive it from `classtraits.json`, or set both to melee with judged life gates; apply SME changes 3 and 7 for the Barbarian (`act2plan.json:171`).
8. **Merc aura fully in data.** Move the default ladder of `internal/town/merc.go:177-181` and `internal/task/act2.go:448` into a `default` row of `internal/d2/classtraits.json:5`, delete `act2plan.json:21`, and allow a build table to override (`merc_aura`).
9. **Mana need from the build.** `internal/bot/kitgate.go:42`, `internal/bot/routeguard.go:64`, `internal/resource/manaheavy.go:65`: one function that takes class and build; drop the "not in Normal" exception or justify it in the table.
10. **Readiness key skill.** `internal/bot/diffready.go:124`: read the key skill level from the build's `main_attack` at the hero's level, key the rows by build, and never print "ready" from a U row.

## 6. Limits of this review

- Nothing ran live. The cap question (C2) is the one finding whose severity depends on a fact not in the repo.
- The data check used the tables cache of the review box; the script is described in 3.1, not committed (docs-only brief).
- Playability is judged from data and SME text, not from soaks; "yes" for Paladin and Sorceress rests on the existing Normal runs, not on anything measured today.
