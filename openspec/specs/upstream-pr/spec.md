# upstream-pr Specification

## Purpose
What an upstream pull request to the decompilation project must prove and disclose before it is asked for: differential proof against game.exe, one claim per PR, evidence pack, authorship and body format, and no posting without the user's go.

## Requirements

### Requirement: Differential proof against game.exe
Every upstream PR claim SHALL carry a differential test that runs game.exe's own code and our code on the same inputs: boundary cases exhaustive plus at least 100000 random cases (or the whole input space when smaller), mismatches counted before the change and ZERO after, with the exact reproduce commands in the PR body. Source: docs/process/upstream-pr-standard.md section 1; user rule 2026-10-30.

#### Scenario: Claim with proof
- **WHEN** a candidate PR is reviewed for go/no-go
- **THEN** the evidence pack lists the case counts, mismatches before (more than 0) and after (0), and commands a reviewer can paste

#### Scenario: Small input space
- **WHEN** the whole input space is under 100000 cases
- **THEN** it is run exhaustively and the pack shows 0 mismatches after the change

#### Scenario: Too few cases
- **WHEN** the random run covers fewer than 100000 cases and the input space is larger
- **THEN** the pack says `PROOF: incomplete` and the coordinator does not ask the user for a go

### Requirement: Address-anchored claims without unverified links
Each function, table, offset and traced behaviour in a claim's causal chain SHALL be tagged V (run or read by us, how), S (upstream doc) or U (unverified), and the chain from game.exe to our change MUST contain no U link: a U is verified or removed from the claim and listed under limits. Source: docs/process/upstream-pr-standard.md section 2.

#### Scenario: U link found
- **WHEN** one link of the chain is tagged U
- **THEN** it is verified by an emulator run or a disassembly read, or the claim is narrowed and the link listed under limits

### Requirement: pr_guard clean
Before a pack is marked complete the branch SHALL pass upstream's `tools/pr_guard.py` with 0 FAIL and 0 WARN. Source: docs/process/upstream-pr-standard.md section 2.

#### Scenario: Guard clean
- **WHEN** the branch is checked with pr_guard
- **THEN** it prints 0 FAIL and 0 WARN

### Requirement: Statistics are corroboration only
Probe statistics (casts per kill, XP per hour, kills per run) SHALL NOT carry a claim; they MAY appear after the proof in a labelled "Measured effect" paragraph that states what they do and do not show. Source: docs/process/upstream-pr-standard.md section 3.

#### Scenario: Statistics after a complete proof
- **WHEN** a pack includes probe statistics
- **THEN** they sit under a "Measured effect" paragraph after the proof

#### Scenario: Numbers without a proof
- **WHEN** a candidate has only statistical evidence
- **THEN** the pack is `PROOF: incomplete` and the recommendation is no-go or narrow the claim

### Requirement: One claim per PR
A PR SHALL change one behaviour; a second gap found on the way becomes a note under limits and a candidate issue, not a second change. Source: docs/process/upstream-pr-standard.md section 4; upstream CONTRIBUTING.

#### Scenario: Second gap found
- **WHEN** the work reveals another disagreement with game.exe
- **THEN** it is written under limits as a candidate issue and the PR keeps one claim

### Requirement: Nothing else moved
Upstream's own tests, `tools/presubmit.py --quick` warnings and the scripted state hashes (`tools/d2ddet`) MUST be unchanged apart from the claimed field, and every caller of a changed function SHALL be listed with what it now does differently. Source: docs/process/upstream-pr-standard.md section 4.

#### Scenario: Hash drift
- **WHEN** a state hash differs in a field other than the claimed one
- **THEN** the pack is incomplete until the difference is explained or removed

#### Scenario: Callers listed
- **WHEN** a changed function has three callers
- **THEN** the pack lists all three with their changed or unchanged behaviour

### Requirement: Independent adversarial verification
Before the user is asked, a second agent that did not write the change SHALL try to break the claim and its report (verdict and at least one recorded attempt) is attached verbatim to the proof doc; each finding is fixed with the differential re-run or listed as a limit. Source: docs/process/upstream-pr-standard.md section 5.

#### Scenario: Verifier finds a counterexample
- **WHEN** the verifier reports a case where the claim fails
- **THEN** the change is fixed and the differential re-run, or the case is added to the limits

