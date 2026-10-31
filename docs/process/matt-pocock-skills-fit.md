# Matt Pocock skills plugin: fit for d2lod-bot (PLUGINFIT, 2026-10-30)

Docs only. Source: the installed plugin `mattpocock-skills` 1.3.1 (MIT, repo github.com/mattpocock/skills), read from `~/.claude/plugins/cache/claude-plugins-official/mattpocock-skills/1.3.1` (README, AGENTS, SCOPE, GLOSSARY, CHANGELOG, `.out-of-scope/`, docs/, every `SKILL.md` and the setup templates). Tags: **V** = I read the file in the plugin (no network, nothing run, nothing modified), **S** = the plugin's own docs describing a behaviour I did not trace in a SKILL.md, **U** = my judgement or untested. Nothing in the plugin was executed, so every "will do X" about runtime behaviour is V-of-the-instructions, not V-of-a-run.

## Recommendation in five lines

1. ADOPT `grill-me` (and `grill-with-docs` for decisions that must land in an ADR) as a USER-run, dedicated-session tool. Never in the coordinator session, never in a subagent (section 3).
2. `setup-matt-pocock-skills`: ADAPT. Run it once, by the user, in a throwaway worktree session, answering "Other" for the tracker (openspec change cards), "no triage" and "single-context". Expected diff: one `## Agent skills` block in AGENTS.md plus two small files under `docs/agents/` (section 2).
3. SKIP the ticket/spec/implement family (`to-spec`, `to-tickets`, `implement`, `implement-spec`, `triage`, `wayfinder`): they build a second work-tracking system on GitHub/GitLab/`.scratch`, we already have openspec cards + changecheck + dispatcher (section 4).
4. ADOPT as zero-cost extras: `handoff`, `retro`, `wait-what`, `diagnosing-bugs`, `writing-for-agents`. ADOPT-LATER: `code-review` (as a card-aware reviewer recipe), `improve-codebase-architecture`, `to-questionnaire`, `pr`.
5. Guard rail needed: the model-invocable `domain-modeling` can write GLOSSARY.md and ADRs on its own. Add one AGENTS.md line so agents never do that unasked (section 6).

## 0. Extra question: would the plugin's flow be a BETTER FIT than OpenSpec for us?

**Answer: no, not as a replacement. COMPLEMENT it at the front only (grill-me / grill-with-docs to decide, then our cards to execute). Confidence about 80%.** The plugin is clearly better at one thing we lack (structured interviewing of the one human); it is clearly worse at the thing our cards exist for (disjoint parallel ownership). Our OpenSpec layer is partly overbuilt for a solo vibe-coder, and trial-notes.md already says where.

**Who the plugin is for (V, README and SCOPE):** "real engineering, not vibe coding", one developer with one agent, "small, easy to adapt, composable", explicitly against process frameworks that "take away your control". Flow: `grill-with-docs` -> `to-spec` (issue) -> `to-tickets` (blocking edges, label `ready-for-agent`) -> `implement` or `implement-spec` -> `code-review`. Its session-survival tool is a phase-boundary decision tree (continue, /clear, /handoff, subagent, /compact; V `ask-matt/PHASE-BOUNDARIES.md`), i.e. one human steering one context. So the user is solo (good match), but the unit of work is a conversation, and the state lives in an issue tracker (GitHub/GitLab) or `.scratch/` files.

