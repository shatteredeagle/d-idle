package bot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"d2lod-bot/internal/input"
)

const (
	maxDeathRecords = 200
	heatCell        = 8 // subtiles per heat-map cell
)

// LifeSample is one point of the life curve before a death: milliseconds before the death and life percent.
type LifeSample struct {
	MsBefore int `json:"msBefore"`
	Pct      int `json:"pct"`
}

// DeathRecord is one entry of the death log.
type DeathRecord struct {
	Time           time.Time    `json:"time"`
	Area           int          `json:"area"`
	AreaName       string       `json:"areaName"`
	Pos            input.Point  `json:"pos"` // where the corpse lies
	Level          int          `json:"level"`
	Difficulty     string       `json:"difficulty"`
	Killer         string       `json:"killer,omitempty"` // nearest hostile at the last living snapshot; best effort, UNVERIFIED
	KillerClass    int          `json:"killerClass,omitempty"`
	KillerBoss     bool         `json:"killerBoss,omitempty"`
	KillerChampion bool         `json:"killerChampion,omitempty"` // a champion pack member killed us: not a boss, not a danger-table lesson
	Enchants       []uint8      `json:"enchants,omitempty"`       // raw monster enchant ids of the killer
	LifeCurve      []LifeSample `json:"lifeCurve,omitempty"`
	CorpseValue    int          `json:"corpseValue"` // estimated gold value of the equipped gear left on the corpse
	Retrieved      bool         `json:"retrieved,omitempty"`
	Abandoned      bool         `json:"abandoned,omitempty"`
	CostGold       int          `json:"costGold"`  // gold lost to the death itself
	CostItems      int          `json:"costItems"` // equipped items that did not come back
	CostSeconds    int          `json:"costSeconds"`
	Why            string       `json:"why,omitempty"`          // policy decision text
	RepeatBanMin   int          `json:"repeatBanMin,omitempty"` // minutes of the repeat-site ban this death triggered (escalation counts these)
	Autopsy        *Autopsy     `json:"autopsy,omitempty"`
}

// HeatPoint is one cell of the death heat map.
type HeatPoint struct {
	Area  int `json:"area"`
	X     int `json:"x"` // cell centre, subtiles
	Y     int `json:"y"`
	Count int `json:"count"`
}

// AreaDeaths summarizes deaths per area.
type AreaDeaths struct {
	Area   int       `json:"area"`
	Name   string    `json:"name"`
	Deaths int       `json:"deaths"`
	Last   time.Time `json:"last"`
}

// DeathReport is the display view of the death log (see Runner.DeathReport).
type DeathReport struct {
	Records         []DeathRecord      `json:"records"` // newest last
	ByArea          []AreaDeaths       `json:"byArea"`
	ByMonster       map[string]int     `json:"byMonster"`
	Heat            []HeatPoint        `json:"heat"`
	Danger          map[string]float64 `json:"danger"` // danger weights raised by the spiral breaker; the fighter may read them
	Total           int                `json:"total"`
	LastHour        int                `json:"lastHour"`
	TotalGold       int                `json:"totalGold"`
	TotalItems      int                `json:"totalItems"`
	AvoidedAreas    map[int]time.Time  `json:"avoidedAreas,omitempty"`
	PausedForDeaths bool               `json:"pausedForDeaths"`
}

// deathLog is the persisted part of the bot state. Guarded by Runner.obMu.
type deathLog struct {
	Version   int                   `json:"version"`
	Records   []DeathRecord         `json:"records"`
	Total     int                   `json:"total"`
	ByArea    map[int]*AreaDeaths   `json:"byArea"`
	ByMonster map[string]int        `json:"byMonster"`
	Heat      map[string]*HeatPoint `json:"heat"`
	Danger    map[string]float64    `json:"danger"`
}

func newDeathLog() *deathLog {
	return &deathLog{Version: 1, ByArea: map[int]*AreaDeaths{}, ByMonster: map[string]int{}, Heat: map[string]*HeatPoint{}, Danger: map[string]float64{}}
}

