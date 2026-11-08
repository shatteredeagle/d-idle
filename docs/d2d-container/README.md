# d2d container: build, run, mount, update

> status: implemented locally (cards d2d-container-build and d2d-container-bot); design and risks in `design.md`, `risks.md`, `gpl.md`, `licensing.md`. Nothing is deployed to the NAS; no image is pushed anywhere.

Files (all in `deploy/d2d-container/`): `Dockerfile.d2d` (staged build), `context.sh` (assembles the build context), `entrypoint.sh`, `d2dping/` (healthcheck and test client), `smoke.sh` (also `tools/d2d-container-smoke.sh`).

## What it is
One container = one d2d (the slot: no lock, one fixed socket `/run/d2d/d2d.sock`). Inside, under tini and a non-root user (uid 1000):
`d2dview` tap (always on, listens on 127.0.0.1:8893, creates `d2d.sock.ctl` full access and `d2d.sock.view` read-only) -> `d2d` headless; in `bot` mode `d2bot --backend d2d --managed --d2d-dial d2d.sock.ctl` starts, drives, saves and quits d2d. d2d takes ONE client, so everything (bot, healthcheck, `docker exec` probes) goes through the tap. No Wine, no X, no ptrace.

## Build (on the worker host; docker build is blocked on the head box)
```
labrun worker -- nice -n10 tools/d2d-container-smoke.sh -k    # context + build + smoke test, keeps the image d2dbot:smoke
```
or by hand: `deploy/d2d-container/context.sh CTX [REF]` then `docker build --build-arg BUILD_JOBS=4 -t d2dbot:dev CTX`.
`context.sh` takes the fork from `FORK=~/d2d-lab/fork` (`git archive` of REF; the default is the pin `lab3-labverbs` commit `f75816c616c8` = the Gate 1 and Gate 2 fork `ee5e06d8ef99` plus the `--lab-verbs` gate, V `git ls-remote` of the fork's private remote, run read-only on 2026-11-06, shows `f75816c616c8253930d88277b90936e3eabe3bc8`; vendored CLI11, sentry-native and FFmpeg 9.0.2 (hash checked) come from that checkout's earlier build), builds our Go binaries (`d2bot d2dsoak d2dview d2dsetup d2dping`, CGO off) and writes `licenses/` and `.dockerignore`. `PREBUILT=/path/d2d` skips the source stage (`--build-arg D2D_SRC=prebuilt`); that binary must match Ubuntu 26.04's libstdc++ (the worker host's GCC 16 binaries probably do not, U).
Never copy fork sources into this repo; the context lives in a scratch dir.

## Run
```
docker run -d --name d2dbot --memory 3g --cpus 2 --cap-drop ALL --security-opt no-new-privileges:true \
  -v /path/to/mpqs:/game:ro -v d2dbot-data:/data -e D2D_NAME=Mychar -e D2D_CLASS=1 d2dbot:dev        # bot mode (default)
docker run ... d2dbot:dev d2d                                                                          # bare d2d, for tests
docker exec d2dbot /opt/d2lod/d2dping ping            # healthcheck verb, read-only tap
docker exec d2dbot /opt/d2lod/d2dping -sock /run/d2d/d2d.sock.ctl state
```
Game data: your own MPQs (`d2data.mpq`, `d2exp.mpq` required; d2char, Patch_D2, d2sfx, d2speech, d2music as in `design.md` section 3), read-only at `/game`, never in an image layer. Volume `/data` (uid 1000): saves in `/data/config/d2d/save/`, bot state, logs.

| Env | Default | Meaning |
|---|---|---|
| `D2D_MODE` | `lockstep` | `lockstep`, `realtime` or `timescale:N` (live product: realtime; soaks: lockstep) |
| `D2D_NAME` / `D2D_CLASS` | `D2dbot` / `1` | character; class in d2d UI order (1 = Necromancer). An existing save of that name is loaded, else a new character is created |
| `D2D_SEED`, `D2D_NOSAVE` | unset, 0 | map seed; `1` = throwaway character |
| `D2BOT_RUN_HOURS`, `D2BOT_GOALS`, `D2BOT_ADDR`, `D2BOT_ARGS` | unset, unset, 127.0.0.1:8420, unset | bot run limit (`--run-for Nh`), goals, API address, extra flags |
| `D2D_VIEW_ADDR` | 127.0.0.1:8893 | viewer tap HTTP (non-loopback needs a token, docs/d2d-view.md) |
| `D2D_DATA`, `D2D_BIN`, `D2D_SOCK`, `D2D_ARGS` | `/game`, `/opt/d2d/d2d`, `/run/d2d/d2d.sock`, unset | paths and extra d2d args |

