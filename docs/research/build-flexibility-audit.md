# Build flexibility audit: what is fitted to the one Sorceress build (agent BUILDFLEX, 2026-11-01)

> status: read-only audit and design, decides nothing. Card `buildflex`. Answers the user direction of 2026-11-01 ("we're focusing on one sorc build for now which is good, but I want this to be far more flexible at the end and have multiple builds, so can we make sure we're not hyper focusing/hard coding anything based on this one sorc build"). Builds on the class-level study `docs/research/bot-generality.md` (framework or stencil), which counted class branches; this doc goes one level down, to the build. Tags: V = read or grepped in this worktree (file:line given), U = judgment. Nothing was run live.

> status update 2026-11-01: rows 1, 2, 3, 6 and 29 are fixed by card palprobe (profile id string, per-class prefs, class row errors); see `docs/classes/paladin-probe.md`.

## 1. Summary in plain words

**How hard-coded is it today?** Less than feared at the data layer, more than hoped at the glue. The bot already treats a "build" as a data file for every class: 20 plan tables in `internal/plandata`, a per-class combat profile file (`internal/combat/classdata/profiles.json`), and a skill-aiming table (`skills.json`). The default lightning Sorceress is not baked into the leveler's stat/skill spending; that part is generic and reads any table. The fitting sits in the small pieces of Go that decide *which* table and *which* profile go together, and in a handful of combat rules that name Sorceress skills by string.

**What is ready.**
- Every class has a plan table; seven have a clvl 2..90 "endgame" table, including a second Sorceress build (`sorceress_blizzard_endgame90`, cold, Blizzard) that is selectable today with `--build endgame` or the config key `levelerBuild`.
- The combat profile (attack order per level band, buffs, F-key layout) is data, for all seven classes.
- Skill facts (mana, level, prerequisites) come from the game's own tables, not from code.
- Stat points, skill points and quest points follow the table; the planner does not know skill names.

**What blocks a second build.** Three things, all small but silent:
1. The combat side only knows two Sorceress builds, as a two-value Go enum (Blizzard, Lightning). A third Sorceress build (fire, a different cold build) cannot be chosen, and picking any table other than the two lightning ones silently gives the Blizzard profile.
2. The table says what the build plays (it carries `main_attack` and `hotkeys`), the profile file says it again, and the code reads only the profile. The two can drift, and the table's version of the truth is parsed and thrown away.
3. Three combat rules name Sorceress skills by string (Static Field opener and fallback, Frost Nova crowd cast) with Sorceress numbers, and Teleport movement is never switched on in the production bot (only test rigs set the key), so a build that depends on Teleport would walk.

**Honest limit on evidence.** The second Sorceress table has never played: no soak, no drill, no mention in `docs/capabilities.md`. "Selectable" is a code fact (V), "works" is not proven. The drills and soaks themselves are Sorceress-shaped (skill ids 36, 38, 40, 45, 49, 54; default class 4), so proving a second build needs the harness generalised first.

## 2. Ranked findings

Severity: **BLOCKS** = a second build would silently misbehave (wrong skills, wrong profile, wrong economy, no warning); **LIMITS** = awkward, a profile or table could override; **COSMETIC** = naming, comments, test coupling. Layers: PD plan data and leveler, CB combat, NV nav and bosses, EC town and economy, HR drills, soaks and entry flags, CF knobs and docs. Seams are named in section 3.

