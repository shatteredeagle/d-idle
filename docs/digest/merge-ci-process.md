# Digest: merge, CI and process

> status: current (TASK-310 digest; the area doc is the authority).

## Purpose
How a job flows: a Backlog.md task, an own branch and worktree, shared-file lines in a merge-lines file, the coordinator's gated merge to `main`, CI on GitHub, and the public-split hygiene checks.

## Key files
- `backlog/tasks/`, `backlog/merge-lines/`, `docs/backlog-howto.md` (read the "For agents" section), `docs/agents/brief-template.md`
- `tools/ci/check.sh`, `tools/ci/doclinks.py`, `tools/contracts/check.py`, `tools/backlog/check.py`, `tools/merge_lines_check.py`
- `.github/workflows/ci.yml`, `docs/ci.md`, `docs/process/ci-checks.md`, `docs/process/outside-prs.md`, `docs/process/public-split.md`
- `docs/contracts/README.md`, `docs/contracts/public-split.md`, `docs/contracts/upstream-pr.md`, `docs/release.md`, `docs/decisions.md`

## Invariants
- Edit only paths in the task's `owns:`; use only `exclusive:` resources; the coordinator edits status and closes tasks (AGENTS.md).
- Agents never push and never run `git stash`; commit often on `agent/<name>` (AGENTS.md).
- Shared files (BACKLOG.md, INDEX.md, CHANGELOG.md, OWNERS, STATUS.md) are not yours: put `BACKLOG:`, `INDEX:` or `CHANGELOG:` lines in `backlog/merge-lines/<branch>.txt` (docs/backlog-howto.md).
- Before reporting run `tools/contracts/check.py`, `tools/backlog/check.py` and `tools/merge_lines_check.py --changed` (AGENTS.md).
- A contract line whose Source is a user decision, and the Numbers tables, change only when the brief says the user approved it (AGENTS.md).
- No hosts, home paths, people's names or secrets in the repo; leakcheck must report zero (AGENTS.md, docs/contracts/public-split.md).
- Community PRs: commit as yourself with an `Agent:` trailer, not the `Claude` identity; one topic per PR (AGENTS.md).
- Never edit ADRs, GLOSSARY.md or decisions.md entries unless the brief names the file (AGENTS.md).

## Gotchas
- The Backlog CLI rewrites quote style and `updated_date`, and a title over about 80 characters becomes a folded block that breaks the checker (docs/backlog-howto.md).
- The CLI does not show the custom keys (`card`, `owns`, `exclusive`, `contracts`, `result`); read the file (AGENTS.md).
- CI runs on pushes to the default branch, tags and pull requests, not on agent branch pushes (docs/ci.md, with its own status banner for the earlier self-hosted setup).
- Whole-repo `go test ./...`, `go vet ./...` and `-race` do not run on a small box; use the worker host (AGENTS.md, Memory limits).
- Do not hand-count commits, attempts or merged items; link to `git log` or the generated STATUS block (AGENTS.md, Doc rules).

## Read next
- [../backlog-howto.md](../backlog-howto.md)
- [../process/ci-checks.md](../process/ci-checks.md)
