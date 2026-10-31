package setup

import (
	"context"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
)

// Env is every host effect a check performs, so checks run offline against fakes. NewEnv returns the
// real one.
type Env struct {
	GOOS      string
	NumCPU    int
	Getenv    func(string) string
	ReadFile  func(string) ([]byte, error)
	Stat      func(string) (fs.FileInfo, error)
	ReadDir   func(string) ([]fs.DirEntry, error)
	LookPath  func(string) (string, error)
	Run       func(ctx context.Context, name string, args ...string) ([]byte, error)
	FreeBytes func(path string) (uint64, error)
	// Listen tries to bind addr and releases it at once; nil error means the port is free.
	Listen func(addr string) error
	// Probe runs the subreaper/memory-read probe in a child process (see probe.go).
	Probe func(ctx context.Context) (ProbeResult, error)
	// GameVersion returns the file version of a PE; SHA256 the hex digest of a file.
	GameVersion func(path string) (string, error)
	SHA256      func(path string) (string, error)
	// TryExtract proves the data tables can be read out of the game dir (writes to a temp dir and removes it).
	TryExtract func(gameDir string) error
	// Dial connects to addr and closes at once; nil error means something listens there.
	Dial func(addr string) error
	// Now is the wall clock.
	Now func() time.Time
	// Player reads the character of a running game (instance "" = the D2BOT_INSTANCE one) for d2.DoctorLine;
	// an error means no game is attached.
	Player func(instance string) (*d2.Player, error)
	// RunnerDirs are searched for Proton builds (compatibilitytools.d).
	RunnerDirs []string
}

// NewEnv is the real host.
func NewEnv() *Env {
	home, _ := os.UserHomeDir()
	c := launch.Config{SteamRoot: filepath.Join(home, ".local", "share", "Steam")}
	dirs := []string{
		filepath.Join(c.SteamRoot, "compatibilitytools.d"),
		filepath.Join(home, ".steam", "root", "compatibilitytools.d"),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam", "compatibilitytools.d"),
		"/usr/share/steam/compatibilitytools.d",
	}
	return &Env{
		GOOS:     runtime.GOOS,
		NumCPU:   runtime.NumCPU(),
		Getenv:   os.Getenv,
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		ReadDir:  os.ReadDir,
		LookPath: exec.LookPath,
		Run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		},
		FreeBytes: func(p string) (uint64, error) {
			var s syscall.Statfs_t
			if err := syscall.Statfs(p, &s); err != nil {
				return 0, err
			}
			return s.Bavail * uint64(s.Bsize), nil
		},
		Listen: func(addr string) error {
			l, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}
			return l.Close()
		},
		Probe: RunProbeChild,
		GameVersion: func(p string) (string, error) {
			v, err := launch.FileVersion(p)
			return v.String(), err
		},
		SHA256: sha256File,
		TryExtract: func(gameDir string) error {
			tmp, err := os.MkdirTemp("", "d2doctor-extract-")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmp)
			return data.Extract(gameDir, tmp)
		},
		Dial: func(addr string) error {
			c, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
			if err != nil {
				return err
			}
			return c.Close()
		},
		Now: time.Now,
		Player: func(instance string) (*d2.Player, error) {
			g, err := mem.OpenInstance(instance)
			if err != nil {
				return nil, err
			}
			defer g.Close()
			return d2.New(g).Player()
		},
		RunnerDirs: dirs,
	}
}
