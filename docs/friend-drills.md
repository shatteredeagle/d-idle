# Friend pack pre-release drills (D1 to D6)

> status: RUN 2026-11-11 (D1 to D6 with fake game files; second run the same day with the real game folder: D3 and D4 now PASS, section "Second run") on the worker host, scratch project `drill-friend`, ports 18991 (feed), 18992/18993 (page, answers). Plan: docs/research/friend-build.md section 10. Tags: V = run, output seen; S = offline test; U = not run. Everything here used a THROWAWAY Ed25519 key and a scratch feed; nothing was uploaded, the real feed host, the sandbox container and the release key were not touched.

## Method

- Image: this repo at origin/main (`e364d7ff`) through `deploy/d2d-container/context.sh` with a prebuilt d2d (taken from the previous release image via a throwaway `docker create`), the source stage stubbed like `tools/dev-update-host.sh` does, `--build-arg DEV=1 --build-arg D2D_SRC=prebuilt`, tag `didle:drill-<stamp>`; the baked `pubkeys/` replaced by the throwaway public key (as `beta.pub`). Never pushed.
- Pack: `tools/friendpack/build.sh --version 0.1.0 --image didle:drill-<stamp> --image-tar T --d2d-src placeholder.tar.gz --d2d-commit <pin> --pubkey drill.pub --out OUT` (5 s; the d2d source tarball was a placeholder because the pack never left the host).
- Feed: `tools/release/app-assets.sh` (versions 0.1.1 to 0.1.6, `--channel beta`) signed with `d2sign`, then `tools/release/feed-publish.sh --base-url https://<worker host>:18991 --out FEEDTREE --pub drill.pub`; the URLs were rewritten to `http://` with `sed` afterwards and served by `python3 -m http.server 18991` under `tools/agentrun.sh`. `d2feed` and the published `D2BOT_UPDATE_FEED` require https by design, so the drill used the sanctioned DEV mode (docs/dev-update.md): a DEV image reading `D2BOT_UPDATE_URL` from a `docker-compose.override.yml` next to the pack's compose file. This means the non-DEV `D2BOT_UPDATE_FEED` path was NOT driven live (S: internal/selfupdate feed_test.go only).
- Game files: fakes (4 files holding the text `fake`; later none valid). No Blizzard data was used and no real hero exists, which decides D3/D4 below.

## Results

| Drill | Result | Detail |
|---|---|---|
| D1 clean install | PASS (V) | See D1. |
| D2 first run, fake MPQs | PASS (V) | The setup page refuses with a clear message; nothing starts, art extraction does not run. |
| D3 update v1 to v2 | PASS (V, second run) | Engine, shim, `./didle update` end to end and the update API behind the banner on a running bot; the banner's own rendering in a browser was not opened (U). |
| D4 update while the hero fights | PASS (V, second run) | Level 3 Paladin, in a fight; saved, restarted on the new release in 14 s, level and pack intact. |
| D5 bad releases refused, rollback | PASS (V) | Six refusals and two rollbacks, old version kept running every time. |
| D7 bad release with a live hero (second run) | PASS (V) | Crash release rolled back in 4 s, hero resumed, banner data present. |
| D6 `./didle report` | PASS (V) | No token, home path, game path, IP or credential; one note below. |

### D1 install from a clean directory (V)

```
sha256sum -c didle-friends-0.1.0.tar.gz.sha256        # OK
tar xzf ...; ./install.sh                             # no game dir, no terminal: licence shown, then
STOP: the licence note needs an answer; run ./install.sh from a terminal.
./install.sh --game-dir BAD  (1 of 4 files)           # STOP: missing in BAD: d2exp.mpq d2char.mpq Patch_D2.mpq. Point --game-dir at your own Diablo II folder ...
./install.sh --port 80                                # STOP: --port must be a number from 1024 to 65534
./install.sh --port 18992 (port in use)               # STOP: port 18992 is in use. Pick another: ./install.sh --port 9100 (uses 9100 and 9101).
VERSION edited after the checksum was made            # STOP: the download is damaged or changed (sha256sum -c SHA256SUMS fails). Fetch it again.
./install.sh --game-dir GAME --port 18992 --project drill-friend --host <worker host>
  ok: docker and compose ... pack checksums ... licence note accepted for 0.1.0 ... game folder (mounted read-only) ... free space and ports ... image loaded (abi 1)
  d.idle 0.1.0 is running.   http://<worker host>:18992/setup.html?t=<TOKEN>
```

The printed URL carried the right origin: `.env` has `D2_WATCH_ORIGIN=http://<host>:18992,http://localhost:18992,http://127.0.0.1:18992`. `GET /setup.html?t=<token>` answered 303 (cookie set), without a token 401. The wrong-origin POST test was inconclusive (setup server answers PUT/POST only on the printed host name; with `127.0.0.1` as host the setup server answers `forbidden origin`, which is the intended behaviour). The licence prompt needs a real terminal (drilled with `script`); a piped `yes` is correctly refused.

Observations: a missing game folder and no terminal prints the whole licence note before the "give your Diablo II folder" error (the game-dir question comes after the licence). Harmless.

