// d2mapdump: live map reader tool (read-only). Run on the worker via tools/live.sh.
//
//	d2mapdump probe              hexdump the map structs around the player
//	d2mapdump dump [ticks]       update a Tracker and print ASCII grid, presets, exits, units
//	d2mapdump record FILE        run a Tracker for a few ticks and save every byte read as a fixture
//	d2mapdump render FILE        replay a fixture offline (works on any machine)
//	d2mapdump watch [secs]       print player pos/level and grid growth twice a second
package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/mem"
)

func main() {
	cmd := "dump"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if cmd == "render" {
		render(os.Args[2])
		return
	}
	s, err := openGame()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	switch cmd {
	case "probe":
		probe(s)
	case "dump":
		n := 3
		if len(os.Args) > 2 {
			n, _ = strconv.Atoi(os.Args[2])
		}
		dump(s, n)
	case "explore":
		explore(s)
	case "explore2":
		explore2(s)
	case "movetest":
		dx, _ := strconv.Atoi(os.Args[2])
		dy, _ := strconv.Atoi(os.Args[3])
		movetest(s, dx, dy)
	case "record":
		record(s, os.Args[2])
	case "watch":
		n := 20
		if len(os.Args) > 2 {
			n, _ = strconv.Atoi(os.Args[2])
		}
		watch(s, n)
	default:
		fmt.Println("unknown command", cmd)
		os.Exit(2)
	}
}

func hexdump(s mem.Source, name string, addr uint32, n int) {
	b := make([]byte, n)
	if err := s.ReadAt(b, addr); err != nil {
		fmt.Printf("%s @%08x: %v\n", name, addr, err)
		return
	}
	fmt.Printf("%s @%08x\n", name, addr)
	for i := 0; i < n; i += 16 {
		fmt.Printf("  +%03x:", i)
		for j := i; j < i+16 && j < n; j += 4 {
			fmt.Printf(" %02x%02x%02x%02x", b[j+3], b[j+2], b[j+1], b[j])
		}
		fmt.Println()
	}
}

func probe(s mem.Source) {
	r := d2map.Reader{S: s}
	pc, err := r.Player()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("player %+v\n", pc)
	act, _ := r.Act(pc.Act)
	fmt.Printf("act %+v\n", act)
	r1, _ := r.Room1(pc.Room1)
	fmt.Printf("room1 %+v\n", r1)
	r2, _ := r.Room2(r1.Room2)
	fmt.Printf("room2 %+v\n", r2)
	hexdump(s, "room2", r1.Room2, 0x70)
	lv, err := r.Level(r2.Level)
	fmt.Printf("level %+v err=%v\n", lv, err)
	for _, l := range r.Levels(act) {
		fmt.Printf("  level %3d rect=%v next=0x%x room2first=0x%x\n", l.No, l.Rect, l.Next, l.Room2First)
	}
	rooms := r.LevelRoom2s(lv)
	fmt.Printf("level %d has %d room2s\n", lv.No, len(rooms))
	nr1 := 0
	for _, rm := range rooms {
		if rm.Room1 != 0 {
			nr1++
		}
	}
	fmt.Printf("  with room1: %d\n", nr1)
	act1 := r.ActiveRoom1s(act, pc.Room1)
	fmt.Printf("active room1s: %d (act list head 0x%x)\n", len(act1), act.Room1First)
	for _, rm := range act1 {
		fmt.Printf("  room1 0x%x rect=%v near=%d coll=0x%x\n", rm.Addr, rm.Rect, len(rm.Near), rm.Coll)
	}
}

func lvlName(n int) string {
	names := map[int]string{1: "Rogue Encampment", 2: "Blood Moor", 3: "Cold Plains", 40: "Lut Gholein", 75: "Kurast Docks", 103: "Pandemonium Fortress", 109: "Harrogath"}
	if s, ok := names[n]; ok {
		return fmt.Sprintf("%d (%s)", n, s)
	}
	return strconv.Itoa(n)
}

