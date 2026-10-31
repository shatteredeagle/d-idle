# Rules for every agent working on d2lod-bot

Read in this order: `openspec/README.md` (how jobs and contracts are written), `docs/plan/STATUS.md` (where we are; the block at the bottom is generated), YOUR CHANGE CARD `openspec/changes/<id>/` (the brief: proposal, tasks, `owns:`, `exclusive:`, `gates:`), `GLOSSARY.md` when a term is unclear, then `docs/d2-1.14d-memory.md` and the doc of your area. `PLAN.md` is the founding plan (historical; see its status block). These rules are not optional.

Maintainers also read `AGENTS.private.md` if present (it is not in this repo; `tools/ops.sh link` imports it into a gitignored `CLAUDE.local.md`).

## Ownership, exclusive resources, change cards
- Every job is a change card. The brief is "implement change X". You edit only the paths in the card's `owns:` (mirrored in `docs/OWNERS`) and use only the resources in its `exclusive:` (`instance:iN`, `d2d-control-socket`, `d2d-lab-binary`, `bot-binary:<path>`, `fork-branch:<name>`, `upstream-pr:<topic>`). Need something else? Write it in your report; do not edit it.
- Tick `tasks.md`, fill `result:` (say verified live or not) at the end. The coordinator archives the card at merge.
- Contracts and numeric gates are in `openspec/specs/<capability>/spec.md` (soak-gates, death-rule, backend-contract, fidelity-gate, merc-policy, early-game-policy, lab-rules). If your work changes a number there, put a delta spec in your change instead of editing the spec.
- `go run ./cmd/changecheck` (warn-only for now) checks all of this; run it before you finish. Questions about how to write a requirement, scenario or card: the `spec-sme` agent.

## The private d2d fork
- Anything that touches the private d2d fork or the lab patch set (the fork checkout, its branches, the lab binary) MUST be committed and pushed to the fork's private remote THE SAME DAY, and a doc may only cite a commit that is on that remote. Patches never go into this repo (GPL, docs/decisions.md). Nothing is posted to GitHub (issues, PRs, comments) without the user's go; drafts stay in the repo.

## Where you work
- You have your own git worktree and branch (path given in your brief). Work ONLY there, only in the directories you own. Do not touch other agents' directories; if you need a change there, write it in your report.
- Go 1.27.3. Module `d2lod-bot`, standard library ONLY. Do not edit `go.mod`/`go.sum` (it would collide with other agents). If you truly need a dependency, say so in your report and write the code so it can be swapped in.
- Keep code idiomatic and small: `gofmt`, `go vet ./...`, `go test ./...` must pass in your worktree before you finish. Comments only where the reason is not obvious. No emoji.
- Commit to your branch with `git -c user.name=Claude -c user.email=noreply@anthropic.com commit` and end the message with `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`. Agents never push and never touch other branches; the coordinator pushes to `origin main` (GitHub is the primary repo).
- Never use `git stash`: stashes are shared by all worktrees of this repo; use commits.
- COMMIT OFTEN: agents get dropped (memory crash, session restart, timeout) and committed work can be merged without you. Commit after every meaningful step and at least every 15 minutes of work, before any risky refactor and before your final report. A partial commit beats lost work. When resumed, first run `git status` and `git log --oneline -5`. Report the last commit hash in your final message.
- Host values (worker host, NAS, board URL, forge URL, cold dir) are never hard-coded: scripts source `tools/env.sh`, which reads the environment and a gitignored `local.env` (template: `local.env.example`). Prose says the role ("the worker host"), not the machine.
- Lab tooling (`tools/live.sh`, `tools/game.sh`, `tools/jobs.sh`, ...) lives in the private ops repo; the old paths here are stubs that call it through `tools/ops.sh` and exit 2 with a one-line message when the ops checkout is absent.

