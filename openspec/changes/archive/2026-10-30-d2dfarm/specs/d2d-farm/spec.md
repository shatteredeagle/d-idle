## ADDED Requirements

### Requirement: Node registration and labels
The farm SHALL treat every machine that runs hidden d2d jobs as a node described by one registry entry: a role-based name, a type (`worker-slots` or `wsl`), a transport target, a maximum number of concurrent jobs, allowed hours, and optional labels. The registry MUST be read from the environment or `local.env` (inline list or a hosts file), with a committed example that uses only placeholder names. A node's label set MUST include its name, its type and its configured labels.

#### Scenario: Registry from the environment
- **WHEN** `D2_FARM_HOSTS` is set
- **THEN** `d2dfarm nodes` lists exactly those nodes with their labels, and the committed example file is not read

#### Scenario: Public repo hygiene
- **WHEN** the repository is scanned for hostnames, addresses or personal names
- **THEN** no farm file contains a real value; real values live only in the environment or the gitignored `local.env`

### Requirement: Capacity probe and admission
Before a job is assigned, the dispatcher SHALL probe each candidate node and treat the verdict as OK, WARN or NO. An unreachable node MUST be NO. A job MUST NOT be started on a NO node, MAY be started on a WARN node only when no OK node qualifies, and MUST NOT exceed the node's maximum concurrent jobs. The probe for an `worker-slots` node SHALL be the host capacity check; the probe for a `wsl` node SHALL consider CPU load, free memory and interactive sessions.

#### Scenario: Node at its job limit
- **WHEN** a node already runs as many jobs as its maximum
- **THEN** a new matching job stays queued and no probe result can override that

#### Scenario: Unreachable node
- **WHEN** the transport cannot reach a node
- **THEN** the probe reports NO with the reason and the job goes to another node or stays queued

### Requirement: Pause and resume
The farm SHALL let an operator pause and resume a node by name; the state MUST persist across invocations. A paused node MUST NOT receive new jobs, including jobs that name it as host, and its running jobs MUST continue and remain collectable.

#### Scenario: Paused node is skipped
- **WHEN** a node is paused and a job matches only that node's labels
- **THEN** the job stays queued until the node is resumed

### Requirement: Queue and dispatch by label
Jobs SHALL be recipes (soak, replay, record, cmd) with seeds, hours and required labels, kept in a persistent local queue. The dispatcher MUST assign a queued job only to a node whose labels include every required label, that is not paused, inside its allowed hours, below its job limit and not probing NO. Among eligible nodes it SHALL prefer OK over WARN, then the lower priority number, then fewer running jobs. A replicated soak MAY be split by seeds into several jobs of one batch.

#### Scenario: Spill to the second node
- **WHEN** the preferred node is full or probes NO and another eligible node exists
- **THEN** the job starts on the other node

#### Scenario: Seed split
- **WHEN** a soak over seeds 1-5 is submitted with a split of 2
- **THEN** two jobs of one batch are queued with seeds 1,3,5 and 2,4

### Requirement: Result collection and one verdict
The farm SHALL copy a finished job's run directory to one local results directory keyed by batch, then remove it from the node. A batch verdict MUST be produced by one `d2dsoak compare -gate v2` over every collected seed directory of the batch, so a soak spread over several nodes yields one verdict.

#### Scenario: Batch over two nodes
- **WHEN** both jobs of a batch are collected
- **THEN** `d2dfarm verdict BATCH` passes all seed directories of both jobs to a single compare run

### Requirement: Idle-only nodes
A node that is another person's machine SHALL be idle-only: it MUST probe NO while an interactive session is active, MUST honor its allowed-hours window, MUST run one job at a time at low priority, and a WSL node SHALL be shut down when its queue is empty and its results are collected.

#### Scenario: User at the keyboard
- **WHEN** the probe finds an interactive session on an idle-only node
- **THEN** no job is started there

#### Scenario: End of queue
- **WHEN** the last job of a WSL node is collected
- **THEN** the distro is shut down and nothing of the farm remains running there

### Requirement: Safety
Jobs SHALL run only the hidden d2d lab binary through the lab recipes; the farm MUST NOT start, stop or touch an original-game instance and MUST NOT carry the private fork source. A job is stopped only by its recorded process group, after checking that the process still belongs to the job; the farm MUST NOT kill by name pattern.

#### Scenario: Stop spares other processes
- **WHEN** a job is stopped while an unrelated process runs on the node
- **THEN** only the job's recorded process group receives the signal

### Requirement: Cross-host determinism as qualification
A node MUST pass a determinism qualification before it takes fidelity work: its d2d binary has the same sha256 as the reference node's, and a recorded scripted scenario replays there with every state-hash checkpoint matching. A mismatch SHALL be reported with the first divergent tick.

#### Scenario: Same build, same checkpoints
- **WHEN** one replay file is played on two nodes with the same binary sha256
- **THEN** both report all checkpoints matched, and the nodes may run the same soak interchangeably
