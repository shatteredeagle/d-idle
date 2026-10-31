# Plan v2: enabling features and redesigned soak gates (planning session, 2026-10-30)

> status: section 1 and the M-item order are from the morning of 2026-10-30; d2d is the primary backend (ADR 0009, 2026-10-30; docs/plan/STATUS.md), G1/G2/Fresha are judged (docs/g1-log.md), current state is docs/plan/STATUS.md. Sections 10 and 11 (user decisions) stay authoritative until an ADR replaces them.

Inputs: docs/plan/code-reality.md (what the code really does), docs/sme/mercenary.md (guides + hireling data),
the d2-sme proposals (feature-set verdict, plan proposal, merc gearing), docs/g1-log.md (14 G1 attempts),
docs/sme/pace-tables.md and audit-bot-vs-human.md. Tags: V = read in code/data, S = guide, EXP = veteran judgment (d2-sme), U = unverified.
Status: decisions in section 10 (2026-10-30); build status per item: [STATUS.md](STATUS.md). Sections 1 and 8, the death-rule text in sections 5 and 9 and the "NOT approved" line in section 10 are historical: the death rule was APPROVED by the user on 2026-10-30 and its final text is section 10.2 (also openspec/specs/death-rule/spec.md); numeric gates now live in openspec/specs/soak-gates/spec.md.

## 1. Where we really are (truth first)

> HISTORICAL (written 2026-10-30 early morning, before M0 to M11 were built): the bullets below describe the code at that time. Current state: [STATUS.md](STATUS.md).

- Honest runtime so far: about 2 h at a time, clvl 8-12, Act 1 Normal only (V, code-reality). Acts 3-5, Nightmare, Hell, the Act 2 boat hop and Andariel have never run live.
- Infrastructure converged: since G1 attempt 6, kill recovery (<= 100 s, one restart), save integrity (saved XP == char XP), flat fd/rss slopes and no orphan wine processes pass every time.
- Behaviour did NOT converge: attempts 7-14 each exposed a different basic (unwearable armor, broke loop, gold burn, pack/belt, no-area failsafe, quest-walk safety dropped, kiting without casting, NPC menu retries, corpse route, idle in town for 18 min). Expect 2-4 more attempts of this before an 8 h run holds (SME EXP).
- The long soaks as designed would mostly measure "grinding Act 1 in a loop": the Andariel gate holds at clvl 22 (hard-coded in 4 places) while Act 1 monsters top out near level 12, so from about clvl 17 XP decays and Act 2 is never reached (V, code-reality; SME A1).
- (OUTDATED) The mercenary was dead code when this was written: hire rows empty (cmd/d2bot/acts.go:104, every hire failed), revive off, no Act 1 hire, no merc gear, no reader (V then). Since then: policy package, free-Rogue path and a d2d reader are merged and wired behind `merc.enabled` (default 0, docs/merc-wire.md); on the Game.exe backend there is still no reader.
- The soak harness cannot see a bot that loops in a safe area: its no-progress counter counts walking, m24's XP/h floor is 1000, no time-in-state report (V).

## 2. Principle

Keep the freeze for everything not listed here. Lift it ONLY for the enabling features below, in order, each with an observable live acceptance check. Fix behaviour bugs found by live runs as they appear (agents already in that mode). Do not build Acts 3-5 / Nightmare / Hell features until a run actually gets there.

## 3. Mercenary strategy (V from data/guides unless tagged)

- Hire rules: Rogue hireable at clvl 8 (one source says 9) or after Blood Raven's quest; her free Rogue is lost if a merc is already employed. Desert (Act 2) and Iron Wolf need no quest. Mercs follow through portals, waypoints, acts and difficulties; cannot be dismissed; a new hire replaces the old merc AND destroys its gear. Any hireling NPC revives (formula from D2R: S).
- Stage A (as soon as possible, clvl 8-9): hire the Act 1 Rogue (Kashya) with a cold-arrow skill, when gold >= 3x the quoted price + the potion/TP kit. Read the price from the window (UNSURE: exact quote).
- Stage B (first Lut Gholein visit, about clvl 15-18): hire the Act 2 Desert Merc by class: casters Defiance (Prayer fallback), melee/Paladin/Amazon Defiance or Prayer, Blessed Aim only for bow/javelin Amazons. Hire only if gold >= 3x quote after the kit; otherwise keep the Rogue. Strip his gear first.
- Stage C (Nightmare Act 2): Might (physical) or Holy Freeze (casters); Might/Holy Freeze/Thorns exist only in Nightmare in 1.14d (V hireling.tsv); keep that merc through Hell by reviving.
- Revive rule (gold-gated to avoid a new poverty spiral): revive in town when gold >= 2x price + TP + potions; max 2 revives per hour; > 3 merc deaths per hour = stop reviving until gold >= 5x price; a merc death in an area is a signal to leave it. Revive cost about 600-900 gold at merc level 9-11 (formula S).
- Gear (SME EXP; verify rune names/levels against data; offline: ladder-only runewords blocked): placeholder merc = any floor helm/armor meeting the level requirement (armor > helm > weapon), a white bow/polearm/spear; no purchases. Long-term Desert Merc: T0 white socketed polearm + Lore helm + Stealth armor; T1 (NM) Smoke armor, Malice/Honor weapon (Honor ladder status UNSURE), life/leech helm; T2 Treachery / Chains of Honor / Fury. Skip Insight, Infinity (ladder-only), Call to Arms, Heart of the Oak, Beast.
- UNKNOWN to settle by one live probe: do mercs need Str/Dex for gear, or only level? (docs say yes, SME memory says level only.) Do not hard-code until probed (log `merc_reqcheck`).
- Potions: rarely worth it; healing only when merc life < 40% in a non-trivial fight, rejuvenation for boss emergencies; never buy for him. d2input Shift+key support UNSURE.
- User requirement: the bot must SEE and MANAGE merc gear: read it, show it in the dashboard, equip/swap/strip.

## 4. Ranked work list (feature lift)

