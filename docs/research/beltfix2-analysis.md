# Gate 1 20261103 BELT (37 vmin) and NOKILL (11 vmin): analysis and fix plan

> status: current. Analysis only; no Go code changed. Supersedes the cause theory of docs/research/beltfix-analysis.md for this run (that fix was active but did not cover this case).

Tags: V run = read from the run data of gate1-20261102-2210 (the minute lines in `docs/probes/soak-logs/gate1-20261103-minutes.jsonl.gz` on branch agent/gate1run-20261103, and `logs/bot-0.log` in the gate run directory on the worker host, read only); V read = read in the code at origin/main c216f970 (bot code of the run: cd4c7cba, the same files for everything cited here); I = inference; U = unverified.

## 1. The windows

The d2dsoak rule was replayed over the 1147 minute lines (internal/d2dsoakstat/invariants.go: a line counts outside the five towns when `packHP == 0` and `beltBoxes` has no `Healing`; NOKILL counts lines with neither new kills nor new XP). The replay gives the same maxima as d2dsoak (37 and 11). V run.

| window | lines (virtual min) | wall | clvl | area | gold | state |
|---|---|---|---|---|---|---|
| BELT A (fails) | 218-254 = 37 | 22:26:19-22:29:17 | 9 | 5 Dark Wood, 10 Underground Passage 1, 4 Stony Field | 332-366 | farming, `Quest: fighting (Level up to 17)`, and 18 lines `Life 58% in area 5, no potion, and the way to town failed 3 times; retrying in ..` |
| BELT B (at the limit, passes) | 268-272 = 5 | 22:31:03-22:31:20 | 9 | 2 Blood Moor | 22-27 | recovery, `Recovery: farming gold` |
| NOKILL A (fails) | 241-251 = 11 | 22:28:22-22:28:55 | 9 | 5 | 349 | inside BELT A: the low life wait |
| NOKILL B (fails) | 651-661 = 11 | 23:12:20-23:13:09 | 13 | 1, 3, 10, 6 Black Marsh | 272 | `Returning to the field`, then farming |

The other NOKILL stretches are 10 (227-236, also inside BELT A), 8 and 7. V run.

This is not a poverty case. During the 37 minutes the character carried 332 to 366 gold, enough for 11 healing potions at 30 gold. V run.

## 2. BELT A, step by step (V run unless tagged)

1. 22:25:45-22:26:19: in Underground Passage 1 and Dark Wood at clvl 9 (Vile Lancer packs, `hostiles: 102` at the back-off cap). Out of mana throughout (`combat out of mana: backing off`, 30+ lines). The belt goes from 6 potions (line 215) to empty (line 218); 6 potions drunk in about 30 wall seconds.
2. 22:25:53: trip `[resource: life low, cannot regenerate, free heal]`. Two portal casts in one second, both logged `d2d portal: cancelled, nothing clicked (bot: safety drink-life / drink-rejuv)`. The tome went from 2 charges to 1 to 0 across the two attempts (`inv-tp-dump counted=2`, `counted=1`, `counted=0`). The bot crossed the cave mouth between them (Entered Underground Passage, Entered Dark Wood, same second). So "nothing clicked" was false: the click was made and the charge spent. The code shows why: `d2dPortalWith` (cmd/d2bot/d2dwire.go) returns `portalDead` ("nothing clicked") also from the wait loop after the right click (V read).
3. From then on, with 0 charges and 0 potions, the bot started an emergency trip `[0 healing potions low on healing potions]` every 3 to 7 wall seconds, 60+ times. Every one failed. From Dark Wood: `waypoint to town failed: nav: waypoint unavailable: none on level 5` (62 times; all 63 waypoint-to-town failures of the run fall in this window). From Underground Passage 1 (a cave): `no town portal available` with no fallback at all.
   - The Dark Wood waypoint was not activated yet; it was first touched at 22:29:18 (`wp: activated area 5`). `nav.openWaypoint` (internal/nav/waypoint.go) reports "none on level N" when the level map has no waypoint preset streamed in and `knownWaypointRel` has no entry. The table only has 75, 103 and 109, so no act 1 field (V read).
   - `walkToTown` (internal/town/trip.go) walks home only from Blood Moor, plus the hop chain Den/Cold Plains/Tristram/Cave. Dark Wood, Stony Field and Underground Passage are not covered (V read).
4. **The decisive defect.** The character was in town six times during the window and left each time without buying anything. Arrivals: 22:26:27, 22:26:44, 22:26:58, 22:27:38, 22:28:09, 22:28:43. Three came from the save point (`Save point: saving the game`), three from the failsafe ladder step 3 Save and Exit (`no progress in state resupplying for 240s`). Every re-entry lands in the Rogue Encampment. Each time the next lines are `state recovering -> clearing`, `planner .. Dark Wood`, `wp: used 1 -> 3`, with no `town trip` line in town.
   - Why (V read, internal/bot/goal.go): in town `choresDue` builds the potion plan (healing 0 < threshold), but `emergency = healing == 0 && !p.InTown()` is false. `kitTrip` returns false in town by design. So `runGoal` applies the full `ResupplyCooldownSec` gap (240 s). `lastTrip` had been stamped by the failed field attempt a few seconds earlier (line 59: "also bounds retries when the chores keep failing"). The heal exception (line 63) needs `belowResume`, and it does not restock. The gate passes, the goal runs and the waypoint takes the character back out with 0 potions. I: one of these six town visits would have ended the window at line 220 to 222 (3 to 5 minutes).
