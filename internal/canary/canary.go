// Package canary validates a mem.Source against the invariants of docs/d2-1.14d-memory.md
// section 6 before anything trusts the offsets. It works on live processes, Snapshots, replay
// Sources and d2map Images alike. A check whose memory is not present (partial fixtures) is
// Skipped, never silently passed; a present-but-wrong value is a Fail.
package canary

import (
	"errors"
	"fmt"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/mem"
)

type Status string

const (
	Pass Status = "pass"
	Fail Status = "fail"
	Skip Status = "skip"
)

// Result is the outcome of one check. Tag is the verification state of the rule itself
// (V verified live, S secondary, U unverified) so a Fail on a U rule reads as "rule may be wrong".
type Result struct {
	Name   string
	Status Status
	Tag    string
	Detail string
}

func (r Result) String() string {
	return fmt.Sprintf("%-4s %-22s [%s] %s", r.Status, r.Name, r.Tag, r.Detail)
}

// Failed returns the failing results.
func Failed(rs []Result) []Result {
	var out []Result
	for _, r := range rs {
		if r.Status == Fail {
			out = append(out, r)
		}
	}
	return out
}

// Gate returns an error naming every failed check, or nil. Use it as the start gate.
func Gate(src mem.Source) error {
	f := Failed(Check(src))
	if len(f) == 0 {
		return nil
	}
	parts := make([]string, len(f))
	for i, r := range f {
		parts[i] = r.Name + ": " + r.Detail
	}
	return fmt.Errorf("canary failed: %s", strings.Join(parts, "; "))
}

// RVAs mirrored from internal/d2/offsets.go (unexported there); TestCommittedFixtures fails if they drift.
const (
	rvaPlayerUnit   = 0x3A6A70
	rvaFirstControl = 0x3D55BC
	rvaServerUnits  = 0x3A5E70
	rvaClientUnits  = 0x3A5270
	tableStride     = 0x200 // 128 buckets x 4 bytes (V docs/mem.md Q2: type*0x200 + base)
	numTables       = 6
	maxChain        = 64
)

// unmapped reports "this memory is not in the source". d2map.Image does not wrap mem.ErrUnmapped,
// hence the text match.
func unmapped(err error) bool {
	return errors.Is(err, mem.ErrUnmapped) || errors.Is(err, mem.ErrShort) || strings.Contains(err.Error(), "unmapped")
}

// Check runs every invariant and returns one Result per check, in a fixed order.
func Check(src mem.Source) []Result {
	var out []Result
	add := func(r Result) { out = append(out, r) }

	add(checkBuild(src))
	add(checkHashArithmetic())

	player, perr := mem.U32(src, mem.Base+rvaPlayerUnit)
	first, ferr := mem.U32(src, mem.Base+rvaFirstControl)
	if perr != nil || ferr != nil {
		add(Result{"menu-or-game", Skip, "V", "player/control variables not present"})
	} else {
		switch {
		case player == 0 && first != 0:
			add(Result{"menu-or-game", Pass, "V", "menu (player==0, control!=0)"})
		case player != 0 && first == 0:
			add(Result{"menu-or-game", Pass, "V", "in game (player!=0, control==0)"})
		default:
			add(Result{"menu-or-game", Fail, "V", fmt.Sprintf("player=0x%x control=0x%x is neither menu nor game", player, first)})
		}
	}
	add(checkServerTable(src))

	if perr != nil || player == 0 {
		for _, n := range []string{"player-type", "player-area", "inventory-owner", "statlist-owner"} {
			add(Result{n, Skip, "V", "not in game or player unit not present"})
		}
		return out
	}
	r := d2.New(src)
	if t, err := mem.U32(src, player); err != nil {
		add(Result{"player-type", Skip, "V", err.Error()})
	} else if t != 0 {
		add(Result{"player-type", Fail, "V", fmt.Sprintf("dwType=%d, want 0", t)})
	} else {
		add(Result{"player-type", Pass, "V", "dwType==0"})
	}
	add(checkArea(r, player))
	add(checkOwner(src, player, 0x60, 0x08, "inventory-owner", "V", "Inventory.pOwner"))
	// V on town.snap/town_inv.snap: StatList+0x04 (D2BS pUnit) is 0 on 1.14d; the player pointer is
	// at +0x44 (type 0 at +8, unit id at +0xC, flags 0x80000000 at +0x10).
	add(checkOwner(src, player, 0x5C, 0x44, "statlist-owner", "V", "StatList+0x44 owner"))
	return out
}

