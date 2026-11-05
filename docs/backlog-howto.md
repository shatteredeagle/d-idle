# Backlog board (Backlog.md) - how to use it

> status: STAGE 3 done (openspec/ removed, tag `openspec-final`; contracts in `docs/contracts/`; the copier and ledger scan run only on that tag). Original text: STAGE 1 of docs/process/openspec-removal-plan.md (migration only). The 12 active and 5 parked cards were COPIED into `backlog/tasks/` on 2026-11-04. `openspec/`, `cmd/changecheck`, CI, the merge scripts and the pre-push hook are unchanged and still authoritative; `docs/cards.md` stays the interim list until stage 2. Tags: V verified in this job (ran the binary on the files), S from the trial report (Backlog.md 1.55.3), U unverified.

## What is where

- `backlog/config.yml`: project `d.idle`, statuses Proposed, Deferred, In Progress, Done; labels G-A..G-F (goals of ADR 0015), `parked`, `upstream`, `soak`; `auto_commit: false`. V
- `backlog/tasks/task-N - <slug>.md`: one file per card, markdown with YAML frontmatter. V
- `tools/backlog/openspec_to_backlog.py`: the one-shot copier (dry run by default, `--apply`, `--force`; idempotent, keeps a card's TASK number). Re-running without `--force` skips cards that already have a file, so CLI edits are not overwritten.
- `tools/backlog/check.py` (+ `test_check.py`, run `python3 -I tools/backlog/test_check.py`): read-only checker, not wired into CI yet. It checks: exactly one G-A..G-G label (no cap on open cards since grill 0005) on each of those, an `owns` key on every task, unique card ids in titles. Result on the 17 migrated tasks: 0 findings. V

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
| `backlog task edit` | keeps them untouched when it rewrites the file. V (2026-11-05, see below) |
| `backlog task create` | cannot write them: add the lines by hand after creating (at least `card`, `owns`). The checker fails a task with no `owns`. |

## Install

Pinned to 1.55.3 everywhere. User-level, no root, no sudo: `tools/backlog/install.sh` runs a user-level `npm install -g --prefix ~/.local` of the npm package `backlog.md` at exactly 1.55.3 (npm checks the registry integrity hash; the prebuilt Bun binary for the platform comes as the optional dependency `backlog.md-<os>-<arch>` of the same version, MIT licence). It is idempotent (prints `already installed` when the version matches) and prints the path and version. V: installed and run on the dev box (linux-x64) and on the worker host on 2026-11-05; both give `~/.local/bin/backlog` and `backlog --version` = 1.55.3.

- Dev box: `tools/backlog/install.sh`
- Worker host: from inside a worktree, `labrun <worker> -- tools/backlog/install.sh` (the worker name from `labrun hosts`) (touches only `~/.local`; never its cache or game dirs)
- Contributors (Linux or macOS, needs Node.js and npm): `sh tools/backlog/install.sh`, then put `~/.local/bin` on PATH. Another prefix: `sh tools/backlog/install.sh /some/dir`. No Node: `install.sh` does NOT fall back; it checks `command -v npm`, prints "npm not found; install Node.js (or download the release binary from the project's GitHub releases)" to stderr and exits 1, installing nothing. V (read the script 2026-11-05). The no-Node route is manual: download the release binary of 1.55.3 for your platform from the project's GitHub releases page, check it against the checksums listed there, `chmod +x`, put it on PATH as `backlog`, and confirm `backlog --version` prints 1.55.3. U (macOS path and the release asset names not tried; S from the npm optional dependency list: darwin-x64, darwin-arm64).
- Upgrade only by changing `VERSION` in the script and this page together.
- `backlog init` is not needed (`backlog/config.yml` is in the repo); if you ever run it, use `--agent-instructions none`.

## What the CLI can and cannot do with our tasks

Checked 2026-11-05 on the 21 real tasks in a scratch worktree (1.55.3).

- Reads: `task list --plain`, `task N --plain`, `task list --json`, `search <word> --plain` all work (rc 0). V. `task N --plain` prints the file path, native fields, description and criteria, no custom keys. V. `list --json` has schemaVersion and only native fields (id, title, status, labels, priority, ordinal, ac counts, isReady, ...). V.
- `backlog board view` with output not on a terminal prints a Markdown kanban table (same as `board export`) after a few "Loading tasks from ... branches" lines; the interactive TUI needs a TTY. V.
- `backlog browser --no-open --port 6420`: binds 127.0.0.1 only; `/` gave 200 and `/api/tasks` gave 200 with the task list as JSON. V. Stopped after the check. Still no authentication.
- `task edit N -s <status>`, `--add-label`, `--check-ac`, `-a`: custom keys (`card`, `owns`, `exclusive`, `contracts`, `result`, ...), their order and the body are kept. V. It DOES rewrite the frontmatter formatting: double quotes become single quotes or none (`title`, labels), `assignee: []` becomes a list when set, and `updated_date` is refreshed to the current time. Expect noise in the diff on the first edit of each file; the data is the same. V.
- `task create "x" -s Proposed -l G-G --ac "a"`: writes `task-N - <slug>.md` with id, title, status, assignee, created_date, labels, dependencies, ordinal and the criteria block. No `card`, `owns`, `exclusive`, `contracts`, no `priority` unless given. `python3 tools/backlog/check.py` then fails it with `no owns key`. V. The id is the next free number (22 here).
- The CLI has no way to set or show our custom keys. V (no option; not in `--json`).
- `auto_commit: false` kept: none of the commands above created a commit. V.

Config (`backlog config list`) matches what we need: check_active_branches true, remote_operations true, auto_commit false, bypass_git_hooks true, filesystem_only false, active_branch_days 30, default_status Proposed, labels and statuses as in config.yml. V. Nothing to change.

## Wrapper conventions (recommended)

1. Create with the CLI, then add the custom keys by hand in the same edit session: at least `card:` and `owns:` (a list), plus `exclusive:`, `contracts:` when they apply; run `python3 tools/backlog/check.py` before committing. Or write the whole file by hand from a sibling task.
2. Use the CLI for status, labels, criteria ticks and assignee (`task edit`); it keeps custom keys. Commit the file yourself.
3. For anything that must see custom keys (owns, exclusive, contracts) read the files or use `tools/backlog/check.py`; do not build on `--json`.
4. After a CLI edit run `git diff` once: quote-style churn and `updated_date` are expected, anything else is a finding.
5. Do not run the browser on the worker host or expose it; use it on the dev box only, loopback, short lived.
6. The MCP server is evaluated (section "MCP server (verified)"); `tools/backlog/new_task.py` is the create wrapper (card, owns, exclusive, contracts from flags).

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

## Drift report

`python3 -I tools/backlog/drift.py [--repo DIR] [--ref origin/main] [--days 14]` is read-only (it never edits a task and exits 0 always). One line per finding, `<code> TASK-N <card> [<status>]: <what>`: A = Proposed/In Progress with every acceptance box ticked; B = Done with an unticked box; C = In Progress whose `owns:` paths have no commit on the ref in the last 14 days; D = a merge subject on the ref names the task (`Merge agent/<card>`, `task-N`) but the task is not Done. The dispatcher tick prints `DRIFT n tasks` plus the first three lines under `--delta` when the count changed. Tests: `python3 -I tools/backlog/test_drift.py`. V (ran on this repo and on a temp-repo fixture).

## Status rules

- A task is Done when its code or docs are merged on main AND every acceptance box that can be checked offline is ticked. A criterion that only a live run can satisfy is not left as an unticked box on a Done task: remove it (`task edit N --remove-ac`, or by hand) and carry it in the Implementation Notes as `merged, offline tested; live proof pending: <log line / gate>`. Say verified live or not in `result:`.
- A task stays In Progress while real work is open (an unbuilt slice, a drill row still to run, a verdict table still to fill).
- A fix card for an upstream PR is Done when that PR is merged upstream (`gh pr list --repo psi29a/d2decomped --author shatteredeagle --state all`), the note names the PR number; it stays In Progress while the PR is only a draft or open.
- `python3 -I tools/backlog/drift.py` (read-only, run it in the tick) reports the mismatches: A all boxes ticked but not Done, B Done with an unticked box, C no recent commit on owned paths, D merged branch but not Done. Fix them with `task edit`; do not leave a D line without a reason in the task's note.
- Caution: some old tasks have duplicate `#N` criterion numbers (tasks 10, 11, 18 before 2026-11-05); `--remove-ac N` and `--check-ac N` act by position, so read `task N --plain` first and verify with `git diff`.

## Verified edge cases

Checked 2026-11-05 in scratch worktrees and clones of main (backlog 1.55.3).

- Cross-branch id allocation. V. `new_task.py` (CLI) in worktree A gave TASK-22; in worktree B on another local branch, created afterward, TASK-23; a third and fourth gave 24 and 25, and a clone whose origin held those local branches gave 26. The CLI reads the other local branches and remote-tracking branches (`check_active_branches`, 30 days), also while the first task was not yet committed. So two worktrees of one repo do not collide. It DOES collide when the other branch is not visible: a single-branch clone of main created TASK-22 again, the same id as an unmerged branch. In this project agents never push, so a branch only another clone has seen is the only case. Rule: ids come from the CLI at creation; create tasks from a checkout that has all live branches (the main clone and its worktrees), `git fetch` first on a separate clone. Backstop: `tools/backlog/check.py` now fails on a duplicate `TASK-N` (new, with a test), so a merge of two colliding branches is caught by `tools/ci/check.sh`; fix by renumbering one file (id, filename) before merging. Not implemented: a max(id)+1 wrapper, because the CLI already covers every branch the repo can see.
- The Pro runner and MCP. V (read `tools/plan-agent.py` in the ops repo, no change made). It starts `claude -p ... --setting-sources user --settings <job settings> --disable-slash-commands --strict-mcp-config` (the health probe uses `--strict-mcp-config` too) and passes no `--mcp-config`. With `--strict-mcp-config` only servers named by `--mcp-config` are used, so a project `.mcp.json` is NOT loaded, and `--setting-sources user` also skips project settings. A Pro-runner job therefore has no Backlog MCP; it uses the CLI (`backlog task ...`, `tools/backlog/new_task.py`, `check.py`) through Bash, subject to the job's permission list (the Bash allow for `backlog` is not checked here: U). Giving it MCP would need an ops-repo change (`--mcp-config <file>`); not made.
- No-Node install: see "Install" above (U for macOS).

## Known problems

- Piping to `head` (or any early-closing reader) crashes the Bun binary with a stack dump (EPIPE). Cosmetic; redirect to a file instead. S
- `backlog browser` binds 127.0.0.1 only and has NO authentication; an unauthenticated POST created a task in the trial. LAN access would need a reverse proxy, which would expose an unauthenticated write API. The user chose to SKIP the phone view (2026-11-04). S
- Ids are `TASK-N` and cannot be set; creating tasks on two branches gives distinct ids locally, a race between machines is untested. U
- No link to spec deltas, `merge-lines.txt` or `docs/OWNERS`; cards with `specs/` directories keep them under `openspec/` for now. S
- The binary is a prebuilt Bun executable from npm; treat it as third-party software. No network features are used or needed.

## Plan reference

docs/process/openspec-removal-plan.md (branch agent/removalplan2 until merged): stage 1 = this migration (copy, nothing removed); later stages switch consumers (changecheck, CI, merge scripts, pre-push hook, STATUS block) to read `backlog/`, then retire `openspec/`. None of that is done.

## Config requirements (from the read-only mirror test, 2026-11-04)

- `backlog/config.yml` sets `check_active_branches: true` and `remote_operations: true`; with either off, cards that exist only on unmerged `agent/*` branches do not show in the web board. The fetch remote must be named `origin`.
- Cards are visible only when COMMITTED on the agent branch (uncommitted worktree edits are invisible): workers commit card changes with their other work.
- Keep `auto_commit: false` and `bypass_git_hooks: true`: the merge queue needs main == origin/main and a clean clone, and Backlog.md auto-commits would break that.
- Pin Backlog.md to one version everywhere (1.55.3). Run `backlog init` with `--agent-instructions none` (AGENTS.md is guard-protected; init otherwise writes agent instruction files).
- `backlog task list` (CLI) does NOT show branch-only cards; only the web board does. Tooling that needs the cross-branch view uses the web API or reads the files, not the CLI list.
- `active_branch_days` (default 30) limits which branches count; very old agent branches do not show.
- The web board is a read-only mirror on another machine fetched every 60 s (no phone view, no write API exposed).

## MCP server (verified)

Checked 2026-11-05, Backlog.md 1.55.3, in a scratch worktree on a branch, with a stdlib JSON-RPC stdio client (initialize, tools/list, tools/call) and with `claude -p` (Haiku). Tags V (ran it), U (not tried).

- Start: `backlog mcp start` (stdio, no port, no daemon, no auth; `backlog mcp --help` has only `start`). It works on the cwd's repo, also from a subdirectory (V: a `claude -p` call from `internal/` read TASK-10). It sends instructions "list the resources and read the first one". V
- Tools (24): `get_backlog_instructions`, `task_create`, `task_list`, `task_search`, `task_edit`, `task_view`, `task_archive`, `task_delete`, `task_restore`, `milestone_list|add|rename|remove|archive`, `definition_of_done_defaults_get|upsert`, `document_list|view|create|update|archive|restore|delete|search`. V
- `task_create` fields: title, description, status, priority, labels, assignee, dependencies, references, documentation, acceptanceCriteria, ordinal, milestone, parentTaskId, finalSummary, ... No free-form frontmatter. `task_edit` adds status, labels, notesSet/notesAppend (the "Implementation Notes" block), planSet, comments, acceptanceCriteriaSet|Add|Remove|Check|Uncheck, finalSummary. V (schema read from tools/list)
- Create (`task_create` with status Proposed, label G-G, priority, two criteria, a reference): wrote `task-22 - mcptest-scratch-task.md` with id, title, status, created_date, labels, ordinal and the criteria block; NO card/owns/exclusive/contracts (check.py would fail it). V
- Edit on a real task (TASK-10: status, check criterion, `notesAppend`): `git diff` shows the custom keys (card, owns, exclusive, branch, agent, result, live, ...) untouched and in order. Same churn as the CLI: title quotes become single, `- "G-C"` becomes `- G-C`, `updated_date` refreshed; `notesAppend` appends a `## Implementation Notes` block with SECTION:NOTES markers at the end of the file (a place for notes we do not use today). V
- Edit on the new task: status In Progress, `notesSet`, `acceptanceCriteriaCheck`, labels `["G-G","soak"]` all landed. V
- Config: statuses are validated (`Bogus` is rejected with the allowed list; `Draft` is also accepted because the tool has it built in, not in our config). Labels are NOT validated: label `nonsense` was accepted on create. So G-A..G-G discipline stays with check.py. Prefix `task`, default status Proposed, ids = next free number (22, then 23 after the archive of 22: ids are not reused) are respected. V
- `task_list`/`task_search`/`task_view` work (text output with file path, status, labels, criteria counts; no custom keys). `task_archive` moves the file to `backlog/archive/tasks/` (the CLI/MCP archive path is not one our checks read; do not use it for finished cards, the coordinator closes by status). V
- Commits: none. `auto_commit: false` holds; every call only changed files in the worktree. V
- Branch behaviour: ran inside a worktree on `agent/backlogmcp`; tasks of that worktree's `backlog/tasks` were listed and written there. Cross-branch id allocation (check_active_branches) with another unmerged branch: not tried. U. File locking between two servers on one repo: not tried. U
- End to end from a worktree: `.mcp.json` at the worktree root with `{"mcpServers":{"backlog":{"command":"backlog","args":["mcp","start"]}}}` made `claude -p --permission-mode dontAsk --allowedTools mcp__backlog__task_view` call the tool and return TASK-10 (V; the same via `--mcp-config` plus `--strict-mcp-config`). Interactive sessions ask once to approve a project-scope server. V for -p, S for the prompt.
- Where it does not fit: the credit runner (`tools/api-agent.sh`) starts `claude` with `--strict-mcp-config` and `--setting-sources user`, so a project `.mcp.json` is ignored and its permission list has no `mcp__backlog__*` rule: no MCP there without changing the runner (S, docs/api-agent.md in the ops repo). The Pro runner sandbox: not checked here, U (stdio MCP is a local child process, so it should work if `backlog` is on PATH).

### Decision: use with the wrapper, CLI stays the default

MCP does nothing the CLI cannot do (same code, same file edits, same churn) and cannot write our keys. Agents keep using the CLI (it works in every runner, `--plain` output is the same text). The committed `.mcp.json` lets main-account sessions and subagents call `task_view`, `task_list`, `task_search` and `task_edit` as tools without a shell; they must still create tasks with `tools/backlog/new_task.py` and edit custom keys by hand. Do not use `task_archive`, `task_delete` or the document/milestone tools.

### The create wrapper

`python3 -I tools/backlog/new_task.py "<card>: <what>" --card <card> --owns <path> [--owns ..] [--exclusive ..] [--contracts ..] [-l G-C] [-s Proposed] [--priority low] [--ac ".."] [-d ".."]` runs `backlog task create`, inserts `card`, `owns`, `exclusive`, `contracts` before the closing `---` of the new file and runs `check.py`. `--card` and `--owns` are required. Tests: `python3 -I tools/backlog/test_new_task.py` (a fake CLI) and one live run against a scratch repo (V: the new file passed check.py with 0 findings).

## Agent instructions

`backlog init` in a scratch copy of the repo (never in the real one), 1.55.3, `--agent-instructions claude,agents,gemini,copilot --integration-mode cli --defaults`: V.

- It APPENDS a `<!-- BACKLOG.MD GUIDELINES START -->` block (about 24 lines) to the existing `AGENTS.md` and `CLAUDE.md` and creates `GEMINI.md` and `.github/copilot-instructions.md` (the same short block, 1.2 KB). It leaves `backlog/config.yml` untouched in an existing project. `cursor` writes AGENTS.md. `--integration-mode mcp` cannot be combined with `--agent-instructions` (init refuses) and writes nothing in that case.
- The block tells agents to run `backlog instructions overview` at the start and to "not edit task markdown files directly, use the CLI". That clashes with us: our custom keys (`card`, `owns`, ...) can only be set by hand, and CLAUDE.md here is the one line `@AGENTS.md`, so the generated copy in CLAUDE.md would be duplicated text. Decision: do NOT run `init` with instruction files here (keep `--agent-instructions none`, as the config requirements say) and use the one block in AGENTS.md ("Backlog.md for agents") instead.
- The ops repo `claude/agents/*.md`: d2-worker, d2-live, d2-general and d2-decomp carry the line "your job is the Backlog.md task the brief names, `backlog/tasks/<id>.md` (keys `owns:` ..)" (d2-decomp in "Conventions" and in the boilerplate); d2-merger checks the tasks; spec-sme and grill only mention tasks. None tells agents HOW to edit tasks beyond that line, so the update is one added sentence at the end of that line in the four agents: "Task how-to: the 'Backlog.md for agents' block in AGENTS.md." (not edited in this job).
