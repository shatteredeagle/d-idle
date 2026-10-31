## Why

The user decided on 2026-10-30: "make the cap soft/floating not hard depending on current usage. if you need to do a big run warn me ahead of time but if I don't respond in a timely manner (5min) go ahead with the run". Today's hard rule (live work only if idle+nice >= 25 % and at most 2 Game.exe, dispatcher brief, soak-gates start guard, lab-rules load 45) blocks work on a quiet machine and ignores the cost of what is being added. All work is on d2d now (the original Game.exe path is frozen), and there is no scheduled timer for board refresh.

## What Changes

- a floating capacity rule: new live work is allowed when the PREDICTED idle+nice after adding its cost stays >= 25 % and iowait stays under 40 %, on a 5 minute average, with a per-item cost table in cores
- `tools/capacity.sh`: read-only script that measures and prints OK, WARN or NO for `--add CORES`
- a big-run protocol: warn the user, silence for 5 minutes means go, any reply hold or no cancels; non-big work is never gated on the user, bug-fix and offline work has no cap
- dispatcher brief health guard rewritten to use the script, plus the "no timer or cron for board refresh" line
- `docs/process/worker-capacity.md` with the rule, costs, protocol, a worked example and how to read the output
- spec delta: MODIFIES lab-rules "Safety limits for lab runs" and soak-gates "Start guard"

Not in scope: applying the delta to `openspec/specs` (the coordinator does that at merge), the soak24.sh and d2dsoak load guards in code (a follow-up reads `tools/capacity.sh`), AGENTS.md and docs/review-strategy.md wording (not owned here, see the result note).
