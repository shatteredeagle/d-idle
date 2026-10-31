# 5. Screeps: the game that is the unattended-script layer

- https://screeps.com, server https://github.com/screeps/screeps (ISC, ~3.4k stars, last push 2026-04-01; V `gh api`). Client is not open source (S, Reddit thread via search).
- A persistent MMO where you write JavaScript that runs 24/7 even while logged off; private servers run locally with NPC bots and mods (S, search summary of official docs/Steam). Official world ~70k rooms on a Node cluster (S).

## Why it matters though it is not "an existing game wrapped"
It is the pure form of our product's contract: the human supplies policy, code runs unattended, state persists, the world keeps going. Lessons:
- **CPU budget per tick** is the core scarce resource and a progression axis (paid CPU tiers) (S). Ours: instance-hours and cores are the resource; a scheduler should budget them.
- **Standalone server with mods** (V README advice via search: extend through mods, not source edits; npm/Workshop mods) is a clean extension model for our "cheats as rewards" and goals as plug-ins.
- Community built the entire tooling ecosystem around it: dashboards, grafana-style stats exporters, deployment tools (U, from memory; not fetched). Opening a stable API (our HTTP+SSE) attracts that.
- Human oversight pattern: console + visualizer, post-mortem of failures from logs (U).

## Limits
- It is a game designed for this, so no stuck-state or UI-fragility lessons; it solves nothing about externally driving a client.
- Reuse: ISC licence allows copying server code, but it is JS and irrelevant to D2. Design only.

## Related "scripting is the game" examples (S, not fetched)
Dwarf Fortress (DFHack), StarCraft BWAPI/SSCAIT bot ladders, Dota 2 bot scripts, Factorio Lua mods: each shows a game adding an official automation surface; D2 has none, which is why our layer reads memory.
