package d2dsoakstat

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// KillTime is one `msg=killtime` line of the bot log (internal/bot/killtime.go; buckets defined there) plus the
// context seen in the log between the previous kill and this one: how often each other message occurred and the
// goal/step/rstate of the last minute line before it.
type KillTime struct {
	Monster                      string
	Area                         int
	Total, Walk, Cast, Wait, Gap float64
	State                        string
	Msgs                         map[string]int
	Goal, Step, RState           string
}

// ParseKillTimes reads slog JSON or text lines in file order; `last line repeated` is ignored (a repeated killtime
// line carries different numbers, so it would only hide kills: kills counted from it are lost, U small).
func ParseKillTimes(r io.Reader) ([]KillTime, error) {
	var out []KillTime
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	msgs := map[string]int{}
	var goal, step, rstate string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		var kv map[string]string
		switch {
		case strings.HasPrefix(line, "{"):
			kv = jsonKV(line)
		case strings.Contains(line, "msg="):
			kv = textKV(line)
		}
		if kv == nil {
			continue
		}
		switch kv["msg"] {
		case "minute":
			goal, step, rstate = kv["goal"], kv["step"], kv["rstate"]
		case "killtime":
			k := KillTime{Monster: kv["monster"], State: kv["state"], Msgs: msgs, Goal: goal, Step: step, RState: rstate}
			k.Area, _ = strconv.Atoi(kv["area"])
			for key, p := range map[string]*float64{"total": &k.Total, "walk": &k.Walk, "cast": &k.Cast, "wait": &k.Wait, "gap_since_last_kill": &k.Gap} {
				*p, _ = strconv.ParseFloat(kv[key], 64)
			}
			out = append(out, k)
			msgs = map[string]int{}
		default:
			if m := kv["msg"]; m != "" {
				msgs[m]++
			}
		}
	}
	return out, sc.Err()
}

// KillSummary aggregates kill times: seconds per kill by bucket (sums of total over kills, with the gap beyond total
// as `out`), a histogram of the per-kill total, and the longest gaps.
type KillSummary struct {
	Kills                      int
	Walk, Cast, Wait, Out      float64 // seconds per kill
	Hist                       []HistBin
	Top                        []KillTime
	ByStep                     map[string]StepStat // per last-minute-line step: kills, summed gap and wait seconds
	WaitOver30Share, WaitShare float64             // share of all wait seconds that sit in kills with total over 30 s; wait share of total
}

type StepStat struct {
	Kills     int
	Gap, Wait float64
}

type HistBin struct {
	Label string
	Kills int
	Secs  float64 // total seconds in the bin
}

var histEdges = []struct {
	label string
	hi    float64
}{{"<=4s", 4}, {"4-10s", 10}, {"10-30s", 30}, {"30-120s", 120}, {">120s", 1e18}}

// SummarizeKills builds the summary; top is the number of longest gaps to keep.
func SummarizeKills(ks []KillTime, top int) KillSummary {
	s := KillSummary{Kills: len(ks)}
	if len(ks) == 0 {
		return s
	}
	s.Hist = make([]HistBin, len(histEdges))
	for i := range s.Hist {
		s.Hist[i].Label = histEdges[i].label
	}
	var tot, waitLong, wait float64
	s.ByStep = map[string]StepStat{}
	for _, k := range ks {
		st := s.ByStep[stepKey(k.Step)]
		st.Kills++
		st.Gap += k.Gap
		st.Wait += k.Wait
		s.ByStep[stepKey(k.Step)] = st
		s.Walk += k.Walk
		s.Cast += k.Cast
		s.Wait += k.Wait
		if k.Gap > k.Total {
			s.Out += k.Gap - k.Total
		}
		wait += k.Wait
		tot += k.Total
		if k.Gap > 30 {
			waitLong += k.Wait
		}
		for i, e := range histEdges {
			if k.Gap <= e.hi {
				s.Hist[i].Kills++
				s.Hist[i].Secs += k.Gap
				break
			}
		}
	}
	n := float64(len(ks))
	s.Walk, s.Cast, s.Wait, s.Out = s.Walk/n, s.Cast/n, s.Wait/n, s.Out/n
	if wait > 0 {
		s.WaitOver30Share = waitLong / wait
	}
	if tot > 0 {
		s.WaitShare = wait / tot
	}
	sorted := append([]KillTime(nil), ks...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Gap > sorted[j].Gap })
	if top > len(sorted) {
		top = len(sorted)
	}
	s.Top = sorted[:top]
	return s
}

// Text prints the summary for humans.
func (s KillSummary) Text(name string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "killtime %s: %d kills, s/kill walk %.1f cast %.1f wait %.1f out-of-game %.1f (total %.1f)\n", name, s.Kills,
		s.Walk, s.Cast, s.Wait, s.Out, s.Walk+s.Cast+s.Wait+s.Out)
	if s.Kills == 0 {
		return b.String()
	}
	fmt.Fprintf(&b, "  wait share of in-game time %.0f %%, of wait seconds in gaps over 30 s %.0f %%\n", 100*s.WaitShare, 100*s.WaitOver30Share)
	for _, h := range s.Hist {
		fmt.Fprintf(&b, "  gap %-8s %5d kills  %5.1f s per kill overall\n", h.Label, h.Kills, h.Secs/float64(s.Kills))
	}
	var keys []string
	for k := range s.ByStep {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return s.ByStep[keys[i]].Gap > s.ByStep[keys[j]].Gap })
	for _, k := range keys {
		st := s.ByStep[k]
		fmt.Fprintf(&b, "  step %-34q %4d kills  %5.1f s gap per kill (wait %.1f), %.0f %% of all gap seconds\n", k, st.Kills,
			st.Gap/float64(st.Kills), st.Wait/float64(st.Kills), 100*st.Gap/(s.sumGap()+1e-9))
	}
	for i, k := range s.Top {
		fmt.Fprintf(&b, "  top%d gap %.0fs (walk %.0f cast %.0f wait %.0f) %s area %d, runner %s, last minute goal=%q step=%q rstate=%s; between kills: %s\n",
			i+1, k.Gap, k.Walk, k.Cast, k.Wait, k.Monster, k.Area, k.State, k.Goal, k.Step, k.RState, topMsgs(k.Msgs))
	}
	return b.String()
}

// stepKey drops the parenthesised detail of a step ("Quest: travelling (Go to X)" -> "Quest: travelling").
func stepKey(step string) string {
	if i := strings.Index(step, " ("); i > 0 {
		step = step[:i]
	}
	return step
}

func topMsgs(m map[string]int) string {
	type kv struct {
		k string
		n int
	}
	var l []kv
	for k, n := range m {
		l = append(l, kv{k, n})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n || l[i].n == l[j].n && l[i].k < l[j].k })
	if len(l) > 6 {
		l = l[:6]
	}
	var p []string
	for _, e := range l {
		p = append(p, fmt.Sprintf("%s x%d", e.k, e.n))
	}
	if len(p) == 0 {
		return "(nothing logged)"
	}
	return strings.Join(p, ", ")
}

func (s KillSummary) sumGap() float64 {
	var g float64
	for _, st := range s.ByStep {
		g += st.Gap
	}
	return g
}
