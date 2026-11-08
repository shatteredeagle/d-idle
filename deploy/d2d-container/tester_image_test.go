package d2dcontainer_test

import (
	"os"
	"os/exec"
	"path/filepath"
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

// TASK-139: every vendored input is a public archive pinned by sha256; no local build dir, home path or private remote.
func TestContextPublicInputs(t *testing.T) {
	c := read(t, "context.sh")
	wantAll(t, "context.sh", c,
		"cli11_url=https://github.com/CLIUtils/CLI11/archive/refs/tags/v2.6.2.tar.gz",
		"cli11_sha=c6ea6b2e5608b3ea8617999bd5f47420c71b2ebdb8dc4767c1034d1da5785711",
		"sentry_url=https://github.com/getsentry/sentry-native/releases/download/0.17.3/sentry-native.zip",
		"sentry_sha=930039d20b8f7d5520c8cb1d9911d63ce8a9e0635b56ae08256936d1f45b66ac",
		"ffurl=https://ffmpeg.org/releases/ffmpeg-9.0.2.tar.xz", "sha256sum -c", "pin_tag=didle-gate2-lab3",
		"https://github.com/shatteredeagle/d2d-didle")
	for _, bad := range []string{"$HOME", "d2d-lab", "_deps", "ffmpeg_build-prefix", "/home/", "d2d-fork", "$FORK/", "${FORK:"} {
		// the only allowed mention of the old private remote is none at all
		if strings.Contains(c, bad) {
			t.Errorf("context.sh must not reference %q (public inputs only)", bad)
		}
	}
	d := read(t, "Dockerfile.d2d")
	for _, bad := range []string{"/home/", "d2d-lab", "$HOME"} {
		if strings.Contains(d, bad) {
			t.Errorf("Dockerfile.d2d must not reference %q", bad)
		}
	}
	// privacy: the image sends no crash reports (empty DSN at build and run time, no opt-in flag anywhere)
	wantAll(t, "Dockerfile.d2d", d, "-DD2D_SENTRY_DSN=", "SENTRY_DSN= \\")
	if strings.Contains(read(t, "entrypoint.sh"), "send-crash-reports") {
		t.Error("entrypoint.sh must not enable --send-crash-reports")
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
		"D2BOT_WATCH_ORIGIN=http://tester.example:8891", "D2BOT_WATCH_TOKEN_FILE=/data/watch.token", "D2BOT_LADDER_STORE=shared",
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

// TASK-25 live defect: `eval d2bot ... &` made $! the eval subshell, so the TERM trap never reached d2bot and a
// container stop killed the character without a save. The background job must exec d2bot.
func TestEntrypointExecsBot(t *testing.T) {
	b, err := os.ReadFile("entrypoint.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "eval exec d2bot") {
		t.Error("entrypoint.sh must start the bot with `eval exec d2bot ... &` so that $! is d2bot")
	}
}

func TestEntrypointStopsBotBeforeTap(t *testing.T) {
	b, err := os.ReadFile("entrypoint.sh")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	i := strings.Index(s, "stop() {")
	j := strings.Index(s[i:], "\n}")
	if i < 0 || j < 0 || !strings.Contains(s[i:i+j], `kill -TERM "$last"`) || strings.Index(s[i:i+j], `wait "$last"`) > strings.Index(s[i:i+j], "for p in $pids") {
		t.Error("stop() must TERM and wait for the bot (last pid) before it stops the tap")
	}
}

// A character saved by the managed d2d (under <data>/instances/<inst>/d2d/config) must be loaded, not recreated.
func TestEntrypointLoadsManagedSave(t *testing.T) {
	data := t.TempDir()
	save := filepath.Join(data, "instances", "main", "d2d", "config", "d2d", "save")
	if err := os.MkdirAll(save, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(save, "Hero.d2s"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := dry(t, "tester", "D2BOT_DATA="+data, "D2D_NAME=Hero", "D2D_CLASS=4")
	if !strings.Contains(out, "--d2d-class -1") {
		t.Errorf("a saved character is not loaded (want --d2d-class -1):\n%s", out)
	}
	if o := dry(t, "tester", "D2BOT_DATA="+t.TempDir(), "D2D_NAME=Hero", "D2D_CLASS=4"); !strings.Contains(o, "--d2d-class 4") {
		t.Errorf("a new character is not created with the wizard class:\n%s", o)
	}
}

// TASK-120: the compose file takes the image from D2_IMAGE and keeps the local build tag as the default.
func TestComposeImageVariable(t *testing.T) {
	wantAll(t, "compose.tester.yml", read(t, "compose.tester.yml"), "image: ${D2_IMAGE:-${D2D_IMAGE:-d2dbot:tester}}")
}

// TASK-120: the publish workflow is inert until a person acts and carries no secret.
func TestTesterImageWorkflow(t *testing.T) {
	w := read(t, "../../.github/workflows/tester-image.yml")
	// triggers: workflow_dispatch only
	on := w[strings.Index(w, "\non:"):strings.Index(w, "\nconcurrency:")]
	if strings.Contains(on, "pull_request") || strings.Contains(on, "push:") || strings.Contains(on, "schedule") || strings.Contains(on, "workflow_run") {
		t.Errorf("tester-image.yml may only be triggered by workflow_dispatch:\n%s", on)
	}
	wantAll(t, "tester-image.yml", w, "workflow_dispatch:", "confirm:", "inputs.confirm == 'publish'", "environment: tester-publish",
		"packages: write", "secrets.GITHUB_TOKEN", "platforms: linux/amd64", "org.opencontainers.image.source=", "org.opencontainers.image.licenses=",
		"org.opencontainers.image.revision=", "context.sh", "push: false", "No Blizzard data")
	// no secret other than GITHUB_TOKEN, no personal token, no raw credential
	for _, m := range regexp.MustCompile(`secrets\.([A-Za-z0-9_]+)`).FindAllStringSubmatch(w, -1) {
		if m[1] != "GITHUB_TOKEN" {
			t.Errorf("tester-image.yml uses secret %s", m[1])
		}
	}
	if regexp.MustCompile(`(?i)(ghp_|github_pat_|password:\s*[^$\s])`).MatchString(w) {
		t.Error("tester-image.yml holds a credential")
	}
	// packages: write only in the publish job, which sits behind the environment
	i := strings.Index(w, "\n  publish:")
	if i < 0 || strings.Contains(w[:i], "packages: write") || !strings.Contains(w[i:], "packages: write") {
		t.Error("packages: write must appear only in the publish job")
	}
	if p := w[i:]; strings.Index(p, "environment: tester-publish") < 0 || strings.Index(p, "environment: tester-publish") > strings.Index(p, "steps:") {
		t.Error("the publish job must declare environment: tester-publish")
	}
	if strings.Contains(w[:i], "push: true") {
		t.Error("the build job must not push")
	}
	// every action is pinned: a 40-hex SHA or a major version, with a comment saying so
	for _, l := range strings.Split(w, "\n") {
		m := regexp.MustCompile(`uses:\s*(\S+)@(\S+)(.*)`).FindStringSubmatch(l)
		if m == nil {
			continue
		}
		if !regexp.MustCompile(`^([0-9a-f]{40}|v[0-9]+)$`).MatchString(m[2]) || !strings.Contains(m[3], "#") {
			t.Errorf("action not pinned by SHA or major with a comment: %s", strings.TrimSpace(l))
		}
	}
}
