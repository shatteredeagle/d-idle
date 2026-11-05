# Issue tracker: Backlog.md tasks

> status: current since OpenSpec removal stage 4. Work is tracked as Backlog.md tasks (https://github.com/MrLesk/Backlog.md, MIT) in `backlog/tasks/`; how the board works, the CLI recipes and the known problems are in `docs/backlog-howto.md`. The retired OpenSpec change cards live under git tag `openspec-final` (index: `docs/archive/openspec-index.md`; `git show openspec-final:<path>`).

Work for this repo is tracked as task files in `backlog/tasks/`. There is no GitHub Issues use, no `gh` or `glab` calls, and no `.scratch/` directory. The forge remote (`forge`, `<owner>/d2lod-bot`) is a push target only; nothing is ever posted to GitHub without the user's go (`docs/contracts/lab-rules.md`).

This file replaces the "issue tracker" template of the setup skill (hand-written, the skill was not run).

## What a "ticket" is here

| skill wording | our equivalent |
|---|---|
| issue / ticket | one task file `backlog/tasks/task-N - <slug>.md` (frontmatter, description, acceptance criteria, implementation notes) |
| spec / PRD | `docs/contracts/<capability>.md` (stable contract); the task names it in `contracts:` and the same branch edits it |
| issue number | `TASK-N` (assigned by the tool); the card id in the title prefix (`<id>: <what>`) names the branch `agent/<id>` |
| labels / state | `status` (Proposed, Deferred, In Progress, Done) and one goal label G-A..G-F (`backlog/config.yml`) |
| assignee | `agent:` and `branch:` frontmatter keys |
| blocked by | `dependencies:` and the `exclusive:` resources; `owns:` globs of two In Progress tasks must not overlap (`tools/backlog/check.py`) |
| close | the coordinator sets the task to Done and writes `merge_hash:` |
| PR | a branch `agent/<id>` merged by the coordinator with `Merge agent/<id>: ...` (no pull requests on the forge) |

## Conventions

- Create (coordinator): `backlog task create "<id>: <what>" -s Proposed -l G-C --ac "first step"`, then add `card`, `owns`, `exclusive`, `contracts` by hand (the CLI cannot write them). Recipes: `docs/backlog-howto.md`.
- Read: `backlog task <n> --plain`, or open the file; list open work with `backlog task list --plain`, plus `docs/plan/STATUS.md` and `docs/BACKLOG.md`.
- Comment / progress: tick the acceptance criteria of your own task; fill `result:` (say verified live or not). Anything for the coordinator goes in the agent's final report, not in files outside `owns:`.
- Check: `python3 -I tools/backlog/check.py` and `python3 -I tools/contracts/check.py` must print 0 findings.

## When a skill says "publish to the issue tracker"

Do not create files on your own. In an agent run, write the proposed task text (title, description, acceptance criteria, gates) in your report; the coordinator creates the task. In a user-run session (grill, to-spec), create a task only when the user says so, never in `.scratch/`.

## When a skill says "fetch the relevant ticket"

Read the task file named in the brief, then the contracts it lists in `contracts:`. Old cards are under tag `openspec-final`.

## Skills that do not apply

`to-tickets`, `triage`, `wayfinder`, `implement` and `implement-spec` assume a GitHub/GitLab/`.scratch` tracker and build a second work-tracking system; they are not used here (`docs/process/matt-pocock-skills-fit.md` section 4).
