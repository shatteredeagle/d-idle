// Package d2dmesh is a leaderless node mesh for the d2d test farm: every node holds the whole
// state table (nodes and jobs), tables merge by per-record rules (last-writer-wins for node
// records, fenced leases for jobs) and spread by push-pull gossip over mutual TLS.
// Design and limits: docs/d2d-mesh.md. Prototype: tested on loopback only, never deployed.
package d2dmesh

import (
	"sort"
	"time"
)

// WireVersion is the gossip format. A peer with another value is refused, not merged.
const WireVersion = 1

// Job states. queued -> running -> done | failed; any state -> cancelled.
const (
	JobQueued    = "queued"
	JobRunning   = "running"
	JobDone      = "done"
	JobFailed    = "failed"
	JobCancelled = "cancelled"
)

// NodeRecord is one member as seen by the mesh. The liveness fields are written only by the
// node itself (merge: higher Clock wins, ties by later LastSeen). Ctl is a second
// last-writer-wins register that any node may write (pause, drain), merged on its own clock,
// so an operator's flag is never lost to the owner's next heartbeat.
type NodeRecord struct {
	Name     string      `json:"name"`
	Addr     string      `json:"addr"`
	Labels   []string    `json:"labels,omitempty"`
	Capacity int         `json:"capacity"`
	Running  int         `json:"running"`
	LastSeen int64       `json:"last_seen"` // unix seconds, written by the node itself
	Version  string      `json:"version,omitempty"`
	Clock    uint64      `json:"clock"`
	Gone     bool        `json:"gone,omitempty"` // tombstone: the node left on purpose
	Ctl      NodeControl `json:"ctl"`
}

// NodeControl is the operator-set part of a node record.
type NodeControl struct {
	Paused   bool   `json:"paused,omitempty"`
	Draining bool   `json:"draining,omitempty"`
	Clock    uint64 `json:"clock"`
	By       string `json:"by,omitempty"` // tie-break: lexically smaller writer wins
}

func newerCtl(current, candidate NodeControl) bool {
	if candidate.Clock != current.Clock {
		return candidate.Clock > current.Clock
	}
	return candidate.By < current.By
}

// JobRecord is one job. Owner and Lease are the lease; Fence is a monotonic counter that
// increases on every ownership change (claim, re-queue, cancel), so an old owner can tell
// it lost the job. Merge rule: higher Fence wins; at equal Fence the higher Clock, then
// the lexically smaller Owner (deterministic tie-break for a split-brain double claim).
type JobRecord struct {
	ID          string            `json:"id"`
	Recipe      string            `json:"recipe"`
	Args        map[string]string `json:"args,omitempty"`
	Labels      []string          `json:"labels,omitempty"` // required node labels
	SubmittedBy string            `json:"submitted_by"`
	Submitted   int64             `json:"submitted"` // unix seconds
	State       string            `json:"state"`
	Owner       string            `json:"owner,omitempty"`
	Lease       int64             `json:"lease,omitempty"` // unix seconds, lease expiry
	Fence       uint64            `json:"fence"`
	Attempt     int               `json:"attempt"`
	Clock       uint64            `json:"clock"`
	Result      *JobResult        `json:"result,omitempty"`
}

// JobResult is what a finished attempt leaves in the table; the files stay on the node
// that ran it, under OutDir, keyed by job id and attempt.
type JobResult struct {
	Attempt int    `json:"attempt"`
	Node    string `json:"node"`
	Exit    int    `json:"exit"`
	Summary string `json:"summary,omitempty"`
	OutDir  string `json:"out_dir,omitempty"`
	Pid     int    `json:"pid,omitempty"`
}

// State is the whole table. It is a value type on the wire; Node guards its copy with a mutex.
type State struct {
	Wire  int                    `json:"wire"`
	Nodes map[string]*NodeRecord `json:"nodes"`
	Jobs  map[string]*JobRecord  `json:"jobs"`
}

func NewState() *State {
	return &State{Wire: WireVersion, Nodes: map[string]*NodeRecord{}, Jobs: map[string]*JobRecord{}}
}

func (s *State) Clone() *State {
	c := NewState()
	for k, v := range s.Nodes {
		n := *v
		n.Labels = append([]string(nil), v.Labels...)
		c.Nodes[k] = &n
	}
	for k, v := range s.Jobs {
		c.Jobs[k] = v.clone()
	}
	return c
}

func (j *JobRecord) clone() *JobRecord {
	c := *j
	if j.Args != nil {
		c.Args = make(map[string]string, len(j.Args))
		for k, v := range j.Args {
			c.Args[k] = v
		}
	}
	c.Labels = append([]string(nil), j.Labels...)
	if j.Result != nil {
		r := *j.Result
		c.Result = &r
	}
	return &c
}