| # | layer | file:line | what it assumes | sev | seam |
|---|---|---|---|---|---|
| 1 | CB | `internal/combat/skills.go:15-18` | `combat.Build` is a two-value enum (Blizzard, Lightning); `Fighter.Build` carries it | BLOCKS | S1: profile id string |
| 2 | CB | `internal/combat/classdata.go:197-211` | `ProfileFor` maps the enum to the strings "blizzard"/"lightning"; a third profile name is unreachable; an unknown class returns the Sorceress profile without a warning | BLOCKS | S1: lookup by id, error on unknown |
| 3 | CB | `cmd/d2bot/class.go:34-37` | only tables `fallback` and `fallback-lightning-v2` flip the combat build to Lightning; any other Sorceress table (the END one, a new one) gets the Blizzard profile | BLOCKS | S1: table declares its profile |
| 4 | PD | `internal/leveler/build.go:79,98` vs `classdata/profiles.json` | the table's `hotkeys` and `main_attack` are parsed (`Build.Hotkeys`) and never consumed in production; `profiles.json` repeats the same facts; two sources of truth that can disagree | BLOCKS | S1: one source (table hotkeys + main_attack feed the profile) |
| 5 | NV | `cmd/d2bot/adapters.go:38,43`, `cmd/d2bot/boss.go:301,410`; set only in `tools/rigs/act2live/main.go:70,934` | `Navigator.TeleportKey` gates Teleport hops, boss `CanTeleport` and `NeedsTeleport` stops; a grep finds no production assignment, so the bot never hops (U: confirm in a drill) | BLOCKS | S3: capability from known skills + layout key |
| 6 | EC | `internal/leveler/prefs.go:30-33` | `PrefsFor` gives every class except Sorceress and Paladin the Sorceress gear preferences (staff, wand, orb bonus) | BLOCKS | S5: per-class prefs table, error on unknown |
| 7 | CB | `internal/combat/fight.go:700,738-754` | Static Field opener: skill literal `"Static Field"`, count and life floors from Sorceress knobs, Blood Raven special case | LIMITS | S2: profile `opener` rule |
| 8 | CB | `internal/combat/fight.go:540,756-763`, `internal/combat/affix.go:142-148` | Static Field is the only fallback when every attack is immune-blocked | LIMITS | S2: profile `immuneFallback` list |
| 9 | CB | `internal/combat/castfirst.go:20-22,95-122` | Frost Nova crowd cast: literal skill, radius 4, crowd 3, gap 2.5 s; needs the skill known | LIMITS | S2: profile `crowd` rule |
| 10 | CB | `internal/combat/classdata/profiles.json` (sorceress/blizzard, first band) | band starts at clvl 26 with Blizzard known; the END table trains Blizzard at clvl 24, so 24 and 25 ignore it; the shared tail band attacks with Charged Bolt first, a skill the END table never trains | LIMITS | S1: derive bands from the table's `main_attack` |
| 11 | CB | `internal/combat/classdata/profiles.json` (sorceress layouts) | layout is a fixed 8-key map per Sorceress profile (Static Field f1, Teleport f2, Frozen Armor f3, Charged Bolt f4); a third build needs a hand-written layout | LIMITS | S1: layout from table `hotkeys` |
| 12 | CB | `internal/combat/necro.go:42-43` | buff state ids: five names in a Go map (Bone Armor, Frozen/Chilling/Shiver Armor, Energy Shield); other buffs (Thunder Storm, Warmth, Hydra) have no state check | LIMITS | S2: `state` column in `skills.json` |
| 13 | CB | `internal/combat/castgap.go:9-25` | cast length model is the Sorceress SC animation (14 frames) and a 21-name skill list; other classes fall back to the flat gap; Lightning/Chain Lightning sequence names are literal | LIMITS | S2: frames per class in `classdata`, skill list from the tables |
| 14 | CB | `internal/combat/rules.go:210-239` | `classOptions`: attack skills by damage type, per class, from memory (U); the Sorceress row lists Fire, Cold and Lightning skills by name | LIMITS | S2: derive from the profile's attack names + skills.txt element |
| 15 | CB | `internal/combat/skills.go:111-113` | `staticRange` special-case in `makeSpec` keyed on the name "Static Field" | COSMETIC | S2: range formula in `skills.json` |
| 16 | CB | `internal/combat/skills.go:123-134` | `attackOrder`, `buffOrder`, `Layout` helpers hard-wired to `d2.Sorceress`; only tests call them | COSMETIC | delete or take a profile |
| 17 | CB | `internal/combat/gearcheck.go:95-98` | only the Amazon reads the table's `main_attack` (`gear.MainAttackAt`); the seam exists but one class uses it | LIMITS | S1: every class reads it |
| 18 | PD | `internal/leveler/build.go:128-151` | name to file map in Go (`buildFiles`); a new table needs a Go edit | LIMITS | S4: scan `plandata`, table names itself |
| 19 | PD | `internal/leveler/build.go:162-170` | `endgameBuild`: one endgame table per class, Sorceress to Blizzard; the `endgame` alias cannot mean fire or cold | LIMITS | S4: `endgame_for` metadata, `--build <name>` already works |
| 20 | PD | `internal/leveler/build.go:196-232` | `freshBuild` pair and `v2Rows = 15`: the legacy/fresh split is hand-coded for Sorceress and Paladin, and the "owns a dropped skill" check only looks at rows up to clvl 15 | LIMITS | S4: `supersedes` + `compat_until_clvl` in the table |
| 21 | PD | `internal/leveler/build.go:255-262` | `defaultBuild`: Sorceress to "fallback" (the legacy table), Paladin to its table; other classes through `gear.DefaultBuild` | LIMITS | S4: `default_for_class` metadata |
| 22 | PD | `internal/leveler/build.go:98-101,292-293` | a table with no `class` field is a Sorceress table; `withEnergy` patches the legacy Lightning table at load time for Sorceress only (stat allocation fixed in Go instead of in the data) | LIMITS | S4: require `class`; fix the data file |
| 23 | PD | `internal/leveler/leveler.go:197-199` | `classFloor`: only Sorceress has a minimum Str (12); the other six classes are "U: not checked" | LIMITS | S5: floor derived from charstats and the cheapest body armor |
| 24 | PD | `internal/leveler/build.go:35-38,43`, `internal/combat/skills.go:21` | respec is "not implemented"; the old primary table needs one at clvl 26; `respecLevel` is a dead constant | LIMITS | out of scope: choose no-respec tables (the END table has none) |
| 25 | EC | `internal/leveler/score.go:26-30,285-300` | +tab skills, +class skills and +all skills are weighted the same for every build; a cold build values +lightning skills equally | LIMITS | S5: table lists `skill_tabs` it wants |
| 26 | EC | `internal/resource/manaheavy.go:14-19` | mana-heavy belt builds are a Go set of four endgame table names; a new mana-heavy table is not seen | LIMITS | S4: `mana_heavy: true` in the table |
| 27 | EC | `internal/resource/types.go:146-153` | `ManaNeed` switch by class (2 for Sorceress, Necromancer, Druid) | LIMITS | S5: class traits table |
| 28 | EC | `internal/bot/task.go:313,315`, `internal/bot/routeguard.go:87,228`, `internal/bot/geargate.go:50`, `internal/merc/policy.go:80` | the triple "Sorceress or Necromancer or Druid" is written out five times, plus `BRHigh` (Sorceress, Druid, Amazon) and the Den floor switch; a class or build that is a caster by skills (a hammer Paladin is not, a Sorceress is) cannot say so | LIMITS | S5: one class-traits table (caster, manaNeed, brHigh, denFloor) |
| 29 | EC | `internal/town/panel/panel.go:74-87`, `internal/town/rebuy.go:90,98,126` | `classRow` returns the Sorceress row for an unknown class; Amazon launcher rules inline in rebuy | LIMITS | S5: class traits, error on unknown |
| 30 | NV | `internal/boss/act1.go:156`, `act2.go:111`, `act3.go:120`, `act4.go:63,85,115`, `act5.go:93,130,152,282`, `farm_scripts.go:63` | 12 boss scripts carry a hand-set `Kite` (range 7 to 10, retreat 10 to 14) tuned for a ranged caster; a melee build would kite away from its own target | LIMITS | S2: profile `ranged` or melee decides, script keeps only the geometry |
| 31 | HR | `internal/drill/d2.go:66,68`; `cmd/d2drill/d15_lab.go:20-25`, `d15b_lab.go:46,330,396`; `internal/drill/d16.go` | drills use literal skill ids (36 Fire Bolt, 38 Charged Bolt, 40 Frozen Armor, 45, 49, 54) and a Sorceress cast time (7 ticks, FCR 0) | LIMITS | S6: resolve ids by name from the character's profile |
| 32 | HR | `cmd/d2drill/main.go:46-47`, `cmd/d2dsoak/run.go:276`, `internal/setup/steps.go:378`, `cmd/d2bot/d2dbackend.go:58` | default class 4 (Sorceress), default character name "Drilly", default `class = "sorceress"` in setup; a non-Sorceress needs flags every time | LIMITS | S6: `--class` and `--build` on every harness, default from config |
| 33 | HR | `internal/resource/sim.go:88-91`, `internal/goals/sim/sim.go:71`, `internal/town/sim/sim.go:107`, `cmd/d2view/sim.go:74-82` | the offline simulators start a clvl 1 or 8 Sorceress with Fire Bolt | COSMETIC | S6: scenario takes a profile |
| 34 | CF | `internal/env/tuning_combat.go:49-50,84,92,125-126` | knobs named after skills (`staticCasts`, `staticCastsBR`, `staticFloor*`, `novaManaFloor`) and `CastFCR: 0` (Sorceress breakpoint) | LIMITS | S2: move into the profile rule, keep the knob as an override |
| 35 | CF | `internal/env/tuning_route.go:30-34`, `docs/knobs.md` | Den and Tools-of-the-Trade knob descriptions say "Sorceress, Necromancer, Druid..." as the rule | COSMETIC | S5: describe by trait |
| 36 | CF | `internal/bot/config.go:67`, `docs/knobs.md:408` | `levelerBuild` is documented as "fallback" or "primary" only | COSMETIC | document `--build <table>` and `endgame` |
| 37 | CF | `docs/capabilities.md:8,27,38,71`, `docs/running.md:83,120` | claims are scoped to one Sorceress, Normal Act 1 (accurate today) | COSMETIC | rescope per build as each is proven |
| 38 | PD | `internal/plandata/plandata_test.go:21-27` | the "has what the consumers load" list names the old tables; the END tables and other endgame files are not in it | COSMETIC | S4: iterate the manifest |
| 39 | HR | `internal/combat/class_test.go`, `bind_test.go`, `castgap_test.go` | tests assert `BuildBlizzard`/`BuildLightning` by enum | COSMETIC | follow S1 |

