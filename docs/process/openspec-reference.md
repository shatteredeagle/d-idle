# OpenSpec reference (SME notes for setting it up and writing its documents)

Researched 2026-10-30 by SPECRESEARCH-A. Tags: **V** = verified (read in a primary source I fetched, or observed by running the CLI), **S** = secondary (blog/snippet), **U** = unsure. Facts about this project's own fit are in `openspec-fit-notes.md`.

Primary sources used (all fetched 2026-10-30):
- P1 repo clone `https://github.com/Fission-AI/OpenSpec` at commit 6d04a1b (2026-10-24), version 1.15.0: README, `docs/*.md`, `schemas/spec-driven/`, `src/core/validation/*`, `src/core/specs-apply.ts`, `openspec/specs/*` (the project is specced with itself).
- P2 `npx -y @fission-ai/openspec@latest` 1.15.0 run in a throwaway dir (`/tmp/claude-1000/os/try`, telemetry off, nothing installed globally). Everything marked "run" below was observed there.
- P3 GitHub API (`gh api repos/Fission-AI/OpenSpec`, releases, contributors, issues).

## 0. Identity, version, licence, maintenance

- Repo `Fission-AI/OpenSpec`, npm `@fission-ai/openspec`, site `openspec.dev`. Licence MIT ("OpenSpec Contributors", 2024). V (P1 LICENSE, P3).
- Current version 1.15.0 (released 2026-10-24). Releases are roughly weekly: 1.9.0 2026-08-13 ... 1.14.3 2026-10-17, 1.15.0 2026-10-24. V (P3). Repo created 2025-08-05, about 73k stars, about 5k forks, 208 open issues, last push 2026-10-30. V (P3).
- Bus factor: lead maintainer Tabish Bidiwale (@TabishB, 522 commits), second maintainer Clay Good (@clay-good, 194), plus a bot "automation maintainer" and one advisor. So effectively 2 humans. V (MAINTAINERS.md, P3 contributors). Everything is Markdown in git; if the project died the specs and folder convention stay usable (section 2).
- Needs Node >= 20.19 for npm install; since 1.15.0 also a single-file binary (curl install script, `brew install openspec`). V (README, CHANGELOG 1.15.0). Anonymous telemetry (command names + version) is on by default; disable with `OPENSPEC_TELEMETRY=0` or `DO_NOT_TRACK=1` (auto-off when `CI` is set). `openspec update` also pings the npm registry (skip with `OPENSPEC_NO_UPDATE_CHECK`). V (README, cli.md).
- The format has moved a lot: 0.x used `openspec/AGENTS.md`, `openspec/project.md`, `/openspec:proposal|apply|archive`; 1.x ("OPSX") replaced them with skills/commands and `config.yaml`. Migration cleans old files. V (docs/migration-guide.md). Expect more churn; pin the CLI version in any setup note. Docs are inconsistent in places (e.g. docs/cli.md shows a stale `validate --json` shape; `docs/agent-contract.md` is the source-verified one). V.

## 1. The model

### 1.1 Layout (verified by `openspec init` run, P2)

```
openspec/
  config.yaml                      # only file init creates besides the two dirs
  specs/<capability-path>/spec.md  # source of truth: how the system behaves NOW
  changes/<change-id>/             # one folder per proposed change
      .openspec.yaml               # metadata (schema, created, skip_specs, retire_capabilities, x-* keys)
      proposal.md  design.md  tasks.md
      specs/<capability-path>/spec.md   # DELTA spec, not a full spec
  changes/archive/YYYY-MM-DD-<change-id>/   # finished changes, kept whole
  schemas/<name>/schema.yaml + templates/   # optional custom workflows
```

- `<capability-path>` may be flat (`combat`) or nested (`identity/user-auth`); follow the existing layout, kebab-case for new segments. V (schema instructions).
- A change must be a directory directly under `changes/` (no namespace folders; `list` warns `nested_change_directory`). V (agent-contract 4.1).
- A delta spec file must be `specs/<capability>/spec.md`; a `spec.md` directly at `specs/` root or any other filename is an ERROR (it would be ignored by archive). V (validator.ts).
- Nothing in `init` writes `AGENTS.md`, `CLAUDE.md` or `project.md` in 1.x (run). Legacy files are removed on `init`/`update`, `project.md` is left for you to migrate into `config.yaml: context`. V.

