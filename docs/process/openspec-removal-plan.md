# OpenSpec removal plan (proposal for the user), revised for Backlog.md

> status: PROPOSAL, written 2026-11-04 after the user retired OpenSpec for new work; revised the same day after the user chose Backlog.md as the card tracker (trial first). Nothing here is applied. Stages marked GO need the user's go. Tags: V verified (file:line read in this session), S secondary (read in a doc, not re-run), U unverified (includes every Backlog.md fact beyond the README summary).

Note on the decision record: `docs/decisions.md` has no 2026-11-04 OpenSpec line yet (V: its only OpenSpec mention is line 17, the soak-minutes card link). The coordinator should add it (proposed text at the end).

Tension: this doc lives under `docs/process/` and is not a card, so `changecheck` neither requires nor owns it. `docs/**` is owned by `main` in `docs/OWNERS` (V: docs/OWNERS line 5), so no OWNERS row is needed. Until removal, "0 warnings" still means the old rules; this plan changes none of them.

## 1. Inventory

### This repo (public)

| item | count | tag |
|---|---|---|
| files under `openspec/` | 2290 | V (`find openspec -type f`) |
| active card dirs `openspec/changes/*` (not `archive`) | 36: 19 in-progress, 13 deferred, 4 proposed | V (`status:` lines of tasks.md) |
| archived cards `openspec/changes/archive/*` | 639 entries | V |
| spec capabilities `openspec/specs/*` | 17 dirs, 17 files (one spec.md each) | V |
| other files in `openspec/` | `README.md`, `.strict.yaml`, `.adopted`/`.last-check` (gitignored, line 27 of .gitignore) | V |
| `cmd/changecheck` | 1931 lines in 14 files: about 1436 code, 495 tests | V (`wc -l`) |
| CI steps that use it | 1 step in `checks` (`.github/workflows/ci.yml:41-42`), a node setup step for the pinned validator (`:40`), a vet step (`:51-52`), a docs-only skip filter naming `openspec/` (`:74`) | V |
| local gates | `tools/hooks/pre-push` (runs `changecheck -cli` then `tools/openspec.sh validate --all --strict`), `tools/ci/check.sh:29-30` (merge-lines lint only) | V |
| files outside `openspec/` that mention openspec/changecheck | 231 (192 under `docs/`, mostly research and probe docs citing card paths) | V (`grep -rIl`) |
| Go source mentioning a card or spec path | 5 comment-only hits (`internal/d2dsoakstat`, `internal/d2dreplay`, `internal/bot/kitgate_test.go`, `cmd/d2dview/brand_test.go`) | V |
| generated state | `docs/plan/STATUS.md` block between `statusgen:begin/end` (line 40) is written by `changecheck status -write` | V |

### The ops repo (private; summary only)

| item | count | tag |
|---|---|---|
| tracked files | not counted (listing denied for this job) | U |
| files mentioning openspec, changecheck or "change card" | outside its `state/` dir: about 30 files (agent definitions, dispatcher and merge docs, tools); including `state/` (old briefs, API-agent runs): about 100 | V (`grep -rIl`, two counts) |
| agent definitions under `claude/agents/` | 8 of 11 mention cards (d2-worker, d2-general, d2-live, d2-decomp, d2-reviewer, d2-merger, spec-sme, grill) | V |
| tools that read the card tree | `tools/merge.sh`, `tools/mergeq.py`, `tools/gate-run.sh`, `tools/baseline-collect.sh`, `tools/merge_lines.py`, `tools/merge_classify.py`, `tools/api-agent.py` (allow-list), `tools/agentstatus/{agentstatus.py,cardboard.py,publish-cards.sh,boardviews.py,changes.html}`, `tools/golive-dry.sh` | V |

### Other checkouts

`d2lod-bot-new`: not present or not readable from this job (U). Say so to the user; if it exists, repeat section 1 there before stage 5.

## 2. Dependency map: what breaks if `openspec/` or `changecheck` goes

### Public repo