func report(t *d2map.Tracker, st d2map.State, s mem.Source) {
	lm := t.Current()
	fmt.Printf("player pos=(%d,%d) level=%s act=%d  activeRooms=%d newRooms=%d\n", st.Pos.X, st.Pos.Y, lvlName(st.LevelNo), st.Act, st.ActiveRooms, st.NewRooms)
	fmt.Printf("player room2: %v  near=%d\n", st.Room.Rect, len(st.NearRooms))
	fmt.Printf("level rect=%v known cells=%d bounds=%v  room2s=%d\n", lm.Rect, lm.Grid.KnownCount(), lm.Grid.Bounds(), len(lm.Rooms))
	fmt.Println("exits:")
	for _, e := range lm.Exits {
		fmt.Printf("  -> %s at %v\n", lvlName(e.Dest), e.Pos)
	}
	fmt.Println("adjacent levels:")
	for _, a := range lm.Adjacent {
		fmt.Printf("  %s rect=%v\n", lvlName(a.LevelNo), a.Rect)
	}
	fmt.Println("level warps:", lm.Warps)
	fmt.Println("presets:")
	for _, p := range lm.Presets {
		fmt.Printf("  kind=%d id=%-4d pos=(%d,%d)\n", p.Kind, p.ID, p.Pos.X, p.Pos.Y)
	}
	marks := map[d2map.Point]rune{{X: st.Pos.X, Y: st.Pos.Y}: '@'}
	for _, p := range lm.Presets {
		ch := map[int]rune{d2map.PresetNPC: 'n', d2map.PresetObject: 'o', d2map.PresetTile: 'T'}[p.Kind]
		if p.Kind == d2map.PresetObject && d2map.IsWaypointObject(p.ID) {
			ch = 'W'
		}
		if p.Kind == d2map.PresetObject && p.ID == 267 {
			ch = 'S'
		}
		if p.Kind == d2map.PresetNPC {
			if c, ok := map[int]rune{147: 'G', 148: 'A', 150: 'K', 154: 'C', 155: 'V'}[p.ID]; ok {
				ch = c
			}
		}
		if ch != 0 {
			marks[p.Pos] = ch
		}
	}
	fmt.Print(lm.Grid.ASCII(lm.Grid.Bounds(), marks))
	// flag histogram
	hist := map[uint16]int{}
	b := lm.Grid.Bounds()
	for y := b.Y; y < b.Bottom(); y++ {
		for x := b.X; x < b.Right(); x++ {
			if f, ok := lm.Grid.Flags(x, y); ok {
				hist[f]++
			}
		}
	}
	var keys []int
	for f := range hist {
		keys = append(keys, int(f))
	}
	sort.Ints(keys)
	fmt.Println("flag histogram:")
	for _, k := range keys {
		fmt.Printf("  0x%04x: %d\n", k, hist[uint16(k)])
	}
}