5. The window ended by chance. The quest route passed the Dark Wood waypoint (22:29:18 `wp: activated`), and the next emergency trip took it `5 -> 1`. The bot sold, bought 4 healing, 3 mana and 2 scrolls, and banked part of the gold (`chores done: sell, repair, belt, bank, shop`).

NOKILL A (lines 241-251) is the same episode: `lowLifeGate` holds (`low life and no way to heal: waiting before retrying`, 2 min) with no kill and no XP.

## 3. BELT B and NOKILL B

- BELT B (5, passes; V run): after BELT A the bot spent the purse in three trips within 60 wall seconds (517, 150 and 100 gold). The last was a kit trip for mana (`[kit: mana potions 0/3 ..]`) that could not afford mana under the new reserve. It bought a 100-gold scroll from 113 gold, which left 13 gold. The next fight drank the 5 healing potions; then recovery, `poverty mode: on` and Blood Moor at 22-27 gold until a sell walk. I: the merc line `gold farm off need=865 gold=929` at 22:25 against 304 carried suggests gold in the stash while the bot reported poverty (U: the bank/withdraw rule was not read).
- NOKILL B (V run): an emergency kit trip at 23:12:19 (580 gold, 4 healing and 3 mana bought) went well. Then `back to field: portal click failed: d2dctl: portal click did not change the level (still 1)`, and `BackToField` (trip.go) gave up on the portal after one click. The Black Marsh waypoint was not active (`level 6 not reached (waypoint not activated?)`), so it walked Rogue Encampment, Blood Moor, Cold Plains, Stony Field, Underground Passage, Dark Wood, Black Marsh in 14 wall s (about 2.5 virtual min). It then roamed Black Marsh ground it had already cleared (`clear killed nothing; moving on`, hostileNear 0 for about 35 wall s). 11 against 10.

## 4. Were the BELTFIX changes active, and did they work?

- Binary: the run record says bot origin/main cd4c7cb built from `git archive`; the supervisor line in the minute archive starts `bot cd4c` (V run). cd4c7cba contains ce5e7f1e and 20cd22ae (V read, git log).
- Kit trip priced at one potion, plus emergency: active. There are 74 `kitgate-trip` lines; for example at 23:12:19, `kit: healing potions 0/4 ..` started with `emergency: true` and was completed (V run). In BELT A `kitTrip` correctly did not fire: Dark Wood is not walkable home, there was no charge and the waypoint was not active. That made no difference, because the older generic path (`choresDue`, `N healing potions`) has no way-home gate and fired anyway. The earlier theory (the trip is not triggered) does not apply here. The trip was triggered 60+ times and could not reach town (V run, V read).
- Mana reserve (130 gold): active and behaving as designed (V run). At 22:30:17 with 113 gold, wantMP 2 and 4 but no mana bought. It cut BELT A by nothing (750 gold at the restock) and fed BELT B indirectly: the protected 100 gold went into a scroll, leaving 13.
- Waypoint as a way home (`wpActive` in `kitTrip`): it only widens the trip trigger. It cannot help when the area's waypoint was never touched, and `waypointToTown` does not touch one. Not exercised in BELT A (U whether it was exercised elsewhere; no log line names it).
- The judge: the replay over the minute lines matches d2dsoak exactly; `beltBoxes` lists every box (empty string while empty). Not an artefact (V run).

## 5. Why 37 and not 12: what is different

