package gear

import "testing"

// needTables skips a test that asserts values of the real game tables when they are not extracted (CI, public forks).
// Run d2tables, or set D2_DATA_DIR, to run them; fixture_test.go covers the extractor with made-up rows.
func needTables(t *testing.T) {
	t.Helper()
	if err := Available(); err != nil {
		t.Skip(err)
	}
}