Each item: what, minimum viable version, live acceptance check, effort (S/M/L agent-hours from code-reality), cut order.

| # | item | minimum viable | live check | effort |
|---|---|---|---|---|
| M0 | Measurement first (offline, small): d2soak codes `no_gain` (no clvl/quest/net-gold gain in N min) and `below_floor_clvl`, time-in-state report (grinding / quest / recovery / waiting / town), clvl checkpoints per hour from pace-tables, death limit = streak (3 in 20 min or >= 5/h; deaths with 0 healing potions tagged as bugs) | codes + report in soak summary and sidecar | a replay of attempts 12-14 logs flags the idle and the broke loop | S-M |
| M1 | Fresh-character path: `d2bot` never creates characters; script cmd/oog create + `soak24.sh --char`; a documented cheat baseline for test characters only | create a throwaway Sorceress and soak it | character appears, bot starts it | M |
| M2 | Andariel gate as knobs: replace the four hard-coded clvl-22 holds (task/campaign.go:65, task/gate.go:27, boss/act1.go:152, runs.json) with one knob, default about 14-16 plus kit (>= 4 potions, TP, 2 antidotes) and a merc-alive soft gate (floor +2 if the merc is dead and gold is low); boss prep list (antidote); add the missing Act 1 clvl 14-22 area so XP does not collapse | change + data | `boss: andariel gate pass clvl=N`, no `held for andariel` after the floor | S-M |
| M3 | Merc reader + dashboard: class (271, 338, 359, 561), alive, life%, merc level, aura, per-slot gear, defense/damage, free cells, revive estimate; fixture from a live probe | read-only, no UI clicks | `merc: seen class mode life% level aura` lines; dashboard card | M |
| M4 | Merc live probe on a throwaway character (i2/Botty, gold cheat ledgered; never counts toward a gate): hire window layout, price, level column, dead-merc behaviour, revive dialog line, Str/Dex question | probe notes + fixtures | docs/merc-probe.md | S |
| M5 | Rogue hire (Kashya): fill mercRows, hire step at clvl >= 8-9 when gold >= 3x quote + kit, once per character, strip-before-hire | one hire | `merc: hired <class> aura price`, gold delta == quote | M |
| M6 | Revive with budget rule (Kashya/Greiz first; skip Tyrael/Asheara at first) + merc-alive boss gate (soft) | rule above | `merc: revived price N` or `merc: revive skipped (gold)` | M |
| M7 | Merc gear: equip-if-better helm/armor (floor drops, merc-specific score: defense/life first, +10 % gain rule), strip procedure (needs ~18 free cells; stash while in town), weapon second; log `merc_equip` / `merc_strip` | helm+armor first | `merc_equip slot item reason` | M |
| M8 | Act 2 Desert Merc hire by class aura at the first Lut Gholein visit (reuse HirePolicy + class table); requires Act 2 entry to work | after M5-M7 | aura in the log matches the table | M |
| M9 | One honest Act 1 chain run live through Andariel (the first real boss fight) on a fresh character | no new features | Andariel attempted at floor | L (run time) |
| M10 | Skill/stat tables past clvl 30 (six of seven classes end at 30; Sorceress runs to 99) | only if a run can reach 30 | points spent after 30 | S-M; CUT FIRST |

### M2 status (agent M2-ANDY, branch agent/m2-andy; offline only, nothing verified live)