### 1.2 Concepts

- **Spec** = behavior contract per capability: Purpose + Requirements, each with scenarios. Describes current truth only.
- **Change** = proposed edit, a folder with artifacts and **delta specs**. Many can be open in parallel.
- **Archive** = apply the deltas to `specs/`, move the folder to `changes/archive/<date>-<id>/`. History is the archive plus git.
- **Artifacts** (schema `spec-driven`, default): proposal -> specs (needs proposal) and design (needs proposal) -> tasks (needs specs and design). "Dependencies are enablers, not gates"; design is optional in practice: run showed `status` 3/4 with design missing and validate passing. V (concepts.md, run).
- Philosophy stated by the project: fluid not rigid, iterative, easy, brownfield-first. No approval gates; artifacts are editable at any time. V (concepts.md, editing-changes.md).

### 1.3 Main spec format (what `validate` expects)

Required headings (exact): `## Purpose` and `## Requirements`; each requirement `### Requirement: <name>`; each scenario `#### Scenario: <name>` (exactly four hashes). V (validator constants: GUIDE_MISSING_SPEC_SECTIONS, GUIDE_SCENARIO_FORMAT; schema text "Scenarios MUST use exactly 4 hashtags ... will fail silently").

Rules enforced (V, `src/core/validation/constants.ts`, validator.ts):
- Requirement text must contain `SHALL` or `MUST` (word match, upper case). Missing = WARNING for ADDED/MODIFIED (error under `--strict` in the sense that `--strict` fails on warnings; see 4.3). Non-English specs: still pass with the warning.
- Every ADDED/MODIFIED requirement needs at least one scenario with a non-empty body (ERROR).
- Requirement text (between header and first scenario) over 500 chars = WARNING. Purpose under 50 chars = WARNING, `TBD`/placeholder purpose = WARNING.
- Spec must have at least one requirement.
- Duplicate requirement names in a spec = ERROR. Names match case-sensitively after trimming whitespace and a trailing `#` run.
- Change-level warning when a change has more than 10 deltas ("consider splitting").
- Scenario step style: the repo convention is `- **WHEN** ... - **THEN** ... - **AND** ...` with optional `**GIVEN**`. Docs also show plain `- GIVEN/WHEN/THEN`; the validator does not check step words, only the `####` header and non-empty body. V.
- Requirement name guidance from the project's own spec: descriptive, under 50 chars; first line after the header is the SHALL statement. V (openspec-conventions spec).

Worked main spec (this is what `openspec/specs/combat/spec.md` looked like after my run; copyable):

```markdown
# Combat Specification

## Purpose
Defines how the bot fights monsters in a room and when it flees, so that deaths stay rare during farming runs.

## Requirements

### Requirement: Retreat At Low Life
The bot SHALL retreat from combat when life falls below 40 percent of maximum.

#### Scenario: Life drops under threshold
- **WHEN** life falls below 40 percent while monsters are in range
- **THEN** the bot drinks a healing potion or leaves the room

### Requirement: Potion Belt Refill
The bot MUST refill the belt to at least 4 healing potions before leaving town.

#### Scenario: Belt low
- **WHEN** the belt holds fewer than 4 healing potions in town
- **THEN** the bot visits the vendor before taking the waypoint
```

Numeric thresholds are just text inside requirement sentences and scenarios; the validator does not parse or check them (run). V.

### 1.4 Change folder, complete worked example

All files below were validated with `openspec validate tune-combat-thresholds --strict` ("is valid") and archived with `openspec archive tune-combat-thresholds --yes` in P2, against the main spec of 1.3 plus a third requirement "Legacy Kill Counter" (SHALL count kills per run; one scenario). V (run).

`openspec/changes/tune-combat-thresholds/.openspec.yaml` (written by `openspec new change`; run):
```yaml
schema: spec-driven
created: 2026-10-30
```

