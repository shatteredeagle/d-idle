## ADDED Requirements

### Requirement: Join with a token
A node SHALL join the mesh by presenting the shared join token to at least one seed node over TLS; a peer that cannot prove possession of the token MUST be refused at the TLS handshake and MUST NOT learn any mesh state. Source: docs/d2d-mesh.md section 2 and 8.

#### Scenario: Correct token
- **WHEN** a new node starts with the farm's token file and one reachable seed address
- **THEN** within two anti-entropy rounds every member lists the new node with its address, labels and capacity

#### Scenario: Wrong token
- **WHEN** a node starts with a different token and dials a member
- **THEN** the TLS handshake fails, the member logs nothing about the state, and the node table is unchanged

### Requirement: Membership convergence
Every member SHALL hold a copy of the whole membership table and the tables of all connected members MUST converge to the same content within 3 anti-entropy rounds after the last change, with no leader. Source: docs/d2d-mesh.md section 2.

#### Scenario: Three nodes, one seed
- **WHEN** nodes A, B and C start and B and C only know A as a seed
- **THEN** after 3 rounds A, B and C each list exactly A, B and C

### Requirement: Node record fields
Each node record SHALL carry name, address, labels, capacity (max concurrent jobs), paused, draining, last-seen time, software version and a Lamport clock; a field change MUST be visible on every connected member within 3 rounds. Source: docs/d2d-mesh.md section 2.

#### Scenario: Pause from another node
- **WHEN** an operator pauses node B through node A's API
- **THEN** B stops claiming new jobs within 3 rounds and every member shows B as paused

### Requirement: Any-node job management
Every member SHALL serve the same job verbs on its one listening port: list, submit, cancel, and pause, resume and drain of any node; the result of a verb MUST be the same whichever member received it. Source: docs/d2d-mesh.md section 4.

#### Scenario: Submit on A, claim on C
- **WHEN** a job is submitted to node A and only node C has free capacity
- **THEN** C claims and runs it and A lists it as running on C

### Requirement: Job leases
A running job SHALL have exactly one owner and a lease expiry; the owner MUST extend the lease by heartbeat before it expires; when a lease has expired any member MAY re-queue the job with a higher fencing counter and a new attempt number. Source: docs/d2d-mesh.md section 3.

#### Scenario: Owner dies mid-job
- **WHEN** the node running a job stops heartbeating for longer than the lease length
- **THEN** another member re-queues the job and a node with capacity runs attempt 2, keyed by job id and attempt

#### Scenario: Stale owner returns
- **WHEN** the original owner comes back after its lease was taken over
- **THEN** it sees a higher fence than its own, stops its run and does not write a result for the old attempt

### Requirement: Partition tolerance
A member that loses contact with the mesh SHALL keep serving its API and running jobs it already owns; when contact returns the tables MUST merge without operator action and a job that was claimed on both sides MUST end with exactly one winner by fence order. Source: docs/d2d-mesh.md section 3 and 10.

#### Scenario: Node drops and returns
- **WHEN** node B is unreachable for several rounds and then reachable again
- **THEN** B's record and job updates made meanwhile on either side are present on every member within 3 rounds of the heal

### Requirement: Authentication and encryption
All mesh traffic SHALL be TLS with mutual authentication; node identity MUST be a certificate signed by a CA derived from the join token (or a pinned certificate), and plaintext HTTP MUST NOT be served on the mesh port. Source: docs/d2d-mesh.md section 8.

#### Scenario: Plaintext request
- **WHEN** a client sends a plain HTTP request to the mesh port
- **THEN** the connection is closed without an application response

### Requirement: One listening port per node
A node SHALL serve gossip, the JSON API and the status page on one TCP port, loopback by default; any other address MUST be an explicit flag. Source: docs/d2d-mesh.md section 5.

#### Scenario: Default listen address
- **WHEN** the node is started without a listen flag
- **THEN** it listens on a loopback address only

### Requirement: Admission rules
A node SHALL claim a job only when it is not paused and not draining, its running count is below its capacity, its host guard (capacity script or idle check) says yes, and the job's required labels are a subset of the node's labels. Source: docs/d2d-mesh.md section 6.

#### Scenario: Host guard says NO
- **WHEN** the host guard returns NO on node B and a job is queued
- **THEN** B does not claim it and a node whose guard says yes does

#### Scenario: Draining node
- **WHEN** node B is set to draining with one job running
- **THEN** B finishes that job, claims no other, and is listed as draining

### Requirement: Safety of what a node runs
A node SHALL run only recipes from a fixed allow-list with validated arguments (never arbitrary shell), start only hidden d2d engines (no original-game instances), record the pid of each run, and stop a run only by its recorded pid. Source: docs/d2d-mesh.md section 8, AGENTS.md live-game rules.

#### Scenario: Unknown recipe
- **WHEN** a job with a recipe not on the allow-list is submitted
- **THEN** every member rejects it with an error and nothing is queued

#### Scenario: Bad argument
- **WHEN** a known recipe is submitted with an argument outside its validated range
- **THEN** the submission is rejected and nothing is queued

### Requirement: Result collection
A finished job's result (exit status, summary, output location on the running node) SHALL be in the job table on every member; the output files MUST be collectable from the running node to the submitting node or any chosen node by job id and attempt. Source: docs/d2d-mesh.md section 4.

#### Scenario: Collect from a third node
- **WHEN** a job submitted on A ran on C and an operator asks B to collect it
- **THEN** B fetches the attempt's output from C and stores it under the job id and attempt

### Requirement: Public-repo hygiene
Committed mesh code, docs and examples SHALL contain no real hostname, address or person; seeds, tokens and names MUST come from flags, the environment or an ignored local env file. Source: docs/process/public-split.md.

#### Scenario: Example configuration
- **WHEN** a committed example names a seed
- **THEN** it uses a reserved example name or a loopback address

### Requirement: Version skew tolerance
A node SHALL accept gossip from a peer of a different software version as long as the wire version matches, ignore unknown fields, and MUST refuse (and log) a peer with a different wire version instead of merging its table. Source: docs/d2d-mesh.md section 9.

#### Scenario: Unknown field
- **WHEN** a newer peer sends a record with a field this node does not know
- **THEN** the known fields merge normally and the unknown one is ignored
