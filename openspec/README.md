# openspec/: contracts and change cards

How this project writes down (1) the stable contracts the bot must honour and (2) every job an agent is given. The layout and headings are OpenSpec's (https://github.com/Fission-AI/OpenSpec, MIT, pinned 1.15.0), used WITHOUT its Node CLI: our checker is `go run ./cmd/changecheck`. If the CLI is ever adopted, nothing here has to be rewritten (the one header block we add was checked against `openspec validate --all --strict`, V 2026-10-30, see docs/process/trial-notes.md). Background: docs/process/spec-approach-evaluation.md (why), docs/process/openspec-reference.md (exact validator rules).

## What goes where

| thing | where | rule |
|---|---|---|
| Stable contract, numbers that must hold (gates, policies, interfaces) | `openspec/specs/<capability>/spec.md` | requirement + scenario form, one owner at a time |
| One job (an agent's brief) | `openspec/changes/<id>/` | one folder per job, deleted from `changes/` by archiving |
| Finished job | `openspec/changes/archive/YYYY-MM-DD-<id>/` | whole folder kept, header has merge hash and result |
| Who owns which directory right now | `docs/OWNERS` | one line per path glob, kept in sync by changecheck |
| Why we chose X | `docs/adr/NNNN-title.md`, one line each in `docs/decisions.md` | MADR style, user decisions only |
| Knowledge, evidence, V/S/U-tagged findings | `docs/*.md` (unchanged) | specs state WHAT must hold; docs say how we know |

Do not copy prose from docs into specs. A spec links to its evidence doc and states the contract in 1 to 2 pages.

## Spec files (`openspec/specs/<capability>/spec.md`)

Exact headings (OpenSpec's validator needs them; changecheck checks the same rules):

```markdown
# <Capability> specification

## Purpose
One or two sentences, at least 50 characters, no TBD.

## Requirements

### Requirement: Short descriptive name
The bot SHALL <one observable behaviour, numbers inline>. Source: <doc or decision>.

#### Scenario: Named case
- **WHEN** <concrete condition>
- **THEN** <observable result>
```

Rules: `SHALL` or `MUST` in every requirement (upper case); at least one `#### Scenario:` (exactly four hashes) per requirement; one requirement per threshold so a later change modifies one small block; requirement text under 500 characters before the first scenario; unique names; behaviour, not implementation. Put the SOURCE of every number in the requirement text (plan-v2 section, decision date, doc).

## Change folder (`openspec/changes/<id>/`)

`<id>` is lowercase kebab-case, the same as the agent name where possible (`d2dwire`, `merc-revive`). Files:

- `proposal.md`: `## Why` (50 to 1000 characters) and `## What Changes` (bullets). Required.
- `tasks.md`: our header block (below) then `## 1. Group` / `- [ ] 1.1 task and how it is verified` checkboxes. Required.
- `.openspec.yaml`: `schema: spec-driven`, `created: YYYY-MM-DD`, `skip_specs: true` when the change adds no spec delta. Needed only if the real CLI is used; `changecheck new` writes it.
- `specs/<capability>/spec.md`: delta spec with `## ADDED|MODIFIED|REMOVED|RENAMED Requirements` (a MODIFIED block is the COMPLETE new requirement incl. all scenarios). Only when the job changes a contract.
- `design.md`: optional, cross-cutting work only.

### The header block (ours; OpenSpec ignores it, V)

First fenced block of tasks.md, language tag `yaml`:

```yaml
change: d2dwire              # = folder name
status: in-progress          # proposed | in-progress | done | archived
agent: d2dwire
branch: agent/d2dwire
worktree: ~/code/d2lod-wt/d2dwire
owns:                        # path globs only this change may edit; no overlap with another active change
  - cmd/d2bot/d2d*.go
exclusive:                   # shared resources only this change may use while active
  - d2d-control-socket
gates:                       # acceptance, each with a number or a named test/command
  - "go test ./cmd/d2bot -run D2D passes"
merge:                       # filled at merge: commit hash
result:                      # filled at merge: one line, say verified live or not
```

Exclusive resource names (use these exact strings so the checker can compare): `instance:iN`, `d2d-control-socket`, `d2d-lab-binary`, `d2d-lab-lock` (/tmp/d2d-lab.lock), `bot-binary:<path>`, `fork-branch:<name>`, `upstream-pr:<topic>` (anything posted to GitHub: only with the user's go), `soak-slot:<n>`.

## Life cycle

1. **propose** (coordinator): `go run ./cmd/changecheck new -id X -agent X -owns 'dir/**' -exclusive 'res'` writes the folder (status `proposed`), the coordinator fills Why, tasks, gates and runs `go run ./cmd/changecheck` (no overlap, no clash). Add a line to `docs/OWNERS`.
2. **in-progress**: the agent is launched ("implement change X", below), sets `status: in-progress`, ticks tasks, stays inside `owns:`.
3. **done**: agent fills `result:` and the report; coordinator merges with a message `Merge agent/X: ...` (or a `Change: X` trailer).
4. **archive** (coordinator, same merge step): `git mv openspec/changes/X openspec/changes/archive/YYYY-MM-DD-X`, set `status: archived` and `merge: <hash>`, apply any spec deltas to `openspec/specs/` by hand (RENAMED, REMOVED, MODIFIED, ADDED order; strip `(Previously: ...)` notes), drop the line from `docs/OWNERS`, `go run ./cmd/changecheck status -write`.

## Launching an agent from a change id

The brief shrinks to this (the standing procedure is in `.claude/agents/*.md` and AGENTS.md):

```
Agent <ID>, implement change <ID>.
Read AGENTS.md, openspec/README.md, docs/plan/STATUS.md and openspec/changes/<ID>/ (proposal, tasks, specs/).
Branch agent/<ID>, worktree ~/code/d2lod-wt/<ID>. Edit only what `owns:` lists; use only the `exclusive:` resources.
Done = every `gates:` line holds; tick tasks.md, fill `result:`, report under 300 words.
```

Add an instance line (`instance: iN` under `exclusive:`) for live agents, and a time box. Anything the card does not say is not in scope.

## Merge checklist (coordinator)

1. `git diff --name-only master...agent/X` is inside the card's `owns:` (`go run ./cmd/changecheck -diff agent/X`).
2. Every `gates:` line holds in the agent's report (numbers, not adjectives); live-only gates stay `U` in `result:`.
3. gofmt, `go vet`, tests of the touched packages; whole repo via `labrun worker` (output to a file, read for FAIL).
4. Merge commit subject `Merge agent/X: ...`; then archive (step 4 above) and regenerate STATUS (`changecheck status -write`).
5. Anything touching the private d2d fork or lab patches is already pushed to the forge `<owner>/d2d-fork` (AGENTS.md rule); nothing was posted to GitHub without the user's go.
6. `docs/OWNERS` and `docs/BACKLOG.md` lines for X removed or updated.

## CLI (optional, pinned)

`tools/openspec.sh validate --all --strict` and `tools/openspec.sh archive <id> --yes` run the real OpenSpec 1.15.0 through `npx` with telemetry off (needs node >= 20.19; nothing installed). `init` and `update` are blocked by the wrapper: they would write skills, slash commands and instruction files that compete with AGENTS.md. `archive` merges the deltas into `openspec/specs/` atomically and aborts with no change on a name mismatch (V, trial in docs/process/trial-notes.md); it does NOT touch our header, so set `status: archived`, `merge:` and `result:` by hand afterwards. `changecheck -cli` calls validate as a second opinion.

## Checker

`go run ./cmd/changecheck` (warn-only; add `-strict` to fail on warnings) checks: required files and header per change, ownership overlap and `exclusive` clashes among active changes, spec and delta format, recent merges naming an existing change, `docs/OWNERS` in sync, generated block of docs/plan/STATUS.md current. `changecheck status [-write]` regenerates that block. Plan for turning checks fatal: docs/process/trial-notes.md.
