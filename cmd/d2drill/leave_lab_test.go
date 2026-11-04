package main

import (
	"context"
	"errors"
	"testing"
)

type fakeCycler struct {
	calls    []string
	leaveErr error
}

func (f *fakeCycler) LeaveGame(context.Context) error {
	f.calls = append(f.calls, "leave")
	return f.leaveErr
}

func (f *fakeCycler) EnterGame(context.Context, string) error {
	f.calls = append(f.calls, "enter")
	return nil
}

func TestLeaveHookExitsThenRejoins(t *testing.T) {
	c := &fakeCycler{}
	told := 0
	if err := leaveHook(c, func() { told++ })(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(c.calls) != 2 || c.calls[0] != "leave" || c.calls[1] != "enter" || told != 1 {
		t.Fatalf("calls %v, told %d", c.calls, told)
	}
}

func TestLeaveHookExitFailureIsNotAnExit(t *testing.T) {
	c := &fakeCycler{leaveErr: errors.New("socket")}
	told := 0
	if err := leaveHook(c, func() { told++ })(context.Background()); err == nil || told != 0 || len(c.calls) != 1 {
		t.Fatalf("err %v, told %d, calls %v", err, told, c.calls)
	}
}
