package d2dinput

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/input"
)

func TestArgs(t *testing.T) {
	p := &Process{Sock: "/s/d.sock", Data: "/data", ExtraArgs: []string{"--log-level", "warn"}}
	cases := []struct {
		s    Spec
		want string
	}{
		{Spec{Name: "Pal", Class: 2, Seed: 3},
			"--data /data --headless --devctl /s/d.sock --no-video --seed 3 --start-screen ingame --start-class 2 --start-name Pal --log-level warn"},
		{Spec{Name: "Hc", Class: 1, Hardcore: true, Classic: true, NoSave: true},
			"--data /data --headless --devctl /s/d.sock --no-video --no-save --start-screen ingame --start-class 1 --start-name Hc --start-hardcore --start-classic --log-level warn"},
		{Spec{Name: "Old", Class: -1},
			"--data /data --headless --devctl /s/d.sock --no-video --start-screen charselect --start-name Old --log-level warn"},
	}
	for _, tc := range cases {
		got, err := p.Args(tc.s)
		if err != nil || strings.Join(got, " ") != tc.want {
			t.Errorf("%+v\n got %q %v\nwant %q", tc.s, strings.Join(got, " "), err, tc.want)
		}
	}
	if _, err := p.Args(Spec{Name: "x", Difficulty: 1}); !errors.Is(err, input.ErrUnsupported) {
		t.Errorf("difficulty: %v", err)
	}
	if _, err := p.Args(Spec{}); err == nil {
		t.Error("empty name accepted")
	}
}

func TestStartWithoutBinary(t *testing.T) {
	t.Setenv("D2D_BIN", "")
	p := &Process{Sock: "/tmp/x.sock"}
	if err := p.Start(context.Background(), Spec{Name: "a", Class: 1}); !errors.Is(err, ErrNoBinary) {
		t.Fatal(err)
	}
}

// A python stand-in for d2d (our own test code): it serves ping/state/save/quit on the --devctl socket.
const fakeD2D = `#!/usr/bin/env python3
import socket, sys, os
a = sys.argv[1:]
sock = a[a.index("--devctl") + 1]
open(sock + ".args", "w").write(" ".join(a))
s = socket.socket(socket.AF_UNIX); s.bind(sock); s.listen(1)
loaded = "charselect" not in a
while True:
    c, _ = s.accept(); f = c.makefile("rw")
    for line in f:
        v = line.split()
        if not v: continue
        if v[0] == "state":
            f.write("screen=%s name=X\nok\n" % ("ingame" if loaded else "charselect")); 
        elif v[0] == "key" and v[1] == "Return":
            loaded = True; f.write("ok\n")
        elif v[0] == "quit":
            f.write("ok\n"); f.flush(); os._exit(0)
        else:
            f.write("ok\n")
        f.flush()
`

func TestProcessLifecycle(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("no python3")
	}
	dir, err := os.MkdirTemp("", "d2dl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	bin := dir + "/d2d"
	if err := os.WriteFile(bin, []byte(fakeD2D), 0o755); err != nil {
		t.Fatal(err)
	}
	p := &Process{Bin: bin, Sock: dir + "/s.sock"}
	ctx := context.Background()
	if err := p.Start(ctx, Spec{Name: "Old", Class: -1}); err != nil {
		t.Fatal(err)
	}
	defer p.Stop()
	if err := p.WaitInGame(ctx, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if p.Controller() == nil {
		t.Fatal("no controller")
	}
	if err := p.SaveExit(ctx); err != nil {
		t.Fatal(err)
	}
	args, _ := os.ReadFile(dir + "/s.sock.args")
	if !strings.Contains(string(args), "--start-screen charselect --start-name Old") {
		t.Fatalf("args %q", args)
	}
	if p.Controller() != nil {
		t.Fatal("controller kept after exit")
	}
	// Restart as a new character.
	if err := p.Restart(ctx, Spec{Name: "New", Class: 2}, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
}
