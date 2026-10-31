package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/d2dctl/fake"
)

func TestProbe(t *testing.T) {
	path, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	srv, err := fake.New(path, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	var out, errb bytes.Buffer
	if code := run([]string{"-sock", path, "npcs"}, &out, &errb); code != 0 || !strings.Contains(out.String(), "Kashya") {
		t.Fatalf("%d %q %q", code, out.String(), errb.String())
	}
	out.Reset()
	if code := run([]string{"-sock", path, "-json", "monsters"}, &out, &errb); code != 0 {
		t.Fatal(code)
	}
	var j struct {
		OK     bool
		Parsed []struct{ Name string }
	}
	if err := json.Unmarshal(out.Bytes(), &j); err != nil || !j.OK || len(j.Parsed) != 2 || j.Parsed[1].Name != "Fallen Shaman" {
		t.Fatalf("%v %s", err, out.String())
	}
	out.Reset()
	if code := run([]string{"-sock", path, "raw", "bogus"}, &out, &errb); code != 1 {
		t.Fatalf("err reply should exit 1, got %d", code)
	}
	if code := run([]string{"-sock", "tcp://127.0.0.1:1", "ping"}, &out, &errb); code != 2 {
		t.Fatal("tcp must be refused")
	}

	sc := filepath.Join(t.TempDir(), "s.txt")
	os.WriteFile(sc, []byte("ping\nnpcs\nexpect Kashya\n"), 0o600)
	out.Reset()
	if code := run([]string{"-sock", path, "-script", sc}, &out, &errb); code != 0 {
		t.Fatalf("%d %s %s", code, out.String(), errb.String())
	}
	os.WriteFile(sc, []byte("npcs\nexpect Nobody\n"), 0o600)
	out.Reset()
	if code := run([]string{"-sock", path, "-json", "-script", sc}, &out, &errb); code != 1 || !strings.Contains(out.String(), `"ok": false`) {
		t.Fatalf("%d %s", code, out.String())
	}
}
