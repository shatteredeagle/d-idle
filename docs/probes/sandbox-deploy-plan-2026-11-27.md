# Sandbox deploy plan, 2026-11-27 (runbook, nothing run)

> status: PLAN, docs only. Nothing was deployed, restarted or pruned in writing this. Tags: V = read in code or docs, or read-only command on the worker host today; S = secondary (an older probe doc); U = unverified, needs the first real run.

Sandboxes (both in the local docker of the worker host, image `d2dbot:devp30`, status healthy when checked, V): Doofer `d2play-20261108`, Questy `d2play-20261125q`. Both are DEV images, so `tools/dev-update.sh` applies (docs/dev-update.md, V). The container, its volume and the shim stay up; only the bot release under `/data/didle` changes.

## 0. Pre-checks (read-only)

1. Disk (V, today): the worker host root is 1.8T, 1.7T used, 23G free, 99 percent. `docker system df`: images 142 (119.4GB, 17.75GB reclaimable), containers 8 (180MB reclaimable), volumes 12 (30.66GB, 30.3GB reclaimable, but see below), build cache 0. A deploy needs well under 1 GB (a Go build with a warm cache, one release tarball in `~/.cache/didle-dev/www`), so it fits; a full disk during the signed publish would fail as ERROR, not corrupt a release (U).
2. Reclaim candidates, FACTS ONLY, do not prune (the owner decides): old `d2dbot:devpNN` images other than devp30 and the one both sandboxes run (about 438MB each, `docker images`); stopped containers (`heuristic_solomon`, `pensive_lehmann`, `tdarr-node`, `tdarr-node-old`, `rocm-whisper-api`, `WinBoat`, all Exited); dangling images (29 at the check); the volumes `docker-model-runner-models` (28.1GB) and `palimpsest-test` (2.2GB) belong to other projects, not ours: ask the owner. NEVER touch `d2play-data` or `d2play-q-data` (the heroes). Read again with `docker system df -v` and `docker images`.
3. Docker JSON logs: the container json logs are root-owned (not readable as the normal user, V). They hold NUL-damaged entries: `docker logs` without `--tail` stops at the first bad entry (S, docs/probes/farmloop-live-20261116.md); use `docker logs --tail 30000` and filter by time. Do not rely on `docker logs` for the post-deploy check. The new bot is meant to write `/data/instances/main/logs/bot.log`; U: I did not find that path written anywhere in the tree (the supervisor writes `instances/<id>/logs/d2bot.log`, docs/container.md), so confirm which file exists first: `docker exec d2play-20261108 ls -la /data/instances/main/logs/`.
4. Deploy window: `tools/play-sandbox/deploy-window.sh status` (due when 2 merges since the last deploy or a request is pending; a user-asked deploy: `request <why>`, then `run`).
5. Tree to ship: this checkout, or `--commit <main tip>` for a clean build (stamp is the commit; `-dirty` only when tracked files differ). Prefer `--commit` from an up-to-date main.
6. Flip watcher alive: `systemctl --user status d2flip-watch` (unit text from `python3 tools/flip/flip.py watch --install-unit`; never enabled by the tool, U whether it is installed). A flip window is open if `python3 tools/flip/flip.py status` lists one: wait for it to close first, since a restart mid-window restarts the bot under the watcher.
7. Dev server: `tools/dev-update.sh --serve status` (must be running, port 18899). Start with `--serve start --ttl 28800` if not.
8. Hero state: the update restarts the bot, which saves and quits the game through the normal TERM path. A hero in a fight was NOT drilled (U, docs/dev-update.md). Pick a moment with the hero in town or idle (`curl` /api/status, or the watch page).

## 1. Commands per sandbox (dry run first)

`dev-update.sh` itself has no dry-run flag (V, read). The dry run is the window wrapper, which prints what it would do and writes no state:

```
cd ~/code/d2lod-bot            # a checkout of main; run from the repo root
tools/play-sandbox/deploy-window.sh run --to d2play-20261108 --dry-run
tools/play-sandbox/deploy-window.sh run --to d2play-20261125q --dry-run
```

