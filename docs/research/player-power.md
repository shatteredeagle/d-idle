# Player power vs area danger: what we have, what the community does, a proposal

> status: research, proposal only (agent powerres, 2026-11-09). No code changed, no game run. Target classic 1.14d; D2R-only facts are flagged "D2R".
> Tags: V = read in this tree (path given) or a page I fetched (URL given); S = search snippet or secondary source; U = unverified or from memory.
> Author: Claude (d2lod-bot research agent).

## Direct answer

No. We have no single player-power number and no single area-danger number. We have a set of separate, partly overlapping gates, each with its own hard-coded inputs, and the one place that compares "the character" with "the area" (`resource.GearGate`) uses a point score (mlvl gap, kit, life, resist floor, deaths) that is not per threat. The risk meter on the watch screen (`internal/watch/unlockrisk.go`) is already a checklist rolled up to a traffic light, but its resist input is stubbed to 0 and no zone resist table exists. The community mostly uses checklists (resists, EHP, per-threat questions) and tiered "can I survive X" thresholds, not one score; the one-number tools (D4 item power, "gear score") are known to be poor predictors.

## Part 1. Repo audit

### 1.1 Is there a player-power number?

No single one. The nearest pieces (all V, read in this tree):

| Piece | File | What it is |
|---|---|---|
| clvl | `d2.Player.Level` | The only universal "power" input. Floors are clvl numbers in plan data (`docs/plan/data/progression-*.json`: `min_clvl`/`enter_clvl`, `leave_clvl`, `kit_clvl`; parsed in `internal/bot/plan.go` lines ~267-273) |
| Item scorer | `internal/gear/score.go` | Defence-equivalent score per item (ResistValue, FCR breakpoints, ethereal factor). Ranks items; it is not a character total and is not compared with an area |
| Endgame milestones | `internal/gear/milestones.json` | Per build: base item codes the build must be able to wear by a clvl (e.g. Enigma base at clvl 55). Data, per build |
| Gear gate points | `internal/resource/gear.go` `GearGate` | Risk points from the kit (below) |
| Risk meter facts | `internal/bot/riskfacts.go`, `internal/watch/unlockrisk.go` | Checklist (guard, kit, potions, life, level, resist, close calls) |
| Max life, resists | `d2.Player` | Read from memory; the "displayed, includes the difficulty penalty" claim is itself U (TODO in `internal/combat/hard.go`: if `Player.Resist` already holds the penalty, `PlayerResists.Effective` applies it twice) |

There is no EHP, no damage-per-second estimate and no "can this build kill X" estimate in the gating code. Damage enters only indirectly (a weapon is equipped; clvl).

### 1.2 Is there an area-danger number?

Three different ones, none unified:

1. Area monster level (V): `Runner.areaMLvl` (`internal/bot/geargate.go`) reads `levels.txt` MonLvl per difficulty from the extracted install data. `docs/difficulty.md` section 2.1 tabulates these (V tables).
2. Pack danger 0..3 (V): `combat.TargetDanger` / `PackDanger` (`internal/combat/affix.go`). Per live pack, from boss flag, champion/unique, affix bits (Extra Fast, Extra Strong, Teleport, Mana Burn, Cursed, Aura, Stone Skin, Fire Enchanted...), Hell minion counts, and a Hell-below-clvl-75 "deadly" rule (`avoidHellLevel = 75`, U: planning number). `combat.ResistDanger` adds +1 when an elemental enchant meets an effective resist below 0 (`minSafeResist = 0`). `CurseDanger` adds a curse bump. This is the only per-threat logic in the repo, and it is per fight, not per area.
3. Death-history weights (V): `Target.Danger` from `hc.Weights` (`internal/combat/target.go`, `death.go`, `avoidWeight = 3.0`) and the area ban after repeated deaths (`internal/bot/areaskip.go`, `deathlog.go`, knobs `bot.spiralDeaths`, `bot.dangerBanMin`).

The "danger rating" sense tier (6, `senses_danger`, price 80) is a ladder row only: `docs/senses-unlock.md` says tiers 5 to 7 have no data keys yet. So the viewer-side danger rating does not exist as a computed value (V doc).

