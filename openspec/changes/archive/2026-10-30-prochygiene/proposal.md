# prochygiene

## Why

Agents leave processes behind (bots, d2d children, slot waiters, `until ... sleep` loops) that hold d2d slots for hours, and `pkill -f` cleanups fail or are denied by the permission classifier. Cleanup has to be automatic and by record, not by pattern.

## What Changes

- `tools/agentrun.sh` starts a command in its own process group, registers it, enforces a TTL and kills the group on exit or signal.
- `tools/agentreap.sh` stops registered groups by agent after verifying pid and starttime, lists them, and reports unregistered orphans read-only.
- `tools/d2dslot.sh`, `tools/d2dview-demo.sh`, `tools/d2ddet/pair.sh` and `tools/d2dloop/*.sh` run through agentrun.
- AGENTS.md cleanup rule; the dispatcher gets a read-only janitor duty.
