# d2dfarm: distributed d2d test nodes

> status: v1 (ssh push). Spec: `openspec/changes/d2dfarm/specs/d2d-farm/spec.md`. Tags: V = run, U = not run. Hidden d2d only,
> never an original-game instance, never the private fork source (a fork change goes in a doc).

## Use
```
export D2_FARM_HOSTS='worker-a lab-slots HOST-A 1 any labels=fast;worker-b wsl HOST-B 1 22-7 labels=idle-only'   # in local.env
tools/d2dfarm.sh nodes                       # name, type, jobs/max, ready|paused|off-hours, labels, last probe, last seen
tools/d2dfarm.sh probe                       # OK|WARN|NO per node, in parallel (about 20 s)
tools/d2dfarm.sh push-tools worker-b         # build d2dsoak/d2dreplay/d2dprobe, ship to LAB/farm/bin (once per node and tool change)
tools/d2dfarm.sh run --seeds 1-8 --hours 2 --split 2 -- -clock bot    # queue 2 jobs of one batch and dispatch
tools/d2dfarm.sh status                      # refresh running jobs; shuts an empty WSL node down
tools/d2dfarm.sh collect BATCH; tools/d2dfarm.sh verdict BATCH       # ONE d2dsoak compare -gate v2 over all seeds
tools/d2dfarm.sh node pause worker-b         # like pausing a Tdarr node; running jobs continue; persisted
tools/d2dfarm.sh stop JOB|BATCH              # by recorded process group, never a name pattern
tools/d2dfarm.sh det FILE.replay.gz [NODE..] # cross-host determinism check
```
Agents call this instead of `ssh` by hand. Recipes: `soak` (d2dsoak run), `replay` (play a `--upload` file), `record` (scripted
scenario from `tools/d2dfarm.fight.txt`, `--seeds N` is the map seed), `cmd` (raw shell in the job dir, `out/` is collected).
State lives in `D2_FARM_DIR` (default `~/.local/state/d2dfarm`): `jobs/ID/job.env`, `nodes/NAME.{paused,probe,seen}`, `results/BATCH/JOB/`.

## Config and rules
- Registry line: `NAME TYPE SSH MAX HOURS [labels= lab= distro= user= nice= prio= build= d2d= d2dbin=]`. Inline in `D2_FARM_HOSTS`
  (`;` separated) or a gitignored `tools/d2dfarm.hosts`; the environment wins over `local.env`. Committed files hold placeholders only.
- Dispatch: label match, not paused, inside HOURS (local clock, `22-7` wraps), below MAX, probe not NO; OK before WARN, then `prio`
  (slots 10, wsl 50), then fewer jobs. A job that fits nowhere stays queued; `dispatch` retries.
- Capacity: slots nodes use `tools/capacity.sh --add 1` over ssh; the soak itself takes the lab lock (one run per node today).
- Safety: nodes only run lab recipes under `LAB/farm/JOB`; `stop` TERMs the recorded pgid after checking `/proc/PID/cmdline`;
  nothing here starts a game, creates a character or touches the fork source.

## WSL node etiquette (a person's PC)
- Probe is NO when an interactive session or explorer is running (V: no `quser` on this Windows, so it reads `Win32_ComputerSystem`
  UserName + explorer), cpu >= 85 % or free RAM < 5 GB; WARN from 60 % / 8 GB. Allowed hours default to a night window you set.
- Cap 2 cores / 4 GB (`.wslconfig`), `nice 19`, one job at a time, `wsl --shutdown` when the queue is empty and collected
  (`status`/`collect`/`reap` do it). Multi-step logic lives in the node script (`d2dfarm-remote.sh`); arguments cross
  ssh + PowerShell + wsl as ONE base64 token, so there is no quoting layer; the probe is a PowerShell `-EncodedCommand`.
- Binary is a Linux build with a bundled glibc (`bin/d2d.<build>.sh` wrapper, see the WSL worker doc (branch venusd2d)): private, never pushed.

## Cross-host determinism (node qualification)
Same binary (sha256 compared on the node) plus one scripted replay: `record` on one node, `det` plays it on each node, every
`detloop-journal-v1` checkpoint must match. 
V (2026-10-30, `det` and the `record` recipe): d2d.lab6 (sha256 a3e0cd59.., identical on both nodes) recorded a scripted walk+fight
(fresh Necromancer, map seed 3, 2120 ticks, 8 checkpoints) on the slots node, then played on the WSL node: OK 8/8 checkpoints,
0 reply differences. Same for the committed lab5 file (aaf1a72c.. on both): OK 8/8 on the WSL node, i.e. Ryzen/Debian 13 with the
bundled the worker host glibc gives the same simulation as the slots node. The player warns "build differs" there only because it hashes
the wrapper script, not the binary; `det` prints the real binary sha256 per node first. The slots-node replay of the lab6 file
is in the report (queued behind other runs on its lab lock).

## Transport (v1 = ssh push; the code has one shell function per verb: `tx_<transport>_{probe,start,state,collect,stop,put,shutdown}`)
Alternative A, forge Actions: each machine already runs a runner with labels and capacity. A job = `workflow_dispatch` workflow
(inputs recipe, seeds, hours), `runs-on: <node label>`, results as artifacts or a pushed results dir; pause = disable the runner.
- Pros: queue, dispatcher, log UI and capacity come for free; same node concept as CI; no ssh keys on the dispatcher; pause in the UI.
- Cons (the forge 11, V in docs/process/forge-runners-plan.md: no runner REST routes, labels only in the web UI/DB): no
  probe-based admission (capacity is a static number; the idle-only rule needs a pre-step that fails the job), `workflow_dispatch`
  inputs and v4 artifacts are weaker on 11 (upload-artifact v4 is unsupported on the forge, use v3 or push to a results dir), a
  Windows host runner must call `wsl -d Debian` (same quoting trap), jobs are killed when the runner restarts, 3 h timeout.
- Recommendation: keep ssh push for now (works, tested). Move to Actions when more than three nodes exist or people other than
  the maintainer submit jobs. Effort U: about 1 day (workflow, node label per runner, result artifact, wrapper calling
  `d2dfarm-remote.sh`), plus the forge upgrade checks.
Alternative B (design sketch only): a small coordinator service on the NAS host as a UI-visible container holds the queue and
verdict; nodes run an agent that PULLS jobs by label and posts heartbeats and results over HTTP. Good for NAT and for the PC
that is off most of the day (the node announces itself when idle), pause = a switch in the UI. Costs a service, auth and a protocol.
