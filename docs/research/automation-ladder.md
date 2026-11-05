# Automation ladder: d.idle as an incremental Diablo II (DRAFT, product direction 2026-11-05)

> status: DRAFT to refine, not a decision record. Written from the user's product discussion of 2026-11-05 (approved as a design draft; the tooling freeze allows design docs). Nothing here is built unless a row says so. Guard files (decisions.md, ADRs, GLOSSARY.md) are NOT edited; proposed text is in the last section.

Tags: **V** = read in this repo (file cited), **S** = veteran-player answer from the d2-sme agent (memory, nothing looked up; its numbers stay unverified), **U** = unverified or my judgment, **D** = the user's stated decision (2026-11-05 discussion). Classic 1.14d. D2R-only facts: none used.

## 1. The idea in one page

- **D** The game is an incremental Diablo II built on d2d (an open reimplementation, so custom features are possible). The player never controls the hero directly: no click-to-move, no casting. "Manual" means task orders ("go to the Blood Moor", "repeat Countess") and answering questions the hero asks.
- **D** Automation is the progression. Each capability is a tier. A tier unlocks by **learn by doing**: at a decision point the hero stops and asks; after N consistent answers it learns the rule and stops asking (revocable). Its quality is raised by **sacrificing** in-game things (gold, gems, runes) at an altar (a custom d2d object).
- **D** Loop: farm runes, sacrifice some, farm better. Unlocks survive a hardcore death (hardcore as rebirth: the character resets, learned automation stays).
- **D** Every run starts from a fresh clvl 1 character. No premade hero. Leveling to farming strength is part of the game.
- **D** End state: a fully automated farming bot driven by a goal ("farm X runes"). The player watches.
- **D** Failures show as **needs-input** (a question card; the hero waits safe in town) and **incident** (a plain-language journal report, rare at high tiers), plus a visible **uptime** stat.
- **D** Game shapes ranked: 4 (automation as progression) first, 1 (watch the hero) second, 2 (job runner) third. The abstract simulator (3) was not chosen: the real engine keeps running the game.
- V PLAN.md already parks an "idle layer" and treats hardcore as the rebirth mechanic ("Decisions": softcore first, hardcore later as prestige, keep character mode in `bot.Policy`); ADR 0015 and decisions.md "Current truth" list the idle layer as deferred. This draft is the design the parked layer would follow; it changes no current gate.
- V Existing hook for upgrades: `Tuning` knobs carry metadata and PLAN.md describes a data-only `goals.Upgrade{ID, Track, Tier, Cost, Requires, Effects}` contract so a UI can change freely (PLAN.md "Decisions"). Knob table: docs/knobs.md.

## 2. The tiers at a glance

| Tier | Name | The hero alone | Status of the code behind it |
|---|---|---|---|
| T0 | Order-taker | goes to a named area, fights, retreats at low life | mostly built, soaked on Act 1 only |
| T1 | Tidy | loots by rule, drinks and restocks potions, does town chores, returns to the field | built, proven live in drills; economy tuning open |
| T2 | Runner | repeats a run (same area or boss) until told to stop | partial: Runs goal and boss scripts exist, mostly U |
| T3 | Planner | picks areas by power vs danger, spends points, equips | partial: leveler and readiness rule built, policy still tuned |
| T4 | Questor | Act 1 quests, merc hire and care | partial: Act 1 quests and merc built; Andariel not cleared |
| T5 | Farmer | given a goal, picks runs by drop tables, stashes, sells, cubes | NOT STARTED as a goal-driven loop (pieces exist) |

The order is a design choice (U): each tier assumes the one below. The learn-by-doing unlock for tier K is only offered once tier K-1 is unlocked, so unlocks form a LINEAR ladder (see Risks: testing).

## 3. Tier by tier

Layout per tier: what the hero does alone; what the player orders; unlock rule; sacrifice axes; acceptance drill; mapping. "Decision points" are the places where a hand-playing veteran decides something (S = SME list, section 5).

