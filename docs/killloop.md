# killloop: where the seconds per kill go on the sandbox Paladin (2026-11-09 log)

> status: analysis, no code change. Tags: V = measured from the sandbox log (11.7 h, 2201 `killtime` lines, read-only copy on the worker host), S = read in code, U = unverified. Follows `docs/killgap.md` (instrument) and `docs/research/beta-readiness-review.md` fun killer 3.

## Result in one paragraph

The "12 of 18 s waiting" is not a fixed sleep. It is a tail: 7 percent of kills (gap over 60 s, 164 kills) hold 12.5 of the 18.3 s per kill and 8.5 of the 11.9 s of "wait". The median kill takes 2.8 s (wait 1.2 s); 65 percent of kills are under 5 s. Silent gaps between log lines add up to only 3.9 s per kill and no single fixed sleep exceeds 0.3 s per kill (V, consecutive-line gap scan). The wait bucket (`internal/bot/killtime.go`) is "not walking and not casting", so it also counts nav stalls, town chores and a fight that cannot act.

## Ranked by seconds per kill (V; class = what the long-gap window contains, first match wins; total / wait per kill)

| rank | what the window shows | kills | total s | wait s |
|---|---|---|---|---|
| 1 | `out of mana` (back-off, cap melee, waiting) | 120 | 4.68 | 3.14 |
| 2 | town trip, shop, walking back to town (no portal) | 68 | 4.54 | 3.16 |
| 3 | all gaps under 15 s | 1831 | 2.90 | 1.53 |
| 4 | `nav stuck` / `stalled by monsters` | 53 | 2.24 | 1.39 |
| 5 | long gap with only nav lines (roam) | 77 | 1.48 | 0.87 |
| 6 | `left skill not selectable` (basic Attack not bindable) | 14 | 0.80 | 0.65 |
| 7 | `watchdog cannot cast` | 9 | 0.51 | 0.40 |
| 8 | cave-mouth swing hold, zone entry, flee/dodge | 29 | 1.15 | 0.77 |

Ranks 1, 6 and 7 are one defect chain: a Paladin at 28 maximum mana runs dry, the basic Attack could not be selected (`skill 0 has no left popup cell`, 85 and 167 lines in the first two hours, 0 after), so it backed off instead of swinging (1285 `backing off` lines in the window). Together 5.99 s of 18.3 s per kill (33 percent). Two fixes are on main and were NOT in the sandbox binary: TASK-210 (`bindleft.go`, bind line gone from hour 15 on, V) and `dd208fa9` packfight (`resource.AttackChoice`: a melee-profile class swings when out of mana at any pack size below danger 3; the log still shows `backing off` at 107 to 226 an hour to the end, so the running binary lacks it; test `TestMeleeClassSwingsInBigPackWithoutMana`, Paladin; Sorceress still retreats, same test). Paladin and Barbarian carry a `melee` block in `classdata/profiles.json`, so the rule is data, not a class check (S).

Rank 2 is economy, not waiting: 53 poverty flips, no town portal, "walking back via the neighbour". Not touched here.

## Fixed sleeps checked (S, `internal/combat`)

`PollMs` 60, `ReactionMs` 120, `CastGapMs` 650 (cast animation, kept), `LowManaWaitMs` 700 (only reached by non-melee classes now), potion gap 1000 / rejuv 300 (safety, kept), 150 to 500 ms binds and aura settles. None shows as a top silent gap. The longest silent gaps are a held swing into a cave mouth (`swing held`, 118 times, 4 s each, 0.21 s per kill) and a snapshot summary line; not worth a change.

## Low-value areas (V, `minute` lines, xp delta per minute by area)

| area | minutes | xp/h | kills/min | xp/kill |
|---|---|---|---|---|
| Blood Moor (2) | 282 | 932 | 4.61 | 3.4 |
| Stony Field (4) | 171 | 5869 | 3.10 | 31.6 |
| Rogue Encampment (1, town-side fights) | 107 | 3043 | 1.84 | 27.5 |
| Cold Plains (3) | 89 | 903 | 1.25 | 12.1 |

The hero spent 41 percent of the minutes in Blood Moor at a sixth of Stony Field's xp/h at clvl 10 to 12: the fastest kill rate there, the slowest return. This is routing; left to the area-thrash agent (not retuned here). A routing rule on xp/kill per area (leave when xp/h falls under a fraction of the best known area) would be the lever.

## Against d2d pacing (S, `docs/soaks/classpacing2.md`, `docs/walksmooth.md`)

Lockstep soaks reach clvl 5 to 6 at 60 to 90 virtual minutes with kills in the same range per hour; the sandbox 184 kills/h equals killgap's 156/h baseline, and there the cast bucket was 3 percent too. Walksmooth removed the 55-of-240-s standing in the same out-of-mana case (V live there).

## Estimate (U, no live run here)

If ranks 1, 6, 7 collapse to the 2.9 s of an ordinary gap, 18.3 s per kill falls to about 12.5 s: 3.3 to about 4.8 kills a minute, upper bound, about +45 percent kills per hour; xp follows the area mix. Verify after the next deploy: `grep -c 'out of mana: backing off'` per hour should drop to near 0 for the Paladin and the `killtime` mean to under 13 s.
