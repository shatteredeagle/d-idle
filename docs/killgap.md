# KILLGAP: where the seconds per kill go on d2d (2026-10-30)

> status: current. Tags: V = measured by this change on hidden d2d (fresh uncheated Sorceress, 4 seeds x 2 virtual hours, `tools/d2dloop/seeds.sh`), U = inferred. Nothing was run on Game.exe.

## The instrument (V by unit test and by d2d runs)

`killtime` log line per kill (internal/bot/killtime.go): `monster area total walk cast wait gap_since_last_kill state`.
Buckets (seconds between two kills, in-game time only): walk = player moving (mode walk/run, or any position change between two snapshots; the mode alone gave walk=0 because nav's blocking MoveTo loops do not feed the observer), cast = attack/cast mode, wait = everything else (idle, search, panels, potions, town chores, link waits). `d2dsoak extract LOG...` appends per log: s/kill by bucket, a gap histogram, s/gap by bot step, the top 5 gaps with the messages logged between the two kills.

## Baseline (V, binary kg0..kg2 = master + instrument, seeds 1-4, 2 h)

XP/h 3640 (3233..4162), kills/h 156 (135..176), s/kill 23 (walk 17, wait 5.3, cast 0.8). The cast bucket is 3 %: casting is not the gap. Walking is about 75 % of the time between kills; wait is 20-25 %, two thirds of the wait seconds in gaps over 30 s.

| gap between kills | kills | seconds per kill contributed |
|---|---|---|
| 4 s or less | 40 % | 0.7 |
| 4-10 s | 20 % | 1.5 |
| 10-30 s | 21 % | 3.7 |
| 30-120 s | 10 % | 6.5 |
| over 120 s | 4 % | 10-14 |

By step (share of all gap seconds): `Recovery: farming gold` 67-71 % (19-27 s per kill), `Quest: fighting` 20-24 % (17-24 s), `Returning to the field` 5-11 % (49-216 s per kill: 2 to 12 kills per run, each after a town trip).

## Causes in the 20 longest gaps (4 seeds x top 5, 200-600 s each, walk is 65-90 % of them) (V)

1. Empty-area roaming: 7 of 20. `clear killed nothing; moving on` x2-12, `roam: coverage points`; the bot walks the whole of Blood Moor / Cave Level 1 for 200-500 s for one kill. The area has no monsters left in reach (U: no respawn inside a game; kills per 30 min fall 103 -> 60).
2. Area-border shuttle at cave mouths (Den of Evil, Cave Level 1 <-> Cold Plains): 6 of 20; `Entered X` x3-16, `nav exit` x5-8, `nav: stepping away from a cave mouth` x10. 22-36 unwanted crossings per run; the border-loop breaker (10 crossings per minute) then bans the area for 3 min and produces 2 more gap kinds: `target skipped: the route crosses a banned area` x8-10 and `area gate refused the walk` (3 of 20).
3. Crowd retreat plus nav stuck at low life with no potion: 1 of 20 (548 s: nav stuck x32, crowd stepping away x28, move failed x28).
4. Town trips and returning to the field (poverty recovery, no scroll): `Returning to the field` and `npc visit` gaps 100-200 s (3 of 20).
5. Not seen as causes: link waits (the 1 m link wait is logged but never in the top 20), path planning slowness, potion/mana waits (wait bucket is 5 s per kill), loot pickup walks.

## Fix tried and reverted (V)

Explore frontiers within 6 subtiles of an unwanted tile exit were skipped (internal/nav/explore.go, commit b835dd4, reverted afterwards). Same seeds 1-3 (seed 4 not finished): 130 kills/h and 2986 XP/h vs baseline 156 and 3640 (seeds 1-3 baseline: 164 and 3745), s/kill 27 vs 22.8; unwanted border crossings 22/20/47 vs 30/36/22. No improvement, within run noise or worse, so the frontier is not the source of the shuttle (flee points and Clear approaches also walk into mouths). Reverted: no Game.exe-affecting change remains.

## Not fixed, ranked next

1. Empty-area roaming (cause 1): needs a policy decision (leave a cleared area after N zero-kill minutes, or reset the game for a fresh spawn as the original's restarts do, U). Not done: it changes what the bot farms.
2. Flee point choice (`combat abort fight: flee to ...` led into the Den mouth in seed 1): the flee target should avoid exit tiles (internal/combat, shared, untested here).
3. A faster border-loop breaker (3 crossings instead of 10) would cut the shuttle gaps but also bans the farming area more often.

## Caveats

d2d truncates spell damage to whole points (UPSTREAMGAPS), so kill speed at low skill is slightly pessimistic. Density is no longer thought to be a factor (UPSTREAMGAPS: placement equals Game.exe). Runs are not deterministic between binaries (same seed gives different kills), so the 4-seed means carry about +-10 % noise (sd of kills/h 21 on 156).
