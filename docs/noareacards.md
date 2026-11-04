# noareacards: three picker change cards (drafts for the coordinator's review, no code)

> Status: SUPERSEDED by `docs/noareacards2.md` (2026-11-04, SME review). Card 1 dropped, card 2 on hold, card 3 rewritten, new first card farmfallback.

Tags: V = verified (code read in this worktree, file:line given), S = secondary (read from `docs/noarea-policy.md` or a comment, not run),
U = unverified. Nothing here was run live. Origin: user decision 2026-11-04 ("no-area: SME pick") after `docs/noarea-policy.md`
(Gate 2 seed 13, clvl 15-16). Policy cards stay paused: this file is design text only, no Go changes, no `openspec/changes/noare-*` cards yet.

## 0. What the XP-based ranking uses (shared by all three cards)

- `areaXP(clvl, area, mlvl, diff)` (internal/bot/recoveryxp.go:76, V): the traced Normal mean XP of a plain kill per area
  (`areaXPTable`, recoveryxp.go:68, tagged S: 12 seeds of the real room populations, `docs/research/monster-stats-trace.md` section 4)
  times the level-gap share `xpLevelPct(clvl, row.mlvl)` (recoveryxp.go:35, V per `docs/sme/mechanics-progression.md` 2.1/2.2). An area without
  a row, or Nightmare/Hell, falls back to `areaXPPerKill = mlvl*10*share` (recoveryxp.go:56, S: a fit, 0.45x-1.15x of the real means).
- `stepXP` (recoveryxp.go:86, V) wraps it for `PickOptions.XP`; no monster level gives -1 = ranked last (warned once).
- Table rows exist only for areas 2-6, 8, 9, 13, 17, 27, 35, 41, 44, 47 (V, recoveryxp.go:69-70). Mausoleum (19), Crypt (18), Catacombs, Jail,
  Outer Cloister, Tower Cellar have NO row: they fall to the fit. Consequence for every card below: ranking among the mid-game Act 1
  dungeons is the weak part of the XP estimate (S), so the cards use XP only to break ties between candidates and never to override a
  guard, a window or a death ban.
- Shared rule for all three cards: the XP number is a rank key, not a threshold; build-specific preferences (a melee build enters
  earlier, `Step.enter(melee)`, plan.go:422) stay in the plan data (`internal/plandata`, `Step.EnterLvl/LeaveLvl/KitLvl`).

## Card 1: idleBreak lifts the best-XP banned area, not the shortest-remaining one

### Problem (V)
`idleBreak` (internal/bot/areaskip.go:93) sorts its candidates (areaskip.go:131-142): faults before death bans, among death bans the
starter field first, then `left` ascending (shortest remaining ban). The first one is un-banned. The shortest remaining ban has no
relation to how much XP the area pays: in seed 13 the 60 s break lifted Cave 1 / Cold Plains (past window, leave 10, S: policy doc 2,
lines 41606-41607) while Cave 2 and Crypt, in window at clvl 15 (leave 15 / 17), stayed banned for minutes. `l.waitSince` is then set so the
next lift comes after 10 s (areaskip.go:149), so a sequence of single lifts walks through the worst areas first.

### Change (one paragraph)
Keep the two existing classes, fault bans lifted before death bans and starter-field-first among death bans (this ordering stays; a death
ban is danger evidence, areaskip.go:9). Inside a class replace the `left` tiebreak by a rank: (1) the plan window still holds the level
(`enter <= clvl <= leave`), (2) guarded areas are never candidates (the guard is applied by `guardedAreas` after the bans, goal.go:873, so a
lifted ban on a guarded area changes nothing today; the candidate list must skip them to avoid a pointless lift), (3) best `areaXP`, (4) shortest
`left`, (5) area id. The ranking lives in the planner (a function `Plan.RankBanned(diff, level, PickOptions, cands)` or an extra field on
`PickOptions`), `idleBreak` only builds the candidate list and asks it, so the same ranking serves card 2.

### Files to touch
internal/bot/areaskip.go (candidate build, call the ranker), internal/bot/plan.go (ranker, reuses the `windowExcluded` / `enter` logic),
internal/bot/goal.go (hands `PickOptions` pieces to the ranker via a small `r.pickOptions(s)` helper extracted from goal.go:879-881),
tests: internal/bot/areaskip_test.go, plan_test.go.