- Gate 1 20261102: clvl 12, 30 to 53 gold, Blood Moor/Stony Field, no trip because of the scroll price. Here: clvl 9, 332+ gold, Dark Wood/Underground Passage on the quest route (`Dark Wood: Inifuss tree (quest)`). The trip fires but has no way home, and town visits do not shop. Different trigger, same symptom (V run).
- The character entered the Dark Wood band at clvl 9 out of mana and burned a full belt in seconds. I: the mana model (Minor Mana at 60 gold, wantMP 1 to 2) and the pack sizes in Dark Wood make "6 potions in one fight" normal there. The belt rule then depends entirely on a working way home.
- d2d backend part:
  - Re-entry in town: same in game.exe (S: D2 places a rejoining character in the act's town).
  - The portal click is dropped while busy and spends the charge (V read, the comment in d2dwire.go cites docs/fixportal.md as d2d behaviour). U whether game.exe spends a charge on a cut cast. The bot side is the same either way: it cast twice in one second during a safety drink at a cave mouth.
  - The waypoint preset is missing until the room streams in, which is d2d-specific. Here it did not matter, because the waypoint was not activated.
  - Shop stock and prices are normal: Minor Healing 30, Minor Mana 60, scroll 100 (V run). Belt placement works: `belt: put .. inBelt=true` for every potion of the 22:29 and 23:12 visits (V run).
  - I: no evidence that d2d causes either window.

## 6. Ranked fix plan (smallest correct change first)

| # | change (files, functions) | test that catches it on a replay | Gate 1 criteria |
|---|---|---|---|
| 1 | **Shop on every town arrival.** internal/bot/goal.go `choresDue`: in town, a potion plan with 0 healing potions (belt + pack) and gold for one potion is an emergency, i.e. drop `!p.InTown()` at the `N healing potions` line. Alternative: `runGoal` skips the cooldown for potion plans in town, since a visit in town costs no scroll. Keep `lastTrip` pacing for field starts only. | `TestTownArrivalRestocksWithoutCooldown`: in town (area 1), belt 0 healing, 332 gold, `lastTrip` = now-10 s, Sorceress and one other class. runGoal sets `l.trip` with `Potions`, no waypoint/task step. Fails today (V read: emergency false, gap 240 s). | BELT A (would end at one of six town visits, I: 3-5 min), NOKILL A (the wait never starts) |
| 2 | **A real way home from the act 1 fields.** internal/town/trip.go `walkToTown`: when there is no portal source and no active waypoint, walk the level chain to town (`ToArea(ctx, town)`, the call `BackToField` already uses for the opposite way; V run: the same chain took 14 wall s at 23:12:20-34). Widen `resource.walkableHome` (internal/resource/town.go) to the act 1 levels the route knows, so `kitTrip` and the resource trigger agree. | `TestToTownWalksChainFromDarkWood` with a fake mover: area 5, empty tome, waypoint inactive, gives ToArea(1) and arrival, no "none on level 5". `TestKitTripDarkWoodNoScroll`: kitTrip true. | BELT A without help from #1 (the save point is every 20 min) |
| 3 | **Do not burn the last charges in a fight.** cmd/d2bot/d2dwire.go `d2dPortalWith`: after the click a dead context returns a "cast cut, source maybe spent" error, not "nothing clicked". internal/town/trip.go `ToTown`: after a cut cast, look for the open portal (openNear) before casting again; a trip start with the safety guard drinking waits for the drink. | `TestPortalCutAfterClickReportsSpent` (cmd/d2bot, fake itemAt and a ctx cancelled in the wait loop). A ToTown test: an open portal near after a cut cast, so it is entered and not recast. | BELT A trigger (2 charges in 1 s) |
| 4 | **Leave town with a working return.** internal/town/trip.go `BackToField`: retry the town portal once (walk to it again) before the waypoint and the full walk. | `TestBackToFieldRetriesPortalOnce`: first enterPortal fails, the second leads to level 6, no ExitTo/ToArea. | NOKILL B (2.5 virtual min walk) |
| 5 | **Do not spend the reserve on a scroll during a mana-only kit trip with less than potion + scroll gold**, and withdraw stash gold before calling recovery/poverty (internal/town/shop.go `buyFirstTP`, bank chore; U: bank rule not read). | `TestManaKitTripKeepsPotionGold` (113 gold, healing 4/4, mana 0/3: no scroll bought). | BELT B (at the limit today) |
| 6 | waypointToTown touches an inactive waypoint if its position is known (needs act 1 entries in knownWaypointRel, measured on d2d; U). | nav test with a fake level map. | BELT A, lower priority than #2 |

#1 alone is a one-line condition plus a test. It removes the stuck loop as soon as any town arrival happens; the failsafe ladder guarantees one within minutes. #2 is the structural fix. #1 and #2 together are the minimum before the next Gate 1. #3 removes the trigger. #4 is the NOKILL B fix; NOKILL B was 11 against 10 and needs #4 or less roaming of cleared ground.

Live check (needs the game; not run): Gate 1 re-run on the merged commit, seed 11 plus one more seed. In bot-0.log, every `Entered Rogue Encampment` while `beltBoxes` has no Healing and gold >= 30 must be followed by `town trip` and `shop: bought .. Healing`. `waypoint to town failed` must not repeat.

## 7. Is the belt rule the right gate? (opinion)

I: yes for what it caught, but it measures a symptom. What must be correct is "a character with no healing potion can get to a vendor, and does when it is there". BELT A is exactly that defect: 37 minutes at hpmin 19 to 25 percent with 300+ gold, six town visits without a purchase. 0 deaths was luck, not design (V run: hpmin 19 at line 239). BELT B and the first Gate 1 window are poverty: the bot cannot buy what it cannot pay for, and the rule then measures the gold model, not the restock logic. A sharper gate would count only stretches with no healing AND gold for one potion (the d2dsoak minute line already has `gold`), plus separately "town arrival with no healing and gold, left without buying" = 0. That would have failed this run on the right reason and passed the poverty minutes, which the economy gates already judge. This is an opinion, not a spec change; the soak-gates spec is unchanged.
