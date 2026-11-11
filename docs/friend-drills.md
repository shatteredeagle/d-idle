# Friend pack pre-release drills (D1 to D6)

> status: RUN 2026-11-11 on the worker host, scratch project `drill-friend`, ports 18991 (feed), 18992/18993 (page, answers). Plan: docs/research/friend-build.md section 10. Tags: V = run, output seen; S = offline test; U = not run. Everything here used a THROWAWAY Ed25519 key and a scratch feed; nothing was uploaded, the real feed host, the sandbox container and the release key were not touched.

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
| D3 update v1 to v2 | PARTIAL (V engine and shim, U page/CLI) | The signed update path works; `./didle update` and the banner need a running bot, which needs real MPQs. |
| D4 update while the hero fights | BLOCKED (U) | No real game data, so no hero, so no fight. Not simulated. |
| D5 bad releases refused, rollback | PASS (V) | Six refusals and two rollbacks, old version kept running every time. |
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

### D4 update while the hero fights (U, blocked)

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

## Findings

1. FIXED here: `./didle update` printed nothing and exited 56 while only the setup server runs (tools/friendpack/didle).
2. Proposed card (container, small): `/opt/didle/boot` started with no arguments in a running container starts a second launcher and overwrites `/data/didle/boot.pid`, after which `check-now` answers `ERROR pid N is not the boot shim` and the next update is rolled back. A friend would only hit it by running the binary by hand; a refusal when another shim runs would be enough.
3. Proposed card (container, small, cause U): after the crash-rollback of 0.1.4 a `rollback-now` went to the base 0.1.0 instead of the previous 0.1.1. A clean chain (0.1.5 to 0.1.6) rolls back one step correctly, so the "previous" pointer is probably dropped by the failed pending release.
4. Proposed card (ops, small): `feed-publish.sh` and `d2feed` only accept https, so a LAN drill must rewrite the tree afterwards (or use the DEV loop). A `--allow-http` flag for scratch drills would remove the sed step; the published feed must stay https.
5. Proposed card (drill, needs the user's go): D3 page/CLI and D4 with a real hero on a throwaway character and a read-only game folder.
