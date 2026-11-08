# Paladin smoke: the fun-loop slices live on a Paladin (TASK-142)

> status: run 2026-11-07 (EST 22:18 to 23:20), worker host, hidden lab slot and one container. Tags: V verified live (how stated), S read in code or doc only, U unverified. Short result: the slices run and answer on a Paladin, but the Paladin hardly plays: no kill in lockstep, 13 kills in 20 realtime minutes.

## What ran
| | Run A: lockstep | Run B: container realtime |
|---|---|---|
| Bot | `d2bot play --backend d2d --managed --d2d-class 2 --char Pally-pa3 --seed 7 --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --goals campaign` (`docs/probes/paladinsmoke/run-a.sh pa3 7 80m`), main at `261f4ec6` (queue #348 merged, TASK-142 present) | image `d2dbot:palsmoke` built from the public fork tag by `context.sh` (the recipe of `docs/tester-live.md`, with the legacy builder: the worker has no buildx), `D2D_MODE=realtime`, `D2D_CLASS=2`, `campaign:normal_act1`, 3 GiB, 2 cores |
| d2d | fork `lab3-labverbs` f75816c6 binary, lab verbs OFF (no `--lab-verbs`), slot 1, nice 10 | the same commit, built in the image, no lab verbs |
| Length | 36 wall minutes = 707 virtual minutes (707 `minute` lines) | 20 real minutes sampled (57 samples every 20 s), container up 21 minutes |
| Writes after start | 0 by construction: no `--lab-verbs`, no `d2cheat` run, `play` writes nothing to the game (V: the d2d answers every setup verb `err lab verbs off`, see `docs/tester-live.md`; not a gate run, no `runstart`) | 0, same reasons |
| Character | Pally-pa3 created by d2d `--start-class 2` (UI order, V in `docs/d2d-fidelity.md`), logged in `docs/probes/test-characters.md` | Palsmoke, same path |

Plan the bot picked (V, log line `class-aware start`): class Paladin, combat profile `paladin`, leveler build `paladin-normal30-v2` (not `normal30` and not `zealot-lite`); the log also prints `combatBuild: blizzard`, which is the generic default label and looks wrong for a Paladin (U: only a label, the profile is `paladin`).

Container class pick (V): the setup wizard refuses the Paladin: `PUT /api/inputs` with `class: paladin` answers 400 `Paladin is not supported in this build`, and `/api/state` lists only the Sorceress as supported (TASK-145). The run therefore wrote `/data/wizard.json` (phase done) and `/data/wizard.env` (`D2D_CLASS='2'`) into the data volume by hand before the first start; the entrypoint then took the same path as after a real wizard. No memory write was involved.

## Pace (AC 2)
Sorceress reference: clvl 4 at about virtual minute 49, clvl 5 at about 80, clvl 6 at about 110 (`docs/probes/shortdrills.md`, drill 1, V on that run).

| Virtual minute | Sorceress (drill 1) | Paladin lockstep (V) | Paladin realtime container (V, real minutes) |
|---|---|---|---|
| 10 | not recorded (clvl 1 to 2 in Blood Moor) | clvl 1, 0 XP, 0 kills | clvl 1, 228 XP, 10 kills (about minute 7; 99 XP, 3 kills at minute 5) |
| 20 | not recorded (between clvl 2 and 4) | clvl 1, 0 kills | clvl 1, 315 XP, 13 kills at the end |
| 49 | clvl 4 (about) | clvl 1, 0 kills | not run |
| 60 | clvl 4 to 5 (interpolated, U) | clvl 1, 0 kills | not run |
| 707 | | clvl 1, 0 XP, 0 kills, 0 deaths | |

The class-early-game expectation (clvl 5 to 6 in about 20 minutes, 1.5 to 2x the Sorceress, `docs/research/class-early-game.md`, UNSURE) is not met in either mode. Corpsefire (Den of Evil) was not reached by play. After a paid charter order the lockstep hero walked to Cold Plains and the Den of Evil at clvl 1 without killing anything (Codex `den_of_evil` at virtual second 14592).

Why (V for the symptoms, U for the cause): in lockstep the log says `combat no bound attack skill, swinging the basic attack` (known skills 0: a Paladin has no skill point at clvl 1), then `combat ignoring target ... casts 15 life 100% engaged 22s` and `12 approaches, 10 subtiles away`; the kill-drought rule then leaves the game about every 3 virtual minutes (64 re-entries in the first 15 wall minutes, `restarts` field of the minute line). The same bot and class in realtime killed 13 monsters, so the failure is lockstep-specific or much worse there (TASK-175, hypothesis in the card). In realtime the hero idles for minutes in Blood Moor in the recovery step (TASK-177).

## Slices (what worked, what is empty)
| Slice | Result |
|---|---|
| Return screen | V (curl, both runs): `/api/return` and `/watch/return` answer from the first minute with real events: headline "Away 53 min. Your Paladin is level 1 in Rogue Encampment. 37 expeditions, 0 deaths.", trips list, a "Closest call" moment from a real event, sections `proposals` (placeholder), `tally`, `codex`, `policy` (the change I made, with a "Change it back" button). Gaps named on the page: gold, waypoints lit and supply cost per trip have no event yet. The landing redirect: V that `/watch` answers 302 to `/watch/return` 20 real minutes after the last look and keeps doing so until `/watch?live=1` (the doc says once; TASK-178). Empty: gold, drops, quests, XP (nothing killed) |
| Tally strip and charter | V: `/api/tally` works without the overlay (mode `events`): strip text, balance, recent credits, `next` list. Charter: an order above the zone range without `approve` answers 402 `charter-needed` with the hero's warning (Stony Field at clvl 1: 80, Cold Plains: 40); with `approve` and too little balance 402 `insufficient`; with `approve` and enough balance 200 and the balance fell 45 to 5 (40 charged). An in-range order (Den of Evil) is free, 200. Defect: every expedition credits +2 even with 0 kills and 0 XP (TASK-176) |
| Codex firsts | V: 4 of 37 in lockstep (`blood_moor` 56 s, `cold_plains` 3449 s, `wp_cold_plains` 3545 s, `den_of_evil` 14592 s; source `quiet`/`default`, 2 asked and left to default), 3 in the container (`blood_moor`, `quill_rat` at 320 s, one more). Missing as expected: shrine, champion mods, Corpsefire, Fallen (no kills). `/api/codex` lists the 33 missing entries by name |
| Policy sheet | V: `GET /api/policy` lists the chores; `POST /api/policy {row: repair, option: never}` answered `{ok, changed}` (200); the row then shows `effect.text` "Since you changed this to "Never repair" (1 min ago): repairs: not reported, deaths: 0 ... (U)" and the return screen shows the before/after line with the revert button. The repairs counter is "not reported" (no event). The card exists and the revert button is offered; no repair happened to compare |
| Session record | V: `<state>/ladder/sessions/<id>/` holds `session.jsonl` (7.4 KB), `decisions.jsonl` (978 KB), `summary.json`, `summary.md`, written at SIGINT; mode 0600, directory 0700; the watch token does not occur in any of the files (grep, V). Size: 83 KB per virtual hour in `decisions.jsonl`, not the 30 KB of `docs/session-record.md`; the summary's per-hour figures use wall time and are misleading in lockstep (TASK-178). `class` is `paladin` in the summary and empty in the start line |
| Launcher | V: `d2bot play` started the ladder, watch, record and token file; I overrode the ports and video through the environment (kept because `play` only fills unset variables). The phone URL is printed to stderr only |
| funwire | V by the above: return screen, tally and Codex all received real goal events (`Fun.OnGoalEvent`) |

## Questions (TASK-137 and TASK-138 fixes)
V: lockstep 30 questions in 707 virtual minutes (`t0.retreat.drink` 12, `t0.area.toohard` 4, `t1.restock` 3, `t1.town.trigger` 3, `t1.return` 3, `t0.first` 3, `t1.repair` 1), 0 answered (I left them to fade on purpose), all 30 faded to their default; no key repeated every minute (the retreat question came in a burst of 3 at the start, then gaps of 84 s and more, and the others 5 to 107 s apart in wall time = many virtual minutes). Container: 13 questions in 20 real minutes, one answered through `POST :8892/api/answer` (200, the first choice of `t0.area.toohard`: "Heading to the Forgotten Tower: the bot wants hero level 15 there and the hero is level 1."), the rest faded; no repeat per minute. Recovery (TASK-138): the hero stays in "Recovery: farming gold" as designed, but with no kills the drought action (leave and re-enter the game) fires over and over in lockstep.

## Container (AC 1, realtime)
V: 21 minutes up, `RestartCount` 0, health healthy, stop took 12 s with exit 0 and `Palsmoke.d2s` in the volume. CPU of the container with `/stream` open all the time: average 35.6 percent of one core, max 42.1 (limit 2 cores). Memory average 581 MiB, max 599 MiB, 549 at the start and 552 at the end: no growth. Video: 4027 JPEG frames in 20 minutes (about 3.4 per second), 322 MB read (about 270 KB/s, 2.2 Mbit/s: heavy for a phone, U whether that matters). Deaths 0, stalls 0.

## Defects
| Id | Sev | What | Card |
|---|---|---|---|
| P1 | B | Paladin lands no hit in lockstep d2d (0 kills in 707 virtual minutes), drought loop | TASK-175 palmelee |
| P2 | M | Realtime Paladin: 13 kills and clvl 1 after 20 minutes, long idle stretches | TASK-177 palpace |
| P3 | M | Tally pays +2 for an empty expedition | TASK-176 tallyempty |
| P4 | L | Session summary uses wall time; redirect repeats; start record has no class | TASK-178 sessionwall |
| P5 | M | The wizard refuses the Paladin | TASK-145 (existing) |
| P6 | L | `combatBuild: blizzard` is printed for a Paladin (label only, U) | not carded |

## What a friend would see
They start the container, type the token URL, and the wizard offers only the Sorceress. If they get a Paladin anyway, the page loads and looks alive: a header with a tally strip that ticks up, a Codex that fills the first entries (Blood Moor, Quill Rats), a return screen that tells them "Your Paladin is level 1 in Blood Moor" and a short list of questions that appear and fade in 45 seconds. The hero walks into Blood Moor and fights, but in the first twenty minutes it kills about a dozen rats and stays level 1, standing in the recovery routine; on the lockstep soak it never kills at all. The charter, the policy change and the before/after card all work when poked through the API, but there is nothing for them to improve yet because the hero does not level, so the fun loop is real and the game under it is not yet.

## Built vs verified
Everything in the slices table is V from this run (curl against the viewer and answer listeners, files read on the worker host). Not covered: the pages in a browser, answering many cards to teach a rule, a Corpsefire fight, the Sorceress run on this same build (the reference numbers are from an earlier run), the tally with the d.idle overlay, and a lockstep run with the Paladin able to kill.

## Bring it back
- Lockstep: `labrun`-free recipe: sync the worktree to the worker, then `docs/probes/paladinsmoke/run-a.sh TAG SEED RUNFOR` there (it builds `d2bot`, takes the first free slot through `d2dslot.sh` and runs `d2bot play`). The viewer is on loopback `9300+slot`, the answer listener `9400+slot`; the token is `watch.token` in the data dir.
- Container: image `d2dbot:palsmoke` and volume `d2pal-data` stay on the worker; run `docs/probes/paladinsmoke/build-img.sh` for the image and start the container with the settings of the run above (loopback ports 9391 and 9392, game folder with hard links of the four required archives, `D2D_CLASS=2`). Remove with `docker rm`, `docker volume rm d2pal-data`, `docker rmi d2dbot:palsmoke`.
