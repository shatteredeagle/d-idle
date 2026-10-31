# process-hygiene Specification

## Purpose
How long-running processes are started, registered, reaped and reported so none is left behind: registry wrapper, group kill and TTL, orphan listing, user approval for stops, bounded waiters and the dispatcher janitor duty.

## Requirements

### Requirement: Long processes start through the registry wrapper
Every process an agent starts that runs longer than its starting command (bots, d2d children, slot waiters, wait loops) SHALL be started through the registry wrapper, which records its agent, pid, group and TTL. Source: AGENTS.md cleanup section; tools/agentrun.sh, tools/d2dslot.sh; change prochygiene.

#### Scenario: Registered start
- **WHEN** an agent starts a bot through agentrun.sh
- **THEN** `tools/agentreap.sh --list` shows its agent, pid, pgid, age, ttl and `alive`

#### Scenario: Unregistered start
- **WHEN** a long process is started without the wrapper
- **THEN** it appears in `tools/agentreap.sh --orphans` once its parent is gone

### Requirement: Group kill on exit and TTL
The wrapper SHALL run the command in its own process group and kill the whole group (TERM, then KILL after 5 s) when the command exits, a signal arrives or the TTL expires. Source: tools/agentrun.sh; tools/test/agentrun_test.sh.

#### Scenario: TTL expiry
- **WHEN** a registered command outlives its `--ttl`
- **THEN** the wrapper kills its group and no child of it is left

#### Scenario: Command exits
- **WHEN** the wrapped command ends and left a child behind
- **THEN** the group kill stops the child and the record is removed

### Requirement: Reap by record with pid and starttime check
`tools/agentreap.sh <agent>` SHALL signal only a registered group whose live pid still matches the recorded starttime and group id; a record whose process is gone or whose pid was reused MUST be removed without signalling anything. Source: tools/agentreap.sh; tools/test/agentrun_test.sh.

#### Scenario: Pid reused
- **WHEN** a record's pid now belongs to a different process
- **THEN** reap reports "stale record removed" and sends no signal

#### Scenario: Reap an agent
- **WHEN** `tools/agentreap.sh <agent>` runs for an agent with two live groups
- **THEN** both groups are stopped and `--list` shows none for that agent

### Requirement: Orphans are listed, never killed
`tools/agentreap.sh --orphans` SHALL report unregistered agent-looking processes read-only and MUST NOT signal any of them. Source: tools/agentreap.sh; AGENTS.md cleanup section.

#### Scenario: Orphan found
- **WHEN** an unregistered agent-looking process with parent 1 exists
- **THEN** `--orphans` prints it and signals nothing

#### Scenario: Orphan still alive after listing
- **WHEN** `--orphans` has printed a process
- **THEN** the same pid is still running afterwards

### Requirement: Orphan stops need the user's approval
Only the coordinator or the user SHALL stop an orphan, by pid, and each stop needs the user's approval for that pid. Source: AGENTS.md cleanup section; user decision 2026-10-30.

#### Scenario: Stop request
- **WHEN** a coordinator wants an orphan stopped
- **THEN** the needs-you list shows the pid and the stop happens only after the user approves it

### Requirement: No pattern kills
Agents SHALL NOT use `pkill`, `killall` or `kill` on a pid they did not start. Source: AGENTS.md cleanup section.

#### Scenario: Leftovers of mine
- **WHEN** an agent's registered process is still running at the end
- **THEN** `tools/agentreap.sh <agent>` stops it and `--list` no longer shows it

#### Scenario: Pattern kill in a script
- **WHEN** a committed script contains `pkill` or `killall`
- **THEN** review rejects it

### Requirement: Waiters are bounded
A waiter loop (`until ... sleep`) SHALL have a deadline or run under `agentrun.sh --ttl`. Source: AGENTS.md cleanup section.

#### Scenario: Waiter needed
- **WHEN** an agent waits for a condition in a loop
- **THEN** the loop has a deadline or its registry record shows a TTL

### Requirement: Agents confirm cleanup in the report
Before the final report an agent SHALL run `tools/agentreap.sh <agent>` and `tools/agentreap.sh --list` on the worker host and state "cleanup: none left" or what it stopped. Source: AGENTS.md cleanup section.

#### Scenario: Clean finish
- **WHEN** an agent finishes and `--list` shows no record of its name
- **THEN** its report says "cleanup: none left"

#### Scenario: Leftover found
- **WHEN** `--list` still shows a live record of the agent
- **THEN** it reaps it and reports what was stopped before finishing

### Requirement: Dispatcher janitor duty
The dispatcher SHALL run `tools/agentreap.sh --list` and `--orphans` each tick (read-only), stop expired registered groups with `--all-expired`, and report orphans to the user instead of acting on them. Source: change prochygiene (dispatcher janitor); docs/process/dispatcher-brief.md.

#### Scenario: Expired group
- **WHEN** a registered group is past its TTL at a tick
- **THEN** `--all-expired` stops it and the tick note lists it

#### Scenario: Orphan at a tick
- **WHEN** `--orphans` lists a process
- **THEN** the dispatcher puts the pid on the user's needs-you list and does not kill it
