// Package geartest lets tests of other packages skip when the gear tables are not extracted (CI, public forks).
package geartest

import (
	"testing"

	"d2lod-bot/internal/gear"
)

// Need skips t unless the real gear tables are available (run cmd/d2tables or set D2_DATA_DIR).
func Need(t testing.TB) {
	t.Helper()
	if err := gear.Available(); err != nil {
		t.Skip(err)
	}
}
