package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"sync"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dworld"
	"d2lod-bot/internal/d2map"
)

// Mon is one monster on the map (cell coordinates, level relative).
type Mon struct {
	X, Y   float64
	Name   string
	HP, Mx int
	Boss   bool
	Dead   bool
	Friend bool `json:",omitempty"`
}

// Snap is what the page draws; it is rebuilt at every poll.
type Snap struct {
	Seq        int64           `json:"seq"`
	OK         bool            `json:"ok"`
	Err        string          `json:"err,omitempty"`
	Level      int             `json:"level"`
	Area       string          `json:"area"`
	MapVer     uint64          `json:"mapVer"`
	MapRect    [4]int          `json:"mapRect"` // x0, y0, w, h in subtiles of the PNG
	X          float64         `json:"x"`
	Y          float64         `json:"y"`
	Name       string          `json:"name"`
	Class      string          `json:"class"`
	CLvl       int             `json:"clvl"`
	XP         int             `json:"xp"`
	Gold       int             `json:"gold"`
	Life       [2]int          `json:"life"`
	Mana       [2]int          `json:"mana"`
	Mons       []Mon           `json:"mons"`
	Dead       int             `json:"deadSeen"`
	GameMS     int             `json:"gameMs"` // virtual clock of the lockstep d2d (resets when d2d restarts)
	Tick       int             `json:"tick"`
	Speed      float64         `json:"speed"` // virtual s per wall s, smoothed over about 2 s
	Lockstep   bool            `json:"lockstep"`
	Exits      [][3]int        `json:"exits"` // dest, x, y (cells)
	Bot        json.RawMessage `json:"bot,omitempty"`
	Mode       Mode            `json:"mode"`
	FrameTicks int64           `json:"frameTicks"` // ticks between the last two sampled frames
	FrameMS    int             `json:"frameMs"`    // wall ms the last screenshot took
	Frames     int64           `json:"frames"`
	Tap        TapInfo         `json:"tap"`
}

// TapInfo shows what the viewer costs, so the page can say it.
type TapInfo struct {
	Steps, Ticks, ViewOps, BotLines int64
}

// Feed polls the d2d through the viewer socket and keeps the latest Snap and map PNG. It polls only while the page is
// being watched (Touch), so an unwatched run pays nothing.
type Feed struct {
	c    *d2dctl.Client
	w    *d2dworld.World
	mux  *Mux
	fr   *Frames
	hz   float64
	bot  string // bot API base URL ("" off)
	mu   sync.RWMutex
	snap Snap
	png  []byte
	mapV uint64
	mapL int
	seen map[int]bool
	// speed estimate
	lastMS   int
	lastWall time.Time
	ema      float64
	seq      int64
	lastSeen int64
	botJSON  json.RawMessage
	botAt    time.Time
}

func NewFeed(viewSock string, mux *Mux, hz float64, bot string) *Feed {
	c := d2dctl.New(viewSock, d2dctl.Options{Timeout: 3 * time.Second, NoReconnect: false})
	f := &Feed{c: c, mux: mux, hz: hz, bot: bot, seen: map[int]bool{}}
	f.w = d2dworld.New(c, d2dworld.Options{Timeout: 5 * time.Second, MinRefetch: 3 * time.Second})
	return f
}

func (f *Feed) Touch() { f.lastSeenStore(time.Now().UnixNano()) }

func (f *Feed) lastSeenStore(v int64) { f.mu.Lock(); f.lastSeen = v; f.mu.Unlock() }

func (f *Feed) watching() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return time.Since(time.Unix(0, f.lastSeen)) < 5*time.Second
}

// Run polls until ctx ends.
func (f *Feed) Run(ctx context.Context) {
	t := time.NewTicker(time.Duration(float64(time.Second) / f.hz))
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if !f.watching() {
			continue
		}
		f.poll(ctx)
	}
}

