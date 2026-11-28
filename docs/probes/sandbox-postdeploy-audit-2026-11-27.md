# Sandbox post-deploy audit, 2026-11-27 (Doofer and Questy, builds 0.0.261127225902 and 0.0.261127225945)

> status: research, read-only (card postdeploy). Window: deploy 17:59 EST (22:59Z) to 19:09 EST (00:09Z next day), 70 min of play; times below are EST unless a `Z` is shown. Target: classic 1.14d offline sandboxes in the local docker. Only `d2bot ops log --tail`, `ops status --why`, `ops quest status [--json] [--why]` were run: nothing written, restarted or flipped, no token printed.
>
> Tags: **V** read in the bot's own log in this window; **S** arithmetic on V lines; **U** unproven (cause or intent). Cause classes A to K: `docs/roadmap-fable.md` ("Operating rule"). Compared with `docs/probes/sandbox-log-analysis-2026-11-27.md` (the "earlier doc").

Log files: V 18.6k lines (Doofer) and 17.5k lines (Questy) from `d2bot start` at 17:59:15 and 17:59:59 (the first line of each is the new version). The `d2bot ops log` tail is the bot's own structured log, so the `inv=`/`bot.tickLogSec` gap of the earlier doc is closed: `tick` (every ~3 min, with `facts`) and `minute` lines exist.

## 1. Health

| | Doofer (Paladin clvl 17, release ...225902) | Questy (Paladin clvl 16, ...225945) |
|---|---|---|
| errors, panics | 0 ERROR, 0 panic lines (V) | 0 ERROR, 0 panic (V) |
| deaths, chickens | 0 deaths (`deaths=0` on every minute line), 0 chickens (V) | 0 deaths, 0 chickens (V) |
| `Save and Exit` lines | 4 (23:28:02, 23:28:04, 00:00:16 + one pair) | 2 (23:18:21 + one) |
| what they are | V: one `xp at risk why="save point"` at 23:28:02, plus 2 `new game requested from farmreset.go:291` (23:28:04, 00:00:16); NOT chickens | V: `Resetting the zone: 85% cleared, going home for a fresh game` at 23:18:21, `farmreset.go:291` |
| d2d relaunches (no `relaunch reason=` line) | S 4 `d2d channel not answering ... socket file is gone` bursts after a Save and Exit, plus the two at start | S 2 bursts after the reset, plus the start |
| hero | xp 308056 (23:00Z) to 335509 (00:08Z), kills 12371 to 13230 | xp 248855 to 255422, kills 18785 to 19547 |
| xp/h | S 24.2k/h (earlier doc: 21.6k to 38.7k/h, 5 h) | S 5.8k/h (earlier doc: 5.0k to 6.0k/h) |
| potions used | 30 | 34 |
| inv | `invPct` 17 to 27 % (V `minute`) | 22 to 30 % |
| `arbiter-starved` invariant | 2 (23:10:33, 23:42:05) | 3 (23:10:28, 23:30:00, 00:06:46) |

Reading: no regression in survival (0 deaths in 70 min on both; Questy had 24 deaths in its run before). Questy's xp rate did not change with the deploy (problem 1). The `Save and Exit (chicken / leave game)` WARN text is the same line for a real chicken and for a farm reset (`farmreset.go:291`), so a log reader or a counter that greps it over-counts chickens by every reset (problem 6).

XP by area (S, `area summary` minutes and xp, window): Doofer Tamoe Highland 39 min 11.9k (18k/h), Jail Level 2 (27) 8 min 6.7k (50k/h), Tower Cellar 3 6 min 4.5k (45k/h), Tower Cellar 2 5 min 2.7k, Black Marsh 4 min 1.5k. Questy Dark Wood 35 min 4.5k (7.8k/h), Cave Level 1 14 min 1.1k (4.7k/h), Cold Plains 3 min.

## 2. Features deployed today

