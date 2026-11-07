package currency

import (
	"strings"
	"testing"
)

func TestDefaultRegistryIsValid(t *testing.T) {
	r := Default()
	if c, err := r.ByRecipe("cube_bank"); err != nil || c.ID != "mark" {
		t.Fatalf("ByRecipe: %v %v", c, err)
	}
	if o := r.Ordered(); o[0].ID != "mark" || o[1].ID != "tally" {
		t.Fatalf("order: %v", o)
	}
	if _, err := r.Get("nope"); err == nil || !strings.Contains(err.Error(), `"nope"`) {
		t.Fatalf("unknown currency error: %v", err)
	}
}

func TestRegistryValidation(t *testing.T) {
	good := func() string { return string(defaultJSON) }
	for _, tc := range []struct{ name, from, to, want string }{
		{"unknown scope", `"scope": "per_character"`, `"scope": "global"`, `unknown scope "global"`},
		{"duplicate id", `"id": "tally"`, `"id": "mark"`, `duplicate currency id "mark"`},
		{"cycle", `"sinks": ["ladder_upgrade"],`, `"sinks": ["ladder_upgrade"], "converts_to": {"currency": "mark", "ratio": 1, "via": "back"},`, `converts_to cycle`},
		{"stack cap", `"cap": 12`, `"cap": 0`, `needs a stack cap`},
		{"cap on ledger", `"stackable": false,
      "cap": 0`, `"stackable": false,
      "cap": 5`, `cap 5 on a non-stackable`},
		{"unknown storage", `"storage": "ledger"`, `"storage": "cloud"`, `unknown storage "cloud"`},
		{"bad ratio", `"ratio": 1`, `"ratio": 0`, `ratio > 0`},
		{"unknown field", `"blurb": "A token`, `"colour": "red", "blurb": "A token`, `unknown field`},
		{"orphan source", `"sources": ["mark_field"]`, `"sources": ["bank"]`, `belongs to another currency`},
		{"bad sink", `"sink": "ladder_upgrade", "currency": "tally", "tier": "T0", "name": "Faster`, `"sink": "nope", "currency": "tally", "tier": "T0", "name": "Faster`, `not a sink`},
		{"schema", `currency.registry/1`, `currency.registry/9`, `schema`},
	} {
		src := good()
		if !strings.Contains(src, tc.from) {
			t.Fatalf("%s: fixture text %q not in registry.json", tc.name, tc.from)
		}
		_, err := Parse([]byte(strings.Replace(src, tc.from, tc.to, 1)))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err %v, want %q", tc.name, err, tc.want)
		}
	}
}
