# Self-update client (Tier 2, part 2)

> status: CURRENT for the code; nothing here has run against real GitHub or in a real container. Design: docs/research/self-update-design.md 3.4 to 3.8. Format and signing: docs/self-update-release.md. Task: TASK-122 (updclient).
> Tags: V = proven by the offline tests named here (fake GitHub through httptest); U = unverified live.

## Package `internal/selfupdate`

| File | What |
|---|---|
| `manifest.go` | schema, canonical bytes, keys, Ed25519 verify, versions (TASK-121) |
| `fetch.go` | releases API with `If-None-Match`, rate-limit detection, size caps, host allow-list (default `api.github.com`, `github.com`, `*.githubusercontent.com`, also enforced on redirects), `User-Agent: didle-updater/<ver>`, inbox loader |
| `store.go` | safe tar.gz unpack (plain files and directories only; absolute, `..`, links, devices refused; size and entry caps), side-by-side `releases/<ver>`, atomic `current` symlink swap, prune |
| `state.go` | `state.json` (current, previous, pending, bad list, `seq_floor`, `version_floor`, ETag) and `rollback.json`, written atomically |
| `engine.go` | state machine `idle, checking, downloading, ready, applying, error`; Check, Download, Apply, Cancel, Rollback, Inbox, daily Run |

Rules the engine enforces (V, `engine_test.go`): the signature is checked before anything else is believed (bad signature, tampered manifest and forged REVOKED are refused and the ETag is not stored); a revoked version (manifest block or signed `REVOKED`) is refused and a signed minimum version raises the monotonic floor; a `seq` or version below the floor is a downgrade and is refused; a version already rolled back is not retried; a hash or size mismatch clears staging and swaps nothing; `min_image` above the image ABI is offered as "image rebuild needed" and cannot be downloaded; `data_schema` must equal the install's (no migrations yet, U), `bot_api` and `d2d_ctl` may not exceed what the install speaks; Apply needs `confirm=<version>` and a safe moment (`IdleSafe`) unless forced; a failed shutdown after the swap points `current` back, marks the version bad and writes `rollback.json` (the banner shows it until dismissed); the last two releases are kept; with checks off the client makes zero outbound calls (V, call counter) and the inbox still works offline with the same verification.

Not done here: the boot shim that runs `current`, health-checks it and rolls back on a crash (TASK-123, `cmd/didleboot`); what "safe" means for the hero (`IdleSafe` is nil in `cmd/d2bot`, so Apply is allowed at any time; the page asks first and Force is a body flag, U); a disk-space probe (`DiskFree` is nil in `cmd/d2bot`).

## Watch screen (`internal/watch/update.go`, `web/update.js`)

- Viewer (GET, behind the same gate as the rest): `/api/update` and `/api/update/status` (JSON of `selfupdate.Status`), `/watch/update.js`.
- Answer listener (POST, token + origin check, JSON body with known fields only): `/api/update/check`, `download` (runs in the background; poll the status), `apply` (body `{"confirm":"<version>"}`, 409 `not-safe` when the hero is busy), `cancel`, `rollback`, `inbox`, `seen`.
- The banner is added by `ServeOptions.Update`: the script tag is injected into the `/watch` HTML pages so the existing pages and their golden files are untouched. Wording: "Update available: X" with Download, then "downloaded and verified" with Install and restart. Nothing downloads or applies without a click.
- Wiring (`cmd/d2bot/update.go`, one line in `ladder.go`): present only when the update root exists (`D2BOT_UPDATE_DIR`, default `/data/didle`) and at least one `*.pub` is in `D2BOT_UPDATE_KEYS` (default `/opt/didle/pubkeys`). `D2BOT_UPDATE_CHECK=off|0|false|no` disables every outbound call. `D2BOT_IMAGE_ABI` is the image's ABI level (TASK-123 sets it in the image). A daily check starts two minutes after boot. Apply stops the bot with SIGTERM, the same path `docker stop` takes (hero saved), and the container restart policy brings the new release up (U: needs the boot shim and a real container).

## Untested live

Everything against real GitHub (ETag behaviour, the redirect host list, rate-limit headers), the SIGTERM apply in a container, the banner in a browser, and the interplay with the boot shim.
