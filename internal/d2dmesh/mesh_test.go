package d2dmesh

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testToken = "test-join-token-not-a-secret"

// fast makes a node config for loopback tests: 40 ms rounds, 300 ms leases.
func fast(name string, seeds []string) Config {
	return Config{Name: name, Listen: "127.0.0.1:0", Seeds: seeds, Token: testToken, Capacity: 1,
		Interval: 40 * time.Millisecond, LeaseTTL: 300 * time.Millisecond, Fanout: 3,
		Logf: func(string, ...any) {}}
}

func startNode(t *testing.T, cfg Config) *Node {
	t.Helper()
	n, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(n.Stop)
	return n
}

// cluster starts A, then B and C seeded only with A.
func cluster(t *testing.T) (a, b, c *Node) {
	t.Helper()
	a = startNode(t, fast("a", nil))
	b = startNode(t, fast("b", []string{a.Addr()}))
	c = startNode(t, fast("c", []string{a.Addr()}))
	return
}

// eventually polls cond for up to 3 s (about 75 rounds at the test interval).
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func names(s *State) string {
	var out []string
	for _, n := range s.NodesSorted() {
		out = append(out, n.Name)
	}
	return strings.Join(out, ",")
}

func TestJoinAndConvergence(t *testing.T) {
	a, b, c := cluster(t)
	eventually(t, "membership a,b,c on every node", func() bool {
		return names(a.Snapshot()) == "a,b,c" && names(b.Snapshot()) == "a,b,c" && names(c.Snapshot()) == "a,b,c"
	})
	// After a quiet period the three tables are identical (Equal ignores nothing that matters).
	eventually(t, "tables equal", func() bool {
		sa, sb, sc := a.Snapshot(), b.Snapshot(), c.Snapshot()
		return sa.Equal(sb) && sb.Equal(sc)
	})
	// Convergence bound: a fresh change is on every node within 3 rounds.
	tr := true
	if err := a.Control("c", &tr, nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * a.cfg.Interval)
	if !b.Snapshot().Nodes["c"].Ctl.Paused || !c.Snapshot().Nodes["c"].Ctl.Paused {
		t.Fatal("pause of c not visible on b and c after 3 rounds")
	}
}

func TestSubmitOnAClaimOnC(t *testing.T) {
	a, b, c := cluster(t)
	eventually(t, "membership", func() bool { return names(c.Snapshot()) == "a,b,c" })
	// Only C can run it: A is paused, B lacks the label.
	tr := true
	if err := a.Control("a", &tr, nil); err != nil {
		t.Fatal(err)
	}
	_ = b
	c.SetLabels([]string{"d2d"})
	id, err := a.Submit("sleep", map[string]string{"seconds": "0"}, []string{"d2d"})
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "job done on c, seen by a", func() bool {
		j := a.Snapshot().Jobs[id]
		return j != nil && j.State == JobDone && j.Result != nil && j.Result.Node == "c" && j.Attempt == 1
	})
}

// slowRunner blocks until cancelled or released; counts attempts.
type slowRunner struct {
	started atomic.Int32
	mu      sync.Mutex
	release chan struct{}
}

func newSlow() *slowRunner { return &slowRunner{release: make(chan struct{})} }

// arm replaces the release channel so the next run blocks again.
func (s *slowRunner) arm() {
	s.mu.Lock()
	s.release = make(chan struct{})
	s.mu.Unlock()
}

func (s *slowRunner) go_() {
	s.mu.Lock()
	close(s.release)
	s.mu.Unlock()
}

func (s *slowRunner) Run(ctx context.Context, j *JobRecord, attempt int) (JobResult, error) {
	s.started.Add(1)
	s.mu.Lock()
	rel := s.release
	s.mu.Unlock()
	select {
	case <-ctx.Done():
		return JobResult{Exit: 130}, ctx.Err()
	case <-rel:
		return JobResult{Exit: 0, Summary: "released"}, nil
	}
}

