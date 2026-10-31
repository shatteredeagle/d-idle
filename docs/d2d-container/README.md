# d2d container: build, run, mount, update

> status: implemented locally (cards d2d-container-build and d2d-container-bot); design and risks in `design.md`, `risks.md`, `gpl.md`, `licensing.md`. Nothing is deployed to the NAS; no image is pushed anywhere.

Files (all in `deploy/d2d-container/`): `Dockerfile.d2d` (staged build), `context.sh` (assembles the build context), `entrypoint.sh`, `d2dping/` (healthcheck and test client), `smoke.sh` (also `tools/d2d-container-smoke.sh`).

## What it is
One container = one d2d (the slot: no lock, one fixed socket `/run/d2d/d2d.sock`). Inside, under tini and a non-root user (uid 1000):
`d2dview` tap (always on, listens on 127.0.0.1:8891, creates `d2d.sock.ctl` full access and `d2d.sock.view` read-only) -> `d2d` headless; in `bot` mode `d2bot --backend d2d --managed --d2d-dial d2d.sock.ctl` starts, drives, saves and quits d2d. d2d takes ONE client, so everything (bot, healthcheck, `docker exec` probes) goes through the tap. No Wine, no X, no ptrace.

## Build (on the worker host; docker build is blocked on the head box)
```
labrun worker -- nice -n10 tools/d2d-container-smoke.sh -k    # context + build + smoke test, keeps the image d2dbot:smoke
```
or by hand: `deploy/d2d-container/context.sh CTX [REF]` then `docker build --build-arg BUILD_JOBS=4 -t d2dbot:dev CTX`.
`context.sh` takes the fork from `FORK=~/d2d-lab/fork` (`git archive` of REF, default `lab5`, the newest branch on the forge remote; vendored CLI11, sentry-native and FFmpeg 9.0.2 (hash checked) come from that checkout's earlier build), builds our Go binaries (`d2bot d2dsoak d2dview d2dping`, CGO off) and writes `licenses/` and `.dockerignore`. `PREBUILT=/path/d2d` skips the source stage (`--build-arg D2D_SRC=prebuilt`); that binary must match Ubuntu 26.04's libstdc++ (the worker host's GCC 16 binaries probably do not, U).
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
| `D2D_VIEW_ADDR` | 127.0.0.1:8891 | viewer tap HTTP (non-loopback needs a token, docs/d2d-view.md) |
| `D2D_DATA`, `D2D_BIN`, `D2D_SOCK`, `D2D_ARGS` | `/game`, `/opt/d2d/d2d`, `/run/d2d/d2d.sock`, unset | paths and extra d2d args |

Ports: none exposed by default (which ones to publish is the user's call: 8431 dashboard as in `design.md` section 7; viewer 8891 and bot API 8420 are loopback inside). The socket lives on the container filesystem; a bind mount from the NAS's shfs may not support unix sockets (U), keep it off `/mnt/user`.
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
  <Overview>D.idle on d2d: idle bot for classic Diablo II LoD 1.14d, headless and offline. Needs your own game MPQs in /game.</Overview>
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