Ports: none exposed by default (which ones to publish is the user's call: 8431 dashboard as in `design.md` section 7; viewer tap 8893 and bot API 8420 are loopback inside). The socket lives on the container filesystem; a bind mount from the NAS's shfs may not support unix sockets (U), keep it off `/mnt/user`.
Healthcheck: `d2dping ping` against `d2d.sock.view`. Stop: SIGTERM, the bot saves and quits d2d (grace 45 s suggested, U).

## Update flow
Rebuild from a newer fork ref (`context.sh CTX <ref>`), run the smoke test, replace the container from the template (data volume untouched). A fork ref must be on the forge `<owner>/d2d-fork` the same day it is used (AGENTS.md); `lab6` was not pushed when this was written.

## NAS template sketch (a DOC for the user to approve; NOT installed)
Follows `deploy/nas/my-d2idle.xml`; apply later through `tpl-apply.sh` so the container shows in the Docker tab with every field visible:
```xml
<Container version="2">
  <Name>d2dbot</Name>
  <Repository>d2dbot:dev</Repository>
  <Network>bridge</Network>
  <Privileged>false</Privileged>
  <Overview>d.idle on d2d: idle bot for classic Diablo II LoD 1.14d, headless and offline. Needs your own game MPQs in /game.</Overview>
  <ExtraParams>--memory=3g --cpus=2 --stop-timeout 45 --cap-drop=ALL --security-opt=no-new-privileges:true --init</ExtraParams>
  <Config Name="Data" Target="/data" Default="/mnt/user/appdata/d2dbot/data" Mode="rw" Type="Path" Display="always">/mnt/user/appdata/d2dbot/data</Config>
  <Config Name="Game data (read-only)" Target="/game" Default="/mnt/user/appdata/d2dbot/game" Mode="ro" Type="Path" Display="always">/mnt/user/appdata/d2dbot/game</Config>
  <Config Name="D2D_MODE" Target="D2D_MODE" Default="realtime" Type="Variable" Display="always">realtime</Config>
  <Config Name="D2D_NAME" Target="D2D_NAME" Default="D2dbot" Type="Variable" Display="always">D2dbot</Config>
  <Config Name="D2D_CLASS" Target="D2D_CLASS" Default="1" Type="Variable" Display="always">1</Config>
  <Config Name="D2BOT_RUN_HOURS" Target="D2BOT_RUN_HOURS" Default="" Type="Variable" Display="always"></Config>
</Container>
```
Autostart stays off until a soak passes. Open: which ports to publish (host 8431 is held by the Wine `d2idle`), where the MPQs live on Tower (`design.md` question 3).

## Smoke test and results
`smoke.sh`: build, run `d2d` mode with MPQs mounted read-only, `--network none`, no capabilities; ping (tap), `state`, create a character (`--start-screen ingame`), `save`, stop and restart, `key Return` on the select screen, `dump player` shows the character.
Results (V, run on the worker host 2026-10-30 via `labrun worker -- nice -n10 tools/d2d-container-smoke.sh -k`, fork ref `lab5`): the d2d stage builds on Ubuntu 26.04 with g++ 15 and the vendored inputs, no network after apt (first build about 5 min wall at -j4 on a loaded worker host, 373 ninja steps; the later run took 39 s from cache); image 386 MB (`docker image inspect`). Smoke PASS: ping answers on the read-only tap, `state` shows `screen=ingame class=1 name=Csmoke`, the character is created and `save` writes `Csmoke.d2s/.key/.ma0/.map` into the volume, the container is stopped and restarted, `key Return` on the select screen loads Csmoke (`dump player`). Container had `--network none --cap-drop ALL`, MPQs `:ro`.
Bot mode (V, one 90 s try, `D2D_NOSAVE=1`, no goals): d2bot started d2d through the tap, character in town, status lines logged, run limit ended it with exit 0 and d2d was stopped. NOT verified: goals running for an hour, the healthcheck going healthy (a first inspect after exit said `unhealthy`; the `ping` verb itself answers, so check the 30 s interval and the stuck-failsafe save-and-exit of a goal-less bot, U), realtime mode, `--read-only` root.
Cleanup done: containers, volumes, the image and the context dir were removed on the worker host.

## Tester profile (cards testerwatch, testertap, testerd2dimage)

`entrypoint.sh tester` (or `TESTER=1` with the default `bot` command; the compose file sets `command: ["tester"]`) is bot mode plus the watch screen. Everything is environment; no secret is on a command line. The default `bot` mode keeps its settings (lockstep, no ladder, no watch), checked by `deploy/d2d-container/tester_image_test.go`.

First run for a tester (U until the image is built and run, phase B of testerd2dimage):
1. Mounts: your own Diablo II folder read-only at `/game` (`d2data.mpq`, `d2exp.mpq`, `d2char.mpq` and `Patch_D2.mpq` are required, V TASK-25; never copied into the image) and a named volume at `/data` (saves, ladder store, the watch token).
2. `D2_GAME_DIR=<your folder> D2_WATCH_ORIGIN=http://localhost:8891 docker compose -f compose.tester.yml up -d`. `D2_WATCH_ORIGIN` is the URL you will type in the browser (the answer route accepts only that origin); for another machine use its name, e.g. `http://mybox:8891`, and `D2_PUBLISH=0.0.0.0` (default `127.0.0.1`).
3. Ports: 8891 watch screen and video, 8892 answer route. The viewer tap (127.0.0.1:8893) and the bot API (127.0.0.1:8420) stay inside the container.
4. The token: on first start d2bot creates `/data/watch.token` (mode 600) and prints the viewer URL once in the log. Read it later with `docker compose -f compose.tester.yml exec d2dbot cat /data/watch.token` (or `docker run --rm -v d2dbot-data:/data busybox cat /data/watch.token`, or `docker logs`). Open `http://localhost:8891/watch` and give the token.
5. Stop with `docker compose stop` (grace 45 s: the bot saves and quits d2d). The same volume keeps the character.

Dry run without the game: `ENTRYPOINT_DRYRUN=1 TESTER=1 D2BOT_WATCH_ORIGIN=http://localhost:8891 sh deploy/d2d-container/entrypoint.sh` prints the resolved `d2dview`, `d2bot` command lines and the `D2*` environment with tokens masked and starts nothing; `DRYRUN_ONLY=1 deploy/d2d-container/smoke.sh` checks it (V, ran on the head box 2026-11-06). `PROFILE=tester deploy/d2d-container/smoke.sh` also runs the container and checks the token file (U, not run).

### Environment table

| Variable | Default | Set by | Meaning |
|---|---|---|---|
| `TESTER` | `0` | compose | `1` selects the tester profile for `bot` |
| `D2D_MODE` | `lockstep` (bot, d2d), `realtime` (tester) | compose / user | `lockstep`, `realtime`, `timescale:N`; the image no longer sets an ENV default |
| `D2D_NAME`, `D2D_CLASS` | `D2dbot`, `1` | compose / wizard | character and class (d2d UI order) |
| `D2D_SEED`, `D2D_NOSAVE`, `D2D_ARGS` | unset, 0, unset | user | map seed, throwaway, extra d2d args (never put a lab flag here for testers) |
| `D2D_DATA`, `D2D_BIN`, `D2D_SOCK` | `/game`, `/opt/d2d/d2d`, `/run/d2d/d2d.sock` | image | paths |
| `D2D_VIEW_ADDR` | `127.0.0.1:8893` | image | tap listen address (also `D2DVIEW_ADDR`, set by the profile); loopback |
| `D2DVIEW_TOKEN` | random per start | entrypoint | tap token shared with d2bot's tap video source; never printed |
| `D2BOT_GOALS` | `campaign:normal_act1` (tester) | profile | goals |
| `D2BOT_RUN_HOURS`, `D2BOT_ADDR`, `D2BOT_ARGS` | unset, `127.0.0.1:8420`, unset | user | run limit, bot API (keep loopback), extra flags |
| `D2BOT_GAMEDIR` | `$D2D_DATA` (tester) | profile | game tables for combat, town and campaign goals |
| `D2BOT_LADDER` | `1` (tester) | profile | decision ladder on |
| `D2BOT_LADDER_STORE` | `shared` (tester) | profile | a plain name, not a path: `shared` is the store under the bot state in the `/data` volume (the container sets `D2BOT_DATA=/data`) |
| `D2BOT_WATCH`, `D2BOT_WATCH_VIDEO`, `D2BOT_WATCH_VIDEO_SOURCE` | `1`, `1`, `tap` (tester) | profile | watch screen, video from the tap |
| `D2BOT_WATCH_ADDR`, `D2BOT_WATCH_ANSWER_ADDR` | `0.0.0.0:8891`, `0.0.0.0:8892` (tester) | profile | listeners inside the container; the host publish is loopback by default |
| `D2BOT_WATCH_ORIGIN` | `http://localhost:8891,http://127.0.0.1:8891` | compose (`D2_WATCH_ORIGIN`, required there) | page origins the answer route accepts |
| `D2BOT_WATCH_TOKEN_FILE` | `/data/watch.token` (tester) | profile | created at first start |
| `D2BOT_WATCH_TAP_URL`, `D2BOT_WATCH_VIDEO_FPS`, `D2BOT_WATCH_VIDEO_QUALITY` | tap URL from `D2DVIEW_ADDR`, d2bot defaults | user | tap source tuning |
| `ENTRYPOINT_DRYRUN` | `0` | tests | `1` prints the resolved command lines and exits |
| Compose only: `D2D_IMAGE`, `D2_GAME_DIR` (required), `D2_WATCH_ORIGIN` (required), `D2_PUBLISH` | `d2dbot:tester`, -, -, `127.0.0.1` | user | image tag, MPQ folder, origin, publish address |

### Lab and cheat verbs (user decision 2026-11-06: flag-gated in the shipped build, not stripped)

The shipped tester d2d runs without `--lab-verbs`: only read-only debug verbs answer; setup verbs reply `err lab verbs off`. The gate is the fork option `--lab-verbs` (default off, fork commit `f75816c616c8`, the pin of `context.sh`); the lab, drill and gate harness pass it, `entrypoint.sh` never does. Every gated and every open verb is listed in `docs/lab-verbs.md`. History: at the earlier pin `ee5e06d8ef99` the `debug` group answered unconditionally on the container-internal socket (V, read in the fork checkout). The bot's play path needs only `dump` and the read-only verbs (V, TASK-42 run, see `docs/lab-verbs.md`).

