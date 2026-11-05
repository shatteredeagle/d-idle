# Contracts after OpenSpec: design

> status: PROPOSAL, 2026-11-04 (job contractsdesign, docs only). Nothing here is applied. It refines stages 3 to 5 of `docs/process/openspec-removal-plan.md`; where it disagrees with that plan it says so. Tags: V = read or run in this worktree (file:line), S = secondary (read in another doc), U = unverified.

## 0. Decisions in one screen

1. **Directory: `docs/contracts/<capability>.md`**, flat, one file per capability (the stage 3 rename already prepared uses exactly this, V). Reasoning in 3.1.
2. **Form kept:** `### Requirement:` / `#### Scenario:` headings, SHALL/MUST, `Source:` on every number. Added later (not in the rename): a 6-key YAML header, and for the three gate contracts a `## Numbers` table that Go tests read.
3. **Change process:** a contract is edited in the same branch as the code that changes the behaviour, named by its Backlog.md task (`contracts:` key). No delta files, no apply step. A number with `Source: user decision` changes only with the user's go cited on the line. ADR, GLOSSARY and decisions.md guard rule unchanged.
4. **Checker:** `tools/contracts/check.py` (stdlib Python, about 150 lines) in `tools/ci/check.sh`: headings, header keys, paths named in `enforced_by` exist, knob citations match `docs/knobs.md`. Gate numbers are checked by Go tests in the enforcing packages, like `tools/knobdump/doc_test.go` already checks `docs/knobs.md`.
5. **`cmd/changecheck` is deleted whole** (stage 5), not reduced. STATUS and the board `/changes` page read `backlog/` through small scripts.
6. **Before any move:** two contract errors found here (5.2) and one judge drift go to the user. None of them blocks the rename.

## 1. What the contracts hold and who reads them

### 1.1 Inventory (V)

18 base files under `openspec/specs/*/spec.md`, 3710 lines, 243 requirements, 431 scenarios (counted with `grep -c` per file). The "21" in the coordinator's note is 18 plus three capabilities that exist only as open-card deltas (`primitives`, `d2d-mesh`, `stream-scene`; ledger notes, `docs/process/spec-ledger-20261104.md`). Nine open cards carry delta files (`openspec/changes/{d2d-container-soak,d2dmesh,m1-liveproof,m2-roam,m2-tripwp,m3-act1run,m3-andariel,m3-merc,stream-grid}/specs/`, V); all nine already have Backlog.md tasks (`card:` keys in `backlog/tasks/*.md`, V).

Three kinds of knowledge, and they need different treatment:

| kind | examples | where the truth really sits today |
|---|---|---|
| Gate numbers (judge a run) | soak-gates (25 req), death-rule (8), fidelity-gate (17) | Go defaults: `cmd/d2soak/deathrule.go:41-48`, `internal/d2dsoakstat/invariants.go:58-65`; ops `tools/gate-judge.py` (hard-coded 1000 XP/h, XP rule); `tools/ab/paired.py` |
| Policy numbers (how the bot plays) | merc-policy, early-game-policy, town-basics | tuning knobs, `internal/env/tuning_*.go`, mirrored to `docs/knobs.md` by `tools/knobdump` (drift test: `tools/knobdump/doc_test.go:33-52`) |
| Rules and interfaces in prose | backend-contract (42 req), lab-rules, public-split, upstream-pr, licence-policy, process-hygiene | prose only; some enforced by tools (leakcheck, gamecode guard, d2d lab lock) |

### 1.2 Consumers (V unless tagged)