## Cleanup (process hygiene)
- CLEAN UP: every process you start must be started through `tools/agentrun.sh <agent> [--ttl SECS] -- cmd...` (or `tools/d2dslot.sh`, which registers; set `D2D_AGENT=<your-agent-name>`) so it can be reaped. Before your final report run `tools/agentreap.sh <your-agent-name>` and `tools/agentreap.sh --list` (on the worker host, where the process registry lives) and confirm nothing of yours is left. Never use `pkill`/`killall`, and never `kill` pids you did not start. Never leave waiter loops (`until ... sleep`) running: write them with a timeout or start them as `agentrun.sh --ttl`. Say in your report what you stopped and that the list is empty. Unregistered leftovers are listed read-only by `tools/agentreap.sh --orphans`; only the coordinator or the user stops those, by pid.

## The live games (shared, fragile)
- Live runs need the lab and its rules (`AGENTS.private.md`). Without it you work offline.
- Each agent owns at most ONE game instance with its own throwaway character. NEVER close, restart or launch a game, never create or delete characters, never use Battle.net or chat, never touch another agent's instance. Input only through the `d2input` helper of your instance (`input.NewClient("")` picks it from `D2BOT_INSTANCE`); use `mem.OpenGame()` / `mem.FindGame()`.
- Memory is READ-ONLY, with ONE exception: test "cheats" (set level/XP/gold, refill life, invincibility, save-file edits) are allowed on throwaway characters in offline instances, ONLY through the logged `internal/cheat` package / `cmd/d2cheat` tool (docs/cheats.md), never by ad-hoc writes: every cheat needs a reason and is recorded in the cheat ledger. Never cheat on any other data.
- Every game window is launched muted (`-w -ns`).
- Prefer offline tests: use `mem.Fake` or recorded snapshots so most of your work needs no game. Use the game only to verify assumptions and to record fixtures. Never block on a long run; every run has a hard time limit; leave the character alive, standing in town, no panels open, and say in your report what you did to it. Clean up when done: stop your own instance and leave nothing of yours running.

## Doc rules
- Do not hand-count things that change (commits, attempts, merged items): link to `git log`, `docs/g1-log.md` or the generated STATUS block. Do not copy a number that has an owner: knob defaults live in `docs/knobs.md` (generated from code), gates in the specs.
- When a doc is superseded or only partly true, add a `> status:` banner under its title (one line, with the pointer to the current doc) instead of leaving it to mislead; archive with `git mv` into `docs/archive/` and leave a one-line stub. Never delete docs.
- Decisions the user makes go into `docs/decisions.md` (one line index) with an ADR under `docs/adr/` when they matter; written by the coordinator or the user's grill session only (guard rule below).

## Facts and honesty
- Every offset, struct layout or behavior you rely on is tagged in code comments or docs as VERIFIED LIVE (say how), or UNVERIFIED. Never present a guess as a fact. Keep a short `docs/<your-area>.md` of what you learned, including dead ends.
- Do not write secrets (CD keys, passwords, tokens) or private hostnames, IPs, home paths or people's names anywhere in the repo (`go run ./cmd/leakcheck`, docs/process/public-split.md).
- This is an offline tool. Do not add anything that talks to Battle.net.

## Memory limits
- On a small box do not run whole-repo `go test ./...`, `go vet ./...` or `-race` builds; run them on the worker host. Per-package tests with `-p 1` are fine.

## Your report (final message)
Under 300 words: what is done and tested (with the exact commands), what is verified live vs not, what you changed on the live character, interface decisions other agents must know, and what you could not finish.

## Agent skills and guard rule
Skills are for the user's dedicated sessions; agent runs use them only when the brief says so. Work is tracked as openspec change cards (`openspec/changes/<id>/`), see `docs/agents/issue-tracker.md`; we do not use triage labels; domain docs: `GLOSSARY.md`, `docs/adr/`, `openspec/specs/` (`docs/agents/domain.md`).
Agents never create or edit `docs/adr/*`, `GLOSSARY.md` or `docs/decisions.md` entries on their own. Only the user's own grill session or an explicit coordinator brief that names the file may. Skills such as `domain-modeling`, `grill-with-docs`, `research` and `grilling` must not write files in agent runs: put proposed text in your report. Missing or wrong glossary term: say so in the report.
