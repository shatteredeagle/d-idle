package container_test

import (
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/fleet"
)

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Every `COPY bin/X` in the Dockerfile must be built to $CTX/bin/X by tools/container-build.sh.
func TestDockerfileBinariesAreBuilt(t *testing.T) {
	docker := read(t, "Dockerfile")
	build := read(t, "../../tools/container-build.sh")
	copies := regexp.MustCompile(`(?m)^COPY bin/(\S+)`).FindAllStringSubmatch(docker, -1)
	if len(copies) < 6 {
		t.Fatalf("expected at least 6 COPY bin/ lines, got %d", len(copies))
	}
	for _, m := range copies {
		if !strings.Contains(build, "$CTX/bin/"+m[1]+" ") {
			t.Errorf("Dockerfile copies bin/%s but container-build.sh does not build it", m[1])
		}
	}
	for _, need := range []string{"d2fleet", "d2doctor", "d2soak", "d2bot", "d2container", "d2input.exe"} {
		if !strings.Contains(docker, "COPY bin/"+need+" ") {
			t.Errorf("Dockerfile lacks %s", need)
		}
	}
}

// Every other COPY source (scripts, runner) must exist in the build context.
func TestDockerfileContextFiles(t *testing.T) {
	for _, m := range regexp.MustCompile(`(?m)^COPY (?:--\S+ )*([^b]\S*) `).FindAllStringSubmatch(read(t, "Dockerfile"), -1) {
		if _, err := os.Stat(m[1]); err != nil {
			t.Errorf("COPY source %s: %v", m[1], err)
		}
	}
}

func TestWizardDefaults(t *testing.T) {
	docker := read(t, "Dockerfile")
	for _, s := range []string{"USER steamuser", "useradd -m -u 1000 steamuser", "EXPOSE 8431", `ENTRYPOINT ["/usr/local/bin/d2container"]`, "d2start.sh"} {
		if !strings.Contains(docker, s) {
			t.Errorf("Dockerfile lacks %q", s)
		}
	}
	if strings.Count(docker, "EXPOSE") != 1 {
		t.Error("only port 8431 may be exposed")
	}
	start := read(t, "d2start.sh")
	for _, s := range []string{"serve -setup", "8431", "-token-file", "umask 077"} {
		if !strings.Contains(start, s) {
			t.Errorf("d2start.sh lacks %q", s)
		}
	}
	if c := read(t, "compose.example.yml"); !strings.Contains(c, "stop_grace_period: 90s") || !strings.Contains(c, "127.0.0.1:8431:8431") {
		t.Error("compose.example.yml needs stop_grace_period 90s and the loopback 8431 mapping")
	}
}

// The whole stop chain must save the character: docker stop > d2container wait > d2fleet stop grace > d2bot worst case.
func TestStopGraceChain(t *testing.T) {
	m := regexp.MustCompile(`stop_grace_period:\s*(\d+)s`).FindStringSubmatch(read(t, "compose.example.yml"))
	if m == nil {
		t.Fatal("compose.example.yml has no stop_grace_period")
	}
	n, _ := strconv.Atoi(m[1])
	if got := time.Duration(n) * time.Second; got != fleet.ComposeStopGrace {
		t.Errorf("compose stop_grace_period %v != fleet.ComposeStopGrace %v", got, fleet.ComposeStopGrace)
	}
	if !(fleet.ComposeStopGrace > fleet.ContainerGrace && fleet.ContainerGrace > fleet.DefaultStopGrace && fleet.DefaultStopGrace > bot.ShutdownWorstCase) {
		t.Errorf("grace chain broken: compose %v > container %v > fleet %v > d2bot %v required",
			fleet.ComposeStopGrace, fleet.ContainerGrace, fleet.DefaultStopGrace, bot.ShutdownWorstCase)
	}
	if bot.ShutdownWorstCase > 50*time.Second {
		t.Errorf("d2bot worst case %v above the 50 s budget", bot.ShutdownWorstCase)
	}
}

// D2FLEET_SOAK=1 adds the supervised sidecar and -soak-dir; without it nothing changes. Stub d2fleet/d2soak record their args.
func TestStartSoakSidecar(t *testing.T) {
	for _, on := range []bool{false, true} {
		bin, data := t.TempDir(), t.TempDir()
		os.WriteFile(bin+"/d2fleet", []byte("#!/bin/sh\necho \"$@\" > "+data+"/fleet.args\nsleep 1\n"), 0o755)
		os.WriteFile(bin+"/d2soak", []byte("#!/bin/sh\necho \"$@\" >> "+data+"/soak.args\nsleep 1\n"), 0o755)
		cmd := exec.Command("sh", "d2start.sh")
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "D2BOT_DATA="+data, "D2FLEET_ADDR=0.0.0.0:9999")
		if on {
			cmd.Env = append(cmd.Env, "D2FLEET_SOAK=1")
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("d2start: %v\n%s", err, out)
		}
		fa := read(t, data+"/fleet.args")
		time.Sleep(300 * time.Millisecond)
		sa, err := os.ReadFile(data + "/soak.args")
		if !on {
			if strings.Contains(fa, "-soak-dir") || err == nil {
				t.Errorf("soak off: fleet args %q, sidecar started=%v", fa, err == nil)
			}
			continue
		}
		if !strings.Contains(fa, "-soak-dir "+data+"/soak") {
			t.Errorf("fleet args lack -soak-dir: %q", fa)
		}
		for _, w := range []string{"sidecar", "-fleet http://127.0.0.1:9999", "-out " + data + "/soak", "-profile m24", "-parent"} {
			if !strings.Contains(string(sa), w) {
				t.Errorf("sidecar args lack %q: %q", w, sa)
			}
		}
	}
}