| criterion | plugin flow | our cards (openspec + changecheck) | edge |
|---|---|---|---|
| ceremony per change | grill (rounds, can be long) + spec with a long user-story list + N ticket files + implement; heavy for a small tuning change | about 3 minutes, 2 files (trial-notes: same as writing the brief it replaces) | ours for small jobs; plugin's grill is deeper for big decisions |
| interviewee | the one human, rounds with recommended answers, by number (V) | none; the coordinator writes cards from what the user said | plugin (this is the gap) |
| parallel agents, disjoint files | NONE. Parallelism = the blocking-edge "frontier" of tickets; implementer subagents each in a worktree, merger subagent resolves (V `implement-spec`). No ownership globs, no shared-resource names, no check that two tickets edit the same dir | `owns:` globs + `exclusive:` resources (`instance:iN`, `d2d-control-socket`, ...) + `changecheck` overlap check; this was built for the failures of night one (shared socket, bot binary path, two agents in internal/town; spec-approach-evaluation.md s1) | ours, decisively |
| numeric gates as requirements | spec has "Testing Decisions" prose; tickets have acceptance checkboxes; to-spec says no file paths/snippets | `### Requirement ... SHALL` with numbers and WHEN/THEN, `gates:` in the card, validated | ours |
| survives restarts/compaction | tracker issues or `.scratch` files survive; the conversation does not (handoff doc goes to /tmp, not the repo) | state is files in the repo: cards, STATUS (generated), ADRs, board | tie on durability; ours is more specific to "what is in flight" |
| machine-checkable | none (labels and prose; `code-review` is an LLM review) | `changecheck`, optional `openspec validate --strict`, merge-names-a-card check | ours |
| where state lives | GitHub/GitLab issues (the forge unsupported by design) or `.scratch/` | repo files | ours; plugin adds a second system or a `.scratch` clone of cards |
| coordinator stays clean | `implement-spec` is a coordinator pattern (subagents, context pointers, "communication sparse"), `chief-of-staff` is the same idea; but the grill and spec skills run in the main session | dispatcher brief + one-line reports | tie; plugin's concept is similar, ours is already built |
| deciding with the user | strong: grill, ADR, glossary | weak: ADRs exist but nothing structures the conversation | plugin |

**Is our layer overbuilt for solo vibe coding?** Partly, and trial-notes.md already admits it (V, "What I would cut"): `.openspec.yaml` in every card, OWNERS duplicating `owns:`, `worktree:` header, 11 archive-card folders for tiny merges. The core that earned its keep (owns/exclusive + overlap check + merge-names-card) is what a single human cannot do by eye when the coordinator launches 5 to 8 agents at once; a solo dev with one agent never needs it, which is why the plugin has no equivalent. The 14-day review rule in trial-notes (drop cards if they prevent fewer than about 2 collisions) still stands and is the right test. Not overbuilt: specs with numeric gates, which are the acceptance for an unattended soak. Possibly overbuilt: archive folders, spec deltas for every change, the optional CLI.

**Minimal merged workflow (one page):**
1. *Idea.* User says it to the coordinator in one or two lines.
2. *Decide only if it is a decision.* If the answer is not already in an ADR/spec, coordinator has a researcher write a pre-brief, then the USER runs `grill-me` (or `grill-with-docs` if an ADR is expected) in a dedicated session/worktree, 20 to 30 minutes. Output: ADR draft + `docs/decisions.md` line. Skip this step for routine jobs (most of them).
3. *Card.* Coordinator (or a subagent) runs `changecheck new` (owns, exclusive, gates with numbers; spec delta only if a contract changes). Replaces `to-spec` + `to-tickets`; the card is the ticket.
4. *Launch.* "implement change X": one agent per card, worktree, `owns` enforced by `changecheck -diff`. Use `tdd`/`diagnosing-bugs` inside the agent if it helps.
5. *Review.* Optional card-aware review (`mattpocock-skills:code-review` with the card path, or `d2-reviewer`) for risky merges only.
6. *Merge and archive.* Dispatcher merges, runs tests, `changecheck`, regenerates STATUS. Archive a folder only when the card carried a spec delta or a decision; otherwise STATUS "recent merges" is enough.
7. *Restart or compaction.* Nothing to do: STATUS + cards + ADRs are the state; use `handoff` only to carry loose conversational context.

**Stop doing (if adopting this):** hand-writing OWNERS lines (generate), `.openspec.yaml` in hand-written cards, archive folders for tiny merges, writing decisions in chat and in BACKLOG both (one ADR line), asking the user ad hoc questions across many messages in the coordinator (batch them into a grill). **Do not start:** GitHub/`.scratch` tickets, `to-spec`, `to-tickets`, `implement-spec`, `triage`, `wayfinder`.

**What would flip the verdict:** if the 14-day review shows cards prevented fewer than 2 collisions and mostly added bookkeeping, drop the cards and keep specs + ADRs + OWNERS + STATUS (the trial-notes fallback), still not the plugin's ticket flow; if agents were run one at a time, the plugin flow becomes reasonable.

## 1. Table of all skills