- Rules: `AGENTS.md:10` ("put a delta spec in your change"), `AGENTS.md:68`, `docs/agents/domain.md` (read order item 2 and the file tree), `README.md:56,90,97`, `GLOSSARY.md` (entries cite "spec soak-gates" etc.), `openspec/README.md`.
- Docs: 61 files outside `openspec/` cite `openspec/specs` by path (`grep -rIl`); capability names alone appear far more often (soak-gates 148 files, upstream-pr 149, fidelity-gate 65, death-rule 53, merc-policy 51; `replay` 278 is mostly the English word).
- Code comments only: `internal/d2dsoakstat/invariants.go:16`, `internal/d2dsoakstat/act1check.go:14`, `internal/d2dreplay/lock_linux.go:13`, `tools/ab/paired.py:2`, `tools/ab/run.sh:4`, `internal/d2dsoakstat/invariants_test.go:421`.
- Code that parses the spec files: `cmd/changecheck/specs.go:166-178` (`specSummaries`, reads `openspec/specs` and silently returns nothing if the dir is gone), `cmd/changecheck/jsonout.go:77,139-150` (board `/changes` "gates" panel built from soak-gates and fidelity-gate requirements), `cmd/changecheck/status.go:39`, `tools/spec/ledger_scan.py:128`.
- Ops repo (S, read by grep): `tools/gate-judge.py:10,21,166,203,477,633` names the soak-gates spec in docstrings and record headers; `tools/gate-run.sh:205,223,348` writes "judged against the soak-gates spec" into record cards; `docs/dispatcher.md:22`; `claude/agents/spec-sme.md` (whole agent).
- Tests: no Go test is tied to a scenario by name. Two test files mention a capability in a comment (V, grep). The death-rule scenarios are covered in spirit by `cmd/d2soak/deathrule_test.go` and `internal/d2dsoakstat/invariants_test.go:234` (S: by reading test names, not by mapping each scenario).

## 2. Failure modes, shown

1. **Code changed a number, the contract did not (V).** `openspec/specs/merc-policy/spec.md:118` says `merc.kitReservePct` (150) and carries "Reconciled 2026-11-04"; the code default is 100 (`internal/env/tuning_merc.go:52`, changed the same day by commit 33c94c6b "merc: revive reserve 100 percent"), and `docs/knobs.md:269` says 100. A 25-line prototype of the knob check in 4.6 found it in under a second: 9 knob citations in the specs, 8 match, 1 mismatch.
2. **The judge disagrees with the contract (V, ops repo read this session).** `openspec/specs/soak-gates/spec.md:148` says XP per hour is NEVER a per-run pass or fail for Gate 1 (ADR 0015 item 8). Ops `tools/gate-judge.py:20-32` still returns FAIL below the baseline minus the tolerance; its row is added at `:344-346` and any FAIL row makes the verdict NOT PASSED (`:437`); the multi-seed path does the same (`:592`, `:610`).
3. **The contract disagrees with itself and with an ADR (V).** soak-gates "Soak progress floors" (`spec.md:37-38`) says the m24 profile MUST fail below 1000 XP/h; ADR 0015 item 8 (`docs/adr/0015-goals-acceptance-on-game-exe.md:21`) and goal G-C (`:32`) say XP/h is never a per-run criterion. The judge follows the old text for Gates 2 and 3 (`tools/gate-judge.py:349-351`, ops).
4. **One number, two copies in code (V).** The death rule's numbers are in `cmd/d2soak/deathrule.go:44-46` AND `internal/d2dsoakstat/invariants.go:63-64` (3 in 30 min, 5 in 60, 15 per 1000, 150 kills). Today they agree. Nothing checks it.
5. **Deltas waited to be applied by hand (V, history).** 70 delta files in 62 archived changes; application was manual. The ledger found 46 NOT APPLIED-IN FORCE rows (ledger summary); the two spec-apply jobs folded them in on 2026-11-04 (the Andariel interim rule the brief cites is now at `soak-gates/spec.md:11`, "source: 2026-11-04-andarielspec"). The mechanism, not the people, was the fault: a merged change could leave the contract stale with no warning.
6. **Silent breakage on the move (V by reading the code, not run).** After `openspec/specs` moves, `changecheck json` still exits 0 with an empty spec and gates list (`specs.go:168` ignores the read error), so the board `/changes` gates panel empties without an error, and `tools/spec/ledger_scan.py` reports zero requirements. The `MERGE_CHANGECHECK=1` rollback (ops `tools/merge.sh:27`) also stops meaning anything once `openspec/` is gone.

What the failures share: the contract text is a copy of numbers whose real owner is code (knobs, judge defaults), and nothing compares the copy with the owner. OpenSpec's validator never checked that; it checked headings. Dropping it loses little; adding a comparison gains most.

## 3. End state

### 3.1 Directory name: `docs/contracts`, not `docs/specs`

