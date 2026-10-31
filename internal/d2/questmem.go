package d2

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"

	"d2lod-bot/internal/mem"
)

// In-memory quest and waypoint data (docs/questmem.md has the evidence).
//
// PlayerData (UnitAny+0x14) layout, VERIFIED LIVE: name[16], then QuestInfo* for Normal, Nightmare, Hell at
// +0x10/+0x14/+0x18, then WaypointInfo* for the same three at +0x1C/+0x20/+0x24. QuestInfo is {Word* buf; DWORD}
// and buf holds the 48 little-endian WORDs of the save's quest block. A waypoint struct is {02 01, 5 bytes of
// bits, ...}, the same bytes as the save's "WS" section.
//
// The CLIENT player unit (global PlayerUnit) has these pointers but its buffers stay empty; only the embedded
// game server's player unit holds the real data. That unit has no static pointer, so FindServerPlayer scans.
const (
	pdQuest    = 0x10
	pdWaypoint = 0x1C
	// QuestWords is the number of WORDs per difficulty.
	QuestWords = 48
	// WaypointBytes is the number of mask bytes read per difficulty (39 bits).
	WaypointBytes = 5
)

// Region is a readable and writable address range of the game process.
type Region struct{ Lo, Hi uint32 }

// RegionLister lets a non-process Source (tests, replay) describe where FindServerPlayer should look.
type RegionLister interface{ Regions() []Region }

// ErrNoServerPlayer: the scan found no server-side copy of the player (not in a game, or still loading).
var ErrNoServerPlayer = errors.New("d2: server-side player not found")

func (r *Reader) regions() ([]Region, error) {
	switch s := r.S.(type) {
	case RegionLister:
		return s.Regions(), nil
	case *mem.Proc:
		return RegionsOfPID(s.PID)
	}
	return nil, errors.New("d2: source cannot list regions")
}

// ServerPlayerOK reports whether addr is the server-side UnitAny of the player p: type 0, same class and id,
// an extended stat list and a PlayerData whose name matches. The client copy (p.Addr) is rejected.
func (r *Reader) ServerPlayerOK(addr uint32, p *Player) bool {
	if addr == p.Addr || !okPtr(addr) {
		return false
	}
	t, e1 := r.u32(addr + uoType)
	c, e2 := r.u32(addr + uoClass)
	id, e3 := r.u32(addr + uoID)
	if e1 != nil || e2 != nil || e3 != nil || t != 0 || c != uint32(p.Class) || id != p.ID {
		return false
	}
	sl, err := r.u32(addr + uoStats)
	if err != nil || !okPtr(sl) {
		return false
	}
	if fl, err := r.u32(sl + slFlags); err != nil || fl&slFlagEx == 0 {
		return false
	}
	pd, err := r.u32(addr + uoData)
	if err != nil || !okPtr(pd) {
		return false
	}
	n, err := r.cstr(pd, 16)
	return err == nil && n == p.Name
}

// FindServerPlayer scans the process for the server-side copy of player p (about a second on the live game).
// Callers cache the result and re-validate it with ServerPlayerOK.
func (r *Reader) FindServerPlayer(p *Player) (uint32, error) {
	regs, err := r.regions()
	if err != nil {
		return 0, err
	}
	for _, rg := range regs {
		buf := make([]byte, rg.Hi-rg.Lo)
		if r.S.ReadAt(buf, rg.Lo) != nil {
			continue
		}
		for o := 0; o+0x70 <= len(buf); o += 4 {
			if binary.LittleEndian.Uint32(buf[o+uoID:]) != p.ID ||
				binary.LittleEndian.Uint32(buf[o+uoClass:]) != uint32(p.Class) ||
				binary.LittleEndian.Uint32(buf[o+uoType:]) != 0 {
				continue
			}
			if a := rg.Lo + uint32(o); r.ServerPlayerOK(a, p) {
				return a, nil
			}
		}
	}
	return 0, ErrNoServerPlayer
}

// PlayerProgress is the quest words and waypoint masks of all three difficulties as the game holds them.
type PlayerProgress struct {
	Unit      uint32 // the server unit read
	Quest     [3][QuestWords]uint16
	Waypoints [3]uint64 // bit n = waypoint n (saves.Waypoint order); the 5 mask bytes little-endian
}

// ReadProgress follows serverUnit -> PlayerData -> QuestInfo -> buffer (and the waypoint structs).
func (r *Reader) ReadProgress(serverUnit uint32) (*PlayerProgress, error) {
	pd, err := r.ptr(serverUnit + uoData)
	if err != nil {
		return nil, err
	}
	out := &PlayerProgress{Unit: serverUnit}
	for d := uint32(0); d < 3; d++ {
		qi, err := r.ptr(pd + pdQuest + 4*d)
		if err != nil {
			return nil, err
		}
		buf, err := r.ptr(qi)
		if err != nil {
			return nil, err
		}
		b, err := r.bytes(buf, QuestWords*2)
		if err != nil {
			return nil, err
		}
		for i := range out.Quest[d] {
			out.Quest[d][i] = binary.LittleEndian.Uint16(b[2*i:])
		}
		wp, err := r.ptr(pd + pdWaypoint + 4*d)
		if err != nil {
			return nil, err
		}
		wb, err := r.bytes(wp, 2+WaypointBytes)
		if err != nil {
			return nil, err
		}
		for i := 0; i < WaypointBytes; i++ {
			out.Waypoints[d] |= uint64(wb[2+i]) << (8 * i)
		}
	}
	return out, nil
}

// PlayerIdent reads just enough of the client player (address, id, class, name) to find and validate the
// server copy. It is much cheaper than Player.
func (r *Reader) PlayerIdent() (*Player, error) {
	pa, err := r.playerAddr()
	if err != nil {
		return nil, err
	}
	id, err := r.u32(pa + uoID)
	if err != nil {
		return nil, err
	}
	class, err := r.u32(pa + uoClass)
	if err != nil {
		return nil, err
	}
	if class > 6 {
		return nil, fmt.Errorf("d2: implausible player class %d", class)
	}
	pd, err := r.ptr(pa + uoData)
	if err != nil {
		return nil, err
	}
	name, err := r.cstr(pd, 16)
	if err != nil {
		return nil, err
	}
	return &Player{Addr: pa, ID: id, Class: Class(class), Name: name}, nil
}

// RegionsOfPID lists the private read-write mappings of a process below 2 GB (where the game's heap lives), from
// /proc/PID/maps. A managed game is read through a plain file (launch.MemReader), so its owner builds a RegionLister
// from this.
func RegionsOfPID(pid int) ([]Region, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Region
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) < 2 || !strings.HasPrefix(fs[1], "rw") {
			continue
		}
		var lo, hi uint64
		fmt.Sscanf(fs[0], "%x-%x", &lo, &hi)
		if lo < 0x10000 || hi > 0x80000000 || hi-lo > 0x10000000 {
			continue
		}
		out = append(out, Region{uint32(lo), uint32(hi)})
	}
	return out, sc.Err()
}
