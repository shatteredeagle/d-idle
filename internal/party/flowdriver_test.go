package party

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/launch"
)

type fakeOOG struct{ calls []string }

func (f *fakeOOG) HostGame(_ context.Context, ch launch.CharSpec) error {
	f.calls = append(f.calls, "host "+ch.Name+" "+ch.LAN)
	return nil
}
func (f *fakeOOG) JoinGame(_ context.Context, ch launch.CharSpec, ip string) error {
	f.calls = append(f.calls, "join "+ch.Name+" "+ip)
	return nil
}
func (f *fakeOOG) LeaveGame(context.Context, time.Duration) error {
	f.calls = append(f.calls, "leave")
	return nil
}

func TestFlowDriverCreateJoinLeave(t *testing.T) {
	dir := NewDirectory()
	g := Game{Name: "pty1", Host: "127.0.0.1"}
	dir.Publish("p", g)
	o := &fakeOOG{}
	d := &FlowDriver{Flow: o, Char: launch.CharSpec{Name: "Combi"}, Dir: dir, PartyID: "p",
		Probe: func(context.Context, string) error { return nil }}
	ctx := context.Background()
	if err := d.Create(ctx, g); err != nil {
		t.Fatal(err)
	}
	if err := d.Join(ctx, g); err != nil {
		t.Fatal(err)
	}
	if err := d.Leave(ctx); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(o.calls, ",")
	if got != "leave,host Combi ,leave,join Combi 127.0.0.1,leave" {
		t.Fatalf("calls: %s", got)
	}
}

func TestFlowDriverRefusesStaleGame(t *testing.T) {
	dir := NewDirectory()
	dir.Publish("p", Game{Name: "pty2", Host: "127.0.0.1"})
	o := &fakeOOG{}
	d := &FlowDriver{Flow: o, Dir: dir, PartyID: "p", Probe: func(context.Context, string) error { return nil }}
	if err := d.Join(context.Background(), Game{Name: "pty1", Host: "127.0.0.1"}); !errors.Is(err, ErrStaleGame) {
		t.Fatalf("got %v", err)
	}
	dir.Revoke("p")
	if err := d.Join(context.Background(), Game{Name: "pty2", Host: "127.0.0.1"}); !errors.Is(err, ErrStaleGame) {
		t.Fatalf("got %v", err)
	}
	if len(o.calls) != 0 {
		t.Fatalf("no menu action expected: %v", o.calls)
	}
}

func TestFlowDriverHostDown(t *testing.T) {
	o := &fakeOOG{}
	d := &FlowDriver{Flow: o, ProbeWait: time.Second, Sleep: func(time.Duration) {},
		Probe: func(context.Context, string) error { return errors.New("refused") }}
	err := d.Join(context.Background(), Game{Name: "x", Host: "127.0.0.1"})
	if !errors.Is(err, ErrHostDown) || len(o.calls) != 0 {
		t.Fatalf("err=%v calls=%v", err, o.calls)
	}
}

var _ Driver = (*FlowDriver)(nil)