// Wait blocks until a snapshot newer than seq exists (or ctx ends).
func (f *Feed) Wait(ctx context.Context, seq int64) Snap {
	for {
		s := f.Snap()
		if s.Seq > seq {
			return s
		}
		select {
		case <-ctx.Done():
			return s
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func (f *Feed) Snap() Snap {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.snap
}

func (f *Feed) PNG() ([]byte, uint64) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.png, f.mapV
}

func (f *Feed) poll(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	// One write for all reads (docs/d2d-pace.md Prefetch); each typed call below finds its reply.
	_ = f.c.Prefetch(cctx, "state", "monsters", "debug level", "clock")
	defer f.c.DropPrefetch()
	s := Snap{Seq: f.seq + 1, Mode: f.mux.Mode()}
	f.seq++
	s.Tap = TapInfo{f.mux.steps.Load(), f.mux.ticks.Load(), f.mux.viewOps.Load(), f.mux.botLines.Load()}
	st, _, err := f.c.State(cctx)
	if err != nil {
		s.Err = err.Error()
		f.put(s, nil)
		return
	}
	s.Name, s.Class, s.CLvl, s.XP, s.Gold = st.Name, st.Class, st.Level, st.Exp, st.Gold
	s.Life, s.Mana = [2]int{st.Life.Cur, st.Life.Max}, [2]int{st.Mana.Cur, st.Mana.Max}
	ds, werr := f.w.Update()
	if werr == nil {
		s.OK = true
		s.Level, s.Area = ds.LevelNo, areaName(ds.LevelNo)
		s.X, s.Y = float64(ds.Pos.X)/5, float64(ds.Pos.Y)/5
	} else {
		s.Err = werr.Error()
	}
	if ms, e := f.c.Monsters(cctx); e == nil {
		for _, m := range ms.Items {
			mo := Mon{X: m.CellX, Y: m.CellY, Name: m.Name, HP: m.HP.Cur, Mx: m.HP.Max, Boss: m.IsBoss(), Dead: m.Dead()}
			if m.Dead() && !f.seen[m.Unit] {
				s.Dead++
			}
			f.seen[m.Unit] = true
			if mo.Dead && (len(s.Mons) > 150 || abs(mo.X-s.X) > 30 || abs(mo.Y-s.Y) > 30) {
				continue // corpses far away are noise on the map
			}
			s.Mons = append(s.Mons, mo)
		}
	}
	if len(f.seen) > 5000 {
		f.seen = map[int]bool{}
	}
	if ci, e := f.c.Clock(cctx); e == nil {
		s.GameMS, s.Tick, s.Lockstep = ci.GameMS, ci.Tick, ci.Lockstep
		s.Speed = f.speed(ci.GameMS)
	}
	f.mu.RLock()
	prev := f.snap.Dead
	f.mu.RUnlock()
	s.Dead += prev
	var pngb []byte
	if werr == nil {
		lm := f.w.Current()
		for _, e := range lm.Exits {
			s.Exits = append(s.Exits, [3]int{e.Dest, e.Pos.X / 5, e.Pos.Y / 5})
		}
		if lm.Grid.Version() != f.mapV || lm.No != f.mapL {
			pngb = renderGrid(lm.Grid)
			b := lm.Grid.Bounds()
			s.MapRect = [4]int{b.X, b.Y, b.W, b.H}
			f.mapV, f.mapL = lm.Grid.Version(), lm.No
		} else {
			f.mu.RLock()
			s.MapRect = f.snap.MapRect
			f.mu.RUnlock()
		}
		s.MapVer = f.mapV
	}
	s.Bot = f.botStatus()
	if f.fr != nil {
		fr := f.fr.Current()
		s.FrameTicks, s.FrameMS, s.Frames = fr.Ticks, fr.WallMS, fr.Seq
	}
	f.put(s, pngb)
}

func (f *Feed) put(s Snap, p []byte) {
	f.mu.Lock()
	f.snap = s
	if p != nil {
		f.png = p
	}
	f.mu.Unlock()
}

// speed is virtual seconds per wall second from game_ms, smoothed; a game_ms that went backwards (d2d restarted) resets it.
func (f *Feed) speed(ms int) float64 {
	now := time.Now()
	if !f.lastWall.IsZero() && ms >= f.lastMS {
		dt := now.Sub(f.lastWall).Seconds()
		if dt > 0 {
			inst := float64(ms-f.lastMS) / 1000 / dt
			if f.ema == 0 {
				f.ema = inst
			} else {
				f.ema += (inst - f.ema) * (1 - 0.85)
			}
		}
	} else {
		f.ema = 0
	}
	f.lastMS, f.lastWall = ms, now
	return f.ema
}

// botStatus reads the bot API status (read-only GET, at most once a second, no token is sent or kept).
func (f *Feed) botStatus() json.RawMessage {
	if f.bot == "" {
		return nil
	}
	if time.Since(f.botAt) < time.Second {
		return f.botJSON
	}
	f.botAt = time.Now()
	cl := http.Client{Timeout: 700 * time.Millisecond}
	resp, err := cl.Get(f.bot + "/api/v1/status")
	if err != nil {
		f.botJSON = nil
		return nil
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if json.Valid(b) {
		f.botJSON = b
	}
	return f.botJSON
}

// renderGrid draws the known cells: walkable dark grey, blocked light, unknown transparent. One pixel per subtile.
func renderGrid(g *d2map.Grid) []byte {
	b := g.Bounds()
	if b.W <= 0 || b.H <= 0 {
		return nil
	}
	pal := color.Palette{color.NRGBA{}, color.NRGBA{28, 32, 40, 255}, color.NRGBA{120, 130, 150, 255}}
	img := image.NewPaletted(image.Rect(0, 0, b.W, b.H), pal)
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			fl, ok := g.Flags(b.X+x, b.Y+y)
			switch {
			case !ok:
			case fl&d2map.BlockWalk != 0:
				img.SetColorIndex(x, y, 2)
			default:
				img.SetColorIndex(x, y, 1)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
