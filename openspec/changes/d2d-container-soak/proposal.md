## Why

The one-slot 24 h soak of the d2d container (risks step 10) is the gate that, per ADR 0009, ends Game.exe in the dev and product path. Needs the user's go for the NAS template (step 9). It also carries restart and kill injection for the d2d process.

## What Changes

- template and first run by a non-agent per docs/container.md (step 9, user go)
- 24 h one-slot soak with restart and kill injection, memory curve (step 10)
- apply the spec delta when this merges
