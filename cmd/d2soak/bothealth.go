package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Bot-health alerts from the once-a-minute summary line of the bot log (internal/bot minuteLine). The runner writes
// one every minute while it steps; the lack of one is the only sign of a hung or dead bot when /health still answers
// (a stuck loop under obMu, docs/archive/review-night.md M4). The log is read from the tail only: a 24 h log is large.

// minuteTailBytes bounds the log read per evaluation (alerts are refreshed every ~2 min).
const minuteTailBytes = 1 << 20

// resolveBotLog returns the file to read: path itself, or the newest *.log when path is a directory (a supervised
// bot logs to a new file per job, tools/live.sh runs/<ts>-d2bot.log).
func resolveBotLog(path string) string {
	fi, err := os.Stat(path)
	if err != nil || !fi.IsDir() {
		return path
	}
	ms, _ := filepath.Glob(filepath.Join(path, "*.log"))
	best, bt := "", time.Time{}
	for _, m := range ms {
		if st, err := os.Stat(m); err == nil && (best == "" || st.ModTime().After(bt)) {
			best, bt = m, st.ModTime()
		}
	}
	return best
}

// tailMinutes parses the minute lines of the last minuteTailBytes of the log. full reports that the whole file was
// read (so "no line" really means none yet) and size is the file size.
func tailMinutes(path string) (ms []Minute, full bool, err error) {
	f, err := os.Open(resolveBotLog(path))
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, false, err
	}
	full = st.Size() <= minuteTailBytes
	if !full {
		if _, err := f.Seek(st.Size()-minuteTailBytes, io.SeekStart); err != nil {
			return nil, false, err
		}
	}
	ms, err = ParseMinuteLines(f) // a cut first line is malformed and skipped
	return ms, full, err
}

// gameRestartsSince is the number of game restarts the minute lines show since cut. "restarts" is a running total
// since the bot process started; a drop means the bot itself restarted and the new total counts from zero.
func gameRestartsSince(ms []Minute, cut time.Time) int {
	n, prev := 0, -1
	for _, m := range ms {
		if prev >= 0 && !m.At.Before(cut) {
			if d := m.Restarts - prev; d > 0 {
				n += d
			} else if d < 0 {
				n += m.Restarts
			}
		}
		prev = m.Restarts
	}
	return n
}

// lastMinute is the newest minute line time (zero when none).
func lastMinute(ms []Minute) time.Time {
	var t time.Time
	for _, m := range ms {
		if m.At.After(t) {
			t = m.At
		}
	}
	return t
}

// botLogAlerts evaluates the rules that need the bot log. A missing or empty log raises nothing by itself (the
// bot may not have started; the health alerts cover that), but a log that has grown past the tail window without a
// single minute line counts as stale.
func botLogAlerts(rules AlertRules, now time.Time) []Alert {
	if rules.BotLog == "" || (rules.TickAge <= 0 && rules.GameRestartsPerHour <= 0) {
		return nil
	}
	ms, full, err := tailMinutes(rules.BotLog)
	if err != nil {
		return []Alert{{Level: "warn", Code: "bot-log", Msg: "cannot read the bot log: " + err.Error()}}
	}
	var out []Alert
	if rules.TickAge > 0 {
		switch last := lastMinute(ms); {
		case last.IsZero() && !full:
			out = append(out, Alert{Level: "crit", Code: "tick-age", Msg: fmt.Sprintf("no minute line in the last %d KB of the bot log: the bot is stalled or dead", minuteTailBytes>>10)})
		case !last.IsZero() && now.Sub(last) > rules.TickAge:
			out = append(out, Alert{Level: "crit", Code: "tick-age", Msg: fmt.Sprintf("no minute line for %s (limit %s): the bot is stalled or dead", mins(now.Sub(last)), mins(rules.TickAge))})
		}
	}
	if rules.GameRestartsPerHour > 0 {
		if n := gameRestartsSince(ms, now.Add(-time.Hour)); float64(n) > rules.GameRestartsPerHour {
			out = append(out, Alert{Level: "warn", Code: "game-restarts", Msg: fmt.Sprintf("%d game restarts in the last hour (limit %.0f): chicken re-entries and level-up resumes count, a login loop is the failure (docs/soak24.md section 5)", n, rules.GameRestartsPerHour)})
		}
	}
	return out
}

// wineAlert is the live winedevice.exe rule: the newest sample that has a count.
func wineAlert(ss []Sample, rules AlertRules) []Alert {
	if rules.WineOrphans <= 0 {
		return nil
	}
	for i := len(ss) - 1; i >= 0; i-- {
		if w := ss[i].Wine; w != nil {
			if w.Orphans > rules.WineOrphans {
				return []Alert{{Level: "warn", Code: "wine-orphans", Msg: fmt.Sprintf("%d orphaned winedevice.exe (limit %d, %d in total): ~35 MB each, left by wineserver restarts; tools/game.sh reap iN when the game is stopped (docs/leakhunt.md)", w.Orphans, rules.WineOrphans, w.Total)}}
			}
			return nil
		}
	}
	return nil
}

// ExtraStatus renders the lines that `soak.sh status` adds below the one-line status: the bot's last minute line, game
// restarts in the last hour, the winedevice.exe count. Empty when there is nothing to show.
func ExtraStatus(ss []Sample, rules AlertRules, now time.Time) string {
	var lines []string
	if rules.BotLog != "" {
		if ms, _, err := tailMinutes(rules.BotLog); err != nil {
			lines = append(lines, "bot log: "+err.Error())
		} else if last := lastMinute(ms); last.IsZero() {
			lines = append(lines, "bot log: no minute line yet")
		} else {
			lines = append(lines, fmt.Sprintf("bot log: last minute line %s ago, %d game restarts in the last hour", mins(now.Sub(last)), gameRestartsSince(ms, now.Add(-time.Hour))))
		}
	}
	if w := summarizeWine(ss); w.Samples > 0 {
		lines = append(lines, w.text())
	}
	return strings.Join(lines, "\n")
}
