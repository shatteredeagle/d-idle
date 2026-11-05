package main

import "testing"

func TestUsageReply(t *testing.T) {
	for out, want := range map[string]bool{
		"usage: debug <verb> ...": true,
		"Usage: cmd ...":          true,
		"ok":                      false,
		"":                        false,
		"inventory cleared":       false,
	} {
		if got := usageReply(out); got != want {
			t.Errorf("%q: %v", out, got)
		}
	}
}
