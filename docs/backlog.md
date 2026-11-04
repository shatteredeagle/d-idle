# Backlog board (Backlog.md) - how to use it

> status: STAGE 1 of docs/process/openspec-removal-plan.md (migration only). The 12 active and 5 parked cards were COPIED into `backlog/tasks/` on 2026-11-04. `openspec/`, `cmd/changecheck`, CI, the merge scripts and the pre-push hook are unchanged and still authoritative; `docs/cards.md` stays the interim list until stage 2. Tags: V verified in this job (ran the binary on the files), S from the trial report (Backlog.md 1.55.3), U unverified.

## What is where

- `backlog/config.yml`: project `d.idle`, statuses Proposed, Deferred, In Progress, Done; labels G-A..G-F (goals of ADR 0015), `parked`, `upstream`, `soak`; `auto_commit: false`. V
- `backlog/tasks/task-N - <slug>.md`: one file per card, markdown with YAML frontmatter. V
- `tools/backlog/openspec_to_backlog.py`: the one-shot copier (dry run by default, `--apply`, `--force`; idempotent, keeps a card's TASK number). Re-running without `--force` skips cards that already have a file, so CLI edits are not overwritten.
- `tools/backlog/check.py` (+ `test_check.py`, run `python3 -I tools/backlog/test_check.py`): read-only checker, not wired into CI yet. It checks: at most 12 non-parked tasks in Proposed/Deferred/In Progress, exactly one G-A..G-F label on each of those, an `owns` key on every task, unique card ids in titles. Result on the 17 migrated tasks: 0 findings. V

## Conventions

- The card id is the title prefix: `<card-id>: <what>`. Task ids are `TASK-N` (assigned by the tool, not settable); the card id still names the branch `agent/<id>`, the `Change:` trailer and the merge subject. V
- One goal label per open task; parked tasks carry `parked` and (for now) no goal label because docs/cards.md gives them none.
- Priority from rank: 1-4 high, 5-8 medium, 9-12 low, parked low. Dependencies only where a card says "After <card>" (today: m3-act1run after m3-andariel).
- Acceptance criteria are the tasks.md checkboxes (checked stay checked). The card's gates are listed in the description, not as criteria, so the progress count shows work, not prose.

## Custom frontmatter fields (ours; the tool has no native field)

`card`, `owns` (list), `exclusive` (list), `branch`, `agent`, `result` (say verified live or not), `live`, `merge_hash`, `overlap` (declared overlap paragraph), `skip_specs`, `source` (old openspec path).

| tool | can read them |
|---|---|
| a text editor, `grep`, `tools/backlog/check.py`, a future changecheck port | yes (plain YAML in the file) |
| `backlog task N --plain`, `list --json`, `task N --json`, search | no: `--json` shows only native fields. V |
| `backlog task edit` | keeps them untouched when it rewrites the file. S |
| `backlog task create` | cannot write them: add the lines by hand after creating (at least `card`, `owns`). The checker fails a task with no `owns`. |

## CLI recipes

Run the binary by absolute path or from PATH once the user installs it; never from inside an untrusted download directory. Never pipe its output to `head` (see problems).

```
backlog task create "<card-id>: <what>" -s Proposed -l G-C --priority medium --ac "first step" --ac "second step"
backlog task edit 12 -s "In Progress"            # status (must be one of config.yml)
backlog task edit 12 --check-ac 2                # tick criterion 2
backlog task edit 12 --add-label upstream
backlog task edit 12 --dep 3                     # on create: --dep TASK-3
backlog task list --plain                        # grouped by status
backlog task list -s "In Progress" -l G-E --plain
backlog task list --json                         # versioned (schemaVersion 1); no description
backlog task list --ready --plain                # not blocked by an open dependency
backlog task 12 --json                           # one task with description and dependency graph
backlog search nexthit --plain
```

Statuses and labels cannot be added by the CLI: hand-edit `backlog/config.yml`. S

## Git rules

- Keep `auto_commit: false`. With it on, the tool commits on every edit and fails without a git identity (it failed three times in the trial; the edit still applied and stayed staged). S
- Commit task files yourself with the agent identity (`git -c user.name=Claude -c user.email=noreply@anthropic.com commit`), like any other file.
- Worktrees are fine: the tool reads and writes the worktree's own `backlog/`. Two agents editing different files merge cleanly; the same field gives ordinary conflict markers inside the frontmatter. S

## Known problems

- Piping to `head` (or any early-closing reader) crashes the Bun binary with a stack dump (EPIPE). Cosmetic; redirect to a file instead. S
- `backlog browser` binds 127.0.0.1 only and has NO authentication; an unauthenticated POST created a task in the trial. LAN access would need a reverse proxy, which would expose an unauthenticated write API. The user chose to SKIP the phone view (2026-11-04). S
- Ids are `TASK-N` and cannot be set; creating tasks on two branches gives distinct ids locally, a race between machines is untested. U
- No link to spec deltas, `merge-lines.txt` or `docs/OWNERS`; cards with `specs/` directories keep them under `openspec/` for now. S
- The binary is a prebuilt Bun executable from npm; treat it as third-party software. No network features are used or needed.

## Plan reference

docs/process/openspec-removal-plan.md (branch agent/removalplan2 until merged): stage 1 = this migration (copy, nothing removed); later stages switch consumers (changecheck, CI, merge scripts, pre-push hook, STATUS block) to read `backlog/`, then retire `openspec/`. None of that is done.
