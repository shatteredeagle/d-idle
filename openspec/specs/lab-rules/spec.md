# Lab rules specification

## Purpose
The rules for running d2d, the private lockstep fork and live games without agents colliding, leaking secrets or breaking the GPL boundary. Source: AGENTS.md (games, secrets, honesty), docs/decisions.md 2026-10-30, docs/d2d-soak.md section 1, docs/d2d-fork.md, docs/d2d-container/README.md (GPL notes). The d2d lab doc (docs/d2d-lab.md) is owned by the d2dlab change.

## Requirements

### Requirement: One d2d at a time
The d2d lab lock SHALL be a pool of 4 slots (slot 0 is the legacy `/tmp/d2d-lab.lock`); a run holds exactly one slot, only while a d2d runs, taken through `tools/d2dslot.sh run`; legacy single-lock holders occupy slot 0. Hidden d2d only; original-game rules are unchanged and the `pgrep -x d2d` refusal no longer applies. Source: user decision 2026-10-30; V live (tools/d2dslot-test.sh).

#### Scenario: Lock held
- **WHEN** another agent holds `/tmp/d2d-lab.lock` (slot 0)
- **THEN** the run takes the next free slot, or exits 75 when none is free within its wait

#### Scenario: Four runs
- **WHEN** four agents call `tools/d2dslot.sh run` at once
- **THEN** each gets a different slot, its own socket under `~/d2d-lab/slots/<n>/` and its own ports

#### Scenario: All slots held
- **WHEN** a fifth run asks for a slot and none frees within its wait time
- **THEN** it exits 75 and starts no d2d

#### Scenario: Legacy holder
- **WHEN** a script still runs `flock /tmp/d2d-lab.lock`
- **THEN** it occupies slot 0 and the pool hands out only slots 1 to 3

### Requirement: Private sockets
Each bot or tool MUST drive its own d2d on its own socket (`<run dir>/d2d-<instance>.sock`), never `~/d2d-lab/fork/d2d.sock` and never a d2d it did not start (no authentication, one client). Source: docs/d2d-wire.md socket safety; docs/d2d-spike.md (control socket owner-only).

#### Scenario: Shared socket path
- **WHEN** a bot is configured with the shared fork socket path
- **THEN** it is refused unless `--d2d-attach` is given

### Requirement: Lab binary protection
The lab binary (`~/d2d-lab/fork/bin/d2d.lab`) SHALL be read-only and never overwritten; new builds get a new name with sha256 and provenance next to them. Source: docs/d2d-lab.md (branch d2dlab, V read at 91606cc).

#### Scenario: Rebuild
- **WHEN** the fork is rebuilt
- **THEN** the old binary file is unchanged and the new one has its own provenance file

### Requirement: Fork on the forge the same day
Every patch to the private d2d fork or lab patch set MUST be committed and pushed to the forge `<owner>/d2d-fork` the same day it is made, and a doc may rely only on a commit hash reachable from a remote. Source: AGENTS.md fork-push rule (coordinator rule 2026-10-30 after patches lived only on an unpushed fork branch; U whether the user restated it).

#### Scenario: New lab patch
- **WHEN** an agent adds a verb to the fork
- **THEN** the commit is on `<owner>/d2d-fork` before the agent reports

### Requirement: GPL separation
d2d code and text MUST NOT be copied into this repository (summaries with file-path citations only); the bot talks to d2d only over its socket as a separate program; the fork stays private and nothing ships or is distributed until a GPL reviewer is named by the user. Source: docs/decisions.md 2026-10-30 rule 3 and user answers.

#### Scenario: Patch file
- **WHEN** a change produces a C++ patch
- **THEN** it lives only in the private fork, not in this repo

### Requirement: No GitHub writes without the user's go
Nothing SHALL be posted, pushed or opened on GitHub (issues, PRs, comments upstream) without the user's explicit go in that conversation; drafts stay in the repo for review. Source: docs/d2d-fork.md (draft upstream issue "for the user to review, nothing was posted").

#### Scenario: Upstream PR ready
- **WHEN** an upstream PR branch is finished
- **THEN** the agent reports it and waits for the user's go

### Requirement: No secrets
No CD key, password, token or credential SHALL be written to the repo, a log, a card or a report; `worker-sudo` is used only through `tools/live.sh` for the ptrace setcap. Source: AGENTS.md.

#### Scenario: Debug output
- **WHEN** a script prints its environment
- **THEN** credentials are redacted or not printed at all

### Requirement: Safety limits for lab runs
Lab runs SHALL be niced (10), refuse above load 45 or under 15 GB free disk, and leave no d2d process behind. New live work MUST also pass the floating capacity rule: `tools/capacity.sh --add <cores>` must not print `NO` (predicted idle+nice >= 25 %, iowait < 40 %, 5 minute average); there is no fixed Game.exe or d2d count, and offline and bug-fix work is not capped. Source: docs/d2d-soak.md section 1 (V); docs/process/worker-capacity.md.

#### Scenario: Run ends
- **WHEN** a seed finishes or its wall limit is hit
- **THEN** d2d is quit, the process group is killed and `pgrep -x d2d` is empty

#### Scenario: Quiet machine, third instance
- **WHEN** two Game.exe are running, the 5 minute average idle+nice is 60 % and a d2d run costing 0.9 cores is requested on 16 cores
- **THEN** `tools/capacity.sh --add 0.9` prints OK and the run starts without asking anyone

#### Scenario: Busy machine
- **WHEN** the predicted idle+nice after the added cost is under 25 %
- **THEN** the verdict is NO and the live work waits or is made smaller

### Requirement: Big run warning
A BIG RUN (more than one new game or d2d instance at once, over 2 cores for over 30 minutes, or any multi-hour soak) SHALL be announced by the coordinator in chat and MUST start after 5 minutes unless the user replied hold or no; silence means go. The dispatch agent re-checks `tools/capacity.sh` before starting and aborts only on NO. Other work SHALL NOT wait for the user. Source: user decision 2026-10-30.

#### Scenario: Silence
- **WHEN** the warning was sent and 5 minutes pass without a reply
- **THEN** the run starts, after a final capacity check that is not NO

#### Scenario: Hold
- **WHEN** the user replies "hold" within 5 minutes
- **THEN** the run does not start until the user says go

#### Scenario: Small run
- **WHEN** a single d2d run of 0.9 cores for 20 minutes is requested
- **THEN** it is not a big run, no warning is sent and only the capacity check applies
