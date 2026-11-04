## Why

ADR 0009 makes d2d the primary backend: --backend d2d becomes the dev default once the bot loop runs on it (cmd/d2bot/d2dbackend.go currently defaults to game). Specs record the new status: d2d primary, Game.exe frozen, the break-glass exit trigger. Blocked by d2d-loop-hours.

## What Changes

- flip the default (env D2BOT_BACKEND keeps working), game stays selectable
- go test ./cmd/d2bot passes with both defaults covered
- apply the spec delta when this merges
