# Spec / documentation / alignment approach: evaluation for d2lod-bot (2026-10-30)

Agent SPECRESEARCH-B, docs only. Tags: **V** = I fetched the primary page (URL given), **S** = secondary (snippet, blog, search summary), **U** = unverified / from memory or my judgement. Target is this repo as of master 6a6ff58. Web facts were fetched 2026-10-30 with a small-model summariser (WebFetch), so wording is paraphrased; treat exact numbers (stars, versions) as approximate.

## Recommendation in one paragraph

Do NOT adopt a spec framework wholesale. Adopt a thin "change card" layer that borrows the one idea all of them share (a small per-change folder with intent, owned paths, numeric acceptance, and a closing step that updates the standing doc) plus ADR-style decision records, keep AGENTS.md as the single rulebook, generate the parts that are currently hand-written (STATUS.md, the agent brief), and prune/archive the docs. If a tool must be tried, OpenSpec is the only candidate that earns a trial, and only for the `changes/` + archive mechanic, not for rewriting 113 docs as specs. Confidence: about 70 % that "thin cards + ADRs + generated status + prune" beats any full framework for this project; about 55 % on the exact folder layout (that is cheap to change). Section 4 lists what would flip it.

## 1. The problem we actually have, ranked

Facts about the repo (V = measured by me with git/wc in the worktree):

- The whole project is 2.9 days old: first commit 2026-10-28 12:41, HEAD 2026-10-30 09:55, 1,516 commits, all authored by one identity (Claude). 378 merge commits, 353 of them `Merge agent/...`: about 120 agent merges per day. 151 commits are docs-only. 540 commits landed after the last bulk doc sweep (10-29 12:05), 252 of them non-merge commits touching `internal/ cmd/ tools/` (V).
- 213 .md files, 22,737 lines under docs/ (V; the brief says 113 docs, the larger number includes sme/, plan/, archive/, classes/). docs/INDEX.md alone is 246 lines, AGENTS.md 47 lines / 8.4 KB, PLAN.md 135, plan-v2 230, STATUS 42, decisions 22, BACKLOG 88.
- Consequence for measurement: "stale by days" is meaningless; staleness here is "how many code commits happened after the doc's last edit". 17 docs were last touched before the 10-29 sweep and sit under 250-430 later code commits each (doc-audit.md).

Ranked problems (rank = cost to the user x frequency):

1. **No single authority that survives a restart, and the hand-written ones rot within hours.** Evidence: STATUS.md header "master 26d0aaf, 08:25" while master moved on (V, it carries a snapshot SHA by design, so it is wrong by construction after the next merge); INDEX.md says "833 commits" and "16 attempts" in g1-log (V: 1,516 commits; g1-log and STATUS say attempt 18); BACKLOG item 8 says "agent D2DFORK in progress (no commit yet)" while docs/d2d-fork.md (192 lines) is merged (V). The coordinating session that restarted today had to reconstruct state from these.
2. **Contradictions between docs of the same weight** (five found, below). Agents read whichever they open first; AGENTS.md sends them to PLAN.md first, and PLAN.md describes a Wine-only architecture with a "superseded in part" banner.
3. **Agent briefs hand-rewritten every time.** `.claude/agents/*.md` (72 lines total) hold the standing procedure, but each brief still re-states name, branch, owned dirs, acceptance, instance; `docs/liveruns-briefs.md` (63 lines) is a hand-pasted set from 10-29 (V). Brief quality varies and nothing checks that a merge matches its brief.
4. **Ownership collisions and shared-resource collisions that no doc encodes.** From tonight's failure list (given in the brief; the commit log corroborates the fixes: `soak24: per-instance bot path, port...`, `Merge agent/soakmulti: per-instance soak layout so two soaks run at once`): shared bot binary path, shared d2d socket, two agents editing internal/town. Disjoint ownership lives only in each brief's prose, not in a file a script can check.
5. **Work done outside the repo is lost** (patches on an unpushed private fork branch). Not a documentation-format problem; it needs a rule plus a check ("anything a doc relies on must be a commit hash on a remote reachable from the box").
6. **Volume.** 22.7 k lines of docs for a 3-day-old project; ~14 % of docs are history that agents still read because INDEX links them. Cost is context tokens per agent start, not disk.