func (l *deathLog) add(rec DeathRecord) {
	l.Records = append(l.Records, rec)
	if len(l.Records) > maxDeathRecords {
		l.Records = l.Records[len(l.Records)-maxDeathRecords:]
	}
	l.Total++
	a := l.ByArea[rec.Area]
	if a == nil {
		a = &AreaDeaths{Area: rec.Area, Name: rec.AreaName}
		l.ByArea[rec.Area] = a
	}
	a.Deaths++
	a.Last = rec.Time
	if rec.Killer != "" {
		l.ByMonster[rec.Killer]++
	}
	cx, cy := rec.Pos.X/heatCell*heatCell+heatCell/2, rec.Pos.Y/heatCell*heatCell+heatCell/2
	key := heatKey(rec.Area, cx, cy)
	h := l.Heat[key]
	if h == nil {
		h = &HeatPoint{Area: rec.Area, X: cx, Y: cy}
		l.Heat[key] = h
	}
	h.Count++
}

func heatKey(area, x, y int) string {
	b, _ := json.Marshal([3]int{area, x, y})
	return string(b)
}

// last returns the newest record, or nil.
func (l *deathLog) last() *DeathRecord {
	if len(l.Records) == 0 {
		return nil
	}
	return &l.Records[len(l.Records)-1]
}

// gearCorpse is the newest corpse other than the one stamped skip that still holds gear worth at least min: not
// retrieved, not already tried and failed, not older than since. In D2 a softcore death leaves the equipped items on the
// body, so a character that died naked again is naked because of that corpse (V: Soaky, 4 deaths, gear gone since an
// earlier abandoned corpse). Whether the body stays after later deaths or a restart is U.
func (l *deathLog) gearCorpse(min int, since, skip time.Time) *DeathRecord {
	for i := len(l.Records) - 1; i >= 0; i-- {
		r := &l.Records[i]
		if r.Time.Before(since) {
			break
		}
		if r.Time.Equal(skip) || r.Retrieved || r.CorpseValue < min || r.Why == whyRetrievalFailed || r.Why == whySecondDeath {
			continue
		}
		return r
	}
	return nil
}

// since counts deaths at or after t, optionally limited to one area (area 0 = any).
func (l *deathLog) since(t time.Time, area int) int {
	n := 0
	for i := len(l.Records) - 1; i >= 0; i-- {
		r := l.Records[i]
		if r.Time.Before(t) {
			break
		}
		if area == 0 || r.Area == area {
			n++
		}
	}
	return n
}

func (l *deathLog) report(now time.Time, avoided map[int]time.Time, paused bool) DeathReport {
	rep := DeathReport{
		Records: append([]DeathRecord(nil), l.Records...), ByMonster: map[string]int{}, Danger: map[string]float64{},
		Total: l.Total, LastHour: l.since(now.Add(-time.Hour), 0), PausedForDeaths: paused,
	}
	for k, v := range l.ByMonster {
		rep.ByMonster[k] = v
	}
	for k, v := range l.Danger {
		rep.Danger[k] = v
	}
	for _, a := range l.ByArea {
		rep.ByArea = append(rep.ByArea, *a)
	}
	sort.Slice(rep.ByArea, func(i, j int) bool { return rep.ByArea[i].Deaths > rep.ByArea[j].Deaths })
	for _, h := range l.Heat {
		rep.Heat = append(rep.Heat, *h)
	}
	sort.Slice(rep.Heat, func(i, j int) bool { return rep.Heat[i].Count > rep.Heat[j].Count })
	for _, r := range l.Records {
		rep.TotalGold += r.CostGold
		rep.TotalItems += r.CostItems
	}
	if len(avoided) > 0 {
		rep.AvoidedAreas = map[int]time.Time{}
		for a, t := range avoided {
			if t.After(now) {
				rep.AvoidedAreas[a] = t
			}
		}
	}
	return rep
}

// loadDeathLog reads path; a missing or unreadable file gives an empty log (and the error for logging).
func loadDeathLog(path string) (*deathLog, error) {
	l := newDeathLog()
	if path == "" {
		return l, nil
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return l, nil
	}
	if err != nil {
		return l, err
	}
	var in deathLog
	if err := json.Unmarshal(b, &in); err != nil {
		return l, err
	}
	if in.ByArea == nil {
		in.ByArea = map[int]*AreaDeaths{}
	}
	if in.ByMonster == nil {
		in.ByMonster = map[string]int{}
	}
	if in.Heat == nil {
		in.Heat = map[string]*HeatPoint{}
	}
	if in.Danger == nil {
		in.Danger = map[string]float64{}
	}
	in.Version = 1
	return &in, nil
}

// save writes the log atomically (tmp file, rename).
func (l *deathLog) save(path string) error {
	if path == "" {
		return nil
	}
	b, err := json.MarshalIndent(l, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