Not rows, grouped: (a) 67 literal clvl comparisons in policy code (V in `bot-generality.md`) are fitted to the one path's difficulty curve but not to a skill or a build; they are rewritten by the difficulty cards, not here. (b) Combat clvl literals specific to a build: none found (V grep of `internal/combat` and `internal/leveler`: the band minimums live in `profiles.json`, the rest are difficulty gates). (c) Skill ids: the 36 to 65 range appears as literals only in drills and test files plus `d2.SkillFireBolt` and `d2.SkillTeleport` (`internal/d2/skills.go:33-34`, neither used outside tests); production code resolves skills by name through `data.SkillByName`. (d) Belt columns: the plan is per class and difficulty (`internal/town/belt`, `internal/resource/gear.go`), with one build-keyed exception (row 26). (e) Quest points and stat allocation per row are data (`quest-points.json`, the table rows) and class-agnostic.

### Counts by layer and severity (V, from the table above)

| layer | BLOCKS | LIMITS | COSMETIC | total |
|---|---|---|---|---|
| PD plan data and leveler | 1 | 7 | 1 | 9 |
| CB combat | 3 | 9 | 2 | 14 |
| NV nav and bosses | 1 | 1 | 0 | 2 |
| EC town and economy | 1 | 5 | 0 | 6 |
| HR drills, soaks, entry flags, tests | 0 | 2 | 2 | 4 |
| CF knobs and docs | 0 | 1 | 3 | 4 |
| **total** | **6** | **25** | **8** | **39** |

