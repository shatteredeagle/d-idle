## Why

The user wants to talk to a grill agent directly: one command that creates the grill worktree and starts a preconfigured session in which the user is the interviewee. Pasting a long prompt each time is error prone and does not enforce the write limits of the grill rule.

## What Changes

- `.claude/agents/grill.md` main agent (Opus) with the embedded grilling procedure
- `tools/grill.sh` and the guard hook `tools/grill-guard.sh`
- `docs/process/grill-agent.md` with the verified mechanism, grill-workflow.md updated, glossary term
