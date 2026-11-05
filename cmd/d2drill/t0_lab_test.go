package main

import "testing"

func TestT0Flags(t *testing.T) {
	c, err := t0Config("cautious,holdout:101,holdout:202", "", 10, "")
	if err != nil || len(c.Profiles) != 3 || len(c.Seeds) != 2 || c.Seeds[1] != 202 {
		t.Fatalf("%+v %v", c, err)
	}
	for _, bad := range []string{"holdout", "holdout:x", "bold:3", "file"} {
		if _, err := t0Config(bad, "", 10, ""); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if _, err := t0Config("file", "a.json", 10, ""); err != nil {
		t.Fatal(err)
	}
}