`proposal.md` (template headings from `schemas/spec-driven/templates/proposal.md`; validator wants `## Why` (50 to 1000 chars) and `## What Changes` non-empty):
```markdown
# Proposal

## Why

Soak runs show the bot dying at the Pindleskin gate because the retreat threshold is too low and the belt is refilled too late. Raising both should cut deaths below the acceptance number without lowering kills per hour noticeably.

## What Changes

- Raise the retreat threshold from 40 to 50 percent of maximum life.
- Rename the belt requirement to match the vocabulary used elsewhere.
- Drop the legacy kill counter (the dashboard reads the run log instead).
- Add a requirement for town trips when the belt is empty.

## Capabilities

### New Capabilities
- `town-trips`: when the bot leaves a run to restock in town.

### Modified Capabilities
- `combat`: retreat threshold, belt refill naming, kill counter removal.

## Impact

Affects the combat loop and town routine only. No memory offsets change.
```
(`## Impact` is in the template but not enforced by the validator, V.) Note the older docs show `Intent/Scope/Approach` proposals; those are doc illustrations, the shipped template and validator use Why / What Changes / Capabilities / Impact. V.

`specs/combat/spec.md` (delta for an EXISTING capability; MODIFIED, RENAMED, REMOVED together):
```markdown
# Delta for Combat

## MODIFIED Requirements

### Requirement: Retreat At Low Life
The bot SHALL retreat from combat when life falls below 50 percent of maximum.

#### Scenario: Life drops under threshold
- **WHEN** life falls below 50 percent while monsters are in range
- **THEN** the bot drinks a healing potion or leaves the room

#### Scenario: Life exactly at threshold
- **WHEN** life is exactly 50 percent
- **THEN** the bot keeps fighting

## RENAMED Requirements
- FROM: `### Requirement: Potion Belt Refill`
- TO: `### Requirement: Belt Refill Before Departure`

## REMOVED Requirements

### Requirement: Legacy Kill Counter
**Reason**: The dashboard now reads kills from the run log.
**Migration**: Use the run log kill events.
```

`specs/town-trips/spec.md` (delta for a NEW capability; first section is `## Purpose`):
```markdown
# Delta for Town Trips

## Purpose

Defines when the bot abandons a run and returns to town to restock, so that fights never start with an empty belt.

## ADDED Requirements

### Requirement: Restock Trip On Empty Belt
The bot SHALL return to town when the belt holds no healing potions.

#### Scenario: Empty belt mid-run
- **WHEN** the last healing potion is used outside town
- **THEN** the bot casts Town Portal at the next safe moment
```

`tasks.md` (checkbox format is what progress tracking parses; numbered `## N.` groups with `N.M` ids; a `- [ x]`/`[x]`/`[X]` counts done, any other marker incl `[~]`, `[-]`, `[]` counts open; non-checkbox lines are untracked):
```markdown
# Tasks

## 1. Combat thresholds

- [ ] 1.1 Change retreat threshold constant to 50 percent and verify the unit test for the boundary passes
- [ ] 1.2 Rename belt refill requirement usages in docs and verify grep finds no old name

## 2. Town trips

- [ ] 2.1 Implement restock trip and verify a fake-memory test covers the empty-belt scenario
```

`design.md` (optional; template headings): `# Design`, `## Context`, `## Goals / Non-Goals` (with `**Goals:**` / `**Non-Goals:**`), `## Decisions`, `## Risks / Trade-offs`, optional `## Migration Plan`, `## Open Questions`. Include only for cross-cutting work, new dependencies, data-model/migration/security/performance concerns. V (schema.yaml instruction). Not validated structurally.

### 1.5 Delta semantics (V, run + `specs-apply.ts`)

