# Paladin aggression in big packs (2026-11-11)

User request: "if we have any knobs on the paladin aggression, turn them up a bit; it is still being weird in large groups of enemies."
Tags: V = counted in the container log, S = secondary (code reading), U = unverified.

## Evidence and its limit

- V: the sandbox container was recreated at 14:23 UTC on 2026-11-11, and `docker logs` of the old container is gone. The six-hour window the brief asked for does not exist; the only evidence is the first 8 minutes of the new container (character Doofer, clvl 11, d2d backend, build 0.0.261111142718). That is a small sample: treat every conclusion as U for the long run.
- Backups: `~/play-backup-20261108/tuning-pre-aggression.json` (the pre-change `tuning.json`, tuning keys only).

### Fights with maxHostiles >= 4 before the change (14:24:00 to 14:32:10 UTC)

| metric | value |
|---|---|
| big-pack fights | 19 (maxHostiles 4 to 18, median 7) |
| kills in them | 57 in 169 s (about 20 per minute, 44 per minute in the fast ones) |
| attacks / moves / approaches | 88 / 45 / 27 |
| backoffs, kites | 0, 0 |
| crowd, portal, drink, abandon, "backing off", "stepping away" lines | 0 |
| lowest life in any fight | 53 percent (one potion used) |
| deaths | 0 (summary `deaths 0`) |
| `lastNoCast` | `wait: cast gap` 11x, `approach: no shot` 7x, `dodge: missile on top of us` 1x |

## Findings

- V: the behaviours the earlier packfight cards fixed (back-off at healthy life, empty attack list) do not occur in this window: zero `backing off`, zero kites, zero crowd events, kills at 40+ per minute in packs.
- V: what remains in big packs is `approach: no shot` (the hero walks to a target it cannot hit yet; 27 approach steps in 19 fights) and the cast gap wait. Neither is a knob problem.
- S (fight.go, "Step back from ONE melee monster"): the kite rule only fires for a single melee threat within 3 subtiles at life below 60 percent and falling; it never fires inside a pack. `combat.kiteLifePct` and `combat.kiteMaxSteps` therefore barely touch pack behaviour. Lowering them is harmless and slightly more aggressive vs single hard hitters, but cannot explain "weird in large groups".
- S (crowd.go): the crowd rule is the only knob-driven pack logic: it drinks at `crowdDrinkPct`, opens a portal at `crowdPortalPct`, abandons a walk at `crowdAbandonN`. These are what "weird in large groups" most plausibly means (the hero stops a clear to drink, portal or walk away from a crowd while winning). The window had no crowd of 10 or more at low life, so the rule was not exercised (U).
- U: poverty mode was on at the start (gold 111, no portal scroll, 2 potions); it ends the crowd portal option until a scroll drops.

## Knob change (live tuning, sandbox container)

Edited `/data/instances/main/state/tuning.json` (the tuner's persisted overrides, `docs/api.md` "Tuning routes"), tuning keys only, then restarted the bot (SIGTERM to d2bot; the container restarted, healthy again after 20 s). Ranges checked against `docs/knobs.md`.

| knob | before (player preset) | after |
|---|---|---|
| combat.crowdN | 8 | 10 |
| combat.crowdDrinkPct | 65 | 58 |
| combat.crowdPortalPct | 70 | 60 |
| combat.crowdAbandonN | 18 (override) | 22 |
| combat.kiteLifePct | 70 (override) | 50 |
| combat.kiteMaxSteps | 6 | 3 |
| combat.chickenPct | 22 (override) | 22 unchanged |
| combat.runnerGuard | 1 | 1 unchanged |
| combat.maxApproaches | 18 (override) | 18 unchanged |

Less aggressive than the brief's candidate on purpose: `crowdN` 10 not 12, `crowdDrinkPct` 58 not 55, `crowdAbandonN` 22 not 20 (the live override was already 18), because the 8-minute window gave no crowd event to justify the full bump. Predicted effect (U): fewer drink, portal and abandon interruptions in packs of 8 to 10; kills per minute in dense packs a bit higher; potion use per pack lower, damage taken per pack a bit higher. Risk: at low clvl with poverty (no portal scroll, 2 potions) a pack of 10 or more at 50 to 58 percent life is now fought on, with the chicken rule (22 percent) as the only net below the drink line.

Rollback: copy `~/play-backup-20261108/tuning-pre-aggression.json` back to the instance `state/tuning.json` and restart the bot, or `PUT knobs` with the five keys set to null for crowd* and kite* (null resets to the profile value; the two old overrides were `crowdAbandonN` 18 and `kiteLifePct` 70).

## Results of the knob change (V, same container log)

| window (UTC) | big fights | kills in them | sec | backoffs/kites | minlife | potions | deaths |
|---|---|---|---|---|---|---|---|
| before 14:24:00-14:32:10 | 19 | 57 | 169 | 0/0 | 53 | 1 | 0 |
| after 14:32:30-14:47 | 32 | 65 | 246 | 0/0 | 62 | 0 | 0 |

Kills per big-fight second 0.34 before, 0.26 after: no gain, and the sample is too small to say more (U). One `crowd: stepping away` at crowd 26 and life 60 (14:38) after the change. No deaths, no chicken. Knobs kept; the knob bump is secondary, see below.

## Detection audit (user observation: runs away at full life, runs back in; may not see revived Fallen)

Checked in code (S) and in the log (V):

- Alive test: `d2.Unit.Alive()` is only mode 0 (death) and 12 (dead); every other mode, including a unit that is being resurrected (D2 sequence mode, U: the mode a shaman revive reports was never read live), counts as alive and hostile in `combat.Hostiles` and `bot.hostile`. A revived Fallen is not filtered by mode, and it keeps its unit id and class. Not a cause (U live).
- Friendly test: revived monsters could be called minions through child state 105 (`friendly.go` says "U: not seen live"); the live log shows Fallen being killed after shaman revives with no such skip, so no evidence for it (U).
- ROOT CAUSE found (V): `sweepPass` (internal/bot/goal.go) calls `Clear` when `hostileNear` (previous snapshot) is above 0. When the last hostile died in between, Clear saw nothing: `fight.tick ticks=0 maxHostiles=0`, killed 0, `remaining 0`. The sweep treated that as futile and for 90 s or 40 subtiles ignored EVERY hostile in its radius (`hostile()` returns false). 20 such `clear killed nothing; moving on ... remaining 0` lines in 23 minutes, nearly one after every pack. In that window a fresh pack, a shaman-revived Fallen or a fleer that comes back is not engaged: the hero explores on at full life, then `nav: followed, turning to fight` (19 times in 8 minutes) when it is attacked. That is the "runs away at full life then runs back in".
- Not a cause here (V): zero backoffs, zero kites in 51 big fights; one `ignoring target` (a Fallen Shaman, unreachable, no path and no firing position, dist 11): the 90 s ignore memory is per unit id, so a shaman behind a wall is skipped for 90 to 360 s (U effect; left as is).

## Fix (rule, class-generic)

`futileFor` (90 s) now applies only when Clear set something aside (`Remaining` above 0, i.e. skipped or avoided units: really unreachable or immune). When Remaining is 0 the window is `futileGoneFor` (3 s). Test `TestGoneTargetClearOnlyShortlySuppressesHostiles` (Sorceress and Paladin; fails with the old window: second Clear came 90 s later). Not live verified.
