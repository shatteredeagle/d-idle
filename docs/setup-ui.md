# Setup wizard UI (WIZARDUI)

Files: `web/setup.html`, `web/setup.js`, wizard styles at the end of `web/style.css` (prefix `wz-`, theme tokens shared with the dashboard), `web/mock/setup_mock.py`. `web/embed.go` embeds the two new files. Serving `/setup.html` and mounting `setup.API.Handler()` at `/api/setup/` is the server owner's job (not done here).

Flow: welcome -> Check my computer (`GET /checks?pre=1&quick=1`, full check on demand) -> game files form (`PUT /inputs`, then `POST /start`) -> progress (polls `/state` every 1.2 s while running or waiting, 5 s idle) -> character form (shown when `current` is `character` and pending) -> done. Which screen shows is derived from `/state`, so a refresh resumes; a stale `#hash` is ignored when a run is active. Failed step: Try again (`POST /reset?step=ID` then `/start`), Change my answers (sources and data-dir), Start over (two-click, `POST /reset`).

Rules kept: no CD key field anywhere; copy says the key is typed only in the game's installer; the form refuses text shaped like a CD key; only the documented input fields are sent. Path validation is client-side (absolute path) plus backend errors: `PUT /inputs` 400/409 text is shown in the form, and the `sources` step `error` is shown on the progress screen.

Interface notes: `Start` runs through `character` in the real backend; the UI asks for name and class only when the step is pending and nothing runs, so the backend should stop before `character` until `charName` is set (the mock does), otherwise the character screen is skipped and the default name Setupbot is used. `display` is shown as text unless it is an http(s) URL, then it becomes an "Open the installer window" link. Status (U): not run against the Go backend, only the mock.

Test: `python3 web/mock/setup_mock.py --port 8440 [--speed 3] [--scenario happy|fail|gamedir]`, open `/setup.html`. `POST /mock/scenario?name=` resets it. Verified with Playwright (Chromium) at 390 and 1280 px, light and dark, every screen: no console errors, no horizontal overflow.
