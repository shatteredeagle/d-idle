# Watch screen design (TASK-20, ADR 0016 items 20 to 23)

> status: built offline on the simulated runner, not verified live. Code: `internal/watch` (backend and page), `cmd/d2dview` (`-watch`). Tags: V = proven by a test named here, S = read in a doc or source, U = unverified. Story and idle text is placeholder, our own words (docs/contracts/licence-policy.md).

## 1. Layout (one screen, `/watch`)

```
+------------------------------------------------------------------------------------+
| d.idle  [state: fighting in Blood Moor]  uptime 91.2% (last 10 min 88%)  2 deaths  t=12:40 |
+--------------------------------------------+---------------------------------------+
| GAME  (viewer frame stream, or a note)     | THOUGHTS                              |
|                                            |  Question card: text, plain context,  |
+--------------------------------------------+  urgency line, "Meanwhile I: ...",    |
| ORDERS  [Go to|Fight here|Come home]      |  fixed buttons + "Not sure, you       |
|   [area v] [10] min  (Give order)          |  decide", fade bar + "fades in 31 s"  |
|   Go to Blood Moor for 10 min - active  x  |  Confirm card: "From now on I will    |
+--------------------------------------------+  always X here (context). OK?" Yes/No |
| RULES   (Revoke all)                       |  Feed lines: idle talk, incidents     |
|   "When asked Q (context), I do X." [Revoke]|  (with the rules in force + Revoke),  |
|   revoked rules stay, struck through      |  rule learned, too late, order changes|
+--------------------------------------------+---------------------------------------+
|                          JOURNAL (permanent, newest first)                         |
|  12:10 Missed: Q (context). It faded ...        12:02 You answered "drink": Q   why did you do that? |
+------------------------------------------------------------------------------------+
```

- Game area: `<img src=/stream.mjpg>` when d2dview has frames (`config.frameUrl`), else a note. The simulated runner has no frame.
- Thoughts: only open, shown questions. A thought leaves the feed when it fades (no ghost); the fade bar uses `remainingSec` plus the page's wall clock times `speed`. A thought the ladder holds during a fight is not shown (V `TestSinkHoldsThoughtsDuringAFight`).
- Plain language: every context field has a phrase (`humanize.go`, V `TestEveryFieldValueHasAPhrase` walks the whole catalogue, so a new band fails until it has its words). Under a fight-time question one urgency line: the urgency band, else the life band (the stage-1 stand-in of the ADR amendment). The urgency inputs (potion restore, recent damage) are NOT shown: the store does not carry them (see API needs).
- Orders (T0 list of docs/research/automation-ladder.md): `go` (area from a fixed list plus minutes 1 to 240), `fight` (here, minutes), `home`. At most 5 open.
- Uptime = progress / (progress + death + recovery + stall); time with no order is not counted (V `TestUptimeLowersOnDeathRecoveryAndStall`, two builds).
- Rules list: plain sentence, risk class, revoke one, revoke all (the page asks first, the route needs `confirm:true`). A revoked rule stays listed. No code path revokes by itself (V: only `Hub.Revoke` and `RevokeAll`, called from the answer route).
- Journal: the store's answer log (persists with the profile) plus the Hub's lines (idle, incident, order, learned, too late; in memory only). Missed questions are `missed` lines (V `TestFadeWritesMissedToJournal`). A line with a context key carries "why did you do that?": `GET /api/why` says rule / revoked / default, the streak (`2 of 3`) and offers revoke. After a death the incident names the rules in force with a Revoke next to each (V `TestDeathNamesTheRulesAndOffersRevoke`).

## 2. Data contract (JSON over HTTP, versioned)

Every body has `"schema": "watch.<name>/<n>"`; adding a field keeps the version, removing or renaming bumps it. Times are virtual seconds relative to the Hub's start. All GETs are `Cache-Control: no-store`, `nosniff`.

Read side, mounted on d2dview (GET and HEAD only, any other method is 405 with `Allow: GET, HEAD`, V `TestViewerIsReadOnly`, `TestWatchRoutesOnTheViewerAreReadOnly`):

