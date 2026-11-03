# Baseline runs: the extra original-game runs (card baselineruns)

> status: current, PREPARED ONLY (2026-11-01). Nothing here was run live; no game was started. The launcher is `tools/baseline-run.sh` in the private ops repo (stub tests only, 0 live runs). Tags: V = verified (a read-only look at the worker host, a committed file or a test, named), S = secondary (another doc's statement), U = unverified (needs the live preflight of section 9). Why the runs: `docs/research/baseline-audit.md` section 6 items 1 and 2 (the user decided to do them). Contracts: `openspec/specs/fidelity-gate/spec.md` (same class, level range and kit as the d2d run; at least 5 runs, 2 h each, profile m1).

Scope: 5 more ORIGINAL-GAME 2 h runs of a Sorceress at clvl 11 to 14 (profile m1) and 3 more fresh-start runs (clvl 1 start, early band up to clvl 7). Original game = Game.exe 1.14d under Wine on the worker host, offline, hidden instances i1..i5. No d2d, no fork binary, no lab binary is needed.

## 1. How the existing runs were produced (V unless noted)

Every existing original-game baseline came from one tool chain: `tools/soak/soak24.sh start iN ... --char NAME` (public repo), which starts the managed bot (`d2bot --managed --goals campaign --autostart --stop-game`, shipped per instance by `tools/live.sh --bg --nocap`) and the read-only harness `cmd/d2soak` (`tools/soak/soak.sh start`) on the worker host, and then sets the two soak knobs `resource.retreatBelt=1`, `resource.minFieldBeltPct=50` over the bot's API (source: `tools/soak/soak24.sh`). The numbers come from the `minute` lines of the bot log (`~/d2lod-live/soak24[-iN]/runs/<ts>-d2bot.log`), turned into rows by `d2dsoak baseline 'NAME|LOG|SOURCE|NOTE'` (`docs/d2d-soak.md` section 1 and 3; the same extractor `d2dsoak extract` is used for d2d, so the XP/h and kills/h definitions cancel).

| run | what | how started | log (worker host) | source |
|---|---|---|---|---|
| a14, a16, a18 | Soaky Sorceress, clvl 11-12 / 13-14 / 14, instance i4, profile m1, 2 h, bots de9b597 / b2e010c / b478bd9 | `soak24.sh start i4 --gate --char Soaky` (`--gate` = `--for 2h --profile m1 --min-xp-per-hour 0`) | `~/d2lod-live/soak24/runs/20261029-235659-d2bot.log`, `...20261030-030600...`, `...20261030-061457...` | `docs/g1-log.md` attempts 14, 16, 18; `tools/d2dsoak/baselines.json` |
| Soaky early | the same character at clvl 1 to 7, old builds | `soak24.sh`, 3.5 h wall | no minute lines (logs start at clvl 8) | S: `docs/soak-analysis.md` via `baselines-early.json` |
| Fresha | fresh uncheated Sorceress, instance i5, 2026-10-30, profile m24, 4 h, bot a920fef | `soak24.sh start i5 --for 4h --profile m24 --char Fresha` (restarted three times on newer builds) | `~/d2lod-live/soak24-i5/runs/20261030-095519-d2bot.log`, `...105216...`, `...120747...` | `docs/d2d-baselines-early.md` section 1: the first 115 minute lines up to the first clvl 7 line, `d2dsoak extract -hours 1.916667` |
| G2 | Soaky, 8 h, profile m24, clvl 14-15, 4 deaths | `soak24.sh start i4 --for 8h --profile m24 --char Soaky` | `~/d2lod-live/soak24/runs/20261030-082115-d2bot.log` | `docs/g1-log.md` (not yet a baseline row) |

How the characters were prepared (V, `docs/g1-log.md`, `docs/fresh-character.md`, the cheat ledger on the worker host): there was no scripted "kit". Soaky was played by the bot itself from clvl 8 to 15 across attempts 1 to 18; each run simply continued where the previous one stopped (clvl 11 and 0 gold at a14, clvl 13 and 211 gold at a16, clvl 14 and 16 gold at a18, so the kit differed per run). The ledger holds 5 lines for Soaky and all are refused or failed `godrestore` entries (V: read-only look at `cheats.jsonl`, 2026-11-01), so Soaky and Fresha (0 ledger lines) count as honest-play. Fresha was created with `tools/soak/fresh.sh` (OOG tooling) and never cheated. Nothing was ever restored to an earlier state; there is no restore tool in the repo.

Seeds: the original game has no seed control in our tooling (V: `baselines.json` description and `docs/research/baseline-audit.md` section 1a: "the original game has no seed control; 3 independent live runs"). Every game rolls its own dungeon. Whether the `.map` file of a character pins or re-rolls the map seed is U (never tested); the recipe below copies the whole file set of a character, so the first live run should compare the `.map` before and after one game (checklist item 6). Runs are therefore "independent live runs on one kit", identified by run name and bot commit, not by a seed.

## 2. What state each run type needs

| | late (5 runs) | fresh (3 runs) |
|---|---|---|
| class, level | Sorceress, clvl 11 to 14 at the start (byte 43 of the `.d2s`; V: Fresha reads 11, Soaky 15 on 2026-11-01) | Sorceress clvl 1, XP 0, softcore expansion Normal (the `fresh.sh` check) |
| kit | honest play: the character as a previous bot run left it; never in the cheat ledger | no cheats ever (`docs/cheats.md` Policy for fresh test characters) |
| profile, length | m1, 2 h (`soak24.sh --gate`) | m24 as Fresha, 3 h wall; the row is cut at the first clvl 7 line (about 2 h), the early file wants 3 h per seed but the band ends earlier (audit section 6 item 2) |
| bot | the current origin/main commit, clean checkout (soak24 refuses a dirty tree and a `d2bot --version` that differs from HEAD) | same |
| instance | any free hidden instance i1..i5 (never i0) | same |

Characters that exist now (V: read-only `.d2s` header read and ledger count on the worker host, 2026-11-01; levels change as soon as anything runs):

| character | instance | class | clvl | ledger lines | use |
|---|---|---|---|---|---|
| Fresha | i5 | Sorceress | 11 | 0 | late kit (clvl 11 at the end of its 4 h run, 2026-10-30) |
| Freshb, Freshc, Freshd | i5 | Sorceress | 1 | 0 | one fresh run each (never run: created by `fresh.sh` for exactly this) |
| Soaky | i4 | Sorceress | 15 | 5 refused | out of band (above 14); keep as is |
| Booti | i4 | Sorceress | 12 | 24 | cheated: not an honest kit |
| Navi 24, Combi 30, Multi 30 | i1, i2, i5 | Sorceress | | | out of band |

So exactly one honest, in-band Sorceress exists (Fresha). Five late runs need either five characters leveled honestly (about 4 h of bot time each, as Fresha took 2026-10-30) or one kit restored five times.

## 3. Restoring the clvl 11-14 kit between runs (decision for the user, see section 9)

What exists: no restore command, no kit file, no `cheat set-level` path to a honest kit. `d2cheat level N` sets level and XP only (V, `docs/cheats.md`), it does not build a kit, and any ledger line disqualifies a character (`docs/cheats.md` policy), so it is NOT used. `d2cheat saveedit` takes a backup before each edit and V live a hand restore of `Multi.d2s` was byte-identical (`docs/cheats.md` LIVEQ, 2026-10-28).

Recipe used by the launcher (a file copy of a character that was played honestly, game stopped, no game memory touched):
1. `start` snapshots every file of the kit character (`Fresha.*`: `.d2s .key .ma* .map`) from its instance into `~/d2lod-live/baseline-kits/<set>/<char>/` on the worker host, byte-copy with `cp -p`, and prints level and a sha256 prefix.
2. Before each late run the driver places the snapshot into the lane instance's `Saved Games/Diablo II` (path V: `~/Games/d2lod-inst/<iN>/proton/pfx/drive_c/users/steamuser/Saved Games/Diablo II`), moving any same-name files to `baseline-kits/<set>/replaced-<iN>/` first, and compares the `.d2s` byte for byte. It refuses when the instance lock is held.
3. The same placement serves the fresh runs (the clvl 1 files of Freshb/c/d are copied into a lane's prefix when the lane is not i5).

Consequences: all 5 late runs start from the identical kit (same level, XP, gold, gear), which is what the spec's "same kit" asks and removes kit variance, but it also means the five runs do not spread over kit differences as a14/a16/a18 did (U: the effect on the baseline's SD). Placing a character into a prefix that did not have it is a file-level creation: agents never create characters, so this needs the user's or the coordinator's go (section 9). The kit's gear and gold are not recorded in the repo (U); the first checklist step reads and records them.

## 4. Run recipes

Per run (the driver does these; the commands are what `tools/baseline-run.sh` runs, from the bot checkout `D2_BOT_DIR`, which must be clean and equal to origin/main):

| step | late | fresh |
|---|---|---|
| restore | kit placed on the lane instance, `.d2s` compared | the fresh character's snapshot placed (or already there on i5) |
| start | `tools/soak/soak24.sh start iN --gate --char Fresha` | `tools/soak/soak24.sh start iN --for 3h --profile m24 --char Freshb` |
| what runs | managed bot for 2 h + 5 min, harness for 2 h; knobs set; graceful save at the end | same for 3 h + 5 min |
| end | the bot log gets a final `[exit 0]` line from `live.sh` (the driver waits for it) | same |
| stop | `tools/soak/soak24.sh stop iN` (also cleans the instance) | same |
| output | bot log `~/d2lod-live/soak24-iN/runs/<ts>-d2bot.log` (minute lines), harness dir `~/d2lod-live/soak/iN-<ts>/` | same |
| duration | 2 h wall = 2 h of play (the original game has no virtual clock: about 1 wall hour per hour) | 3 h wall; the row uses the first 1.5 to 2.2 h |

Rows (V recipe, the Fresha precedent): the driver keeps only the `minute` lines, cuts a late run to its planned length (a 2 h run keeps 120 lines; the bot runs 5 min past the harness) and a fresh run at the first line with clvl 7 or more, then `d2dsoak extract -hours <lines/60> minutes.log` (hours = minute lines / 60, restart gaps excluded, the a14/a16/a18 and Fresha definition) and, in `collect`, `d2dsoak baseline 'NAME|LOG|SOURCE|NOTE'` for every run that has data. Late rows go to `rows-late.json` (for `tools/d2dsoak/baselines.json`), early rows to `rows-early.json` (for `baselines-early.json`). The tracked baseline files are not edited by the launcher: merging changes what the gate compares with, so it is a coordinator step; `record.md` carries the exact merge snippet.

Quality flags written per run (`quality=ok|weak|invalid` in the row note, reasons spelled out; the baseline files have no `weak` field, so the note text starts with `WEAK:`):
- weak: fewer than 95 percent of the planned late hours (a14 was a partial run); fresh band shorter than 1.5 h; start clvl outside 11-14 (late) or not 1 (fresh); a non-zero `restarts` counter inside the run (a16 had an OOM restart); a non-zero bot exit code; more than 2 percent unparsable minute lines.
- invalid (no row): fewer than 10 minute lines ("run produced no data", the judge rule of the gate runs), or the character has any cheat-ledger line.
- notes only (row kept): ended above clvl 14 (late), band not completed (fresh).
The d2soak harness verdict (leak slopes, failsafe events) is stored as `harness-status.txt` per run, not parsed into the flag. Deaths are kept as counted (a 2 h run with a death is valid data; the audit wants the real death rate).

## 5. Number of characters and instances

Late: 1 kit character (Fresha) restored before each of 5 runs, or 5 honest characters if the user rejects restores (then each needs about 4 h of honest leveling first, which is out of scope here). Fresh: 3 characters (Freshb, Freshc, Freshd, one run each; a fresh character is used up after its run). Instances: a run needs one hidden instance for its whole length; instances i1..i5 are assigned to other agents in `tools/instances.conf` but are only busy while an agent runs something there; the preflight checks the real state (lock, bot process, soak unit, helper).

Schedule from `tools/baseline-run.sh plan` (V: tool output and the stub test; capacity measured read-only, V: `capacity2.sh --add 3` and `--add 6` both gave `verdict: OK` on 2026-11-01 with idle+nice+phantom-wait about 90 percent; it changes by the minute):

| lanes (parallel runs) | instances | total wall time for all 8 runs | cores added (1.3 per lane) |
|---|---|---|---|
| 3 (default) | i5 i4 i3 | 465 min = 7.75 h (fresh x3 at 3 h on the three lanes, then late x5 over the three) | 4 |
| 5 | i5 i4 i3 i2 i1 | 330 min = 5.5 h | 7 |
| 1 | one | 21 h | 2 |
Per-run overhead is 15 min (restore, the 5 min bot tail, stop, pull, extract). The audit's estimate (about 5 h and 3 h in two batches) is the same order. The capacity rule is the `capacity2.sh` verdict after adding the cores (idle+nice+phantom-wait at least 25 percent, memory and load guards); one instance costs about 1.05 cores and 265 MB (V, `docs/multi.md`), the bot and harness share (U). The preflight refuses while any d2d gate run is active on the worker host and while another baseline set is active.

## 6. The launcher (ops repo, `tools/baseline-run.sh`)

```
tools/baseline-run.sh plan      [--late 5] [--fresh 3] [--parallel 3] [--instances "i5 i4 i3 i2 i1"] [--late-hours 2] [--fresh-hours 3]
                                [--kit-char Fresha] [--kit-inst i5] [--fresh-chars "Freshb Freshc Freshd"] [--overhead 15] [--no-capacity]
tools/baseline-run.sh preflight [same options] [--deep]       # --deep also runs soak24.sh preflight per lane
tools/baseline-run.sh start     [same options] --user-go [--set NAME]
tools/baseline-run.sh status | abort | collect [SET]
```
- `plan` is read-only (one capacity probe). `preflight` refuses (exit 3, 4 or 5) unless: capacity OK, no gate run, no other set, bot checkout clean and equal to origin/main, the kit and fresh characters exist at the right class and level and are not in the ledger, and every lane instance has a free lock, no bot, no soak unit, a running helper and no restart notice.
- `start` needs `--user-go` (the user's go), re-runs the preflight, builds `d2dsoak`, writes `plan.env` (every value single-quoted, checked with `bash -n` and a read-back, the gate-run lesson), snapshots the kits and starts the driver detached under `tools/agentrun.sh baseline --ttl <1.5 x longest lane + 1 h>` on the head box. The driver (`tools/baseline-sup.sh`) works one lane per instance and records `~/.cache/d2-baseline/<set>/` (state, sup.log, runs/<name>.state, runs/<name>/{bot.log.gz, minutes.log, extract.txt, quality.env, harness-status.txt}).
- Loud failures: a bad or unsafe plan, a missing soak24 or d2dsoak, a failed start, a timeout (1.5 x hours + 30 min without an `[exit]` line) or an unexpected driver exit write `phase=error` and `result=ERROR: ...`, never `phase=done`; a run with fewer than 10 minute lines is `error`, not a row. Instance ids are checked against `^i[1-9]$` (hidden instances only; the bot's own id rule `[a-z0-9-]{1,16}` is met), set names against `[a-z0-9-]`.
- `abort` writes a stop marker, runs `tools/agentreap.sh baseline`, calls `soak24.sh stop` for the lanes that are active and writes `phase=aborted`.
- `collect` builds the rows and the branch `agent/baselineruns-<date>` in `~/code/d2lod-wt/` from origin/main exactly like `gate-run.sh --collect` (record, band minute logs, card in-progress with `skip_specs: true`, OWNERS lines, `merge-lines.txt`), committed locally, never pushed; an error set with no data builds nothing (exit 2).
- Tests: `tools/tests/test_baseline_run.sh` (scratch HOME as worker, fake soak24 and d2dsoak, real agentrun and agentreap): validation, plan numbers, every preflight refusal, quoting, the full 3-run set with collect, loud failures, weak flags, abort.

## 7. Per-run checklist (human or coordinator, first set; later sets drop the first four)

1. User's go for original-game runs (ADR 0013 retired Game.exe gates; the lab is shared).
2. Agree the kit: one restored kit (section 3) or five leveled characters; record the kit's gear and gold from the snapshot.
3. Make sure the bot checkout is clean at origin/main and the helper on the worker host matches this checkout's build (soak24 preflight fails otherwise: `tools/helper.sh deploy` from this checkout, which restarts a helper, so announce it).
4. Make sure the lane instances are free (no agent driving them) and that i0 is not in the list.
5. `tools/baseline-run.sh plan` then `preflight --deep` (all PASS, capacity OK).
6. First lane only, before the full set: run one late run alone (`--late 1 --fresh 0 --parallel 1`), then compare the kit's `.map` file before and after the game and read the minute lines (section 9 items).
7. `start --user-go`; follow with `status` every 20 to 30 minutes (no polling agent: the driver and the games live on after the session).
8. After `done`: `collect`, review `record.md` (flags), then the coordinator merges rows into the baseline files and re-runs `d2dsoak compare` on an existing d2d set to see the new baseline interval (the audit's "print the baseline interval" proposal).
9. Cleanup: `tools/game.sh status` and `tools/agentreap.sh --list` show nothing of the set; the characters are saved; `baseline-kits/<set>` stays as the record of the kit.

## 8. Dead ends and limits

- `d2cheat level` as the "restore to clvl 11-14" path: rejected (ledger disqualifies the character, kit not built, XP/gear differ from honest play).
- Running the late runs on d2d: rejected (the baseline must come from the original game; `docs/research/baseline-audit.md` section 4b).
- Leveling five fresh characters honestly first: possible but about 4 h of bot time per character (Fresha 2026-10-30), and every level-up run has the older-build kit problem; not planned.
- 5 lanes are only possible when the other agents leave i1..i5 free; the plan with 3 lanes needs three free instances.
- The Gate 1 per-clvl XP numbers of the original were never recorded, so a "per band" XP table cannot be built from the old runs (audit section 7).

## 9. What must be supplied, and what is unverifiable until a live preflight

Supplied by a human or the coordinator before `start` (the agent that wrote this did none of it):
1. The user's go (`--user-go`), and the decision on the kit restore (a file-level copy of an honest character, section 3).
2. Characters: Fresha (clvl 11, kit) and Freshb/c/d (clvl 1) exist in i5 now; for lanes other than i5 someone must authorise placing copies into those prefixes (or create fresh characters there with `tools/soak/fresh.sh`, OOG tooling). Five honest in-band characters do not exist.
3. Free instances and a quiet worker host at the time (the preflight decides); a bot checkout at a pushed commit; the helper deployed from that checkout.
4. No fork binary, no lab binary, no the worker host/d2d slot is needed (V: nothing in `soak24.sh` touches d2d).

Unverifiable until a live preflight (U, none of these was run):
- That the restored `.d2s` set loads in the game and the bot continues as usual (V only for a manual byte-identical restore of Multi, `docs/cheats.md`); and whether the `.map` file pins the map seed (then restored runs would replay layouts: independence lost).
- The per-lane CPU cost of the bot and harness (1.3 cores is the plan's assumption), and whether 3 or 5 hidden instances plus the bot stay under the capacity rule for 2 to 3 h.
- The exact bot-log format of the `[exit N]` line, `soak24.sh status` output and the log file names for the lanes other than i4/i5 (the stub test imitates them from `tools/live.sh` and `tools/soak/layout.sh`).
- `soak24.sh` from a fresh head-box checkout (`go build` of d2bot and the 386 helper on a 4 GB box) and its helper-hash check.
- The kit's gear and gold (not recorded in the repo).
- The `weak` thresholds (95 percent of hours, 1.5 h fresh band) are my choice (I), derived from the a14 and Fresha precedents; the coordinator may tighten them.

## 10. Hooked lanes: `--timescale K` (the paired experiment k=K against k=1)

> status: tooling prepared offline (card `lockstepbase`); not run live. Tags: V = by the offline tests of `tools/tests/test_baseline_run.sh` (ops repo, fake game folders), U = unverified. Background: `docs/timescale.md` (the bot clock), `docs/research/lockstep-p4.md` (the hook at k=2 and k=4 with a scaled bot clock). The hook itself is a private tool; nothing of it is in this repository.

What it is: a lane whose game runs K times faster than real time (K = 1, 2 or 4; default 1 is the unchanged behaviour) with the bot's clock scaled the same (`D2BOT_TIMESCALE=K`). Hours are always GAME hours: a 3 h lane at K=2 is 90 real minutes. `baseline-run.sh plan --timescale 2 ...` prints the regime line, the real and game lengths, the capacity verdict (cores taken as K times the k=1 figure, U) and the full list of commands `start` would run.

| topic | rule |
|---|---|
| one minute line | one GAME minute at K != 1 (S: the observer takes its time from `env.Clock`, which delegates to `internal/gclock`; the pacing check below is the live test, U). `hours = lines / 60` is therefore game hours and `d2dsoak extract -hours` gives XP per game hour as before |
| pacing check | the quality script compares the wall span of the minute lines with `(lines - 1) x 60 / K` seconds; more than 15 percent off marks the lane weak ("minute lines are not paced as game minutes", the sign of an unscaled bot clock) |
| series | `runs/<name>/series.tsv` (driver host only): per minute line the game minute, the real minute (log time), clvl, xp, kills, deaths, XP per game minute and XP per real minute; `quality.env` has `gamemin`, `realmin`, `xp_game_h`, `xp_real_h` |
| `--for` / `--run-for` | soak24's `--for` and the bot's `--run-for` are host timers (real seconds): the driver passes `--for <game hours x 3600 / K>s` (2 h at k=2: 3600 s; the bot runs 5 real minutes longer, the extra lines are cut). Farm duration and goal ActiveSeconds stay real time (docs/timescale.md) |
| env forwarding | `D2BOT_TIMESCALE` is set on the `soak24.sh start` call and forwarded by `tools/live.sh` (ops repo) to the bot, only when set. The bot logs `gclock: timescale K` at start (`gclock.StartNote`); the driver waits for that line AND the hook's first STAT line (5 min), else it stops the instance (`soak24.sh stop`, `game.sh stop`) and retries once |
| bot commit | the pin must contain `internal/gclock` and the log line: preflight refuses otherwise. `b023979a` (set `kit`) does not (V: `git cat-file`, 2026-11-02), so a pair needs a newer pin and the pair then differs by that bot diff, not only by K |
| per lane | `game.sh stop` first, kit restore, hook install (the game's `ijl11.dll` is renamed to `ijl11_real.dll`, never written, the proxy is copied as a new file, `d2hook.cfg` with `offline=1`, `k=K`, `stat=10`), helper, start. At the end and on every failure or abort: stop, hook log archived as `runs/<name>/hook-stat.log.gz` (driver host; path only in the record), original restored and compared by sha256 (a mismatch is a loud error) |
| watchdog | every 3rd poll the driver reads the hook STAT lines: scaled/real time ratio more than 2 percent off K for 5 minutes, or no new STAT line for 3 minutes, marks the lane weak with the reason; it never restarts anything |
| record | `plan.env`, each lane's state file, `quality.env`, the `collect` rows (note starts `REGIME k=K`) and `record.md` carry K and the sentence `regime: scaled clock k=K, not comparable to k=1 without the pair`. Rows of a K != 1 set must not be merged into the k=1 baseline files |
| fresh characters for the pair | `--fresh-from-set SET` takes the clvl-1 files from the snapshots of set SET (`baseline-kits/SET/<char>`, taken at that set's start) instead of the kit instance, so the pair uses the same characters (same `.map` seed, U whether that pins the layout) without touching the kit instance or creating anything |

Unverified (U): that a hooked lane reaches the first STAT line within 5 min from a cold start; the CPU share at K=2 (taken as 2x); that the bot's farm and goal timers (real time) distort nothing at K=2; that soak24's harness alerts (xp stagnation windows) use game or real time; whether the `.map` seed pins dungeon layouts across copies.
