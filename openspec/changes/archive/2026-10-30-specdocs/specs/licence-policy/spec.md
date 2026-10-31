## ADDED Requirements

### Requirement: Repository licence
The repository SHALL be licensed GPL-3.0-or-later, with the GPLv3 text in the root LICENSE file and the plain-words notice naming the identifier. Source: docs/adr/0010-licence-same-as-d2d.md (user, 2026-10-30); docs/licensing.md.

#### Scenario: Licence present
- **WHEN** the tree is checked at the root
- **THEN** LICENSE holds the GPLv3 text and docs/licensing.md names GPL-3.0-or-later

#### Scenario: Notice names the identifier
- **WHEN** docs/licensing.md is read
- **THEN** it states GPL-3.0-or-later

### Requirement: SPDX header on new source files
New .go, .sh, .py and .js files SHALL start with `SPDX-License-Identifier: GPL-3.0-or-later` (`//` or `#`, after a shebang); existing files are not mass-edited. Source: docs/process/spdx.md; ADR 0010.

#### Scenario: New file without header
- **WHEN** a diff adds a source file without the SPDX line
- **THEN** review asks for the line (no tool checks it yet; a warn-only changecheck rule is proposed in docs/process/spdx.md)

#### Scenario: Existing file touched
- **WHEN** an agent edits an existing file that has no header
- **THEN** the line MAY be added in that commit; no mass edit is required

### Requirement: Third-party code needs a compatible licence
Code copied into the repo SHALL come only from a GPL-compatible licence with its notice kept and listed in THIRD_PARTY.md; code from a project without a licence file MUST NOT be copied. Source: docs/adr/0010-licence-same-as-d2d.md third-party policy; THIRD_PARTY.md (Kolbot: ideas only; Tyniann terms unverified, U).

#### Scenario: MIT source
- **WHEN** code from an MIT project is copied
- **THEN** its notice is kept and THIRD_PARTY.md lists it

#### Scenario: Unlicensed source
- **WHEN** a project has no licence file
- **THEN** only ideas, never code, enter the repo

### Requirement: Redivivus numbers only
From redivivus the repo SHALL take numbers and paths only, tagged with their source, and no code or prose. Source: docs/adr/0010-licence-same-as-d2d.md; THIRD_PARTY.md.

#### Scenario: Table value used
- **WHEN** a value comes from redivivus
- **THEN** the docs tag it S with the source and the code holds only the number

### Requirement: Stdlib-only dependencies
The Go module SHALL have no third-party dependency: `go list -m all` prints only `d2lod-bot`, so no dependency licence is in play until a card adds one with its licence check. Source: docs/licensing.md; AGENTS.md; ADR 0010.

#### Scenario: Module check
- **WHEN** `go list -m all` runs
- **THEN** it prints only the module itself

### Requirement: Fork patches stay downstream
Patches to the d2d fork SHALL stay in the private downstream fork and never enter this repo until the user decides otherwise; an upstream contribution is an opt-in PR authored as the user and posted only with the user's go. Source: docs/adr/0003 and ADR 0010 follow-up (user, 2026-10-30); docs/licensing.md.

#### Scenario: Patch in a diff
- **WHEN** a diff adds a d2d fork patch file (a `.patch` or `.diff` for d2d sources)
- **THEN** `git diff --name-only` shows it and the diff is rejected; the work goes to the fork branch

### Requirement: Distribution needs the corresponding source
A container image or binary SHALL NOT be given to anyone outside the project unless the matching source, including any d2d patches baked in, is offered with it, and game files never go in the image. Private or internal use is not distribution. Source: docs/licensing.md; docs/adr/0010-licence-same-as-d2d.md.

#### Scenario: Image to a third party
- **WHEN** an image is about to be shared
- **THEN** the matching source offer is attached and the image holds no game file

#### Scenario: Internal run
- **WHEN** the image runs only on the maintainer's own hosts
- **THEN** no source offer is needed
