# Dispatcher brief

> status: moved to the private ops repo (d2lod-ops, `docs/process/dispatcher-brief.md`). Not part of the public tree.

## Proposed dispatcher update (needs the user's go to replace the live prompt)

Replaces the standing step 4 ("G1 DRIVER") of the private brief. Not live until the user says go. Source: ADR 0013, docs/review-strategy.md "d2d gate ladder".

4. GATE DRIVER (d2d). When the drill suite passes for the S and A primitives (verdict column of docs/primitives.md; Gate 0) and main is clean (`git status`, merges done, `go run ./cmd/changecheck` 0 warnings):
   - Check capacity first: `tools/capacity.sh --add <soak cost>` must not print NO, and the worker host top idle+nice must be at least 25 percent (health guard).
   - Start the Gate 1 soak (2 h) on one hidden d2d slot, started only through `tools/agentrun.sh <agent> --ttl SECS -- tools/d2dslot.sh run -- d2bot --backend d2d --managed ... --run-for 120m` (flags as in section 4 of docs/d2d-baselines-early.md; `tools/d2dsoak/run.sh soak SEEDS HOURS -- CMD...` for replicated runs; `tools/ab/run.sh` is for A/B pairs, not gates).
   - Inject ONE fault at about 60 min: kill the slot's d2d only via the process registry (`tools/agentreap.sh <agent>` scoped to that slot; never `pkill`).
   - Judge at 2 h with `d2dsoak report` (the invariants report including the primitives invariant) and the Gate 1 criteria of the soak-gates spec. PASS -> start Gate 2 (8 h, profile m24), then Gate 3 (24 h). FAIL -> launch ONE fix agent for the failing criterion only, no feature work in parallel, then rerun the rung.
   - Merge-agent rules we learned: never `git pull --rebase` in the shared clone; merges wait for each other (one merge at a time); the coordinator pushes, agents never do.

## Process rules for the dispatcher and the coordinator (TASK-310)

- Every brief follows [../agents/brief-template.md](../agents/brief-template.md) (12 lines) and links [../agents/rules-short.md](../agents/rules-short.md) (stable path; not a session scratchpad file).
- One agent = one PR family; report under 250 words; no standing re-reading; model by the table in rules-short.md; do not spawn for a lookup or for work a running agent owns (SendMessage it).
- At most 6 agents at once; deploy to the sandbox in windows (docs/dev-update.md); one-line replies. Full list: [coordinator-efficiency.md](coordinator-efficiency.md).