No per-area table of immunities, dominant damage types or champion affix exposure exists. `docs/difficulty.md` section 3 is a prose/table list of immunities by difficulty (V tables), not wired. `RiskFacts.ResistGap` is always 0 (V `internal/bot/riskfacts.go` comment, `docs/unlock-risk.md`).

### 1.3 Where they are compared

| Decision | Where | Inputs compared | Verdict |
|---|---|---|---|
| Enter an area (planner) | `progression-*.json` via `internal/bot/plan.go` | clvl vs `min_clvl` / `kit_clvl`; kit (4 potions + TP, `PickOptions.FullKit`) | skip step or level up (V `docs/bossgate.md` WP3) |
| Boss quest go/no-go | `internal/task/gate.go` `bossGates` + `BossGate.NotReady` | clvl floor, healing potions, TP, mana potions (casters), antidotes, gold; Blood Raven class floors (9 caster / 8 melee, knobs `boss.brCasterLevel`/`brMeleeLevel`) | "not ready" -> chores trip or level step (V). Hard-coded table keyed by quest; only Burial, Tools, Tower, Andariel (V `internal/task/gate.go:160-168`) |
| Route guard | `internal/task/route.go` `TowerWhy`, `RouteWhy`; `internal/bot/routeguard.go` | clvl, merc alive, max life, kit; Normal only | blocks Monastery road / Tower with a reason string and a log line (V) |
| Graded gear risk | `internal/resource/gear.go` `GearGate`, `internal/bot/geargate.go` | see 1.4 | low / medium / high; HIGH gates areas to the first field (V) |
| In-fight abort/engage | `internal/resource/{preengage,abort,attack}.go` | life %, belt, merc, PackDanger (0..3) | proceed, rest, retreat, avoid, melee vs cast (V) |
| Chicken/drink | stance knobs (`combat.chickenPct`, `drinkLifePct`, `kiteLifePct`) | life % | per stance Cautious/Balanced/Aggressive (V `docs/stance.md`) |
| Viewer risk meter | `internal/watch/unlockrisk.go` | guard, kit, potions, life, level vs zone min + class lead, resist (stub), close calls | ready / marginal / turn_back + gauge. It reads the bot's own answers so it cannot contradict them (V `docs/unlock-risk.md`, test `TestRiskVerdictAgainstTheGuards`) |
| Whyline | `internal/bot/whyline.go`, `docs/whyline.md` | decision reason codes | one plain line; reason codes, not a readiness check (V) |

Retreat itself is not power-aware: it is life-percent plus pack danger plus a "monster too tough" path (V `docs/retreatfix.md`, `abort.go`); nothing says "this zone is above your kit, go home" except the route/gear gates evaluated at planning time.

### 1.4 What is hard-coded vs data

`GearGate` (V `internal/resource/gear.go`), points: no body armor +1+2d; no weapon +d (caster) or +HighAt (melee); difficulty +3d; act 2+ +1, act 4+ +2; max life under `lowLife = {40, 250, 500}` +2; displayed min resist < 0 in NM/Hell +d; area mlvl minus clvl >= 3 +1, >= 6 +3; no potions +2, none and no gold +HighAt; deaths in the hour and here. MediumAt 3, HighAt 6 are knobs (`resource.gearRiskMedium/High`); the point weights and `lowLife` are code constants. Mlvl gap is a flat clvl comparison with no allowance for resists, damage type or class.

| Area | Data (table / knob) | Code constant |
|---|---|---|
| Area floors per step | progression JSON (R/U tags), not per class | clvl floors are class-blind except Blood Raven |
| Boss kit | `bossGates` map, `boss.*` knobs for BR and Andariel | counts 4/2/3 hard-coded; only 4 quests |
| Gear risk weights | knobs for thresholds | all point weights, `lowLife`, `GearGateLevel = 5` |
| Pack danger | affix bits in code | affix-to-danger mapping, `deadlyLevel`, `deadlyMinions`, `minSafeResist` |
| Class weapon need | `neededWeapon` map (Amazon, Paladin, Barbarian, Assassin) | `Caster` flag is a class switch in `geargate.go` (Sorceress/Necromancer/Druid), against the AGENTS.md build-flexibility rule |
| Risk meter | `internal/watch/risk.json`, `decide.ProposeTable` classes | no zone resist table |
| Milestones | `internal/gear/milestones.json` per build | - |

