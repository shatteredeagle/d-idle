package main

import (
	"fmt"
	"time"

	"d2lod-bot/internal/saves"
	"d2lod-bot/internal/telemetry"
)

// The bot's /stats XP total adds every positive memory-exp step. Experience that is lost when the game dies without
// Save and Exit (D2 writes the .d2s only then, V arithmetic in docs/soak-analysis.md section 6) is earned a second time
// after the reload and counted twice. So the report measures character experience instead: from the .d2s when the soak
// reads it (-save-file), else from memory (level table + in-level charXp), always as a running maximum so lost play is
// not counted and then counted again.

// readSave fills the save fields of a sample from the .d2s. An unreadable file (mid-write, absent) is no error: the
// sample just has no save reading.
func readSave(path string, s *Sample) {
	if path == "" {
		return
	}
	if lvl, exp, err := saves.ExpFile(path); err == nil {
		s.SaveLevel, s.SaveXP = lvl, int64(exp)
	}
}

// memExp is the absolute experience the game showed at this sample (level table + in-level XP), -1 when unknown.
// U: the table is internal/telemetry's recalled classic table, levels 1 to 30 only.
func memExp(s Sample) int64 {
	if s.Level <= 0 || s.CharXP <= 0 && s.Level > 1 {
		return -1
	}
	base, ok := telemetry.DefaultExpForLevel(s.Level)
	if !ok {
		return -1
	}
	return int64(base) + s.CharXP
}

// levelWorth is the XP of the level the sample's character is in (0 when unknown).
func levelWorth(level int) int64 {
	a, ok1 := telemetry.DefaultExpForLevel(level)
	b, ok2 := telemetry.DefaultExpForLevel(level + 1)
	if !ok1 || !ok2 {
		return 0
	}
	return int64(b) - int64(a)
}

// CharXP is the character-experience view of a run.
type CharXP struct {
	Source     string // "save", "memory" or "" (no data)
	Gain       int64  // monotone-max experience at the end minus at the start
	Lost       int64  // experience that vanished between two samples (memory exp fell)
	LostEvents int
	LostAt     []time.Time
}

// charXP computes the view. The gain comes from saved experience when any sample has it.
func charXP(ss []Sample) CharXP {
	var c CharXP
	var first, top int64 = -1, -1
	feed := func(v int64) {
		if v < 0 {
			return
		}
		if first < 0 {
			first = v
		}
		top = max(top, v)
	}
	for _, s := range ss {
		if s.SaveXP > 0 {
			feed(s.SaveXP)
		}
	}
	if first >= 0 {
		c.Source = "save"
	} else {
		for _, s := range ss {
			feed(memExp(s))
		}
		if first >= 0 {
			c.Source = "memory"
		}
	}
	if first >= 0 {
		c.Gain = top - first
	}
	prev := int64(-1)
	for _, s := range ss {
		v := memExp(s)
		if v < 0 {
			continue
		}
		if prev >= 0 && v < prev {
			c.Lost += prev - v
			c.LostEvents++
			c.LostAt = append(c.LostAt, s.T)
		}
		prev = v
	}
	return c
}

// xpAtRisk is the newest sample's memory experience minus its saved experience (the soak's view of what a hard stop
// would lose now), with the XP worth of the current level. ok is false without both readings.
func xpAtRisk(ss []Sample) (risk, worth int64, ok bool) {
	for i := len(ss) - 1; i >= 0; i-- {
		s := ss[i]
		if v := memExp(s); v >= 0 && s.SaveXP > 0 {
			return v - s.SaveXP, levelWorth(s.Level), true
		}
	}
	return 0, 0, false
}

func (c CharXP) lostMsg() string {
	return fmt.Sprintf("%d XP vanished in %d samples (first %s): the game fell back to its last save, a hard stop without Save and Exit",
		c.Lost, c.LostEvents, c.LostAt[0].Format(time.RFC3339))
}
