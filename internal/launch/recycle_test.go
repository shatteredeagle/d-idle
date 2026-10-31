package launch

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func fakeRecycler(p RecyclePolicy, fds *int, calls *[]string) *Recycler {
	step := func(n string) func(context.Context) error {
		return func(context.Context) error { *calls = append(*calls, n); return nil }
	}
	t := time.Unix(1000, 0)
	return &Recycler{Policy: p, FDs: func() (int, error) { return *fds, nil },
		Leave: step("leave"), Stop: step("stop"), Start: step("start"), Login: step("login"),
		Now: func() time.Time { t = t.Add(time.Minute); return t }}
}

func TestRecycleZeroPolicyNeverDue(t *testing.T) {
	fds := 99999
	var c []string
	r := fakeRecycler(RecyclePolicy{}, &fds, &c)
	for i := 0; i < 1000; i++ {
		r.NoteCycle()
	}
	if ok, err := r.Maybe(context.Background()); ok || err != nil || len(c) != 0 {
		t.Fatal(ok, err, c)
	}
}

func TestRecycleByCyclesAndOrder(t *testing.T) {
	fds := 60
	var c []string
	r := fakeRecycler(RecyclePolicy{MaxCycles: 3}, &fds, &c)
	r.NoteCycle()
	r.NoteCycle()
	if ok, _ := r.Maybe(context.Background()); ok {
		t.Fatal("early")
	}
	r.NoteCycle()
	if ok, err := r.Maybe(context.Background()); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if want := []string{"leave", "stop", "start", "login"}; !reflect.DeepEqual(c, want) {
		t.Fatal(c)
	}
	if ok, _ := r.Due(); ok || r.Recycles() != 1 {
		t.Fatal("counter not reset")
	}
}

func TestRecycleByFDsAndMinInterval(t *testing.T) {
	fds := 500
	var c []string
	r := fakeRecycler(RecyclePolicy{MaxFDs: 400, MinInterval: time.Hour}, &fds, &c)
	if ok, err := r.Maybe(context.Background()); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if ok, _ := r.Maybe(context.Background()); ok {
		t.Fatal("MinInterval ignored") // fds still high, but the fake clock moved 1 min
	}
}

func TestRecycleLeaveFailureContinuesStartFailureReported(t *testing.T) {
	fds := 0
	var c []string
	r := fakeRecycler(RecyclePolicy{MaxCycles: 1, MinInterval: time.Hour}, &fds, &c)
	r.Leave = func(context.Context) error { return errors.New("stuck") }
	boom := errors.New("boom")
	r.Start = func(context.Context) error { return boom }
	r.NoteCycle()
	if _, err := r.Maybe(context.Background()); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if r.Recycles() != 0 {
		t.Fatal("counted a failed recycle")
	}
	if ok, _ := r.Due(); ok {
		t.Fatal("retry storm: MinInterval must apply after a failure")
	}
}

func TestRecyclePolicyFromEnv(t *testing.T) {
	t.Setenv("D2BOT_RECYCLE_CYCLES", "7")
	t.Setenv("D2BOT_RECYCLE_FDS", "0")
	t.Setenv("D2BOT_RECYCLE_MIN_INTERVAL", "5m")
	p := RecyclePolicyFromEnv()
	if p.MaxCycles != 7 || p.MaxFDs != 0 || p.MinInterval != 5*time.Minute {
		t.Fatal(p)
	}
}

func TestCountFDsSelf(t *testing.T) {
	if n, err := CountFDs(selfPID()); err != nil || n < 3 {
		t.Fatal(n, err)
	}
}

func TestPolicyDuePureAndStatus(t *testing.T) {
	p := RecyclePolicy{MaxCycles: 5, MaxFDs: 300, MinInterval: time.Hour}
	now := time.Unix(10000, 0)
	cases := []struct {
		st  RecycleStats
		due bool
	}{
		{RecycleStats{Cycles: 4, FDs: 100, Now: now}, false},
		{RecycleStats{Cycles: 5, FDs: 100, Now: now}, true},
		{RecycleStats{Cycles: 0, FDs: 300, Now: now}, true},
		{RecycleStats{Cycles: 0, FDs: -1, Now: now}, false},
		{RecycleStats{Cycles: 9, FDs: 999, Last: now.Add(-time.Minute), Now: now}, false}, // loop guard
		{RecycleStats{Cycles: 9, FDs: 999, Last: now.Add(-2 * time.Hour), Now: now}, true},
	}
	for i, c := range cases {
		if got, _ := p.Due(c.st); got != c.due {
			t.Errorf("case %d: %+v got %v", i, c.st, got)
		}
	}
	fds := 120
	var calls []string
	r := fakeRecycler(p, &fds, &calls)
	r.NoteCycle()
	s := r.Status()
	if !s.Enabled || s.Cycles != 1 || s.FDs != 120 || s.Due || !s.NextEligible.IsZero() {
		t.Fatalf("%+v", s)
	}
	fds = 400
	if ok, err := r.Maybe(context.Background()); !ok || err != nil {
		t.Fatal(ok, err)
	}
	s = r.Status()
	if s.Recycles != 1 || s.Cycles != 0 || s.NextEligible.IsZero() || s.LastRecycle.IsZero() || s.Due {
		t.Fatalf("%+v", s)
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("D2BOT_RECYCLE_CYCLES", "7")
	t.Setenv("D2BOT_RECYCLE_FDS", "0")
	t.Setenv("D2BOT_RECYCLE_MIN_INTERVAL", "2m")
	r := NewRecycler()
	if r.Policy != (RecyclePolicy{MaxCycles: 7, MaxFDs: 0, MinInterval: 2 * time.Minute}) {
		t.Fatalf("%+v", r.Policy)
	}
	if (&Recycler{}).FromEnv().Policy != r.Policy {
		t.Fatal("FromEnv")
	}
}

func TestRecycleProgressPerPhase(t *testing.T) {
	fds := 0
	var c, ph []string
	r := fakeRecycler(RecyclePolicy{}, &fds, &c)
	r.Progress = func(p string) { ph = append(ph, p) }
	if err := r.Recycle(context.Background(), "t"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(ph, ","); got != "leave,stop,start,login" {
		t.Fatalf("phases %s", got)
	}
	r.Start = func(context.Context) error { return errors.New("boom") }
	ph = nil
	_ = r.Recycle(context.Background(), "t")
	if got := strings.Join(ph, ","); got != "leave,stop" {
		t.Fatalf("failed start must stop reporting progress, got %s", got)
	}
}

func TestRecycleGuardBusyAndReleased(t *testing.T) {
	fds := 0
	var c []string
	r := fakeRecycler(RecyclePolicy{}, &fds, &c)
	released := 0
	r.Guard = func() (func(), error) { return func() { released++ }, nil }
	if err := r.Recycle(context.Background(), "t"); err != nil || released != 1 {
		t.Fatalf("err=%v released=%d", err, released)
	}
	c = nil
	r.Guard = func() (func(), error) { return nil, ErrLocked }
	if err := r.Recycle(context.Background(), "t"); !errors.Is(err, ErrLocked) || len(c) != 0 {
		t.Fatalf("busy lock must refuse before touching the game: err=%v calls=%v", err, c)
	}
}
