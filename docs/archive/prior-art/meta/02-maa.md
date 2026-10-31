# 2. MAA (MaaAssistantArknights): task queue, headless CLI, integration surface

Repo: https://github.com/MaaAssistantArknights/MaaAssistantArknights. AGPL-3.0, ~24k stars, active (pushed 2026-10-28) (V, `gh api`). Docs: https://docs.maa.plus/en-us/.
Sibling framework: https://github.com/MaaXYZ/MaaFramework (LGPL-3.0, 5k stars, "image-recognition automation framework") (V metadata).

## What it is (V, README)
- "One click completes all dailies" for Arknights using image recognition; C++20; Windows, Linux, macOS; emulators or devices (V).
- Features list is exactly a goal menu: sanity (stamina) farming with drop recognition, smart base shift scheduling with custom schedule JSON schema, auto recruitment, store/credits, rewards, and **"roguelike (Integrated Strategy) fully automated farming of originium ingots and level, auto burn/restart to reach a target"** (V, README, machine-translated from Chinese).
- "Copy homework": select a community-made JSON route and it replays it (V). This is the same idea as Tyniann recorded routes and Kolbot per-script data.
- **Integrations**: bindings for C, Python, Java, Rust, Go, plus Java/Rust HTTP interfaces "to build your own MAA" (V). CLI for headless servers on Linux/macOS/Windows (V).
- Uploads drop statistics to community sites (Penguin Stats) (V): crowd-sourced game data as a side effect of automation.

## Architecture lessons
- **Core library plus thin frontends**: GUI, CLI and bindings all call the same task engine (V by the listed interfaces). Matches PLAN.md (`internal/api` over `internal/goals`).
- **Declarative task data**: schedules, base layouts and roguelike strategies are JSON schemas published in docs (`protocol/` dir in repo, V that dir exists). Users edit data, not code.
- **Roguelike mode is the closest thing to "hardcore prestige loop" we found**: the bot runs a run-based mode repeatedly and the goal is a meta currency (ingots) and level, i.e. the loop resets and the meta layer accumulates (V claim from README, mechanics U).
- Multi-language, five language docs, big community: maintenance model is volunteers plus a data repo for UI templates (U).

## Weaknesses
- Vision again: updates to the game change UI; non-CN servers get thin testing ("very few foreign users, many features untested", V README).
- Tied to one game; no cross-game goal engine.

## Reuse
- License AGPL: ideas only. Take: JSON task/strategy schemas as public protocol, CLI parity with GUI, binding surface, drop-stat recording.
- Not reusable: image recognition stack (we read memory).
