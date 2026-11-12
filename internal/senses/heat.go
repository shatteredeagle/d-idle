package senses

import (
	"fmt"
	"sort"
	"time"
)

// HeatCell is the side of a heat cell in subtiles: the roam coverage spacing (nav roamSpacing, 7), so the layers line
// up with the coverage grid (S: kept equal by hand, internal/nav is not imported here).
const HeatCell = 7

// MaxHeatCells caps the cells published for one area (the busiest win).
const MaxHeatCells = 3000

// heatCell counts per cell: kills, damage taken (life percent points) and time spent (ms).
type heatCell struct{ kills, dmg, dwellMs int }

type heatKey struct{ x, y int }

// Heat is the per-area density memory of the whole game (kills, time spent, damage taken). It is reset for an area
// when the level map of that area is a new object (a new game). Not safe for concurrent use; Publisher holds its lock.
type Heat struct {
	areas map[int]map[heatKey]*heatCell
	ids   map[int]string // identity of the level map each area's counts belong to
}

func (h *Heat) cell(area, x, y int) *heatCell {
	if h.areas == nil {
		h.areas = map[int]map[heatKey]*heatCell{}
	}
	m := h.areas[area]
	if m == nil {
		m = map[heatKey]*heatCell{}
		h.areas[area] = m
	}
	k := heatKey{floorDivN(x, HeatCell), floorDivN(y, HeatCell)}
	c := m[k]
	if c == nil {
		c = &heatCell{}
		m[k] = c
	}
	return c
}

func floorDivN(v, n int) int {
	if v < 0 {
		return -((-v + n - 1) / n)
	}
	return v / n
}

// seeLevel resets an area's counts when its level map identity changed (a new game).
func (h *Heat) seeLevel(area int, id string) {
	if h.ids == nil {
		h.ids = map[int]string{}
	}
	if old, ok := h.ids[area]; ok && old != id {
		delete(h.areas, area)
	}
	h.ids[area] = id
}

// HeatPoint is one published cell: its top-left subtile and the three counts (dwell in seconds).
type HeatPoint struct {
	X, Y, Kills, DwellS, Dmg int
}

// Area returns the cells of one area, the busiest MaxHeatCells of them, in x,y order.
func (h *Heat) Area(area int) []HeatPoint {
	var out []HeatPoint
	for k, c := range h.areas[area] {
		out = append(out, HeatPoint{k.x * HeatCell, k.y * HeatCell, c.kills, c.dwellMs / 1000, c.dmg})
	}
	if len(out) > MaxHeatCells {
		sort.Slice(out, func(i, j int) bool {
			return out[i].Kills*5+out[i].DwellS+out[i].Dmg > out[j].Kills*5+out[j].DwellS+out[j].Dmg
		})
		out = out[:MaxHeatCells]
	}
	sort.Slice(out, func(i, j int) bool { return out[i].X < out[j].X || out[i].X == out[j].X && out[i].Y < out[j].Y })
	return out
}

// observeHeat adds the time since the last observation to the cell the hero stood on. Holding p.mu.
func (p *Publisher) observeHeat(no int, id string, now time.Time, x, y int) {
	p.heat.seeLevel(no, id)
	if !p.heatAt.IsZero() && p.heatArea == no {
		if dt := now.Sub(p.heatAt); dt > 0 {
			if dt > time.Second {
				dt = time.Second // a pause in the feed is not time spent on that cell
			}
			p.heat.cell(no, p.heatX, p.heatY).dwellMs += int(dt / time.Millisecond)
		}
	}
	p.heatAt, p.heatArea, p.heatX, p.heatY = now, no, x, y
}

// AddDamage records life lost (percent points) at the hero's cell and a damage marker on the trail.
func (p *Publisher) AddDamage(pct int, text string) {
	p.mu.Lock()
	if p.have {
		p.heat.cell(p.areaNo, p.snap.Hero.X, p.snap.Hero.Y).dmg += pct
	}
	p.mu.Unlock()
	p.AddEvent(EvDamage, text, false, nil)
}

// HeatOf returns the heat cells of the current area and its id.
func (p *Publisher) HeatOf() ([]HeatPoint, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.heat.Area(p.areaNo), p.areaNo
}

func levelID(lm any) string { return fmt.Sprintf("%p", lm) }
