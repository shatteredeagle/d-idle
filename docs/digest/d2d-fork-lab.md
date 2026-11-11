# Digest: d2d fork and lab

> status: current (TASK-310 digest; the area doc is the authority).

## Purpose
d2d is the headless native reimplementation of the 1.14d engine and the primary backend; original Game.exe is the frozen oracle. Our additive patches (lockstep clock, `dump` and debug verbs) live in a private fork and a lab binary, never in this repo.

## Key files
- `internal/d2dctl/` (control socket client and verbs), `internal/d2dsrc/` (state reader), `internal/d2dmesh/`, `internal/d2dreplay/`
- `cmd/d2dsoak/`, `cmd/d2dprobe/`, `cmd/d2dsetup/`
- `tools/d2dslot.sh`, `tools/live.sh`, `tools/agentrun.sh`, `tools/agentreap.sh`
- `docs/d2d-fork.md`, `docs/d2d-lab.md`, `docs/d2d-backend.md`, `docs/d2d-determinism.md`, `docs/d2d-fidelity.md`, `docs/cheats.md`, `docs/multi.md`
- `docs/contracts/lab-rules.md`, `docs/contracts/backend-contract.md`, `docs/contracts/process-hygiene.md`, `docs/contracts/replay.md`

## Invariants
- Anything touching the fork or lab binary is committed and pushed to the private remote the same day; a doc may cite only a commit on that remote. Patches never enter this repo (GPL); nothing is posted to GitHub without the user's go (AGENTS.md, docs/contracts/lab-rules.md).
- The lab binary is never overwritten; a rebuild gets its own provenance file (docs/contracts/lab-rules.md).
- One d2d per slot, private sockets; a shared socket path is refused unless `--d2d-attach` is given (docs/contracts/lab-rules.md).
- The d2d control socket is single-client; a stranger socket fails the start (docs/contracts/backend-contract.md).
- Every long process starts through `tools/agentrun.sh`; reap with `tools/agentreap.sh <agent>`; never `pkill`/`killall` (AGENTS.md, docs/contracts/process-hygiene.md).
- Memory writes only for SETUP, INSTRUMENTATION, DIAGNOSIS, FIXTURE, via the logged tool; a gameplay write after a measured run starts makes it CHEATED (AGENTS.md, docs/cheats.md).
- Each agent owns at most one hidden instance and one throwaway character; never close, launch or touch another's (AGENTS.md).

## Gotchas
- Rendering is fused into the town update; rerun the determinism check after every rebase (docs/d2d-fork.md, Risks).
- `--time-scale` above about 11 loses ticks; use lockstep for fast runs. Frame counters include start-up; use `game_ms` or `stepped` (docs/d2d-fork.md, Risks).
- Memory grows by roughly 100 MB per first-visited level; a long soak must restart the process (docs/d2d-fork.md, Risks).
- docs/d2d-lab.md is partly stale (lab2/lab3 history); the current binary is named in docs/d2d-fork.md and docs/d2d-loop.md.
- Read data with the project reader, not ad hoc (docs/d2d-lab.md, Data reading trap); never commit Blizzard-derived data (docs/contracts/public-split.md).

## Read next
- [../d2d-fork.md](../d2d-fork.md)
- [../contracts/lab-rules.md](../contracts/lab-rules.md)
