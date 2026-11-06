package d2dcontainer_test

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func wantAll(t *testing.T, name, text string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(text, w) {
			t.Errorf("%s lacks %q", name, w)
		}
	}
}

// The static pins of the tester image: ports, read-only game data, no secret in the file, loopback publish, grace.
func TestComposeTester(t *testing.T) {
	c := read(t, "compose.tester.yml")
	wantAll(t, "compose.tester.yml", c,
		`:/game:ro`, `d2dbot-data:/data`, `"${D2_PUBLISH:-127.0.0.1}:8891:8891"`, `"${D2_PUBLISH:-127.0.0.1}:8892:8892"`,
		`stop_grace_period:`, `D2BOT_WATCH_ORIGIN: ${D2_WATCH_ORIGIN:?`, `TESTER: "1"`, `D2D_MODE: ${D2D_MODE:-realtime}`, `cap_drop: [ALL]`)
	for _, bad := range []string{"8893", "8420", ".mpq", "Game.exe", "D2DVIEW_TOKEN", "D2BOT_WATCH_TOKEN:"} {
		if strings.Contains(c, bad) {
			t.Errorf("compose.tester.yml must not mention %q (tap and bot API stay inside; no token in the file)", bad)
		}
	}
	if regexp.MustCompile(`(?mi)^\s*[A-Z_]*TOKEN[A-Z_]*:\s*\S`).MatchString(c) {
		t.Error("a token value is set in compose.tester.yml")
	}
}

func TestDockerfileTester(t *testing.T) {
	d := read(t, "Dockerfile.d2d")
	wantAll(t, "Dockerfile.d2d", d, "EXPOSE 8891 8892", `"/opt/d2lod/entrypoint.sh"`, "D2D_DATA=/game", "VOLUME /data")
	if regexp.MustCompile(`(?m)^\s*D2D_MODE=`).MatchString(d) {
		t.Error("Dockerfile.d2d must not set D2D_MODE: it would override the tester default (realtime)")
	}
	if regexp.MustCompile(`(?mi)^(COPY|ADD) .*\.(mpq|exe)`).MatchString(d) {
		t.Error("Dockerfile copies game data")
	}
}

// context.sh pins the Gate 2 fork commit and refuses game data.
func TestContextPins(t *testing.T) {
	c := read(t, "context.sh")
	wantAll(t, "context.sh", c, "pin_commit=f75816c616c8253930d88277b90936e3eabe3bc8", "pin_ref=lab3-labverbs", "./cmd/d2dsetup",
		"8c3850283eb25fa026482078a04051e0be17347b09ef81a0849bec15a96e002e", "-iname '*.mpq'", "-iname 'game.exe'", "*.mpq")
	if strings.Contains(c, "lab5") {
		t.Error("context.sh still names lab5")
	}
}

// Run the entrypoint dry run: the resolved tester profile, tokens masked; the default bot mode unchanged.
func dry(t *testing.T, mode string, env ...string) string {
	t.Helper()
	cmd := exec.Command("sh", "entrypoint.sh", mode)
	cmd.Env = append([]string{"PATH=" + os.Getenv("PATH"), "ENTRYPOINT_DRYRUN=1", "XDG_CONFIG_HOME=" + t.TempDir()}, env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("entrypoint dry run: %v\n%s", err, out)
	}
	return string(out)
}

func TestEntrypointTesterProfile(t *testing.T) {
	out := dry(t, "tester", "D2BOT_WATCH_ORIGIN=http://tester.example:8891")
	wantAll(t, "tester dry run", out, "D2BOT_WATCH_ACCESS_FILE=/data/access.json", "D2BOT_LADDER=1", "D2BOT_WATCH=1", "D2BOT_WATCH_ADDR=0.0.0.0:8891", "D2BOT_WATCH_ANSWER_ADDR=0.0.0.0:8892",
		"D2BOT_WATCH_ORIGIN=http://tester.example:8891", "D2BOT_WATCH_TOKEN_FILE=/data/watch.token", "D2BOT_LADDER_STORE=/data/ladder",
		"D2BOT_WATCH_VIDEO=1", "D2BOT_WATCH_VIDEO_SOURCE=tap", "D2BOT_GAMEDIR=/game", "d2d-mode=realtime", "D2DVIEW_TOKEN=***",
		"d2dview -addr 127.0.0.1:8893", "-jpeg", "--addr 127.0.0.1:8420", "--goals 'campaign:normal_act1'")
	if regexp.MustCompile(`TOKEN=[0-9a-f]{8}`).MatchString(out) {
		t.Errorf("token printed:\n%s", out)
	}
	if strings.Contains(out, "--lockstep") || strings.Contains(out, "--lab") {
		t.Errorf("tester profile passes a lockstep or lab flag:\n%s", out)
	}
	if o := dry(t, "tester", "D2D_MODE=lockstep"); !strings.Contains(o, "d2d-mode=lockstep") || !strings.Contains(o, "--d2d-args '--lockstep'") {
		t.Errorf("D2D_MODE=lockstep is not honoured:\n%s", o)
	}
	if o := dry(t, "bot", "TESTER=1"); !strings.Contains(o, "D2BOT_LADDER=1") {
		t.Errorf("TESTER=1 does not select the tester profile:\n%s", o)
	}
}

func TestEntrypointBotDefaultUnchanged(t *testing.T) {
	out := dry(t, "bot")
	wantAll(t, "bot dry run", out, "d2d-mode=lockstep", "--d2d-args '--lockstep'", "tester=0")
	for _, no := range []string{"D2BOT_LADDER", "D2BOT_WATCH", "D2BOT_GAMEDIR", "-jpeg", "D2DVIEW_TOKEN"} {
		if strings.Contains(out, no) {
			t.Errorf("default bot mode got tester settings (%s):\n%s", no, out)
		}
	}
}
