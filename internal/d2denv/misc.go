package d2denv

import (
	"errors"
	"fmt"

	"d2lod-bot/internal/d2"
)

var errNoSock = errors.New("d2denv: no control socket (--d2d-sock / D2D_SOCK)")

func sprintf(f string, a ...any) string { return fmt.Sprintf(f, a...) }

// noMem is env.Env.Mem for a backend without process memory: every read fails, so the few places that still try
// (town's page scan, leveler base stats, flow checks) take their "unreadable" path instead of dereferencing nil.
type noMem struct{}

func (noMem) ReadAt([]byte, uint32) error { return d2.ErrUnsupported }