| Section | Body | Archive effect | Gotchas |
|---|---|---|---|
| `## ADDED Requirements` | full requirement blocks | appended to main spec | name must not already exist (collision aborts) |
| `## MODIFIED Requirements` | the COMPLETE new block (header, text, all scenarios) | replaces the block with the same name | header must match existing name exactly (whitespace-insensitive); missing = archive aborts, nothing changed (run: `MODIFIED failed ... not found`, "Aborted. No files were changed."); partial content loses scenarios at archive, so copy the whole existing block first. An inline note like `(Previously: 40 percent)` is NOT stripped: it ends up in the main spec (run). Validate also runs a scenario-loss check (MODIFIED drops scenarios present in main spec) |
| `## REMOVED Requirements` | `### Requirement: <name>` plus `**Reason**` and `**Migration**` (bullets of names also accepted) | block deleted. Removing the last requirement retires the capability (deletes spec.md) ONLY if `.openspec.yaml` has `retire_capabilities: true`; else archive aborts | REMOVED of a name not in spec = warning "treating as already removed" |
| `## RENAMED Requirements` | lines `- FROM: \`### Requirement: Old\`` immediately followed by `- TO: \`### Requirement: New\`` | retitles block in place (position kept) | to also change content, put the block under MODIFIED using the NEW name; FROM+TO unpaired = ERROR; MODIFIED using the old name = ERROR |
| `## Purpose` | 1-2 sentences, new capability only | seeds Purpose of the created main spec; ignored for existing specs (otherwise placeholder `TBD - created by archiving change <id>. Update Purpose after archive.`) | |