func TestOwnerDiesJobRequeued(t *testing.T) {
	a := startNode(t, fast("a", nil))
	bcfg := fast("b", []string{a.Addr()})
	br := newSlow()
	bcfg.Runner = br
	bcfg.Labels = []string{"x"}
	b := startNode(t, bcfg)
	ccfg := fast("c", []string{a.Addr()})
	cr := newSlow()
	ccfg.Runner = cr
	c := startNode(t, ccfg)
	eventually(t, "membership", func() bool { return names(c.Snapshot()) == "a,b,c" })
	tr := true
	_ = a.Control("a", &tr, nil) // a never runs anything in this test
	id, err := a.Submit("sleep", map[string]string{"seconds": "5"}, []string{"x"})
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "b running attempt 1", func() bool {
		j := a.Snapshot().Jobs[id]
		return j != nil && j.State == JobRunning && j.Owner == "b" && j.Attempt == 1
	})
	b.Stop() // crash: no tombstone, no more heartbeats
	eventually(t, "lease expired and re-queued", func() bool {
		j := a.Snapshot().Jobs[id]
		return j != nil && j.Fence >= 2 && (j.State == JobQueued || j.State == JobRunning)
	})
	// C has no "x" label: the job must stay queued; give it the label and it claims attempt 2.
	time.Sleep(3 * a.cfg.Interval)
	if j := c.Snapshot().Jobs[id]; j.State == JobRunning {
		t.Fatalf("c claimed a job whose labels it lacks: %+v", j)
	}
	c.SetLabels([]string{"x"})
	eventually(t, "c running attempt 2", func() bool {
		j := a.Snapshot().Jobs[id]
		return j != nil && j.State == JobRunning && j.Owner == "c" && j.Attempt == 2 && j.Fence == 3
	})
	cr.go_()
	eventually(t, "done by c attempt 2", func() bool {
		j := a.Snapshot().Jobs[id]
		return j != nil && j.State == JobDone && j.Result.Node == "c" && j.Result.Attempt == 2
	})
	if br.started.Load() != 1 || cr.started.Load() != 1 {
		t.Fatalf("attempts: b=%d c=%d", br.started.Load(), cr.started.Load())
	}
}

func TestPartitionHealAndSplitBrainClaim(t *testing.T) {
	a, b, c := cluster(t)
	eventually(t, "membership", func() bool { return names(c.Snapshot()) == "a,b,c" })
	// Cut B off. Pause A on A's side and drain B on B's side; submit a job on each side.
	b.SetPartitioned(true)
	time.Sleep(3 * a.cfg.Interval) // let exchanges that were in flight at the cut finish
	tr := true
	_ = a.Control("a", &tr, nil)
	_ = b.Control("b", nil, &tr)
	idA, _ := a.Submit("sleep", map[string]string{"seconds": "0"}, nil)
	idB, _ := b.Submit("sleep", map[string]string{"seconds": "0"}, nil)
	eventually(t, "a-side job done by c", func() bool {
		j := a.Snapshot().Jobs[idA]
		return j != nil && j.State == JobDone && j.Result.Node == "c"
	})
	time.Sleep(3 * a.cfg.Interval)
	if _, ok := a.Snapshot().Jobs[idB]; ok {
		t.Fatal("b's job leaked across the partition")
	}
	if j := b.Snapshot().Jobs[idB]; j.State != JobQueued {
		t.Fatalf("draining b ran its own job: %+v", j)
	}
	b.SetPartitioned(false)
	eventually(t, "heal: both sides merged", func() bool {
		sa, sb := a.Snapshot(), b.Snapshot()
		jb := sa.Jobs[idB]
		return jb != nil && jb.State == JobDone && sb.Jobs[idA] != nil && sb.Jobs[idA].State == JobDone &&
			sa.Nodes["b"].Ctl.Draining && sb.Nodes["a"].Ctl.Paused
	})
	// Split-brain double claim: the same job record claimed on two sides converges to one
	// owner by (fence, clock, owner) order and the loser's run is cancelled.
	x := &State{Wire: WireVersion, Nodes: map[string]*NodeRecord{}, Jobs: map[string]*JobRecord{
		"j1": {ID: "j1", Recipe: "sleep", State: JobRunning, Owner: "p", Fence: 1, Clock: 7, Attempt: 1, Lease: time.Now().Add(time.Minute).Unix()}}}
	y := x.Clone()
	y.Jobs["j1"].Owner = "q"
	y.Jobs["j1"].Clock = 7
	x2, y2 := x.Clone(), y.Clone()
	x2.Merge(y)
	y2.Merge(x)
	if x2.Jobs["j1"].Owner != "p" || y2.Jobs["j1"].Owner != "p" {
		t.Fatalf("split-brain tie did not converge: %s vs %s", x2.Jobs["j1"].Owner, y2.Jobs["j1"].Owner)
	}
}

