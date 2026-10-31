package sentinel

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"d2lod-bot/internal/launch"
)

// recSys is a launch.System that only records; no game ever appears.
type recSys struct{ calls []string }

func (r *recSys) Procs() ([]launch.Proc, error)        { return nil, nil }
func (r *recSys) Marker(int) string                    { return "" }
func (r *recSys) Signal(int, syscall.Signal) error     { return nil }
func (r *recSys) Exists(string) bool                   { return true }
func (r *recSys) HelperUp(int) bool                    { return true }
func (r *recSys) Sleep(context.Context, time.Duration) {}
func (r *recSys) Run(_ context.Context, _ []string, _, name string, a ...string) ([]byte, error) {
	r.calls = append(r.calls, "run "+name+" "+strings.Join(a, " "))
	return nil, nil
}
func (r *recSys) Spawn(_ []string, _, _, name string, a ...string) error {
	r.calls = append(r.calls, "spawn "+name+" "+strings.Join(a, " "))
	return nil
}

// The restart steps go through launch.Life, never through game.sh (Cfg.GameSh is ignored).
func TestLifeStepUsesLaunch(t *testing.T) {
	dir := t.TempDir()
	proton := filepath.Join(dir, "proton")
	os.WriteFile(proton, nil, 0o755)
	rs := &recSys{}
	a := &LiveActor{Cfg: Config{GameSh: "/nonexistent/game.sh"}, Life: &launch.Life{
		Sys: rs, HelperWait: 1, StartWait: 1, Environ: []string{},
		Base: launch.Config{Root: filepath.Join(dir, "d2lod"), SteamRoot: dir, ProtonScript: proton}}}
	r := a.lifeStep(context.Background(), "stop", "i5")
	if !r.OK || !strings.Contains(r.Detail, "launch stop i5: i5 not running") {
		t.Fatalf("%+v", r)
	}
	if len(rs.calls) != 1 || !strings.HasSuffix(rs.calls[0], "files/bin/wineserver -k") {
		t.Fatalf("calls %v", rs.calls)
	}
	r = a.lifeStep(context.Background(), "helper", "i5")
	if !r.OK || !strings.Contains(r.Detail, "listening on 47017") {
		t.Fatalf("%+v", r)
	}
	if r := a.lifeStep(context.Background(), "bogus", "i5"); r.OK {
		t.Fatal("unknown step must fail")
	}
}
