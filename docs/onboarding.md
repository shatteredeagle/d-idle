# Onboarding on /watch: welcome card, first 30 minutes, reconnect banners

> status: BUILT offline (TASK-247 firstminutes, TASK-249 reconnectux, 2026-11-11). Node DOM goldens and Go tests only; not looked at in a real browser, not verified against a live bot. Beta exit criteria: docs/research/project-review-2026-11-10.md section 5. The in-app tour and state banners of docs/research/onboarding-ux-design.md stay TASK-48; setup is docs/onboarding-slice-a.md.
> Tags: V = verified in this job (test named), S = secondary, U = unverified.

## What a friend sees

1. First visit: a welcome card at the top of /watch, five short lines in the d.idle voice: the bot plays for you, you equip and buy, repair is one button, any area can be ordered, the Journey strip shows the path, policies are the rules. Three buttons open Inventory, Policy and the Journey; "Got it, hide this" dismisses it for this hero.
2. Under it, "Your first 30 minutes": a checklist of five steps. It collapses (the choice is kept) and hides itself when every step is done.
3. Header "Help" button: "Show the welcome card again" and "Show the first 30 minutes list again" (works when the panel is otherwise hidden).
4. When the page loses the bot, restarts or the player was away, a banner (below).

| Step | Ticks by itself when | Else |
|---|---|---|
| Equip your starter gear | an equip request is accepted (`POST /api/inventory/equip` answered 200) | Mark done |
| Check the policy sheet | `/watch/policy` is opened (GET answered 200) | Mark done |
| Watch your first fight | the first `kill` event reaches the away log | Mark done |
| Open the Journey strip | the player follows the link (`tickOnOpen`; the strip itself is TASK-338) | Mark done |
| Mark junk for sale | `POST /api/inventory/mark` with mark `sell` answered 200 | Mark done |

V: `TestOnboardingAutoTicksAndHides`. A tick is kept forever (it is a latch, not a live reading).

## Reconnect banners (TASK-249)

All in the same panel, most urgent first (`banners()` in `panel.js`, pure, tested):

| Banner | When | Text (en.json) |
|---|---|---|
| Reconnecting | two polls of `/api/onboarding` fail in a row (network error, 5xx, 503 while the bot starts) or the browser is offline | "Reconnecting to your hero..." |
| Restarted | the reply's `boot` id changed since the last reply; shown 20 s | "The bot restarted..." |
| Away | the first reply of the page says the last look is older than `AwayAfter` (the return screen's tracker); stays until Dismiss, with a link to `/watch/return` | "Your hero was away for 2h 14m: see the Return screen" |
| Stream | the video hook `window.d2video.state()` (web/mjpeg.js) says `down` or `reconnecting`; Retry calls `d2video.reconnect()` | "The video stream is reconnecting..." |

A poll of `/api/onboarding` is not a look (only `/watch` and `/api/thoughts` touch the tracker). The reconnect words are kept in the browser from the last good reply so a page opened while the bot is down still speaks; a first visit with the bot down gets the one built-in line. A 404 (route off) hides the panel without a banner. U: the other panels keep the stale or empty state their own `update(null)` paths give (some hide); the shell's failure path is not changed here, so "no blank panels" is met by the banner explaining it, not by freezing every panel.

## Art preparing

While `GET /api/art` would say `extracting` (docs/paperdoll-art-build.md), the reply carries `art: {state, done, total}` and the guide shows the sentence of `art.preparing` ("Preparing the item pictures ... {done} of {total}"). Read from the art source's progress without decoding (`artProgress`); absent otherwise.

## Texts

All words are in `data/onboarding/en.json` (`watch.onboarding.texts/1`), embedded by `data/onboarding/onboarding.go` and sent whole in the reply (`texts`), so a wording change needs no JS change. Edit the file, then `UPDATE=1 node internal/watch/testdata/onboarding_dom.js internal/watch/web/panels/onboarding/panel.js` to refresh the DOM golden. Steps: `id`, `label`, `hint`, `href`, optional `tickOnOpen`; the five ids are latched by Go (`TestOnboardingStepsMatchLatches` fails if the set changes). Texts are shown through `textContent`; the file holds no markup or addresses (`TestOnboardingTextsPlain`).

## Interface

- `GET /api/onboarding` (viewer, panel endpoint, poll 3 s): `watch.onboarding/1` with `boot`, `hero`, `showWelcome`, `showList`, `collapsed`, `allDone`, `done`, `steps[{id,label,hint,href,tickOnOpen,done}]`, `away{absent,seconds}`, optional `art`, `texts`.
- `POST /api/onboarding` (answer listener, same origin/token/JSON/size gate as the other answer routes, `policyGate`): `{"action": dismiss_welcome | show_welcome | collapse | expand | show_list | tick, "step": id}`; unknown fields, actions and steps are 400. The viewer side stays GET only (405).
- State: `<state dir>/onboarding.json` (`watch.onboarding.state/1`, 0600, atomic rename), keyed by hero name, so a second hero sees the card again. `FunConfig.OnboardingPath` (cmd/d2bot/funwire.go sets it next to `visit.json`); empty keeps it in memory. V: `TestOnboardingStatePersistsPerHero`.
- Panel: `web/panels/onboarding/` (manifest v1, band side, priority 8, fit, endpoint `/api/onboarding`), placed full width at the top of every breakpoint of `web/layout.json` (a hidden panel takes no cell, so the page rises when it is done). The Help button is `#btn-help` in `watch.html`; `shell.js` turns its click into a `d2help` window event the panel listens for (panels may not query outside their root).
- Tests: `onboarding_test.go` (store, endpoints, gates, away), `testdata/onboarding_dom.js` + `.txt` (welcome, checklist states, Help, banners), pattern tests (text only, CSP safe, no focus or scroll jump on a poll), uilint, the compat fixture `testdata/compat/panels-v1/onboarding.json`.

## Not done / not proven

- Not looked at in a browser or on a phone; no live bot; the real "was away" case needs a visit older than `AwayAfter`.
- The Journey step has no strip to open yet (TASK-338): the link is `#journey` and does nothing on a page without it; the step still ticks, and "Mark done" is there.
- The welcome copy is a first draft in the d.idle voice for the owner to edit.