func TestPauseAndDrain(t *testing.T) {
	a := startNode(t, fast("a", nil))
	bcfg := fast("b", []string{a.Addr()})
	br := newSlow()
	bcfg.Runner = br
	b := startNode(t, bcfg)
	tr, fl := true, false
	_ = a.Control("a", &tr, nil)
	eventually(t, "membership", func() bool { return names(b.Snapshot()) == "a,b" })
	id1, _ := a.Submit("sleep", map[string]string{"seconds": "9"}, nil)
	eventually(t, "b runs id1", func() bool {
		j := a.Snapshot().Jobs[id1]
		return j != nil && j.State == JobRunning && j.Owner == "b"
	})
	// drain from A: the running job finishes, nothing new is claimed
	_ = a.Control("b", nil, &tr)
	eventually(t, "b sees draining", func() bool { return b.Snapshot().Nodes["b"].Ctl.Draining })
	id2, _ := a.Submit("sleep", map[string]string{"seconds": "0"}, nil)
	br.go_()
	eventually(t, "id1 done", func() bool { j := a.Snapshot().Jobs[id1]; return j != nil && j.State == JobDone })
	time.Sleep(4 * a.cfg.Interval)
	if j := a.Snapshot().Jobs[id2]; j.State != JobQueued {
		t.Fatalf("draining b claimed id2: %+v", j)
	}
	// undrain + pause: still nothing; resume: it runs
	_ = a.Control("b", &tr, &fl)
	time.Sleep(4 * a.cfg.Interval)
	if j := a.Snapshot().Jobs[id2]; j.State != JobQueued {
		t.Fatalf("paused b claimed id2: %+v", j)
	}
	_ = a.Control("b", &fl, nil)
	eventually(t, "id2 done after resume", func() bool { j := a.Snapshot().Jobs[id2]; return j != nil && j.State == JobDone })
	// cancel a running job: fence moves, state cancelled on both
	br.arm()
	id3, _ := a.Submit("sleep", map[string]string{"seconds": "9"}, nil)
	eventually(t, "id3 running", func() bool { j := b.Snapshot().Jobs[id3]; return j != nil && j.State == JobRunning })
	if err := a.Cancel(id3); err != nil {
		t.Fatal(err)
	}
	// the cancel may race b's claim at the same fence: cancelled must win either way and b's run stops
	eventually(t, "id3 cancelled on b, run stopped", func() bool {
		j := b.Snapshot().Jobs[id3]
		return j != nil && j.State == JobCancelled && b.RunningJobs() == 0
	})
}

func TestHostGuardAdmission(t *testing.T) {
	a := startNode(t, fast("a", nil))
	bcfg := fast("b", []string{a.Addr()})
	var say atomic.Bool
	bcfg.Guard = say.Load
	b := startNode(t, bcfg)
	tr := true
	_ = a.Control("a", &tr, nil)
	eventually(t, "membership", func() bool { return names(b.Snapshot()) == "a,b" })
	id, _ := a.Submit("sleep", map[string]string{"seconds": "0"}, nil)
	time.Sleep(4 * a.cfg.Interval)
	if j := b.Snapshot().Jobs[id]; j == nil || j.State != JobQueued {
		t.Fatalf("guard NO but job not queued: %+v", j)
	}
	say.Store(true)
	eventually(t, "guard yes: done", func() bool { j := a.Snapshot().Jobs[id]; return j != nil && j.State == JobDone })
}

