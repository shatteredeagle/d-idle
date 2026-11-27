# Return screen mock (returnmock, TASK-109)

> status: MOCK ONLY (2026-11-27). Nothing here is verified live and nothing ran against the bot. It is the design input of returnscreen (TASK-127). Source of the content: fun-ideas.md sections 3.3, 3.6, 13 (ADR 0018). Tags: V verified (code read), U unverified.

Mock: [mocks/return.html](mocks/return.html), one file, fake data, open it in a browser. Looks like [mocks/orders.html](mocks/orders.html) (same tokens, light and dark, 44 px targets, no colour-only status).

## What the page shows, top to bottom
1. Headline: away time, level, zone, trip and death counts.
2. The five moments as tiles (closest call, named boss, best drop, level-up and unlock, stats that moved), each only when it happened.
3. The day as expedition cards, newest first (one per town trip: zone, length, cost, return, closest call, deaths).
4. A waiting proposal with the default already applied (wait while away, D R3-1).
5. Missed questions grouped by kind. Each group lists its questions and has one answer row "Answer once for all N"; pressing an answer shows the rule that would be made (`source: you`). That is the "one answer may teach a rule" requirement.
6. A three-sentence box for the session record.

Not in this mock (later cards): tally strip and next buys, policy before/after cards, charters.

## Journal export fields the page needs
V: `GET /api/journal` returns `watch.journal` lines (`internal/watch/views.go` `JournalLine`): `id`, `t` (seconds since the run start), `kind` (`question`, `answer`, `notsure`, `confirm-yes`, `confirm-no`, `missed`, `revoked`, `learned`, `late`, plus the Hub's order and uptime kinds), `text`, `thought`, `key`, `why`, `rules[]` (`key`, `context`, `label`, `live`).

What each mock block reads:
| Block | Fields |
|---|---|
| Missed questions | lines with `kind=missed`: `text` (question and context), `key` (decision-point key, the group), `t`, `thought` |
| Group kind | derived from `key` family (U: the table key family to the four kinds in the mock, e.g. fight and life keys to "Danger and retreat", town keys to "Town chores", is owned by the returnscreen card) |
| Answer once for all | `key` plus the answer the player picks; writes a rule through the existing answer route (V: the decide store keys rules by `key`) |
| Expedition card, closest call, deaths, firsts, best drop | U: needs a trip grouper over journal and Hub events (fun-ideas 3.3 says U that a clean trip boundary can be read from today's events) and min-life, death cause and item events that the journal does not carry as fields today |
| Headline, stats | `t` of the first and last line; level and zone from the Hub snapshot (U) |

Open for returnscreen: a "since you last looked" cursor (U: no such field), and whether grouped questions are answered by thought ids or by key.
