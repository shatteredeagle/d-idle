# Dev-loop update: new bot builds in a running container, no image rebuild

> status: CURRENT. Task: TASK-192 (devupdate). Builds on the merged updater (docs/self-update-release.md, docs/self-update-client.md, docs/self-update-drill.md). Dev tooling only: no registry, no CI key, no GitHub. Tags: V = verified live in the drill below (2026-11-08, worker host, a throwaway container with no hero and no game started); S = offline tests; U = unverified.

## What it is

One command builds the bot from a tree, signs it with a DEV key, publishes it to a small LAN directory, tells the running container to take it, and the container restarts only the bot process (the launcher and its children) on the new release. The container, its volume and the shim stay up. The measured time was 5 to 6 seconds per update with a warm Go cache (V).

```
tools/dev-update.sh --to d2dev-20261108          # this checkout, uncommitted changes included
tools/dev-update.sh --commit REF --to NAME       # a commit
tools/dev-update.sh --worktree DIR --all         # another worktree, every running DEV container
```

Output is one line: `UPDATED <container> <old> -> <new> in <N>s`, or `REJECTED <container>: <reason>`, or `ERROR ...`. `-v` shows the progress lines. `N` includes the build.

## The pieces

- A DEV image: `ARG DEV=1` in `deploy/d2d-container/Dockerfile.d2d` sets `DIDLE_DEV=1` and the label `io.didle.dev=1`. A normal image has `DEV=0` and ignores everything below (S, `deploy/d2d-container/update_image_test.go`, `internal/selfupdate/dev_test.go`).
- Three variables, read by the shim (`cmd/didleboot`) and by the bot (`cmd/d2bot/update.go`) only when `DIDLE_DEV=1`:
  - `D2BOT_UPDATE_URL` base URL of the LAN directory. The client asks `<URL>/repos/<repo>/releases` like the GitHub API and may talk to that host only (also on redirects).
  - `D2BOT_UPDATE_PUBKEY` the DEV public key, as text or as a file path. It is trusted in addition to the baked keys (none are baked yet).
  - `D2BOT_UPDATE_CHECK` `on` (default, daily check) | `off` (no outbound call; the old spellings 0, false, no still work) | `manual` (no daily check, a request still asks). The dev compose override sets `manual`.
- The shim (`/opt/didle/boot`), DEV image only: SIGHUP stops the launcher the normal way (bot saved, then tap) and starts the selection again; `check-now` and `rollback-now` do the update and send that SIGHUP; a pending release is confirmed after 15 seconds of staying up (no probes: there may be no hero) and still rolled back when it dies sooner (V). The launcher now runs in its own process group and leftovers are swept after it ends (V: without it the setup wizard survived a TERM and held its port, and the new release could not start; found by the first drill).
- `tools/dev-update.sh` (driver, runs anywhere `labrun` works) and `tools/dev-update-host.sh` (worker-host half): build with `nice -n 19`, `d2sign manifest/sign/verify`, publish under `~/.cache/didle-dev/www`, then `docker exec <c> /opt/didle/boot check-now` and wait until the new release's processes run (V: checked through `DIDLE_RELEASE=<ver>` in `/proc/*/environ` and `image.json`).
- The DEV key: `~/.config/didle-dev/update-ed25519.{key,pub}` on the worker host, made on first use, key file 0600, never printed, never committed, never the release key. `tools/dev-update.sh --pubkey` prints the public half.
- Versions are `0.0.<UTC yymmddHHMMSS>` (the updater needs `x.y.z`), `seq` the same number, so every build is newer than the last (monotonic, also within one second).

## One-time setup

1. Start the LAN server: `tools/dev-update.sh --serve start [--ttl SECS]` (python http.server under `agentrun.sh devupdate`, default TTL 8 h, port 18899 or `DIDLE_DEV_PORT`). Stop: `--serve stop`. It serves signed bot builds only, no secrets.
2. Rebuild the image ONCE with DEV=1: `tools/dev-update.sh --image d2dbot:dev --d2d <the d2d binary of your current image>` (context from `context.sh` with `PREBUILT`; 16 to 21 s with warm layers). The legacy builder runs every stage, so the script makes the source stage a stub when a prebuilt d2d is given.
3. Start the container from that image with the override: `D2_IMAGE=d2dbot:dev D2_DEV_URL=http://<worker host>:18899 D2_DEV_PUBKEY="$(cat <the .pub file>)" D2_GAME_DIR=... D2_WATCH_ORIGIN=... docker compose -f compose.tester.yml -f docker-compose.dev.yml up -d` (the file is `deploy/d2d-container/docker-compose.dev.yml`). The same three variables work with `docker run -e`.

