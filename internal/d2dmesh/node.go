package d2dmesh

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"
)

// maxWire bounds one gossip body; a table bigger than this is refused (DecodeState).
const maxWire = 4 << 20

// Runner executes one claimed attempt. It must return when ctx is cancelled (lease lost or
// node stopping) and must never launch anything but a hidden d2d recipe.
type Runner interface {
	Run(ctx context.Context, job *JobRecord, attempt int) (JobResult, error)
}

// Config of one node. Zero values are filled by Start.
type Config struct {
	Name     string
	Listen   string // "127.0.0.1:0" in tests; loopback by default
	Seeds    []string
	Token    string
	Labels   []string
	Capacity int
	Version  string
	Interval time.Duration // anti-entropy period
	LeaseTTL time.Duration
	Fanout   int                  // peers contacted per round
	Guard    func() bool          // host admission guard; nil = always yes
	Runner   Runner               // nil = DryRunner
	Recipes  map[string]Recipe    // nil = DefaultRecipes
	Logf     func(string, ...any) // nil = log.Printf
	Now      func() time.Time     // nil = time.Now
}

// Node is one mesh member: table, gossip client and server, worker loop.
type Node struct {
	cfg    Config
	mu     sync.Mutex
	st     *State
	clock  uint64
	peers  map[string]struct{} // learned addresses (seeds and node records), never self
	cut    bool                // test hook: simulate a partition (refuse in, skip out)
	labels []string            // current labels (cfg.Labels at start; SetLabels changes them)
	runs   map[string]*run
	ln     net.Listener
	srv    *http.Server
	client *http.Client
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	loops  sync.WaitGroup
}

type run struct {
	fence  uint64
	cancel context.CancelFunc
}

// New builds a node; nothing listens until Start.
func New(cfg Config) (*Node, error) {
	if cfg.Name == "" {
		return nil, errors.New("d2dmesh: node name required")
	}
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:0"
	}
	if cfg.Interval == 0 {
		cfg.Interval = 3 * time.Second
	}
	if cfg.LeaseTTL == 0 {
		cfg.LeaseTTL = 30 * time.Second
	}
	if cfg.Fanout == 0 {
		cfg.Fanout = 2
	}
	if cfg.Capacity == 0 {
		cfg.Capacity = 1
	}
	if cfg.Runner == nil {
		cfg.Runner = DryRunner{}
	}
	if cfg.Recipes == nil {
		cfg.Recipes = DefaultRecipes
	}
	if cfg.Logf == nil {
		cfg.Logf = func(f string, a ...any) { log.Printf("d2dnode "+cfg.Name+": "+f, a...) }
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	n := &Node{cfg: cfg, st: NewState(), peers: map[string]struct{}{}, runs: map[string]*run{}, labels: append([]string(nil), cfg.Labels...)}
	for _, s := range cfg.Seeds {
		if s != "" {
			n.peers[s] = struct{}{}
		}
	}
	return n, nil
}

// Start listens, announces the node and starts the gossip and worker loops.
func (n *Node) Start() error {
	srvTLS, cliTLS, err := TLSConfigs(n.cfg.Token, n.cfg.Name)
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", n.cfg.Listen)
	if err != nil {
		return err
	}
	n.ln = ln
	ln = tlsOnly{ln}
	n.client = &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: cliTLS, MaxIdleConnsPerHost: 4}}
	n.srv = &http.Server{Handler: n.handler(), TLSConfig: srvTLS, ReadHeaderTimeout: 5 * time.Second, ErrorLog: log.New(io.Discard, "", 0)}
	n.ctx, n.cancel = context.WithCancel(context.Background())
	n.mu.Lock()
	n.touchSelfLocked()
	delete(n.peers, n.ln.Addr().String())
	n.mu.Unlock()
	n.loops.Add(3)
	go func() { defer n.loops.Done(); _ = n.srv.ServeTLS(ln, "", "") }()
	go func() { defer n.loops.Done(); n.gossipLoop() }()
	go func() { defer n.loops.Done(); n.workLoop() }()
	return nil
}

