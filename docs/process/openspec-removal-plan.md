# OpenSpec removal plan (proposal for the user)

> status: PROPOSAL, written 2026-11-04 after the user retired OpenSpec for new work. Nothing here is applied. Stages marked GO need the user's go. Tags: V verified (file:line read in this session), S secondary (read in a doc, not re-run), U unverified.

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

## 3. Card-tracking replacement options

What a card carries today (V: `docs/agents/issue-tracker.md`): id, agent, branch, status, `owns:`, `exclusive:`, `gates:`, tasks, `result:`, delta spec. What the process really uses: (1) the brief, (2) the lock on shared resources (`exclusive:`), (3) the result line, (4) the merge naming. Path ownership is already enforced by the dispatcher's briefs, and `owns:` lint is mostly bookkeeping (S, judgement).

### a. One `docs/cards.md` table, capped at 12 rows, each row mapped to a goal G-A..G-F

Columns: id, goal (G-A..G-F), agent, branch, status, exclusive resources, result (one line, live or not).
- Pros: matches the six goals in ADR 0015; tiny; readable by a person in one screen; a cap forces closing or dropping work; no tooling.
- Cons: one shared file means merge conflicts (agents must not edit it; the coordinator does, as with BACKLOG); no per-job task checklist; 36 active cards do not fit in 12, so about 24 must be closed, folded or parked first (that is a feature but it is a decision); no history of closed rows beyond git.
- Rewrite: AGENTS.md rules, briefs (result goes in the report), merge.sh (no card, no archive), mergeq (drop card tests), dispatcher, board `/changes` replaced or hidden, STATUS block generator reads the table or goes away.
- Cost: low.

### b. GitHub issues in the private forge

- Pros: real status, comments, assignee, links to PRs; the board and user can see it on a phone.
- Cons: AGENTS.md (V `AGENTS.md:15`) says nothing is posted to GitHub without the user's go; the forge remote is "a push target only" (V `docs/agents/issue-tracker.md:3`), so this reverses a standing rule; agents are headless and cannot call `gh` (runner notes, S); issue ids do not exist offline; the exclusive-resource lock has no home; leak risk (private text on a forge that may be mirrored, U).
- Rewrite: everything in (a) plus a bot token, a forge-side workflow, and a new decision.
- Cost: high. Not recommended.

### c. Keep card directories as plain markdown; drop spec validation and changecheck strictness

Keep `openspec/changes/<id>/` (or move to `docs/cards/<id>.md`, one file), delete the validator, delete spec lint, overlap, status, and strict dates. Keep only a small merge-subject lint.
- Pros: lowest risk; history and all 639 archived paths stay valid; merge.sh, mergeq, gate-run, cardboard keep working with minor edits; the `/changes` and `/cards` boards keep a source.
- Cons: the user said OpenSpec is retired for new work, so keeping the layout invites new cards by habit; archive step and OWNERS bookkeeping remain; still a heavy ritual for 10-line jobs.
- Rewrite: delete `tools/openspec.sh`, CI node step, pre-push validator call, spec lint (`specs.go`), `.openspec.yaml` writers (gate-run, baseline-collect), AGENTS.md wording.
- Cost: low-medium. Good as a transitional state only.

### d. The existing board cards page as the store

Today `/cards` only stores approve, defer and reject decisions and scans cards from git (V `cardboard.py:34-48`; S board-cards doc). It has no cards of its own.
- Pros: the user already uses it from a phone; decisions are already there.
- Cons: it lives in a container without the repo (S); it is not in the repo, so agents and offline checkouts cannot read it; no history in git; a new write API and auth would be needed; couples work tracking to one host.
- Rewrite: new card model in the board, new dispatcher reads, merge.sh calls.
- Cost: medium-high.

### Recommendation

Go to (a) as the end state, passing through (c) as the bridge. Concretely: new work gets no card now (stage 0); the coordinator keeps one `docs/cards.md` of at most 12 rows (G-A..G-F) as the only tracker; the existing 36 active cards are closed, folded into rows, or archived with `git mv`; `openspec/specs/*` move to `docs/contracts/` unchanged (they are contracts, not process). The board `/changes` page can read the same table later (S: a small parser, optional). Reason: ADR 0015 already reduced the project to six goals, so a card per agent job is over-structure; a briefed job plus a report line is enough, and path ownership comes from the brief.

