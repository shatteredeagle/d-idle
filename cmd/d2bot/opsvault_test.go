package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/vault"
)

// `d2bot ops vault check`: read only, exit 0 clean, 1 with findings, 2 on a usage error.
func TestOpsVaultCheck(t *testing.T) {
	for _, hero := range []string{"sorceress", "paladin"} {
		root := t.TempDir()
		st, err := vault.OpenHero(root, hero, vault.Options{NoSync: true, Now: time.Now})
		if err != nil {
			t.Fatal(err)
		}
		rec, err := st.Add(vault.Item{Raw: "aabbcc", Code: "rin", Name: "ring", Quality: 4, Ilvl: 10, Identified: true})
		if err != nil {
			t.Fatal(err)
		}
		dir := vault.HeroDir(root, hero)
		var out bytes.Buffer
		if code, err := opsMain([]string{"vault", "check", "--dir", dir}, &out); code != 0 || err != nil || !strings.Contains(out.String(), "0 finding(s)") {
			t.Fatalf("clean: %d %v %q", code, err, out.String())
		}
		if _, err := st.Reserve([]string{rec.ID}, "s-lost", 0); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		if code, _ := opsMain([]string{"vault", "check", "--dir", dir}, &out); code != 1 || !strings.Contains(out.String(), "orphan-hold") {
			t.Fatalf("orphan hold: %d %q", code, out.String())
		}
		if code, err := opsMain([]string{"vault", "repair"}, &out); code != 2 || err == nil {
			t.Fatalf("unknown verb: %d %v", code, err)
		}
		if code, err := opsMain([]string{"vault", "check", "--dir", root + "/none"}, &out); code != 1 || err == nil {
			t.Fatalf("missing vault: %d %v", code, err)
		}
		st.Close()
	}
}