Gaps (V by absence): no EHP, no damage check, no per-element resist requirement per difficulty/act, no per-threat result, no player-power value on the wire for the dashboard, and `ResistGap` stubbed.

## Part 2. Community practice

### 2.1 Classic D2

| Practice | What it uses | Form | Source |
|---|---|---|---|
| Difficulty resist penalty | Normal 0 / NM -40 / Hell -100, floor -100, cap 75 (95 with max-res items) | fact, drives a "cap all four before Hell" rule | V https://classic.battle.net/diablo2exp/basics/resistances.shtml (and our install tables, `docs/difficulty.md`). Resistances page gives no per-difficulty target, only "very important" in later acts |
| "Gear points needed" | 75 gear points in Normal, 115 in Nightmare to sit at the cap (75+40); 175 in Hell by the same arithmetic | arithmetic checklist | S (search snippet of a guide); arithmetic follows from the V penalties above |
| Hell advice | "entering Hell with -60% fire resist is a death sentence", cap resists first | checklist | S (search snippet, forum/guide) |
| Immunity planning | Hell monsters nearly all immune to something; plan a second damage type or merc | per-threat | V https://www.purediablo.com/?p=4278 (breaking immunities: only Conviction/Lower Resist, at 1/5 strength; 1.14-era text, applies); Maxroll https://maxroll.gg/d2/resources/immunities is D2R (Sunder Charms, patch 2.5) and does NOT apply to 1.14d |
| Boss-specific prep | per-boss notes (Baal Hell 50% all resists, mlvl 99; Uber Baal cold 185, fire 115) | per-threat | S (Gamer Guides, RankedBoost snippets); our own V values: `docs/difficulty.md` boss resist table from install tables |
| MF / clvl vs area | area monster level from levels.txt; Arreat Summit "stay in Act V Normal until about level 40" | clvl vs mlvl rule of thumb | V `docs/difficulty.md` section 2 (Arreat difficulty page, fetched earlier); no community "formula" beyond rules of thumb found. Our rule (mlvl - 2/-5/-8) is itself R/U |
| Arreat Summit / planners | monster tables, resist and affix tables, skill and item data | reference, no readiness score | V (earlier research, docs/difficulty.md); none computes "ready" |
| Calculators | Amazon Basin / Maxroll builds planners show defence and resist totals; damage calculators (e.g. "D2 Damage Calculator") per skill | stat sheet | U from memory; no EHP-vs-area tool found |
| Hardcore community | survival rules: full resists, life > 1500 for Act 5 Hell (and "block/FHR breakpoints"), merc, no one-shot (Hammerdin/Enigma recommended gear) | informal checklist | U from memory; not fetched |

### 2.2 D2 bots (readiness inside a bot)

- Kolbot-SoloPlay (the closest prior art): V https://bhdocs.github.io/SoloPlay/faq: it moves to the next difficulty only at a minimum character level (`CharInfo.levelCap` in the per-class build file `libs\SoloPlay\BuildFiles\<class>\<class>.js`), "will not start the next difficulty with negative resistances", and overrides that check if more than 5 levels above the minimum and still lacking resists. Gear requirements apply only to switching to the final build. So: per-class table (data), two inputs (clvl, resists >= 0), one soft override. Exact resist thresholds and where the check lives were not read (code not fetched; GitHub raw 404): U.
- Kolbot/D2BS configs: search found no resist/level entry gate in the stock configs (only a party `CharacterDifference` filter): S.
- Interpretation: bot authors settled on clvl floor + non-negative resists + "good enough" override, per class, as a data file. That is close to our `progression` + `GearGate` and does not model damage or threats.

### 2.3 Other ARPGs