## 4. Interim rule for agents (until stage 2)

Briefs should say: "No new OpenSpec change. Put owns, exclusive resources and the result line in your report (and in the commit trailer `Result:` if useful); work is named by the branch `agent/<id>`."

Two lines for the brief: (1) no new `openspec/changes/<id>/`, no `changecheck new`, no `.openspec.yaml`; (2) `owns:` and `exclusive:` come from the brief and go back in the report, `result:` (verified live or not) goes in the report.

How merges are named today (V, `cmd/changecheck/checks.go`):
- The merge subject regex is `Merge (?:branch )?'?agent/([a-z0-9-]+)` (`:117`); a `Change: <id>` trailer wins over the subject (`:125-133`).
- Rule (`:219-229`): an id must be found AND `openspec/changes/<id>` (active or archived card) must exist. So `Merge agent/<id>:` alone is NOT enough without a card: it yields the warning `names change "<id>" which has no card`.
- Exempt: merges whose subject starts `merge master` / `merge branch 'main'`, GitHub's synthetic `Merge <sha> into <sha>` commit, and hashes in `allow_merges` of `openspec/.strict.yaml` (`:205-207`).
- Merge and diff findings are fatal in `-cli` mode since 2026-10-30 (V `openspec/.strict.yaml:4-5`), and `-cli` is what CI (V `ci.yml:42`) and pre-push (V `tools/hooks/pre-push:22`) run.
- Today's cardless path is `merge.sh --card none`, which makes a second commit adding the merge hash to `allow_merges` (S: ops `tools/merge.sh:7,396-415`; V that 12 such entries exist in `.strict.yaml`). Its subject is still `Merge agent/<id>: ...`.
- Interim recommendation: cardless work is merged with `merge.sh --card none` (works today, no code change, costs one extra commit). Alternatively the first removal step (stage 1) makes `checkMerges` accept a cardless `Merge agent/<id>:` and then `--card none` no longer needs the allow_merges commit. Agents must not create a card just to satisfy the merge check. `changecheck -cli` 0 warnings stays the bar for the commit an agent makes on its own branch (the merge check runs on merges only).
- `-diff agent/<id>` (`checks.go:252`) fails with "no card for" when there is no card; mergers must skip it for cardless branches (S).
- Existing active cards keep working until closed; their agents continue as briefed.

## 5. Staged removal plan

Order of repos: public repo first for rules and checks that guard its main; ops repo scripts must accept both worlds BEFORE the public repo stops creating cards (ops merge.sh reads the public tree, so it must tolerate a missing card and a missing `openspec/.strict.yaml`).

### Stage 0 - stop creating cards (GO needed: the user's go on this plan)
- What: dispatcher and briefs state the interim rule (section 4); coordinator stops running `changecheck new`.
- Files: dispatcher text, agent definitions in the ops repo (one sentence each); `docs/agents/issue-tracker.md` banner.
- Risk: agents whose brief still says "your card" invent cards; cardless merges need the allow_merges commit.
- Rollback: revert the text. No data touched.

### Stage 1 - replacement in place (GO: yes, choice of option a vs c)
- What: create `docs/cards.md` (12 rows max, G-A..G-F); decide the fate of the 36 active cards (close, fold, archive); teach `checkMerges` that a cardless `Merge agent/<id>:` is fine; teach ops `merge.sh` and `mergeq.py` to accept no card without writing `allow_merges`; gate-run and baseline-collect stop creating cards (they would write a row or nothing); move `merge-lines.txt` to a new place (`docs/merge-lines/<id>.txt`) and update `merge_lines_check.py`.
- Files: `cmd/changecheck/checks.go`, `tools/merge_lines_check.py`, `tools/ci/check.sh`, ops `tools/{merge.sh,mergeq.py,gate-run.sh,baseline-collect.sh}`, `docs/cards.md`.
- Risk: medium (merge scripts are the shared path; ops tests `tools/tests/test_merge_sh.sh`, `test_mergeq.py` cover it, S). 
- Rollback: revert commits; old cards untouched.