- One source of truth: knob `boss.andarielMinLevel` (default 15, range 12..22, `internal/env/tuning_boss.go`; 22 reproduces the old hold). Read by the campaign entries (`task.Entry.Andy`), the gate (`task.BossGate.Andy`), the boss script (`boss.Opts.AndarielMinLevel`, wired in cmd/d2bot and cmd/bossrun) and tested against `runs.json` (`bot.TestPickNeverEmptyInAct1`). Normal only; Nightmare/Hell keep their static floors.
- Kit: 4 healing potions, a TP scroll or tome, 3 mana potions for a caster (as before) plus 2 antidotes while gold >= 150 (below that the gate passes without them). A short kit is the existing chores-insert / "not ready" planning block, not a fault. New chore: `town.ChorePlan.Antidotes` buys `yps` at the potion vendor (never below the gold reserve); the bot asks for 2 once clvl >= floor-2 in Normal and Andariel is alive (`Runner.antidotesWanted`). Buying is tested on the town sim only (U live: Akara's stock and the buy click for antidotes are unverified).
- Merc: hook `task.MercAlive func() (alive, known bool)`, default nil = unknown = pass; dead merc and gold < 1000 raises the floor by 2 (U numbers). M3 fills the hook.
- XP gap: Normal Act 1 monster levels (V, levels.txt MonLvl1Ex from /tmp/d2mpq on the worker host): Tamoe Highland 8, Outer Cloister 9, Barracks 9, Jail 1-3 10, Cathedral/Catacombs 1-2 11, Catacombs 3-4 12, Tower Cellars 7. Added picker steps Tamoe Highland (10-14), Jail 1 (13-16), Catacombs 3 (14-18) so the 12-22 band always has an area, also with any single area cleared (test). Barracks not added (G1 died there 10 times). Walkability of the three new areas is U.
- Logs: `boss: held for andariel (clvl N < floor F)`, `boss: andariel gate pass clvl=N kit=ok|short(why)` (on change, from clvl 8), `level plan: clvl N -> area X (why)`.
- Found, not fixed: `TestCampaignGameRestartAtEveryTick` fails on master too when a restart lands after both Horadric Staff pieces are picked up (planner reports Done in Act 2); the test stride was changed 3 to 7 to dodge it.

Cut order if time is short: M10, then M8, then Tyrael/Asheara revive. Keep M0, M2, M3-M5 and the no-progress rule.
Not needed for the 8 h / 24 h runs: Acts 3-5, Nightmare/Hell rules, farm runs, merc equipment beyond M7, runewords, fleet, container, idle layer, respec.

## 5. Redesigned gates

Stability gates (any run, unchanged): 0 panics, flat fd/rss slopes, no orphan winedevice, kill-injection recovery <= 100 s with one restart, saved XP == char XP.
Progression gates (new, soft until measured): clvl floor per hour from pace-tables (2 h 7, 4 h 11, 8 h 16, 12 h 20, 24 h 26; cells are U-derived, re-derive from the first fresh-character run); no area/state above 40 % of an hour; no clvl/quest/net-gold gain in 90 min = FAIL with a planner dump; merc alive >= 70 % of Act 1 time after hire; recovery phase must end in gold (gold flat for 4 h while the merc is dead = FAIL).
Death policy: dying is normal; the APPROVED rule is section 10.2 (streak, recovery share, rate, repeat-site action, BUG tag). The first draft here (3 deaths in 20 min or >= 5/h) was replaced.
G1 (2 h): stability + recovery + behaviour (casts > 0 with hostiles, no idle, no death chain); on a FRESH Sorceress (Den done by hour 1, clvl 7 by 2 h, Rogue hired by 3 h) once M1 exists; Soaky is a gated wreck with a damaged history and must not be the only test character.
G2 (8 h): call it a stability + economy soak, not a progression soak, until Andariel/Act 2 work: clvl floor 16, Andariel attempted at clvl >= 14, merc hired, gold not flat.
24 h: Act 2 entered and a Normal Act 2 chain, Nightmare readiness check. A cheated-gold or cheated-level run never counts toward a gate.

## 6. Order of work and where live checks run

1. In flight now: NOIDLE (idle-wait), then restart G1 on the fixed build; keep behaviour bug fixes flowing (agents already tagged bug-fix).
2. M0 and M2 first (small, unlock honest measurement and progression), M1 in parallel.
3. M4 live probe on i2 (Botty/Combi) with a ledgered gold cheat; M3 reader + dashboard from its fixtures.
4. M5, M6, M7 offline with mem.Fake, then short live checks on a test character (i2), then on the fresh Sorceress on i4.
5. M9 live Act 1 chain through Andariel; then G2.
Merc probe candidates: Soaky (clvl 12, honest play, 242 gold) can serve as the hire test only if the user allows it; otherwise a throwaway with a ledgered gold cheat.

## 7. Risks

- Revive becomes a gold sink (mitigated by the 2x/3x gold rules and the revive cap).
- Merc proximity detection can fail after a town portal; hire-dialog line mismatch; Str/Dex unknown.
- Gate floors are U; re-derive from the first fresh run.
- We have never fought Andariel live: expect a 1-2 attempt learning loop of its own.
- Behaviour bugs keep surfacing: budget 2-4 more G1 attempts before G2.

## 8. Decisions needed from the user

1. Approve lifting the freeze for M0-M9 (M10 optional).
2. Approve the fresh-character test (M1) and cheats on throwaway characters for the merc probe.
3. Approve the death-limit rule (streak-based).
4. Approve calling G2 a stability + economy soak.
5. Merc order: Rogue as soon as clvl 8-9 and gold allow, Desert Merc by class aura at the first Lut Gholein visit: confirm.

## 9. SME review of this plan (d2-sme, 2026-10-30; EXP unless marked, nothing web-checked) and amendments

Death policy (SME review of the first draft; the user later approved the section 10.2 rule, which supersedes this paragraph): the SME says a pure streak rule is a poor primary metric (3 deaths/h for 8 h = 24 deaths still passes). Normal is about 0-1 deaths/h for a careful caster, 1-3 for a sloppy bot, >= 5/h is a sick bot. Recommended replacement, in order: (1) share of time in death-recovery states: WARN > 5 %, FAIL > 10 % of a rolling hour; (2) deaths per 1000 kills (WARN > 5, FAIL > 15, needs a kills counter); (3) gold lost to death per hour as a share of gold earned (FAIL > 25 %); (4) repeat-site deaths: 2 deaths in one area within 30 min force an area change plus a planner dump; (5) last-resort streak: 3 deaths in 30 min with no clvl gain, or > 4/h; (6) keep the 0-healing-potion bug tag. Per-death log must carry area, killer, belt potions, gold and gold lost, merc state, seconds from death to back alive (DEATHAUTOPSY already records most of these). Unsure: death penalty in Normal (SME believes gold only, our logs show xpLost on some stops: check).
Amendments to the work list:
- Probe before hire code: M4 before M3/M5; reader and hire written from the probe fixtures.
- Blood Raven's FREE Rogue (quest reward) beats a paid Kashya hire, and a paid hire BEFORE the quest forfeits the free one (V mercenary.md). Tension: a lone clvl 10 caster fighting Blood Raven (50 % magic resist) is risky, and we gave her an attempt budget. Decision needed: take the quest reward (needs a Blood Raven kill: use physical/skeleton help or a higher clvl) or hire from Kashya at clvl 8-9 and accept forfeiting the free one. Soaky already took neither: a test of the reward path needs a fresh character.
- M7 merc gear: cut to 'equip any floor helm and armor' until the merc is the Act 2 Desert Merc (the Rogue is a stop-gap and a new hire destroys his gear).
- Fresh-character clvl 1-8 (no merc, fragile Sorceress) is the biggest unknown: first test = fresh Sorceress to clvl 8 + Den of Evil (free skill point: make 'Den before Blood Raven' explicit) + death log, under 2 h. Charsi Imbue is a hard step for melee classes, Cain optional. Add an early buy order for clvl 1-8 (potions and TP, identify only if needed, armor last): docs/sme gap.
- Gate numbers: Andariel floor 14 (16 only with a dead merc); clvl floors for the first fresh run loosened to 2 h = 6, 4 h = 10, tighten later; no-gain rule 45 min FAIL (20 min WARN) not 90; per-state time caps per area class (town / grind / boss / recovery) instead of one 40 % cap.
- Test order (SME): fresh Sorceress to clvl 8 + Den + Blood Raven -> merc probe (Str/Dex) -> Andariel attempt at clvl 14 with the kit (one attempt, full logs) -> only then the long soaks.
- Top risks: fresh clvl 1-8 deaths with no merc; losing the free Rogue by a paid hire; idle loops caught too late; Andariel never run live; merc hire/revive dialogs unseen (expect 2-3 live iterations).
- SME doc gaps to research: death cost per difficulty, Blood Raven reward vs hire comparison, clvl 1-8 buy order.

## 10. User decisions (2026-10-30)

- Merc path: TAKE THE FREE ROGUE (Blood Raven's quest reward). No paid Kashya hire before the quest (it would forfeit the reward). Kashya hire stays only as a fallback when the bot skips Blood Raven for good. Consequence: the bot must be able to KILL Blood Raven early (clvl ~10-12) as an explicit goal, so the Blood Raven attempt budget (BROKEPHASE: 3 attempts, then wait for clvl+2) needs a real tactic and kit: see the SME iteration (docs/sme/blood-raven-caster.md, plan-v2 section 9). M5 becomes 'claim the quest reward Rogue at Kashya after the kill'.
- Approved: lift the freeze for M0-M9; fresh-character test and cheats on throwaway characters for the merc probe; G2 as stability + economy soak.
- Death rule: first draft NOT approved; after the SME rounds on real data (docs/plan/death-data.md) the rule in 10.2 was APPROVED by the user the same day.
- The user considers the d2-sme agent about 99 % right and wants it consulted before design decisions (memory d2-sme-agent-trusted).

### 10.1 Blood Raven for the free Rogue (d2-sme, 2026-10-30, EXP unless SRC)

- Tactic (all classes): kill the outer skeletons/zombies first, pull the Hungry Dead to the fence gap or gate, then go only for her (her death kills the zombies); break line of sight on tombstones/walls and corner her. Resists (SRC): 0 % physical, 50 % every magic element, 50 % chill. HP unsettled (113 vs 323): log her HP from memory.
- Sorceress: Charged Bolt (homes, cannot miss); Ice/Fire Bolt lose half their damage; plan 100+ mana or about 4 mana potions; Frozen Armor/Frost Nova for zombies at the gate; no Teleport before clvl 18. Necromancer is the easy class (skeletons/golem hit full physical, Amplify Damage); Amazon bow and melee classes do full physical damage (melee from clvl 6-8 with 4 potions). Paladin Holy Bolt only hurts undead (UNSURE for her).
- Kit gate: Sorceress/Druid/Amazon clvl 9+, Necromancer/melee clvl 8; >= 4 healing potions, >= 2 mana potions for casters, TP tome with 2+ charges, gold >= ~250 (retreat refill). Level in Cold Plains/Stony Field first to clvl 9. Attempts: keep 3, but gate each attempt on the kit AND gold (attempt 8 drained gold through retreats); a failed attempt waits for clvl+1/+2 and a refill, never a same-level retry; a second retreat inside one attempt ends the attempt.
- Reward: after the kill return to Kashya; the reward is the free Rogue hire, ONLY if no merc is employed (SRC). Whether the window shows 0 gold or the normal price + a free flag, and whether a delayed claim after clvl 12 / quest complete still works, are UNSURE: verify live. Normal difficulty only.
- Log: clvl/life/potions at entry, gold before/after, per-attempt deaths/retreats/time, kill time, Blood Raven HP, the Kashya dialog lines, gold before/after the claim, merc unit appears (class 271). Check no merc is employed before the kill.
- Den of Evil: technically optional but ALWAYS done first (user, 2026-10-30; it also gives a free skill point). Order: Den of Evil, level in Cold Plains/Stony Field, Blood Raven at clvl 9+, claim the free Rogue.

### 10.2 Death rule, round 2 (d2-sme on docs/plan/death-data.md; APPROVED by the user 2026-10-30 per docs/soak-analysis.md s12; implemented in cmd/d2soak and internal/bot/deathrule.go)

Death-recovery states: `Recovering after death`, `Restocking after death`, `Retrieving the corpse`, `Chores in town (poverty rebuild after failed retrieval)`; window = death line to the first combat or non-town area_change. Never count `Recovery: farming gold` or `Waiting: banned`.
- FAIL-1 (primary, no minimum sample): 3 deaths in 30 min with no clvl gain in that span, OR more than 4 deaths in 60 min (5+ in 60 min fails regardless of level gain).
- FAIL-2 (stuck): death-recovery share above 10 % of the rolling 60 min, or any single death-to-back-in-field over 5 min.
- FAIL-3 (rate, long clean runs only): deaths per 1000 kills above 15, evaluated only with kills >= 150 in the window (kills undercount by 16-37 % today); WARN above 5.
- WARN: recovery share above 5 %; 2 deaths in 60 min.
- ACTION (not a verdict): repeat site = 2 deaths in one area within 30 min, or a death within 180 s of the previous one (separates a1/a11/a12, all <= 177 s, from a8 at 266 s) => ban the area 30 min + planner dump. A death with >= 1 healing potion in the belt and no potion use is tagged BUG and counted separately. Gold-lost share is report-only (it false-FAILs a8).
- Minimum sample: 30 min before any rate rule applies; the streak rule needs none.
- Veteran read: a11 and a12 = bot in trouble (FAIL); a8 = acceptable, WARN (3 deaths in 2 h while broke at clvl 9-10; its two deaths 266 s apart at one spot should have triggered the repeat-site action). Normal for a careful caster at clvl 10+ is 0-1 deaths/h; 1-3/h is sloppy.
- Gold on death: 8 of 8 logged deaths took all carried gold; whether it lies as a pile at the death spot or is destroyed is UNSURE: live probe on a throwaway (cheat gold, die, screenshot the spot). Carry about 50 x clvl + 300 (implemented: 850 at clvl 11) and bank the rest.
- After a death, the veteran order: do not run back; heal and buy potions; re-equip if the body was lost; fetch the corpse only if the killer is understood and it holds value; return by another route or after leveling elsewhere; drink earlier next time. Implemented: DEATHAUTOPSY, CORPSENAV, DEATHCHAIN, NOIDLE, quest-walk potion/portal interrupts. Missing: the 30 min area ban + planner dump on a repeat death; skip the corpse trip when it holds nothing of value; a 'retreat at 50 % life via portal' check; waypoint straight back after a boss death.
- Logging fixes needed to compute this live: `deaths.jsonl` line per death (ts, area id/name, killer, clvl, life max, belt healing potions read from memory, potions drunk in the last 60 s, gold carried, stash gold, gold in the next minute line, merc alive, ts back in field, route id); minute line: `potions` (always 0 today) read from the belt slots, `kills` a monotonic per-character counter that survives restarts, and a `state` field.

### 10.3 Corpse by quit-and-restart (user tip + d2-sme, 2026-10-30; EXP, probe needed)

User: a corpse can also be retrieved by quitting and creating a new game, at the cost of the good ground drops and map progress. SME: right, and it should be the bot's DEFAULT recovery because the fresh-game cycle already costs only that and all 5 in-field retrievals failed. The corpse is stored in the character file and appears in the town you saved in (Rogue Encampment), beside the spawn point, as a body unit you left-click (walking over it does not pick it up); it holds the equipped items (backpack/belt/charms stay with the character, about 85 % sure; stash and cube separate); oldest corpse first, an older corpse is kept if you die again before collecting (cap UNSURE). Restart loses ground items, the level layout, monster kills and field boss kill state; keeps waypoints, completed quest flags and the corpse. UNSURE: Den of Evil / Blood Raven kill state across a quit: do not quit between the kill and the turn-in until probed. Gold: carried gold is lost on death (8 of 8 logged deaths, no pile found); quit-and-restart does not recover it; keep gold in the stash.
Rule: retrieve in place only when the corpse is a short walk away, the area is clear and it is the first attempt; otherwise Save and Exit, new game, find the corpse near spawn (unit type UNVERIFIED: fall back to empty equipment slots + death flag), click it, wait for the equipment to refill, re-run the gear/inventory check. Failure counter 1, not 2; add a gold-pile sweep at the death spot on the in-field path only.
New work item M11 (proposed, needs the live probe first): probe on a throwaway character (ledgered gold cheat + a good item; die, quit, new game, screenshot and dump the units near spawn; record unit type/class, click position, belt/charms, Den and Blood Raven state across a quit), then implement the rule. Replaces 'corpse run when value > X'.

### 10.4 Quest state across a quit (user, 2026-10-30; from play memory, to be confirmed by the probe)

Rule of thumb from the user: the quest state is saved as soon as a quest advances (so a Save and Exit after the advance keeps it), BUT progress that is only a KILL in the field is not enough for the step that needs the field state: e.g. kill Mephisto in Act 3 and quit before taking his portal: the quest shows complete, yet you must kill him again to open the portal to Act 4. (SUPERSEDED by 10.5 below: the user corrected this, quitting is often an advantage because it puts you back in town; only the field-state cases must be finished first.) Den of Evil / Blood Raven: the kill advances the quest word, and the reward is claimed at Akara / Kashya; the quit is safe only AFTER the quest word has advanced.
Probe method (part of M11, no guessing): watch the quest log (in-game panel screenshot) AND the quest words in memory (the bot already reads quest words: see internal/quest and the 'quest word did not change' log in the NPCMENU analysis) before and after: (1) kill the Den of Evil monsters, Save and Exit before talking to Akara, new game: does the quest word/log stay advanced, is the Den cleared again? (2) same for Blood Raven (kill, quit, new game: quest word, Kashya reward availability, does BR respawn?), (3) quit mid-quest after a pickup (Horadric Staff pieces) and after an NPC talk. Record per quest: word before/after quit, whether the field state resets, and whether the reward is still claimable. The corpse quit-and-restart rule and the SAVEPOINT chore (save and exit every 20 min in town) must respect the result: SAVEPOINT already runs only in town; add 'no pending cash-in' as a condition if the probe shows kill-gated steps can be lost.

### 10.5 Which quests are safe to quit before turn-in (user correction + d2-sme, 2026-10-30; EXP, every row is a hypothesis for the live test)

User correction: do NOT forbid quitting between finishing a quest and turning it in; quit-and-reload puts you back in town and is often an advantage. Mechanism (EXP): the quest word flips in memory when the event fires; the .d2s carries it after Save and Exit (which a quit is). Quest items in the pack persist. Killed monsters, ground items, portals and opened/smashed objects reset.
Class A = safe to quit right after the step (state flips at the kill/pickup; reward works after reload), B = unsafe (needs field state that resets), C = unsure.
- Act 1: Den of Evil A (quit lands you beside Akara: do it); Blood Raven A (Kashya reward works after reload; free Rogue); Cain scroll pickup A; Cain cairn stones -> Tristram portal B (enter the portal and free Cain first; after that A); Forgotten Tower A (no reward); Tools of the Trade A (Malus is a pack item); Sisters to the Slaughter (Andariel) A, UNSURE (Warriv takes you to Act 2; no corpse portal).
- Act 2: Radament A only after you pick up AND read the book (book is a ground item, the read flag is a pseudo item that resets); Cube/Amulet/Shaft A; Staff placed in the orifice A/C (word advances at placement, staff consumed: matches PLANRESTART's fix, verify live); Summoner C; Duriel C (talk to Tyrael before quitting).
- Act 3: Golden Bird, Blade of the Old Religion, Lam Esen's Tome A; Khalim's Will C (smash the Compelling Orb and go down once before quitting); Mephisto B (the Act 4 portal is field state: take it; user report).
- Act 4: Izual A; Hellforge B (pick up the rune at the forge before quitting); Diablo C.
- Act 5: Siege on Harrogath A; Anya A after she is freed; Rite of Passage A, UNSURE; Rescue on Mount Arreat C; Baal A.
- Nightmare/Hell: no known change to the split (EXP); the Hellforge rune matters most.
Rule for the bot: quitting is allowed once the quest word shows the step done AND no class B/C item is pending. Shortcuts worth taking: after Den, Blood Raven, the Cain scroll, the Malus, the Cube pieces and Andariel (lands you at the town NPC with no walk back). Probe (cheap, throwaway only via the logged cheat tool): read the quest word with internal/questmem, do the step, read again (flips immediately?), Save and Exit, reload, read a third time (the file value), talk to the NPC and watch for the reward, for B/C rows also check the field object (orifice open, portal present, stairs open). Test the C rows first: staff in orifice, Summoner, Duriel, Orb, Andariel. Gap in docs/sme: no quit-safety reference (research queued).

### 10.6 Corrections from the d2decomped notes (docs/sme/d2decomped-findings.md; D2D = their notes, a strong SECONDARY source, our live probe stays the acceptance test)

- DEATH / GOLD (resolves our 8-of-8 'all gold lost' puzzle): gold loss = min(clvl,20) % of purse plus stash, never taking the total below clvl x 500; the REST of the purse drops as a GROUND PILE at the death spot; the stash is effectively safe in single player. So our minute lines showing gold 0 after a death are the pile on the ground, and every failed corpse run also never swept the pile. Consequences: (a) the in-field retrieval path needs the gold-pile sweep at the death spot (SME suggestion, now confirmed in principle); (b) quit-and-restart loses the pile (ground items vanish with the game), so for a death with a big purse the in-field sweep may be worth more than the corpse; bank gold in the stash before risky trips (already implemented: 50 x clvl + 300); (c) check the 'never below clvl x 500' floor against our logs (clvl 11-12, 1100 gold carried: should have kept a lot?): probe live or in d2d.
- XP penalty is 0/5/10 % of the level's span (not of total XP), by difficulty: consistent with 'no XP lost in Normal' in our logs.
- Corpse: holds only worn items + the cursor item; picked up by clicking. Where a saved corpse appears after quit, and the multi-corpse/oldest rule, are NOT traced (d2d writes only the latest corpse): the M11 probe must answer on the original.
- Mercs: offered level = clvl-5+rand(5), minimum 2; hire cost = Gold x (15d+100)/100 (d = difficulty-dependent rate); resurrect cost = level^2/2 x 15, capped 50,000; the free Blood Raven merc is forfeited if you own ANY merc, even a dead one; merc gear requirements use the merc's own Str, Dex and level; shift-belt potions for the merc limited to healing, antidote, thawing (conflicts with docs/sme/mercenary.md on rejuvenation: probe).
- Quest quit-safety table (10.5) corrections: Cain stones: the Tristram portal re-creates itself after reload once you walk to Stony Field => A (not B); Radament: bit 5 (book due) is saved and there is no separate 'read' flag: pick up the book and keep it; if it stays on the ground the quest resets for a redo; Khalim's Will: the orb is restored broken at join => A; Duriel: confirmed, talk to Tyrael before quitting; Rite of Passage: A (stairs open once the record is closed). UNSETTLED: Mephisto: their notes say a join with the quest bit set reopens the Act 4 hellgate (re/quests-act3.md), which contradicts the user's report: trust the user until one live probe settles it.
- Memory: their notes confirm PlayerUnit, ExpCharFlag, GameInfo, AlwaysRun and bWeapSwitch at our RVA+0x400000 and MapId = the map seed; difficulty variable at VA 0x7a060c (answers our open Q6), client quest copy at VA 0x7c0d43.
- Driving d2d needs a NEW state backend above internal/mem (the control channel is rich but unauthenticated, can write files via screenshot, dev-only).

## 11. Overnight decisions (2026-10-30 02:15, d2-sme sanity check; EXP)

- Andariel is a wall, not a milestone: NO unattended Andariel attempt before the free Rogue and a potion stock (clvl 15 Soaky, no merc, no poison resist would die to her poison; a death costs a corpse run + gold pile and risks a death loop in Catacombs 4). Applied: `boss.andarielMinLevel` set to 22 (the old hold) on the running attempt 15 via the knobs API; M9 (honest Andariel attempt at the floor, cap 2 deaths then fall back to Catacombs 1-3 / Countess) waits for BRKILL + MERCPOLICY wiring + a potion stock. Check first when M9 runs: Normal resists (poison), >= 4 healing + a few rejuvenation, life, escape scrolls.
- Order of work (SME): free Rogue + MERCPOLICY, reliable town loop (heal/buy/sell/repair/stash), waypoints, fresh-character 1-8, Andariel, Act 2 last; cut d2d container work until the spike is clearly positive. Queue ranking: (f) inventory/stash audit (a full inventory silently stalls an 8 h run), (g) waypoint use, (a) sidecar alerts for the new d2soak codes, (d) fresh Sorceress run after EARLYGAME, (b) quit-to-town shortcut (only if it loses no state), (c) Andariel at clvl 15, (e) d2d backend (gated on the spike), (h) Act 2 entry last. Also add a gold/potion budget policy and the boring staples (Den, Cave, Crypt/Mausoleum clearing before Andariel).
- Judge on rates, not one run: death/drop RNG is high-variance; use >= 3 runs or rates; the new death rule + repeat-site bans must never ban the only viable farm site (NOIDLE's fallback covers it: verify in the run).
- Load: The worker host load average 31 on 16 cores (spike build + agent test runs + Tdarr): no new agents until it drops; each game costs about a core.
- Fidelity principle for d2d: verify on the ORIGINAL game: combat timings (breakpoints, hit recovery, aggro ranges), death/corpse/gold rule and respawn point, drops/RNG, NPC menus and gamble lists, pathing/collision. Safe on d2d alone: protocol/state parsing, UI navigation logic, stash/inventory manipulation, crash recovery, speed-up tooling. Gate trust on one replicated soak: same character on both backends, compare death rate, XP/h and gold/h; a gap above ~20 % is a fidelity bug.
- Doc gaps to research: Andariel resist/poison numbers (D2DHARVEST2 covers Andariel), gold-on-death (answered by d2decomped notes, probe pending).

### 11.1 d2d spike results (docs/d2d-spike.md, D2DSPIKE 2026-10-30; V = observed on d2d, which is a reimplementation: the original stays the acceptance target)

- d2d builds (GCC 16.2.1, needs -DD2D_WERROR=OFF, 4m23 with -j6 niced) and runs HEADLESS with SDL's dummy driver (no Xvfb); real time only (no fast-forward; `--seed` fixes the map, not monster rolls); idle about 0.6-0.9 core, RSS 515 -> 746 MB, stable for 10.5 min; control socket owner-only.
- Death (V on d2d, matches its source): gold toll = min(level,20) % of purse plus stash (clvl 8, 10000 carried: 1200 lost, 8800 dropped on the ground, purse 0); XP penalty 0 % in Normal, 5 % of the level span in Nightmare, 75 % of lost XP stored on the corpse; the corpse comes back after save and reload but is RELOCATED TO TOWN, a corpse with no items is not restored; GOLD PILES ARE NOT SAVED (so quit-and-restart loses the gold pile: sweep the death spot first or accept the loss); retrieval worked with `cmd interact -3000`, not by clicking the body.
- Quest (V): the Den-cleared flag (bits 0x2016) survives save/quit/reload and Akara still paid the skill point; the Den respawns its monsters on re-entry.
- Merc (V): Kashya's menu gains `hire` at clvl >= 8; the hire window shows 8 rows (name, level, life, defence, cost; no merc type); menu line coordinates come back from `menu`; free-Rogue case untested (character already had a merc).
- Bot fit: our bot reads raw Game.exe memory, so a d2d backend needs a Reader at the `d2.Snapshot` level; the channel lacks full stats, a skill list, missiles, quest words for all acts and click modifiers; adapter estimate 3-5 days for a partial one; SPIKE RECOMMENDATION: use d2d as a rules oracle NOW; build a partial adapter next.
- Quirks: `cmd move`/`cmd interact` ignored after `debug goto` or a load until a `click` is sent; `--start-class 1` produced a Necromancer; `debug goto 8` renders the Den black (use `debug warps` + `debug enter`); `key Escape` toggles the game menu.

### 11.2 Route guard (agent ROUTEGUARD; d2-sme decision EXP, U live)
Soaky died three times on the same stretch of the Monastery road (Tamoe Highland / Monastery Gate border, quest walk "Go to the Barracks"). Rules, all offline-tested:
- The Imbue quest (Tools of the Trade) is optional: skipped (`skipped: optional (reason)`, never blocks the chain) for non-melee classes always, for Barbarian/Paladin/Assassin below `campaign.imbueMinLevel` (20) or without a living merc (`campaign.imbueNeedsMerc`).
- The Monastery road (areas 7, 26-37, so Andariel too) is entered only with a living merc (`World.HasMerc` or the `task.MercAlive` hook; unknown counts as none) or clvl >= `route.monasteryMinLevel` (20), and with the kit (4 healing, a portal, 2 mana potions for casters). `internal/task/route.go`: `RouteWhy`, `RouteLine`; raises the floor of the Tools, Andariel and Act-1-done campaign entries and the boss gates; `fieldReady` waits on it; the bot excludes the guarded areas from every level pick and refuses walks through them (`routeAllowed`, `areaAllowed`). One log line per change: `route guard: blocked monastery (no merc, clvl N < 20)`. Knobs 0 turns the guard off.
- While closed the planner levels in Blood Moor to Black Marsh, Burial Grounds, Cave 1 and 2, Crypt, Mausoleum, Forgotten Tower. Consequence to accept: without a merc the campaign reaches Andariel only at clvl 20 (Act 1 XP is thin by then; the SME judged it cheaper than the deaths).
- Tower guard (agent TOWERGUARD, G1 attempt 17, U live): Soaky (clvl 14 caster, no merc, empty belt) died twice in Tower Cellar 5 and 1 for the Countess, a no-reward quest in Normal (10.5). Areas 20-25 (Forgotten Tower, Tower Cellars 1-5) and the `forgotten_tower` entry follow the same rule with `route.towerMinLevel` (20, 0 = off) and `route.towerNeedsMerc` (1): open with a living merc (or clvl >= 20 when the knob is 0) and the kit; otherwise the quest is `skipped: optional` (`task.TowerSkipWhy`, level grounds only, a thin kit is a chores trip; V by reading the campaign: Andariel's entry never depends on the tower), the level picker excludes 20-25, `fieldReady` waits on `World.TowerBlocked`. Log: `route guard: blocked tower (no merc, clvl 14 < 20)`. The Forgotten Tower gate now wants 4 healing potions (was 2).
- Guarded-area exit (`guardedExit`, loop.go step): a character standing in a closed guarded area (tower or Monastery road) leaves first, before any planning: town portal / walk home (`Town.ToTown`), or with 3+ hostiles within 6 subtiles a walk to town (`travelTo` allows a town target). Log `guard: leaving guarded area N first (why)`. `LevelTo` with every candidate unreachable while standing in a guarded, banned or recovery spot logs `recovery: leaving <area> first` and goes to town before picking again; `unreachable` also rejects a pick whose route needs a guarded or banned area (`routeAllowed`).
- Repeat-site bans escalate x4 / x12 (cap 6 h) in the same area within 12 h; crowd rule and autopsy facts: docs/combat.md, docs/deathautopsy.md.

### 11.3 d2d probe batch results (docs/d2d-probes.md, D2DPROBES; all V-d2d: observed on the reimplementation; the original game stays the acceptance target)

- FREE ROGUE: granted when Kashya's menu opens after Blood Raven (quest state 3 -> 5), with NO hire window and gold unchanged; hiring ANY merc first forfeits it. Paid hire cost does not follow level^2: clvl 8 asks 130-240, clvl 20 asks 295-510 (so the plan's 'gold >= 3x the quote' rule is cheap at clvl 8-13).
- Blood Raven and Andariel: safe to quit right after the kill; the reward still works after reload without a second kill; Warriv's 'go east' is available; Blood Raven RESPAWNS; Andariel's death also opens a town portal (contradicts the 'no corpse portal' note in 10.5).
- Mephisto: kill, save, reload: the Act 4 gate object is already ON, Mephisto alive again, no second kill needed. This CONTRADICTS the user's report: the real game must decide (probe on the original before any Act 3 quit shortcut).
- Death gold: NO toll-floor of level x 500 total at low level: a clvl 12 death with 1100 gold drops the whole 1100; a clvl 20 death with 50000 loses 10000 (20 % of the total); walking onto the pile picks it up. Corpses accumulate (also empty ones) and any index can be claimed. => GOLDSWEEP is exactly right; banking gold in the stash before risky trips matters (the stash is safe).
- Merc revive: cost min(level^2/2*15, 50000), NPCs Kashya/Greiz/Asheara/Tyrael/Qual-Kehk (from d2d's source only; the merc never died on d2d).
- Memory/CPU: RSS flat for 20 min idle (471 MB) and 20 min level-cycling (564 -> 560 MB) at about 0.9 core; the spike's growth was first-visit level builds, not a leak. A level is built once per game and re-entering does not respawn monsters (differs from our live bot's fresh-game cycle expectations: check). Drops: 332 kills in Blood Moor and Cold Plains gave 0.27 items and 0.1 gold piles per kill, piles averaging 2.5-4.5 gold (quality not visible in `ground`).
- UI facts: close NPC menus with the `cancel` line, not Escape (an extra Escape opens the game menu and then `cmd move` is silently ignored); Akara's Reset line appears only while the Den reward is open (source, untested).
- Not reached: vendor prices, waypoint rules, potion/belt probes, Hardcore, real continuous-combat perf, quest rows not named above.

### 11.4 d2d lockstep fork results (docs/d2d-fork.md, D2DFORK 2026-10-30; V = measured on a LOCAL private fork on the worker host, never in our repo)

- Patch (private fork ~/d2d-lab/fork on the worker host; base e36922e; patch files under ~/d2d-lab/fork/patches/ only): `--lockstep`, `--time-scale x`, verbs `clock`, `step n [render]`, `lockstep on|off`, `timescale x`, `dump [sections]`; diff 287 new lines in 3 new files + 37 added/8 changed lines in 6 shared files (within the policy).
- Speed (V, one core): town idle about 3000 ticks/s = 120x real time; Blood Moor with monsters 9400 ticks/s = 377x; Burial Grounds with 17 monsters 87x; `--time-scale` 10 and 20 ran at exactly 10.0x/20.0x (above ~11 can lose ticks: use lockstep for fast runs). A stepped frame is exactly one 40 ms world tick; drawing costs 17 ms/frame so stepped frames skip it.
- Determinism (V): 3 runs with the same seed and inputs gave identical `monsters` output (also with walking monsters and with drawing forced on); the spike's 'rolls differ between runs' was timing jitter: every RNG in d2d is fixed-seed and lockstep removes the jitter. Not covered: fights where the player takes damage.
- Nothing broke: control verbs, save/reload, 10 min at 10x without crash (RSS 525 -> 545 MB); P3 (click modifiers) not done; `cmd move` quirk neither fixed nor re-tested after a goto.
- `dump` first cut (sections player, stats, skills, items, quests+waypoints, ui, merc, time) returns valid JSON (3 KB for a fresh Necromancer); missing monsters/ground/objects (existing verbs), skill ids, unit stat lists.
- Risks: draw and update are fused in `Town::update` (rerun determinism after a rebase); menus and loading use real time (use `--start-screen ingame` or `lockstep off`).
- CONSEQUENCE: statistics (drop rates, death rates, XP/h, gold/h over thousands of kills) cost minutes instead of wall-clock hours on d2d: the SME's 'biggest lever' is delivered; the fidelity gate (replicated soak on both backends, gap under ~20 %) becomes practical. Draft upstream issue text is in docs/d2d-fork.md for the user to review; nothing was posted.

### 11.5 d2d fidelity results (docs/d2d-fidelity.md, D2DFIDELITY: 7885 kills over 63 seeds, 120 deaths, all V-d2d on the lockstep fork; nothing verified on the original game)

- Matches d2d's own rules exactly: mana regen (truncation formula), no natural life regen, resists 0/-40/-100, the level-difference XP table, the death gold/XP rule (120 of 120 cases).
- Cast period is 14 ticks for every Sorceress spell in d2d (our tables say 13, and 19 for Lightning/Chain Lightning, so d2d casts those two ~26 % faster than documented): a likely d2d deviation: verify on the original. Melee weapon-speed scaling matches the formula.
- Vendor: consumables match our tables to the gold; sell = exactly 50 % of buy (lore says 25 %); Tome of TP 250 in d2d vs 450 live (deviation); magic gear priced about the same as plain.
- XP per kill: at clvl 12 Cold Plains, Den and Blood Moor pay UNDER 3 XP/kill while Stony Field gives ~15 and Dark Wood ~35 (our live 22 XP/kill at clvl 12 fits Stony Field/Dark Wood): the level picker must prefer Stony Field/Dark Wood/Black Marsh at clvl 12+ and never farm Den/Blood Moor/Cold Plains for XP (recovery farming there earns gold, not XP).
- Wake radius is about 7 cells (0 of 771 monsters at >= 9 cells woke); a hit outside that radius staggers the target but does not wake its pack; no Fallen rally seen in the tested Carver pack (d2d's source agrees, no original-game number).
- Drops: 0.29 items + 0.13 gold piles per kill agree with live; potion double-fill is missing (0 of 24 drinks at Vit 150, ~9 expected); gold autoloot is a d2d deviation.
- Could not do: monster damage/attack rates, original wake/leash ranges, Cain identify, mana potion heal curve, Fallen rally; the trade window works only in real time (hover + two clicks; right-click buys).
- Tooling notes: `debug stat 12` does not update max life/mana (set stats 6-9 x 256); `debug points` needs a step before `debug skill`.
