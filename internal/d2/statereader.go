package d2

import "errors"

// ErrUnsupported is returned by a StateReader that cannot answer a query by design (for example address based reads on a
// source that has no process memory). It is never a fake value: callers must treat it as "unknown".
var ErrUnsupported = errors.New("d2: not available from this state source")

// StateReader is the read side the bot calls on env.Env.Read: the whole-state Snapshot plus the few address based reads
// that decision code still does itself. *Reader (process memory) and d2dsrc.Source (d2d) implement it. Addresses
// (Unit.Addr, Player.Addr, stat list pointers) only mean something to a memory-backed reader.
type StateReader interface {
	Snapshot() (*Snapshot, error)
	Player() (*Player, error)
	UI() (UIState, error)
	Units(typ int) ([]Unit, error)
	UnitStats(unit uint32) (Stats, error)
	ReadStatList(sl uint32) (Stats, error)
	ReadBaseStatList(sl uint32) (Stats, error)
	RawStatList(unit uint32, n int) ([]byte, error)
}

var _ StateReader = (*Reader)(nil)