## 3. What already works as data (the foundation)

- **Plan tables are the unit of a build.** `internal/plandata/build-*.json` (rows per clvl with Str/Dex/Vit/Ene and skill points, `class`, `hotkeys`, `main_attack`, optional `respec`), parsed by `leveler.ParseBuild` (`build.go:92`). `Build.Plan` (`plan.go`) spends points from any table and knows no skill name. A new table needs no planner change (V).
- **Quest points and the stat floor** are data and class-agnostic (`quest-points.json`; `Leveler.floor`).
- **The second Sorceress build exists and is selectable.** `sorceress_blizzard_endgame90` (clvl 2..90, no respec, `build.go:146`) is reachable by `--build endgame`, by `--build sorceress_blizzard_endgame90`, or by config `levelerBuild` (`ResolveBuild`, `build.go:176`). The loot and belt code already special-cases it by name (`internal/loot/loot_test.go:444`, `internal/town/belt/belt_test.go:88`). It carries a `Frozen Armor` and `Static Field` plan with `main_attack` bands (Ice Bolt, Ice Blast, Glacial Spike, Blizzard at 24). **Was it ever run? No evidence found:** it appears in no soak report, drill, probe or capabilities row (V grep of `docs/probes`, `docs/capabilities.md`, `docs/g1-log.md`); `docs/plan/STATUS.md` says "opt-in, never a default". Only unit tests touch it. So: selectable (V), unproven (V).
- **Combat profiles are data.** `classdata/profiles.json` has eight profiles for seven classes (two Sorceress), with level bands, `requires` gates, aura ladders, buffs, F-key layout and melee parameters; `classdata/skills.json` gives each skill's kind, hand, range and hold. `SelectProfile` even re-derives the class from the character's skills when the class byte is wrong (`classdata.go:268`).
- **Skill facts come from the game tables** (`makeSpec`, `deriveMeta`): cost, level, prerequisites, a derived aim for skills not in `skills.json`.
- **Per-class endgame tables for all seven classes** (`endgameBuild`, `build.go:162`) and per-class gear builds in `internal/plandata` (the `internal/gear/builds` copies are deleted).
- **Boss scripts and farm runs are data-shaped** (`farm_sched.go` takes `Class` and `Teleport` as inputs; scripts mark `NeedsTeleport` per stop).

