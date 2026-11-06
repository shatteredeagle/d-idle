# Onboarding UX for the d.idle tester build (design, before any code)

> status: approved by the user 2026-11-06 (mock and recommendations); slice A built (docs/onboarding-slice-a.md).

> status: DESIGN with a clickable mock, for the user to react to. No product code changed. Mock: [mocks/onboarding.html](mocks/onboarding.html) (open it in a browser; the "Jump" box at the top goes to every screen and state). Written 2026-11-06 (agent onboardux) after the user asked whether the onboarding flow had been designed; it had not past the first-run wizard (setup of cmd/d2dsetup, task testerd2dwiz).
> Tags: V = read in this repo or in the Palimpsest project (file named), S = secondary, U = unverified or my judgment. Classic 1.14d; nothing here is D2R. Inspiration: Palimpsest (the user's own standalone SETI-style contributor container, GPL-3.0-or-later like this repo): its audience rules, onboarding wizard and dashboard. The copy below is written fresh for d.idle; no Palimpsest text or code was copied.
> Parents: [tester-build-plan.md](tester-build-plan.md) (tasks 23 to 33), [watch-screen-design.md](watch-screen-design.md), [orders-ux-design.md](orders-ux-design.md), [automation-ladder.md](automation-ladder.md), [ADR 0016](../adr/0016-automation-ladder.md), `cmd/d2dsetup/README.md`, `docs/d2d-container/README.md`.

## 1. What exists and what is missing (V)

- V `cmd/d2dsetup` (merged today): a three-step first-run page (game files check, character with Sorceress locked and a goals preset, Start that warms the table cache), state derived from `/data`, resumable, no CD-key field, token gate, the token returned once at the end (`cmd/d2dsetup/README.md`, `wizard.go`). Its page is `cmd/d2dsetup/page/index.html` and `app.js` (145 lines); it has one friendly hint line and no stepper, no "why we need this", no go-ahead, no access choice (grep of `app.js`; Back is not offered, U: I did not run the page).
- V `internal/watch/web` (watch.html, app.js, app.css): Game, Thoughts, Orders (map), Rules, Journal, state bar, toasts. No tour, no help, no sample thought, no report button, no pause/stop control (grep for tour, sample, help, report, pause, stop in the page code found nothing).
- V Docs for the tester: README text in `docs/d2d-container/README.md` (env table, "token printed once in the log"); the guide `docs/tester-guide.md` is task testerdocs (not written). The report bundle is task testerreport (not built).
- Missing, and the subject of this doc: the first 60 seconds, the explanation and consent screens, the access choice, the first 10 minutes of play, Help, Report, and the states in section 6.

## 2. What we borrowed from Palimpsest (V, its DESIGN.md "Audience", "Onboarding wizard", "Dashboard", `onboarding.py`, `passcode.py`, `gamefiles.py`, `web/static/index.html`)

| Palimpsest idea | Our version |
|---|---|
| One command, one volume, the log prints the URL | `docker compose up -d`; the log prints one URL (section 3) |
| Progress stepper with a "why we need this" box on every step, Back, resumable | Same shape, six steps, one fixed "Why we need this" box per screen |
| Plain-English errors that carry a fix hint and a Retry | Every red box has a bold problem, one fix sentence, a Retry or Check again button |
| Honest time estimates on heavy steps, close the page and come back | The one slow step (table warm-up) says "1 to 3 minutes (U)", survives a closed page |
| Go-ahead page that states the rules in plain words and needs a tick | Screen 2, with what runs, what is stored, how to stop |
| Game files checked by hash with friendly guidance, never uploaded | Screen 3, one plain line per file, a hash verdict, a wrong-version message |
| Passcode chosen by the user, scrypt-hashed, resettable by a command; a one-time setup token in the log for claim-race hardening | Screen 5 offers a passcode (recommended) or the generated token; section 5 |
| "Report a problem" builds a redacted bundle the user reads first | Section 7, built by testerreport, with a preview |
| Big status, activity feed, Pause/Resume, "Show details" for logs | The watch screen already has the first two; we add Pause/Stop/Reset and a Help button |
| Updates are a notice only, never automatic | Upgrade banner (section 6) |
| Not borrowed | Claude and GitHub sign-in, approval queue, usage governor: d.idle has none of them |

## 3. The first 60 seconds

Tester prerequisites (from the tester plan, V): Docker on Linux x86_64, their own classic Diablo II folder (d2data, d2exp, d2char, Patch_D2 at least), about 2 cores, 3 GB RAM, 5 GB disk (U).

1. They unpack the pack, edit two lines of `compose.tester.yml` (`D2_GAME_DIR`, `D2_WATCH_ORIGIN`), and run `docker compose -f compose.tester.yml up -d` (V `docs/d2d-container/README.md`, tester profile). The goal for later: one command with the two values as flags in a helper script (U, not designed).
2. They run `docker compose logs -f`. The log must say, in this order and in plain words (proposed, replaces today's token line):
```
d.idle tester build 0.4.2 (engine d2d, classic Diablo II 1.14d)
First run. Open this address in your browser:
    http://localhost:8891/setup.html?t=<one-time token>
Other computer or phone? See "Other devices" in the guide.
Your game files are read-only. Nothing leaves this computer.
```
   The token is the existing watch token (V: 32 characters, `/data/watch.token`, printed once on creation, `?t=` sets the cookie and strips itself, `cmd/d2dsetup/README.md`). On every later start the log prints only `d.idle is running: http://localhost:8891/watch` with no token.
3. The tester opens that URL. The page opens on screen 1 of the stepper. Time from `up` to a usable page: seconds (the wizard starts before the bot, V README "Where it is served").
4. If they open the URL without the token the page says "This link needs its token. Copy the whole address from the log, or run `docker compose exec d2dbot cat /data/watch.token`" (V: every request needs the token today; the friendly page for a missing token is new, U).

## 4. The setup stepper (six screens)

Common frame (mock: setup screens): a header, a stepper of six segments (number and name; on a phone only the current name shows), one card, a "Why we need this" box, Back on the left, the main button on the right. The step is derived from `/data` so a refresh or a restart resumes, as today (V). Back never loses what was typed (the page keeps the inputs; U for the exact storage, `PUT /api/inputs` exists, V `server.go`). All text via `textContent`, CSP as today (V).

| # | Screen | What it says and does | Why we need this (the box) |
|---|---|---|---|
| 1 | Welcome | Five plain sentences: an idle Diablo II; the hero plays alone; it asks you questions and your matching answers teach it rules; offline single player only, no Battle.net, no CD key, no account; you bring your own game files, about 5 minutes. Button: Get started. | So you know what you are starting before anything is checked or stored. |
| 2 | Go-ahead | What runs (one container: the engine d2d and the player d.idle, nothing elsewhere); your game files stay here, read-only, never uploaded, never in the image or the project; offline only; what we store (save, answers, learned rules, logs, passcode scrambled) and that there is no telemetry; how to stop (Stop button or `docker compose stop`, the hero saves first; Reset deletes only the hero); early build, the hero can die or get stuck. A tick box "I understand and want to continue" enables Continue. The tick is stored with the date in `wizard.json` (U). | We would rather you decide with the facts than find out later. |
| 3 | Game files | One row per file: name, a plain verdict (OK, Missing, Problem, Wrong version, Unconfirmed) and one sentence. Below it one verdict box. Buttons: Back, Check again, Next (disabled until the files pass or are unconfirmed). See 4.1. | The engine cannot run without the original data and we cannot give it to you; the version check saves you an evening with a hero that plays the wrong game. |
| 4 | Hero | Name (letters, 2 to 15, V `validCharName` rule), class (Sorceress shown as the only choice with the note "the other six are not taught yet in this build"), goal (a select of presets; recommended "Act 1 on Normal"). | The hero needs a name for its save and a goal to work on; you can change the goal later (U: changing the goal live is not built). |
| 5 | Access | Passcode (recommended) or the generated token; see section 5. | Anyone who can open the page can answer for the hero, and the answers change how it plays. |
| 6 | Start | A summary of the choices; the one slow job with a time estimate (4.2); on success "Sorra is awake" and the button Open the watch screen. | The one slow step, done now so the hero starts at once next time. |

### 4.1 Game files screen (V checks in `cmd/d2dsetup/check.go`; copy is new)

Verdicts, one sentence each (the mock shows all of them):
- Missing file: "Patch_D2.mpq: Not found in the folder. It comes with the 1.14d patch." Fix line: copy it from your classic Diablo II folder into the folder you gave the container, then Check again. No restart is needed (the check re-reads `/game`, V).
- Too small or not an MPQ: "Found, but smaller than expected (a copy that stopped half way?)". The size bounds are guesses today (U, README), so this message must say "looks" and never "is".
- No `.mpq` at all (V: today's check already treats this as "classic LoD 1.14d needed, not Resurrected"): the Resurrected box in the mock. D2R-only fact flagged: D2R keeps data in a different container and has no `d2data.mpq`; nothing else of D2R is used.
- Wrong version (hash of `Patch_D2.mpq` in the pinned list but not matching, or a known other-version hash): "These files are not from version 1.14d... install the 1.14d patch in your own game, copy the new Patch_D2.mpq here, then Check again." Next stays disabled.
- Unconfirmed (V today: `patch_sha256` list is EMPTY, so the verdict is "unverified" and the hash is shown): allowed to continue, with a yellow box and the fingerprint shown so it can go in the report. Once the owner pins the real hash from a real copy, the common case is the green "Version 1.14d confirmed".
- Optional files (sound, music, speech): never an error; one line "not needed: the hero plays silently" (V tables.json lists them optional).

### 4.2 The Start screen and the slow step

The one slow job is the table-cache warm-up (`internal/data.Open`). The screen shows a three-step list ("reading the game data tables", "starting the game engine", "creating Sorra"), a progress bar, "about 1 to 3 minutes, once" (U: unmeasured; testerlive measures it and the number is edited then), and "you can close this page; setup continues". A failure shows the plain red box with Retry and Report (the mock's "d2d not ready").

## 5. Access: passcode or token

Recommendation: offer both, default to a passcode the tester chooses.
- Reasons for the passcode (V Palimpsest uses it; the reasoning is mine, U): the tester will want the watch screen on a phone, and a 32-character token is hard to type there; a chosen passcode can be changed or reset; it survives reinstalls of the browser.
- Reasons for the token (V it exists today): zero choices, nothing weak can be chosen, and it is already wired as the gate and cookie of both the wizard and the watch screen.
- Rules for the passcode (U, borrowed in spirit): six or more characters, no composition rules, shown "four random words work well", stored only as a salted scrypt hash in `/data` (Go has no scrypt in the standard library; the project is stdlib only, so use `crypto/pbkdf2` from Go 1.24 or later instead, U which Go version has it; the repo's Go is 1.27, V AGENTS.md). Same-origin writes and the answer-route origin check are unchanged.
- Claim-race hardening: until a passcode exists, anyone who reaches the port could choose one, so the first-run URL carries the one-time token (as in section 3), exactly as Palimpsest prints a setup token.
- How a token user gets it again: `docker compose exec d2dbot cat /data/watch.token` (V README). How a passcode user resets: a `d2dsetup reset-passcode` verb run with `docker compose exec` (proposed, U).
- Other devices: the screen says plainly that the container must be published on the home network and started with the address the tester types, with the two compose lines in the guide, and that the page is plain HTTP, never for the open internet (V tester plan G8).

## 6. States (mock: Jump menu)

| State | What the tester sees | Source |
|---|---|---|
| No game files / wrong version / Resurrected | Screen 3 with the matching red box and no Next; can still go Back | 4.1 |
| d2d not ready | Screen 6 failure box: "the game engine did not start", likely causes (memory, port), Retry, Report; details fold open to the last log lines | 4.2; the 60 s wait is a guess (U) |
| Slow first-run steps | Progress bar and estimate; closing the tab is fine | 4.2 |
| First view of the watch screen | The tour starts (section 7); the setup page's "done" view returns the token once, V | README |
| Returning user | `/watch` directly, no tour, no banner; the container log says only the running URL; the wizard exits at once when `wizard.json` says done (V README) | V |
| Hero dead or recovering | A banner "Sorra died. That happens, and she is already recovering... nothing for you to do. It is in the Journal." plus the red state dot. Today the state bar shows a deaths badge and a recovery state (V watch.html, `uptime.go`); the plain banner text is new | U |
| Hero stuck / engine restarted | Same banner family: "The game engine restarted; the hero was saved." (U: needs a bot event) | U |
| Upgrade | An info banner on first view of a new version: "Updated to 0.4.2. Your hero and rules were kept. What changed." (links to CHANGELOG text). The wizard does not run again (same volume). If the new version needs new tables, the warm-up job reruns with the Start-screen progress (U) | tester plan G7 |
| Browser lost the token | A friendly "needs its token" page (section 3) | U |

## 7. The first 10 minutes: the guided tour

Today nothing happens when the watch screen first opens; a stranger sees five panels and no explanation. The tour fixes that and does not depend on a real event.

Trigger: the first time the watch screen opens after setup (flag `tour_seen` in `/data/wizard.json`, U where it lives; the browser may also keep a local copy). A returning user never sees it unless they press Help, Replay the tour. A banner offers Start tour or Skip, so it never blocks. The tour is a five-step coach card that dims the page and outlines the real panel it talks about (mock tour steps 1 to 5):

1. This is your hero. Game panel. "The picture is the real game, live... You do not need to click anything here." (V the Game panel streams the video, `tapcapture.go`.)
2. It asks you things. Thoughts panel with a SAMPLE card (badge "Sample"; the page injects it locally, it is not in the Hub's data and answering it changes nothing). The card is the real card layout: headline, "If you do nothing: ..." line (V `cardtext.go` already writes it, with the fade seconds), the "Why am I being asked?" fold, fixed choices plus "Not sure, you decide", and the fade bar. The coach text explains: this is a moment where a veteran player would choose; the hero carries on with its default; ignore it and it fades, the Journal notes you missed it, nothing bad happens (V ADR 0016 item 10 and 11: fade, default, "missed" journal line). The sample does not fade while the tour is open.
3. After a few matching answers it makes a rule. Rules panel showing one SAMPLE rule with Revoke: "Give the same answer in the same situation 3 times in a row (2 for safe things, 5 for risky ones)... it tells you first" (V ADR 0016 item 12: N is 2, 3 or 5; confirm card before a rule, V watch-screen-design).
4. You can send it somewhere. Orders map: pick an area and minutes. "You are not driving: it still drinks potions and runs away when it should" (V orders design: guard decides on arrival). Optional: a "Try it" hint without a forced order.
5. The Journal says what it did and why. Journal panel; "tap a line to ask why" (V `/api/why`). Ends with Done and "Help is always at the top".

Behaviour: Next, Back, Skip tour on every step; progress dots; the current step number is stored so a closed tab resumes at that step; Replay from Help. On a phone the coach card is a bottom sheet and the tour switches the tab strip to the panel it describes (mock); on desktop all panels show and the card moves aside so the outlined panel stays visible. Keyboard: the coach card is a dialog with focus inside, Escape skips, Tab order Next last-visited.

What happens on the first real thought: after the tour a toast says "Your first real question will appear in Thoughts" when a real thought arrives for the first time (U; the live bot asks only 9 of 22 decision points and the T0 live drill answered 0 thoughts, V tester plan G10, so the guide and Help must say "questions can take a while; the hero is not broken").

Realtime caveat (V tester plan G6): realtime play of the ladder has never run with a person answering; the tour's claims about fading assume `watch.Config.Speed` 1, which is the default (V `hub.go`).

## 8. Help and Report

Help (button at the top of both pages, always visible; mock Help sheet):
- What the numbers mean: Uptime (the share of time the hero made progress, not counting dead, recovering or stuck, V `uptime.go`), Deaths, "Fades in N s".
- Words in plain language, one line each: Thought, Rule, Order, Journal, Goal, Risk (safe/normal/risky = 2/3/5 answers). Terms follow GLOSSARY.md and ADR 0016; the Help text is its own, never a GLOSSARY copy (GLOSSARY has no entries for thought, rule or order, so the coordinator may add them, section 11).
- Replay the tour, Report a problem, and three common fixes (stuck hero, page will not load, questions are rare).

Report a problem (mock Report sheet; built on task testerreport, V tester plan G10 and task 27):
- One optional text box ("What went wrong?"), then a plain list "Included" (logs of the last two hours, versions, setup choices with the passcode/token only as a kind, the ledger line with counts and `writes after start`) and "Left out" (passcode, token, game files, folder paths, host names and addresses), then "Preview the file" showing the file tree and a few lines.
- Output: a tarball saved by the page or by `docker compose exec` (the script exists as testerreport); nothing is sent anywhere. The tester shares it themselves.
- A test greps the bundle for the token, key-shaped text, home paths and host names (V plan). Whether the page can build it, or only show the command, depends on the report script living in the container (U: recommend a small authenticated `POST /api/report` that runs the script; slice 2).

Pause, Stop, Reset (mock "Pause / Stop"): Pause stands the hero still, safe; Stop saves and shuts the engine (up to 45 s, V stop grace in README); Reset needs the word RESET typed, deletes the hero, rules and journal but not the game files or the passcode, then setup runs again. U: today the bot has no pause API through the watch listener; check `cmd/d2bot` for a control route before promising Pause (gap G7).

## 9. Phone first and accessibility (V pattern from `style.css` and the watch page)

- Phone width 390: single column, tab strip for the five panels, bottom sheets for the tour, Help, Report and Stop; every tap target at least 44 px high (the mock's buttons are 44; the watch page's `.btn` is 36, a gap). Desktop (900 and up): the panels sit in a grid and the tab strip hides.
- No horizontal scroll at 390 or 1280 (checked on the mock with a headless browser: scroll width equals client width in all 15 states).
- Colour is never the only signal: verdicts carry words (OK, Missing...), state dots have a text label; contrast uses the d.idle tokens, light and dark and `prefers-color-scheme`.
- Keyboard: skip link, visible focus ring (V `:focus-visible` in `style.css`), dialogs trap focus and close on Escape (U: the mock does not trap focus; the build must). Live regions announce step changes and toasts (V setup.js `wz-live`). `prefers-reduced-motion` stops transitions. Text via `textContent`.
- Text is plain: reading level about grade 6, no jargon without a gloss; "d2d" appears once with "the game engine".

## 10. Gap list (what this changes)

| # | Where | Change | Tag |
|---|---|---|---|
| G1 | `cmd/d2dsetup/page/index.html`, `app.js` | six-step stepper with a why box, Back, welcome, go-ahead, access, richer game-file rows, progress view; today three steps | V gap |
| G2 | `cmd/d2dsetup/wizard.go`, `server.go`, `tables.json` | state for `ahead_ok` (date), access kind, passcode hash; routes `PUT /api/access`, `POST /api/reset-passcode`; unknown-field rejection kept; hash verdict with a wrong-version list | V gap |
| G3 | `cmd/d2dsetup/main.go` and the entrypoint hook | first-run log text with the token URL (section 3); later starts print only the running URL | V gap |
| G4 | `cmd/d2bot/ladder.go`, `internal/watch/serve.go` | watch gate accepts a passcode cookie as well as the token; the wizard and watch share one credential file | U |
| G5 | `internal/watch/web/app.js`, `watch.html`, `app.css` | tour overlay, Help sheet, Pause/Stop/Reset sheet, Report sheet, state banners, upgrade banner, 44 px targets | V gap |
| G6 | `internal/watch` (hub or a new small module) | sample thought and rule served locally by the page (no Hub change) or a `sample:true` thought; `tour_seen` flag and step in `/data`; `GET/PUT /api/onboarding` | U |
| G7 | `cmd/d2bot` | pause/resume and stop routes on the answer listener (POST, origin-checked) | U |
| G8 | `deploy/d2d-container/entrypoint.sh`, testerreport | report route or command; "needs its token" page | U |
| G9 | `docs/tester-guide.md` (testerdocs) | the first-60-seconds text, the other-device compose lines, the expected waits | V task exists |

## 11. Implementation plan (three slices)

| Slice | Content | Hours | Absorbed by |
|---|---|---|---|
| A: setup stepper | G1, G2, G3 and the access choice (passcode hash, token path) | 8 | extends testerd2dwiz (TASK-32, merged; a follow-up card) and the log text in testerd2dimage (TASK-31) |
| B: tour, Help, state banners | G5, G6 (page-side sample, `tour_seen`), 44 px targets | 8 | NEW task 1 (watch screen owns `internal/watch/web`) |
| C: controls and report | G7 pause/stop/reset routes, G8 report route and page, guide text G9 | 7 | testerreport (TASK-27) and testerdocs (TASK-26) absorb the report and guide; NEW task 2 covers the controls |

Total about 23 hours. Proposed NEW tasks (not created; the coordinator creates them): (1) `onboardtour: tour overlay, Help and state banners on the watch screen` owns `internal/watch/web`, about 8 h; (2) `onboardctl: pause, stop and reset routes plus the passcode gate` owns the bot side of G4 and G7, about 5 h. Slice A needs no new task if TASK-32's owner opens a follow-up.

## 12. Open questions for the user (recommendations)

1. Passcode or token as the default (section 5)? Recommend: offer both, default to passcode, because a phone user cannot type 32 characters.
2. Is the go-ahead tick stored and required (screen 2)? Recommend yes, stored with the date in `wizard.json`, never sent anywhere.
3. Should Pause and Reset exist in the tester build, or only Stop (compose)? Recommend Pause and Stop in slice C, Reset behind a typed word, because testers will want a clean restart and a command line reset scares non-technical friends.
4. Should the tour use an injected sample thought (page-only, section 7) or a real tagged thought from the bot? Recommend page-only: it works before any real event and cannot teach the hero anything wrong; the real first question gets a toast.
5. Is the Help text allowed to define terms without adding them to GLOSSARY.md? Recommend yes for now (plain words for testers), and the coordinator adds Thought, Rule, Order, Journal to the glossary when ADR 0016's words are stable.

## 13. Not verified

The mock is a static sketch: no real backend, the passcode check is fake, the time estimates (1 to 3 minutes, 60 s engine wait, 45 s fade) are placeholders except the fade seconds, which are the Hub's `FadeSec` (V `cardtext.go`). None of this ran against the real container. The 1.14d fingerprint is not pinned (V README), so the "confirmed" verdict cannot appear until the owner pins it.