### Stage 2 - stop changecheck and the CI step (GO: yes)
- What: remove the CI step and node setup (`ci.yml:36-42,51-52`), pre-push validator call, the merge gate named `openspec` in ops merge.sh, `tools/openspec.sh`; keep a minimal lint if wanted (merge-lines lint, leakcheck unchanged). Turn changecheck into warn-only first for a week (the README says warn-only was the original mode), then delete `cmd/changecheck` or reduce to `status` plus `json`. Move `needsyou.go` parsing out first (the board needs it).
- Files: `.github/workflows/ci.yml`, `tools/hooks/pre-push`, `tools/hooks/test-pre-push.sh`, `docs/ci.md`, `tools/session-brief.sh`, `.gitignore`, ops `merge.sh`, `api-agent.py:296`, `publish-cards.sh`.
- Risk: lost guard against overlapping owners; mitigated by brief-level ownership and the 12-row cap.
- Rollback: revert the commit; CI step returns; `.strict.yaml` kept until stage 3.

### Stage 3 - archive then delete `openspec/` (GO: yes, this is the irreversible-looking step)
- What: tag the last tree (`git tag openspec-final`) and push the tag to the forge and the public remote after the user's go; copy `openspec/changes/archive` to an orphan branch or a tag-only snapshot; `git mv openspec/specs docs/contracts`; leave `openspec/README.md` as a one-line stub pointing to the tag (AGENTS.md rule: never delete docs, stub them); rewrite links to `openspec/specs/...` and `openspec/changes/...` (about 190 doc hits, script-assisted, rewrite to `docs/contracts/<name>/spec.md` or to the tag path). History is kept in git either way.
- Files: `openspec/**`, doc links, Go comments (5), `docs/OWNERS` (drop `openspec/specs/**` row, rename or remove the file, since nothing checks it), `docs/decisions.md:17`.
- Risk: medium (many link edits; contracts cited by ADRs); leakcheck and a link check run after.
- Rollback: `git revert` the move commits; the tag keeps the tree.

### Stage 4 - rewrite rules (GO: yes for AGENTS.md; briefs by the coordinator)
- What: rewrite `AGENTS.md:3,7-12,67` (read order, "Ownership" section, "Agent skills" line), `docs/agents/issue-tracker.md` (point to `docs/cards.md`), `docs/agents/domain.md`, `GLOSSARY.md` (card terms; proposed text only, per the guard rule), `PLAN.md` banner; ops: the 8 agent definitions (remove the "Card conventions" bullet), retire `spec-sme.md`, update `dispatcher.md`, `merging.md`, `gate-run.md`, and the board pages. Agent definitions are in the ops repo, so this stage is a separate commit there.
- Risk: low; wording only.
- Rollback: revert.

### Stage 5 - other repos (GO: yes per repo)
- What: `d2lod-bot-new` (U, check existence), the ops repo leftovers (`state/` stays as history, do not edit), any contributor docs that cite cards. Verify with `grep -rIl -i "openspec\|changecheck"` returning only history dirs.
- Risk: low. Rollback: revert.

### Which stages need the user's go
Stage 0 (the user's go on this plan, then it is a text change); stage 1 (pick the option); stage 2 (turning off the CI guard); stage 3 (moving and tagging the contracts); stage 4 for AGENTS.md and glossary text (guard rule: `GLOSSARY.md` and `docs/decisions.md` are written only by the coordinator or the user); stage 5 per repo. The coordinator may run the mechanical parts of stage 1 and 4 after the go.

### Proposed text for `docs/decisions.md` (not written by this job)
`- 2026-11-04 (user, no ADR): OpenSpec retired for new work; no new change cards; existing cards close out; contracts under openspec/specs are kept and move under docs/ at removal. Plan: docs/process/openspec-removal-plan.md.`

### Open questions for the user
1. Option a (recommended) or c as the end state?
2. Do the 13 deferred and 4 proposed cards get closed, folded into the 12 rows, or archived as is?
3. Keep the board `/cards` approve/defer page? It reads cards and has no other source (S).
4. Do the `openspec/specs/*` contracts keep their requirement/scenario form after the move (recommended: yes, text unchanged)?
