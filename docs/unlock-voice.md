# Hero voice and diary (TASK-155, unlockvoice)

Second of the ten unlockables (`docs/research/unlockables.md`); sits on the ladder (`docs/unlock-ladder.md`). Tags: V verified by the offline tests named, U unverified (needs a live run).

## What it is

The hero says one short line at the first level-up, at the first death and at every new Codex entry, and each one becomes a diary page (milestone title plus the line). Ladder row `voice`: `level` min 2, or a first death in the Codex. A locked voice shows as a silhouette with its teaser. Page parts: a quote strip under the header (the latest line), a "Diary" tab (newest page first), a "Mute voice" button.

## Pieces

- `data/voice/lines.json`: the lines, 6 or more general lines per kind (`levelup_first`, `death_first`, `codex_monster|area|waypoint|shrine|drop|champion|death`), optional per-class flavour under `classes.<class>` (class names listed in the file's `classes`). Placeholders `{name}` (the thing just met) and `{n}` (the level) only. `ParseVoiceLines` rejects: a missing or unknown kind, fewer than 6 lines, a class not in the list, an unknown placeholder; each error names the kind or class.
- `internal/watch/unlockvoice.go`: `Voice`, `GET /api/voice` (schema `watch.voice/1`; locked: teaser only), `POST /api/voice/mute {muted}` on the answer listener (same origin list, token, JSON-only rules as the buy route, shared helper `unlockPostGate`).
- State: `voice-<profile>.json` beside the ladder file (spoken keys, diary, mute flag; atomic write). A restart keeps the pages and never repeats a milestone.
- The choice is seeded: `fnv(profile/kind/key) mod pool`, pool = general lines + the hero class's flavour. A replay or a reload says the same line. An unknown hero class is an error naming it (in `Unlocks.Errors()`), and only the general lines are used.
- Muted: the diary still fills, nothing is said in the feed. Unmuted: the line goes to the feed as an idle line (`Hub.Say`).

## Rules kept

- Never silent: opening the voice writes the ladder's reveal (Chronicle line, away event, "Unlocked since you left").
- Spoilers: a line names only what the Codex entry says was just met; a test checks no line names a monster, area, waypoint or shrine of the Act 1 table.
- Codex firsts met before the voice opened are kept (up to 100, in memory) and written as quiet diary pages, without a spoken line, when it opens. A restart before the voice opens loses them (U; the Codex itself keeps them).
- The voice never reaches the bot.

## Verified

V offline: `go test ./internal/watch` (golden payloads for Sorceress and Paladin, locked and open states, replay and reload add nothing, seeded choice stable, mute gate and effect, unknown row and class named, table checks, spoiler check). U: the strip and tab in a browser; a real run.
