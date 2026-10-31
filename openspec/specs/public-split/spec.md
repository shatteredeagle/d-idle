# public-split Specification

## Purpose
What is public and what stays private at the public cut of the repository: file classes, private-name and host-config rules, the leakcheck warn and fail modes, no Blizzard-derived data, and the cut preconditions and verification.

## Requirements

### Requirement: Classes decide what is public
Every tracked path SHALL be classed `public`, `private` or `scrub-needed` by the first matching rule of `docs/process/public-manifest.json`; private-class files (lab layout, scripts that drive the lab, status board, agent definitions, go-live material) live in a separate private ops repo. Source: docs/process/public-split.md section 1; user decision 2026-10-30.

#### Scenario: New file
- **WHEN** a path matches no private rule
- **THEN** it is public and leakcheck scans it

### Requirement: No scrub-needed entries at go-live
At go-live `go run ./cmd/leakcheck -gen-manifest` SHALL print no `scrub-needed` entry. Source: docs/process/public-split.md sections 1 and 5.

#### Scenario: Go-live manifest
- **WHEN** `-gen-manifest` runs on the go-live tree
- **THEN** it prints no `scrub-needed` entry

### Requirement: No private names in committed files
Committed files SHALL contain no private hostname, private IP address, home path, person's name, machine name, secret or email shape; `go run ./cmd/leakcheck` reports them, exceptions exist only in `docs/process/leakcheck-allow.txt` and are reviewed like code. Source: docs/process/public-split.md sections 1 and 7.

#### Scenario: Home path added
- **WHEN** a commit adds a `/home/<user>` path to a public-class file
- **THEN** leakcheck reports the hit and, in fail mode, the push is refused

#### Scenario: Allow-list entry
- **WHEN** a file must name a private string to work
- **THEN** it gets an allow-list line reviewed in the same commit

### Requirement: Host config from the environment with role names
Public code and scripts SHALL read lab values from environment variables (process environment or a gitignored `local.env`) named by role (`D2_WORKER_HOST`, `D2_NAS_HOST`, `D2_BOARD_URL`, `D2_FORGE_URL`, `D2_COLD_DIR`, `D2_GAME_DIR`), with defaults on the reserved `.example` domain; prose says the role, never the machine. Source: docs/process/public-split.md section 2.

#### Scenario: Unset variable
- **WHEN** `D2_WORKER_HOST` is unset
- **THEN** the script uses `worker.example` and leakcheck finds no hit

#### Scenario: Prose names a machine
- **WHEN** a doc names a machine instead of "the worker host"
- **THEN** leakcheck reports it

### Requirement: Leakcheck warns before go-live
Until go-live, `go run ./cmd/leakcheck -mode warn` SHALL run in the pre-push hook and CI, report every hit and exit 0. Source: docs/process/public-split.md section 7.

#### Scenario: Before go-live
- **WHEN** a push contains a hit
- **THEN** the hook prints it and the push goes through

### Requirement: Leakcheck fails from go-live
From the go-live step, leakcheck SHALL run with `-mode fail` in the pre-push hook and CI and exit 1 on any hit. Source: docs/process/public-split.md section 5 step 3.

#### Scenario: After the flip
- **WHEN** a push contains a hit
- **THEN** the hook and the CI step exit 1 and the push is refused

### Requirement: Blizzard-derived data is never committed
No game data, extracted tables or Blizzard-derived content (including gear tables) SHALL be committed to any repo or image; such data is extracted at setup from the user's own game files into a gitignored location. Source: docs/adr/0010-licence-same-as-d2d.md; docs/licensing.md; docs/process/public-split.md.

#### Scenario: Setup on a fresh clone
- **WHEN** a user follows the setup steps with their own game files
- **THEN** the tables are created locally and `git status` shows no new tracked file

#### Scenario: Data file staged
- **WHEN** a commit stages an extracted table
- **THEN** review rejects it and the path is added to .gitignore

### Requirement: Cut preconditions
The go-live history cut SHALL start only with the user's go, on a quiet master (every agent branch merged or abandoned) where `go run ./cmd/leakcheck -mode fail` exits 0. Source: docs/process/public-split.md section 5 steps 1 to 2.

#### Scenario: Branch still open
- **WHEN** an agent branch is unmerged at cut time
- **THEN** the cut does not start

#### Scenario: No go
- **WHEN** the user has not given the go
- **THEN** no history is rewritten and no remote is renamed

### Requirement: Cut result and verification
The cut SHALL produce one orphan commit "Initial public snapshot" tagged `v0-public`, keep the old history only in a private read-only archive repo, and pass a fresh-clone check on another machine (`go build ./...`, `go test -short ./...`, leakcheck fail mode). The step-by-step runbook stays in docs/process/public-split.md section 5. Source: same.

#### Scenario: Cut verified
- **WHEN** the cut finishes
- **THEN** a fresh clone builds, passes the short tests and leakcheck fail mode exits 0

#### Scenario: History check
- **WHEN** `git log --oneline` runs on the public master
- **THEN** it shows the single snapshot commit and no earlier history

### Requirement: Ops material stays private
New private material (lab layout, runner and capacity docs, scripts driving the lab, instance tables) SHALL go to the private ops repo and not "temporarily" into the public repo; public stubs call `tools/ops.sh`, which exits 2 with a one-line message when the ops checkout is missing. Source: docs/process/public-split.md sections 4 and 7.

#### Scenario: Missing ops checkout
- **WHEN** a public stub runs without the ops repo present
- **THEN** it prints "ops repo not present" and exits 2

#### Scenario: New lab doc
- **WHEN** an agent writes a doc that names the lab layout
- **THEN** it is committed in the ops repo, not here