// Stop cancels running jobs and closes the listener without a tombstone (a crash, for the
// other members: the leases expire). Use Leave for a clean exit.
func (n *Node) Stop() {
	if n.cancel == nil {
		return
	}
	n.cancel()
	ctx, done := context.WithTimeout(context.Background(), 2*time.Second)
	defer done()
	_ = n.srv.Shutdown(ctx)
	n.loops.Wait()
	n.wg.Wait()
}

// Leave writes the tombstone, pushes it once and stops.
func (n *Node) Leave() {
	n.mu.Lock()
	self := n.st.Nodes[n.cfg.Name]
	if self != nil {
		self.Gone = true
		self.Clock = n.tickLocked()
	}
	peers := n.peerListLocked()
	n.mu.Unlock()
	for _, p := range peers {
		_ = n.exchange(p)
	}
	n.Stop()
}

// Addr is the bound address (useful with Listen "127.0.0.1:0").
func (n *Node) Addr() string { return n.ln.Addr().String() }

// Name of this node.
func (n *Node) Name() string { return n.cfg.Name }

// SetLabels changes the node's labels; the next round announces them.
func (n *Node) SetLabels(labels []string) {
	n.mu.Lock()
	n.labels = append([]string(nil), labels...)
	n.mu.Unlock()
}

// RunningJobs is the number of attempts this node is executing right now.
func (n *Node) RunningJobs() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.runs)
}

// Snapshot returns a copy of the table.
func (n *Node) Snapshot() *State {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.st.Clone()
}

// SetPartitioned simulates a network cut for tests: incoming pushes are refused and no
// outgoing exchange is made while cut is true.
func (n *Node) SetPartitioned(cut bool) {
	n.mu.Lock()
	n.cut = cut
	n.mu.Unlock()
}

func (n *Node) tickLocked() uint64 { n.clock++; return n.clock }

func (n *Node) touchSelfLocked() {
	self := n.st.Nodes[n.cfg.Name]
	if self == nil {
		self = &NodeRecord{Name: n.cfg.Name}
		n.st.Nodes[n.cfg.Name] = self
	}
	if self.Gone {
		return
	}
	self.Addr = n.ln.Addr().String()
	self.Labels = append([]string(nil), n.labels...)
	self.Capacity = n.cfg.Capacity
	self.Running = len(n.runs)
	self.Version = n.cfg.Version
	self.LastSeen = n.cfg.Now().Unix()
	self.Clock = n.tickLocked()
}

func (n *Node) peerListLocked() []string {
	for _, rec := range n.st.Nodes {
		if rec.Name != n.cfg.Name && rec.Addr != "" && !rec.Gone {
			n.peers[rec.Addr] = struct{}{}
		}
	}
	out := make([]string, 0, len(n.peers))
	for p := range n.peers {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// Round does one anti-entropy round: refresh self, exchange with up to Fanout random peers.
func (n *Node) Round() {
	n.mu.Lock()
	n.touchSelfLocked()
	ids := n.st.expireLeases(n.cfg.Now(), n.clock+1)
	if len(ids) > 0 {
		n.tickLocked()
		n.cfg.Logf("lease expired, re-queued %v", ids)
	}
	cut := n.cut
	peers := n.peerListLocked()
	n.mu.Unlock()
	if cut {
		return
	}
	for _, i := range pick(len(peers), n.cfg.Fanout) {
		if err := n.exchange(peers[i]); err != nil {
			n.cfg.Logf("exchange %s: %v", peers[i], err)
		}
	}
}

func pick(n, k int) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	for i := 0; i < n && i < k; i++ {
		r, _ := rand.Int(rand.Reader, big.NewInt(int64(n-i)))
		j := i + int(r.Int64())
		idx[i], idx[j] = idx[j], idx[i]
	}
	if k < n {
		idx = idx[:k]
	}
	return idx
}

func (n *Node) gossipLoop() {
	t := time.NewTicker(n.cfg.Interval)
	defer t.Stop()
	n.Round()
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-t.C:
			n.Round()
		}
	}
}