Apply order inside one spec: RENAMED, REMOVED, MODIFIED, ADDED. Cross-section conflicts (same name in MODIFIED+REMOVED, MODIFIED+ADDED, ADDED+REMOVED, RENAMED TO colliding with ADDED) are ERRORs. A brand-new capability may contain only ADDED (MODIFIED/RENAMED need an existing spec; REMOVED ignored with warning). V.
Content in a delta outside the four sections or outside requirement blocks (notes, prose) is NOT carried to the main spec; validate warns. A well-formed requirement placed outside a delta section is ignored with a warning. V.
Requirement-level granularity: there is no scenario-level delta. Two changes MODIFYing the same requirement conflict; the second archive overwrites the first (project's own plan doc admits this and proposes fingerprints; validate now has a scenario-loss check, and INFO on merge conflicts). V (openspec-parallel-merge-plan.md; whether the plan is fully implemented is U).

### 1.6 Change metadata `.openspec.yaml`
Keys: `schema`, `created`, `skip_specs: true` (change has no spec-level behavior, zero deltas accepted; delta files then become an error), `retire_capabilities: true`, optional `goal`, and your own top-level keys prefixed lowercase `x-` (never touched, e.g. `x-tracker: SB-5497`). Unknown other keys are reported. V (cli.md, validator).
Change ids: lowercase kebab-case, letters/digits/single hyphens, leading digit allowed (`00001-add-auth`), no uppercase/underscore/double hyphen. V (cli.md). The `add-/fix-/update-/remove-` verb prefix is convention only (V in docs examples, not enforced).

## 2. Lifecycle and CLI

Real command names (1.15.0, from `docs/cli.md` and run). All `openspec ...` run in the terminal; `/opsx:*` run in the agent chat. V.

| Command | Purpose / notes |
|---|---|
| `openspec init [path] --tools claude\|all\|none [--force --profile core\|custom --language X]` | creates `openspec/{specs,changes/archive,config.yaml}` and tool skills/commands. `--tools` makes it non-interactive. |
| `openspec update [--dry-run --json --force]` | regenerate skills/commands after upgrading the CLI; generated files are OpenSpec's, hand edits are overwritten |
| `openspec new change <id> [--description --goal --schema --json]` | creates `changes/<id>/.openspec.yaml` only (run); artifacts are written by you/agent |
| `openspec list [--specs] [--json] [--sort recent\|name]` | active changes with task counts, or specs with requirement counts |
| `openspec show <item> [--type change\|spec] [--json --deltas-only --no-scenarios -r N]` | read one |
| `openspec validate [item] [--all --changes --specs --archived --strict --json --type]` | structure check; exit 1 on failure. `--strict` fails on warnings. `--archived` = fail if any archived change has unticked tasks |
| `openspec status --change <id> [--json]` | artifact state (done/ready/blocked/skipped), nextSteps |
| `openspec instructions <artifact\|apply\|archive> --change <id> --json` | what an agent needs to write an artifact / apply tasks (template, rules, context, task list, state) |
| `openspec archive <id> [--yes --skip-specs --no-validate]` | validate, merge deltas, move to `archive/<date>-<id>`; non-interactive needs `--yes` |
| `openspec view` | terminal dashboard (human only) |
| `openspec schemas / schema init|fork|validate|which`, `templates`, `config [profile]`, `store ...`, `workset ...`, `doctor`, `context`, `feedback`, `completion` | schemas, settings, beta multi-repo "stores" |

Archive details (run + cli.md): validates first; incomplete tasks only WARN and `--yes` continues (run: "3 incomplete task(s) found. Continuing due to --yes flag"); all spec writes are checked first and the whole archive aborts with no files changed on any failure; date prefix is the archive day; change files are preserved byte for byte. Changes with only deltas already synced archive with zero totals. Validation of an archived item set later via `validate --all` passes for the merged specs (run). V.
`/opsx:sync` merges deltas into main specs without archiving (for long changes or when another change must build on them); the agent version is a "semantic" merge (can add scenarios without duplication), the CLI `archive` is deterministic block replacement. V (commands.md, specs-apply.ts). The two paths can differ, so decide which one you use.

Slash-command lifecycle (core profile): `/opsx:explore` (optional) -> `/opsx:propose <id-or-description>` (creates the change + all planning artifacts, then STOPS; explicitly planning only) -> review -> `/opsx:apply [id]` (works unchecked tasks, ticks boxes, resumable) -> `/opsx:sync` (optional) -> `/opsx:archive`. Also `/opsx:update` (revise planning artifacts coherently, never code). Expanded profile (enable via `openspec config profile` then `openspec update`): `new`, `continue`, `ff`, `verify` (completeness/correctness/coherence report CRITICAL/WARNING/SUGGESTION, non-blocking), `bulk-archive` (several changes, resolves spec conflicts by checking code, chronological), `onboard`. V.

### 2.1 Using it WITHOUT the Node CLI (pure convention)
The CLI is a scaffolder, validator and deterministic merger; the model is plain Markdown + git, and OpenSpec never touches git. V (team-workflow.md; init created only folders and config). To go CLI-free a human/agent must replicate:
1. Layout in 1.1; create `changes/<kebab-id>/` with proposal.md, tasks.md, optional design.md, `specs/<capability>/spec.md` deltas. `.openspec.yaml` is optional metadata (the schema default applies).
2. Validation by hand: headings exactly as in 1.3 and 1.5; four hashes for scenarios; SHALL/MUST in every requirement; at least one scenario per requirement; Why >= 50 chars; every MODIFIED/REMOVED/RENAMED name must exist in the main spec; no name in two sections.
3. Archive by hand: apply RENAMED, REMOVED, MODIFIED, ADDED into `specs/<cap>/spec.md` (create it with `## Purpose` + `## Requirements` if new), strip `(Previously: ...)` notes, then `git mv changes/<id> changes/archive/YYYY-MM-DD-<id>`.
4. Progress = count `[x]` in tasks.md.
What you lose: the validator (scenario-loss, name matching, conflicts), `status/instructions` JSON for agents, atomic merge, the generated slash commands/skills. A reasonable compromise is to run the CLI only as a checker (`npx ... validate --all --strict` in CI) and keep authoring conventions in a repo doc. U for how well agents follow a hand-written convention without the generated skills (no evidence found).

## 3. Claude Code integration

- `openspec init --tools claude` (run) writes `.claude/skills/openspec-{explore,propose,apply-change,archive-change,sync-specs,update-change}/SKILL.md` and `.claude/commands/opsx/{explore,propose,apply,archive,sync,update}.md`; expanded profile adds new/continue/ff/verify/bulk-archive/onboard (files `openspec-new-change`, `openspec-continue-change`, `openspec-ff-change`, `openspec-verify-change`, `openspec-bulk-archive-change`, `openspec-onboard`). Typed as `/opsx:propose` in Claude Code (folder `commands/opsx/` namespaces it). V.
- It does NOT write or edit `AGENTS.md`/`CLAUDE.md` in 1.x (run). Project-specific guidance goes in `openspec/config.yaml`: `context:` (injected into every artifact prompt as `<context>`), `rules:` per artifact id (`proposal`, `specs`, `design`, `tasks`, injected as `<rules>` only for that artifact), `operations.apply.guidance` / `operations.archive.guidance` (advisory, delivered via `openspec instructions apply|archive --json`). Keep `context` short; it is sent on every request. V (customization.md, config.yaml run).
- Each skill carries a long preamble (store selection, "project check", planning boundary) and calls the CLI with `Bash(openspec:*)` allowed. So the Node CLI must be on PATH for the agent flow. V.
- Agent contract: `/opsx:propose` creates planning artifacts only and must not start code; apply is a separate explicit request. Archive skill: runs `instructions archive --json`, checks `status --json` artifacts, `list --json` task counts (warns, asks to confirm if incomplete), assesses sync state, then archives. V (skill text).
- Machine-readable surface: `--json` on list/show/validate/status/instructions/new change/archive, one JSON doc on stdout, errors as `status` diagnostics, exit 1 on failure; full shapes in `docs/agent-contract.md`. V.
- Recommended by the project: high-reasoning models (README names Codex 5.5 and Opus 4.7), clear context before implementation. V.

## 4. Writing good specs and changes

### 4.1 Granularity
- A capability is a durable, cohesive system behavior (`user-auth`, not `add-login-endpoint`); avoid catch-all capabilities; reuse existing paths (`openspec list --specs` first) rather than near-duplicates. V (schema instruction).
- Count: the project's own repo has about 36 capabilities (flat names like `cli-archive`, `artifact-graph`, `config-loading`). V (P1 `openspec/specs/`). The docs give no hard number of requirements per spec; only the 500-char per-requirement soft limit and "one behavior per requirement". The validator has no cap on requirements per spec. Practical heuristic (U, mine): split a capability when it passes about 15-20 requirements or two unrelated owners.
- Organize flat for small systems; by domain (`identity/user-auth`) for big ones. V.

### 4.2 Requirements and scenarios (V, docs/writing-specs.md and spec-driven instruction)
- One observable behavior per requirement, one SHALL/MUST (SHOULD/MAY allowed but the validator only recognizes SHALL/MUST). "A tester who has never seen the code could tell if it passed."
- Behavior not implementation: no class names, libraries, step-by-step; those belong in design.md/tasks.md. Test: if the implementation can change without changing observable behavior, it is not spec.
- Scenarios: WHEN/THEN (optionally GIVEN/AND), concrete, testable, named for the case ("Rejects an expired token"), cover edge/error cases, not only the happy path; a scenario that restates the requirement tests nothing.
- Progressive rigor: "lite" by default; full detail only for cross-team, API/contract, migration, security work.
- Numbers: put them in the sentence and the scenario; the toolchain treats them as text. Consider one requirement per threshold so a later MODIFIED replaces one small block.

### 4.3 Change sizing and naming
- One intent you can say in one sentence. Too big if: scope is a list of unrelated features, review takes an afternoon, two people could not work on it without colliding, half the tasks could ship alone. Validator warns above 10 deltas. One owner per change folder. V.
- `skip_specs: true` for pure refactor/tooling/docs: specs describe behavior, do not invent a requirement to satisfy validation. V.
- Tasks: small enough for one session, ordered by dependency, each states how it is verified, tests/docs land inside the group that owes them (no final catch-all test group). V (tasks instruction).
- Naming: kebab-case id, verb prefix by convention (`add-`, `fix-`, `update-`, `remove-`, `refactor-`). V (docs examples).

### 4.4 Pitfalls and anti-patterns (V unless marked)
1. `### Scenario:` or bullets instead of `#### Scenario:`: not recognized, "fails silently" (INFO line only, or error for missing scenario).
2. MODIFIED with partial content: archive replaces the whole block, so omitted scenarios vanish. Use ADDED for new concerns.
3. MODIFIED/REMOVED/RENAMED against a name that does not match the main spec (even case): archive aborts. Run `validate` against the real main specs before archive (validate reports "Archive would refuse this delta").
4. Leaving `(Previously: X)` or other prose notes in a MODIFIED block: they get copied into the main spec (run).
5. Delta file in the wrong place (`specs/spec.md`, `specs/foo.md`): ERROR, ignored by archive.
6. Two parallel changes MODIFYing the same requirement: last archive wins; keep requirements small and changes disjoint; use `/opsx:sync` or archive in order and re-check.
7. ADDED a requirement that already exists, or described new behavior as MODIFIED.
8. Back-filling specs for code you are not changing: they go stale. Secondary sources agree reverse-engineered specs "look plausible but diverge". S.
9. Archived with unchecked tasks (`--yes` only warns); use `validate --archived` in a hook.
10. Hand-editing generated skills/commands: overwritten by `openspec update` (commands) or left until version changes (skills).
11. Letting `## Purpose` stay `TBD` after archiving a new capability: validate warns; edit the main spec directly.
12. Stale artifacts after code drift: run `/opsx:verify` (expanded) and reconcile before archive, because archive makes the specs the record of truth. V.

## 5. Quick cheat sheet

```
openspec init --tools claude            # once; commit openspec/ and .claude/
openspec new change add-foo             # scaffold (or /opsx:propose add-foo)
  # write proposal.md, specs/<cap>/spec.md (deltas), design.md?, tasks.md
openspec validate add-foo --strict      # must pass
openspec status --change add-foo
  # implement, tick [x] in tasks.md   (/opsx:apply)
openspec validate --all --strict
openspec archive add-foo --yes          # merges deltas, moves to archive/DATE-add-foo
openspec list [--specs]  |  openspec show <id> --type spec
```
Headings to memorize: `## Purpose`, `## Requirements`, `### Requirement: Name`, `#### Scenario: Name`, `## ADDED|MODIFIED|REMOVED|RENAMED Requirements`, proposal `## Why`, `## What Changes`, `## Capabilities`, `## Impact`, tasks `## 1. Group` + `- [ ] 1.1 ...`.

## 6. Alternatives the project compares itself against (V, README "How we compare")
- vs GitHub Spec Kit: "thorough but heavyweight, rigid phase gates, lots of Markdown, Python setup". vs Kiro (AWS): "locked into their IDE". vs nothing. This is marketing by the project; treat as claims, not measurement. Secondary blogs broadly repeat it (OpenSpec lighter, better brownfield, Spec Kit heavier with a "constitution"), plus caveats: ceremony if review is lax, spec drift, one author reported spec review roughly doubling feature time (Medium, 2025-26, single anecdote). S.

## 7. Source list (fetched 2026-10-30)
- https://github.com/Fission-AI/OpenSpec (README.md, LICENSE, MAINTAINERS.md, CHANGELOG.md, openspec-parallel-merge-plan.md) V
- docs/{concepts,writing-specs,getting-started,existing-projects,team-workflow,agent-contract,cli,commands,customization,supported-tools,how-commands-work,migration-guide,faq,editing-changes,reviewing-changes,workflows}.md in the same repo V
- schemas/spec-driven/schema.yaml and templates/*.md V; src/core/validation/{constants,validator}.ts, parsers/requirement-blocks.ts, specs-apply.ts V
- GitHub API: repo stats, releases, contributors, issues #1714 (worktrees + stores gaps), #1729 (multiple spec roots in a monorepo), #1942 (closed, monorepo per-project) V
- npx @fission-ai/openspec 1.15.0 trial in a temp dir V
- Web search (snippets only, not fetched): medium.com/@mathivananmani/reviving-brownfield-projects-with-ai-a-comparative-look-at-github-spec-kit-openspec-and-bmad-..., besthub.dev OpenSpec deep dive, codemyspec.com/blog/openspec-explained, augmentcode.com spec-driven brownfield guide S
- Not verified: openspec.dev web docs (README links there; repo docs/ are the same source), Discord, talks, any independent multi-agent or monorepo experience report.