| Route | Body (golden: `internal/watch/testdata`) |
|---|---|
| `GET /watch`, `/watch/app.js`, `/watch/app.css` | embedded page, CSP `script-src 'self'`, no inline code, no CDN |
| `GET /api/thoughts` | `thoughts[]` (id, kind question or confirm, question, key, context, fields, urgency, choices[id,label] incl. `notsure`, meanwhile, learn, expiresAt, remainingSec), `feed[]` (idle, story, incident with rule refs, order, learned, late), fadeSec, speed |
| `GET /api/journal?limit=&kind=` | `lines[]` (id, t, kind, text, thought, key, why, rules) oldest first |
| `GET /api/rules` | `rules[]` (key, point, question, context, fields, answer, label, risk, text, learnedAt, revoked), `live` |
| `GET /api/uptime` | state, label, uptimePct, windowPct, progress/death/recovery/stall seconds, deaths, text, `seq` |
| `GET /api/orders` | orders[id, kind, area, minutes, state, note, text], areas, kinds, maxMinutes |
| `GET /api/why?key=` or `?line=` | source (rule, revoked, default), streak, needed, text, revoke flag |
| `GET /api/config` | where the answer route is (`answerPort` or `answerBase`), `frameUrl`, fadeSec |
| `GET /api/events` | SSE: `hello` then one `change` event per Hub change with only `{seq}`; the page re-reads the views. Polling fallback: compare `seq` of `/api/uptime` (V `TestSSEStream`) |

Write side, a SEPARATE listener (`-watch-answer-addr`, default `127.0.0.1:8892`), POST only, JSON only:

| Route | Body | Refusals (V `TestAnswerRouteValidation`, `TestAnswerRouteGuards`) |
|---|---|---|
| `POST /api/answer` | `{thought, answer}` | answer must be one of the point's ids or `notsure`, never free text (400 `bad-choice`, lists allowed); unknown id 404; already answered 409; faded 410 (a late answer also writes the "too late" line, nothing counts); a confirm id 400 |
| `POST /api/confirm` | `{thought, yes}` | same, on a confirm thought only |
| `POST /api/revoke` | `{key}` | no such live rule 404 |
| `POST /api/revoke-all` | `{confirm:true}` | without `confirm` 400 |
| `POST /api/orders` | `{kind, area, minutes}` | kind, area from the list, minutes 1 to 240, 5 open at most (400 / 409) |
| `POST /api/orders/cancel` | `{id}` | unknown 404, finished 409 |