Invocation: **user** = `disable-model-invocation: true`, only reachable when the user types it (V frontmatter); **model** = the model may fire it by description (V). Plugin namespace is `mattpocock-skills:`; user-invoked ones are typed `/mattpocock-skills:<name>` (U whether the bare `/name` also resolves). Only `engineering/` and `productivity/` are registered by the plugin (`.claude-plugin/plugin.json` lists 27 skills, V); `misc/`, `in-progress/`, `deprecated/` exist in the cache folder but are NOT installed as skills (AGENTS.md of the plugin says they must not appear in the manifest, V). `deprecated/` holds only a README.

### Engineering (plugin registered)

| skill | what it does / invoked by / what it writes | verdict | why for us |
|---|---|---|---|
| `setup-matt-pocock-skills` | one-time per-repo config interview. user. writes `docs/agents/*.md` + `## Agent skills` block in CLAUDE.md or AGENTS.md | ADAPT (section 2) | other skills nag "run setup" without it; but needs the forge/openspec answers |
| `grill-with-docs` | grilling + domain-modeling: interview, then GLOSSARY.md and ADRs updated inline. user. writes `GLOSSARY.md`, `docs/adr/NNNN-*.md` in cwd | ADOPT | the right tool for pending user decisions that end in an ADR; run in a worktree (section 3) |
| `ask-matt` | router: "which skill fits". user. read-only | SKIP | routes to skills we skip |
| `to-spec` | synthesises the conversation into a "Problem / User Stories / Implementation Decisions" spec, no interview, published to the tracker with `ready-for-agent`. user. writes a tracker issue | SKIP | our spec form is `### Requirement ... SHALL` + WHEN/THEN scenarios with sources (openspec/README.md); to-spec wants "a LONG numbered list of user stories" and bans file paths; wrong shape for numeric bot contracts |
| `to-tickets` | splits a plan into tracer-bullet tickets with "blocked by" edges. user. writes issues or `.scratch/<feature>/issues/NN-*.md` | SKIP | we split by disjoint `owns:`/`exclusive:` globs on a change card, not by a blocking graph; a second ticket format = two systems |
| `implement` | build a spec/ticket, drive `/tdd`, then `/code-review`, commit on the current branch. user. writes code | SKIP | our `d2-worker`/`d2-live` agent definitions + card already carry the procedure (worktree, board, tags, trailer) |
| `implement-spec` | whole spec on one integration branch with implementer subagents per ready ticket, merger subagents, final code-review. user. writes branches/worktrees | SKIP | duplicates the dispatcher (`docs/process/dispatcher-brief.md`: merge to master, tests on the worker host, cards, push to forge); it also assumes issue-tracker closing semantics |
| `triage` | label state machine for GitHub/GitLab issues and external PRs, agent briefs, `.out-of-scope/` KB. user. writes labels, comments, `.out-of-scope/*.md` | SKIP | no issue tracker in use, no outside reporters; every comment must start with an AI disclaimer (V) |
| `wayfinder` | charts a huge effort as a "map" issue plus decision tickets worked one at a time (types research/prototype/grilling/task). user. writes tracker issues (or `.scratch/<effort>/map.md`) | SKIP (WATCH) | decision tracking is `docs/decisions.md` + ADRs + BACKLOG; adding a map doubles it. Its useful idea (fog-of-war list, HITL vs AFK tickets) can be borrowed by hand |
| `retro` | retrospective on a session (default the current one): reads session logs, proposes environment fixes (navigation, automated checks, standards, AGENTS.md size, tool economy), most severe first. user. writes nothing (presents candidates) | ADOPT | exactly the question "why did last night go wrong"; run in a fresh dedicated session pointed at a transcript, not in the coordinator |
| `code-review` | two parallel subagents, Standards (repo standards + Fowler smell baseline) and Spec (originating spec), side-by-side report. model-invocable (also typed). writes nothing | ADOPT-LATER | pass the change card path as the spec; needs a nested-agent-capable caller (U). Name collides with the harness `/code-review` |
| `pr` | PR body template: Summary visual, Before/After evidence, Merge Danger (door, blast radius). model. writes a PR body | ADOPT-LATER | we merge `--no-ff` locally; useful only for upstream PRs to d2d (`upstream-pr:*` is user-gated) |
| `diagnosing-bugs` | feedback-loop-first debug discipline in phases, with a redact step for secrets. model | ADOPT | matches our soak debugging; description triggers on "broken/failing/slow" which is common here but harmless (reads files, no repo writes) |
| `tdd` | red-green-refactor, one vertical slice at a time. model | ADOPT-LATER | principles fit Go tests at `mem.Fake` seams; the bundled examples are TypeScript/jest (V, `tdd/tests.md`, `mocking.md`); live-game behaviour cannot be TDD'd |
| `prototype` | throwaway prototype: single-file HTML state-machine demo, or several UI variants on one route. model. writes throwaway code near the target | WATCH | could test a state model (death rule, town loop) cheaply; UI branch assumes a web app. Keep off the repo: its rule is "commit to a throwaway branch" |
| `research` | background agent investigates against primary sources, writes a cited Markdown file. model | SKIP | `d2-researcher` is the same with V/S/U tags and D2-specific rules; risk the model fires this one instead (section 6) |
| `domain-modeling` | active glossary/ADR building, offers ADRs when hard-to-reverse + surprising + real trade-off. model. writes `GLOSSARY.md`, `docs/adr/` inline | ADOPT only via `grill-with-docs` | unasked, an agent could write ADRs; our ADRs are "user decisions only" (docs/adr/README.md) |
| `codebase-design` | deep-module vocabulary (module, interface, depth, seam, adapter, locality). model. read-only reference | ADOPT-LATER | helpful vocabulary for the architecture reviews; no cost until invoked |
| `improve-codebase-architecture` | subagent scans hot spots, writes an HTML report to the OS temp dir (nothing in the repo), then grills on the chosen candidate. user | ADOPT-LATER | do after the soaks settle; the grill half uses the user (section 3) |
| `wizard` | generates an interactive bash wizard for human-only steps (URLs, secrets, `.env`, `gh secret`). model | WATCH | only for a stranger-test container setup; we already have the d2fleet wizard |

