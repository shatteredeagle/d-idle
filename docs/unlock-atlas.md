# Waypoint atlas (TASK-160, unlockatlas)

Sits on the unlock ladder (`docs/unlock-ladder.md`); copied from `docs/unlock-chronicle.md`. Tags: V verified by the offline tests named, S secondary, U unverified.

## What it is

The Act 1 waypoints as numbered nodes: lit for the ones the hero has taken, silhouettes for the rest, and the zone the hero proposes next (zone proposals, `docs/proposezone.md`) marked. Panel "Atlas" on the watch page.

| ladder row | trigger | opens |
|---|---|---|
| `atlas` | `codex_first` `waypoint` (any: the first waypoint taken) | the atlas |

## Pieces

- `data/atlas/waypoints.json`: the nine Act 1 waypoints in the SME's order (S, `docs/research/fun-ideas.md` Q-S6): Rogue Encampment, Cold Plains, Stony Field, Dark Wood, Black Marsh, Outer Cloister, Jail Level 1, Inner Cloister, Catacombs Level 2. Each has the levels.txt area id (V `internal/saves/progress.go` waypointTable) and the Codex id that records it (V `internal/decide/firsts_act1.json`, checked by `TestAtlasTableAndErrors`). Nothing outside Act 1 yet.
- `internal/watch/unlockatlas.go`: `Atlas`, `GET /api/atlas`, schema `watch.atlas/1`. Lit comes from `decide.Codex.Has`, the proposal from the first pending `ProposalDesk` item. The town waypoint is lit from the start (S: it is active in a new game).
- `internal/watch/web/unlockui.js`, `watch.html`: the panel.

## Rules kept

- A silhouette carries only its number, no id, name or area, even when it is the proposed zone (`next:true`); the view's `next` states the zone the proposal already names.
- Nothing reads the game; the bot's existing waypoint hook (`Fun.OnWaypoint`) feeds the Codex.
- A repeat of a waypoint changes nothing and writes no second reveal.

## Verified

V offline: `go test ./internal/watch -run Atlas` (golden payloads for Sorceress and Paladin with the proposal on a dark node and on a lit one, token gates, replay, table order and Codex ids, errors naming the node or the row). U: the page in a browser, a real waypoint stream, a proposal to a zone without a waypoint (the node list then marks nothing).