// exchange is one push-pull: send our table, merge the reply.
func (n *Node) exchange(addr string) error {
	body, err := json.Marshal(n.Snapshot())
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(n.ctx, http.MethodPost, "https://"+addr+"/mesh/v1/push", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer answered %s", resp.Status)
	}
	theirs, err := DecodeState(io.LimitReader(resp.Body, maxWire))
	if err != nil {
		return err
	}
	n.mergeIn(theirs)
	return nil
}

func (n *Node) mergeIn(theirs *State) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, r := range theirs.Nodes {
		if r.Clock > n.clock {
			n.clock = r.Clock
		}
	}
	for _, j := range theirs.Jobs {
		if j.Clock > n.clock {
			n.clock = j.Clock
		}
	}
	changed := n.st.Merge(theirs)
	// A merged record may have taken a job away from us: cancel that run.
	for id, r := range n.runs {
		if j := n.st.Jobs[id]; j == nil || j.Owner != n.cfg.Name || j.Fence != r.fence {
			n.cfg.Logf("job %s: lease lost (fence %d), cancelling run", id, r.fence)
			r.cancel()
		}
	}
	return changed
}

// DecodeState parses one gossip body and checks wire version and record keys.
func DecodeState(r io.Reader) (*State, error) {
	var s State
	dec := json.NewDecoder(io.LimitReader(r, maxWire))
	if err := dec.Decode(&s); err != nil {
		return nil, err
	}
	if s.Wire != WireVersion {
		return nil, fmt.Errorf("wire version %d, want %d", s.Wire, WireVersion)
	}
	if s.Nodes == nil {
		s.Nodes = map[string]*NodeRecord{}
	}
	if s.Jobs == nil {
		s.Jobs = map[string]*JobRecord{}
	}
	for k, v := range s.Nodes {
		if v == nil || v.Name != k || !ident.MatchString(k) {
			return nil, fmt.Errorf("bad node record %q", k)
		}
	}
	for k, v := range s.Jobs {
		if v == nil || v.ID != k || !ident.MatchString(k) {
			return nil, fmt.Errorf("bad job record %q", k)
		}
	}
	return &s, nil
}

// Submit validates and queues a job; it is visible mesh-wide after gossip.
func (n *Node) Submit(recipe string, args map[string]string, labels []string) (string, error) {
	if err := Validate(n.cfg.Recipes, recipe, args); err != nil {
		return "", err
	}
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	id := fmt.Sprintf("%s-%d-%s", recipe, n.cfg.Now().Unix(), hex.EncodeToString(b[:4]))
	n.mu.Lock()
	n.st.Jobs[id] = &JobRecord{ID: id, Recipe: recipe, Args: args, Labels: labels, SubmittedBy: n.cfg.Name,
		Submitted: n.cfg.Now().Unix(), State: JobQueued, Clock: n.tickLocked()}
	n.mu.Unlock()
	return id, nil
}

// Cancel takes the job away from its owner (fence+1) and marks it cancelled.
func (n *Node) Cancel(id string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	j := n.st.Jobs[id]
	if j == nil {
		return fmt.Errorf("no job %s", id)
	}
	if terminal(j.State) {
		return nil
	}
	j.Fence++
	j.Owner = ""
	j.Lease = 0
	j.State = JobCancelled
	j.Clock = n.tickLocked()
	if r := n.runs[id]; r != nil {
		r.cancel()
	}
	return nil
}

// Control sets the operator flags of any node (its own LWW register).
func (n *Node) Control(name string, paused, draining *bool) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	rec := n.st.Nodes[name]
	if rec == nil {
		return fmt.Errorf("no node %s", name)
	}
	if paused != nil {
		rec.Ctl.Paused = *paused
	}
	if draining != nil {
		rec.Ctl.Draining = *draining
	}
	rec.Ctl.Clock = n.tickLocked()
	rec.Ctl.By = n.cfg.Name
	return nil
}

// Worker loop: claim admissible jobs, run them, heartbeat, write results under the fence.

func (n *Node) workLoop() {
	t := time.NewTicker(n.cfg.Interval / 2)
	defer t.Stop()
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-t.C:
			n.claimJobs()
		}
	}
}