`docs/contracts` wins. The word "spec" is overloaded here: OpenSpec specs, delta specs, `docs/process/spec-*.md` (ledger, apply, approach evaluation), "spec-sme", and the general sense "design spec" in research docs; a reader seeing `docs/specs/` would expect the old OpenSpec rules (deltas, validate) to still apply. "Contract" is already the project's word for this content (`AGENTS.md:10` "Contracts and numeric gates", `openspec/README.md` "the stable contracts the bot must honour", removal plan stage 3) and says what the files are: promises the bot and the judges keep, not plans. The stage 3 worktree already stages `docs/contracts/<capability>.md` renames (V, `git status`), so this choice costs nothing. The one wart, `docs/contracts/backend-contract.md`, is accepted: renaming a capability would break 34 citing files for no gain.

Flat files (`<capability>.md`), not `<capability>/spec.md`: one file per contract is enough, links get shorter, and nothing else lives next to a spec today (V: every dir holds only `spec.md`). Pending (not yet in force) text gets its own subdir, 3.7.

### 3.2 File format

Stage 3 moves the files byte for byte (pure rename, so `git log --follow` works). A later commit (stage 3b) adds the header and fixes the H1 ("# Soak gates specification" becomes "# Soak gates contract"):

```markdown
---
id: soak-gates                    # equals the file name
goal: G-C                         # G-A..G-G of ADR 0015 and 0016, or "rule" for standing rules (lab-rules, public-split, ...)
status: in-force                  # in-force | proposed (only under docs/contracts/proposed/)
enforced_by:                      # repo paths (file or dir) that implement or judge it; "prose" if none
  - internal/d2dsoakstat/invariants.go
  - cmd/d2soak/deathrule.go
judged_by_ops: tools/gate-judge.py   # optional, ops-repo path, not checked by the public checker
adr: [0005, 0013, 0015]           # optional
---
# Soak gates contract

## Purpose
...
## Requirements
### Requirement: <name>
The bot SHALL ... Source: <doc, ADR or user decision YYYY-MM-DD>.
#### Scenario: <name>
- **WHEN** ...
- **THEN** ...
```

Kept from the OpenSpec form (it works and 243 requirements are written in it): one requirement per threshold, SHALL/MUST, at least one scenario, the `Source:` rule. Dropped: the 500-character limit, the `## Purpose` length rule, `.openspec.yaml`, every delta header. The "Reconciled 2026-11-04 (source: ...)" tails stay as history; new edits do not add them (git log is the record).

I recommend AGAINST the brief's `last_reconciled` date with an age limit: for a solo developer an age alarm fires on contracts nobody needs to touch (licence-policy, public-split) and trains everyone to bump dates. Code drift is caught by value checks (3.4), not by the calendar.

### 3.3 How a contract changes

- Who: any agent whose brief or Backlog.md task names the contract in `contracts:` (3.6); the coordinator; the user. Others write the proposed text in their report.
- How: edit `docs/contracts/<id>.md` in the same branch and PR as the code or judge change. No delta file, no apply step: what merges is in force. The commit message names the task (`Change: <card>` trailer as today).
- User go: a line whose `Source:` is a user decision, and every row of a `## Numbers` table (3.4), changes only when the brief says the user approved it; the new `Source:` cites that date. This is the old "gate numbers need the user" rule, now on the line instead of in a delta.
- Review: the merger shows contract diffs in the merge summary (ops `merge.sh`, one `git diff --stat -- docs/contracts/` line); no extra gate.
- Guard rule unchanged: agents never edit `docs/adr/*`, `GLOSSARY.md`, `docs/decisions.md` unless the brief names the file. Contracts are NOT under the guard rule (too many routine edits); the user-go rule above covers the numbers that matter.
- Not yet in force (a design for deferred work): `docs/contracts/proposed/<card>.md`, `status: proposed`, written as the full requirement text it would become; folded into the main file by the PR that implements it.

### 3.4 Numbers: contracts, knobs, judges and records

Rule: **every number has one owner, and the contract either cites it checkably or owns it.**

