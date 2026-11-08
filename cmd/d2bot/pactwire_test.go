package main

import (
	"context"
	"strings"
	"testing"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/watch"
)

// Pacts are off in gates and measured runs: nothing is subscribed or started (a nil manager would panic otherwise).
func TestStartPactsOffInGatesAndMeasuredRuns(t *testing.T) {
	for _, env := range []map[string]string{{"D2BOT_PACTS": "off"}, {"D2BOT_RUN": "g2c"}} {
		startPacts(context.Background(), ladderOptions{}, &ladderRig{}, nil, nil, t.TempDir(), func(k string) string { return env[k] })
	}
}

func TestPactRefsNameTheMissingRow(t *testing.T) {
	cat, _ := decide.Shipped()
	tbl, err := decide.ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	// a sheet needs a store; the refs only read the table through DefaultOption, which a bare table can answer
	st, err := decide.Open(t.TempDir(), "pact", cat, &decide.ManualClock{})
	if err != nil {
		t.Fatal(err)
	}
	ps, err := watch.NewPolicySheet(st, cat, tbl, func() string { return "paladin" }, func() float64 { return 0 }, watch.NewAwayLog(nil), nil)
	if err != nil {
		t.Fatal(err)
	}
	r := pactRefs(ps)
	if err := r.Option("repair", "low"); err != nil {
		t.Error(err)
	}
	if err := r.Option("nosuchrow", "x"); err == nil || !strings.Contains(err.Error(), "nosuchrow") {
		t.Errorf("unknown row: %v", err)
	}
}