| Game | Inputs | Form | Communicated how | Source |
|---|---|---|---|---|
| Path of Exile | layered defences: avoidance, mitigation, recovery, life pool; ele resist 75 "minimum", chaos ~40; spell suppression; ailment/stun/curse protection; PoB EHP is "not fully representative"; 16 threat types each with a way to cover it, no pass/fail numbers | per-threat checklist, explicitly no single score | guide prose | V https://www.pathofexile.com/forum/view-thread/3275670 |
| PoB boss presets | Configuration tab "Is the enemy a Boss?" (Guardian/Pinnacle, Uber presets), EHP/one-shot vs average EHP; DPS targets for Ubers (~3M uber / ~10M pinnacle by a seller blog) | calculator inputs + player judgement | numbers in PoB | S (search snippet; the DPS numbers are from a boost-seller site, low trust) |
| Diablo 4 | item power, level, Torment tier; Maxroll: play "the highest difficulty your build can handle efficiently" and does not define it; Torment drops Ancestral/Mythic | gate on level + player judgement; item power is known to be a weak predictor | V https://maxroll.gg/d4/getting-started/powering-up-in-diablo-4 ; S community analysis (PC Gamer): NMD tiers beyond ~40 mostly not worth it |
| Diablo 3 | no gear score; player finds a "realistic cap" GR: reference A-tier build comfortable at ~GR125, 10-16 tier RNG range, go 2-3 tiers below the cap for safety; monster HP x1.17 per tier; toughness/resist cited qualitatively | trial and error with benchmarks | guide prose, no numeric defensive thresholds | V https://maxroll.gg/d3/resources/greater-rifts |
| Last Epoch | monolith recommended levels per timeline (58 to 90), corruption cap 50 and empowered = level 100; no gear check | recommended level on the content | in-game recommended level | S (search snippets, Maxroll/guide sites) |
| Warframe | armor curve armor/(armor+300), Steel Path armor multiplier (2.5x to 3.5x, sources conflict), partial armor strip; calculators show EHP incl. shield gate | calculator output | EHP number | S (west-games calculator, Warframe forums); multiplier U |
| Grim Dawn, Torchlight, PoE 2 | not researched beyond a PoE2 defensive-layers snippet (same layered idea) | - | - | U |

### 2.4 What the sources share

