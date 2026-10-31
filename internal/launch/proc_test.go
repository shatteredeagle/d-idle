package launch

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func fakeProc(t *testing.T, root string, pid, ppid int, comm, cmdline string) {
	t.Helper()
	d := filepath.Join(root, itoa(pid))
	os.MkdirAll(d, 0o755)
	os.WriteFile(filepath.Join(d, "stat"), []byte(itoa(pid)+" ("+comm+") S "+itoa(ppid)+" 1 1 0\n"), 0o644)
	os.WriteFile(filepath.Join(d, "cmdline"), []byte(cmdline), 0o644)
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestProcParsing(t *testing.T) {
	root := t.TempDir()
	fakeProc(t, root, 10, 1, "app", "app\x00")
	fakeProc(t, root, 11, 10, "python3", "python3\x00proton\x00run\x00")
	fakeProc(t, root, 12, 11, "Game.exe", `C:\Program Files (x86)\Diablo II\Game.exe -w`+"\x00\x00\x00")
	fakeProc(t, root, 13, 1, "a (b) c", "x\x00")
	ps, err := listProcs(root)
	if err != nil || len(ps) != 4 {
		t.Fatalf("procs=%v err=%v", ps, err)
	}
	g := FindImage(ps, "GAME.EXE")
	if len(g) != 1 || g[0].PID != 12 {
		t.Fatalf("FindImage = %v", g)
	}
	if !IsDescendant(ps, 12, 10) || IsDescendant(ps, 13, 10) || IsDescendant(ps, 10, 10) {
		t.Fatal("IsDescendant wrong")
	}
	for _, p := range ps {
		if p.PID == 13 && p.Comm != "a (b) c" {
			t.Fatalf("comm = %q", p.Comm)
		}
	}
}

func TestImageBase(t *testing.T) {
	maps := `00400000-00705000 r-xp 00000000 103:07 1  /g/Game.exe
00705000-0074d000 rwxp 00305000 103:07 1  /g/Game.exe
7f000000-7f001000 r--p 00000000 00:00 0
`
	b, err := imageBaseFrom(strings.NewReader(maps), "game.exe")
	if err != nil || b != 0x400000 {
		t.Fatalf("base=%x err=%v", b, err)
	}
	if _, err := imageBaseFrom(strings.NewReader(maps), "nope.exe"); err == nil {
		t.Fatal("want error")
	}
}
