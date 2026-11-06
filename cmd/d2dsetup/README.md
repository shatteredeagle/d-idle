# d2dsetup: first-run wizard of the d2d tester container

Standalone, stdlib only, no Wine. Task TASK-32 (card testerd2dwiz). Plan: `docs/research/tester-build-plan.md` gap G2.

## What it does
Steps (the page derives the step from `/data`, so a refresh or a restart resumes):
1. **Game files**: lists `/game` case-insensitively. Required: `d2data`, `d2exp`, `d2char`, `Patch_D2` (`.mpq`); optional: sfx, music, speech, xmusic, xtalk. Per file: present, regular, readable, MPQ header (`MPQ\x1a`), size above a loose lower bound (U: bounds are guesses, not pinned to a real install). One plain message per problem naming the file; a folder with no `.mpq` at all is told it needs classic LoD 1.14d, not Resurrected. 1.14d proof: SHA-256 of `Patch_D2.mpq` against `tables.json` `patch_sha256`. That list is EMPTY (U: no real 1.14d hash is verifiable in the repo), so the verdict is "unverified" and the hash is shown and stored; pin it from the owner's copy at the live run. A non-empty list that does not match fails the check. "Check again" re-runs it.
2. **Character**: name (letters only, 2 to 15, same rule as `validCharName` of the Wine wizard), class (Sorceress default; the others are listed as "not supported in this build"; table `tables.json`, `d2_index` in d2d UI order V docs/d2d-fidelity.md), goals preset (`campaign:normal_act1`, each preset is validated with `fleet.Spec.GoalsArg`). Choices only; the only free text is the name. Unknown JSON fields are a 400 (a `cd_key` field is never stored), there is no CD key field.
3. **Start**: re-checks the files, warms the game table cache (`internal/data.Open` into `<d2bot data root>/cache/tables`, the directory d2bot reads), then writes `wizard.env` and, last, `wizard.json` (phase `done`, 0600). A failed extraction shows the error and can be retried.

Builds are data: `tables.json` holds files, classes, presets and hashes.

## Where it is served, token
Its own listener (default `0.0.0.0:8891`, `-addr`/`D2SETUP_ADDR`) in its own process, started BEFORE the bot, exiting (20 s after done, `-linger`) before d2bot binds 8891 for the watch screen. No coupling with the watch code. EVERY request needs the watch token (bearer header, or `?t=` which sets the watch screen's own `d2dview` cookie and redirects without the token), loopback included. The token is the watch token file (`-token-file`, default `/data/watch.token`, 32 characters, 0600, created when missing exactly as `cmd/d2bot/ladder.go` does), so the browser that passed the wizard is already signed in on `/watch`. The URL with the token is printed once, on creation. Writes also need a same-origin `Origin` and a JSON content type. The done view returns the token once (flag in `wizard.json`); the page shows it and polls `/watch` until the bot answers, then goes there. CSP `default-src 'self'`, all text via `textContent`.

## Entrypoint hook (NOT applied: `deploy/` belongs to another task)
In `deploy/d2d-container/entrypoint.sh`, tester mode, before the `d2data.mpq`/`d2exp.mpq` check (the wizard reports missing files itself):
```sh
if [ "$mode" = tester ] && [ "$dry" = 0 ] && ! grep -q '"phase": "done"' /data/wizard.json 2>/dev/null; then
	echo "entrypoint: first run: open http://<host>:8891/setup.html (token: see below)"
	d2dsetup -data /data -game "$D2D_DATA" -origin "${D2BOT_WATCH_ORIGIN:-}"   # blocks until Start finished
fi
[ ! -f /data/wizard.env ] || . /data/wizard.env     # D2D_NAME, D2D_CLASS (d2d index), D2BOT_GOALS
```
`d2dsetup` itself exits 0 at once when `wizard.json` says done, so the grep is only an optimisation. `context.sh` must build `./cmd/d2dsetup` into the image (next to d2bot, d2dview) and the tester process must run it as the container user (`/data` is theirs). A stop (SIGTERM) during the wizard exits non-zero, the container restarts and resumes from `/data`.

## Not proven
Offline only: fake MPQs, fake extractor. Not run: the real MPQs (sizes, 1.14d hash), the real `data.Open` on a real install (only its failure path on fake files), the page in a browser (`node --check` only), the handover to the real watch screen on 8891.

## Slice A (onboardA, 2026-11-06): six screens, go-ahead, access
The page is now welcome, go-ahead, game files, hero, access, start (docs/onboarding-slice-a.md). New state in `wizard.json`: `ahead_at` (the go-ahead tick with its time; Start is refused without it) and `started_at`; new routes `PUT /api/ahead` and `PUT /api/access` (passcode or token; the hash goes to `/data/access.json`, 0600). `d2dsetup reset-access [-passcode]` is the way back in. Hashing: PBKDF2-HMAC-SHA256 from the standard library, 600000 iterations. The watch side is `internal/watch/gate.go` and `d2bot --watch-access-file`.