| breaks | where | why |
|---|---|---|
| CI `checks` job fails (missing `./cmd/changecheck`, missing `openspec/`) | `.github/workflows/ci.yml:41-42`, `:51-52` | step runs `go run ./cmd/changecheck -cli`; vet names the package. V |
| pre-push hook blocks pushes to main | `tools/hooks/pre-push:22-37`, test `tools/hooks/test-pre-push.sh` | runs changecheck and `tools/openspec.sh`. V |
| the pinned validator wrapper | `tools/openspec.sh` | `openspec validate --all --strict`; called by CI, hook, merge gate, README. V |
| STATUS generation and its freshness check | `cmd/changecheck/status.go:97-122`; `docs/plan/STATUS.md:40` | block lists cards; `checkStatus` warns when stale. The coordinator STATUS commits depend on it. V |
| OWNERS consistency | `cmd/changecheck/checks.go:44-108` | `docs/OWNERS` rows are hand-edited and checked against card `owns:`. Without cards the file has no meaning (its `-` rows only mark base dirs). V |
| card lint, overlap, spec lint, strict dates | `cmd/changecheck/card.go`, `checks.go:13-35`, `specs.go`, `strict.go`, `openspec/.strict.yaml` | including `allow_merges` list that merge.sh edits. V |
| merge subject rule | `cmd/changecheck/checks.go:186-232` | see section 4. V |
| `changecheck new` (card scaffolding) | `cmd/changecheck/new.go` | referenced by `docs/agents/issue-tracker.md`. V |
| board JSON | `cmd/changecheck/jsonout.go`, `needsyou.go` | `changecheck json` feeds the board; `needsyou.go` also parses `docs/needs-you.md`, so that part must move before deleting. V |
| merge-lines lint | `tools/merge_lines_check.py:49,62`, `tools/ci/check.sh:29-30`, `tools/ci/tests/test_merge_lines_check.py` | looks for `openspec/changes/*/merge-lines.txt`. If cards go, shared-file lines need a new home. V |
| credit tool | `tools/credit/credit.py:110-158` | reads `credit:` blocks of cards. V |
| session brief | `tools/session-brief.sh:11-14` | prints `openspec/.last-check`. V |
| chunker | `tools/contrib/chunker.py:366` | reads `openspec/changes`. V |
| agent-process tools (comments only) | `tools/agentrun.sh:2`, `agentreap.sh:2`, `d2dslot.sh:2,86`, `d2dfarm.sh:2` | comment pointers to cards; harmless, dead links after archive move. V |
| `tools/ops-cutover.sh:54` | text hint | harmless. V |
| docs | `AGENTS.md:3,7-12,67`; `docs/agents/issue-tracker.md`, `docs/agents/domain.md`; `docs/ci.md:15-17,39,47`; `openspec/README.md`; `docs/process/{openspec-reference,openspec-fit-notes,spec-approach-evaluation,trial-notes,ci-checks,hooks}.md`; `docs/decisions.md:17` and other links into `openspec/changes/...`; `GLOSSARY.md`; `PLAN.md`; `docs/BACKLOG.md`, `CHANGELOG.md` | dead links and wrong instructions. V/S |
| contracts | `openspec/specs/*` (soak-gates, death-rule, backend-contract, fidelity-gate, merc-policy, early-game-policy, lab-rules, ...) cited by `AGENTS.md:10`, code comments, ADRs | these are the real content; they must be moved, not deleted (section 5, stage 3). V |
| public-split manifest | `docs/process/public-manifest.json` does not name openspec (grep found none) | no break. V |

### The ops repo (S: read by grep this session, not run)

