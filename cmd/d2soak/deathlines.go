package main

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// DeathLine is one per-death telemetry line of the bot log: "death diff= area= clvl= mlvl= gap= killer= class= boss=
// affixes= dist= build=" (bot.recordDeath, combat.DeathInfo.Line plus the build). mlvl and gap are 0 when the monster
// level was unknown.
type DeathLine struct {
	Build, Diff, Killer, Affixes string
	Area, CLvl, MLvl, Gap        int
	Boss                         bool
}

// ParseDeathLines reads a bot log (JSON or slog text; the death line is inside msg="death ...") and returns the lines in order.
// Unrelated and malformed lines (no diff= or clvl=) are skipped.
func ParseDeathLines(r io.Reader) ([]DeathLine, error) {
	var out []DeathLine
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		body, _, ok := telemetryBody(sc.Text(), "death diff=")
		if !ok {
			continue
		}
		kv := map[string]string{}
		for _, f := range strings.Fields(body) {
			if k, v, ok := strings.Cut(f, "="); ok {
				kv[k] = strings.Trim(v, `"`)
			}
		}
		n := func(k string) int { v, _ := strconv.Atoi(kv[k]); return v }
		if _, ok := kv["clvl"]; !ok {
			continue
		}
		out = append(out, DeathLine{Build: kv["build"], Diff: kv["diff"], Killer: kv["killer"], Affixes: kv["affixes"],
			Area: n("area"), CLvl: n("clvl"), MLvl: n("mlvl"), Gap: n("gap"), Boss: kv["boss"] == "true"})
	}
	return out, sc.Err()
}

// DeathText renders deaths per build and difficulty with the mean mlvl-clvl gap (over lines with a known mlvl) and the
// commonest affix, for the soak report.
func DeathText(ds []DeathLine) string {
	if len(ds) == 0 {
		return "deaths: none\n"
	}
	type agg struct {
		n, gapN, gapSum, maxClvl int
		aff                      map[string]int
	}
	by := map[string]*agg{}
	for _, d := range ds {
		k := d.Build + " " + d.Diff
		a := by[k]
		if a == nil {
			a = &agg{aff: map[string]int{}}
			by[k] = a
		}
		a.n++
		a.maxClvl = max(a.maxClvl, d.CLvl)
		if d.MLvl > 0 {
			a.gapN++
			a.gapSum += d.Gap
		}
		for _, f := range strings.Split(d.Affixes, ",") {
			if f != "" && f != "-" {
				a.aff[f]++
			}
		}
	}
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("deaths: build difficulty count mean-gap(mlvl-clvl) top-affix\n")
	for _, k := range keys {
		a := by[k]
		gap, top := "?", "-"
		if a.gapN > 0 {
			gap = fmt.Sprintf("%+.1f", float64(a.gapSum)/float64(a.gapN))
		}
		best := 0
		for f, c := range a.aff {
			if c > best || c == best && f < top {
				best, top = c, f
			}
		}
		fmt.Fprintf(&b, "  %s %d %s %s\n", k, a.n, gap, top)
	}
	return b.String()
}