### Five real contradictions / stale claims (all V, read in the worktree)

| # | claim A | claim B |
|---|---|---|
| C1 | docs/INDEX.md line 24: merc.md is a "pure policy package (not wired)"; its per-act table says "reader, hire policy ... not started" | docs/merc-wire.md and `git log` (`Merge agent/mercwire: d2dmerc reader, merc policy wired behind merc.enabled`, 10-30 09:29) say it is wired behind a knob |
| C2 | docs/boss.md section 6: "Andariel (clvl >= 22)" and "Andariel at 22+" | plan-v2 M2 status: knob `boss.andarielMinLevel` default 15, range 12..22 (22 reproduces the old hold); SME says 14 |
| C3 | plan-v2 section 1 (not marked historical; only s8 and part of s9 are): "The mercenary is dead code: hire rows are empty (cmd/d2bot/acts.go:104, every hire fails)" | INDEX: "hire window rows probed live (Kashya, Greiz)"; plan-v2 s10: merc path decided; mercwire merged |
| C4 | plan-v2 section 5 states the death policy as a streak rule | section 9 of the same file: "still NOT approved"; section 10: "NOT approved as written". Three states in one file; an agent reading only s5 implements the wrong rule |
| C5 | AGENTS.md: "Read PLAN.md and docs/d2-1.14d-memory.md first" | PLAN.md describes a Wine/Proton architecture ("The game itself runs under Wine/Proton") while docs/decisions.md 2026-10-30 says d2d is likely the primary backend and Wine container work is PAUSED. PLAN.md has a banner, but its architecture block and status prose still read as current |

Additional (lesser): BACKLOG "keep about 5 agents" vs the session's 5-8; INDEX "milestones last rescored 10-28"; decisions.md says Fresha "not started as a soak" while g1-log has "Fresha first fresh-character run started".

What is already good (do not break it): AGENTS.md is short, binding and specific; V/S/U tags exist everywhere; decisions.md is 22 lines, dated and user-owned; each agent has a card on a status board; `.claude/agents/` already carries the standing procedure; numeric gates are written down (plan-v2 s5).

## 2. What the candidates really are (web facts)