| feature | verdict | evidence |
|---|---|---|
| admission shadow | works, never differs | V `ops status`: Doofer agree 673 / differ 0, Questy 753 / 0, mode `shadow` (Doofer was 306/0 at 18:34 and 15/0 earlier). Counts grow by about 5 per minute. |
| akarafix hand-in (`hand-in queued/served/dropped`) | not exercised | V 0 lines on both. Doofer's `search_for_cain/free_cain` is `ready` (need=legacy_only), so the first real hand-in has not come yet; Questy has no Cain chunk running. Cannot be called working or broken. |
| quest board, Questy | the Cain chain is NOT progressing | V `quest status --why`: `den_of_evil` done; `sisters_burial_grounds/blood_raven` ready (word=safe floor=7 need=no_merc), `search_for_cain/scroll` ready (floor=11 no_merc), `forgotten_tower/countess` ready, `tools_of_the_trade` ready, the rest locked. No chunk started in 70 min, no `paused` or park reason, no `last-refused`. `need=no_merc` is a reason beside a ready word (readiness.go `reasons`), not a refusal; why nothing starts is U (auto-start off for these chunks is the likely reason: on Doofer the earlier doc found every chunk started by the player). The hero instead grinds `Quest: fighting (Level up to 17)` in Dark Wood for 68 of 68 minute lines. |
| quest board, Doofer | one park, same cause as before | `forgotten_tower/countess`: park cleared after release e19701a to 7f4bec7 at 22:59:21 ("retrying once"), started 23:01:41, `gave_up after 3 tries` at 23:05:25, state `blocked code=gave_up` (V). Cause: section 3. |
| `paused` reason code | none seen | V: no `paused` in either board; the codes seen are `gave_up` and `need=not_ready wake="any of level, kit, gold or area changes"` (`last-refused=23:34:49Z`, countess). |
| fieldstate hand-over (`handing over to the fighter`) | not seen | V 0 lines; the line exists in `internal/bot/loop.go:447` (`back to field: hostiles near on the way, handing over to the fighter`). Only one `back to field: life below the resume line in town, ending the trip` on Doofer. Not exercised in the window. |
| town7 hand-over text, short-gold table | not seen | V 0 matching lines. `town plan (shadow)` lines: 14, all `verdict=ready needs healing 2/8,portal 1/1`, `would buy nothing`, except one at 23:34:20 `has healing 3 ... would buy healing x5(150)`; the real chores then did `[kit: healing potions 3/4]` (agrees). |
| advisor lines | none | V 0 (knob off, as expected). |
| `combat aim/stance/minion summary` | not deployed | V 0 lines; `combattelem` is merged on main (162e07854) but both builds predate it. The existing `combat fight.tick` / `combat health` lines are there. |
| `inv=` and tick lines | tick and minute lines work | V `tick` carries `chain`, `facts` (potions, plan_buy, stash_gold, ...), `minute` carries `invPct`, `beltBoxes`, `dmgTaken`. There is no literal `inv=` line (the earlier doc named it; it is `invPct`/`inv-tp-dump`). |
| arbiter shadow | diverges by design | V Doofer 2 and Questy 4 lines; all say chain chose chores/level, arbiter would pick `quest/step` because `level/sweep: class filler below objective` (one on Questy the other way: arbiter would pick level/sweep because `quest/step: travel budget: going to Dark Wood undoes Dark Wood->town 0s ago`). The same pair underlies the `arbiter-starved` invariant. |

### Planner shadow (TASK-984 AC1)

V 70 lines on each sandbox (every 60 s), all identical per sandbox after masking the level:

| sandbox | line (masked) | distinct picks |
|---|---|---|
| Doofer | `plan=Zeal/4 profile=Zeal agree=true best=Zeal aura=Might auraProfile=Might arch=bruiser unplayed=0 warnings=mana_starved,tourist via=act1:too_hard` | 1 |
| Questy | same, but `auraProfile=Holy Fire` against `aura=Might` | 1, with one disagreement |

Review (S/U): the skill pick Zeal agrees with the profile on every line, so no wrong skill pick. Two points to flag. (a) Questy's aura pick (planner Might, profile Holy Fire) disagrees on every line while `agree=true`: the `agree` flag ignores the aura field, so the shadow hides the one difference it measures. (b) The planner carries `warnings=mana_starved,tourist` on every line for both heroes: `mana_starved` is plausible (maxMana 37 to 46 in `minute`), `tourist` is U (meaning not read here). Doofer's aura (Might both) agrees. `via=act1:too_hard` is the board's word, matching `sisters_to_the_slaughter` too_hard.

## 3. "no path" report

V Doofer only, at the Forgotten Tower chunk, `Tower Cellar Level 3`: `exit to Tower Cellar Level 4: nav: no path: gate to level 24 is on the map but not reachable from {45 20} (cut off, no way out found)`, repeated from `{42 20}` (3 tries, 23:04:35 to 23:05:10Z), preceded by `nav: cut off from the gate, stepping to the nearest connected cell` (23:04:27Z) and `the gate is cut off, walking to the nearest reachable cell to look for its door` (23:03:59Z). The board retried the chunk once after the release cleared the park (22:59:21) and gave up again at 23:05:25. This is the lines the coordinator's "no path" report most likely means; the earlier doc could not read them. It is NOT a "no path to Dark Wood": 0 lines mention Dark Wood and `no path` on Questy (V). Dark Wood is reachable (Questy stands there all window). Cause class D (navigation geometry): the exit is cut off by a door or blocked cells in Tower Cellar 3 and the door search found nothing. Similar siblings in the window (S): `leveling area unreachable exit to Barracks: nav: no exit to that level: level 28 from 27` (23:01:41, twice) and `the level changed during the walk: 27 -> 1` after `stalled in a pocket, escaping by town portal` (23:31:28); a `Outer Cloister` walk ended on another level. 4 `area skipped unreachable` warnings.

Also seen at 23:29:35 `the gate is cut off by a door, opening it` (twice): the door handling does work on some maps.

## 4. Problems not in the earlier doc, ranked