### D2 first run with fake MPQs (V)

`GET /api/state` and `POST /api/check` on the setup server:
`d2data.mpq is not a Diablo II archive (wrong file header). Copy the original file.` for each of the four files, summary `Problems with: d2data.mpq, d2exp.mpq, d2char.mpq, Patch_D2.mpq.`, verdict `not_mpq`, optional archives `info`. `POST /api/start` answered `400 {"error":"The game files are not ready: ...","fix":"Copy the original file again; this one is not a Diablo II archive."}`. No d2bot, no d2d, no art cache. A fake with a valid MPQ header would still stop at the pinned `Patch_D2` hash (`wrong_version`, S: cmd/d2dsetup tests; not run live). With the wizard bypassed (drill shortcut: `{"phase":"done"}` written into the scratch volume) the bot dies at once with `d2bot: gear: /game/Patch_D2.mpq: mpq: header not found` and the container restarts every 3 to 8 seconds; that state cannot be reached by a friend because the wizard gates it.

### D3 update from the feed (partly V)

```
feed holds 0.1.1 only:  boot check-now      UPDATE 0.1.0 0.1.1   -> running release 0.1.1, "release 0.1.1 confirmed healthy"
publish 0.1.2:          boot check-now      UPDATE 0.1.1 0.1.2   -> running release 0.1.2, confirmed healthy
marker file in /data (stand-in for the hero save) read back after every step: intact
```