1. **Tuning knobs** (policy numbers): owner is the code; `docs/knobs.md` is its generated mirror (already drift-tested by `tools/knobdump/doc_test.go`). A contract cites a knob in the form the specs already use, `` `merc.reviveGoldMult` (1) `` (V: `merc-policy/spec.md:36,118`). The checker compares the value in parentheses with column 2 of the knobs table. Chain: code -> knobs.md (Go test) -> contract (checker). A bare number next to a knob name without the backtick-paren form is not checked; the README tells writers to use the form.
2. **Gate numbers** (soak-gates, death-rule, fidelity-gate): owner is the contract, because the user approves them. Each of these three files gets a `## Numbers` table:

   ```
   | key | value | unit | enforced in |
   |---|---|---|---|
   | death.streakDeaths | 3 | deaths | cmd/d2soak DefaultDeathLimits.StreakDeaths; internal/d2dsoakstat DefaultInvLimits.StreakDeaths |
   | gate1.noKillMin | 10 | virtual min | internal/d2dsoakstat DefaultInvLimits.NoKillMin |
   ```

   A `contract_test.go` in each enforcing package (`cmd/d2soak`, `internal/d2dsoakstat`) reads the table from `../../docs/contracts/<id>.md` and asserts its defaults, the knobdump pattern. That also kills failure 4 (the two death-rule copies are both tested against one table). About 30 rows in total (U: counted by eye, not listed). Requirements keep their prose numbers for readers; the checker warns if a requirement's number for a key differs from the table (a key mention plus a number in the same sentence; best effort, U).
3. **Ops judge** (`tools/gate-judge.py`): reads the Numbers table of the merged tree it judges (it already runs against a checkout; S) instead of its own constants, or at least prints the table version (commit) into every record. Record lines change from "criteria: the soak-gates spec" to "criteria: docs/contracts/soak-gates.md at <commit>". Then a record says which contract version judged it, which today it does not.
4. **Soak and gate records** stay evidence, never contract: they cite the contract and commit; a contract cites records only in `Source:`.

### 3.5 Scenarios to tests

Do not convert 431 scenarios. Rule per kind:

| kind | becomes | example |
|---|---|---|
| Gate and judge scenarios with log input and a verdict | a fixture case in the judge's Go test (table test over minute-log snippets) | soak-gates "Long silent stretch", "Poverty window"; death-rule "Fast second death" |
| Policy scenarios with a knob and a decision | a table test in the policy package, written against two builds (AGENTS.md build-flexibility rule) | merc-policy "Revive too expensive" |
| Live-only scenarios (d2d behaviour, portals, waypoints) | a drill check (`cmd/d2drill`) when one exists, else prose | backend-contract portal and waypoint rows |
| Process rules (lab-rules, public-split, upstream-pr, licence, hygiene) | prose; enforced by the tool that exists (leakcheck, gamecode guard, lab lock) | |

Link by name, not by id: the test case name or a comment carries the scenario name (`// contract death-rule: Fast second death`). A scenario with a test gets nothing new in the contract; the checker can later count "scenarios with a test" (grep of `contract <id>: <scenario>` in `*_test.go`) as an INFO line, never a failure. New work: a new or changed gate or policy scenario ships with its test in the same PR.

### 3.6 Backlog.md tasks reference contracts

- New custom frontmatter key `contracts: [soak-gates, death-rule]` on a task (Backlog.md keeps unknown keys on edit, S per `docs/backlog.md`; `task create` cannot write them, add by hand). `tools/backlog/check.py` warns when a listed id has no `docs/contracts/<id>.md`.
- Acceptance criteria name the contract and requirement: "soak-gates: Gate 1 belt and portal holds on seeds 11,16 (record docs/soaks/...)".
- The old `skip_specs` key in the migrated tasks is dead: drop it in the stage 3 sweep.

### 3.7 Open-card deltas

The nine open delta files must not disappear into the tag: m3-andariel and m3-act1run (G-B, wanted) hold the only written text for the Act 1 campaign rules. Stage 3 converts each to `docs/contracts/proposed/<card>.md` (one file per card, ADDED blocks written out as plain requirements, MODIFIED blocks as the full new text with a line naming the requirement it replaces), and the card's task gets `contracts:`. The m3-merc MODIFIED block is dropped (stale, ledger resolution 8, V) and its three ADDED blocks kept. Parked work (d2dmesh, stream-grid, d2d-container-soak) is converted the same way; it costs 3 small files and saves a tag dig later.

