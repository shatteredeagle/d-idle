package party

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/event"
)

type fakeDriver struct {
	mu         sync.Mutex
	log        *[]string
	id         string
	failJoin   int // number of Join calls that fail
	failCreate bool
}

func (f *fakeDriver) rec(s string) {
	f.mu.Lock()
	*f.log = append(*f.log, f.id+" "+s)
	f.mu.Unlock()
}
func (f *fakeDriver) Create(_ context.Context, g Game) error {
	f.rec("create " + g.Name)
	if f.failCreate {
		return errors.New("boom")
	}
	return nil
}
func (f *fakeDriver) Join(_ context.Context, g Game) error {
	f.rec("join " + g.Name)
	if f.failJoin > 0 {
		f.failJoin--
		return errors.New("refused")
	}
	return nil
}
func (f *fakeDriver) Leave(context.Context) error { f.rec("leave"); return nil }

type fakeBeh struct {
	log  *[]string
	id   string
	lead string
}

func (b *fakeBeh) Follow(_ context.Context, l string) error {
	*b.log = append(*b.log, b.id+" follow "+l)
	return nil
}
func (b *fakeBeh) Assist(_ context.Context, l string) error {
	*b.log = append(*b.log, b.id+" assist "+l)
	return nil
}
func (b *fakeBeh) Stop() { *b.log = append(*b.log, b.id+" stop") }

type rig struct {
	log []string
	drv map[string]*fakeDriver
	p   *Party
	bus *event.Bus
}

func newRig(t *testing.T, roles ...Role) *rig {
	t.Helper()
	r := &rig{drv: map[string]*fakeDriver{}, bus: event.NewBus("x", nil)}
	var ms []Member
	for i, ro := range roles {
		id := string(rune('a' + i))
		d := &fakeDriver{log: &r.log, id: id}
		r.drv[id] = d
		b := &fakeBeh{log: &r.log, id: id}
		ms = append(ms, Member{ID: id, Character: "Char" + strings.ToUpper(id), Role: ro, Driver: d, Following: b, Assisting: b})
	}
	p, err := New(Config{ID: "pty", Bus: r.bus, Sleep: func(context.Context, time.Duration) error { return nil }}, ms)
	if err != nil {
		t.Fatal(err)
	}
	r.p = p
	return r
}

func (r *rig) has(s string) bool {
	for _, l := range r.log {
		if l == s {
			return true
		}
	}
	return false
}

func TestValidate(t *testing.T) {
	d := &fakeDriver{log: new([]string)}
	m := func(id string, ro Role) Member { return Member{ID: id, Character: id, Role: ro, Driver: d} }
	cases := map[string][]Member{
		"one":      {m("a", Leader)},
		"five":     {m("a", Leader), m("b", Killer), m("c", Killer), m("d", Killer), m("e", Killer)},
		"solo":     {m("a", Leader), m("b", Solo)},
		"2leaders": {m("a", Leader), m("b", Leader)},
		"mules":    {m("a", Mule), m("b", Mule)},
		"dup":      {m("a", Leader), m("a", Killer)},
	}
	for n, ms := range cases {
		if _, err := New(Config{}, ms); err == nil {
			t.Errorf("%s: want error", n)
		}
	}
	if _, err := New(Config{}, []Member{m("a", Leader), m("b", Follower)}); err != nil {
		t.Fatal(err)
	}
}

func TestStartHostPreferenceAndOrder(t *testing.T) {
	r := newRig(t, Killer, Mule, Leader, Follower) // size 4
	if err := r.p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	sn := r.p.Status()
	if sn.State != Running || sn.Host != "c" || sn.Game.Password != "" || sn.Game.MaxPlayers != 4 {
		t.Fatalf("%+v", sn)
	}
	if r.log[0] != "c create pty1" {
		t.Fatalf("host first: %v", r.log)
	}
	if !r.has("a join pty1") || !r.has("b join pty1") || !r.has("d join pty1") {
		t.Fatal(r.log)
	}
	// followers follow the leader's character; the mule does not follow, the host (leader) runs nothing
	if !r.has("a follow CharC") || !r.has("d assist CharC") || r.has("b follow CharC") || r.has("c follow CharC") {
		t.Fatal(r.log)
	}
	if sn.Scaling.LifeFactor != 2.5 {
		t.Fatalf("scaling %+v", sn.Scaling)
	}
	if err := r.p.Start(context.Background()); !errors.Is(err, ErrState) {
		t.Fatal(err)
	}
}

func TestJoinRetryThenFail(t *testing.T) {
	r := newRig(t, Leader, Killer, Teleporter)
	r.drv["b"].failJoin = 2 // third try works
	r.drv["c"].failJoin = 99
	if err := r.p.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	st := map[string]MemberState{}
	for _, m := range r.p.Status().Members {
		st[m.ID] = m.State
	}
	if st["b"] != In || st["c"] != Failed || st["a"] != In {
		t.Fatalf("%v", st)
	}
	n := 0
	for _, l := range r.log {
		if l == "b join pty1" {
			n++
		}
	}
	if n != 3 {
		t.Fatalf("b tries %d", n)
	}
}

