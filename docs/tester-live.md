# Tester container: first end-to-end run (TASK-25)

> status: run 2026-11-07 on the worker host. Tags: V verified live (how stated), U unverified. Slice 1 (d2d route) of `docs/research/tester-build-plan.md`; decision D3, ADR 0017.

## Setup of the run
- Image: `deploy/d2d-container/context.sh` with `FORK_URL=<the published fork>` (the d2d source is a clone of the PUBLIC fork at tag `didle-gate2-lab3`, commit `f75816c616c8`, checked against the pin in the script) then `docker build` on the worker host at nice 19. V: `licenses/SOURCE.txt` in the image names the public repo and that commit. The vendored inputs (CLI11, sentry-native, FFmpeg 9.0.2, hash checked) still come from the local fork checkout's earlier build dir, not from the public repo: a clean machine cannot rebuild yet without that checkout (see defect D9).
- Compose: `deploy/d2d-container/compose.tester.yml`, project `d2tester-<date>`, publish bound to the LAN for the second-machine check (`D2_PUBLISH=0.0.0.0`, origin set to the host name the second machine types), game folder read-only at `/game`, a fresh named volume.
- Game data: the owner's own MPQs (the d2d lab copy of the 1.14d install on the worker host). Only hard links of `d2data`, `d2exp`, `d2char`, `Patch_D2` were mounted (the minimal set). Nothing from the game folder entered an image, a repo or a commit.
- Second machine: the head box (curl over the LAN) for the wizard, the API, the answer route and the video. No browser was used (the page is U in a browser; see defects).

## Steps and results
| Step | Result | Tag |
|---|---|---|
| Image builds from the public tag | yes (408 MB) | V |
| First start prints the wizard URL with the token once | yes | V |
| Wizard: data check, 1.14d verdict | `ok`, version `1.14d`: the owner's `Patch_D2.mpq` hash matches the pinned one, all four required files `pass` | V |
| Wizard: go-ahead, name, class, goals, access (token), Start | all accepted over `PUT /api/ahead`, `/api/inputs`, `/api/access`, `POST /api/start`; `wizard.json` phase `done`; table extract (`data.Open`) on the real MPQs worked with the minimal set | V |
| Bot starts after the wizard, hero created through the wizard class (Sorceress, d2d index 4) | yes, after the entrypoint fixes below; `class-aware start ... Sorceress`, town, then Blood Moor, fights, levels up | V |
| Watch screen from the second machine with the token | `/watch` 200, no token 401, answer route: wrong origin 403, no token 401, POST on the viewer 405 | V |
| A thought card appears, an answer lands | `GET /api/thoughts` lists the `t0.area.toohard` question with choices; `POST :8892/api/answer {"thought":1,"answer":"skip"}` answered `ok`, the journal shows `You answered "skip it"` | V |
| An order is given | `POST :8892/api/orders {"kind":"go","area":"Blood Moor","minutes":10}` gave `ok`, order `queued` in `GET /api/orders`, the bot then fought in Blood Moor | V |
| Video (tap source) delivers frames | `GET /stream` is `multipart/x-mixed-replace`, 113 JPEG frames in 30 s (about 3.8 per second), 800x600; `GET /shot.jpg` shows the hero walking in Blood Moor | V |
| Stop, start keeps the character | only after three fixes (D2, D3, D4); then `Tester.d2s` is written on stop and the next start loads it from the select screen | V |
| Same-volume image upgrade keeps the character and the ladder store | the image was rebuilt and the container recreated four times on the same volume; the ladder store (`/data/instances/main/state/ladder/decide-human.json`) and the save survived each time | V |

## The minimal MPQ set (AC 2)
Tried on the worker host with hard links of the owner's own files (V, bare d2d in the image, 25 s each):

| Files in `/game` | Result |
|---|---|
| `d2data`, `d2exp`, `d2char`, `Patch_D2` (the minimal set) | wizard check `ok` and 1.14d verdict `ok`; table extract works; the bot plays for an hour (below). d2d warns for each missing sound and music file and carries on |
| minimal set minus `d2char` | d2d exits 1 at once: `[ERROR] open_game_data: no d2char.mpq` |
| minimal set minus `Patch_D2` | d2d starts and creates the hero but the hero has no stats (`life=0/0 str=0`) and logs `[ERROR] finish_scene: stoi`: the 1.14d tables are in the patch |
| all nine files | works (stats as normal); sfx, music, speech are not needed |

Answer: the required set is confirmed as `d2data`, `d2exp`, `d2char`, `Patch_D2`; nothing needs widening. The optional files (`d2sfx`, `d2music`, `d2speech`, `d2xmusic`, `d2xtalk`) stay optional. The wizard list was already right; the entrypoint only checked two of the four and now checks all four (defect D6).

## Defects found
Severity: B blocker, H high, M medium, L low. "Fixed" means fixed in this branch with a test in `deploy/d2d-container/tester_image_test.go`.