| breaks | where |
|---|---|
| merge script: archive step, `--card none` allow_merges edit, `.strict.yaml` required, `Change:` trailer, openspec gate | `tools/merge.sh:7-9,203-205,224,238-243,278-281,311,396-415,437,487,496` |
| merge queue: stale OWNERS filter, card-dir test for the `Change:` trailer, card dir in snapshot | `tools/mergeq.py:292,302,330,353-359` |
| gate-run and baseline record branches create a card, `.openspec.yaml`, OWNERS rows | `tools/gate-run.sh:184-239,325-364`; `tools/baseline-collect.sh:37-102` |
| merge classification (docs-only includes `openspec/`) | `tools/merge_classify.py:12` |
| board `/changes` page and `/cards` approval page | `tools/agentstatus/publish-cards.sh:12`, `agentstatus.py` (routes `/changes`, `/cards`, `/api/cards`), `cardboard.py:34-48` (scans `openspec/changes` in git) |
| API-agent permission allow-list | `tools/api-agent.py:296` (names `tools/openspec.sh`, `tools/changecheck`) |
| dispatcher text (card board reminder, card-based launch, merge queue `--card`) | `docs/dispatcher.md` lines 11, 14, 27; `docs/process/dispatcher-brief*.md`; `docs/merging.md`; `docs/gate-run.md` |
| agent definitions: "Card conventions" bullet and "your card is openspec/changes/<id>/" | `claude/agents/{d2-worker:21-22, d2-general:14-15, d2-live:18-19, d2-decomp:18-31, d2-reviewer:14, d2-merger:12-33}.md`; `spec-sme.md` (whole agent) |
| brief files | `briefs/*.md` has 2 mentions only (S); `state/briefs/*` historical, leave alone |

## 3. Card-tracking replacement: Backlog.md (decided)

