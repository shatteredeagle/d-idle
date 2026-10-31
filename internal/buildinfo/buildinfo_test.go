package buildinfo

import (
	"bytes"
	"runtime/debug"
	"testing"
)

func TestHandle(t *testing.T) {
	var b bytes.Buffer
	if !Handle("x", []string{"--version"}, &b) || b.String() != "x dev\n" {
		t.Fatalf("got %q", b.String())
	}
	for _, a := range [][]string{nil, {"run"}, {"--version", "x"}} {
		if Handle("x", a, &b) {
			t.Fatalf("%v matched", a)
		}
	}
}

func TestFromSettings(t *testing.T) {
	set := func(kv ...string) (o []debug.BuildSetting) {
		for i := 0; i < len(kv); i += 2 {
			o = append(o, debug.BuildSetting{Key: kv[i], Value: kv[i+1]})
		}
		return
	}
	for _, c := range []struct {
		in   []debug.BuildSetting
		want string
	}{
		{nil, "dev"},
		{set("vcs.revision", "0123456789abcdef"), "0123456"},
		{set("vcs.revision", "0123456789abcdef", "vcs.modified", "true"), "0123456-dirty"},
		{set("vcs.revision", "abc", "vcs.modified", "false"), "abc"},
	} {
		if got := fromSettings(c.in); got != c.want {
			t.Errorf("%v: got %q want %q", c.in, got, c.want)
		}
	}
}

func TestCurrentPrefersStamp(t *testing.T) {
	old := Version
	defer func() { Version = old }()
	Version = "v1-3-gabc"
	if Current() != "v1-3-gabc" {
		t.Fatal(Current())
	}
}
