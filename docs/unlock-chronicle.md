# Chronicle (TASK-154, unlockchronicle)

First of the ten unlockables (`docs/research/unlockables.md`); sits on the ladder (`docs/unlock-ladder.md`). Tags: V verified by the offline tests named, U unverified (needs a live run).

## What it is

A scrolling log in Diablo II words: kills, finds, new areas, level-ups, deaths and first sights. Panel "Chronicle" on the watch page (a tab on the phone). A locked part shows as a silhouette with its teaser under "Still locked".

| ladder row | trigger | opens |
|---|---|---|
| `chronicle` | `kills_of_kind` `*` min 1 (the first kill) | the kill and loot feed |
| `chronicle_tape` | `seen` `expedition_card` (the viewer is served the first return page that has a trip) | the run timeline tape under the feed |

Both opens write the ladder's reveal (Chronicle line `Unlocked: ...`, away event, "Unlocked since you left").

## Pieces

- `internal/watch/unlockchronicle.go`: `Chronicle` (ring of 300 lines), `GET /api/chronicle[?since=N]`, schema `watch.chronicle/1`. Locked: `open:false`, `teaser`, no lines, no trips. The tape is the return screen's trips (`ReturnConfig.Return`), oldest first.
- `data/chronicle/wording.json`: the wording, 3 or more variants per kind (kill, kill_xp, boss, drop, area, level, death, first). The variant is chosen from the event `Seq` (stable on replay). An unknown kind or too few variants is an error naming it.
- `internal/watch/web/unlockui.js` (route `/watch/unlockui.js`): polls `/api/unlocks` and the open features; shared by the later unlock panels. All text via `textContent`.
- Engine additions (`internal/unlock`): trigger kind `seen` (arg `what`), event `EvSeen`, `Engine.Has/Profile/Dir`. `Unlocks.OnGoalEvent` now counts kills per monster name (in memory, a replayed `Seq` never counts twice) and feeds `kills_of_kind`; `Unlocks.Seen(what)` reports a viewer-side moment. `ReturnConfig.OnRead` is called when the HTML return page is served.
- `cmd/d2bot/funwire.go`: `newFun` opens the shipped ladder per profile (`FunConfig.Unlock`). This turns the whole ladder on in the real bot.

## Rules kept

- Kills come from the same goals events the tally reads; the Chronicle reads no memory.
- Drop lines never name a quality or colour: the identify reveal (TASK-157) owns that. Only non-normal drops get a line.
- First-sight lines come from new Codex entries only (what was met), so nothing unmet is named.
- Lines recorded before the feed opens are kept but not served until it is open.

## Verified

V offline: `go test ./internal/unlock ./internal/watch ./cmd/d2bot` (golden payloads for Sorceress and Paladin, locked and open states, replay adds nothing, `since`, tape opens on the first page read and not on the JSON route, reveal line exists, unknown row and bad wording named). U: the page in a browser, a real run's kill stream (the in-memory kill count restarts with the process; opening is persisted).
