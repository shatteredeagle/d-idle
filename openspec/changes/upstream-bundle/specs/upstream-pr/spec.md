## MODIFIED Requirements

### Requirement: One claim per PR
A change to the upstream project SHALL carry one claim per commit; a PR MAY bundle several claims when the maintainer prefers it, and each claim keeps its own differential proof, verifier verdict, ponytails and commit. A second gap found on the way is a limit note or its own commit, never folded into a claim. Source: docs/process/upstream-pr-standard.md section 4; maintainer (psi29a) 2026-10-31.

#### Scenario: Two proven claims
- **WHEN** two claims each have a complete proof and a verifier verdict
- **THEN** they MAY go in one PR as two commits, the body has one section per claim, and each section has its own numbers and limits

#### Scenario: Second gap found
- **WHEN** the work reveals another disagreement with game.exe
- **THEN** it is written under limits as a candidate, or becomes its own commit with its own proof, and no claim grows

#### Scenario: Claim without proof
- **WHEN** a candidate has no complete proof or verifier verdict
- **THEN** it stays out of the bundle and the pack lists why

### Requirement: PR spacing and bundled follow-ups
Our upstream work SHALL have at most one open PR per topic bundle unless the maintainer says otherwise; review fixes SHALL go out as one push after `gh pr checks` is green. The fixed one-hour spacing no longer applies. Source: docs/process/upstream-pr-standard.md section 8; maintainer 2026-10-31.

#### Scenario: Second PR too soon
- **WHEN** a bundle for a topic is still open
- **THEN** new proven claims for that topic join it as commits or wait; no second PR is opened

#### Scenario: Review fixes
- **WHEN** a PR needs three small fixes after review
- **THEN** they go out as one push with the user's go
