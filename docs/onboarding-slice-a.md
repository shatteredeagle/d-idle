# Onboarding slice A: setup stepper, go-ahead and access

> status: BUILT offline (task TASK-44, card onboardA, 2026-11-06). Design and mock: [research/onboarding-ux-design.md](research/onboarding-ux-design.md), [research/mocks/onboarding.html](research/mocks/onboarding.html); the user approved the mock and its five recommendations on 2026-11-06. Slices B and C: TASK-48 (tour, Help, banners), TASK-49 (Pause, Stop, Reset, report hook).
> Tags: V = verified in this job (test or run named), S = secondary, U = unverified.

## What was built

| Where | What | Tag |
|---|---|---|
| `cmd/d2dsetup/page/` | six screens: welcome, go-ahead, game files, hero, access, start (plus the done view); a "Why we need this" box on each, Back, plain errors with a fix line and Retry, `onboard.css` on the tokens of `web/style.css`, no inline script or style, text through `textContent` | V (`TestPageIsXSSSafe`; looked at in a headless browser at 390 and 1280 px, welcome, hero and access screens) |
| `cmd/d2dsetup/wizard.go` | stored go-ahead tick with its time (`ahead_at` in `wizard.json`), start time (`started_at`), phases `ahead, data, character, access, start, running, failed, done`, warm-up progress | V (`TestHappyPath`, `TestGoAheadEnforcedAndStored`, `TestProgressSurvivesClosedPage`, `TestInterruptedStartIsResumable`) |
| `cmd/d2dsetup/check.go` | file verdicts: ok, unconfirmed (hash not pinned), wrong_version, resurrected (no `.mpq` at all), missing, too_small, not_mpq, unreadable, problems (mixed), no_folder; each with a fix sentence | V (`TestFileVerdicts`) |
| `internal/access` | the access record `/data/access.json` (0600), PBKDF2 hash, session cookie value, failed-login limiter | V (`internal/access` tests) |
| `internal/watch/gate.go`, `serve.go` | passcode login at `/login`, cookie gate on the viewer and the answer listener | V (`gate_test.go`, `TestWatchServePasscodeAccess` in `cmd/d2bot`) |
| `cmd/d2bot/ladder.go` | `--watch-access-file` (env `D2BOT_WATCH_ACCESS_FILE`) | V |
| `cmd/d2dsetup/reset.go` | `d2dsetup reset-access` | V (`TestResetAccess`) |

## The six screens against the mock

Same order, copy and states as the mock. Differences: the stepper shows the step name only for the current step on a phone (as in the mock) and always on 560 px and wider; the file list hides the optional files behind one line "not needed"; on the start screen the progress bar is an estimate from the stored start time (120 s, U: unmeasured, `warmEstimate` in `wizard.go`), the three steps are honest (only the first is done by this process, the other two happen when the bot takes over); a failed start shows the error, the fix line and Retry but not the mock's "details" fold or Report button (those belong to slice C, the report). Welcome has no server state: it shows on a fresh visit and a returning tester resumes at the furthest step.

## Access: passcode or token

- Passcode: 6 or more characters (counted as characters, not bytes), at most 128 bytes, no composition rules, typed twice. Stored as PBKDF2-HMAC-SHA256, 600000 iterations (S: the OWASP 2023 figure for this PRF), 16 random bytes of salt per record, 32 byte output, using `crypto/pbkdf2` of the Go standard library (Go 1.24 and later; no hand-written helper needed). Go has no scrypt or argon2 in the standard library and `go.mod` stays untouched. PBKDF2 is not memory-hard, so a stolen `access.json` is cheaper to attack than an scrypt one; the file is 0600 in the data volume and online guessing is rate limited. The iteration count is in the record, so it can be raised later without breaking old files.
- Token: the generated 32 character token of testerwatch (`watch.token`), as before. `access.json` then holds only `{"kind":"token"}`.
- Gate: with a passcode record the viewer accepts only the session cookie; `?t=` and the bearer token are NOT accepted (so the token is not a back door once a passcode exists). With the token kind, or no record, the old rules apply unchanged (`tokenGate`; its tests were not touched). With no token and no record the screen answers 401 "access is not set up yet" (closed, never open). The record file is re-read when its time or size changes, so a reset needs no restart.
- Cookie `d2dpass`: value is an HMAC (SHA-256, keyed by the stored hash) of a fixed string, so a new or reset passcode kills all old cookies; `HttpOnly`, `SameSite=Strict`, `Path=/`, 30 days. `Secure` is NOT set: the screen is plain HTTP on a home network and a Secure cookie would never be sent back. Anyone who can read that network's traffic can read the cookie; the guide says never to publish the port on the internet. The cookie is per host, not per port, so the login on 8891 also reaches the answer listener on 8892 (V, `TestWatchServePasscodeAccess`).
- Rate limit: 5 failed logins within a minute lock that client (peer address, proxy headers are not trusted) for 5 minutes; the reply is 429 with `Retry-After`, even the right passcode is refused while locked. A success clears the count. In memory only. The passcode is never logged (the server writes no request log at all) and never returned.
- The login form posts to `/login` (cross-origin Origin refused), CSP `default-src 'none'; style-src 'self'; form-action 'self'`.
- During setup the page itself stays behind the setup token (the claim-race rule of the design); after the passcode is chosen, the session cookie also opens the setup page again.
- Reset: `docker compose -f compose.tester.yml exec d2dbot d2dsetup reset-access` switches to the generated token (created if missing, never printed: the command names the file); `... exec -T d2dbot d2dsetup reset-access -passcode` reads a new passcode (one line) from standard input. Both take effect at once.

## What the container still has to do (not in this task, `deploy/` is another task's)

1. Ship `d2dsetup` in the image and run it before the bot (hook text in `cmd/d2dsetup/README.md`; context.sh builds `./cmd/d2dsetup`).
2. Export `D2BOT_WATCH_ACCESS_FILE=/data/access.json` in the tester profile of `entrypoint.sh`.
3. Print the first-run URL with the token as in the README (G3 of the design).

## Not proven

Offline only. Not run against a real container, a real install (sizes, the 1.14d fingerprint is still not pinned, so "confirmed" cannot appear), a phone, or a screen reader; the files, error and running screens were not looked at in a browser (the welcome, hero and access screens were). The 120 s estimate is a guess. Cookie behaviour across ports was tested with Go's cookie jar, not a real browser.
