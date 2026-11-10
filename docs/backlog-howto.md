# Backlog board (Backlog.md) - how to use it

> status: STAGE 3 done (openspec/ removed, tag `openspec-final`; contracts in `docs/contracts/`; the copier and ledger scan run only on that tag). Original text: STAGE 1 of docs/process/openspec-removal-plan.md (migration only). The 12 active and 5 parked cards were COPIED into `backlog/tasks/` on 2026-11-04. `openspec/`, `cmd/changecheck`, CI, the merge scripts and the pre-push hook are unchanged and still authoritative; `docs/cards.md` stays the interim list until stage 2. Tags: V verified in this job (ran the binary on the files), S from the trial report (Backlog.md 1.55.3), U unverified.

## For agents (one page; read this, the rest is on demand)

- Your task is `backlog/tasks/task-<n> - <slug>.md` (the brief names it; `backlog task <n> --plain` also works). Read its keys `owns:`, `exclusive:`, `contracts:` and the acceptance criteria. Edit only paths under `owns:`; use only resources under `exclusive:`; change a contract only if `contracts:` names it. Need more: say so in your report.
- Create a task only when the brief says so: `python3 -I tools/backlog/new_task.py "<card>: <what>" --card <card> --owns <path> --component <c> --kind <k> --ac ".."` (closed label set below; keep the title under 80 characters).
- Tick criteria with `backlog task edit <n> --check-ac <k>` (or the checkbox by hand), fill `result:` by hand (say verified live or not). After ANY CLI edit run `git diff` (expect quote-style noise) and `python3 -I tools/backlog/check.py`; a folded `title: >-` breaks the checker, rewrite it as one quoted line.
- Never run `backlog init`, `archive` or `delete`; never edit another task's file or the status; the coordinator closes tasks at merge.
- Shared files (BACKLOG.md, INDEX.md, CHANGELOG.md, OWNERS, STATUS.md) are not yours: put lines in `backlog/merge-lines/<branch>.txt` as `BACKLOG: ..`, `INDEX: ..`, `CHANGELOG: ..`.
- Before reporting: `python3 -I tools/backlog/check.py`, `python3 -I tools/contracts/check.py`, `python3 tools/merge_lines_check.py --changed`.
- Brief shape you will receive: `docs/agents/brief-template.md`.

---

## What is where

