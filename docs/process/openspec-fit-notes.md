# OpenSpec fit notes for d2lod-bot

Researched 2026-10-30 (OpenSpec 1.15.0). Tags V/S/U as in `openspec-reference.md`. This repo: 100+ prose docs, PLAN.md, many parallel agents with disjoint ownership in git worktrees, gates and acceptance numbers.

## Strengths
- Brownfield-first and delta-based: you spec only what a change touches, existing prose docs stay as source material. The project explicitly advises against converting PRDs/docs wholesale. V (docs/existing-projects.md).
- Pure Markdown + git; OpenSpec never touches git, so it fits worktree/branch flow; a change is one folder on a branch. V (team-workflow.md).
- Strict, scriptable validation (`validate --all --strict --json`) usable as a CI/pre-merge gate; archive is atomic (aborts with no files changed). V (run).
- Numeric thresholds fit naturally as requirement sentences plus WHEN/THEN scenarios ("below 50 percent", "at least 4 potions"); a threshold change is a small MODIFIED block with a reviewable diff. V (run) for mechanics.
- Archive keeps every change whole (why/what/tasks), giving an audit trail. V.
- Active project, weekly releases, MIT, can be used CLI-free if it dies. V.

## Weaknesses and risks
- Requirement-level merge only: two parallel changes MODIFYing the same requirement make the second archive overwrite the first (the project's own plan doc admits it; partial mitigation: validate's scenario-loss check). With many agents, assign each capability (spec file) one owner at a time and archive serially. V/U.
- No notion of gates, phases, status board, or acceptance numbers beyond text: "tracking status" is just `[x]` counts in tasks.md and `openspec list`. Acceptance numbers from soak runs (kills, deaths) are measured outside; the spec can only state the target. V.
- Specs describe behavior, not measurements; a spec like "deaths per 1000 kills SHALL be below N" is allowed but nothing verifies it. `/opsx:verify` is an LLM read of code vs spec, not a test run. V.
- Worktree support is weak when using stores (issue #1714 open); single-repo unscoped use with nearest-root resolution works per worktree. V (issue text). Monorepo with several roots is an open proposal (#1729). V.
- Heavy dependence on generated skills (long preambles, Node CLI on PATH, `Bash(openspec:*)`). Behavior, command names and file layout changed between 0.x and 1.x; weekly releases means churn. Pin the version. V.
- Process cost: secondary reports say review discipline is the real cost; lax review makes it ceremonial; one author measured roughly 2x feature time. S.
- Default telemetry and registry ping on `update`; disable by env. V.

## What is unclear
- How well agents follow a hand-written convention without the generated Claude skills (U, no evidence).
- Whether the parallel-merge plan (fingerprints, rebase command) shipped; only the scenario-loss check and an INFO conflict notice were seen in 1.15.0. U.
- Granularity at scale: no guidance on max requirements per spec; the project itself uses about 36 flat capabilities. A 100-doc prose corpus mapped into capabilities has no documented precedent. U.
- How to represent gates/acceptance runs (e.g. "soak 1000 runs") as spec vs task vs separate doc. Suggest requirements for the numeric target and a tasks.md verification item that points to the soak command. U (my judgment).
- Behavior of `/opsx:sync` (agentic) vs `openspec archive` (deterministic) when used together in many-agent flows. U.

## Suggested minimal trial (not done)
Init in a worktree with `--tools claude`, spec ONE area (e.g. the town routine) as `openspec/specs/town/spec.md` via one real change, run `validate --all --strict` in the check script, and judge after 3 to 5 changes. Keep PLAN.md as is.