### T0 Order-taker

- **Alone:** walk to the ordered area (waypoint, portal or exits), fight what it meets, drink at a life threshold, leave by portal or run at a lower one, never fight with an empty belt (S).
- **Orders:** "go to AREA", "fight here", "come home".
- **Unlock (the first tier starts the player's day one):** the hero asks at its first retreat decision ("life is 30 percent, a Fire Enchanted pack is near: drink, run, or portal?"). After N consistent answers it applies the same threshold itself. Asked points: when to drink, when to leave, run vs portal (S), two retreats in a row in one area (S), a pack type known to be lethal (S), an area far above the hero's level (S). N is a tunable, start N=3 (U).
- **Sacrifice axes (quality):** retreat precision (fewer needless retreats, fewer deaths), threat recognition (how many dangerous pack types it knows), reaction speed (U, may be cosmetic).
- **Acceptance:** drill: fresh clvl 1 character, order "Blood Moor", 10 virtual minutes, no death, retreat happens at the learned threshold, 0 writes after start (AGENTS.md crutch rule).
- **Mapping:** combat loop and retreat/get-hit decision (V docs/contracts/combat-loop.md; the get-hit rule matches game.exe, V docs/research/pr-get-hit-rule-proof.md in docs/research/README.md), resource policy for potions and mana (V internal/resource), navigation with stuck ladder (V internal/nav; failsafe ladder P25 in docs/primitives.md), potion drink P3 (V docs/primitives.md). Retreat knobs live in docs/knobs.md group `combat` and `resource` (V). Done: single-class (Sorceress) Act 1, soaked 2 h twice (V docs/capabilities.md). Missing: the "ask" mechanism itself and any learned-rule store (U: nothing like it in internal/; `internal/goals` holds goals, not learned rules).

### T1 Tidy

- **Alone:** loot by rule, belt and TP restock, heal, repair, identify, sell, stash when the pack is nearly full, take the portal back so the area stays open (S).
- **Orders:** "go to town and tidy", or implicitly after any run.
- **Unlock:** asked points: which item kinds to keep (borderline rares, charms, jewels, runes, gems), when to go to town (about 4 free rows, S), whether to spend scarce gold on restock vs repair, a lost TP or closed portal (S), a full stash (S). N=3 per item class (U).
- **Sacrifice axes:** loot-filter precision (fewer junk pickups, fewer missed uniques; start from the starter pickit), trip frequency tuning, town-visit speed.
- **Acceptance:** drill: 10 virtual minutes of fighting with a seeded drop set, afterwards belt filled, pack under X percent, junk sold, gold delta equals shop prices, cursor empty, in the field again. The Gate 0 primitive drills already assert most end states (V docs/primitives.md, P1 to P17).
- **Mapping:** `internal/town` (heal, shop, repair, identify, stash, room, trip; V package comment), `internal/loot` (pickit rules: `Starter()` and `ParsePickit`, V internal/loot/rules.go), `internal/d2dtown`, `internal/d2dloot` adapters. Proven live on d2d in drills, partial counts (V docs/capabilities.md row 1). Contracts: docs/contracts/town-basics.md. Open (V backlog/tasks): task-18 mana restock (In Progress), task-10 belt window fix, task-11 trip-loop progress fix. Missing: gamble is listed as a class only, P19 "not built" (V docs/primitives.md); no learned pickit.

### T2 Runner

- **Alone:** repeat the same run (route, kill, loot, town, new game so bosses respawn, S) until the order's count or a stop rule fires.
- **Orders:** "run Countess x20", "run Pindleskin until I say stop".
- **Unlock:** asked points: stop after a failure streak (about 3 deaths, S), what to do when run time or yield is worse than expected (S), when a boss is unsafe at current gear (S). N=2 per run kind (U).
- **Sacrifice axes:** run speed (route knowledge, less roaming), run safety (stop thresholds, retreat before the boss), reset speed (portal vs new game, U).
- **Acceptance:** drill: 5 runs of one kind with a fixed seed: each run ends in town, loot handled, per-run time recorded, no run starts with an empty belt.
- **Mapping:** `internal/goals` has a `Runs` goal type ("runs:NAME:N", `ParseRuns`, V internal/goals/runs.go) and `internal/boss` has act scripts plus farm loops and farm scripts (Pindleskin, Eldritch; V internal/boss/farm_scripts.go, which says "All of it is U: nothing here ran against the live game"). No Countess or Mephisto-run script is named in the farm scripts file (V grep found Pindleskin and Eldritch there); Mephisto and Countess belong to act scripts (U, not read in detail). Status: partial, offline only.

### T3 Planner

- **Alone:** choose the next area by power vs danger, spend stat and skill points per the plan data, equip by requirement and fit, move down when potion use is high (S).
- **Orders:** "level up", "go further", or a target level.
- **Unlock:** SME says this is almost always an ask (S): stat and skill allocation, respec, build-defining item choices, unique or set swaps. So here the unlock is per build axis: a first answer on "skill order" teaches the plan table to follow; N=1 for stats under a chosen build, N=3 for item swaps (U).
- **Sacrifice axes:** danger-estimate precision (readiness margins), gear-evaluation quality (how many affixes it understands), build breadth (how many builds it can follow).
- **Acceptance:** drill: from clvl 1, reach the Andariel gate at the readiness floor without a premature death; this is the current milestone-time metric (V decisions.md Current truth: milestone time per seed).
- **Mapping:** `internal/leveler` (stat and skill points, gear evaluation, equip; V package comment), `internal/plandata` (build tables, V), readiness rule power vs danger (V ADR 0011; thresholds in docs/contracts/early-game-policy.md and merc-policy.md), combat profiles `internal/combat/classdata` (V AGENTS.md "Build flexibility"). Partial: one Sorceress build is the proven one; Paladin offline only (V docs/capabilities.md).

### T4 Questor

- **Alone:** run Act 1 quests in order (Den of Evil, Blood Raven, Cain, Forgotten Tower, Tools of the Trade, Andariel; S), hire and keep a merc, enter the next act when ready.
- **Orders:** "do the quests", "next act".
- **Unlock:** asked points: which merc (permanent for that merc, S), hire cost, boss ahead of power (S), when to enter the next act (S). N=1 for merc type, 2 for act entry (U).
- **Sacrifice axes:** boss-fight quality (Andariel-style phase handling, poison/cold management), merc upkeep (revive and gear), quest-order efficiency.
- **Acceptance:** **Gate 2 and Gate 3 are the T4 acceptance** (this is the proposed link): Gate 2 is stability and economy over 8 h with the clvl 16 floor and the Andariel criterion at clvl 14+; Gate 3 is 24 h (V docs/contracts/soak-gates.md, "Gate 2 and Gate 3 meaning and floors"), and the project's "done" is a fresh uncheated 24 h Act 1 clear plus the A:B verdict (V decisions.md Current truth). Under this ladder that bar is a tier acceptance, not the product bar.
- **Mapping:** `internal/quest`, `internal/questmem` (quest state, V package comments), docs/contracts/act1-campaign.md, `internal/boss` (act1.go, act2..act5), `internal/merc` and `internal/d2dmerc`, `internal/task` (step graph), `internal/goals` Campaign goal (V internal/goals/campaign_test.go). Status (V docs/capabilities.md): quests beyond early Act 1 "Not yet verified"; merc hire/revive "Experimental"; Andariel never cleared. Open tasks: task-1 act1run, task-2 andariel, task-3 merc (V backlog/tasks, Deferred or Proposed).

### T5 Farmer (NOT STARTED as a loop)

- **Alone:** given a goal ("farm Hel", "farm 10 Ist"), choose the runs with the best chance per unit time from the drop tables and magic find, run them (T2), and between runs stash, sell, gamble and cube (S).
- **Orders:** one sentence, the goal.
- **Unlock:** asked points: goal change, a goal item is found (stop or continue?), cube vs keep, large sells, time or run limit hit (S). N=2 per decision kind (U).
- **Sacrifice axes:** run-choice quality (how well the drop tables are used), magic-find gear use, stash/sell/cube discipline, uptime.
- **Acceptance:** drill: 2 h virtual with goal "farm any 3 runes of rank r05 or better": the hero picks runs from the table, banks runes, cubes duplicates, reports a plain-language journal. Compare drops per hour with the model (U).
- **Mapping, what exists (V):**
  - Drop tables: `internal/data/loot.go` (treasure class struct) and `internal/data/supply.go` (per-kill supply for Normal Act 1 and 2, checked against game.exe's roller by 104,000 rolls; one player, MF 0). The same file notes the numbers describe the monster, not the character.
  - Cube: `internal/cubejobs` plans gem and rune upgrades that cost no gold, with a rune cap and keep-list (V internal/cubejobs; tests `TestRuneChain`, `TestRuneCapAndKeep`).
  - Inventory/stash/sell: `internal/town` and primitives P4 to P17 (V docs/primitives.md). Cube transmute P6 is offline only there.
  - Farm scripts: Pindleskin and Eldritch (V, U for live), `Runs` goal.
  - Fidelity pieces: game.exe differentials for XP and spawn density done (V docs/research/xp-award-trace.md, spawn-density-trace.md).
- **Missing (U unless noted):** drop tables for Nightmare and Hell and all acts (supply.go covers Normal Acts 1 and 2 only, V); magic-find and gold-find effect on the tables (MF 0 only, V); a run-choice planner mapping a goal item to the best runs; rune upgrade recipes beyond the cube planner (recipes list, rune cap policy); run scripts for Countess, Mephisto and the rest (Pindleskin and Eldritch exist, V); gamble (P19 not built, V); a stash policy for goal items; a journal. Also the altar object and the learned-rule store (all tiers).

## 4. Learn-by-doing and sacrifice: mechanics sketch (U)

- **Decision point:** a named event with a small answer set. The hero stops in a safe place (town if possible), shows a question card with the situation, and waits. Catalogue per tier above; a full catalogue is open question 4.
- **Learning:** a rule is learned when the last N answers at the same decision point, with the same relevant state bucket, were the same. Count-based, deterministic, stored per account (not per character) so it survives hardcore rebirth. Revoke: the player can unlearn a rule; the hero asks again.
- **Why count-based:** a replayable log of answers gives the same learned rules for the same seed (fits the lockstep/determinism lane, V docs/d2d-determinism.md exists; not read in detail).
- **Sacrifice:** the altar consumes items; each tier has quality levels 0..L, each level costs a rising amount of the sacrificed currency. Quality maps to knobs in the existing `Tuning` metadata (V docs/knobs.md section 1: each knob has min, max, "better" direction), so an upgrade is data: knob, value, cost. The bot is already built to run "optimal then degrade by profile" (V PLAN.md, knobs.md handicap profiles): the ladder runs that backwards, starting degraded and unlocking the optimum.
- **Uptime stat:** fraction of wall time the hero was acting rather than waiting for input or in an incident (U).

## 5. SME answer folded in (S; unverified)

The d2-sme agent (asked 2026-11-05, answering from memory, nothing looked up) gave: a task-level hero is sensible for T0 to T2 and T4; at T3 and T5 it should propose and wait because the choices are build-defining or irreversible. Decision points are already merged into the tier sections above. Its early-game farm target answer, SME (U):

| clvl | Area or target | Why (SME) |
|---|---|---|
| 1 to 10 | Blood Moor / Cold Plains, then Den of Evil | safe XP, first quest; not a rune source |
| 10 to 18 | Cave, Burial Grounds, Crypt, Tower | Countess (Forgotten Tower) is the first reliable rune source in Normal; which tiers: unsure |
| 18 to 25 | Andariel, then Act 2 | move on with the story |
| 25 to 35 | Normal Act 3 and 4, Mephisto, Countess repeats | Countess runs top out around Lem or Pul in Normal, Pul or Um rare (SME unsure) |
| 35+ | Nightmare Pindleskin, Act 5 | real rune farming usually starts in Nightmare or Hell |

The SME was not sure of exact rune tiers and thresholds, and said it has no drop probabilities memorised. Treat every number as U. It asked for research on the Countess rune table and per-difficulty rune cutoffs; this repo has a treasure trace (`docs/research/treasure-trace.md`, cited by V internal/data/supply.go) that can answer it from data, which is the better source (U: not read here). One SME detail worth checking: the Den of Evil reward as one skill point on Normal (the SME flagged it unsure; the repo's policy ADR 0004 "Den of Evil first, free skill point" says so, V docs/decisions.md index).

## 6. Re-ranking the backlog by tier (recommendations only; no card edits)

Open Backlog tasks (V backlog/tasks, status as of this branch):

| Task | Status | Tier served | Class |
|---|---|---|---|
| task-18 mana restock | In Progress | T1 (caster restock) | low-tier reliability |
| task-10 low-kit belt window | In Progress | T1 | low-tier reliability |
| task-11 trip-loop progress | In Progress | T1/T2 | low-tier reliability |
| task-12 ledger key per save | In Progress | measurement (all) | infra; keeps "uncheated" honest |
| task-9 m1-liveproof | In Progress | T0/T1 (Gate 0) | foundation; the drills ARE the T0/T1 acceptance |
| task-5 armorhit, task-6 nexthit, task-7 skmiss, task-8 skresistfix | In Progress | T0 fidelity (upstream PRs) | engine fidelity, not tier work |
| task-3 m3-merc | Proposed | T4 | mid-tier |
| task-2 m3-andariel, task-1 m3-act1run | Deferred | T4 / T3-T4 | mid-tier, the Gate 2/3 acceptance |
| task-16 m2-roam, task-17 m2-tripwp | Deferred | T0/T1 efficiency | low-tier polish (speed) |
| task-4 gameexe-retire-soak | Deferred | Gate A | fidelity acceptance |
| task-13 d2d container soak | Proposed | none (packaging) | unrelated to tiers |
| task-14 d2dmesh, task-15 stream-grid | Deferred | farm / watch view | product-adjacent; stream-grid serves the "watch the hero" shape (rank 2) |

Recent bot fixes (V git log since 2026-11-02): failsafe rung 3 cap and backoff, cursor clear before pickup and bounded retries, nav stuck ladder at low life near hostiles, portal/waypoint fallback in back-to-field, healing guard and shop fixes. All are **low-tier reliability** (T0/T1). They matter because every higher tier is built on a hero that does not loop, which is also why they stay valuable, but none of them moves a high tier.

Recommendations (U):
1. Keep T0/T1 reliability work only where it blocks a soak (it is the foundation), and stop adding new town/economy tuning cards.
2. The top-tier gap is unclaimed: no card covers T5 (goal-driven farm), the altar/learn-by-doing mechanism, or the question-card UI. If one card is added, it is a small spec for the decision-point store, not code.
3. T4 cards (Andariel, merc) stay the nearest high-value bot work and keep their Gate 2/3 role.
4. Upstream fidelity PRs are neutral to the ladder: they protect the engine all tiers run on.
5. Treat the `m2-roam` and `m2-tripwp` cards as T0 speed polish, below T4 work.

## 7. Open questions for the user

1. **Currency.** Runes are both the sacrifice currency and runeword material. Accept the tension (D says accepted), or add a separate currency, or let the altar take only duplicates and low runes? How does a player afford a runeword while also upgrading?
2. **The value of N.** One global N or per decision kind? Is "consistent" strict (N in a row) or a majority? Does a contradicting answer reset the count?
3. **Tiers vs acts.** Does T4 stop at Act 1, and which tiers cover Acts 2 to 5, Nightmare and Hell? Is there a T6 for later difficulties, or are later difficulties just harder goals for T5?
4. **Decision-point catalogue.** Section 3 lists candidates from the SME; which are the real set, and which are always-ask (build-defining, per the SME)?
5. **Gate 2/3 vs T4.** Is "Gate 2/3 = T4 acceptance" right, so the 24 h Act 1 clear stays the fidelity acceptance and not the product bar? (decisions.md currently calls it "done".)
6. **Watch view.** Is the embedded viewport (V internal/viewport, docs/d2d-view.md) a tier-independent feature available from T0, or itself something to unlock?
7. Does a hardcore death reset only the character (learned rules and quality levels survive), and does the altar quality survive too?
8. Mapping of "rank 2 job runner": does the order UI (task orders) already satisfy it, or is it separate?

## 8. Risks

- **The manual early game must be short and fun.** A fresh clvl 1 character with no direct control and only orders could be dull before T0 unlocks. Mitigation (U): the first minutes are decision-dense by design (the hero asks often), and the first tiers unlock after few answers.
- **Testing explosion** if unlocks are not a linear ladder: k independent capabilities give 2^k hero configurations. Keep it linear (each tier needs the one below) and run the gate ladder at every tier boundary. Hardcore rebirth adds a state (learned rules present, character fresh) to test.
- **Cores per hero.** The lab note in AGENTS.private.md budgets about one core per game instance; the user's rough figure is 2 to 3 cores per hero including d2d and the bot (U, not measured here). A farm of heroes is capped by that.
- **EULA and data.** The player supplies the D2 data files; no Blizzard files are shipped. V docs/contracts/licence-policy.md exists for this; not read in detail here. The d2d fork rules (private fork, GPL, no d2d code in this repo; V AGENTS.md) are unchanged. An open d2d with custom objects must stay offline and never touch Battle.net (V AGENTS.md "Facts and honesty").
- **Measurement.** Learned rules make runs depend on the answer log; a "writes after start" line and a fixed answer script are needed so soaks stay comparable (U).
- **Scope.** The current engineering is T3/T4; the product value sits in T0..T2 feel and T5. Chasing T5 before T4 passes would repeat the "seven streams" problem of docs/roadmap.md section 1.

## 9. Proposed record text (NOT written to the guard files)

**decisions.md line (proposed):** `2026-11-05 (user, no ADR yet): d.idle is an incremental Diablo II on d2d: no direct hero control; automation tiers T0 Order-taker to T5 Farmer unlock by learn by doing (N consistent answers, revocable) and are upgraded by sacrificing gold, gems and runes at an altar; every run starts from a fresh clvl 1 character; learned automation survives hardcore rebirth; end state a goal-driven farming bot. Draft: docs/research/automation-ladder.md. Gate 2/3 = T4 acceptance is proposed, not decided.`

**ADR outline (proposed):** title `00NN: Automation as progression: the tier ladder`. Context: the project is a faithful bot; the parked idle layer needs a design; shapes 1 to 4 compared (user ranked 4 > 1 > 2; 3 rejected). Decision: tiers T0 to T5; learn by doing; sacrifice upgrades; fresh clvl 1 start; unlocks survive hardcore rebirth; needs-input/incident/uptime. Consequences: Gate 2/3 become T4 acceptance; backlog re-ranked by tier; new work items (decision-point store, altar object, T5 planner). Open: currency, N, tiers per act.

**Glossary terms (proposed):** Tier (an automation capability level T0..T5); Decision point (a named event where the hero stops and asks); Learn by doing (unlock by N consistent answers); Altar (the custom d2d object where items are sacrificed); Sacrifice (spending items to raise a tier's quality); Needs-input (a question card, hero safe in town); Incident (a plain-language journal report of a failure); Uptime (share of time the hero acts rather than waits); Rebirth (a hardcore death that resets the character but keeps learned automation).
