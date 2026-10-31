# Issue tracker: openspec change cards

Work for this repo is tracked as openspec change cards in `openspec/changes/<id>/`. There is no GitHub Issues use, no `gh` or `glab` calls, and no `.scratch/` directory. The forge remote (`forge`, `<owner>/d2lod-bot`) is a push target only; nothing is ever posted to GitHub without the user's go (`openspec/specs/lab-rules`).

This file replaces the "issue tracker" template of the setup skill (hand-written, the skill was not run). Format and life cycle: `openspec/README.md`.

## What a "ticket" is here

| skill wording | our equivalent |
|---|---|
| issue / ticket | one change card folder `openspec/changes/<id>/` (`proposal.md` = Why and What Changes, `tasks.md` = header block + checkbox tasks) |
| spec / PRD | `openspec/specs/<capability>/spec.md` (stable contract) plus a delta spec inside the card when the job changes a contract |
| issue number | the card id (lowercase kebab-case, normally the agent name, e.g. `d2dwire`) |
| labels / state | the `status:` line of the card header: `proposed`, `in-progress`, `done`, `archived` |
| assignee | `agent:`, `branch:` and `worktree:` in the header, plus `docs/OWNERS` |
| blocked by | `exclusive:` resources and `owns:` globs (changecheck rejects overlap); gates are the `gates:` lines |
| close | the coordinator archives: `openspec/changes/archive/YYYY-MM-DD-<id>/`, header `status: archived`, `merge: <hash>` |
| PR | a branch `agent/<id>` merged by the coordinator with `Merge agent/<id>: ...` (no pull requests on the forge) |

## Conventions

- Create: `go run ./cmd/changecheck new -id X -agent X -owns 'dir/**' -exclusive 'res'` (coordinator), then fill `proposal.md`, `tasks.md` and the `gates:` list.
- Read: `openspec/changes/<id>/` (proposal, tasks, specs/). List open work: `ls openspec/changes` (everything except `archive/`), plus `docs/plan/STATUS.md` and `docs/BACKLOG.md`.
- Comment / progress: tick the checkboxes in `tasks.md`; fill `result:` (say verified live or not). Anything for the coordinator goes in the agent's final report, not in files outside `owns:`.
- Check: `go run ./cmd/changecheck -cli` must print 0 warnings.

## When a skill says "publish to the issue tracker"

Do not create files on your own. In an agent run, write the proposed card text (title, Why, tasks, gates) in your report; the coordinator creates the card with `changecheck new`. In a user-run session (grill, to-spec), create `openspec/changes/<id>/` only when the user says so, using the format in `openspec/README.md`, never `.scratch/`.

## When a skill says "fetch the relevant ticket"

Read `openspec/changes/<id>/proposal.md` and `tasks.md`, then the specs the card links. Archived cards are under `openspec/changes/archive/`.

## Skills that do not apply

`to-tickets`, `triage`, `wayfinder`, `implement` and `implement-spec` assume a GitHub/GitLab/`.scratch` tracker and build a second work-tracking system; they are not used here (`docs/process/matt-pocock-skills-fit.md` section 4).