- **OpenSpec** (Fission-AI/OpenSpec, MIT). V https://github.com/Fission-AI/OpenSpec : specs in `openspec/specs/<domain>/spec.md` describe current behaviour; each proposed change is a folder `openspec/changes/<name>/` with `proposal.md`, `design.md`, `tasks.md` and delta specs using `## ADDED/MODIFIED/REMOVED Requirements`, `### Requirement:` and `#### Scenario:` WHEN/THEN blocks; `/opsx:propose`, `/opsx:apply`, `/opsx:archive`; archive merges deltas into main specs and moves the change to a dated `archive/` folder. "Built for brownfield not just greenfield". Standalone binary needs no Node; npm install needs Node >= 20.19. About 73 k stars (as shown on page 2026-10-30), 1,009 commits. V https://github.com/Fission-AI/OpenSpec/blob/main/docs/concepts.md : several change folders can coexist; two changes touching the same spec file do not conflict if they modify different requirements; the page does not say how same-requirement overlap is handled (U). Validation exists (`validate`, `/opsx:verify`) but I could not confirm its rules from the pages (U).
- **GitHub Spec Kit** (github/spec-kit, MIT). V https://github.com/github/spec-kit : `/speckit-constitution`, `specify`, `plan`, `tasks`, `implement`, plus a `converge` loop; Python 3.11+ and `uv` CLI (`specify init`); `.specify/` folder; brownfield only via a linked "Existing projects" guide with no detail on the page; a 1.0 video dated 2026-10-16. S (Fowler article, below): feels heavy for mid-sized features; branch-per-change in practice.
- **Kiro specs** (AWS, IDE product, not a library). V https://kiro.dev/docs/specs/ and /feature-specs/ : per spec `requirements.md` (or `bugfix.md`), `design.md`, `tasks.md`; requirements use EARS ("WHEN [event] THE SYSTEM SHALL [behavior]"); tasks run in dependency waves, concurrently within a wave; Quick Spec skips approval gates. File location not stated on the pages (U). It is tied to the Kiro IDE/CLI, so it is a pattern to copy, not a tool for a Claude Code workflow.
- **BMAD-METHOD**. V https://github.com/bmad-code-org/BMAD-METHOD : persona agents (product, architecture, UX, dev, test), briefs/specs/architecture artifacts, scale-adaptive tracks, dedicated brownfield guide; install `npx bmad-method install` needs Node 20.12+ and Python 3.10+; v6.12 stable with a v7 alpha with breaking changes; about 55 k stars. It is a full process with its own agent roles; we already have our own roles (`.claude/agents/`), so it would compete with them.
- **ADRs.** V Nygard, 2011-11-15 https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions : Title, Context, Decision ("We will ..."), Status (proposed/accepted/deprecated/superseded), Consequences; one or two pages; numbered sequentially, never reused; superseded records are kept. V MADR https://adr.github.io/madr/ : `NNNN-title.md` in `docs/decisions/`, sections Context and Problem Statement, Considered Options, Decision Outcome (+ optional Consequences, Confirmation), YAML front matter with status/date/decision-makers; 4.0.0 released 2024-09-17; MIT or CC0.
- **arc42 / C4**. U (not fetched, from memory): arc42 is a 12-section architecture template, C4 is a four-level diagramming model (context, container, component, code). Both describe a stable system structure for human readers; this project's layers are already one table in PLAN.md. They answer "what is the architecture" and none of our top-4 problems.
- **AGENTS.md convention.** V https://agents.md : plain Markdown at repo root, no required fields, nearest file wins in a tree, stewarded by the Agentic AI Foundation under the Linux Foundation, "over 60k open-source projects", supported by Codex, Cursor, Copilot agent, Gemini CLI and others (Claude Code reads CLAUDE.md natively; U whether it reads AGENTS.md without a pointer). We already use it.
- **Critical view of SDD tools.** S https://www.martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html (Böckeler, Oct 2025; only excerpts and secondary write-ups seen, not the full text): three levels (spec-first, spec-anchored, spec-as-source); Kiro is spec-first with specs deleted after a feature ships; spec-kit aspires to spec-anchored but is branch-per-change in practice; tools are too heavy for small changes (a small bug became four stories and 16 criteria); non-determinism persists. Directly relevant to us: most of our changes are small and numeric.
- **"Do nothing, prune."** No source needed; it is the baseline.

### Real multi-agent examples (what other teams do)