func TestTokenRejected(t *testing.T) {
	a := startNode(t, fast("a", nil))
	bad := fast("evil", []string{a.Addr()})
	bad.Token = "another-token"
	e := startNode(t, bad)
	time.Sleep(5 * a.cfg.Interval)
	if n := a.Snapshot().Nodes["evil"]; n != nil {
		t.Fatal("node with the wrong token joined")
	}
	if n := e.Snapshot().Nodes["a"]; n != nil {
		t.Fatal("wrong-token node learned a's record")
	}
	// Plain TLS without a client certificate (even with the right CA) is refused too.
	_, cli, _ := TLSConfigs(testToken, "x")
	cli.Certificates = nil
	c := &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: cli}}
	if resp, err := c.Get("https://" + a.Addr() + "/api/v1/state"); err == nil {
		resp.Body.Close()
		t.Fatal("request without a client certificate succeeded")
	}
	// Plaintext on the TLS port gets no application response.
	conn, err := net.Dial("tcp", a.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "GET /api/v1/state HTTP/1.0\r\n\r\n")
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 64)
	k, _ := conn.Read(buf)
	if bytes.Contains(buf[:k], []byte("HTTP/1.")) {
		t.Fatalf("plaintext got an HTTP response: %q", buf[:k])
	}
	_ = tls.VersionTLS13
}

func TestRecipeAllowList(t *testing.T) {
	a := startNode(t, fast("a", nil))
	cases := []struct {
		recipe string
		args   map[string]string
		ok     bool
	}{
		{"sleep", map[string]string{"seconds": "1"}, true},
		{"sleep", map[string]string{"seconds": "9999"}, false},
		{"sleep", map[string]string{}, false},
		{"sleep", map[string]string{"seconds": "1", "shell": "rm -rf /"}, false},
		{"bash", map[string]string{"c": "true"}, false},
		{"d2d-soak", map[string]string{"hours": "2", "class": "1", "mode": "lockstep"}, true},
		{"d2d-soak", map[string]string{"hours": "2", "class": "9"}, false},
		{"d2d-soak", map[string]string{"hours": "2", "class": "1", "name": "../x"}, false},
		{"d2d-replay", map[string]string{"scenario": "town-loop"}, true},
	}
	for _, c := range cases {
		_, err := a.Submit(c.recipe, c.args, nil)
		if (err == nil) != c.ok {
			t.Errorf("%s %v: ok=%v err=%v", c.recipe, c.args, c.ok, err)
		}
	}
	// over HTTP the rejection is a 422 and nothing is queued
	_, cli, _ := TLSConfigs(testToken, "cli")
	hc := &http.Client{Timeout: time.Second, Transport: &http.Transport{TLSClientConfig: cli}}
	body, _ := json.Marshal(submitReq{Recipe: "sh", Args: map[string]string{"c": "id"}})
	resp, err := hc.Post("https://"+a.Addr()+"/api/v1/jobs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if n := len(a.Snapshot().Jobs); n != 3 {
		t.Fatalf("%d jobs queued, want the 3 valid ones", n)
	}
}

