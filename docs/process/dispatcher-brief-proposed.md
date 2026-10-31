# Dispatcher brief, proposed replacement (agent ROADMAP, 2026-10-31)

> status: PROPOSAL (its WIP limit 3 and milestone rule were ADOPTED by the user on 2026-10-31, docs/roadmap.md decision D1; the file replaces the private brief only when the user applies it). The standing brief lives in the private ops repo (`docs/process/dispatcher-brief.md` there; the public file is a pointer). This is the text ROADMAP proposes to replace it with, written against docs/roadmap.md. The user decides (roadmap decision 1); until then the standing brief holds.

## The one rule

Every agent launched implements a change card that belongs to the current milestone (docs/roadmap.md section 3), or is a bug fix whose proposal cites a finding (a log line, a failing test, a soak anomaly) and whose gate is a scenario test that asserts the wanted end state. No card, no agent. "Needs a G1/G2 check" is not a gate (Game.exe is frozen, ADR 0009): a shared-code change is verified on d2d or its result says U.

## WIP limit

- At most 3 implementation agents at once: 2 on the current milestone's critical path (the launch order in docs/roadmap.md section 4), 1 on a side lane (an upstream PR reply, a stream or process card, a user request). Read-only agents (d2-sme, spec-sme, d2-reviewer, an observer) do not count.
- At most 1 new upstream PR in flight; replies to open PRs within a day; a new candidate only from a gap a milestone run found.
- Live work: one d2d slot per live agent, one slot kept for the livestream soak; a card that is blocked (licence, a decision) must not hold a slot claim: set it `deferred`.
- Why 3: the merge checklist (owns diff, gates, whole-repo test on the worker, archive, OWNERS, STATUS) is done by one coordinator; at 3 agents the queue is one merge per 40 minutes, which the 2026-10-31 rate (12 merges in 2 hours) shows is the limit where cards start merging without a milestone.

## The tick (every 2 hours, or when an agent reports)

1. `go run ./cmd/changecheck status -write`; read the current milestone's exit criterion in docs/roadmap.md and write its current number in one line (XP/h and verdict for M2, coverage rows for M1, seeds passed for M3).
2. Merge what is done per the checklist in openspec/README.md; archive; a card whose gates do not hold is not merged (a partial result goes back as a smaller card with a finding).
3. Promote: for each `deferred` card whose named dependency is archived, set `status: proposed`, add its `docs/OWNERS` lines, run `go run ./cmd/changecheck` (0 warnings, no overlap) and `tools/openspec.sh validate --all --strict`.
4. Launch the next `proposed` cards in the launch order up to the WIP limit, with the one-paragraph brief from openspec/README.md ("implement change X"), the instance line for live cards and a time box.
5. Janitor duty (process-hygiene spec): `tools/agentreap.sh --list`, `--all-expired`, `--orphans` to the needs-you list.
6. Needs-you: keep the decisions list of docs/roadmap.md section 8 current; a blocked card names its decision.
7. Nothing else: no ad hoc research agents, no "while we wait" cards, no second agent on a symptom another card owns.

## The weekly review (the user, 20 minutes)

- The milestone table (docs/roadmap.md section 3) with the number per exit criterion and the date it last moved.
- Cards archived since the last review, grouped by milestone; cards archived outside any milestone (should be zero, or bug fixes with a finding).
- The decisions list; what the dispatcher is waiting on.
- What to stop: any lane that produced merges but moved no exit number.

## Stop doing

- Research docs without a card gate or a PR consequence.
- Knobs that default off "for a later check"; a fix ships on, with its scenario test and its spec line, or it is not done.
- A second fix of a symptom an archived card already fixed, without a scenario test that failed first.
- "Needs a G1/G2 check" notes.
- Proposed cards that hold `exclusive:` slot claims while blocked.
- Counting progress in merges. Progress is the exit number.

## Promotion of a deferred card (convention proposed for openspec/README.md, owned by specfix today)

`deferred`: written and reviewed but queued behind a named dependency or a user decision (named in proposal.md under `## Dependencies`). Its `owns:` and `exclusive:` are intended, not claimed: no docs/OWNERS line, no overlap check. When the dependency merges the dispatcher sets `proposed`, adds the OWNERS lines and re-runs changecheck before launch; a deferred card whose delta MODIFIES a requirement another active card also modifies waits for that card's archive and re-copies the then-current requirement.
