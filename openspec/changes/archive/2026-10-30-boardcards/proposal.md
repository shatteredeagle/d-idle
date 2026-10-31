## Why

The change cards under openspec/ are only readable as markdown and as the text output of changecheck. The user approved a page on the existing d2board that shows active changes, collisions, gates, specs and recent merges at a glance, fed by the same parser so there is one source of truth.

## What Changes

- `changecheck json` mode that prints the card, spec, gate, merge and warning data as JSON
- new `/changes` page on the board plus a `/api/changes` store, fed by a publish script on the host that has the repo
- docs/process/board-cards.md with the deploy steps for the NAS container

Merged: 95edbdf (agent/boardcards).