### Productivity (plugin registered)

| skill | what it does / invoked by / writes | verdict | why for us |
|---|---|---|---|
| `grill-me` | thin wrapper: calls `grilling`. user. writes nothing | ADOPT | the user's explicit ask; for decisions without code |
| `grilling` | the interview loop: rounds, design tree, recommended answers. model-invocable (the only one in the family). writes nothing | ADOPT | in the model listing, so an agent could start interviewing a user who is not there (section 6) |
| `handoff` | compacts the conversation into a doc for another agent, saved to `$TMPDIR` or /tmp, NOT the workspace; includes "suggested skills"; redacts secrets. user. | ADOPT | the coordinator's restart path; writes outside the repo. Our durable state stays STATUS.md/cards |
| `wait-what` | re-pitches the last message in ASD-STE100 simple English using GLOSSARY.md terms. user. read-only | ADOPT | cheap fix for dense agent reports; works without a glossary, better with one |
| `to-questionnaire` | interviews the user about the SEND (recipient, what you need back), then writes `to-questionnaire-<slug>.md` in the CURRENT DIRECTORY. user | ADOPT-LATER | for the GPL/licence question to an outside person; tell it to write under `docs/process/` not repo root |
| `teach` | multi-session teaching workspace (MISSION.md, lessons/*.html, learning-records/) in cwd. user | SKIP | would scatter a workspace in the repo; if wanted, run in a folder outside it |
| `writing-for-agents` | style guide for skills, AGENTS.md/CLAUDE.md, pointer-reached docs. model | ADOPT | our AGENTS.md is 47 lines / ~8 KB and the plugin's rule is "navigation pointers, use sparingly"; read-only guidance |

### misc, in-progress, deprecated (NOT registered by the plugin)

| skill | what it does | verdict | why |
|---|---|---|---|
| `misc/git-guardrails-claude-code` | copies a PreToolUse hook blocking `git push`, `reset --hard`, `clean -f`, `branch -D`, `checkout .` | SKIP | blocks `git push` outright (V), but the dispatcher pushes to forge; our permission system already gates this. `misc/` is frozen/unmaintained (SCOPE.md, V) |
| `misc/setup-pre-commit` | Husky + lint-staged + Prettier + npm scripts | SKIP | Node tooling; a Go pre-commit (gofmt/vet) is a different job and vet is barred on the head box (AGENTS.md) |
| `misc/migrate-to-shoehorn` | rewrite TS test `as` casts to @total-typescript/shoehorn | SKIP | TypeScript only |
| `misc/scaffold-exercises` | course exercise folders | SKIP | irrelevant |
| `in-progress/chief-of-staff` | "long-running goal in one session by coordinating subagents; all work in subagents; protect your context window"; strategic view: improve the environment each time | WATCH | same philosophy as our coordinator rule; read for ideas, beta and unshipped (V: README of the bucket says beta, not shipped) |
| `in-progress/claude-handoff` | handoff plus `claude --bg --name ... -- "$(cat file)"` launches a fresh background agent | WATCH | the `--bg` launch is the interesting part for session restarts; unshipped |
| `in-progress/loop-me` | stateful grilling that writes workflow specs to `workflows/*.md` | SKIP | personal-life workflows |
| `in-progress/setup-ts-deep-modules` | dependency-cruiser for TS | SKIP | TypeScript |
| `in-progress/writing-{fragments,beats,shape}` | three-stage article writing | SKIP | irrelevant |
| `deprecated/` | README only | n/a | empty |

## 2. `setup-matt-pocock-skills`: exactly what it does

Read in full (V: `skills/engineering/setup-matt-pocock-skills/SKILL.md` + templates). It is prompt-driven, not a script: explore, present findings, ask one section at a time (each leading with a recommended answer), show a draft, let you edit, then write.

**Explore step** reads: `git remote -v`, `AGENTS.md` and `CLAUDE.md` at the repo root, `GLOSSARY.md`/`GLOSSARY-MAP.md`, `docs/adr/` and `src/*/docs/adr/`, `docs/agents/`, `.scratch/`, whether `triage` is installed, monorepo signals (pnpm-workspace, package.json workspaces, `packages/*`).

**Questions (at most three):**
- A. Issue tracker: GitHub (`gh`), GitLab (`glab`), local markdown (`.scratch/<feature>/`), or Other (one paragraph, recorded as prose). Default proposal follows the git remote; ours is `forge` at an ssh URL on Tower, which matches neither GitHub nor `gitlab.com` (U: it may still propose GitLab for a "self-hosted host"; ours is the forge, not GitLab, so refuse).
- B. Triage labels: asked only if `triage` is installed (it is, in the plugin). One question: "keep the default triage labels? (recommended yes)". Defaults `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`.
- C. Domain docs: single-context is written WITHOUT asking unless monorepo signals exist. We have none (Go module with `internal/` and `cmd/`).

**Writes:**
1. `docs/agents/issue-tracker.md` (from the github/gitlab/local template, or prose for Other).
2. `docs/agents/triage-labels.md` (only if B ran).
3. `docs/agents/domain.md` (consumer rules: read GLOSSARY.md and `docs/adr/` if they exist, "proceed silently" if not; use glossary vocabulary; flag ADR conflicts).
4. A `## Agent skills` block with `### Issue tracker`, `### Triage labels`, `### Domain docs` one-liners in ONE root file: "If CLAUDE.md exists edit it, else if AGENTS.md exists edit it, never create the other." We have AGENTS.md only (V: no CLAUDE.md in the repo root; `~/.claude/CLAUDE.md` is global and not consulted), so it edits AGENTS.md.
5. On GitHub/GitLab only: `gh label create` / `glab label create` for each missing label. With "Other" nothing is created remotely, no `gh` call is made.

It never creates GLOSSARY.md or `docs/adr/` (lazy, by `domain-modeling`) and never touches `openspec/`.

**Collisions with our setup**
- `docs/adr/`: same path. Our ADRs are MADR with status, theirs is "1-3 sentences, Status optional". Both fit; numbering rule (highest+1) matches ours. They do NOT update `docs/decisions.md` (our index); must be told (answer sheet).
- `docs/decisions.md` / `docs/adr/README.md`: unknown to it.
- AGENTS.md read-order (line 3: openspec/README.md, STATUS, change card, d2-1.14d-memory, area doc): the appended block sits after it; agents read the whole file anyway, but the block adds about 12 lines to a file every subagent loads. Keep it to 3 one-liners.
- Tracker: `to-spec/to-tickets/triage/wayfinder/implement-spec/code-review` all say "if no tracker was provided, tell the user to run setup"; the block silences that.
- `docs/agents/` is a new directory; no clash (V: does not exist).
- The forge: GitHub template needs `gh`; do not pick it. The GitHub account is for upstream PRs only.

**Answer sheet**

| question | answer |
|---|---|
| A. tracker | "Other." Paragraph: "No issue tracker. Work items are openspec change cards: `openspec/changes/<id>/` (proposal.md, tasks.md with a yaml header: status/owns/exclusive/gates), created with `go run ./cmd/changecheck new`, checked with `go run ./cmd/changecheck`, archived to `openspec/changes/archive/` at merge. Contracts are `openspec/specs/`. User decisions are ADRs in `docs/adr/` with a one-line index in `docs/decisions.md`. The forge (the NAS) is a git remote only; do not use `gh`, GitHub issues, labels or `.scratch/`. Skills that say 'publish to the issue tracker' must stop and ask the user instead." |
| B. triage labels | Decline: "we do not use `triage`; omit Section B and the triage-labels file". If it insists, accept defaults and delete `docs/agents/triage-labels.md` and the `### Triage labels` sub-block afterwards. |
| C. domain docs | single-context (default). |
| which file | AGENTS.md (the only one). |
| draft review | edit `domain.md` to add: "ADRs follow `docs/adr/README.md`; add a one-line entry to `docs/decisions.md`. Only the user decides ADRs: agents propose, never write one." |

**Expected diff:** AGENTS.md +about 12 lines; new `docs/agents/issue-tracker.md` (about 10 lines), new `docs/agents/domain.md` (about 40 lines, copy of the template with our edit). No labels, no remote calls, no GLOSSARY.md, no other file.
**Undo:** `git revert` the commit (or `git checkout -- AGENTS.md && rm -r docs/agents`). Nothing outside the worktree changes.

**Verdict:** ADAPT, run once. Cheaper alternative with the same result: hand-write the three files from this sheet (10 minutes, no interactive session). I recommend running the real skill once because the later skills read `docs/agents/*.md` by that exact path (V), but do it in a worktree branch and review the diff before merging. Confidence about 75% that it is net positive, mainly for the silencing effect; if we never use another engineering skill besides `grill-with-docs`, it matters little (`grill-with-docs` runs without setup, V: its SKILL.md only calls `grilling` and `domain-modeling`, neither mentions the tracker).

## 3. Grill workflow (the user is the interviewee)

**What the interview is (V `skills/productivity/grilling/SKILL.md`, `docs/productivity/grilling.md`):** NOT one question at a time and not everything at once. Each round asks the whole current frontier (all decisions whose prerequisites are settled), each as `Q<n> - title: body`, then a recommended answer on its own line, worded so "yes" accepts it. You answer by number. The agent recomputes the tree, asks the next round. It dispatches subagents itself for FACTS (filesystem, tools, docs) and does not block on them; decisions are only the user's. Ends "when the frontier is empty", then waits for the user to confirm shared understanding before acting. No question cap (`.out-of-scope/question-limits.md`, V): "some plans need three questions, some fifty"; stop it by telling it to wrap up. Output: plain `grill-me` writes nothing; `grill-with-docs` also edits `GLOSSARY.md` and creates ADRs in cwd as terms/decisions resolve (ADR only if hard to reverse + surprising + real trade-off).

**Can a subagent grill? No.** Our subagents return one report to the caller and cannot hold a multi-round dialogue with the user (U about the exact channel, but the Agent tool contract above is one prompt in, one result out). The skill itself uses subagents only for research facts. A subagent that "answers its own questions" is what wayfinder calls broken (V `wayfinder/SKILL.md`). So grilling is the one thing that cannot be delegated.

**Proposed workflow (keeps the coordinator clean):**
1. Coordinator (or user) names the topic and has a researcher/SME subagent write a pre-read: `docs/process/grill-<topic>-prebrief.md` (options, evidence, existing ADRs, V/S/U tagged). Facts are then not asked of the user.
2. User opens a DEDICATED session (new the head box pane or terminal; not the coordinator) in a throwaway worktree `~/code/d2lod-wt/grill-<topic>` on branch `agent/grill-<topic>`, and types `/mattpocock-skills:grill-with-docs` (or `grill-me` for a pure decision) with the pre-read path. It runs in rounds; the user answers by number, time-boxed (suggest 20 to 30 minutes, and say "wrap up" at the end).
3. The grill session itself writes the ADR draft (`docs/adr/NNNN-*.md`, status proposed) and the `docs/decisions.md` line; it is the only context holding the dialogue, so it must be the writer (handing a summary to a subagent loses the nuance). Our README rule: user decisions only, so the user reads the ADR before status becomes accepted.
4. The coordinator then launches a `d2-worker`/`spec-sme` subagent to turn the accepted ADR into change cards and spec deltas (`changecheck new`), and a dispatcher merges the grill branch. The coordinator context only ever sees "ADR 0009 accepted, branch X, cards Y".
5. Do NOT run grill in the coordinator session (a 30-round interview floods it and compaction could eat the decisions) and not overnight (nobody answers).

**Where to use it first (all are user-owned decisions):**
- d2d primary or not (ADR 0002 is "accepted, gated on proof"; the grill is for when the proof lands).
- GPL / licence / reviewer for the private d2d fork (ADR 0003; `docs/decisions.md` says "the user has no idea who reviews"): `to-questionnaire` for the outside person, grill for the user's own side.
- Merc strategy beyond ADR 0006; container direction (ADR 0001 open items: viewer port, stranger test).
- Spec writing: grilling the user about WHAT must hold (gates, thresholds) is useful; the writing of `### Requirement` blocks stays with `spec-sme`/changecheck. Do not use `to-spec`.

**Adjustment needed:** `domain-modeling` writes ADR format "1-3 sentences, status optional"; our format is MADR with status/date/decision-makers. Put in the grill prompt: "ADRs follow docs/adr/README.md (copy the newest), and add the docs/decisions.md line". (U whether the skill follows that over its own ADR-FORMAT.md; check the first diff.)

## 4. Overlap with change cards: one system only

| their step | ours | decision |
|---|---|---|
| to-spec (spec as issue) | `openspec/specs/<cap>/spec.md` + delta in a change | keep ours |
| to-tickets (tickets with blocking edges) | change card per job; disjointness via `owns:`/`exclusive:`, `changecheck` | keep ours. Idea to borrow: a `blocked-by:` line in the header (U, not needed yet) |
| implement / implement-spec (build, merge, review) | agent definitions + brief "implement change X" + dispatcher merge + changecheck | keep ours |
| triage / labels | card header `status: proposed|in-progress|done|archived` | keep ours |
| wayfinder map | `docs/decisions.md` + ADRs + BACKLOG | keep ours |
| code-review Spec axis | card's proposal/gates | feed the card path; the one place their skill plugs into ours |

Rule: grill output ends as an ADR (decision) and then change cards (work). Never as a GitHub issue or `.scratch/`. The `docs/agents/issue-tracker.md` paragraph above enforces it.

## 5. Which skills can run from subagents

- **Safe/usable in a subagent (no user dialogue, no repo surprises):** `diagnosing-bugs`, `tdd`, `codebase-design`, `writing-for-agents`, `pr`, `prototype` (offline logic branch). `code-review` spawns two sub-agents of its own; whether our subagent type can nest (the Agent tool is not available to every type, U) must be tested once; the plugin explicitly leaves recursion to the harness (`.out-of-scope/subagent-recursion.md`, V).
- **Needs the user live:** `grill-me`, `grilling`, `grill-with-docs`, `to-questionnaire`, `improve-codebase-architecture` (grill half), `teach`, `wizard` (human steps), `setup-matt-pocock-skills`, `triage`/`wayfinder` (HITL tickets), `retro` (presents to the user; works as a dedicated session, not a subagent).
- **Hard-blocked from agents by design:** every user-invoked skill (`disable-model-invocation: true`); a subagent cannot type a slash command. Whether subagents see the plugin's model-invocable skills in their listing is U.

## 6. Risks

1. **Listing cost: small.** Only 11 skills are model-invocable (`diagnosing-bugs, tdd, prototype, research, domain-modeling, codebase-design, code-review, pr, wizard, grilling, writing-for-agents`); their descriptions total 2,156 characters, about 540 tokens (V: measured with a script over the frontmatter). The other 16 user-invoked skills have `disable-model-invocation`, and my own skill listing shows exactly those 11 (V). So "35 skills in the listing" is not a real cost; bodies load only on invocation.
2. **Unexpected repo writes.** `domain-modeling` (model-invocable; description: "when discussing codebase terminology, writing or editing a GLOSSARY.md, or recording or editing an ADR") can create `GLOSSARY.md` and `docs/adr/*` in any session that fits. `research` is model-invocable and writes a Markdown file "where the repo keeps notes" while spawning a background agent (competes with `d2-researcher`, with the same job and weaker tagging rules). `grilling` fired by a coordinator would start an interview in a headless overnight run. `to-questionnaire` and `teach` write into cwd. Mitigations: (a) one AGENTS.md line: "Agents never create or edit ADRs, GLOSSARY.md or docs/agents/ unless their change card says so; do not use mattpocock-skills research/grilling/domain-modeling, use d2-researcher and the card." (b) U: a `permissions.deny` rule of the form `Skill(mattpocock-skills:research)` in project `.claude/settings.json` may hard-block them (I did not verify the rule syntax or whether it applies to subagents; test once). (c) `retro`/`handoff`/`improve-codebase-architecture` write only to the OS temp dir (V).
3. **Name collisions.** `mattpocock-skills:code-review` vs the harness `code-review`; `research` vs `anthropic-skills:deep-research`; `tdd`/`pr` generic names may be matched by loose triggers.
4. **Auto-update.** README: "A plugin updates itself" (V). 1.3.0 to 1.3.1 within days, 8 changelog sections in total; prompt text can change under a running project. Pin or snapshot the plugin directory if we start depending on `docs/agents/` formats (U how to pin).
5. **GitHub-centric defaults.** Templates are GitHub/GitLab/local; "mainstream issue trackers only" is out of scope for the plugin (`.out-of-scope/`, V), so no the forge template will ever appear. Triage comments must carry an AI disclaimer (V), irrelevant if skipped.
6. **in-progress / misc / deprecated.** Beta, unshipped (`in-progress`), frozen and unmaintained (`misc`: SCOPE.md says every issue is closed), empty (`deprecated`). None are installed; do not copy them in. `git-guardrails` would break the dispatcher's `git push forge`.
7. **Philosophy fit.** Pocock's pitch is "small, composable, you keep control" against GSD/BMAD/Spec-Kit (V README). That matches our "thin cards, no framework" decision (spec-approach-evaluation.md, 70% confidence). The plugin is for one human with one agent at a time; the many-parallel-agent, overnight parts of our workflow have no counterpart here except `implement-spec` and `chief-of-staff`, both of which overlap what we already built.
8. **TS bias.** `tdd` examples, `setup-pre-commit`, `shoehorn`, `setup-ts-deep-modules` are TypeScript (V). The principles transfer to Go; the snippets do not.

## 7. Adoption plan, in order

1. **Now, no change needed:** the plugin is installed. Use `/mattpocock-skills:grill-me` for the next pending decision in a dedicated session (section 3). Try `/mattpocock-skills:retro` on last night's session in a fresh session. Cost: zero repo changes.
2. **Today:** one coordinator-approved change (a card `pluginfit-adopt`, owns `AGENTS.md`, `docs/agents/**`): run the setup in a worktree with the answer sheet; add the AGENTS.md guard line (section 6.2); review the diff; merge via the dispatcher. Test the `Skill(...)` deny rule on `research` once.
3. **After the first grill-with-docs ADR:** check that the ADR matches `docs/adr/README.md` and that `docs/decisions.md` got its line; adjust `docs/agents/domain.md` wording if not.
4. **Later (after soaks stabilise):** `improve-codebase-architecture` once on `internal/`; `code-review` with a card path as the Spec source (test nested-agent support first); `to-questionnaire` for the licence/GPL reviewer question; `pr` for the first upstream d2d PR.
5. **Watch:** `chief-of-staff` and `claude-handoff` (in-progress) for ideas on session restarts; plugin CHANGELOG for breaking changes.
6. **Re-evaluate in a week:** if grill and retro each produced a decision or a change that stuck, keep; otherwise uninstall costs nothing (no files outside the optional step-2 diff).

## Open questions for the user

1. Is a dedicated grill session (its own pane, worktree, 20 to 30 minutes, ADR draft written by that session) acceptable, instead of grilling in the coordinator?
2. Run the real setup skill (interactive, 3 questions) or let an agent hand-write the same three files from the answer sheet?
3. Do you want a `GLOSSARY.md` at all (D2 and project jargon: soak, gate, den, instance names)? It makes `wait-what` and agent reports shorter, but is a new doc to keep in sync.
4. Accept the AGENTS.md rule that agents never write ADRs/GLOSSARY and must not use the plugin's `research`/`grilling`/`domain-modeling`?
5. Pin the plugin version to avoid prompt drift (cost: manual updates)?