| Id | Sev | Defect | Status |
|---|---|---|---|
| D1 | B | `D2BOT_LADDER_STORE=/data/ladder` is not a plain name: d2bot exits at once, the container crash-loops (V, log line `is not a plain name`) | fixed: the profile uses `shared` (the store lives under `/data/instances/main/state/ladder` in the volume) |
| D2 | H | `eval d2bot ... &` made `$!` the eval subshell, so a container stop never reached d2bot: no save, instant exit (V, `stop took 0s`, no `.d2s` in the volume) | fixed: `eval exec d2bot` |
| D3 | H | the stop handler sent TERM to the tap and the bot together; the bot's save-and-quit goes through the tap, failed, the bot relaunched d2d and docker killed the container after the 45 s grace (V, stop took 45 s) | fixed: bot first, wait, then the tap; a stop now takes 13 s and the save is written |
| D4 | H | the "saved character" check looked in `$XDG_CONFIG_HOME/d2d/save`, but the managed d2d saves under `/data/instances/<inst>/d2d/config`: every restart created a NEW hero over the saved one (V, d2d command line `--start-screen ingame --start-class 4`) | fixed: both places are checked; V after the fix: `--start-screen charselect`, a clvl 3 hero came back at clvl 3 after a recreate |
| D5 | L | `init: true` in compose plus tini in the image: tini warns it is not PID 1 | fixed: `init: true` removed |
| D6 | L | the entrypoint's early file check named only `d2data` and `d2exp`; a missing `d2char` or `Patch_D2` ended in a d2d log line | fixed: all four checked |
| D7 | M | the same ladder question (`t0.area.toohard` 17 times, `t0.retreat.drink` 9) reopens about every minute after an answer (V, journal and sampler log) | card TASK-137 (ladderrepeat) |
| D8 | M | a fresh hero stayed in "Recovery: farming gold" for the hour: 128 kills, clvl 3, gold 13 (V, bot `minute` lines); a tester sees a hero that never leaves Blood Moor | card TASK-138 (freshstart) |
| D9 | M | the d2d source now comes from the public fork tag, but the vendored inputs (CLI11, sentry-native, FFmpeg) come from the local fork build dir | card TASK-139 (imgrepro) |
| D10 | L | the video frame is 800x600 from the tap, not the 832x640 slot display of TASK-23 (V, `shot.jpg`) | to TASK-23 (decide whether the tester page needs 832x640) |
| D11 | L | the setup page and the watch page were not opened in a browser (U); the wizard URL with the token is printed once into the container log (by design, README) | open; browser check is a human step |
| D12 | L | the video client reconnected twice in an hour (V, sampler `err`=2, cause not found, no stall on the bot side) | U |

## Realtime session
V live, one container, `D2D_MODE=realtime` (the tester default), image from the public fork tag, 2026-11-07 (EST 18:39 to 19:40), the character kept from the first hours of the run (a Sorceress, created through the wizard path). The session is one continuous start of 61 minutes (`docker inspect` StartedAt to the last sample).

A script on the worker host sampled every 20 s: container CPU and memory (`docker stats`), `/api/uptime`, `/api/thoughts`, answered every open question with its first non-"not sure" choice through the answer route, and read `/stream` for the whole hour (the video was open the whole time).

| Measure | Result |
|---|---|
| Length | 61 min, 174 samples |
| Stalls | 0 (`stallSec` max 0), uptime 100 percent in every sample |
| Deaths | 0 (`deaths` 0 in every sample; bot `minute` lines say `deaths:0`) |
| Container restarts | 0 (`RestartCount` 0, health `healthy`); no `heartbeat flat`, no `d2d stopped answering pings`, no `campaign stalled`, no drought action |
| CPU with the video open | average 39.6 percent of one core, max 131.9 percent (a burst), end 35.6 percent; the container is limited to 2 cores |
| Memory | average 607 MiB, max 641 MiB, limit 3 GiB, no growth trend (570 MiB at 2 min, 619 MiB at 60 min) |
| Video | 12363 frames in 61 min (about 3.4 per second), 947 MB read; 2 client reconnects |
| Thoughts | 33 opened (`t0.area.toohard` 17, `t0.retreat.drink` 9, `t1.return` 3, `t1.repair` 2, `t1.restock` 2), 27 answered through the answer route, the rest faded to their default; one `confirm` thought was not answered by the script (U: what a "yes" does is not tested) |
| Play | 128 kills, clvl 1 to 3, xp 2919, gold 13, always in Blood Moor / town (defect D8) |
| Stop and upgrade | after the hour the container was recreated on the same volume (13 s stop): the hero came back at clvl 3, the ladder store survived |

Writes after start: 0. No cheat tool and no memory write was used on the game (d2d runs in a container without `--lab-verbs`: setup verbs answer `err lab verbs off`); the run is uncheated. It is not a gate run (no `d2cheat runstart`).

Not covered: a session longer than one hour, a person watching in a browser, two testers, a d2d freeze, the campaign-stall exit and restart path (rows 3 and 1 of `docs/tester-realtime.md`).
