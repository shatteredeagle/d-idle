package questspec

import (
	"strings"
	"testing"

	"d2lod-bot/internal/questboard"
)

func hasRule(fs []Finding, rule int, sub string) bool {
	for _, f := range fs {
		if f.Rule == rule && strings.Contains(f.Msg, sub) {
			return true
		}
	}
	return false
}

func TestRule12Order(t *testing.T) {
	ok := questboard.Quest{ID: "q", Chunks: []questboard.Chunk{{ID: "a"}, {ID: "b", After: []string{"a"}}}}
	if fs := checkOrder([]questboard.Quest{ok}); len(fs) != 0 {
		t.Fatalf("valid order: %v", fs)
	}
	bad := questboard.Quest{ID: "q", Chunks: []questboard.Chunk{{ID: "b", After: []string{"a"}}, {ID: "a"}}}
	if fs := checkOrder([]questboard.Quest{bad}); !hasRule(fs, 12, "not a topological order") {
		t.Fatalf("misordered chunks not found: %v", fs)
	}
	unk := questboard.Quest{ID: "q", Chunks: []questboard.Chunk{{ID: "a", After: []string{"zz"}}}}
	if fs := checkOrder([]questboard.Quest{unk}); !hasRule(fs, 12, "not a chunk") {
		t.Fatalf("unknown after not found: %v", fs)
	}
}

func volCaps() *Caps {
	return &Caps{Backend: "b", Readers: map[string]Reader{
		"visited": {Tier: TierProxy, Volatile: true},
		"flag":    {Tier: TierSave},
	}}
}

func latchChain(post string, latched ...string) *Chain {
	return &Chain{ID: "c", Latched: latched, Steps: []Step{
		{ID: "s1", Chunk: "k1", Post: []string{post}},
		{ID: "s2", Chunk: "k2", Post: []string{"flag(x)"}},
	}}
}

func TestRule13Latched(t *testing.T) {
	caps := []*Caps{volCaps()}
	if fs := CheckLatched(latchChain("visited(x)", "k1"), caps); len(fs) != 0 {
		t.Fatalf("marked chain: %v", fs)
	}
	if fs := CheckLatched(latchChain("visited(x)"), caps); !hasRule(fs, 13, "list the chunk in latched") {
		t.Fatalf("missing mark not found: %v", fs)
	}
	if fs := CheckLatched(latchChain("flag(x)", "k1"), caps); !hasRule(fs, 13, "durable reader") {
		t.Fatalf("needless mark not found: %v", fs)
	}
	if fs := CheckLatched(latchChain("flag(x)", "nope"), caps); !hasRule(fs, 13, "no chunk") {
		t.Fatalf("unknown chunk not found: %v", fs)
	}
	if fs := CheckLatched(latchChain("visited(x)|flag(x)"), caps); len(fs) != 0 {
		t.Fatalf("a durable alternative needs no latch: %v", fs)
	}
}

func TestRule14GuardCodes(t *testing.T) {
	known := func(c string) (string, bool) {
		r, ok := questboard.ReasonOf(c)
		return string(r.Remedy), ok
	}
	if fs := CheckGuardCodes("t", []string{"needs_potions", "too_weak"}, known); len(fs) != 0 {
		t.Fatalf("valid codes: %v", fs)
	}
	if fs := CheckGuardCodes("t", []string{"made_up"}, known); !hasRule(fs, 14, "not a code") {
		t.Fatalf("unknown code not found: %v", fs)
	}
	noRemedy := func(string) (string, bool) { return "", true }
	if fs := CheckGuardCodes("t", []string{"x"}, noRemedy); !hasRule(fs, 14, "no remedy") {
		t.Fatalf("missing remedy not found: %v", fs)
	}
}