func dump(s mem.Source, ticks int) {
	t := d2map.NewTracker(s)
	t.RescanEvery = 1
	var st d2map.State
	var err error
	for i := 0; i < ticks; i++ {
		if st, err = t.Update(); err != nil {
			fmt.Println("update:", err)
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	report(t, st, s)
	units(s, t, st)
}

func record(s mem.Source, path string) {
	rec := d2map.NewRecorder(s)
	t := d2map.NewTracker(rec)
	t.RescanEvery = 1
	for i := 0; i < 3; i++ {
		if _, err := t.Update(); err != nil {
			fmt.Println("update:", err)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	// also record the unit table for cross-checks
	collectUnits(rec)
	img := rec.Image()
	f, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	if err := img.Save(f); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("saved %s (%d bytes of memory)\n", path, img.Size())
}

func render(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	img, err := d2map.LoadImage(f)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	t := d2map.NewTracker(img)
	st, err := t.Update()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	report(t, st, img)
	units(img, t, st)
}

func watch(s mem.Source, secs int) {
	t := d2map.NewTracker(s)
	t.RescanEvery = 4
	last := ""
	for i := 0; i < secs*2; i++ {
		st, err := t.Update()
		line := fmt.Sprint(err)
		if err == nil {
			line = fmt.Sprintf("pos=(%d,%d) level=%d active=%d known=%d levels=%v", st.Pos.X, st.Pos.Y, st.LevelNo, st.ActiveRooms, t.Current().Grid.KnownCount(), t.LevelNos())
		}
		if line != last {
			fmt.Println(time.Now().Format("15:04:05.0"), line)
			last = line
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// explore prints raw Room2 details: tile lists, preset kind counts and where AlternateTile cells are.
func explore(s mem.Source) {
	t := d2map.NewTracker(s)
	t.RescanEvery = 1
	if _, err := t.Update(); err != nil {
		fmt.Println(err)
		return
	}
	lm := t.Current()
	r := d2map.Reader{S: s}
	for _, rm := range lm.Rooms {
		kinds := map[int]int{}
		for _, p := range r.Presets(rm) {
			kinds[p.Kind]++
		}
		tiles := r.RoomTiles(rm)
		fmt.Printf("room2 0x%x rect=%v room1=0x%x tiles=0x%x (%d) presets=0x%x kinds=%v near=%d flags=0x%x\n", rm.Addr, rm.Rect, rm.Room1, rm.Tiles, len(tiles), rm.Preset, kinds, len(rm.Near), rm.Flags)
		for _, tl := range tiles {
			d, _ := r.Room2(tl.Dest)
			lv, _ := r.Level(d.Level)
			fmt.Printf("    tile num=%d dest room2=0x%x level=%d\n", tl.Num, tl.Dest, lv.No)
		}
	}
	b := lm.Grid.Bounds()
	for y := b.Y; y < b.Bottom(); y++ {
		for x := b.X; x < b.Right(); x++ {
			if f, ok := lm.Grid.Flags(x, y); ok && f&d2map.AlternateTile != 0 {
				fmt.Printf("alt-tile cell (%d,%d) flags=0x%04x\n", x, y, f)
			}
		}
	}
}

// explore2 scans the static Room2 data of every initialised level in the act.
func explore2(s mem.Source) {
	r := d2map.Reader{S: s}
	pc, err := r.Player()
	if err != nil {
		fmt.Println(err)
		return
	}
	act, _ := r.Act(pc.Act)
	for _, l := range r.Levels(act) {
		if l.Room2First == 0 {
			continue
		}
		rooms := r.LevelRoom2s(l)
		fmt.Printf("level %d rect=%v room2s=%d\n", l.No, l.Rect, len(rooms))
		withR1, withPre, preNoR1, tileNoR1 := 0, 0, 0, 0
		for _, rm := range rooms {
			ps := r.Presets(rm)
			if rm.Room1 != 0 {
				withR1++
			}
			if len(ps) > 0 {
				withPre++
				if rm.Room1 == 0 {
					preNoR1++
				}
			}
			if rm.Tiles != 0 && rm.Room1 == 0 {
				tileNoR1++
			}
		}
		fmt.Printf("  with room1=%d with presets=%d presets-without-room1=%d tiles-without-room1=%d\n", withR1, withPre, preNoR1, tileNoR1)
		for _, rm := range rooms {
			ps := r.Presets(rm)
			tiles := r.RoomTiles(rm)
			kinds := map[int]int{}
			for _, p := range ps {
				kinds[p.Kind]++
			}
			if len(tiles) > 0 || kinds[d2map.PresetTile] > 0 {
				fmt.Printf("  room2 0x%x rect=%v room1=0x%x kinds=%v\n", rm.Addr, rm.Rect, rm.Room1, kinds)
				for _, p := range ps {
					if p.Kind == d2map.PresetTile {
						fmt.Printf("    tile preset id=%d pos=%v\n", p.ID, p.Pos)
					}
				}
				for _, tl := range tiles {
					d, _ := r.Room2(tl.Dest)
					lv, err := r.Level(d.Level)
					fmt.Printf("    roomtile num=%d dest room2=0x%x level=%d err=%v\n", tl.Num, tl.Dest, lv.No, err)
				}
			}
		}
	}
}
