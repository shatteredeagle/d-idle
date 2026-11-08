# Alerts and pings (TASK-162, unlockalerts)

Sits on the unlock ladder (`docs/unlock-ladder.md`). Tags: V verified by the offline tests named, S secondary, U unverified. The core is channel-agnostic: a rules table, an in-page channel, and an outbound `Sender` interface whose implementations are built but OFF.

## Rules table (V `TestAlertRulesShippedAndErrors`, `TestAlertRulesFiltersAndCooldowns`)

`data/alerts/rules.json` (embedded by `data/alerts`, parsed by `watch.ParseAlertRules`): `id`, `event`, `filters`, `cooldownSecs`, `severity` (`info`, `good`, `warn`, `alarm`), `free`, `text` (the hero's voice) and an optional `classText` per class. Parse errors name the rule: unknown event, severity, class or placeholder, a filter that does not fit the event, a drop rule without qualities, a kill rule without `boss`, a negative cooldown, a duplicate id.

| rule | event | filter | cooldown | free |
|---|---|---|---|---|
| `death` | death | none | 0 | yes |
| `low_life` | low_life | `belowPct` 30 | 120 s | yes |
| `unique_drop` | drop | unique, set | 0 | no |
| `rare_drop` | drop | rare, rune | 300 s | no |
| `boss_kill` | kill | `boss` | 0 | no |
| `level_up` | level_up | `minLevel` 2 | 0 | no |
| `back_in_town` | town | none | 900 s | no |
| `pact_pause` | pact_pause | none | 0 | no |
| `proposal_waiting` | proposal | none | 600 s | no |

Death and low life are basic safety info: a `free` rule is never behind the ladder. A test fails if any other rule is made free. Gated rules do not fire at all while the row is locked (nothing is stored, so buying later does not replay them). `classText` for death and level up exists for all seven classes (tests: Sorceress and Paladin).

## Sources (what feeds an event)

- V from the goals stream (`Alerts.OnGoalEvent`, called by `Fun.OnGoalEvent`): drop, death, level up, boss kill, and `town` (an area change into a town from a known non-town area, so the start-up spawn is not a return). A replayed stream (Seq not above the highest seen) raises nothing.
- V `Alerts.Tick` (called by `Fun.Settle`): a new pending zone proposal (`proposal_waiting`) and a Hub incident line starting `Pact: ` that is not a resume (`pact_pause`; this is the line `cmd/d2bot/pactwire.go` writes when it pauses the hero). U live: the pause line text is matched by prefix, not by an event.
- U `low_life`: `Alerts.OnLife(pct, area)` exists and is tested, but no emitter feeds it yet (`HeroInfo` carries no life). The status poll that learns life must call it.

## In-page channel (V `TestAlertsGoalEventsTwoClasses`, `TestAlertsAPIAndTokenGates`, `TestAlertsSincePaging`)

- `GET /api/alerts?since=N` (viewer, same token gate as the other viewer routes; 405 for other methods, 400 for a bad `since`): `{schema: watch.alerts/1, open, title?, teaser?, price?, seq, alerts[], outbound{enabled, kind?, target?, consent}, notifications{browser, background}}`. The last 100 alerts are kept in memory (not persisted).
- `web/alerts.js` (served at `/watch/alerts.js`): polls every 4 s, shows toasts (max 4, alarm stays 20 s), a notification centre (tab "Alerts" with an unseen badge), a sound toggle (off by default, kept in localStorage) and a locked-state line with a buy button that calls the existing `POST /api/unlocks/buy {id: alerts}`. The first poll after load shows history in the centre without toasts.
- Browser notifications: opt-in. `Notification.requestPermission()` is called only from the click handler of the button; a notification is raised only when permission is granted and the tab is hidden. U live: not run in a real browser.
- NOT built: background push with the tab closed. That needs a service worker and Web Push (a push service and VAPID keys), which is outbound by nature. The page says so.

## Outbound channel (off) (V `TestOutboundDefaultOffNeverContactsTheNetwork`, `TestOutboundConfigErrorsAndConsent`, `TestWebhookSenderSignsAndDoesNotRetry`, `TestNtfySender`)

`Sender` (`Name`, `Send(ctx, Alert)`): one attempt, no retry; a failure is kept in `Alerts.Errors()`. At most 4 sends in flight; more are dropped and noted.

- `WebhookSender`: POST JSON (`watch.alert/1`: seq, t, rule, kind, severity, text), header `X-D2-Signature: sha256=<hex HMAC-SHA256 of the body>` when a secret is set.
- `NtfySender`: POST the alert text to an ntfy-style topic URL with `Title`, `Priority` (info 2, good 3, warn 4, alarm 5) and `Tags`.
- Both refuse redirects and time out after 5 s.

Default OFF, in three layers: the zero `OutboundConfig` is off; `NewSender` returns nil for any setting with `enabled` false even when a URL is on file; `NewAlerts` builds a sender only for an enabled setting. The URL, kind and secret come only from the profile config (`OutboundConfig`, strict JSON via `ParseOutboundConfig`); no route changes them and no URL is in the repo. `FunConfig.Alerts.Outbound` is the hook; U: `cmd/d2bot` does not read a profile file into it yet (not in this card's paths), so nothing can send today.

Consent: the settings block shows `Outbound alerts are off. Nothing leaves this machine.` or, when on, `Sends data to <url without user info or query>: the text of each alert at or above severity <min>. Switch off in the profile config.` The minimum severity defaults to `warn`. The secret never appears in the API.

## Ladder

Row `alerts` (group `ui`, trigger `tally_spend`, price 60 tally; U price is a first guess to tune by play): the in-page centre, toasts, sound and browser notifications for every rule. Free rules work with the row locked.

## Open choices (not decided here)

- Whether a Pact pause belongs in the free set (the brief names only death and low life; a paused hero waits for the player).
- The price of the row.
- Where a profile file for the outbound setting lives and who reads it into `FunConfig.Alerts`.

## Verified

V offline: `go test ./internal/watch ./internal/unlock`. U live: nothing run against a game or a real browser; the outbound senders were exercised only against httptest servers.