// newerNode reports whether candidate should replace current.
func newerNode(current, candidate *NodeRecord) bool {
	if current == nil {
		return true
	}
	if candidate.Clock != current.Clock {
		return candidate.Clock > current.Clock
	}
	return candidate.LastSeen > current.LastSeen
}

// terminal reports whether a job state is final.
func terminal(state string) bool {
	return state == JobDone || state == JobFailed || state == JobCancelled
}

// newerJob reports whether candidate should replace current. The order is lexicographic on
// (fence, terminal, clock, owner), so every node applies the same total order whatever the
// merge order: a cancel issued concurrently with a claim at the same fence always wins.
func newerJob(current, candidate *JobRecord) bool {
	if current == nil {
		return true
	}
	if candidate.Fence != current.Fence {
		return candidate.Fence > current.Fence
	}
	if terminal(candidate.State) != terminal(current.State) {
		return terminal(candidate.State)
	}
	if candidate.Clock != current.Clock {
		return candidate.Clock > current.Clock
	}
	return candidate.Owner < current.Owner
}

// Merge folds other into s and reports whether anything changed. It never downgrades a
// record and ignores a table with another wire version.
func (s *State) Merge(other *State) (changed bool) {
	if other == nil || other.Wire != s.Wire {
		return false
	}
	for k, v := range other.Nodes {
		if v == nil || v.Name != k {
			continue
		}
		cur := s.Nodes[k]
		if newerNode(cur, v) {
			n := *v
			n.Labels = append([]string(nil), v.Labels...)
			if cur != nil && !newerCtl(cur.Ctl, v.Ctl) {
				n.Ctl = cur.Ctl
			}
			s.Nodes[k] = &n
			changed = true
		} else if newerCtl(cur.Ctl, v.Ctl) {
			cur.Ctl = v.Ctl
			changed = true
		}
	}
	for k, v := range other.Jobs {
		if v == nil || v.ID != k {
			continue
		}
		if newerJob(s.Jobs[k], v) {
			s.Jobs[k] = v.clone()
			changed = true
		}
	}
	return changed
}

// Equal reports whether two tables hold the same records (used by convergence tests).
func (s *State) Equal(o *State) bool {
	if len(s.Nodes) != len(o.Nodes) || len(s.Jobs) != len(o.Jobs) {
		return false
	}
	for k, a := range s.Nodes {
		b, ok := o.Nodes[k]
		if !ok || a.Clock != b.Clock || a.LastSeen != b.LastSeen || a.Ctl != b.Ctl || a.Gone != b.Gone {
			return false
		}
	}
	for k, a := range s.Jobs {
		b, ok := o.Jobs[k]
		if !ok || a.Fence != b.Fence || a.Clock != b.Clock || a.State != b.State || a.Owner != b.Owner {
			return false
		}
	}
	return true
}

// JobsSorted returns the jobs oldest first (then by id) so every node scans the queue in the same order.
func (s *State) JobsSorted() []*JobRecord {
	out := make([]*JobRecord, 0, len(s.Jobs))
	for _, j := range s.Jobs {
		out = append(out, j)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Submitted != out[b].Submitted {
			return out[a].Submitted < out[b].Submitted
		}
		return out[a].ID < out[b].ID
	})
	return out
}

// NodesSorted returns live members by name.
func (s *State) NodesSorted() []*NodeRecord {
	out := make([]*NodeRecord, 0, len(s.Nodes))
	for _, n := range s.Nodes {
		if !n.Gone {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Name < out[b].Name })
	return out
}

// hasLabels reports whether node carries every label the job requires.
func hasLabels(node, want []string) bool {
	for _, w := range want {
		found := false
		for _, l := range node {
			if l == w {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Lease helpers. All take the table lock from the caller.

// claim takes a queued job for node at now with ttl; the caller checked admission.
func (s *State) claim(j *JobRecord, node string, now time.Time, ttl time.Duration, clock uint64) {
	j.Fence++
	j.Attempt++
	j.Owner = node
	j.Lease = now.Add(ttl).Unix()
	j.State = JobRunning
	j.Clock = clock
}

// expireLeases re-queues every running job whose lease has passed; returns the ids.
func (s *State) expireLeases(now time.Time, clock uint64) (ids []string) {
	for _, j := range s.Jobs {
		if j.State == JobRunning && j.Lease < now.Unix() {
			j.Fence++
			j.Owner = ""
			j.Lease = 0
			j.State = JobQueued
			j.Clock = clock
			ids = append(ids, j.ID)
		}
	}
	sort.Strings(ids)
	return ids
}
