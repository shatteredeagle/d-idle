package d2dsrc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
)

// script is a tiny d2d stand-in: exact command line -> raw reply (payload lines then status). It exists
// because internal/d2dctl/fake has no "dump" verb (not our directory); every other verb reuses
// fake.Sample() payloads. A line with no entry answers "err unknown verb".
type script struct {
	mu    sync.Mutex
	reply map[string]string
	seen  []string
}

func sampleReplies() map[string]string {
	f := fake.Sample()
	pl := func(ls []string) string { return strings.Join(ls, "\n") + "\n" }
	return map[string]string{
		"info":          "w=800 h=600 frame=100\nok\n",
		"state":         f.State + "\nok\n",
		"monsters":      pl(f.Monsters) + "ok\n",
		"npcs":          pl(f.NPCs) + "ok\n",
		"ground":        pl(f.Ground) + "ok\n",
		"menu":          pl(f.Menu) + "ok\n",
		"items":         pl(f.Items) + "ok\n",
		"debug players": pl(f.Players) + "ok\n",
		"debug portals": pl(f.Portals) + "ok\n",
		"debug corpses": pl(f.Corpses) + "ok\n",
		"debug warps":   pl(f.Warps) + "ok\n",
		"debug level":   "ok " + f.Level + "\n",
	}
}

func startScript(t *testing.T, reply map[string]string) (*script, *d2dctl.Client) {
	t.Helper()
	dir, err := os.MkdirTemp("", "d2dsrc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "d.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	s := &script{reply: reply}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				rd := bufio.NewReader(c)
				for {
					line, err := rd.ReadString('\n')
					if err != nil {
						return
					}
					line = strings.TrimSpace(line)
					s.mu.Lock()
					s.seen = append(s.seen, line)
					r, ok := s.reply[line]
					s.mu.Unlock()
					if !ok {
						r = "err unknown verb '" + strings.Fields(line)[0] + "'\n"
					}
					c.Write([]byte(r))
				}
			}()
		}
	}()
	t.Cleanup(func() { ln.Close() })
	cl := d2dctl.New(path, d2dctl.Options{})
	t.Cleanup(func() { cl.Close() })
	return s, cl
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer // the channel carries one JSON line
	if err := json.Compact(&out, b); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

const headSections = "dump player,stats,skills,ui,merc,time"

// fakeTables is a Tables with a few rows, enough for the fixtures.
type fakeTables struct{}

func (fakeTables) ItemRow(code string) (int, bool) {
	n, ok := map[string]int{"wnd": 100, "hp1": 587, "tsc": 530, "cap": 200, "jav": 300, "gld": 523}[code]
	return n, ok
}
func (fakeTables) MonsterClass(name string) (int, bool) {
	n, ok := map[string]int{"fallen1": 2, "fallens1": 3, "Kashya": 150}[name]
	return n, ok
}