Guards: `Content-Type: application/json` (415 otherwise), body at most 4 KB, unknown fields refused, `Origin` must be in `-watch-origin` (default: the viewer's port on localhost) or absent, optional bearer or `d2dview` cookie token (same token as the viewer), CORS only for the allowed origin with credentials. The viewer never forwards a write: V `TestStartWatchServesAnswersOnItsOwnPort`. Text safety: every bot-supplied string is cut to plain text (`safeText`), JSON escapes markup, the page writes only `textContent` (V `TestXSSIsInertText` also greps `app.js` for HTML sinks).

A GET never writes: expired thoughts are hidden at once, but their fade line is written when the bot side calls `Hub.Tick` (the ladder does on every `Decide`; `Hub.RunTicker` is the idle-bot timer) (V `TestFadeWritesMissedToJournal`).

## 3. Bot integration (the real bot, later, with no screen change)

`Hub` implements `ladder.Sink` and `ladder.AnswerSource` and subscribes to the store. What the bot calls:

1. Build the store as today (`decide.Open`), then `hub, _ := watch.New(watch.Config{Store, Cat, Clock, Areas, Orders: &watch.GoalsOrders{...}, WaitForSink: true})`, then `ladder.New(ladder.Config{Store, Sink: hub, Answers: hub, InFight: ...})`. Not `ladder.Open`: it makes the store itself and the Hub needs it first (API need 7).
2. Serve `hub.ViewHandler()` through d2dview (`-watch` mounts it) and `hub.AnswerHandler(opts)` on its own listener; `go hub.RunTicker(ctx, time.Second)`.
3. Answers: the route calls `store.Answer` directly. `Hub.Reply` returns nothing because `AnswerSource` is a one-shot pull made when the thought is first shown (V read in `ladder.pumpLocked`).
4. Orders (V mapping `OrderToGoal`, test `TestOrderToGoal`, `TestGoalsOrdersAgainstAFakeBot`; the goals queue is the order queue today, U whether T0 gets its own entry):

| Order | Goal (`internal/goals`) |
|---|---|
| go to AREA for N min | `farm_area`, `Params.Area`, `Params.UntilMins = N`, priority 5 |
| fight here for N min | `farm_area` on the area the hero is in (`GoalsOrders.Here`), `UntilMins = N` |
| come home | `idle`, 1 minute (U: no "go to town" goal type exists; the bot may need one) |

`GoalsOrders.OnGoalEvent` maps `goal_started/done/failed/skipped/removed` to the order state and `death` to an incident plus the death period; `*goals.Manager` satisfies `GoalQueue` as is.
5. Uptime source: the bot reports `Hub.Activity(kind, label)` on every change (kinds progress, death, recovery, stall, idle). Today only `goals.Status` (`ActivityFromStatus`: running is progress, error is stall) and `EvDeath` exist; a recovery period and a town stall have no source yet (U), so the first live version will show uptime that only drops on deaths.
6. Idle talk and incidents: `Hub.Say`, `Hub.Incident(text, ruleKeys...)`; the rules in force are the live rules of the store (the sim and the goals bridge pass all live rules; the exact rule that decided is API need 1).

## 4. Simulated runner (`watch.NewSim`, `d2dview -watch [-watch-class sorceress|paladin]`)

A fake bot on a virtual clock: the real `ladder.Decide` against the real catalogue at `t0.retreat.drink` every 30 s of fighting, orders, scripted death and town stall (`Kill`, `Stall`), idle talk. Two builds as data (`SimBuilds`): a softcore Sorceress (N=3) and a hardcore Paladin (risky: N=5, then a confirm card). `d2dview -watch` with no `-upstream` serves the screen alone; with an upstream it mounts the same routes next to the frame stream. d2dash was not used (U).

## 5. API needs (internal/decide, internal/ladder, not changed here)

1. A "rule used" signal: `Decision` has `Source` but `Ladder.Decide` returns only the answer, and the store emits no event when a rule answers. The death incident and "why did you do that?" for a real decision need the key of the rule that decided (e.g. `Ladder.DecideFull` returning the Decision, or `EvRuleUsed{Key}`).
2. `Store.Log()` copies the whole log per call and the Hub scans it for double answers: give `Store.ThoughtState(id)` (open, answered, faded, unknown) and `LogSince(seq)`.
3. The log does not name the key on answer, fade or confirm lines, a confirm thought has no open line, and "rule learned" is not logged at all. After a restart the journal can say less than in the session (learned lines vanish, confirm lines lose their context). Put the key on every line and add a learn line.
4. `ThoughtInfo` has no default answer and no urgency inputs (potion restore, recent damage, hits to live) that the ADR amendment wants in the feed.
5. `RuleView.Learned` has no json tag; a "streak toward a rule" read (`Record` has it) is fine.
6. `Store.NextExpiry()` so the idle loop ticks exactly when a thought fades, instead of a timer.
7. `ladder`: a push route for a human answer (`Ladder.Answer(id, ans)` and `Confirm`) so `Stats.Answered` counts it; `Open` taking a sink built from the store; the `Sink` does not report confirm thoughts or who answered.
8. Plain-language field phrases live in `internal/watch/humanize.go`; they belong beside the catalogue (`catalogue.json` labels) and the coverage test would move with them.
9. `goals`: a go-to-town goal and the orderable area list (the page has a placeholder Act 1 list from cmd/d2dview/areas.go, U beyond Rogue Encampment).

## 6. Stubbed or not done

Live bot connection, the real frame in the game area with a bot (frame stream only mounts when d2dview has an upstream), persistence of the Hub's own journal lines, story text, mobile layout, an in-browser test of the page (no browser here: the page is checked by string tests only, U), the countdown while the page is in a background tab, and the uptime recovery and stall sources of section 3.5. Run it: `go run ./cmd/d2dview -watch -addr 127.0.0.1:8891` then open `/watch`.

## 7. Look and feel (user decision 2026-11-05)

The watch screen uses the design language of the original dashboard (`web/style.css`, V by reading it and by rendering both): the same CSS tokens copied into `internal/watch/web/app.css` (`--bg #f3efe6`, `--surface`, `--line`, `--accent`, `--good/--warn/--bad/--info`, `--radius`, `--shadow`; light by default, dark through `prefers-color-scheme`), the same header (sticky `.top` with the diamond brand mark drawn in CSS, status dot, Theme button cycling auto/light/dark and stored under the same `d2theme` key), `.card` panels, `.btn` / `.btn.primary` / `.btn.danger`, `.badge`, `.bar` and the underlined `.views` tab strip. Thought cards use an accent left edge (confirm cards: warn colour), the urgency line in the warn colour and the fading timer as the `.bar`. Panels: Game, Thoughts, Orders, Rules, Journal.
Layout: at 900 px and wider a 12-column grid shows every panel; below that (phone) the tab strip shows one of Thoughts / Orders / Rules / Journal. V by screenshot at 390 px: no horizontal scroll (document width 390).
Still self-contained: no CDN, no external font, no inline script or style attribute, text only through `textContent`; V `TestWatchAssetsHaveNoThirdPartyURL`. The JSON views and element ids are unchanged. U: the question-card look was not captured in a screenshot (the simulated runner had no open question in the run).

## 8. Game video (user choice 2026-11-05: "Real video stream")

What the user sees: the Game panel of the page shows the hero's game window live, beside Thoughts on a wide screen and as its own "Game" tab on a phone. The panel and tab stay hidden unless the server offers a stream (`/api/config` `frameUrl` is set only with video on).

How (`internal/watch/video.go`, `capture.go`; `d2bot` flags in `cmd/d2bot/ladder.go`):
- Off by default. `d2bot ... --ladder --watch --watch-video [--watch-video-fps N] [--watch-video-quality Q]` (env `D2BOT_WATCH_VIDEO`, `_FPS`, `_QUALITY`). Without the flag no route, no process, no capture exist; the ladder and legacy paths are untouched. Refused at start: `--watch-video` without `--watch`, instance i0 or a non-iN id (only iN with N>=1 has a private display), fps outside 1 to 15, quality outside 2 to 31, neither capture tool installed.
- Routes on the existing viewer listener, GET only, same token gate as the other viewer routes (`?t=` sets the cookie the `<img>` then sends): `GET /stream` (multipart/x-mixed-replace, one `image/jpeg` part per frame with Content-Length) and `GET /shot.jpg` (newest frame, waits up to 5 s for the first frame of a fresh capture). `d2dview` does not mount them (it has its own `/stream`).
- The display is derived from the instance id, never a literal: `:(100+N)` as `launch.XvfbDisplay` names it (i5 gives `:105`). The capture reads that private Xvfb only; the desktop is never read.
- One capture shared by all viewers, started when the first viewer joins, stopped 5 s after the last one leaves (a viewer returning within that time keeps the same capture). Viewer cap 3 (HTTP 429 beyond it), frames over 512 KiB dropped, a slow viewer only misses frames. If the capture process dies the streams end and the page reconnects with a growing back-off; the next viewer starts a new capture.
- Capture tool, chosen at start: ffmpeg `x11grab` with piped MJPEG (`-f x11grab -draw_mouse 0 -framerate FPS -i :105 -c:v mjpeg -q:v Q -f image2pipe -`; default 8 fps, quality 5), else ImageMagick `import -window root` polled at up to 2 fps. Neither: the start error names both. Nothing is installed by the bot.
- Page: `<img src=/stream?t=...&r=...>`, no inline script or style, no CDN, text via `textContent`; on error it hides the picture and retries.

Tests (`internal/watch/video_test.go`, `cmd/d2bot/ladder_test.go`): fake frame source through httptest: token required, multipart framing by Content-Length, viewer cap, on-demand start and delayed stop, linger reuse, capture death and restart, oversize frames, two independent instances, routes absent when off, POST refused, JPEG splitter, tool choice and command line. V offline. The live measure is in the report of the change (section "Live check" below, if filled).