### 3.8 The checker: `tools/contracts/check.py`

Stdlib Python, read-only, pattern of `tools/shims/check.py`, called from `tools/ci/check.sh` (so CI and `merge.sh --hygiene-only` run it). Fails (exit 1) on:

1. a file in `docs/contracts/` (not `README.md`) with no header or a missing key (`id`, `goal`, `status`, `enforced_by`), `id` not equal to the file name, `status: proposed` outside `proposed/`;
2. an `enforced_by` path that does not exist in the repo;
3. no `### Requirement:` block, a requirement with no SHALL/MUST or no `#### Scenario:` (the parts of `cmd/changecheck/specs.go` that earned their keep, about 20 lines);
4. a knob citation `` `group.key` (N) `` whose key is not in `docs/knobs.md` or whose value differs from the knobs default;
5. a `## Numbers` table with a duplicate key.

Warns only: a requirement with no `Source:`; `--changed` mode lists contract files and Numbers rows the branch changed (for the merge summary); a task `contracts:` id with no file (if `tools/backlog/check.py` does not do it).

Not checked: whether prose is true, scenario-to-test coverage, ops judge constants (different repo). Size: S (one day with tests). The Go `contract_test.go` files of 3.4 are separate, size S each.

### 3.9 What `docs/contracts/README.md` says (proposed text, about 30 lines)

1. What a contract is: a promise the bot, a judge or the lab keeps; numbers that decide a gate or a policy. Not evidence (that is `docs/*.md` with V/S/U tags), not a decision record (that is `docs/adr/`), not a work item (that is `backlog/`).
2. The file list with one line each and its goal or "rule".
3. The format (3.2) with the header keys explained.
4. Numbers: the knob citation form and the `## Numbers` table; "never copy a number that has an owner, cite it".
5. How to change one (3.3), the user-go rule for decided numbers and the Numbers tables, and where proposed text goes.
6. How to check: `python3 -I tools/contracts/check.py`; the contract tests run with `go test ./cmd/d2soak ./internal/d2dsoakstat`.
7. History: up to tag `openspec-final` these lived in `openspec/specs/<id>/spec.md` and changed through delta specs in `openspec/changes/`; the ledger and resolutions docs (`docs/process/spec-ledger*.md`) record how they were reconciled.

## 4. De-provisioning map

Removal plan stages: T trial (done), 0 interim rule (done), 1 Backlog.md install and migrate (done, `docs/backlog.md`), 2 stop changecheck and CI step (done; ops gates off by default), 3 archive and move contracts, 4 rewrite rules, 5 other repos. Each OpenSpec or changecheck function:

