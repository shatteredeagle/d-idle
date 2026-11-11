# Digest: bot loop and goals

> status: current (TASK-310 digest; the area doc is the authority).

## Purpose
The bot core: one `bot.Runner` decides one action per step, a guard goroutine watches the running action, and `goals.Manager` holds the goal chain
(reach level, boss runs, campaign). It ties nav, combat, town and loot together; everything is per instance, no package-level game state.

## Key files
- `internal/bot/` (runner, loop, planner, profiles, death policy, `contracts.go` consumer-side interfaces)
- `internal/goals/` (goal, manager, chain, runner)
- `internal/decide/` (decision rules and catalogue)
- `cmd/d2bot/` (the binary; flags and wiring)
- `docs/bot.md`, `docs/goals/README.md` (research layer), `docs/botlogs.md` (countable guard lines)
- `docs/knobs.md` (generated knob defaults)

## Invariants
- The Run goroutine acts; the guard only watches and cancels with a typed cause, so input stays single-threaded (docs/bot.md, Architecture).
- Pause/Resume/Stop/Start/Status are O(1), take a mutex and never touch the game (docs/bot.md).
- Per-game memory (cleared areas, sweep kills, step counters) is forgotten on every new game; a bot-started fresh game counts as a recycle (docs/contracts/bot-loop.md).
- Decision code stays backend-agnostic: read state through `env`, no d2d import in behaviour packages (docs/contracts/backend-contract.md).
- Builds and classes are data (plan table, combat profile, class traits); no skill id, skill name or class check in Go, no silent fallback to the Sorceress (AGENTS.md, Build flexibility).
- Do not copy knob defaults into prose; they live in `docs/knobs.md`, cited as `group.key` (value) (AGENTS.md).
- Deaths are judged by the death rule only (docs/contracts/death-rule.md); soak and gate numbers live in docs/contracts/soak-gates.md.
- A measured run uses the real input channel only; a gameplay write after the start makes it CHEATED (AGENTS.md, docs/cheats.md).

## Gotchas
- Fatal versus non-fatal errors differ: a gone or hung game process is fatal and `Start` is refused afterwards; everything else is a retry state (docs/bot.md, Architecture).
- If a behaviour package emits events through `env.Emit`, list them in `disableObserve` or they are counted twice (docs/bot.md).
- The event channel drops the oldest event when full (docs/bot.md).
- Failed actions back off and escalate after `maxStepFailures`; the watchdog ladder is close panels, reset in town, then a visible Error state (docs/bot.md).
- Goal-layer research docs (docs/goals/) are tagged U where facts were not read from memory; do not treat the catalog as implemented.

## Read next
- [../bot.md](../bot.md) for the runner, state machine and failsafes.
- [../contracts/bot-loop.md](../contracts/bot-loop.md) for the numbered promises.
