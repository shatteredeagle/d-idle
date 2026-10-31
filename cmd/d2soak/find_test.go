package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func fakeProc(t *testing.T, root string, pid int, comm, cmd, env string) {
	d := filepath.Join(root, strconv.Itoa(pid))
	os.MkdirAll(d, 0o755)
	os.WriteFile(filepath.Join(d, "comm"), []byte(comm+"\n"), 0o644)
	os.WriteFile(filepath.Join(d, "cmdline"), []byte(cmd), 0o644)
	os.WriteFile(filepath.Join(d, "stat"), []byte(strconv.Itoa(pid)+" ("+comm+") S 1 1 1 0 0 0 0 0 0 0 1 1 0 0 20 0 3 0 777 0 0\n"), 0o644)
	if env != "" {
		os.WriteFile(filepath.Join(d, "environ"), []byte(env), 0o644)
	}
}

func TestFindPerInstance(t *testing.T) {
	root := t.TempDir()
	fakeProc(t, root, 100, "Game.exe", "Game.exe\x00-w\x00", "D2BOT_INSTANCE=i1\x00DISPLAY=:101\x00")
	fakeProc(t, root, 200, "Game.exe", "Game.exe\x00-w\x00", "D2BOT_INSTANCE=i4\x00DISPLAY=:104\x00")
	fakeProc(t, root, 150, "wineserver", "wineserver\x00", "D2BOT_INSTANCE=i4\x00")
	fakeProc(t, root, 300, "d2bot", "timeout\x00d2bot\x00--instance\x00i4\x00", "")
	fakeProc(t, root, 90, "bash", "bash -c d2bot --instance i4", "")
	fakeProc(t, root, 400, "d2input.exe", "d2input.exe\x00-addr\x00127.0.0.1:47016\x00", "")
	fakeProc(t, root, 410, "d2input.exe", "d2input.exe\x00-addr\x00127.0.0.1:47013\x00", "")
	fakeProc(t, root, 500, "Xvfb", "Xvfb\x00:104\x00-screen\x00", "")
	fakeProc(t, root, 510, "Xvfb", "Xvfb\x00:101\x00-screen\x00", "")
	cases := []struct {
		spec string
		want int
	}{
		{"find:comm=Game.exe;env=D2BOT_INSTANCE=i4", 200},
		{"find:comm=Game.exe;env=D2BOT_INSTANCE=i1", 100},
		{"find:comm=Game.exe;env=D2BOT_INSTANCE=i9", 0},
		{"find:comm=wineserver;env=D2BOT_INSTANCE=i4", 150},
		{"find:comm=wineserver;env=D2BOT_INSTANCE=i1", 0},
		{"find:comm=d2bot;cmd=--instance i4", 300},
		{"find:comm=d2input.exe;cmd=-addr 127.0.0.1:47013", 410},
		{"find:comm=Xvfb;cmd=Xvfb :104 ", 500},
		{"find:comm=Game.exe", 100},
	}
	for _, c := range cases {
		f, err := parseFind(c.spec[len("find:"):])
		if err != nil {
			t.Fatal(err)
		}
		if got := findPID(root, f, 0); got != c.want {
			t.Errorf("%s = %d, want %d", c.spec, got, c.want)
		}
	}
	f, _ := parseFind("comm=Game.exe")
	if got := findPID(root, f, 100); got != 200 {
		t.Errorf("skip = %d", got)
	}
	c := &Collector{ProcRoot: root, Procs: []ProcSource{{"game", "find:comm=Game.exe;env=D2BOT_INSTANCE=i4"}}}
	if p := c.procs(); p["game"].PID != 0 && p["game"].PID != 200 {
		t.Errorf("collector pid %d", p["game"].PID)
	}
	if _, err := parseFind("bogus=1"); err == nil {
		t.Error("want error")
	}
}

func TestPrintProcs(t *testing.T) {
	root := t.TempDir()
	fakeProc(t, root, 200, "Game.exe", "Game.exe\x00-w\x00", "D2BOT_INSTANCE=i4\x00")
	ps := []ProcSource{{"game", "find:comm=Game.exe;env=D2BOT_INSTANCE=i4"}, {"bot", "find:comm=d2bot"}}
	var b strings.Builder
	if rc := printProcs(&b, &Collector{ProcRoot: root, Procs: ps}, ps); rc != 1 {
		t.Errorf("rc %d", rc)
	}
	if !strings.Contains(b.String(), "pid 200") || !strings.Contains(b.String(), "bot        MISSING") {
		t.Errorf("out %q", b.String())
	}
}