| function today | where (V unless S) | action | stage | size | user go |
|---|---|---|---|---|---|
| card lint (proposal, tasks, header) | `cmd/changecheck/card.go` | DELETE (Backlog.md tasks + `tools/backlog/check.py` replace it) | 5 | S | covered by stage go |
| owns overlap and OWNERS sync | `cmd/changecheck/checks.go:13-108`, `docs/OWNERS` card rows | REPLACE by an overlap warning in `tools/backlog/check.py` over open tasks' `owns:` (about 40 lines); `docs/OWNERS` keeps the base `-` rows only, card rows dropped at stage 3 | 3 | S | yes (the removal plan asked "drop or port"; I recommend port as WARN: it is the one guard for parallel agents and costs little now that `owns` is in task frontmatter) |
| merge subject rule, `allow_merges`, `--card none` | `checks.go:117-133,186-232`, `openspec/.strict.yaml` | DELETE; ops merge already off by default | 5 | S | no |
| spec lint | `cmd/changecheck/specs.go` | REPLACE by `tools/contracts/check.py` items 3 | 3b | S | no |
| delta specs and the manual apply | `openspec/changes/*/specs/` | DELETE the mechanism; open deltas to `docs/contracts/proposed/` (3.7) | 3 | S | no |
| `openspec validate --all --strict`, node pin | `tools/openspec.sh` (deleted, stage 2b), CI node step (removed) | DONE | 2 | - | - |
| STATUS block | `cmd/changecheck/status.go:97-122`, `docs/plan/STATUS.md:40` | REPLACE by `tools/backlog/status.py` (reads `backlog/tasks/*.md` frontmatter; about 60 lines) writing the same markers; called by ops `merge.sh` `regen_status` instead of changecheck | 3 | S | no |
| board `/changes` JSON (cards, archive, gates panel, merges) | `cmd/changecheck/jsonout.go`, ops `tools/agentstatus/publish-cards.sh:12`, `cardboard.py:34-48`, `changes.html` | REPLACE: `backlog task list --json` plus a 40-line script for the custom keys; gates panel reads the `## Numbers` tables of the three gate contracts | 3 (before the move, or the panel empties) | M | no |
| needs-you parser | `cmd/changecheck/needsyou.go` | MOVE into the board code (ops) | 5 | S | no |
| `changecheck new` scaffolding | `new.go` | DELETE (`backlog task create`) | 5 | - | no |
| archive step, "Archive <id>" commit | ops `merge.sh` `archive_card` (only with `MERGE_CHANGECHECK=1`) | DELETE; close = task status Done + `merge_hash` in the task file | 3 | S | no |
| rollback switch `MERGE_CHANGECHECK=1` | ops `merge.sh:27`, `mergeq.py:337`, `merge_batch.sh:8` | DELETE at stage 3 (after the move it cannot work); say so in the ops merge doc | 3 | S | no |
| gate-run and baseline record cards | ops `gate-run.sh:184-239,325-364`, `baseline-collect.sh:37-102` | REPLACE: create a Backlog.md task file only if asked; record text says "criteria: docs/contracts/soak-gates.md at <commit>" | 3 | M | no |
| merge classification | ops `merge_classify.py:12` | `openspec/` -> `docs/contracts/` and `backlog/` in the docs-only list | 3 | S | no |
| api-agent allow-list | ops `api-agent.py:296` | drop `tools/openspec.sh`, `tools/changecheck` | 3 | S | no |
| golive dry run | ops `golive-dry.sh:32-36` | drop the changecheck step | 5 | S | no |
| ticks and session brief | `tools/session-brief.sh:11-14`, ops `tickstatus.sh` (S) | drop `.last-check` lines | 3 | S | no |
| merge-lines file path | `tools/merge_lines_check.py:49,62` | REPLACE path by `backlog/merge-lines/<card>.txt` (or keep the report fallback) | 3 | S | no |
| credit blocks | `tools/credit/credit.py:110-158` | ask the user (removal plan question 5) | 3 | S | yes |
| ledger scanner | `tools/spec/ledger_scan.py` | KEEP as history: banner "runs only on tag openspec-final" | 3 | - | no |
| spec-sme agent | ops `claude/agents/spec-sme.md` | REPLACE by a 10-line "contracts" section in d2-reviewer, or retire | 4 | S | coordinator |
| archived changes (656 dirs) | `openspec/changes/archive/` | NOT migrated. Tag `openspec-final` holds them (exists, S). Add `docs/archive/openspec-index.md`: one line per change, `date id merge-hash result-first-sentence` (generated once from the tag; about 660 lines). Doc links into `openspec/changes/archive/...` are rewritten to the index anchor by a script, dead ones listed. | 3 | M | no |
| `cmd/changecheck` itself | 1936 lines, 14 files | DELETE whole after STATUS and `/changes` have replacements; keep nothing | 5 | S | yes (part of stage 5 go) |

Why delete changecheck rather than keep a reduced Go tool (the removal plan kept `status`/`json`): its parsers read card dirs and `openspec/specs`; after stage 3 every input is gone, and porting the two outputs to read Backlog.md frontmatter is smaller in Python next to `tools/backlog/check.py` than in Go next to a dead card parser. AGENTS.md's stdlib-only Go rule does not forbid Python tooling (`tools/shims/check.py`, `tools/merge_lines_check.py` exist, V).

### 4.1 Order (refines removal plan stages 3 to 5)

