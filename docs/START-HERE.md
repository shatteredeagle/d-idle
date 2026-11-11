# START HERE

> status: current (cleanup batch B, 2026-11-05). One page; not a second index. The full map by area is [INDEX.md](INDEX.md).

## What this project is (5 lines)
- d.idle: an idle-game layer on a Go bot that plays one classic Diablo II: Lord of Destruction 1.14d character, offline single player, with your own game install.
- The bot runs on d2d (a native reimplementation of the 1.14d engine, primary backend, ADR 0009); original Game.exe is the frozen acceptance reference.
- Direction: the automation ladder (ADR 0016): the player starts by deciding, the bot takes over rung by rung, each rung gated by measured behaviour.
- Standard library only, no game files in the repo, no Battle.net; facts are tagged V (verified, say how), S (secondary) or U (unverified).
- Honest current state: [plan/STATUS.md](plan/STATUS.md) (generated block at the bottom) and [capabilities.md](capabilities.md).

## Where the direction lives
- Decisions: [decisions.md](decisions.md) ("Current truth" block first), ADRs in [adr/](adr/README.md); the ladder is [adr/0016-automation-ladder.md](adr/0016-automation-ladder.md) with the working text in [research/automation-ladder.md](research/automation-ladder.md), the decision-point store in [research/decision-point-store-spec.md](research/decision-point-store-spec.md) and the SME review [research/decision-point-sme-review.md](research/decision-point-sme-review.md).
- Plans: [plan/plan-v2.md](plan/plan-v2.md), [plan/plan-20261103.md](plan/plan-20261103.md); the watch screen is described by the ladder cards in `backlog/tasks/`.
- Terms: [../GLOSSARY.md](../GLOSSARY.md). Rules for agents: [../AGENTS.md](../AGENTS.md).

## How work flows
1. A job is a Backlog.md task in `backlog/tasks/` (keys `owns:`, `exclusive:`, `contracts:`; see [backlog-howto.md](backlog-howto.md) and [process/contracts-after-openspec.md](process/contracts-after-openspec.md)).
2. You work on your own branch `agent/<name>` in your own worktree, only inside the paths you own, and commit often. Agents never push.
3. The coordinator queues the branch in the merge queue; it runs the gate set and merges to `main`. The shared files (BACKLOG, CHANGELOG, INDEX, OWNERS, STATUS block) are generated at merge from your task card and the queue summary; you do not write them.
4. Anything about the private d2d fork or lab binary goes to the private fork the same day; nothing is posted to GitHub without the user's go.

## The six most-read docs
1. [../AGENTS.md](../AGENTS.md): the binding rules.
2. [plan/STATUS.md](plan/STATUS.md): where we are.
3. [d2-1.14d-memory.md](d2-1.14d-memory.md): game memory facts with their V/S/U tags.
4. [contracts/README.md](contracts/README.md): the promises and numeric gates.
5. [capabilities.md](capabilities.md): what the bot can and cannot do, with evidence.
6. [decisions.md](decisions.md): what was decided and why.

## The gates
Numbers live in the contracts, not here: [contracts/soak-gates.md](contracts/soak-gates.md) (Gate 0 to 3 ladder, soak invariants), [contracts/fidelity-gate.md](contracts/fidelity-gate.md), [contracts/death-rule.md](contracts/death-rule.md). Records: [soaks/](soaks/) and [probes/](probes/); history [g1-log.md](g1-log.md).
A measured run uses the real input channel only; a gameplay-affecting memory write after the start makes it CHEATED and it never counts ([cheats.md](cheats.md), ADR 0014).

## How to run the checks
```sh
gofmt -l .                                        # must print nothing
go vet ./pkg/... && go test -p 1 ./pkg/...        # per package on a small box; whole repo on the worker host
python3 tools/contracts/check.py                  # contracts: enforced_by paths and links
python3 tools/backlog/check.py                    # backlog tasks
python3 tools/ci/doclinks.py                      # relative markdown links
go run ./cmd/leakcheck -mode fail                 # no hosts, paths, secrets
tools/ci/check.sh                                 # the CI set in one go
```

## For agents: the reading order
`docs/START-HERE.md`, `docs/plan/STATUS.md`, your backlog task (named in the brief), `docs/contracts/README.md`, `GLOSSARY.md` when a term is unclear, then `docs/d2-1.14d-memory.md` and the doc of your area (find it in [INDEX.md](INDEX.md)). Contacts for people: the repo's Discord (see the [README](../README.md)); for agents, the brief names the coordinator.