Real run, one sandbox at a time (Doofer first, check section 3, then Questy). `run --now` skips the "due" test when the user asked for the deploy; the record of the deployed commit goes to `~/.cache/didle-dev/deploy-window`:

```
tools/play-sandbox/deploy-window.sh run --now --to d2play-20261108 -- --commit <main tip> -v
tools/play-sandbox/deploy-window.sh run --now --to d2play-20261125q -- --commit <main tip> -v
# or without the wrapper:
tools/dev-update.sh --commit <main tip> --to d2play-20261108 -v
```

Output is ONE line: `UPDATED <container> <old> -> <new> in <N>s`, or `REJECTED <container>: <reason>` (nothing swapped), or `ERROR ...` (the shim rolls back by itself when the new launcher dies within 15 s, V in the 2026-11-08 drill). Write the printed version into the session record so a log analysis can exclude the restart.

## 2. What the update changes

- Changes (V): the bot release (`d2bot`, `d2dping` and the other cmd binaries of the signed bundle) under `/data/didle/releases/<ver>`, the `current` pointer, and the engine keeps the last two releases. Everything merged today is Go code in that bundle, so it arrives together; the watch page assets are served by the bot and arrive with it.
- Does NOT change (V, docs/dev-update.md): the shim, Dockerfile, packages, baked keys, the d2d fork binary, game data, the volume (hero, watch token, wizard state). If today's merges touched the d2d fork binary or the image, this deploy does not carry them (U: check `git log` for `deploy/` and the fork pointer since the last deploy; an image change needs a rebuild, a separate decision).
- Restart: only the bot process group. Measured 5 to 6 s per update with a warm cache, plus the bot restart; plan for 5 to 20 s of `UPDATED` time and then 1 to 3 minutes before the game is up again and the hero is back in a town (S: a restart is a warm-up like any other). The hero is saved and quit the normal way. Art cache re-warms in the background after the start (`d2bot art extract` inside the container to warm on purpose).

## 3. Tuning knobs survive (V by reading)

The knobs live in the volume, not in the release: `/data/instances/main/state/tuning.json`, the store envelope `{version, data:{profile, overrides}}` (tools/flip/flip.py, internal/bot/tuner.go). On every bot start `NewTuner` loads that file and takes `st.Overrides` and `st.Profile` over the config (internal/bot/tuner.go, `NewTuner`), then rebuilds; a saved override that no longer resolves stops the bot with `saved tuning ...: <err>` (so a renamed or removed knob in the new build is the one real risk: V from the code, never seen live). Therefore these overrides persist through the release swap and a container restart: `bot.arbiterShadow`, `bot.tickLogSec`, `agency.planner`, `combat.deliveryUpkeep` (2), `town.plan` (3), and on Questy `quests.stepExec`. Confirm before and after per sandbox:

```
docker exec <c> /data/didle/current/bin/d2bot ops status --why     # the knobs block shows the live values
docker exec <c> cat /data/instances/main/state/tuning.json         # overrides still there (no tokens in it)
```

`D2_FLIP_BIN` defaults to that same binary path. Check that every knob name above still exists in the new build's knob table (`docs/knobs.md` generated from code), and that each flipped value is within its Min/Max (the new build may have tightened a range). If `ops status --why` is missing a knob after the deploy, STOP and roll back (section 5).

Expected per-knob effect of the deploy: none beyond the code that reads them; the flip ledger rows (`data/darklaunch.json`) are repo state, not sandbox state, and are unchanged.

## 4. Post-deploy verification (read-only)

Per sandbox, within 5 minutes of `UPDATED`, then at 30 minutes:

1. Version: the `UPDATED ... -> 0.0.<stamp>` line, and `docker exec <c> readlink /data/didle/current` (U exact path) shows the same version.
2. Process and health: `docker ps` shows `healthy`; `docker inspect -f '{{.RestartCount}}' <c>` unchanged (the shim restarts the bot, not the container).
3. New verbs (U: written from the brief; `ops log` and the bot.log file were not found in this tree, so check `d2bot ops --help` inside the container first and use the file name that exists):
   ```
   docker exec <c> /data/didle/current/bin/d2bot ops log --tail 5 --grep 'inv=|tick'
   docker exec <c> /data/didle/current/bin/d2bot ops status --why
   ```
   Expect `tick` lines about every `bot.tickLogSec` seconds and an `inv=` invariant summary line; no `ERROR`, no `saved tuning` failure.