- **3a, before the move (ops, M):** board `/changes` and STATUS read `backlog/`; gate-run record text; merge summary contract-diff line; `MERGE_CHANGECHECK` retired. Reason: otherwise the move empties the board panel silently (failure 6).
- **3, the move (public, M):** pure `git mv openspec/specs/<c>/spec.md docs/contracts/<c>.md` (staged already in the stage 3 worktree), link rewrite (script, 61 path-citing files plus code comments), `docs/contracts/proposed/` for the nine deltas, `docs/archive/openspec-index.md`, `openspec/README.md` stub, OWNERS rows. Run `tools/ci/doclinks.py` and leakcheck.
- **3b, the format (public, S):** headers, H1 rename, README, `tools/contracts/check.py`, wire into `tools/ci/check.sh`, Numbers tables + `contract_test.go` for the gate contracts.
- **4, rules (public and ops, S):** AGENTS.md text (section 7), `docs/agents/domain.md`, `docs/agents/issue-tracker.md`, agent definitions, dispatcher text.
- **5, deletion (public and ops, S):** `cmd/changecheck`, `openspec/` (stub README only), golive step, needsyou move. Verify with `grep -rIl -i "openspec\|changecheck"` returning history only.

Rollback for every step: `git revert` of that step's merge; the tag `openspec-final` holds the whole old tree.

## 5. Migration risks and the ledger rows

### 5.1 The 3 CONFLICT and 5 UNSURE rows (`docs/process/spec-ledger-resolutions-20261104.md`)

None blocks the move. The move is a rename; every resolution is a content decision that applies to the file wherever it lives, and the base text already reflects the proposed resolution in each case (resolutions doc, "Finding"). Row by row: 1, 2, 4-6 (gatedelta death and proxy rules) are confirmations that the base text is right, no edit; 3 (Gate 1 bands advisory) same; 7 (backend-contract "Leaving d2d") has one OPEN QUESTION line in the base (`backend-contract/spec.md:66`) that moves with the file; 8 (m3-merc stale MODIFIED block) is handled by 3.7 (dropped when the delta is converted) and needs the go the resolutions doc already asks for. The ADR 0011 note is a guard-rule edit for the coordinator, independent of the move.

### 5.2 New findings that do not block the move but need the user

1. **Gate 1 judge still fails on XP/h** (failure 2, ops `tools/gate-judge.py:20-32,344,592`) while the contract and ADR 0015 item 8 say never. A Gate 1 run with good stability and a slow seed would be recorded NOT PASSED. Fix: the XP row becomes advisory (status INFO) for Gate 1. Size S, ops repo. Needs nothing new from the user (ADR 0015 already decided); it should land before the next gate run.
2. **soak-gates "Soak progress floors" (m24 1000 XP/h MUST fail) contradicts ADR 0015 item 8 and G-C** (failure 3). The judge applies it to Gates 2 and 3. Needs the user: drop the XP floor for Gates 2/3 (my recommendation: keep "XP does not change for more than 10 minutes" and the no-gain window, which are stall checks, and drop the 1000 XP/h average), or confirm the floor as a stall guard and amend the wording.
3. **merc-policy `merc.kitReservePct` 150 vs code 100** (failure 1). Needs the user only to say which is intended; the commit message of 33c94c6b reads like a deliberate change, so the contract line should say 100 (U).

### 5.3 Risk register

| risk | likelihood | impact | mitigation |
|---|---|---|---|
| Board `/changes` gates panel and ledger scanner go empty silently at the move | high if 3a skipped | low-medium (wrong board, no wrong verdict) | do 3a first; the move PR checks the panel output once |
| Links into `openspec/` rot (61 path-citing docs, ADRs, decisions.md) | medium | low | scripted rewrite; ADR and decisions.md links are a coordinator edit (guard rule); `doclinks.py` before merge |
| An agent edits a gate number without the user's go (no delta review step any more) | low-medium | high (a gate verdict on a moved goalpost) | user-go rule on the line; `## Numbers` tables with Go tests make any change visible as a test edit; merge summary lists contract diffs |
| Contract and code drift again (the main failure today) | high without a check | medium | knob citation check, Numbers tests, judge reads the table |
| The open-card deltas are lost in the tag | medium if 3.7 skipped | medium (G-B design text) | `docs/contracts/proposed/` |
| `MERGE_CHANGECHECK=1` used after the move (rollback that cannot work) | low | medium (merge fails, confusion) | retire the switch in 3a |
| Backlog.md hides custom keys from `--json` (S) so `contracts:` is invisible to tools using JSON | certain | low | scripts parse frontmatter directly, as `tools/backlog/check.py` does |
| Ops `gate-judge.py` reading a public-repo file couples the repos | medium | low | judge runs against a checkout of the bot tree already (S); fall back to its own constants with a printed warning |