1. **Questy farms Dark Wood at 5.8k xp/h while the hero stands past that window** (class G/A; xp/h flat versus before). V `level plan: clvl 16 -> area 5 Dark Wood (past every window, last eligible (leaves at 11))` and `zone choice: Dark Wood margin 2.0 ... xp/min 88`, and 35 of 70 minutes there (the other areas Cave Level 1 14 min at 4.7k/h). `quest board` meters say `Quest: fighting (Level up to 17)`. Doofer at the same clvl runs 3 to 8 times faster (Jail 2 50k/h). Cause is U: the quest board's level gap plus no merc (route-guard floors +1, `merc=none`) keeps the hero on the safest zone; the zone picker takes "best pay among the safe zones". Card `questyzone`: owns `internal/bot/zonepick*.go` (read first), AC: with no merc and the hero 1 level under the quest floor, the picker still lists the first window above the legacy one and the shadow prints why it did not choose it; offline scenario.
2. **Questy runs without a merc and nothing tries to get one** (class C/H). V `merc policy on (merc.enabled=1)` at start, `route guard ... (merc=none)` on every board call, 0 hire lines, 3 visits to the Rogue Encampment; the Act 1 Rogue is free after Blood Raven (`ShouldHire`: `ReasonFreePending`) and Blood Raven is `ready` but not started. Doofer has a merc (died 23:05, revived for 1680 gold at 23:27 after 22 min dead although the stash holds 98.5k gold). Card `mercfree`: the board starts Blood Raven (or the hero hires the paid rogue) when the hero is above the floor and has no merc; the dead-merc revive waits no longer than the next town visit when gold is above the quote (V `merc revive: decision` 7 times 23:05 to 23:27).
3. **Aura thrash on Questy** (class A). V 95 `combat aura up skill=Holy Fire` and 97 `skill=Might` in 70 min, strictly alternating every 3 to 10 s (23:00:35 Holy Fire, 23:00:39 Might, 23:00:51 Holy Fire, ...); Doofer 2 and 2 (Might, Prayer). The planner shadow names the cause: aura plan Might, profile Holy Fire. Each swap spends a cast and (U) mana; `combat health` on Questy shows `castsPerMin` 2.6 to 31 and `killsPerMin` 1.6 to 35 in the same area. Card `aurathrash`: one decider for the standing aura of a Zeal paladin (the planner's or the profile's), a test that the same fight never swaps twice; the shadow `agree` field includes the aura.
4. **Doofer: second chunk park for the same unreachable gate** (class D/B). The retry after the release spent all 3 tries in 4 minutes on the same cut-off cell. Card `parkreason`: a park whose cause is `no path` retries only after the area's map data or the hero's position class changes (or after N hours), with the reason printed on the board; and the Tower Cellar 3 to 4 fixture ({42 20} and {45 20}, door search) as a nav fixture.
5. **Doofer 98.5k stash gold unused, kit short** (class C, U). `tick` facts `stash_gold=98514`, `potions=3`, `plan_buy=5`, `pack_gold=1150`; chores `[kit: healing potions 3/4]` and `[hp belt columns below 50%]`, belt shows 2 Light mana and 7 Minor Healing potions in slots at clvl 17. Whether higher potions are in stock is U. Card `goldsink`: a town plan stage 5 row that spends stash gold above a floor on the potion tier of the vendor.
6. **Misleading `Save and Exit requested (chicken / leave game)`** (class K). V the same WARN line for a farm reset (`new game requested from farmreset.go:291`), a save point and real chickens; plus two `d2d channel not answering` WARNs in the 2 s after each leave although the d2d process is `still running` (V pid unchanged, then a new pid after the re-enter). Card `leavereason`: the reason in the line (`reset|chicken|savepoint`) and the `relaunch reason=` the earlier work wanted.
7. **`arbiter-starved` invariant is a standing fact, not a fault** (class K). V 5 firings, always `level/sweep ... outranked by quest/step`, once with `quest board: forgotten_tower/countess gave_up`. While the board has a step the level family is starved by design; the invariant needs an exemption or a count of "starved while the starver made progress" (arbitration). Card `starvedquiet`.
8. **Planner shadow `agree` ignores the aura** (class K), see section 2. Card `plannershadow-aura` (a few lines in the shadow).
9. **Doofer Jail 2 (27) pocket stall** (class D): `stalled in a pocket, escaping by town portal` 23:31:28, leaving the 50k/h zone. Fixture card `jail2pocket` with the map hash from `d2dworld: levelmap level 27` (not read here).
10. **`reflex: the cancelled action did not return in time`** (Questy 23:02:10, class A, one line, `combat fight ... end=error remaining=8 life=56` at 23:02:07): the guard did the safety action itself; the cancelled action took more than its budget. No card yet, watch for a repeat (V 1 line).

Not a problem now (checked): 0 errors, 0 deaths, admission 0 differ, `town plan (shadow)` agrees with the real chores in the one case that bought.

## 5. Commands run

All read only, on the two containers: `d2bot ops log --tail 200000` and `--tail 400000` (redirected to a local temp file and masked with `jq`), `d2bot ops status --why`, `d2bot ops quest status [--json] [--why]`. Temp files stay out of the repo.