- `backlog/config.yml`: project `d.idle`, statuses Proposed, Deferred, In Progress, Done; labels: see "Labels and fields" below; `auto_commit: false`. V
- `backlog/tasks/task-N - <slug>.md`: one file per card, markdown with YAML frontmatter. V
- `tools/backlog/openspec_to_backlog.py`: the one-shot copier (dry run by default, `--apply`, `--force`; idempotent, keeps a card's TASK number). Re-running without `--force` skips cards that already have a file, so CLI edits are not overwritten.
- `tools/backlog/check.py` (+ `test_check.py`, run `python3 -I tools/backlog/test_check.py`): read-only checker. It checks the closed label scheme and fields (section "Labels and fields"), an `owns` key on every task, unique card and task ids, no folded `title: >-` block. V

## Conventions

- The card id is the title prefix: `<card-id>: <what>`. Task ids are `TASK-N` (assigned by the tool, not settable); the card id still names the branch `agent/<id>`, the `Change:` trailer and the merge subject. V
- Labels: see "Labels and fields" below. Keep a title under 80 characters: the CLI rewrites a longer title as a folded `title: >-` block on its next edit, which hides the card id from greps and from `check.py` (it now fails such a file; `new_task.py` unfolds on create; after any `task edit` on a long title look at the first lines). V (2026-11-06, TASK-24 showed a bare `>-`).
- Priority and dependencies: see "Labels and fields".
- Acceptance criteria are the tasks.md checkboxes (checked stay checked). The card's gates are listed in the description, not as criteria, so the progress count shows work, not prose.

## Labels and fields (closed scheme of 2026-11-07)

User go 2026-11-07. The label set is CLOSED: `backlog/config.yml` lists it and `tools/backlog/check.py` fails any card (not archived) that breaks the rules below. The CLI and MCP do not validate labels; the checker does (V). Cards are tracked by area: filter the board by component, kind, track or milestone.

| Family | Labels | Rule |
|---|---|---|
| Component | `bot` (the d.idle bot core: internal/, cmd/d2bot, combat, nav, town, merc, decide, ladder), `d2d` (the engine fork and its lab verbs/overlay), `container` (tester image, wizard, compose, licence pack), `watch` (watch screen, orders map, video, cards UI), `ops` (lab tooling, gates, merge pipeline, board), `docs` (docs, decisions, glossary) | exactly one per card |
| Kind | `feature`, `bug`, `probe`, `oracle`, `drill`, `research`, `chore`, `readonly`, `d2d-measure`, `model-check`, `upstream-pr`, `bot-fix` | exactly one per card |
| D2d area | `d2d-combat`, `d2d-skills`, `d2d-monsters`, `d2d-items`, `d2d-loot`, `d2d-town`, `d2d-vendor`, `d2d-movement`, `d2d-pathing`, `d2d-levels`, `d2d-network`, `d2d-time`, `d2d-data`, `d2d-ui` | one or more, only on a `d2d` card or a `fidelity` card |
| Track | `fidelity`, `gate`, `tester`, `onboarding`, `metacurrency`, `ladder` | at least one when it applies |
| Tier / gate | `T0`..`T5` (at most one), `G-A`..`G-G` (the Gate A..G groups, exactly as used) | when it applies |
| Modifier | `soak` (needs a soak slot), `parked` (kept, not ranked; also status Deferred) | when it applies |

Mapping of the retired labels: `bot-defect` is kind `bug` (or `bot-fix` for a bot change that answers a d2d deviation) with component `bot`; `upstream` is kind `upstream-pr` with component `d2d`; `drills` is kind `drill`; `tooling` is component `ops`; `decisions` is component `docs`; `watch` stays as a component; `tester`, `onboarding`, `gate` stay as tracks. Component by `owns:` paths: internal/ and cmd/ is `bot` (cmd/d2dsetup, deploy/, the image is `container`; internal/watch is `watch`; fork work is `d2d`), tools/ and ops is `ops`, docs only is `docs`. A fidelity card is `d2d` for oracle, probe, upstream and measure work on the engine, `bot` for tooling and policy work in internal/. `docs/cards.md` shows the tier, else the first label (the component), in its goal column.

Fields on every card (the checker enforces the first three for all cards, the rest for Proposed and In Progress):
- `priority`: `high` = blocks a gate or the friends/tester build; `medium` = planned; `low` = parked or nice to have.
- `dependencies`: real task ids (`backlog task edit N --dep M`), never a label or a card name.
- `assignee`: the runner or agent type that does the work: `pro-runner` (bot defects only), `credit-runner` (docs, tooling, offline Go), `live-agent` (needs a live game instance), `user` (a user decision). Required on Proposed and In Progress; Deferred and Done cards may stay empty (a Done card keeps the runner when its result line says so).
- `milestone`: the native Backlog.md mechanism. Files in `backlog/milestones/` (`backlog milestone add "<name>" -d ".."`), and the task front matter key `milestone: m-N` (`backlog task edit N --milestone m-N` also works). Active: `m-0` Gate 2 pass, `m-1` Friends build, `m-2` Fidelity Gate A, `m-3` Metacurrency v1. `check.py` fails a `milestone:` that has no file. V
- Description: first line `Size: S|M|L` (S up to 4 h, M up to 8 h, L more), then a one-sentence why, and a `Done when:` line derived from the acceptance criteria. Do not invent facts.

`docs/cards.md` is generated and gitignored (TASK-116): it is not committed and nothing regenerates it for you. Run `python3 -I tools/backlog/cards_md.py --write` locally if you want to read it; CI only checks `cards_md.py --check` (the generator runs). If a merge of main reports a conflict in `docs/cards.md` (your branch still edits it), take the deletion: `git rm --cached docs/cards.md` and keep the file out of your commit.

Creating a card: `python3 -I tools/backlog/new_task.py "<card>: <what>" --card <card> --owns <path> --component <c> --kind <k> [-l <area|track|tier|modifier>] [--priority medium] [--assignee credit-runner] [--size M] [--milestone m-N] [--dep TASK-N]`. `--component` and `--kind` are required (an old call without them stops with the allowed values); an unknown or retired label is refused. Fidelity cards: map docs/research/fidelity-cards.md. Dependencies are the `dependencies:` key, never a label.

## Standing rule: strategy changes need a paired A/B (user 2026-11-06)

A change that claims to be a strategy improvement (not a bug fix) gets a paired A/B (at least 8 seeds, paired mean with a 95 percent interval) before its long gate; a bug fix gets a short drill in its exact failing scenario. Tooling and protocol: `tools/ab/`, docs/ab-design.md (the "Variants" section covers effects that a clvl 1 run does not reach). The task for a strategy change lists the A/B as an acceptance criterion; the task for a bug fix names its drill.

## The rule that keeps the board true

Every agent job gets a task. Create it BEFORE launching the agent (`new_task.py`, status Proposed) or name an existing one in the brief; a job with no task is invisible on the board and was the main cause of the mess found on 2026-11-06 (merged work with Proposed tasks, stale statuses, 14 pieces of open work with no task). The agent ticks its boxes and fills `result:`; the coordinator sets the status at merge (Done only when every box is ticked, else In Progress with a note naming the open box). `python3 -I tools/backlog/drift.py` in the coordinator tick shows the misses. The grooming pass of 2026-11-06 is in docs/research/backlog-groom-2026-11-06.md.

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
backlog task create "<card-id>: <what>" -s Proposed -l bot -l bug -a credit-runner --priority medium --ac "first step" --ac "second step"
backlog task edit 12 -s "In Progress"            # status (must be one of config.yml)
backlog task edit 12 --check-ac 2                # tick criterion 2
backlog task edit 12 --add-label d2d-skills
backlog task edit 12 --dep 3                     # on create: --dep TASK-3
backlog task list --plain                        # grouped by status
backlog task list -s "In Progress" -l d2d --plain
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

`python3 -I tools/backlog/new_task.py "<card>: <what>" --card <card> --owns <path> --component <c> --kind <k> [--owns ..] [--exclusive ..] [--contracts ..] [-l <label>] [-s Proposed] [--priority medium] [--assignee ..] [--size M] [--ac ".."] [-d ".."]` runs `backlog task create`, inserts `card`, `owns`, `exclusive`, `contracts` before the closing `---` of the new file and runs `check.py`. `--card` and `--owns` are required. Tests: `python3 -I tools/backlog/test_new_task.py` (a fake CLI) and one live run against a scratch repo (V: the new file passed check.py with 0 findings).

## Agent instructions

`backlog init` in a scratch copy of the repo (never in the real one), 1.55.3, `--agent-instructions claude,agents,gemini,copilot --integration-mode cli --defaults`: V.

- It APPENDS a `<!-- BACKLOG.MD GUIDELINES START -->` block (about 24 lines) to the existing `AGENTS.md` and `CLAUDE.md` and creates `GEMINI.md` and `.github/copilot-instructions.md` (the same short block, 1.2 KB). It leaves `backlog/config.yml` untouched in an existing project. `cursor` writes AGENTS.md. `--integration-mode mcp` cannot be combined with `--agent-instructions` (init refuses) and writes nothing in that case.
- The block tells agents to run `backlog instructions overview` at the start and to "not edit task markdown files directly, use the CLI". That clashes with us: our custom keys (`card`, `owns`, ...) can only be set by hand, and CLAUDE.md here is the one line `@AGENTS.md`, so the generated copy in CLAUDE.md would be duplicated text. Decision: do NOT run `init` with instruction files here (keep `--agent-instructions none`, as the config requirements say) and use the one block in AGENTS.md ("Backlog.md for agents") instead.
- The ops repo `claude/agents/*.md`: d2-worker, d2-live, d2-general and d2-decomp carry the line "your job is the Backlog.md task the brief names, `backlog/tasks/<id>.md` (keys `owns:` ..)" (d2-decomp in "Conventions" and in the boilerplate); d2-merger checks the tasks; spec-sme and grill only mention tasks. None tells agents HOW to edit tasks beyond that line, so the update is one added sentence at the end of that line in the four agents: "Task how-to: the 'Backlog.md for agents' block in AGENTS.md." (not edited in this job).