### Knob defaults (data in internal/env/tuning_bot.go)
`IdleLiftByXP` int, default 1 (0 = today's shortest-left order, for A/B and rollback). No other new knob: `AreaIdleSec` (60) and
`StarterBanMin` (3) stay. No class or skill literal: the window comes from the plan step, the XP from the traced table.
Build-specific rule: none new; a melee build uses its own `enter` through `Step.enter(melee)`.

### Tests (table-driven, two builds)
Sorceress clvl 15 (the seed 13 set: Cave 2 left 200 s, Crypt left 100 s, Cave 1 left 20 s, Cold Plains left 40 s, all fault bans) expects
Cave 2 or Crypt by XP, not Cave 1; Barbarian (melee, `enter` earlier per plan data) clvl 15, same bans, expects the best-XP
in-window area for ITS enter levels. Rows also: death ban present with a fault ban (fault first, unchanged), only death bans (starter first,
unchanged), a guarded area among candidates (skipped), an area without an XP row (ranked after rows, then by `left`), knob 0 equals today's
order (regression row).

### Risks and what could regress
- Deaths: a death-streak area is still lifted last and only after the idle limit (`starterBanMax` for death-only sets, areaskip.go:125); the
  card does not move that. Risk is the Cave 2 / Crypt pair being the nav-fault areas of seed 13 (S): lifting them first retries the exact
  walk that failed. The G1 attempt 14/17 doubling ban exists for that: `targetFailSkip` (areaskip.go:48) keeps its counts, and if an area has
  failed twice within `targetFailMemory` (30 min, areaskip.go:69) the ranker drops it to the bottom of its class (the card 2 cap, same data).
- XP rows missing for mid-game dungeons (section 0): the order among them is partly the fit. Mitigated by "window first" and by the
  knob.
- One lift per call stays; no new loop.

### Size and live rerun
S (about 60 lines of Go, 1 test file). Live: soak seed 13 as in the policy doc's rerun check; additionally every `area wait: ... farming X anyway`
line (areaskip.go:150) must name an area whose window holds the clvl when one is banned; list the log pairs (lifted, other candidates).

## Card 2: nav-fault bans become a ranking penalty, with a cap per area per hour

### Problem (V)
`unreachable` (task.go:926 walk failed, task.go:893/899 no route to the pick) and `nav-stuck` (loop.go:641, S line from the policy doc) call
`skipArea` and the ban is a hard exclude in `target0`: every banned area goes into `ex` (goal.go:866-871) and `Plan.Pick` drops it
(plan.go:404). Seed 13 (S): the walk fault from Cave 2/Burial Grounds toward Cold Plains (`trigger area of an unwanted exit`) banned
every area one after the other until `Pick` returned nothing (policy doc section 2, line 41596), yet the cause was one nav fault with an
empty tome, not danger. The doubling (`targetSkipDoublings = 3`, areaskip.go:45: 150/300/600/1200 s) exists to stop repeated failing walks
(G1 attempt 14/17), so it cannot just be removed.

### Change (one paragraph)
Split bans by cause. Death bans (`whyDeath`, repeat-site) stay hard excludes. `whyUnreachable` and `whyNavStuck` (and `whyGate`, see
questions) become SOFT: `target0` no longer puts them in `ex`; it passes them as `PickOptions.Soft map[int]time.Duration` (remaining penalty
time). `Plan.Pick` / `pickByXP` rank a soft area below every non-soft eligible candidate of the same tier, so a soft area is only returned
when nothing else is eligible (today this is "wait in town or fresh game"). A per-area cap `NavRetryCapPerHour` counts failing attempts (one
per `skipArea` with a nav reason, counted in a ring per area): at the cap the area is a hard exclude again until the hour has passed, so the
failing-walk loop of G1 attempt 14/17 cannot return. The existing `targetFailSkip` doubling stays as the penalty length (and the time the area
is "soft"), the cap is the new hard limit.

### Files to touch
internal/bot/areaskip.go (soft vs hard in the ban record: `avoidWhy` is already stored, V areaskip.go:33; add an attempt ring `navTries map[int][]time.Time`),
internal/bot/goal.go (target0 ex-building goal.go:862-872: skip soft reasons, build `Soft`), internal/bot/plan.go (`PickOptions.Soft`, ordering in
Pick/pickByXP, windowExcluded must still treat soft areas as NOT excluded), internal/bot/routeguard.go is read only (guards stay hard),
internal/bot/recoveryxp.go (`recoveryXPArea` takes the same soft penalty so recovery and level picking agree), internal/bot/task.go (the
`LevelTo` call sites record the attempt), internal/env/tuning_bot.go. `idleBreak` (card 1) then only has death bans and hard-capped faults to
lift. Tests: areaskip_test.go, plan_test.go, goal_test.go.

### Knob defaults (internal/env/tuning_bot.go, shown in docs/knobs.md by the generator)
`NavSoftBan` int default 1 (0 = old hard bans, the rollback), `NavRetryCapPerHour` int default 3 (the doubling gives 150/300/600 s per try;
three tries per area per hour, fourth is a hard exclude until the oldest of the three is an hour old), `NavSoftPenaltyXP` 0 (not used in v1; the
penalty is ordering, not a number, so no class or area literal and no tuning of XP). U: 3 is a judgment from the doubling table, to be set from the first rerun.
Nothing build-specific: the reason strings are the generic fault reasons.

### Tests (table-driven, two builds)
Sorceress and Paladin at clvl 15 with the seed 13 ban set recast as soft: Pick returns the best non-soft step; with every step soft, Pick returns
the soft one with the best XP and the nearest unbanned neighbour first; a soft area at the cap becomes a hard exclude (Pick returns "nothing");
a death-banned area stays hard even when also soft; `windowExcluded` ignores soft areas (a soft in-window area must not make the floor rule
refuse a past-window pick, plan.go:434); knob 0 equals today. A walk-failure sequence test (fake runner, no game): the same area fails 4 times in
an hour, the 4th pick excludes it and the 5th after one hour does not.

### Risks and what could regress
- This is the card with the death risk and the G1 14/17 risk. A soft area is picked again within 150 s; if its fault is a death trap exit the
  character walks into it again. Mitigation: the cap, the ranking below every healthy area, and `deadNow()` (areaskip.go:157) keeps a death
  walk out of the attempt count. Do not apply soft to repeat-site or death reasons.
- A soft pick right after the failure retries the same unreachable cell. The ranker therefore prefers a non-soft area; only a world where
  everything else is excluded retries. Watch for `leveling area unreachable` repeating more often than the cap (a counter in the log).
- `targetFailSkip` counts and the new ring must agree; one source of truth (the ring) should feed both, else two counters drift.
- Interaction with card 1: the ranking helper is shared, so land card 1's helper first.
- Interaction with route/border bans (`whyRoute`, `borderLoopSkip`, loop.go:808/1173): left hard in v1, say so in the card result.

### Size and live rerun
M (picker data flow in 4 files plus tests). Live: soak seed 13 twice (same seed, knob 1 vs 0 for A/B): `no eligible reachable` lines per hour,
`leveling area unreachable` lines per area per hour (must be <= cap), deaths per hour no worse, XP past clvl 16; extra grep: every soft pick
logged as `area picked soft` (a log line the card adds) with the preceding fault.

## Card 3: gold-for-kit recovery when the kit guard closes the high-XP chain

### Problem (V for the code facts, S for the seed)
The kit guards close the chain: `guardWorld` (internal/bot/routeguard.go:74) feeds `task.RouteWhy` / `TowerWhy` with belt healing potions plus
pack healing, mana potions, `HasTP`, merc alive (routeguard.go:78-88); closed guards put the Monastery road and the tower into `ex`
(`guardedAreas`, routeguard.go:137-141). Seed 13 (S): `0 mana potions < 2`, `no town portal`, no merc. `Pick` then falls to the open field and
caves (low XP), and on the broken exits to nothing. The bot does have a recovery phase (internal/bot/broke.go), but it starts only through
`resource.RecoveryCheck` (internal/resource/recovery.go:45): gold < `RecoveryKitGold` (500) AND `Shortfall > Gold`, outside a cooldown
(`RecoveryCooldownMin` 15, unless `Force` or `LowKit`, recovery.go:56). Two gaps follow: (a) a guard closed by a potion shortfall that the gold
can already pay (`Shortfall <= Gold`, recovery.go:54) never starts recovery and the kit trip (`kitTrip`, kitgate.go:96) runs only from `KitGateLevel`
(8) and only with a way home; (b) nothing connects "the guard is the reason the high-XP chain is closed" to the recovery start, so a closed guard
sits behind the 15 min cooldown and the generic "gold below 500" rule, and recovery's farm area is `recoveryArea` (broke.go:129): the old safe tier
(`safeTier` broke.go:36) plus `recoveryXPTier` (recoveryxp.go:14), without Mausoleum, Crypt, Catacombs; gold per hour there is low (a plain kill
drops a pile 13 percent of the time, recoveryxp.go:9-13, S).

### What exists
- Recovery state machine and its farm slice (`updateRecovery`, `recoveryStep`, broke.go:56/181), XP pick for the farm area
  (`recoveryXPArea`, recoveryxp.go:115, knob `RecoveryXPPick` 1), kit price knobs (`KitTPGold` 100, `KitPotionGold` 30, `RecoveryKitGold` 500,
  `RecoveryTargetGold` 900, `RecoveryPotionFloor`, tuning_resource.go:40-90), the kit definition (`kit`, `kitRead`, `missing()`, kitgate.go:26-60) and
  trip (`kitTrip`, `KitTripCap` 2), the town sell and shop code (`Town.sell` chores.go:490, `shopVisit` chores.go:425, `needSell` chores.go:352,
  `kitGold`/`kitPotions` basicsfix.go:83/131, `buyFirstTP`).
- The merc gold farm (`merc.GoldFarm`, internal/merc/farm.go:18; `MercGoldFarm` hook routeguard.go:121, knob `merc.goldFarmMaxLevel` 16):
  keeps the character in `merc.FarmAreas` while a merc is missing and gold cannot pay the next merc action. It is not a kit-gold step and it
  excludes every area outside `FarmAreas` (routeguard.go:127), so it is a different goal with a fixed area list.

### What is missing
1. A trigger: "chain closed by the kit guard and gold < kit cost" is not a recovery start condition, and the `Start` check ignores the guard state.
2. A kit cost that matches the GUARD (potions to open the Monastery guard, tome refill) rather than `Shortfall` of the belt plan: the guard minimum
   (`task.RouteWhy` potion and mana minimum, U: values not read) and the kit minimum (`KitMinHP`/`KitMinMP`) can differ.
3. The area choice for this recovery: the best-XP area that is safe for a kit-less character (needs the kit guards of `guardedAreas`, the
   `kitSafeTier` narrowing while `kitGateActive`, and card 2's soft penalty so a faulty area is not the farm area).
4. A `then` step: after gold reaches the cost, shop (potions, tome) and `Pick` again. This exists (recovery End at `RecoveryTargetGold`, the
   chore plan buys the kit, `stepArea = 0` re-picks, broke.go:86), only the end condition must be "kit cost reached", not 900.
5. A bound: `RecoveryMaxMin` (45) already stops a dry area; the card needs a per-hour cap so a dry game cannot loop (G1 attempt 9: gold 37 for 90 min, S broke.go:38).

### Change (one paragraph)
Add a recovery start reason `guard-kit`: in `updateRecovery` (broke.go:56) a new input `GuardChain bool` to `resource.RecoveryInput` is true when
(a) a route or tower guard is closed (`routeGuardWhy`/`towerGuardWhy` non-empty) for a kit reason (potions, tome; not merc, not clvl floor),
(b) the chain behind the guard would be in window and rank above the best open area by `areaXP` (the guarded area's XP minus the best open area's
XP >= `GuardKitXPGainPct`), and (c) gold < the cost of the minimal kit that opens the guard (`GuardKitGold`, default derived from `KitTPGold`,
`KitPotionGold`, `KitMinHP/MP`, not a new literal). When true, RecoveryCheck starts (cooldown bypassed, like `Force`) with a lower end target
(the guard kit cost, not `RecoveryTargetGold`). The farm area is `recoveryXPArea` (best XP among safe areas, soft-penalized nav areas last).
The existing sell/shop chores run at the end; the bot re-picks. No new walk code, no new sell code.

### Files to touch
internal/resource/recovery.go (input fields and the new start reason, end target), internal/bot/broke.go (build the input; end at kit cost),
internal/bot/routeguard.go (expose the closed-guard reason class: kit vs merc vs clvl, read only; `task.RouteWhy` is another owner), internal/bot/recoveryxp.go
(soft penalty hook), internal/env/tuning_resource.go (`GuardKitRecovery`, `GuardKitXPGainPct`, `GuardKitCapPerHour`), docs/knobs.md is generated.
Tests: internal/resource/recovery_test.go, internal/bot/broke_test.go.

### Knob defaults (internal/env/tuning_resource.go)
`GuardKitRecovery` int 1 (0 = off), `GuardKitXPGainPct` int 25 (U: judgment; the guarded chain must pay at least a quarter more XP per kill than the
best open area, else the plain pick stands), `GuardKitCapPerHour` int 2 (guard-kit recoveries per game hour). Class- and build-specific rules: the
mana-potion need is read from `resource.ManaNeed(class)` (kitgate.go:41), a melee or no-mana build gets `minMP = 0` through the existing data, so
no class literal is added. Which areas are "behind the guard" comes from the plan's guarded area sets (task.RouteGuarded/TowerGuarded), data in
internal/task, not here.

### Tests (table-driven, two builds)
Sorceress (needs mana, `minMP = KitMinMP`) and Barbarian (no mana need): gold 40, healing 0 / mana 0 / TP missing, guard closed, Crypt in window;
expects the start with reason `guard-kit` for both, the end target equal to the guard kit cost (higher for the Sorceress), and the farm area =
best XP area from `recoveryXPArea`; rows: gold covers the kit (no start, existing trip), guard closed for merc/clvl (no start, not a kit
reason), XP gain below the percent (no start), cooldown bypass only for this reason, cap per hour reached (no start), `kitGateActive` narrows the
farm area to Blood Moor/Cold Plains, recovery ends at kit cost, then `Pick` reruns (fake runner, no game).

### Risks and what could regress
- Deaths: recovery farms without potions. The existing safety (`recoveryPackDanger = 1`, `kitSafeTier`, `dungeonMouths` navigator avoid) stays;
  the card must not widen the farm tier. Any new area in the farm list is a policy decision (question 4).
- Time sink: low gold per hour (S). A recovery of 45 min that earns 30 gold is the G1 attempt 9 shape; the cap and `RecoveryMaxMin` bound it.
- The recovery cooldown (15 min) exists against flapping; bypassing it only for `guard-kit` must still obey the per-hour cap.
- Potion price changes across acts or with a charm/tome stock (U: Normal Act 1 prices only; the card is Normal Act 1 like kitgate).
- Interaction: `MercGoldFarm` already excludes all areas outside `FarmAreas` (routeguard.go:127). The two must not both own the farm area; the
  card says the merc farm wins (it is the older rule) and `guard-kit` waits while it is on.

### Size and live rerun
M (resource state machine plus bot wiring plus tests). Live: a throwaway character at clvl 12-14, gold 40, empty belt and tome, guard closed
(setup writes declared before the run start): expect `recovery: start (guard-kit ...)`, a farm area line with the XP estimate, gold progress
lines, the kit bought, `recovery: end`, and a `level plan` line naming an area behind the guard; deaths 0 in the run. Soak seed 13 as in the
policy doc, compare time spent in town and XP/hour.

## Order of work and dependencies

1. Card 1 first (S, no data flow change). It creates the shared ranker (candidate list plus window plus `areaXP`) and the `r.pickOptions(s)` helper.
2. Card 2 next (M): needs the ranker and helper of card 1; it changes `ex` building, so it must not run in parallel with card 1 in the same files
   (areaskip.go, goal.go, plan.go). After it, `idleBreak` mostly has death bans left, which is the shape card 1 assumed.
3. Card 3 last (M): its farm area uses the soft penalty of card 2 (a faulty area must not be the recovery farm) and the XP rank of card 1. It touches
   internal/resource (another owner) and routeguard.go read-only, so it needs its own card with `owns:` and a spec delta for the new recovery start
   reason in `openspec/specs/` (the death-rule or soak-gates numbers do not change; check `docs/process` for whether recovery has a spec).
Cards 1 and 3 can be developed in parallel only if the ranker is a separate file (plan_rank.go); card 2 is the serial one.

## Questions for the user

1. Card 2: which reasons are soft? Proposed: `unreachable` and `nav-stuck`. `gate` (a gate refusal) and `route` / border-loop bans are left hard; say if
   gate should be soft too.
2. Card 2: is 3 failing attempts per area per hour the right cap, or should it be 2 (start strict, loosen after a rerun)?
3. Card 1: is a lift of an in-window area acceptable BEFORE a past-window starter field even for a death-banned class (a death ban stays last)? The
   card says yes within the fault class.
4. Card 3: may a recovering character without a kit farm areas beyond the current safe tier (e.g. Crypt or Mausoleum) for better gold per hour, or
   must it stay in Blood Moor / Cold Plains (kit gate)? Default is: no widening.
5. Card 3: is the guard-kit XP gain threshold (25 percent) the right bar, or should recovery start whenever a kit-closed guard hides an in-window
   area, whatever the XP?
6. Card 3: should the merc gold farm and the guard-kit recovery be merged into one gold goal later (one area list), or stay two rules?
7. XP table gaps (section 0): do you want the traced XP table extended to Crypt, Mausoleum and the Cloister/Catacombs areas before cards 1 and 3 land?
   A separate trace job (needs the emulator, no live game) would remove the "fit" from the ranking.

## Shared-file lines for the merger
See openspec/changes/noareacards/merge-lines.txt.