### 5.4 Cost and benefit

Lost: a validator of headings (keeps form, not truth); the delta mechanism that let a proposed contract change sit next to the card until merge; the formal ADDED/MODIFIED vocabulary reviewers could scan; the spec-sme agent's niche. Why acceptable: there is one developer and one coordinator; deltas were applied by hand and fell 46 requirements behind (failure 5), so their review value was not realised; the PR diff of a contract file is the same review surface a delta was, without the apply step. Kept and gained: the requirement/scenario form, the user-go rule, and, new, value checks that tie the contract to code (the drift the old tool never saw). Cost: about 2 S-days public (checker, Numbers tests, README, headers), about 1 M-day ops (board, gate-run, judge reads the table), one M-day for the move and link rewrite.

## 6. Recommendation and first week

Recommend: run 3a, 3, 3b in that order with `docs/contracts/`, delete changecheck at stage 5, add only the small checker and the Numbers tests. Disagreements with `openspec-removal-plan.md`: (a) keep the owner-overlap check as a WARN in `tools/backlog/check.py` (the plan recommended dropping it); (b) do not keep a reduced changecheck for status/json (the plan did); (c) stage 3 must preserve the open-card deltas as proposed contracts, not leave them in the tag; (d) the board and STATUS must switch before the move, not after.

First week:
1. Day 1: user answers 5.2 items 2 and 3 (one popup); ops agent makes the Gate 1 XP row advisory in `gate-judge.py` (5.2 item 1).
2. Day 1: ops agent, 3a: `/changes` and STATUS from `backlog/`; retire `MERGE_CHANGECHECK`; gate-run record text.
3. Day 2: stage 3 agent finishes the move in its worktree (renames already staged): links, `proposed/`, archive index, OWNERS, stub.
4. Day 2: merge stage 3; check the board `/changes` page and `doclinks.py` output.
5. Day 3: 3b agent: headers, README, `tools/contracts/check.py`, wire into `tools/ci/check.sh`; fix the kitReservePct line per the user's answer.
6. Day 3: Numbers tables for soak-gates, death-rule, fidelity-gate + `contract_test.go` in `cmd/d2soak` and `internal/d2dsoakstat`.
7. Day 4: ops: `gate-judge.py` reads the Numbers table (or prints the contract commit in every record).
8. Day 4: stage 4 wording: AGENTS.md (user go), domain.md, issue-tracker.md, agent definitions, dispatcher.
9. Day 5: stage 5: delete `cmd/changecheck`, `openspec/` down to the stub, golive step; grep verification.
10. Day 5: one reviewer pass over the contracts for remaining knob numbers not in the citation form (INFO list from the checker).

## 7. Proposed AGENTS.md wording (stage 4; NOT applied, user go needed)

Replace the `AGENTS.md:10` bullet with:

> - Contracts and numeric gates are in `docs/contracts/<id>.md` (README there). Change a contract in the same branch as the code it describes, and name it in your task's `contracts:` key. Numbers marked `Source: user decision` and the `## Numbers` tables change only when your brief says the user approved it. Cite knobs as `` `group.key` (value) ``; `python3 -I tools/contracts/check.py` must pass.

In the read order (`AGENTS.md:3`), replace "`openspec/README.md` (how jobs and contracts are written)" by "`docs/contracts/README.md` (how contracts are written)" and "YOUR CHANGE CARD `openspec/changes/<id>/`" by "YOUR TASK `backlog/tasks/<file>` (named in the brief)". In `AGENTS.md:68`, `openspec/specs/` becomes `docs/contracts/`.

## 8. Unknowns (U)

- Whether ops `gate-judge.py` always runs against a full checkout of the bot tree (needed for 3.4 item 3); read in its docstring only.
- The exact count of gate numbers for the Numbers tables (about 30, by eye).
- Whether `backlog task edit` keeps a list-valued custom key like `contracts:` intact (the trial checked `owns`, a list, S; assumed the same).
- Whether `d2lod-bot-new` (removal plan stage 5) exists and cites `openspec/specs`.