## 4. Proposed seams (short design)

A **build** is three pieces of data that must always agree, plus per-class constants. Today they are three files with a Go enum gluing two of them.

- **Build table (leveler data), extended.** `build-*.json` gains a small header, all optional with today's behaviour as the default: `profile` (the combat profile id it plays, S1), `default_for_class`, `endgame_for`, `supersedes` (table name) with `compat_until_clvl`, `mana_heavy`, `skill_tabs` (the tabs worth +skills, for gear scoring). Names come from a scan of `plandata`, not a Go map.
- **Combat profile = class + skill set + rules, as data.** One profile per build, addressed by id string (`sorceress/lightning`, `sorceress/blizzard`, later `sorceress/fire`). It already holds bands, buffs and layout. Add the rules now written as Go for Static Field and Frost Nova: `opener {skill, casts, lifeFloorByDifficulty, special cases}`, `immuneFallback [skills]`, `crowd {skill, radius, minCount, gapMs, manaFloor}`, `ranged` (kite or hold). The existing knobs stay as overrides of these values. The table's `main_attack` and `hotkeys` become the single source for bands and layout (the profile keeps only what a table cannot say: auras, melee params, the rules above), so S1 removes the row 4 drift.
- **Per-class tuning = one class-traits table** (`classdata/classes.json`): caster, manaNeed, brHigh, denFloor kind, cast frames, base Str floor, gear preference weights, inventory row name. Replaces the five copies of the caster triple and the silent Sorceress fallbacks (an unknown class is an error with a name, never a Sorceress).
- **Movement capability from skills, not a key.** Teleport (and later Enigma-granted Teleport on any class) is "the character knows Teleport and the layout binds it": nav, boss `CanTeleport` and farm scheduling read one `CanTeleport()` derived from that, wired in production.
- **Harness takes the build as a parameter.** Drills and soaks resolve skill ids by name from the character's profile and accept `--class` and `--build`; the simulators take a profile.

Seam names used in the table: S1 table-declared profile id (replaces the enum), S2 rules as profile data, S3 teleport capability, S4 table metadata and manifest, S5 class traits table, S6 parameterised harness.

## 5. Roadmap (phased)

Sizes: S under a day, M one to three days, L more than three days of agent work. "Live" marks phases whose result must be shown on the live game, not only offline.