Decision (the user, 2026-11-04): the replacement is Backlog.md (https://github.com/MrLesk/Backlog.md, MIT). Facts taken from its README as given to this job: tasks are markdown files in the repo; `backlog browser` serves a web board; there is a CLI and an MCP interface; `backlog init` sets up a repo. Everything else about Backlog.md below is tagged U (unverified, no network access in this job) and is a question for the trial (section 7).

Why not the others (short): GitHub issues reverse the standing "nothing is posted to GitHub without the user's go" rule and headless agents cannot call `gh` (S); Beads and Vibe Kanban were not chosen; a single `docs/cards.md` table is the interim list only (it is migrated, then replaced) because one shared file conflicts between agents and has no per-job checklist, and the board `/cards` page has no store in the repo. The end state is Backlog.md.

What a card carries today (V: `docs/agents/issue-tracker.md`): id, agent, branch, status, `owns:`, `exclusive:`, `gates:`, tasks, `result:`, delta spec. What the process really uses: the brief, the lock on shared resources (`exclusive:`), the result line, the merge naming. Path ownership is enforced by the briefs; `owns:` lint is mostly bookkeeping (S, judgement).

## 4. Migration spec: the open cards to Backlog.md

### 4.1 Scope

Source: `docs/cards.md` on branch `agent/cardsapply` (V: read this session). It lists 12 active rows, 5 parked, 4 "needs a decision" (fixcastgap, fixmanamodel, fixskilllvl, skmeleewire; plus d2d-default-backend as a fifth entry in that table, so the user's "4 left open" is 4 or 5, confirm), and one resolved (gate1run-20261102). The 14 archived cards of the 36 are already out of the active set. Migrate: 12 active + 5 parked = 17 task files; the "needs a decision" cards migrate only after the user decides (status To Do with label `needs-decision` is the fallback).

Archived work: the 639 `openspec/changes/archive/*` entries are NOT migrated. They stay as history under a tag (`openspec-final`, stage 3) and in git. Links from docs into `openspec/changes/archive/...` are rewritten to the tag path or left as dead history links (stage 3).

### 4.2 Where the files live (all U until the trial confirms)

- Folder: the Backlog.md default `backlog/` at the repo root (U: layout assumed `backlog/tasks/`, `backlog/config.yml`, a drafts and an archive folder, and one file per task named from id and title). Alternative if the trial shows a configurable root: `docs/backlog/`. Recommendation: keep the default root; `docs/**` stays prose.
- `docs/OWNERS` row for the folder: `backlog/** main`-style row (the coordinator owns the folder; agents edit only their own task file, see 7.3).
- `docs/cards.md` stays until all 17 are migrated, then becomes a one-line stub pointing at the folder (AGENTS.md rule: stub, never delete docs).
- Task files carry a link back to the old card dir (`openspec/changes/<id>/`) for the proposal text; the old dirs stay in place until stage 3.

### 4.3 Task file format (U: from Backlog.md conventions as the author knows them; trial must confirm names)

Markdown with YAML frontmatter, then free text sections:

```
---
id: task-<n>              # Backlog.md assigns; our card id goes in the title and a label
title: "<card id>: <what>"
status: In Progress       # Backlog.md default set assumed: To Do, In Progress, Done (U)
assignee: []
labels: [G-E, upstream]   # goal label + optional topic
dependencies: []          # other task ids
priority: medium          # assumed high|medium|low (U)
---
## Description
## Acceptance Criteria    # checkbox list: our tasks.md checkboxes
## Implementation Notes   # result: line, live:, merge: hash, overlap paragraph
```

### 4.4 Field mapping

| our card field | Backlog.md | note |
|---|---|---|
| id (kebab, equals agent name) | title prefix `<id>:` plus label `card:<id>`? Backlog id (`task-N`) is auto (U) | keep our id stable: it names branch `agent/<id>`, the `Change:` trailer and merge subject. The trial must say whether task ids can be set by hand |
| goal G-A..G-F | label `G-A`..`G-F` | one goal label per task; rule "at most 12 active" becomes a board filter plus a reviewer check (no tool enforces it, U) |
| status in-progress | In Progress | |
| status proposed / deferred | To Do (deferred also label `deferred`) | |
| status parked | To Do plus label `parked` (or Backlog.md draft/archive if the trial shows one fits) | parked is off the cap |
| status done / archived | Done, then Backlog.md archive (U) | archived openspec cards are not migrated |
| `agent:`, `branch:`, `worktree:` | assignee = agent name; branch in Implementation Notes (`branch: agent/<id>`); worktree dropped (machine path) | |
| `owns:` list | Implementation Notes block `owns:` as a yaml list in the body, or a frontmatter field if the file tolerates unknown keys (U) | only used if the overlap check survives (section 5) |
| `exclusive:` resources | label `excl:<resource>` is NOT enough for a lock; keep a `exclusive:` list in the notes block and let the dispatcher text read it | the lock stays a dispatcher rule, not a tool, as today (S) |
| `gates:` | Acceptance Criteria items | |
| tasks (checkboxes) | Acceptance Criteria checkboxes (U: CLI can tick them) | |
| `result:` (verified live or not) | Implementation Notes `result:` line, same rule: say verified live or not | agents write it in the report; the coordinator or the agent edits the file |
| `live:` | Implementation Notes `live:` line | |
| `merge:` hash | Implementation Notes `merge: <hash>` set at close | replaces the "Archive <id>" commit content |
| overlap paragraph ("Overlap (declared)") | Implementation Notes paragraph, same heading | only matters if the overlap check survives |
| delta specs / `skip_specs` | dropped; contracts move to `docs/contracts` (stage 3) | |
| proposal.md | Description (the Why/What in a few lines plus a link to the old card dir) | |
| `merge-lines.txt` | kept as a file next to the work, `backlog/merge-lines/<id>.txt`, or in the agent report (fallback already accepted) | `tools/merge_lines_check.py:49,62` must follow (U decision) |

## 5. Edits needed in each consumer (file:line from section 2, V unless noted)

Principle: each consumer first learns to accept both worlds (a card dir OR a Backlog.md task file OR neither), then the old path is removed.

### Public repo

- `cmd/changecheck`: what survives? Recommendation: nothing of the card/spec lint. Owner/overlap check against task `owns` (`checks.go:13-108`) is the one real candidate; cost to keep: a frontmatter parser for a format we do not control (U) and the body-block convention. Recommendation: retire it, since briefs carry ownership (S, judgement); if the user wants it, port only `checks.go:44-108` into a 150-line `cmd/taskcheck`. Merge subject rule (`checks.go:117-133,186-232`): stage 1 makes it accept `Merge agent/<id>:` with no `openspec/changes/<id>` (the `:219-229` test becomes "task file exists OR no card needed"), so `--card none` and the `allow_merges` commit go away; `-diff` (`checks.go:252`) skips when no card. `status.go:97-122` and `jsonout.go`, `needsyou.go` (parses `docs/needs-you.md`): keep as a tiny `status` tool until STATUS is regenerated from tasks, then move `needsyou` into the board repo code. Delete `new.go`, `specs.go`, `strict.go`, `card.go`, `openspec/.strict.yaml` at stage 2/3.
- `tools/hooks/pre-push:22-37` and `test-pre-push.sh`: stage 2 removes the `changecheck -cli` and `tools/openspec.sh` calls; the hook keeps leakcheck/other checks. Test updated in the same commit.
- `.github/workflows/ci.yml:40-42,51-52,74`: stage 2 removes the node setup, the changecheck step and the vet package name; the docs-only filter (`:74`) drops `openspec/` and gains `backlog/`.
- `tools/openspec.sh` deleted (stage 2); `tools/ci/check.sh:29-30` keeps the merge-lines lint with the new path.
- STATUS block (`docs/plan/STATUS.md:40`, `status.go:97-122`): stage 1: generator reads both `docs/cards.md`-style rows and Backlog.md task files if the CLI/frontmatter allow (U); simplest fallback: the block lists task titles and statuses read from `backlog/tasks/*.md` frontmatter with a 60-line script. `checkStatus` stale warning stays tied to the generator.
- `tools/merge_lines_check.py:49,62`, `tools/credit/credit.py:110-158` (`credit:` blocks: move to the task notes or drop, ask the user), `tools/session-brief.sh:11-14` (drop `.last-check`), `tools/contrib/chunker.py:366`, `.gitignore:27`, `docs/OWNERS` rows.
- Comment pointers (`tools/agentrun.sh:2`, `agentreap.sh:2`, `d2dslot.sh:2,86`, `d2dfarm.sh:2`, 5 Go comments): leave until stage 3, then script-rewrite.
- Docs: `docs/agents/issue-tracker.md` (rewritten for Backlog.md: create/read/comment/check commands; stage 4), `docs/agents/domain.md`, `docs/ci.md:15-17,39,47`, `openspec/README.md` stub, `docs/process/*` openspec notes (history banner), `docs/cards.md` stub.
- `AGENTS.md:3,7-12,67` (read order, Ownership section, Agent skills line): stage 4, needs the user's go.

### The ops repo (S: read by grep, not run; U for line drift)

- `tools/merge.sh:7-9,203-205,224,238-243,278-281,311,396-415,437,487,496`: gates and the archive step. New flow: (1) no `openspec` gate; (2) `--card none` and the `allow_merges` edit removed; (3) the "Archive <id>" commit becomes "Close <id> (<hash>)": set the task file to Done, write `merge: <hash>` (a CLI call or a file edit, U), run the STATUS generator; (4) `.strict.yaml` no longer required; (5) the `Change:` trailer stays optional and names the task by our id.
- `tools/mergeq.py:292,302,330,353-359`: `--card <id>` becomes "task file `backlog/tasks/*<id>*` exists (or none)"; the card-dir test for the `Change:` trailer and the snapshot of the card dir are dropped; the OWNERS filter keeps only rows that remain.
- `tools/gate-run.sh:184-239,325-364`, `tools/baseline-collect.sh:37-102`: record branches stop creating a card, `.openspec.yaml` and OWNERS rows; they create a task file only if the brief says so (default: report line only).
- `tools/merge_classify.py:12`: docs-only list `openspec/` replaced by `backlog/`.
- `tools/agentstatus/publish-cards.sh:12`, `agentstatus.py` routes `/changes`, `/cards`, `/api/cards`, `cardboard.py:34-48`: the scan source changes from `openspec/changes` to `backlog/tasks` in git (frontmatter parse). The `/cards` approve/defer page keeps its decision store; only the scan changes. Option for later: replace `/changes` by a link to `backlog browser` (U: port, auth and hosting, see 7). `boardviews.py`, `changes.html` follow.
- `tools/api-agent.py:296`: allow-list drops `tools/openspec.sh`, `tools/changecheck`; adds the Backlog.md CLI invocation only if agents use it (U).
- `docs/dispatcher.md:11,14,27`: step 2 (card board reminder) now reads the Backlog.md board or `backlog task list`; step 3 (card-based launch) briefs name the task id and goal label instead of "your card is openspec/changes/<id>/"; step 5b (merge queue `--card`) uses the new `--card`. Also `docs/process/dispatcher-brief*.md`, `docs/merging.md`, `docs/gate-run.md`.
- Worker conventions: the "Card conventions" bullet in `claude/agents/{d2-worker:21-22, d2-general:14-15, d2-live:18-19, d2-decomp:18-31, d2-reviewer:14, d2-merger:12-33}.md`. New text: "Your task is `backlog/tasks/<file>` (id in the brief): tick its acceptance criteria and fill the `result:` line (say verified live or not); do not edit other task files; shared-file lines go in `merge-lines`." `spec-sme.md` retired. `briefs/*.md` (2 mentions) edited by the coordinator; `state/briefs/*` untouched.
- Concurrency note: agents editing task files in their own worktrees merge by git; the rule "one agent edits only its own task file, the coordinator edits the status/closing" avoids conflicts (U whether the CLI is safe for that, see 7.2).

## 6. Revised staged plan

Order of repos: ops scripts accept both worlds BEFORE the public repo stops creating cards.

### Stage T - trial (no repo change; GO: already given by the user as a trial)
A separate agent runs `backlog init` in a scratch copy, hand-migrates 5 cards by section 4.4, runs `backlog browser`, and reports the answers to section 7. Files: none in this repo. Risk: none. Rollback: delete the scratch copy. No stage below starts before the trial report and the user's go.

### Stage 0 - interim rule (DONE)
No new OpenSpec change; owns, exclusive and result go into the report (section 8); `docs/cards.md` is the interim list (card `cardsapply`).

### Stage 1 - install in the repo, migrate (GO: yes)
- What: `backlog init` config in the repo (folder per 4.2, committed); migrate the 12 active + 5 parked cards (17 files), each with a link to the old card dir; the 4/5 "needs a decision" after the user's call; `docs/cards.md` becomes a stub when done; teach `checkMerges` the cardless/Backlog.md case; teach ops `merge.sh`, `mergeq.py`, `publish-cards.sh`/`cardboard.py` to accept task files without requiring a card dir or `allow_merges`; gate-run/baseline-collect stop creating cards; merge-lines path and `merge_lines_check.py`; STATUS generator reads task files.
- Files: `backlog/**`, `docs/OWNERS`, `docs/cards.md`, `cmd/changecheck/{checks,status}.go`, `tools/merge_lines_check.py`, `tools/ci/check.sh`, ops `tools/{merge.sh,mergeq.py,gate-run.sh,baseline-collect.sh,agentstatus/*}`.
- Risk: medium (merge scripts are the shared path; ops tests `tools/tests/test_merge_sh.sh`, `test_mergeq.py`, S). Task ids might not match our card ids (U).
- Rollback: revert the commits; the old card dirs and `docs/cards.md` are untouched until stage 3.

### Stage 2 - stop changecheck and the CI step (GO: yes)
- STATUS: done in the public repo on branch `agent/removestage2`, merged by mergeq (merge hash to be added after the merge). Done: ci.yml node setup, changecheck step and vet package removed; pre-push runs only the leak scan; test-pre-push.sh updated. Deliberately NOT done yet: `tools/openspec.sh` and `cmd/changecheck` stay because the ops merge scripts still call them (merge.sh runs both as gates inside the merged tree); delete them after the ops scripts are updated.
- What: changecheck warn-only for a week (per the old README), then remove the CI step and node setup, the pre-push validator call, the `openspec` merge gate, `tools/openspec.sh`; delete `cmd/changecheck` except the `status`/`json` bits (or move them).
- Files: `.github/workflows/ci.yml`, `tools/hooks/pre-push`, `tools/hooks/test-pre-push.sh`, `docs/ci.md`, `tools/session-brief.sh`, `.gitignore`, ops `merge.sh`, `api-agent.py:296`, `publish-cards.sh`.
- Risk: lost guard against overlapping owners (see 5: brief-level ownership). Rollback: revert; `.strict.yaml` stays until stage 3.

### Stage 3 - archive openspec, move contracts (GO: yes)
- STATUS: DONE 2026-11-04 (branch `agent/removestage3`, queue #164; ops half committed to the ops repo master). Done: 18 contracts `git mv` to `docs/contracts/<capability>.md` (pure rename), `docs/contracts/README.md`, `docs/contracts/proposed/` (the 9 open delta specs), `docs/archive/openspec-index.md` (one line per old card; links into `openspec/changes` rewritten to its anchors), `openspec/` removed down to a stub README, `docs/OWNERS` card rows dropped, `tools/backlog/status.py` (STATUS block and board `/changes` JSON from `backlog/tasks`), merge-lines now in `backlog/merge-lines/<branch>.txt`, `cmd/changecheck` kept with a guard (prints a note and exits 0 when `openspec/changes` is missing; deleted at stage 5). Ops: merge.sh/mergeq read the new merge-lines path, gate-run/baseline-collect create no card dirs, publish-cards.sh uses status.py, `MERGE_CHANGECHECK` retired, docs-only list gained `backlog/`. Not done (stage 3b/4): contract headers, checker, Numbers tables, AGENTS.md rewrite, the board `/cards` page (still reads `openspec/changes`, now empty), GLOSSARY/ADR/decisions links into `openspec/` (guard files, coordinator).
- What: tag `openspec-final` (push to the forge and the public remote after the user's go); keep the 639 archived entries only in that tag (and optionally an orphan branch); `git mv openspec/specs docs/contracts`; leave `openspec/README.md` as a stub; rewrite about 190 doc links (script-assisted); Go comments (5); `docs/OWNERS` rows; `docs/decisions.md:17` (coordinator).
- Risk: medium (many links, contracts cited by ADRs and `AGENTS.md:10`); run leakcheck and a link check. Rollback: `git revert`; the tag keeps the tree.

### Stage 4 - rewrite rules (GO: yes for AGENTS.md and glossary text; coordinator for agent definitions)
- What: `AGENTS.md:3,7-12,67`, `docs/agents/issue-tracker.md`, `docs/agents/domain.md`, `GLOSSARY.md` (proposed text only), `PLAN.md` banner; ops: the 8 agent definitions, retire `spec-sme.md`, `dispatcher.md` steps 2/3/5b, `merging.md`, `gate-run.md`.
- Risk: low (wording). Rollback: revert.

### Stage 5 - other repos (GO: yes per repo)
- What: `d2lod-bot-new` (U, not readable from this job; check existence, repeat section 1 there), the ops repo leftovers (`state/` stays as history). Verify with `grep -rIl -i "openspec\|changecheck"` returning only history. Risk low; rollback revert.

### Which stages need the user's go
All of stages 1, 2, 3, 5 (each its own go), and stage 4 for `AGENTS.md`, `GLOSSARY.md` and `docs/decisions.md` text (guard rule). Stage T needs no repo go; its report is the gate for stage 1. The coordinator may run the mechanical parts of stage 1 and 4 after the go.

## 7. Open questions about Backlog.md (for the trial agent; none answerable from the README)

1. Browser: does `backlog browser` need a port, auth or a login? Can it bind to localhost only, and run on the worker host and be reached from a phone (the user uses `/cards` from a phone today)? Read-only, or can the board edit files?
2. Concurrency: what happens when several agents (each in its own worktree) edit task files and the coordinator edits others? Are ids allocated from a counter file (merge conflict risk across branches)? Does the CLI lock anything?
3. Worktrees and branches: does the CLI/browser find `backlog/` from a worktree and from a branch that is not main? Does it read the branch's tasks or only main's? Does it commit on its own (a setting)?
4. Ids: can a task id or file name be set by hand to equal our card id? What does the file name look like?
5. Frontmatter: which fields exist and which are free? Are unknown keys (`owns`, `exclusive`) preserved on rewrite by the CLI? Are statuses configurable (we need parked, deferred)? Do labels, dependencies, priority exist as assumed?
6. Acceptance criteria: can the CLI tick a checkbox and edit notes non-interactively (headless agents)?
7. MCP: what does the MCP server expose; can a headless agent use it or the CLI under our allow-lists (no network, no sudo)? Does it need network at install time or at run time (the worker host has none for agents)?
8. Archive: how does Backlog.md archive a done task, and does it keep the file readable by a plain `grep`?
9. License and install: MIT (stated); install method (npm, binary), pinned version, offline install on the worker host.
10. Machine-readable output: does the CLI have a JSON/plain list for a STATUS generator and for `cardboard.py`?

## 8. Interim rule for agents (stage 0, in force until stage 2)

Briefs say: "No new OpenSpec change. Put owns, exclusive resources and the result line in your report; work is named by the branch `agent/<id>`." That is: (1) no new `openspec/changes/<id>/`, no `changecheck new`, no `.openspec.yaml`; (2) `owns:` and `exclusive:` come from the brief and go back in the report, `result:` (verified live or not) goes in the report.

How merges are named today (V, `cmd/changecheck/checks.go`): the subject regex is `Merge (?:branch )?'?agent/([a-z0-9-]+)` (`:117`); a `Change: <id>` trailer wins (`:125-133`); an id must be found AND `openspec/changes/<id>` must exist (`:219-229`), so `Merge agent/<id>:` alone warns `names change "<id>" which has no card`. Exempt: `merge master` / `merge branch 'main'`, GitHub's synthetic `Merge <sha> into <sha>`, and hashes in `allow_merges` of `openspec/.strict.yaml` (`:205-207`). Merge and diff findings are fatal in `-cli` mode (`openspec/.strict.yaml:4-5`), which CI (`ci.yml:42`) and pre-push (`tools/hooks/pre-push:22`) run. Cardless merges use `merge.sh --card none` (extra commit adding the hash to `allow_merges`; 12 such entries exist, V) until stage 1 fixes `checkMerges`. `-diff agent/<id>` (`checks.go:252`) fails with "no card for": mergers skip it for cardless branches (S). Agents must not create a card just to satisfy the merge check.

## 9. Proposed decision line and questions for the user

Proposed text for `docs/decisions.md` (not written by this job): `- 2026-11-04 (user, no ADR): OpenSpec retired for new work; card tracking moves to Backlog.md after a trial; contracts under openspec/specs are kept and move to docs/contracts at removal. Plan: docs/process/openspec-removal-plan.md.`

Questions for the user: (1) the 4 or 5 "needs a decision" cards (fixcastgap, fixmanamodel, fixskilllvl, skmeleewire, d2d-default-backend)? (2) drop the changecheck owners/overlap check (recommended) or port it? (3) keep the board `/cards` approve/defer page (S: it needs only a new scan source)? (4) keep contracts' requirement/scenario form unchanged after the move (recommended)? (5) `credit:` blocks of cards: move or drop?
- Ops merge gates removed (changecheck, openspec validate; rollback MERGE_CHANGECHECK=1), verified by agent/opscheck1.
- Stage 2 complete: public CI and pre-push no longer run changecheck/openspec (agent/removestage2, merge 0cb6c897), the ops merge pipeline gates are off by default (verified by agent/opscheck1, merge 0c9ee4a9; rollback MERGE_CHANGECHECK=1 in the ops merge doc). `tools/openspec.sh` is deleted by agent/removestage2b; `changecheck -cli` skips the validator when the wrapper is absent.