4. Quests (Questy, then Doofer): `curl -s http://<watch>/api/quests` (token as for the watch page; never print it). Expect each paused row to carry a `reason.code` (the closed set is in `internal/questboard/sm_reason.go`): `paused` for a held quest, `handin_waiting` while a hand-in is queued ("Hand-in is queued: ..."). A row with no code is a bug to report.
5. Explain: `curl -s http://<watch>/api/explain` answers 200 with the current decision text (U: shape not read).
6. Flip state: `python3 tools/flip/flip.py status` (no unexpected window), the watcher heartbeat fresh.

## 5. Rollback

- Code: `tools/dev-update.sh --rollback --to <container>` returns to the previous release (the engine keeps two), or to the baked image base (`devp30`) when none; V in the 2026-11-08 drill (`ROLLED-BACK ... in 2s`). A rolled-back version is not offered again; the next build has a new version. Do it on a stuck start, a `saved tuning` failure, a missing knob or any hard invariant breach.
- Image: nothing to roll back; the image `d2dbot:devp30` is untouched.
- Tuning: `flip.py` makes one backup per flip window: the old file text, tag `bak-flip-<ts>`, kept in the window state under `D2_FLIP_STATE` (default `~/.cache/d2flip`) until the window closes clean; a revert writes it back and restarts. No flip is part of this deploy, so there is no new backup; if tuning.json is damaged, restore from the last window state if it exists (U) or from a manual copy: take one BEFORE the deploy:
  `docker exec <c> cp /data/instances/main/state/tuning.json /data/instances/main/state/tuning.json.pre-deploy-20261127`.

## 6. Watch for 30 minutes and the auto-revert path

- Poll every 5 minutes: container healthy and RestartCount unchanged, `ops status --why`, the latest `inv=` and `tick` lines, deaths and refusals counters, `/api/quests` reason codes.
- Hard breach list (docs/process/dark-launch.md): a death, a stall, a potion not drunk, a refusal-as-failure storm. For an open flip window the watcher `d2flip-watch` (`flip.py watch --daemon`, polls every 60 s) restores the tuning backup, restarts and marks the ledger rows `dark`. The watcher does NOT roll back a bad CODE release: that is `--rollback` above, by hand. The shim itself reverts a release whose launcher dies within 15 s (V).
- Do not open a flip window during the 30 minutes; one change at a time.

## 7. Manual checks for the owner on a real page (after the deploy)

Tick each on the watch page (hard refresh first). All are U until someone does them; nothing here was seen in a browser.

- [ ] Vault: first load shows items without a long blank (art is pre-extracted, TASK-268); `Sell all` asks for confirmation and sells only the selection; the gold count changes.
- [ ] Counts refresh: inventory, vault and stash counters update after a sell or pickup without a page reload.
- [ ] Mercenary: equip an item on the merc; it shows on the merc panel and the bot does not undo it.
- [ ] Charms: `Carry` moves a charm to the inventory, `Stow` returns it to the stash/vault; counts agree.
- [ ] Akara `Hand In` button: it shows a busy state while the hand-in runs; when it waits it says WHY (the `handin_waiting` text, "Hand-in is queued: ..."); it does not double-fire on a second click.
- [ ] Return screen tour: leave the page idle, come back; the Return screen tour appears once, steps through, and can be dismissed.
- [ ] Any red error banner or console error: copy the line into the report.

## 8. What is U in this plan

The `ops log` verb and the `bot.log` path (not in this tree); the shape of `/api/explain`; whether `d2flip-watch` is installed and running; a hero mid-fight at restart; whether any of today's merges touched the d2d binary or the image; a full-disk failure mode; the browser checks in section 7; flip's container restart mode `bot` (not drilled).