func (n *Node) admissible(self *NodeRecord) bool {
	if self == nil || self.Ctl.Paused || self.Ctl.Draining || len(n.runs) >= n.cfg.Capacity {
		return false
	}
	return true
}

func (n *Node) claimJobs() {
	if n.cfg.Guard != nil && !n.cfg.Guard() {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, j := range n.st.JobsSorted() {
		if !n.admissible(n.st.Nodes[n.cfg.Name]) {
			return
		}
		if j.State != JobQueued || !hasLabels(n.labels, j.Labels) {
			continue
		}
		n.st.claim(j, n.cfg.Name, n.cfg.Now(), n.cfg.LeaseTTL, n.tickLocked())
		ctx, cancel := context.WithCancel(n.ctx)
		n.runs[j.ID] = &run{fence: j.Fence, cancel: cancel}
		n.wg.Add(1)
		go n.execute(ctx, j.clone())
	}
}

func (n *Node) execute(ctx context.Context, j *JobRecord) {
	defer n.wg.Done()
	fence, attempt := j.Fence, j.Attempt
	n.cfg.Logf("job %s: claimed, attempt %d, fence %d", j.ID, attempt, fence)
	stopHB := make(chan struct{})
	go n.heartbeat(j.ID, fence, stopHB)
	res, err := n.cfg.Runner.Run(ctx, j, attempt)
	close(stopHB)

	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.runs, j.ID)
	cur := n.st.Jobs[j.ID]
	if cur == nil || cur.Owner != n.cfg.Name || cur.Fence != fence {
		n.cfg.Logf("job %s: attempt %d finished after the lease was lost; result dropped", j.ID, attempt)
		return
	}
	res.Attempt, res.Node = attempt, n.cfg.Name
	cur.Result = &res
	cur.Owner, cur.Lease = "", 0
	cur.Clock = n.tickLocked()
	if err != nil {
		cur.State = JobFailed
		if res.Summary == "" {
			res.Summary = err.Error()
		}
	} else {
		cur.State = JobDone
	}
	n.cfg.Logf("job %s: %s (attempt %d, exit %d)", j.ID, cur.State, attempt, res.Exit)
}

// heartbeat extends the lease while we still own the job at this fence.
func (n *Node) heartbeat(id string, fence uint64, stop <-chan struct{}) {
	t := time.NewTicker(n.cfg.LeaseTTL / 3)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			n.mu.Lock()
			if j := n.st.Jobs[id]; j != nil && j.Owner == n.cfg.Name && j.Fence == fence && j.State == JobRunning {
				j.Lease = n.cfg.Now().Add(n.cfg.LeaseTTL).Unix()
				j.Clock = n.tickLocked()
			}
			n.mu.Unlock()
		}
	}
}

// tlsOnly drops a connection whose first byte is not a TLS record, so plaintext gets no
// application response at all (net/http would otherwise answer a 400 in the clear).
type tlsOnly struct{ net.Listener }

func (l tlsOnly) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		br := bufio.NewReader(c)
		_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
		b, err := br.Peek(1)
		_ = c.SetReadDeadline(time.Time{})
		if err != nil || b[0] != 0x16 {
			c.Close()
			continue
		}
		return peekConn{c, br}, nil
	}
}

type peekConn struct {
	net.Conn
	r *bufio.Reader
}

func (p peekConn) Read(b []byte) (int, error) { return p.r.Read(b) }

// DryRunner sleeps for the recipe's seconds (or a short default) instead of launching d2d.
type DryRunner struct{}

func (DryRunner) Run(ctx context.Context, j *JobRecord, attempt int) (JobResult, error) {
	d := 100 * time.Millisecond
	if s, ok := j.Args["seconds"]; ok {
		if v, err := time.ParseDuration(s + "s"); err == nil {
			d = v
		}
	}
	select {
	case <-ctx.Done():
		return JobResult{Exit: 130, Summary: "cancelled"}, ctx.Err()
	case <-time.After(d):
		return JobResult{Exit: 0, Summary: fmt.Sprintf("dry run %s attempt %d", j.Recipe, attempt)}, nil
	}
}
