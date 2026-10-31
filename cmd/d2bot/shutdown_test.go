package main

import (
	"reflect"
	"testing"
	"time"
)

func TestShutdownSavesBeforeWaitingOnGoroutines(t *testing.T) {
	var got []string
	rec := func(n string) func() { return func() { got = append(got, n) } }
	s := shutdownSeq{
		stopBot:      rec("stop"),
		settled:      func() bool { got = append(got, "settled"); return true },
		settleWait:   time.Second,
		summary:      rec("summary"),
		stopRunner:   rec("cancel"),
		leave:        rec("leave"),
		runnerDone:   rec("waitRunner"),
		closeBot:     rec("close"),
		apiDone:      rec("waitAPI"),
		stopGame:     true,
		shutdownGame: rec("kill"),
	}
	s.run()
	want := []string{"stop", "settled", "summary", "cancel", "leave", "waitRunner", "close", "waitAPI", "kill"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
}

func TestShutdownWithoutOwnedGame(t *testing.T) {
	var got []string
	rec := func(n string) func() { return func() { got = append(got, n) } }
	shutdownSeq{stopBot: rec("stop"), settled: func() bool { return true }, summary: rec("s"), stopRunner: rec("c"),
		runnerDone: rec("r"), closeBot: rec("x"), apiDone: rec("a"), shutdownGame: rec("kill")}.run()
	for _, g := range got {
		if g == "kill" {
			t.Fatal("game stopped though not owned")
		}
	}
}

func TestShutdownQuiescesRunnerBeforeLeave(t *testing.T) {
	var got []string
	rec := func(n string) func() { return func() { got = append(got, n) } }
	shutdownSeq{stopBot: rec("stop"), settled: func() bool { return true }, summary: rec("s"), stopRunner: rec("cancel"),
		quiesce: rec("quiesce"), leave: rec("leave"), runnerDone: rec("rd"), closeBot: rec("c"), apiDone: rec("a")}.run()
	want := []string{"stop", "s", "cancel", "quiesce", "leave", "rd", "c", "a"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
}