- Anthropic's C compiler build. V https://www.anthropic.com/engineering/building-c-compiler (Carlini, 2026-02-05): 16 parallel Claudes (S figures from press: about 2,000 sessions, ~$20 k), each in its own container cloning a bare upstream repo; a task is claimed by writing a text file into `current_tasks/`, git sync forces a collision to pick another task; merge, push, then delete the lock; no orchestrator; fresh sessions have no context so the prompt demands extensive README and progress files and a running doc of failed approaches; verifier quality is decisive; a known-good oracle (GCC) was what let agents parallelise on the kernel. Take-away: claim-by-file, progress files on disk, numeric/oracle verifier. They had no human-facing alignment layer because there was no human in the loop; we have one.
- Practitioner posts (S, one web search, anecdotal, 2026; no independent verification): worktree per agent, narrow path ownership with Git-derived scope checks, one integrator for merges, "describe each agent's job in one sentence", progress file on disk survives compaction; worktrees stop file clashes but not semantic clashes on shared contracts (https://www.mindstudio.ai/blog/parallel-ai-coding-agents-git-worktrees, https://engineering.myhoai.com/posts/orchestrating-parallel-agents, https://braindetox.kr/en/posts/multiple_ai_agents_one_repo_2026.html). Note these match what we already do (worktrees, a central integrator = the coordinating session, STATUS board).
- I found no source describing CODEOWNERS-style files for agents beyond the "narrow path ownership" advice (U: absence of evidence from one search). GitHub CODEOWNERS itself is V-known as a review-routing mechanism but enforces nothing locally; for us a 20-line `OWNERS` file consumed by a script is the equivalent.

## 3. Comparison against OUR criteria

Score: Y = fits, P = partial, N = no. Notes are why.

| criterion | OpenSpec | Spec Kit | Kiro-style specs | BMAD | ADR + living status | arc42 / C4 | AGENTS.md + index | prune only |
|---|---|---|---|---|---|---|---|---|
| Brownfield with 100+ docs | P: main specs must be written/extracted per domain; deltas work on top | P: guide exists, details unseen | P: per-feature, no brownfield story | P: has a brownfield guide, heavy | Y: add one file at a time | P: needs a rewrite | Y | Y |
| Parallel agents, disjoint ownership | P: one folder per change, deltas on distinct requirements coexist; no ownership field (we add one) | P: branch-per-change, no ownership | P: waves inside one spec, not across agents | P: persona roles, not file ownership | N: decisions, not tasks | N | Y for rules, no per-change owner | N |
| Numeric gates as requirements | Y: Requirement + Scenario WHEN/THEN fits "G1: 0 deaths, XP/h >= 6.6k" | Y: spec section | Y: EARS "SHALL" | P | P: Confirmation section | N | P | N |
| Low ceremony (about a page per change) | P: 3-4 files per change by default; the core profile is lighter | N: constitution + spec + plan + tasks per feature; Böckeler found it heavy (S) | N: 3 files, 4 stories for a small bug (S) | N | Y: one page | N | Y | Y |
| Survives session restarts | Y: state on disk, `changes/` listing = in-flight work | Y | Y | Y | Y | Y | Y | N |
| No heavy dependency | Y standalone binary, or just copy the folder convention; Node 20+ otherwise | N: Python 3.11 + uv | N: IDE product | N: Node + Python | Y | Y | Y | Y |
| Machine-checkable (script verifies a merge names a change) | P: CLI validates structure (U on rules); merge-to-change link is ours to add | P | N | N | Y if numbered and a commit trailer names it (ours to write) | N | P: lint links only | N |
| Fit with Claude Code | Y: slash commands generated for Claude Code (V: tool list on README, not itemised; U for exact command install) | Y: supported agent integration (U) | N | Y: plugin marketplace (V, v7 alpha) | Y: it is only Markdown | Y | Y native | Y |

Reading the table: no framework wins the "low ceremony" and "no dependency" rows together except ADRs and conventions we write ourselves; OpenSpec is the only one that also gives requirement-with-scenario structure and archive-on-merge.

## 4. Recommendation

### 4.1 What to adopt (thin, ours, OpenSpec-shaped)

A **change card** system, 1 page each, in `docs/changes/` (name and shape borrowed from OpenSpec's `changes/` + archive, V concept; no tool dependency):

```
docs/changes/
  README.md                      # the format, 30 lines
  active/<id>-<slug>.md          # one per in-flight work package
  done/<id>-<slug>.md            # moved on merge (this is the "archive" step)
docs/decisions/NNNN-title.md     # MADR-minimal ADRs; docs/decisions.md becomes the index
docs/OWNERS                      # path -> single owner (agent or "main"), 20 lines
```

Card = YAML front matter (`id`, `agent`, `branch`, `owns:` list of path globs, `exclusive:` shared resources claimed e.g. `d2d-socket`, `bot-binary`, `instance: i2`, `gates:` list) then four short sections: Why (3 lines), Requirements (numbered, each with a numeric or observable check, WHEN/THEN allowed), Out of scope, and Result (filled at merge: commit, verified live or not, doc updated). The `exclusive:` list is the lesson from tonight's shared socket / binary path: any card claiming an already-claimed resource or overlapping `owns:` fails the check.

### 4.2 Six to eight first cards (what to write first, from live open items)

1. G2 (8 h) soak and its numeric gate set (plan-v2 s5): becomes the template for "gate as requirement".
2. d2d fidelity gate within 20 % (docs/d2d-fidelity.md) as requirements with measured baseline.
3. d2d private fork: patches live in `<owner>/d2d-fork`, rule "must be pushed to the forge; repo never gets patches" as a checkable requirement (addresses the lost-patches failure).
4. Per-instance resources: socket, bot binary path, instance map (merges AGENTS.md prose, `tools/instances.conf`, soak24 layout).
5. Mercenary path (free Rogue, M3-M7) with the user's 10-30 decision as the requirement source.
6. Town layer ownership (`internal/town`): who owns it, and the d2d-town vs Wine-town split (the two-agents-one-dir collision).
7. Andariel gate knob (M2) reconciling C2.
8. Container/d2d primary decision (decisions.md 10-29/10-30) as ADR 0001 plus its open items.

### 4.3 Existing docs: migrate or archive

- Keep as is: AGENTS.md (trim; see below), decisions.md (turn into ADR index, keep the dated entries verbatim as ADR 0001-0004), plan-v2, sme/, classes/, area docs that are "live" in doc-audit.md.
- Fix now (30 min of an agent): C1-C5 above; mark plan-v2 s1 and s5 historical or edit them; PLAN.md architecture block gets a one-line "Wine-era; d2d: see decisions".
- Archive candidates (4 + 3 superseded in doc-audit.md): move to `docs/archive/` in one commit, leave a stub line in INDEX. Do not delete.
- INDEX.md: stop hand-editing the per-act table; generate the doc list (path, first heading, last commit date) with a script and keep only a 20-line hand-written "start here".
- STATUS.md: generate the factual block (master SHA, G-run summary lines pulled from g1-log, list of active cards, list of merged cards in the last 24 h) and keep only a human "headline" paragraph.

### 4.4 Agent brief template

A brief becomes: "Read AGENTS.md and your card `docs/changes/active/<id>.md`; do what its Requirements say." The orchestrator creates the card (a short script `tools/change new <slug> --agent X --owns ... --excl ...` that prefills branch, worktree path, instance from `tools/instances.conf`, and the standing-procedure link), so the human-visible brief is the card and the agent definition under `.claude/agents/` stays the procedure. Final report = fill the card's Result section, which also feeds the status board.

### 4.5 The script (`tools/changecheck`, shell or Go stdlib, runs in the merge step and CI)

1. Every merge commit `Merge agent/<name>` (or the commit trailer `Change: <id>`) names an existing card id; fail otherwise.
2. Changed paths of the branch are inside the card's `owns:` globs (git diff --name-only vs master merge-base); report violations.
3. No two active cards have overlapping `owns:` or the same `exclusive:` entry.
4. Cards in `active/` whose branch is merged are moved to `done/` (or flagged); `done/` cards have a Result with a commit hash.
5. Relative links in docs/INDEX and cards resolve; every card gate line contains a number or a named check.
6. Optional later: a "docs touched" warning when a merge changes `internal/X` but no doc under `docs/` mentioning X (cheap heuristic against staleness; warning only).

Estimated size: about 150 lines. This replaces most of what OpenSpec's CLI would validate, but is checkable against OUR rules (ownership, exclusive resources) which no off-the-shelf tool has.

### 4.6 What NOT to adopt

- Spec Kit, Kiro and BMAD wholesale: Python/Node/IDE dependency, constitution/spec/plan/tasks ceremony per change, persona agents that duplicate `.claude/agents/`, and (S) a reported heaviness on small changes. Our change stream is hundreds of small, numeric changes per day.
- Rewriting 113 docs as OpenSpec "specs" (domain spec files with Requirement/Scenario). The area docs are knowledge and live evidence (V/S/U tagged), not behavioural contracts; a rewrite is days of agent time and creates a second copy of every claim.
- arc42/C4: it documents a structure we already have in one table and addresses none of the ranked problems.
- Spec-as-source ideas (Tessl-style) (S): out of scope for a reverse-engineering project whose truth is a closed-source game.
- EARS wording as a rule: allow "WHEN/THEN" if an author likes it; the point is the numeric check, not the grammar.

### 4.7 Risks of adopting this

- **Double bookkeeping**: card, status board, STATUS.md, BACKLOG, commit message. Mitigation: the card is the only place agents write intent; the board card, STATUS block and BACKLOG "READY" list are generated from it or dropped. If that does not happen in week one, the layer is net negative: stop.
- **Overhead per change**: if a card takes more than about 5 minutes or a page it will be skipped. Mitigation: `change new` prefill; "quick" cards with only owns + one requirement for hotfixes (still named in the merge so the script passes).
- **The check becomes a blocker**: the user's rule is that only real blockers stop work. Make the script warn-only for two weeks, then fail only on rules 1 and 3.
- **A new stale surface**: `active/` cards for dead agents. Mitigation: rule 4 plus a "last commit on branch" age column.
- **Cards describe intent that live runs overturn**: the Result section must record "verified live / not", not the plan. Keep V/S/U tags.
- **Orchestrator discipline**: the main session must create the card before launching. If the coordinating session restarts mid-flight, `active/` is the recovery list (this is the main benefit).

## 5. What would make this recommendation wrong

- If the real cause of the failures was not missing structure but a lack of **verification** (the C compiler post says verifier quality dominates; V): e.g. the lost fork patches and shared-socket collisions would be caught by a pre-merge integration check (build, bring up two instances, run the soak smoke) better than by any card. Then spend the effort on the check harness, and the cards are decoration. Test: after one week, count failures a card/`owns:` check would have prevented versus ones it would not.
- If the user will not read cards or the coordinator will not write them (a human who steers by chat). Then the status board plus a generated STATUS is all that is used, and cards are overhead.
- If OpenSpec's `validate` and archive turn out to cover ownership-style checks, or its core profile is lighter than I assumed (I did not install it; U): a one-hour trial on one card might show adopting the real tool is cheaper than 150 lines of script. A 1-hour trial is cheap; I recommend it before building `changecheck`.
- If parallel-agent count drops to 1-2 (then worktree + AGENTS.md is enough).
- If the project's 3-day pace continues and docs are simply cheaper to regenerate than to maintain: "prune and generate" might beat everything, and cards add friction for no gain. The 2.9-day age of the repo makes my staleness figures a snapshot of a very fast start, not a steady state; I have no long-run data (U).
- My evidence on what "other teams" do is thin: one primary source (the C compiler post) and a handful of practitioner blogs surfaced by one search (S). I did not find a published case of a team using OpenSpec/Spec Kit with 5-8 parallel agents and measured outcomes (U: not found, may exist).
- Böckeler's critique was read via excerpts, not the article (S).

## 6. Minimal adoption plan (order)

1. (30 min) Fix C1-C5; archive the 7 candidates; add stub line in INDEX.
2. (1 h) Trial OpenSpec standalone binary on one card (G2 soak); decide copy-or-build.
3. (2 h agent) `docs/changes/README.md`, `docs/OWNERS`, `tools/change new`, `tools/changecheck` warn-only; write the 8 first cards.
4. (1 h agent) Generated STATUS block and INDEX doc list.
5. Week 2: make checks 1 and 3 fatal in the merge step; review whether the board, BACKLOG and cards still duplicate; drop what does.
6. ADRs: only for decisions the user states (decisions.md already is that); no retro-fitting.

## Sources

- https://github.com/Fission-AI/OpenSpec and .../docs/concepts.md (V, fetched 2026-10-30)
- https://github.com/github/spec-kit (V)
- https://kiro.dev/docs/specs/ and https://kiro.dev/docs/specs/feature-specs/ (V)
- https://github.com/bmad-code-org/BMAD-METHOD (V)
- https://adr.github.io/madr/ (V), https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions (V)
- https://agents.md (V)
- https://www.anthropic.com/engineering/building-c-compiler (V, 2026-02-05)
- https://www.martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html (S, via search excerpts)
- practitioner blogs via search (S): mindstudio.ai, myhoai.com, braindetox.kr
