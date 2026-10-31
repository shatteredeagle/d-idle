package launch

import (
	"os"
	"testing"
)

// The tests that spawn through a fake System or build commands must not depend on bubblewrap being installed on the
// machine that runs them; the sandbox tests set D2_SANDBOX themselves.
func TestMain(m *testing.M) {
	os.Setenv("D2_SANDBOX", "0")
	os.Exit(m.Run())
}