| phase | what | rows | size | live drills |
|---|---|---|---|---|
| 0 | Guardrails now, text in section 6: new cards derive from data, no clvl or skill literal in Go, no silent class fallback. Add a `go test` that fails when a new `d2.Sorceress` or skill-name literal appears outside an allowlist (the allowlist is this audit's rows). | all | S | no |
| 1 | S1: replace `combat.Build` by a profile id; table header `profile`; `planForClass` reads it; unknown profile or class is an error; fix the band mismatch (row 10); feed layout and bands from the table (rows 1 to 4, 10, 11, 16, 17, 39). Offline tests keep today's two profiles byte-identical (golden test on `AttackNames` and `Layout` for every clvl). | 1-4,10,11,16,17,39 | M | no (offline equivalence); one bind check live in phase 6 |
| 2 | S4: table metadata and manifest; remove `buildFiles`, `freshBuild`, `endgameBuild`, `defaultBuild`, `v2Rows` maps; `mana_heavy` and `skill_tabs` from the table; require `class`; fix the legacy energy patch in the data (rows 18 to 22, 25, 26, 38). | 18-22,25,26,38 | M | no |
| 3 | S5: class traits table; delete the five caster triples; unknown class errors; per-class `PrefsFor`, `classRow`, Str floor (rows 6, 23, 27 to 29, 35). | 6,23,27-29,35 | M | no (class floors need one live check per new class) |
| 4 | S2: rules as profile data: opener, immune fallback, crowd, ranged/kite, buff states, cast frames, immune options; move Static Field and Nova knobs into the profile with the old knobs as overrides (rows 7 to 9, 12 to 15, 30, 34). Equivalence golden tests against current behaviour for both Sorceress profiles. | 7-9,12-15,30,34 | L | yes: Static Field opener on an elite, crowd cast, kite vs hold, on the live game (Drilly) |
| 5 | S3: Teleport capability derived from skills, wired in production, boss and farm read it (row 5); confirm first whether the production bot ever hops. | 5 | M | yes: one hop drill and an Act 2 stop |
| 6 | S6: parameterise drills and soaks by class and build; skill ids by name; sims take a profile (rows 31 to 33). Include a `--build` flag in `tools/live.sh` driver docs. | 31-33 | M | yes (re-run one existing drill per class) |
| 7 | **Proof 1: second Sorceress build end to end.** Run `sorceress_blizzard_endgame90` as a fresh character: offline simulation of the whole table, bind layout drill, Blizzard cast with delay, Frost Nova crowd, then a Normal Act 1 soak of the same shape as the lightning one (clvl 1 to 13, same gates). Report next to the lightning numbers. Fix what it finds (expect row 10 style gaps). | - | L | yes (soak, drills) |
| 8 | Third Sorceress build (fire or cold) as data only: a table plus a profile, zero Go changes is the acceptance test. | - | M | yes (one soak) |
| 9 | **Proof 2: a second class end to end**, using what exists as data (the Paladin hammer tables and profile are the nearest; the class-level smoke runs are in `docs/classes/smoke-*.md`). Rescope `docs/capabilities.md` per build. | 37 | L | yes (soak) |

Order rationale: phases 1 to 3 are offline and low risk, and remove the silent failures first; phase 4 is the largest change to live behaviour, so it is gated on golden equivalence and a live check; phase 7 is the first time a second build runs at all, so everything before it is preparation, and it should not wait on phases 8 and 9. Phases 1 to 3 can run in parallel cards (they own different files: `combat`, `leveler`, `bot`/`resource`/`town`); phase 4 follows phase 1.

## 6. Guardrail text for agent briefs (proposed, not applied)

For `AGENTS.md`, section "Rules for every agent". The coordinator or the user decides:

```
- Builds are data. Do not write a skill id, a skill name string, a character-level literal that belongs to one build, or a class check in Go. Read it from the plan table (internal/plandata), the combat profile (internal/combat/classdata) or the class traits table, or add it there.
- A new rule that only makes sense for one build or class (Static Field, Frost Nova, a kite range, a belt column) goes into that build's profile or table as a field with today's value as the default, not into a function body or a knob named after the skill.
- Never fall back silently to the Sorceress for an unknown class, table or profile: return an error that names the class or table.
- Write the test for a new combat or economy rule against two builds (the default Sorceress and one other class or build); a rule that only passes for the default Sorceress is not done.
- Say in the card result which part of the change is build-specific and where its data lives; docs/research/build-flexibility-audit.md lists the known leftovers, do not add to it.
- Drills, soaks and simulators take the class and build as parameters; resolve skill ids by name from the character, never by a numeric literal.
```

## 7. Shared-file lines for the merger

Not edited by this card; the coordinator applies them.

- **docs/BACKLOG.md** (ROADMAP): `multi-build flexibility (user direction 2026-11-01): remove the hard-coding to the one lightning Sorceress, phases 0 to 9 in docs/research/build-flexibility-audit.md (guardrails now, then profile id + table metadata + class traits + rules as data, then the Blizzard Sorceress as the proof, then a second class)`
- **docs/INDEX.md**: `- docs/research/build-flexibility-audit.md: what is hard-coded to the one Sorceress build (39 findings by layer and severity), what already works as data, seams, phased roadmap to several builds, guardrail text for briefs (BUILDFLEX)`
- **docs/decisions.md** (proposed one-line decision text): `2026-11-01: builds and classes are data (plan table + combat profile + class traits); no skill or build literals in Go for new work, unknown class or table is an error, never a Sorceress fallback; the second Sorceress build (Blizzard) is the proof, then a second class (user direction, docs/research/build-flexibility-audit.md)`
- **AGENTS.md**: the guardrail block of section 6 (proposal only).

## 8. Dead ends and limits of this audit

- Nothing was run; every claim is a read of code or docs. Row 5 (no production Teleport assignment) rests on a grep for `TeleportKey` over all non-test, non-rig code; a drill must confirm it before a card is written.
- Whether the END Sorceress table plays well was not judged (that is an SME and soak question). Only its plumbing was audited.
- Other classes' profiles are U (Kolbot ranges, not exercised live, per `classdata.go:17`); this audit treats them as data that exists, not data that works.
