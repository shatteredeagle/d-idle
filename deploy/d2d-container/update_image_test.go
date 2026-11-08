package d2dcontainer_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TASK-123: the image carries the boot shim and the public keys, starts through the shim, and stays rebuildable.
func TestDockerfileBootShim(t *testing.T) {
	d := read(t, "Dockerfile.d2d")
	wantAll(t, "Dockerfile.d2d", d,
		"COPY bin/didleboot /opt/didle/boot", "COPY pubkeys/ /opt/didle/pubkeys/",
		`ENTRYPOINT ["/usr/bin/tini", "--", "/opt/didle/boot", "--launcher", "/opt/d2lod/entrypoint.sh"]`,
		`LABEL io.didle.image.abi="1"`, "D2BOT_IMAGE_ABI=1", "DIDLE_BASE_VERSION=$APP_VERSION", "ARG APP_VERSION")
	// the shim sits under tini (signal forwarding, zombie reaping), never the other way round
	if !regexp.MustCompile(`ENTRYPOINT \["/usr/bin/tini", "--", "/opt/didle/boot"`).MatchString(d) {
		t.Error("the shim must run under tini")
	}
	// the label and the ENV carry the same ABI level
	label := regexp.MustCompile(`io\.didle\.image\.abi="(\d+)"`).FindStringSubmatch(d)
	env := regexp.MustCompile(`D2BOT_IMAGE_ABI=(\d+)`).FindStringSubmatch(d)
	if label == nil || env == nil || label[1] != env[1] {
		t.Errorf("image ABI label %v and ENV %v differ", label, env)
	}
}

func TestContextBuildsShimAndKeys(t *testing.T) {
	c := read(t, "context.sh")
	wantAll(t, "context.sh", c, "./cmd/didleboot", `"$ctx/pubkeys"`, "deploy/d2d-container/pubkeys/*.pub")
}

// The private key is never in the repo: the only key files allowed under deploy/ are public ones.
func TestNoPrivateKeyInDeploy(t *testing.T) {
	err := filepath.WalkDir("..", func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(p, ".key") {
			t.Errorf("%s: a .key file in the repo tree", p)
		}
		if strings.HasSuffix(p, ".pub") {
			b, _ := os.ReadFile(p)
			if !strings.HasPrefix(string(b), "d2sign-ed25519-public-v1\n") {
				t.Errorf("%s is not a d2sign public key file", p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The entrypoint stays the launcher of the app layer: no release logic in it (design 3.9).
func TestEntrypointStaysALauncher(t *testing.T) {
	e := read(t, "entrypoint.sh")
	for _, no := range []string{"didle/releases", "state.json", "MANIFEST", "d2sign"} {
		if strings.Contains(e, no) {
			t.Errorf("entrypoint.sh mentions %q: release logic belongs in the shim", no)
		}
	}
}

// A downloaded release must be able to bring its own launcher, so the release builder ships it.
func TestReleaseBuilderShipsLauncher(t *testing.T) {
	b := read(t, "../../tools/release/app-assets.sh")
	wantAll(t, "app-assets.sh", b, "deploy/d2d-container/entrypoint.sh", "bin/entrypoint.sh")
}
