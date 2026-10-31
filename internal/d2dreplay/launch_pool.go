package d2dreplay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Pool is the d2d slot pool geometry. tools/d2dslot.sh is the single source: it writes
// ~/d2d-lab/slots/pool.json on every call; D2D_SLOTS overrides the size (same clamp as the script).
type Pool struct {
	Slots     int `json:"slots"`
	FreeSlots int `json:"free_slots"` // slots below this are admission-free
	BotBase   int `json:"bot_base"`
	ViewBase  int `json:"view_base"`
}

// MaxSlots is the hard ceiling, same as MAXSLOTS in tools/d2dslot.sh.
const MaxSlots = 12

// DefaultPoolFile is where d2dslot.sh publishes the pool.
func DefaultPoolFile() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "d2d-lab", "slots", "pool.json")
}

// LoadPool reads path (missing file: the built-in defaults, matching the script) and applies D2D_SLOTS.
func LoadPool(path string) (Pool, error) {
	p := Pool{Slots: 4, FreeSlots: 4, BotBase: 9100, ViewBase: 9200}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &p); err != nil {
			return p, fmt.Errorf("d2dreplay: %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return p, err
	}
	if v := os.Getenv("D2D_SLOTS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return p, fmt.Errorf("d2dreplay: bad D2D_SLOTS %q", v)
		}
		p.Slots = n
	}
	p.Slots = min(max(p.Slots, 1), MaxSlots)
	return p, nil
}

// LockFile is slot n's lock; slot 0 is the legacy lab lock.
func (p Pool) LockFile(n int) string {
	if n == 0 {
		return LabLock
	}
	return fmt.Sprintf("/tmp/d2d-lab-slot-%d.lock", n)
}

func (p Pool) BotAddr(n int) string { return fmt.Sprintf("127.0.0.1:%d", p.BotBase+n) }
func (p Pool) ViewPort(n int) int   { return p.ViewBase + n }

// NeedsAdmission reports whether taking slot n is subject to the host check done by d2dslot.sh.
func (p Pool) NeedsAdmission(n int) bool { return n >= p.FreeSlots }