#### Scenario: No verifier report
- **WHEN** the evidence pack has no attached verifier report
- **THEN** the gate fails and no go is requested

### Requirement: Limits disclosed in the PR body
The PR body SHALL state what the proof does not cover (generator domain, hooked functions, unexercised callers, excluded data cases) in a "Limits" paragraph before the checklist. Source: docs/process/upstream-pr-standard.md section 6.

#### Scenario: Limits paragraph
- **WHEN** the PR body draft is checked
- **THEN** a "Limits" paragraph precedes the checklist

### Requirement: Evidence pack and go/no-go before asking
The coordinator SHALL give the user a one-page evidence pack (`docs/research/pr-<topic>-evidence-pack.md`: claim, proof summary, numbers, commands, residual risks, verifier verdict, `PROOF: complete` or `PROOF: incomplete`, recommendation, push and PR commands not run) and MUST NOT ask for a go while any of the eight checklist gates in docs/process/upstream-pr-standard.md section 7 fails. Source: user rule 2026-10-30.

#### Scenario: All gates pass
- **WHEN** gates 1 to 8 pass
- **THEN** the pack says `PROOF: complete` with a go recommendation and the unrun commands

#### Scenario: One gate fails
- **WHEN** any gate fails
- **THEN** the pack says `PROOF: incomplete` with the reason and recommends no-go or a narrower claim

### Requirement: PR authorship and body format
Upstream PR commits SHALL be authored and committed as the user (noreply identity) with no Co-Authored-By or other Claude trailer and a body of at most 3 lines, and the PR body MUST contain `Agent: Claude Code`. Source: upstream AGENTS.md section 5; docs/process/upstream-pr-standard.md section 8.

#### Scenario: Commit check
- **WHEN** the PR branch commits are listed before the pack goes to the user
- **THEN** author and committer are the user's noreply identity and no commit carries a trailer

#### Scenario: PR body
- **WHEN** the PR body draft is checked
- **THEN** it contains the line `Agent: Claude Code`

### Requirement: Nothing posted without the user's go
No agent SHALL push, post, comment on or label anything on the upstream GitHub project, and each push and each PR needs the user's go on its own. Source: AGENTS.md (upstream rules); `upstream-pr:<topic>` exclusive resource.

#### Scenario: Agent asked to push
- **WHEN** an agent holds a finished PR branch without the user's go
- **THEN** it stops and reports; the commands stay unrun in the pack

### Requirement: PR spacing and bundled follow-ups
Our PRs to the upstream project SHALL be at least 1 hour apart and follow-up commits SHALL be bundled into one push, after `gh pr checks` is green. Source: docs/process/upstream-pr-standard.md section 8.

#### Scenario: Second PR too soon
- **WHEN** less than 1 hour has passed since our last upstream PR
- **THEN** the next PR waits

#### Scenario: Review fixes
- **WHEN** a PR needs three small fixes after review
- **THEN** they go out as one push with the user's go

### Requirement: No game data in an upstream PR
An upstream PR SHALL contain no decompiled Blizzard code and no game data; oracle scripts are attached as evidence commands and a reviewer supplies their own game copy. The repo-wide rule is owned by capability public-split. Source: upstream CONTRIBUTING and AGENTS; `pr_guard` evidences it.

#### Scenario: Oracle needs game files
- **WHEN** a reviewer reproduces the proof
- **THEN** they supply their own game copy and the PR contains none of its data

### Requirement: Every upstream post is logged
Every push or post to the upstream project (branch push, PR open, comment, reply) SHALL be recorded as a dated line `posted: <date> <what> <url or sha> <user go ref>` in that topic's evidence pack (`docs/research/pr-<topic>-evidence-pack.md`); a post made by the coordinator alone needs the log line, not a change card. Source: user go 2026-10-31 (spec-sme review); backfilled in the packs of PRs 134, 135 and 136.

#### Scenario: Review reply posted
- **WHEN** the coordinator replies to a review comment on an open PR
- **THEN** the topic's evidence pack gains a `posted:` line with the date, what was posted, its URL and the user's go reference

#### Scenario: Post without a line
- **WHEN** an upstream push or comment exists but its pack has no `posted:` line
- **THEN** the pack is incomplete and the line is added before the next post
