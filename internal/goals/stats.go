package goals

import "time"

const (
	minuteBuckets = 1440 // 24 h of per-minute buckets
	hourBuckets   = 720  // 30 days of per-hour buckets
	maxDrops      = 100
	maxAreas      = 100
)

type Totals struct {
	Kills         int64 `json:"kills"`
	BossKills     int64 `json:"bossKills"`
	Drops         int64 `json:"drops"`
	Deaths        int64 `json:"deaths"`
	Potions       int64 `json:"potions"`
	LevelUps      int64 `json:"levelUps"`
	Errors        int64 `json:"errors"`
	XP            int64 `json:"xp"`
	ActiveSeconds int64 `json:"activeSeconds"` // time a goal was actively being played
}

// Bucket is one time slice of the series; T is the slice start in unix seconds.
type Bucket struct {
	T       int64 `json:"t"`
	XP      int64 `json:"xp"`
	Kills   int   `json:"kills"`
	Drops   int   `json:"drops"`
	Deaths  int   `json:"deaths"`
	Potions int   `json:"potions"`
}

type DropRecord struct {
	Time   time.Time `json:"time"`
	Item   Item      `json:"item"`
	Area   string    `json:"area,omitempty"`
	GoalID string    `json:"goalId,omitempty"`
}

type AreaVisit struct {
	Area    string    `json:"area"`
	AreaID  int       `json:"areaId,omitempty"`
	Enter   time.Time `json:"enter"`
	Seconds int64     `json:"seconds"` // 0 for the visit in progress
}

// Stats is all counters and series. Every slice is bounded, so memory and file size are constant.
type Stats struct {
	Since     time.Time      `json:"since"`
	Totals    Totals         `json:"totals"`
	Minutes   []Bucket       `json:"minutes"`
	Hours     []Bucket       `json:"hours"`
	Drops     []DropRecord   `json:"drops"`
	Areas     []AreaVisit    `json:"areas"`
	Qualities map[string]int `json:"qualities"` // drop counts by quality
}

func newStats(now time.Time) *Stats {
	return &Stats{Since: now, Qualities: map[string]int{}}
}

func bump(bs *[]Bucket, step, max int64, t int64, f func(*Bucket)) {
	t -= t % step
	s := *bs
	for i := len(s) - 1; i >= 0 && i >= len(s)-4; i-- {
		if s[i].T == t {
			f(&s[i])
			return
		}
		if s[i].T < t {
			break
		}
	}
	b := Bucket{T: t}
	f(&b)
	// keep sorted: insert after the last bucket with T <= t
	i := len(s)
	for i > 0 && s[i-1].T > t {
		i--
	}
	s = append(s, Bucket{})
	copy(s[i+1:], s[i:])
	s[i] = b
	if int64(len(s)) > max {
		s = append([]Bucket(nil), s[int64(len(s))-max:]...)
	}
	*bs = s
}

func (s *Stats) bump(t time.Time, f func(*Bucket)) {
	bump(&s.Minutes, 60, minuteBuckets, t.Unix(), f)
	bump(&s.Hours, 3600, hourBuckets, t.Unix(), f)
}

func (s *Stats) record(ev *Event, goalID string) {
	t := ev.Time
	switch ev.Type {
	case EvKill:
		s.Totals.Kills++
		if ev.Boss {
			s.Totals.BossKills++
		}
		s.Totals.XP += ev.XP
		s.bump(t, func(b *Bucket) { b.Kills++; b.XP += ev.XP })
	case EvDrop:
		s.Totals.Drops++
		it := Item{}
		if ev.Item != nil {
			it = *ev.Item
		}
		q := it.Quality
		if q == "" {
			q = "normal"
		}
		s.Qualities[q] += it.qty()
		s.Drops = append(s.Drops, DropRecord{Time: t, Item: it, Area: s.currentArea(), GoalID: goalID})
		if len(s.Drops) > maxDrops {
			s.Drops = append([]DropRecord(nil), s.Drops[len(s.Drops)-maxDrops:]...)
		}
		s.bump(t, func(b *Bucket) { b.Drops++ })
	case EvDeath:
		s.Totals.Deaths++
		s.bump(t, func(b *Bucket) { b.Deaths++ })
	case EvPotion:
		s.Totals.Potions++
		s.bump(t, func(b *Bucket) { b.Potions++ })
	case EvLevelUp:
		s.Totals.LevelUps++
	case EvError:
		s.Totals.Errors++
	case EvAreaChange:
		if n := len(s.Areas); n > 0 && s.Areas[n-1].Seconds == 0 {
			s.Areas[n-1].Seconds = max(1, int64(t.Sub(s.Areas[n-1].Enter).Seconds()))
		}
		s.Areas = append(s.Areas, AreaVisit{Area: ev.Area, AreaID: ev.AreaID, Enter: t})
		if len(s.Areas) > maxAreas {
			s.Areas = append([]AreaVisit(nil), s.Areas[len(s.Areas)-maxAreas:]...)
		}
	}
}

func (s *Stats) currentArea() string {
	if n := len(s.Areas); n > 0 {
		return s.Areas[n-1].Area
	}
	return ""
}

// Point is one step of a time series. XPPerHour is the XP of the step scaled to an hour.
type SeriesPoint struct {
	T         int64   `json:"t"`
	XP        int64   `json:"xp"`
	XPPerHour float64 `json:"xpPerHour"`
	Kills     int     `json:"kills"`
	Drops     int     `json:"drops"`
	Deaths    int     `json:"deaths"`
	Potions   int     `json:"potions"`
}

// Series returns window/step points ending at now, zero-filled. Steps below 1 h read the minute ring
// (so only the last 24 h are available), longer steps read the hour ring.
func (s *Stats) Series(now time.Time, window, step time.Duration) []SeriesPoint {
	if step < time.Minute {
		step = time.Minute
	}
	if window < step {
		window = step
	}
	n := int(window / step)
	if n > 1000 {
		n, window = 1000, step*1000
	}
	src, base := s.Minutes, int64(60)
	if step >= time.Hour {
		src, base = s.Hours, 3600
	}
	st := int64(step / time.Second)
	end := now.Unix() - now.Unix()%st + st // exclusive end of the current step
	start := end - int64(n)*st
	out := make([]SeriesPoint, n)
	for i := range out {
		out[i].T = start + int64(i)*st
	}
	_ = base
	for _, b := range src {
		if b.T < start || b.T >= end {
			continue
		}
		p := &out[(b.T-start)/st]
		p.XP += b.XP
		p.Kills += b.Kills
		p.Drops += b.Drops
		p.Deaths += b.Deaths
		p.Potions += b.Potions
	}
	for i := range out {
		out[i].XPPerHour = float64(out[i].XP) * 3600 / float64(st)
	}
	return out
}

// XPPerHour is XP gained over the last `over`, divided by the time actually covered (at least a minute),
// so a freshly started session is not diluted by empty history.
func (s *Stats) XPPerHour(now time.Time, over time.Duration) float64 {
	from := now.Add(-over).Unix()
	var xp int64
	first := int64(0)
	for _, b := range s.Minutes {
		if b.T+60 <= from {
			continue
		}
		if first == 0 || b.T < first {
			first = b.T
		}
		xp += b.XP
	}
	if xp == 0 {
		return 0
	}
	span := float64(now.Unix() - max(first, from))
	return float64(xp) * 3600 / max(span, 60)
}