The version line (`DIDLE_RELEASE` of the processes) changed each time; the setup server restarted with the same token. NOT covered: `./didle update` end to end and the watch-screen banner, because both live in the bot's update API, which does not run until the wizard accepts real MPQs. `./didle update` in the setup state used to exit 56 with no text; fixed in this branch (`STOP: the bot's update service does not answer. ...`, exit 1; S: bash -n, the pack tests; the new line itself was not run live).

### D4 update while the hero fights (first run: blocked; PASS in the second run below)

Needs a real hero in a game: d2d refuses fake archives (`mpq: header not found`). The stop path (SIGTERM, then 45 s grace) is the same one docker stop uses (S: docs/self-update-client.md); nothing was observed. To drill it a 1.14d game folder must be mounted read-only into the drill container, which this run did not do (brief: fake files only). Proposed: run D3/D4 once with the lab's own game folder on a throwaway hero.

### D5 bad releases and rollback (V)

Running 0.1.2, feed newest release 0.1.3 unless stated; every line ended with `running 0.1.2` unchanged:

| Fault | Answer |
|---|---|
| signed by another key | `REJECTED signature does not verify` |
| `.sig` listed but missing | `REJECTED MANIFEST.json.sig: HTTP 404` |
| manifest edited after signing | `REJECTED signature does not verify` |
| tarball longer than the signed size | `REJECTED ... larger than the manifest says` |
| same size, one byte flipped | `REJECTED downloaded file does not match the manifest: ...: sha256 mismatch` |
| signed `REVOKED` bans 0.1.3 | `REJECTED this version was revoked: 0.1.3` |
| only an older signed release (seq 9 below 12) | `REJECTED refusing a release older than the floor (downgrade): seq 9 below 12` |
| signed 0.1.4 whose launcher exits at once | `UPDATE 0.1.2 0.1.4`, then the shim: `rolling back 0.1.4 to "0.1.2": exited with code 1 before it was confirmed healthy`, running 0.1.2 again; the retry says `0.1.4 was rolled back before and is not retried` |
| good 0.1.3 after that | `REJECTED ... older than the floor (downgrade): seq 13 below 14` (the failed 0.1.4 raised the floor; recovery is a higher fixed version) |
| `boot rollback-now` right after that | `ROLLBACK 0.1.2 0.1.0` (back to the base, not to 0.1.1: see Findings) |
| fixed 0.1.5, then 0.1.6, then `rollback-now` | `UPDATE 0.1.0 0.1.5`, `UPDATE 0.1.5 0.1.6`, `ROLLBACK 0.1.6 0.1.5`, 0.1.6 not offered again |

### D6 `./didle report` (V)

Four files (`versions.txt`, `ps.txt`, `log.txt`, `update-status.json`). Greps over the extracted report for the real token (32 characters), the home path, the game folder path, any IPv4 other than loopback, `Bearer` followed by a value: 0 hits each. The host name the friend typed for `--host` appears (it is the friend's own name, set on purpose); the `update-status.json` file was empty because the bot was not running.

## Second run: real game folder, live hero (2026-11-11, V)

Scratch project `drill-live` on the worker host (own volume, image `didle:drill-<stamp>`, ports 18991 feed / 18992 page / 18993 answers, own throwaway DEV key, nothing shared with the player sandbox: its feed directory, key and agent name were separate and it was not touched). The game folder was the lab's own Diablo II install mounted read-only (`:ro`); no Blizzard file was copied. Image from the working tree (origin/main plus this branch) with `tools/dev-update-host.sh --image` (DIDLE_DEV_DIR, DIDLE_DEV_KEYDIR, DIDLE_DEV_PORT pointed at scratch dirs), pack with `tools/friendpack/build.sh --pubkey <scratch pub>`, install with `./install.sh --game-dir <real folder> --port 18992 --project drill-live --host <worker host>` (licence answered through `script`), `docker-compose.override.yml` with `D2BOT_UPDATE_URL`, `D2BOT_UPDATE_PUBKEY`, `D2BOT_UPDATE_CHECK: manual`. The non-DEV https `D2BOT_UPDATE_FEED` path was again NOT driven (S only).

Hero: `Drillo`, Paladin, created through the setup wizard (`PUT /api/inputs`, `/api/ahead`, `/api/access` token, `POST /api/start`), zero cheats and zero memory writes (docs/probes/test-characters.md). Run records: writes after start: 0.

| Step | Result (trimmed) |
|---|---|
| wizard check | all 4 required and 4 optional archives `pass`; start accepted |
| first start, art | `game art ready icons=394 missing=0` 3 s after `d2bot start` (fresh volume, `/data/cache/art` 1.9 MB); `GET /api/art` `state: ready, done 394/394`; the hero was already clearing Blood Moor at level 1 about 60 s after `POST /api/start` |
| first page load | `/watch` 200 in 0.5 ms (8.6 KB); `/` answers 302; `/api/inventory` 12 ms (loopback, from the worker host; not a browser) |
| banner data | publish a signed release (`dev-update-host.sh --url ...` = v2), `POST /api/update/check` on the answer port, then `GET /api/update/status` shows `"available":{"version":"0.0.261111220150","notes":"dev build tree","size":37637413,"needs_image":false}` (what `update.js` renders) |
| D3/D4 `./didle update` while fighting (level 3, `state: clearing`, 3 hostiles near) | `release 0.0.261111220150 is available.` then after `y`: `applied: the bot restarts on 0.0.261111220150`. Log: `Bot stopped`, `stopping d2d (save and quit)` 6 s after the click, `Drillo.d2s` written, `didleboot: starting release ... pending=true` at +14 s, `class-aware start ... "level":3`, `release ... confirmed healthy` 15 s later. Status after: `running 0.0.261111220150`, `rollback` empty. |
| hero persisted | before/after: class Paladin, level 3, equipment Buckler + Short Sword, pack Club, Leather Armor, 2x Leather Gloves, 2x Minor Healing, 2x Minor Mana, 2x Identify, TP scrolls; after: same set plus a Short Bow and Short Staff picked up during the minutes between (the hero was fighting) and one TP scroll fewer (the town trip). Gold 26 to 28. No item lost. |
| D7 bad v3 with the hero alive (`--fault crash`, signed, launcher exits 1) | `./didle update` applied it; `FAULT crash`, `didleboot: rolling back 0.0.261111221053 to "0.0.261111220150": exited with code 1 before it was confirmed healthy`, old release started 4 s after the stop, hero level 3 resumed, status `running 0.0.261111220150` with `rollback:{from,to,reason,seen:false}` (the banner shows "failed and was rolled back"). The container restarted for both the update and the rollback (`RestartCount 2`, the designed docker restart after the stop). |
| `./didle status`, `./didle report` | right version, page URL with the token; report greps: token 0, home path 0, game folder path 0; `update-status.json` now carries the real status including the rollback (earlier it was empty). |
| capacity | host load stayed under 3 on 16 cores during the run; the scratch d2d used about 1 core. |

Not covered: the banner drawn in a browser (the API and `update.js` source were read, the DOM was not opened), the non-DEV https feed, `./didle update` with the hero in town (the hero was deliberately fighting), a second bad-release chain (`rollback-now` after a crash rollback, see Findings 3).

Cleanup: `docker compose -p drill-live down -v`, image tag removed, the feed server reaped, scratch dirs deleted; the player sandbox container stayed up and healthy throughout.

## Findings

1. FIXED here: `./didle update` printed nothing and exited 56 while only the setup server runs (tools/friendpack/didle).
2. FIXED in agent/frienddrills2: `/opt/didle/boot` started with no arguments in a running container started a second launcher and overwrote `/data/didle/boot.pid`, after which `check-now` answered `ERROR pid N is not the boot shim` and the next update was rolled back. DEV mode now refuses a second shim while boot.pid names a live boot process (`otherShimRunning`, S: cmd/didleboot TestOtherShimRunning; the refusal was not run live).
3. Cause found by reading (S): after the crash-rollback of 0.1.4 `rollback-now` went to the base 0.1.0 instead of 0.1.1 because the engine keeps only two releases (`Prune(new, old)` at apply) and `Layout.Rollback` clears `Previous`: 0.1.1 was pruned when 0.1.4 was applied. This is the documented two-release retention, not a pointer bug. Proposed card (container, small): keep three releases, or say in docs/self-update-client.md that a rollback after a failed update goes to the base.
4. Proposed card (ops, small): `feed-publish.sh` and `d2feed` only accept https, so a LAN drill must rewrite the tree afterwards (or use the DEV loop). A `--allow-http` flag for scratch drills would remove the sed step; the published feed must stay https.
5. DONE in the second run: D3 page/CLI engine and D4 with a real hero (above). Still open: open the banner in a real browser once (U).