func TestMergeRules(t *testing.T) {
	s := NewState()
	s.Nodes["n"] = &NodeRecord{Name: "n", Clock: 5, LastSeen: 10, Ctl: NodeControl{Paused: true, Clock: 3, By: "z"}}
	// older liveness record with a newer control register: only the register moves
	o := NewState()
	o.Nodes["n"] = &NodeRecord{Name: "n", Clock: 2, LastSeen: 1, Ctl: NodeControl{Paused: false, Clock: 4, By: "a"}}
	if !s.Merge(o) || s.Nodes["n"].Clock != 5 || s.Nodes["n"].Ctl.Paused {
		t.Fatalf("ctl register merge wrong: %+v", s.Nodes["n"])
	}
	// newer liveness record with an older register: liveness moves, register stays
	o.Nodes["n"] = &NodeRecord{Name: "n", Clock: 9, LastSeen: 20, Ctl: NodeControl{Paused: true, Clock: 1}}
	if !s.Merge(o) || s.Nodes["n"].Clock != 9 || s.Nodes["n"].Ctl.Paused || s.Nodes["n"].Ctl.Clock != 4 {
		t.Fatalf("liveness merge wrong: %+v", s.Nodes["n"])
	}
	// jobs: a terminal state at a higher fence beats running; a lower fence never downgrades
	s.Jobs["j"] = &JobRecord{ID: "j", State: JobRunning, Owner: "n", Fence: 2, Clock: 1}
	o.Jobs["j"] = &JobRecord{ID: "j", State: JobDone, Fence: 3, Clock: 0}
	s.Merge(o)
	// a cancel at the same fence as a running claim with a higher clock still wins
	s.Jobs["c"] = &JobRecord{ID: "c", State: JobRunning, Owner: "n", Fence: 1, Clock: 50}
	o.Jobs["c"] = &JobRecord{ID: "c", State: JobCancelled, Fence: 1, Clock: 2}
	if !s.Merge(o) || s.Jobs["c"].State != JobCancelled {
		t.Fatalf("concurrent cancel lost: %+v", s.Jobs["c"])
	}
	o.Jobs["j"] = &JobRecord{ID: "j", State: JobRunning, Owner: "m", Fence: 1, Clock: 99}
	if s.Merge(o) || s.Jobs["j"].State != JobDone {
		t.Fatalf("job merge wrong: %+v", s.Jobs["j"])
	}
	// another wire version is ignored
	o.Wire = 2
	o.Jobs["k"] = &JobRecord{ID: "k", State: JobQueued}
	if s.Merge(o) || s.Jobs["k"] != nil {
		t.Fatal("merged a foreign wire version")
	}
	// unknown fields are ignored by the decoder, known ones kept
	st, err := DecodeState(strings.NewReader(`{"wire":1,"nodes":{"q":{"name":"q","clock":1,"future_field":true}},"jobs":{}}`))
	if err != nil || st.Nodes["q"] == nil || st.Nodes["q"].Clock != 1 {
		t.Fatalf("decode with unknown field: %v %+v", err, st)
	}
	if _, err := DecodeState(strings.NewReader(`{"wire":1,"nodes":{"q":{"name":"other"}},"jobs":{}}`)); err == nil {
		t.Fatal("key/name mismatch accepted")
	}
}

func TestLeaveTombstone(t *testing.T) {
	a := startNode(t, fast("a", nil))
	b := startNode(t, fast("b", []string{a.Addr()}))
	eventually(t, "membership", func() bool { return names(a.Snapshot()) == "a,b" })
	b.Leave()
	eventually(t, "b gone on a", func() bool {
		n := a.Snapshot().Nodes["b"]
		return n != nil && n.Gone && names(a.Snapshot()) == "a"
	})
}

func FuzzDecodeState(f *testing.F) {
	f.Add([]byte(`{"wire":1,"nodes":{},"jobs":{}}`))
	f.Add([]byte(`{"wire":1,"nodes":{"a":{"name":"a","addr":"127.0.0.1:1","clock":3,"ctl":{"paused":true,"clock":1}}},"jobs":{"j":{"id":"j","recipe":"sleep","state":"queued","fence":1}}}`))
	f.Add([]byte(`{"wire":2}`))
	f.Add([]byte(`{"wire":1,"nodes":{"a":null}}`))
	f.Add([]byte(`{"wire":1,"nodes":{"a":{"name":"b"}}}`))
	f.Add([]byte(`not json`))
	f.Fuzz(func(t *testing.T, data []byte) {
		s, err := DecodeState(bytes.NewReader(data))
		if err != nil {
			return
		}
		// a decoded table always merges into a fresh one without panicking and round-trips
		fresh := NewState()
		fresh.Merge(s)
		out, err := json.Marshal(fresh)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeState(bytes.NewReader(out)); err != nil {
			t.Fatalf("re-decode: %v", err)
		}
	})
}