Every later change is only `tools/dev-update.sh --to <container>`.

## Rollback

`tools/dev-update.sh --rollback --to <container>` goes back to the previous release, or to the baked base when there is none, and restarts the bot process (V: `ROLLED-BACK ... in 2s`). The engine keeps the last two releases. A rolled-back version is not offered again; the next build has a new version and applies normally.

## Drill (V, 2026-11-08)

A throwaway container (own name, volume, ports, image tag) on the worker host, setup-wizard state, no game started, `writes after start: 0` (no game memory exists to write). Results:

| Step | Result |
|---|---|
| image once with DEV=1 | built in 16 to 21 s |
| update 1 (commit A) | `UPDATED ... 0.0.0 -> 0.0.261108153617 in 6s` |
| update 2 (commit B) | `UPDATED ... 0.0.261108153635 -> 0.0.261108153704 in 5s`; `d2dping devbanner` of the current release printed drill-B. (An earlier B run still showed drill-A: rsync skipped a same-size file with an equal mtime; the driver now extracts with `tar -m`.) |
| bad signature (other key) | `REJECTED ... signature does not verify`, nothing swapped |
| no `.sig` asset | `REJECTED ... has no asset MANIFEST.json.sig` |
| manifest tampered after signing | `REJECTED ... signature does not verify` |
| tarball differs from the signed hash | `REJECTED ... larger than the manifest says` |
| older but validly signed release served alone (rollback attack) | `REJECTED ... refusing a release older than the floor (downgrade)` |
| signed build whose launcher dies | the shim rolled back to the previous release by itself; the script says `ERROR ... did not stay up and the shim went back to ...` |
| `--rollback` | `ROLLED-BACK ... in 2s`, previous banner back |
| `docker restart` | stays on the dev release; marker file in the volume survived every step |

The drill hooks are `tools/dev-update.sh --fault badsig|unsigned|tamper|hash|crash|stale --to <c>`. They publish a fault release for one check and clean it up at the next real publish.

## What is NOT covered (needs an image rebuild or is untested)

- The shim, the Dockerfile (packages, ENV, user), baked keys and the ABI label: by design nothing an update can change.
- The d2d fork binary and game data: the dev script ships the bot only. The release format and the shim can carry more (`d2d` and `data` destinations, `D2D_BIN` points at `d2d/d2d` of the release when it exists, `bin/entrypoint.sh` already updates the mode logic through this channel), so a later `--with-d2d PATH` is a small addition to `dev-update-host.sh`; it was not built or run (U). Runtime readers of the `data` destination do not exist yet (U).
- A running hero: the update restarts the bot, which saves and quits the game through the normal TERM path and starts it again; the setup-wizard state was drilled, a hero in a fight was not (U). Treat each update like a `docker stop` of the bot.
- The updater's daily GitHub check is untouched and off here; the watch-screen update banner works against the dev directory too but was not opened in a browser (U).
- The dev server is plain HTTP on the LAN: anyone on the LAN can read the builds; they cannot make a container take an unsigned one.

## Switching a running player container (decision for the owner, nothing here does it)

Create the DEV image, then recreate the container ONCE from it with the three variables. The data volume is untouched (hero, watch token, access file, wizard state); the game directory stays a read-only mount. The bot is stopped the normal way (character saved) and started again, a one to three minute warm-up like any restart. Keep the same published ports and watch origin. After that every change is one command.

## Warm the item art after a deploy (TASK-268)

The inventory page's game art is prepared in the background at every start of `d2bot` (`<data>/cache/art`, from the owner's own install, docs/paperdoll-art-build.md). After a dev update there is nothing to do; to warm it on purpose or to see a count run `d2bot art extract` inside the container (idempotent, a finished cache costs a few milliseconds). `/api/art` reports `state` and `done/total` while it runs.

## Deleted files (TASK-281)

`labrun --send` is a plain rsync without `--delete` (S: read in the labrun source), so a file removed from the repo used to survive in the worker copy of `.devupdate-src` (a stale `subreaper.go` collided with `subreaper_linux.go`: `IsSubreaper redeclared`). The driver now runs `labrun HOST -- rm -rf .devupdate-src` before the send, so the worker copy mirrors the extracted tree exactly. Regression test (S, offline, fake labrun, scratch repo, never the sandbox): `tools/dev-update-clean-test.sh` deploys tree A with a file, then tree B without it, and checks it is gone and absent from the build input. Not run against a real worker host (U).