- Inputs used: character level against content level, the four elemental resists against a cap/floor (with difficulty penalty), life/EHP, kit (potions, escape), class-specific damage-type coverage for immunities, damage output only for bosses (PoB DPS targets, D3 clear times).
- Single score vs checklist: serious defence guidance (PoE forum guide, Kolbot-SoloPlay) is a checklist; single numbers (D4 item power, "gear score") are communicated but not trusted. D3 and D4 push readiness to player judgement with benchmarks and a "go 2-3 tiers below your cap" margin.
- Communication: recommended level on content (Last Epoch), per-threat questions (PoE's 16 types), build-file thresholds in a bot, EHP number in calculators. Traffic lights are rare in these games; our meter's ready/marginal/turn_back is our own idea.
- Not found: any tool that scores a D2 character against the specific area (immunities + enchants + mlvl) automatically; the D2 planners only show stat totals (S/U).

## Part 3. Recommendation for d.idle (proposed, user decides)

### 3.1 Shape: a readiness check, not a number

A `Readiness` check set, evaluated for (hero, target area or boss, difficulty). Each check returns `pass | warn | fail`, a plain reason string with the numbers ("fire res -25 < 0: Fire Enchanted packs here can one-shot"), and the knob/table that decides it. The roll-up is a traffic light with the same semantics as the risk meter (any fail = red/turn_back, any warn = amber/marginal, else green/ready). No composite score is shown or stored; the roll-up lists which checks caused it. This is what the risk meter already does for the watch view; the proposal is to generalise its checks and make the bot's gates consume the same set so they cannot disagree (the existing `TestRiskVerdictAgainstTheGuards` invariant).

### 3.2 Per-threat checks (data per class and difficulty)

Candidate checks, each a row in a data table with per-class and per-difficulty fields (no class switch in Go; unknown class or table is an error naming it, per AGENTS.md):

1. Level vs area: clvl against mlvl (or the plan floor), with margin per difficulty and class (today's clvl-vs-mlvl +3/+6 points become a table).
2. Resist floor: each of fire, cold, lightning, poison, effective after penalty, against a floor per difficulty/act (green >= cap target, amber 0..target, red < 0). Needs the V decision on whether `Player.Resist` already includes the penalty.
3. Enchant exposure: "survive a Fire/Cold/Lightning Enchanted, Spectral Hit, Mana Burn, Extra Strong pack" questions answered from resists + max life + potions (extends `ResistDanger`; rows map to the affix bits).
4. Life pool: max life against a per-area burst reference (a champion/unique hit estimate from monster damage rows and the V Arreat multipliers in `docs/difficulty.md`) rather than the flat `lowLife` table.
5. Damage coverage: for the area's immunities (per-area table from `monstats` + `levels`), can the build's skills (or merc) hurt each? Red when a likely pack type is unkillable, amber when it needs the fallback. Data: area immunity table plus the combat profile's damage types.
6. Kit: potions, belt plan, TP, mana potions, gold (existing `brShort`, `routeKit` etc. folded into one check).
7. Escape and recovery: portal scroll/tome, merc alive, town distance.
8. Boss gate row: per boss, the above plus boss-specific rows (poison for Andariel, cold for Duriel, etc.) from a table, replacing the four-quest `bossGates` map.
9. Recent history: deaths here, close calls (existing).

### 3.3 Feeding existing surfaces

- Risk panel: the meter becomes a view of the check set; each check is a row with pass/warn/fail and reason; `ResistGap` is replaced by the resist check.
- Whyline: new reason codes for "readiness red/amber because <check>" (a template per check), shown when the planner holds back or retreats.
- Retreat/planner: `areaRulesWhy` and `BossGate.NotReady` call the same evaluator; HIGH risk (red) keeps today's behaviour so no gate gets looser; amber stays a log line.
- Boss gate: boss rows drive the "not ready" reason.
- Policy sheet: a "readiness strictness" row (Strict / Balanced / Bold) that shifts margins per class, like the stance row, allowing player-chosen suboptimal choices (V policy-sheet principle).
- Danger rating sense tier (6): show per-area check counts only after purchase, derived from the same tables, so the tier is no longer an empty row.

### 3.4 Proposed cards (not created)

| Card | Kind | Acceptance criteria |
|---|---|---|
| ready-audit | research/probe | Live read of `Player.Resist` on a Nightmare and a Hell character (zero setup writes) settles whether it includes the penalty; result in `docs/probes/`; resolves the TODO in `internal/combat/hard.go` |
| ready-area-table | feature (data) | Generated per-area table (mlvl, immunities, common elite affixes) from install data for all difficulties; validated against `docs/difficulty.md` section 2.1 and 3; tests for two classes |
| ready-model | feature | `internal/readiness` evaluator, checks above as data rows per class and difficulty; unknown class/table is an error naming it; table-driven tests for Sorceress and Paladin; no new class literals in Go |
| ready-wire-gates | bot-fix | `areaRulesWhy`, `BossGate`, route guards call the evaluator; existing gate tests unchanged; new test: no verdict looser than today for the same inputs |
| ready-meter | watch | Risk meter rows are the check set; `ResistGap` live; golden JSON for two classes |
| ready-whyline | watch | Reason codes and templates (2 phrasings each) for red/amber checks; held-back planner steps show the reason |
| ready-bossrows | feature (data) | Boss table for all act bosses with per-boss rows (resists, immunities, kit); Normal/NM/Hell values from install data; `bossGates` map retired |
| ready-policy | feature | "Readiness strictness" policy row with per-class knob values; default changes nothing |
| ready-calibrate | drill | Replay deaths from `docs/deathautopsy.md` / death logs: the evaluator would have flagged red/amber before at least N of M recorded deaths (N, M set when the card is groomed); false-red count on survived sessions reported |

### 3.5 Risks and open questions

- Thresholds will start as U numbers; calibrate against recorded deaths (ready-calibrate) before they gate anything.
- Damage coverage and burst estimates need monster damage data not yet loaded by `internal/data` (U: check what `monstats` columns are extracted).
- Whether `Player.Resist` includes the difficulty penalty (above) changes check 2 and 3 by 40/100 points.
- A single "power" number for the dashboard is not recommended; if the user wants one for the viewer, derive it from the roll-up (share of checks passing) as the meter does today, never as an input to a gate.
- Kolbot-SoloPlay's resist logic and the Warframe/Grim Dawn/Torchlight material are unread; extend only if the user wants more prior art.

## Sources fetched (V)

- https://classic.battle.net/diablo2exp/basics/resistances.shtml
- https://www.purediablo.com/?p=4278
- https://bhdocs.github.io/SoloPlay/faq
- https://www.pathofexile.com/forum/view-thread/3275670
- https://maxroll.gg/d4/getting-started/powering-up-in-diablo-4
- https://maxroll.gg/d3/resources/greater-rifts
- https://maxroll.gg/d2/resources/immunities (D2R page, used only to flag the difference)