func checkBuild(src mem.Source) Result {
	err := mem.VerifyBuild(src)
	switch {
	case err == nil:
		return Result{"build", Pass, "V", fmt.Sprintf("PE stamp 0x%x size 0x%x", mem.ExpectTimestamp, mem.ExpectSizeOfImage)}
	case errors.Is(err, mem.ErrBuild) && unmappedHeader(src):
		return Result{"build", Skip, "V", "PE header page not in this source (legacy fixture)"}
	}
	return Result{"build", Fail, "V", err.Error()}
}

func unmappedHeader(src mem.Source) bool {
	var b [4]byte
	err := src.ReadAt(b[:], mem.Base)
	return err != nil && unmapped(err)
}

// The client table sits 0xC00 below the server table (V by arithmetic: 6 tables x 0x200 stride
// separate the two arrays; docs section 6).
func checkHashArithmetic() Result {
	if int64(rvaClientUnits)-int64(rvaServerUnits) != -0xC00 || tableStride*numTables != 0xC00 {
		return Result{"hash-table-delta", Fail, "V", "client-server table delta != -0xC00"}
	}
	return Result{"hash-table-delta", Pass, "V", "client - server == -0xC00"}
}

func checkServerTable(src mem.Source) Result {
	const name = "server-table-chain"
	seen, bad := 0, ""
	readable := false
	for typ := 0; typ < numTables && bad == ""; typ++ {
		for b := 0; b < 128 && bad == ""; b++ {
			head, err := mem.U32(src, mem.Base+rvaServerUnits+uint32(typ*tableStride+b*4))
			if err != nil {
				continue
			}
			readable = true
			u := head
			for n := 0; u != 0 && n < maxChain; n++ {
				ut, err := mem.U32(src, u)
				if err != nil {
					break
				}
				if int(ut) != typ {
					bad = fmt.Sprintf("table %d bucket %d unit 0x%x has dwType %d", typ, b, u, ut)
					break
				}
				seen++
				if u, err = mem.U32(src, u+0xE4); err != nil {
					break
				}
			}
		}
	}
	switch {
	case bad != "":
		return Result{name, Fail, "V", bad}
	case !readable:
		return Result{name, Skip, "V", "server unit tables not present"}
	case seen == 0:
		return Result{name, Skip, "V", "tables present but no unit readable"}
	}
	return Result{name, Pass, "V", fmt.Sprintf("%d units, dwType matches table index", seen)}
}

func checkArea(r *d2.Reader, player uint32) Result {
	a, err := r.AreaOf(player)
	switch {
	case err == nil && a >= 1 && a <= 136:
		return Result{"player-area", Pass, "V", fmt.Sprintf("area %d", a)}
	case err == nil:
		return Result{"player-area", Fail, "V", fmt.Sprintf("area %d outside 1..136", a)}
	case unmapped(err):
		return Result{"player-area", Skip, "V", "area chain not in this source"}
	}
	return Result{"player-area", Fail, "V", err.Error()}
}

// checkOwner follows player+ptrOff to a struct and expects the pointer at +backOff to be the player.
func checkOwner(src mem.Source, player, ptrOff, backOff uint32, name, tag, what string) Result {
	p, err := mem.U32(src, player+ptrOff)
	if err != nil {
		return Result{name, Skip, tag, err.Error()}
	}
	if p == 0 {
		return Result{name, Fail, tag, fmt.Sprintf("player+0x%x is null", ptrOff)}
	}
	back, err := mem.U32(src, p+backOff)
	switch {
	case err != nil && unmapped(err):
		return Result{name, Skip, tag, "struct not in this source"}
	case err != nil:
		return Result{name, Fail, tag, err.Error()}
	case back != player:
		return Result{name, Fail, tag, fmt.Sprintf("%s=0x%x, want player 0x%x", what, back, player)}
	}
	return Result{name, Pass, tag, what + " == player"}
}