func TestCreateFailsAndNobodyJoins(t *testing.T) {
	r := newRig(t, Leader, Killer)
	r.drv["a"].failCreate = true
	if err := r.p.Start(context.Background()); err == nil {
		t.Fatal("want error")
	}
	if r.p.Status().State != Idle {
		t.Fatal(r.p.Status().State)
	}
	if _, ok := r.p.cfg.Dir.Lookup("pty"); ok {
		t.Fatal("game must be revoked")
	}
	r2 := newRig(t, Leader, Killer)
	r2.drv["b"].failJoin = 99
	if err := r2.p.Start(context.Background()); err == nil || !r2.has("a leave") {
		t.Fatalf("%v %v", err, r2.log)
	}
}

func TestDeathAndRevive(t *testing.T) {
	r := newRig(t, Leader, Killer, Teleporter)
	ctx := context.Background()
	_ = r.p.Start(ctx)
	r.log = nil
	if err := r.p.MemberDied("a"); err != nil {
		t.Fatal(err)
	}
	if !r.has("b stop") || !r.has("c stop") {
		t.Fatal("followers stop on leader death", r.log)
	}
	if r.p.Status().Scaling.Players != 3 {
		t.Fatal("dead members still count for scaling")
	}
	if err := r.p.MemberDied("a"); !errors.Is(err, ErrState) {
		t.Fatal(err)
	}
	if err := r.p.MemberRevived(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if !r.has("b follow CharA") {
		t.Fatal(r.log)
	}
}

func TestNonHostLostAndRejoin(t *testing.T) {
	r := newRig(t, Leader, Killer, Follower)
	ctx := context.Background()
	_ = r.p.Start(ctx)
	if err := r.p.MemberLost(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	if r.p.Status().State != Running {
		t.Fatal("party keeps running")
	}
	r.log = nil
	if err := r.p.Rejoin(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	if !r.has("b join pty1") || !r.has("b follow CharA") {
		t.Fatal(r.log)
	}
	if err := r.p.Rejoin(ctx, "b"); !errors.Is(err, ErrState) {
		t.Fatal("already in:", err)
	}
}

func TestHostLostHandoff(t *testing.T) {
	r := newRig(t, Leader, Killer, Teleporter)
	ctx := context.Background()
	_ = r.p.Start(ctx)
	r.log = nil
	if err := r.p.MemberLost(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	sn := r.p.Status()
	if sn.State != Running || sn.Host != "c" || sn.Game.Name != "pty2" {
		t.Fatalf("%+v", sn)
	}
	// everybody still in left the old game, then the new host created and the other joined the new name
	if !r.has("b leave") || !r.has("c create pty2") || !r.has("b join pty2") || r.has("a join pty2") {
		t.Fatal(r.log)
	}
	// the lost leader comes back to the new game; it is the leader again for followers
	r.log = nil
	if err := r.p.Rejoin(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if !r.has("a join pty2") || !r.has("b follow CharA") {
		t.Fatal(r.log)
	}
}

func TestLeaveHostDisbandsWhenTooFew(t *testing.T) {
	r := newRig(t, Leader, Killer)
	ctx := context.Background()
	_ = r.p.Start(ctx)
	if err := r.p.Leave(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if r.p.Status().State != Disbanded || !r.has("b leave") {
		t.Fatalf("%v %v", r.p.Status().State, r.log)
	}
}

func TestLeaveMemberAndDisband(t *testing.T) {
	r := newRig(t, Leader, Killer, Follower)
	ctx := context.Background()
	_ = r.p.Start(ctx)
	if err := r.p.Leave(ctx, "c"); err != nil {
		t.Fatal(err)
	}
	if err := r.p.Rejoin(ctx, "c"); err != nil { // Left members may be invited back explicitly
		t.Fatal(err)
	}
	if err := r.p.Disband(ctx); err != nil {
		t.Fatal(err)
	}
	if r.p.Status().State != Disbanded {
		t.Fatal()
	}
	if _, ok := r.p.cfg.Dir.Lookup("pty"); ok {
		t.Fatal("revoked")
	}
}

func TestEvents(t *testing.T) {
	r := newRig(t, Leader, Killer)
	sub := r.bus.Subscribe("t", 64)
	_ = r.p.Start(context.Background())
	var states []string
	for len(sub.C()) > 0 {
		e := <-sub.C()
		states = append(states, e.Payload().(event.Lifecycle).State)
	}
	want := "party.creating,party.running"
	if strings.Join(states, ",") != want {
		t.Fatal(states)
	}
}

func TestScaling(t *testing.T) {
	s := ScalingFor(4)
	if s.LifeFactor != 2.5 || s.DamagePct != 18.75 || s.XPPerKillRel < 1.2 || s.XPPerKillRel > 1.3 {
		t.Fatalf("%+v", s)
	}
	if len(ScalingFor(1).Notes) != 0 || ScalingFor(0).Players != 1 {
		t.Fatal()
	}
}

func TestRoleParse(t *testing.T) {
	if r, _ := Parse(""); r != Solo {
		t.Fatal()
	}
	if _, err := Parse("tank"); err == nil {
		t.Fatal()
	}
	if PolicyFor(Mule).Fights || !PolicyFor(Follower).Follows {
		t.Fatal()
	}
}
