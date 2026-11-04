package d2dsoakstat

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// SkillStats are the per-skill fields of docs/research/sorc-act-gate.md section 2 that the bot log can supply
// (V by unit test on fixture lines; the message names are read from internal/combat: "combat decision" act=cast|nova
// at debug level, "combat buff", "combat buff cut by a hit, will retry"). Skills are keyed by the name the log
// carries (from the skill data), never by id.
type SkillStats struct {
	Casts          map[string]int // casts started per skill (combat decision act=cast)
	Buffs          map[string]int // buff casts per skill (combat buff)
	BuffRetries    map[string]int // buff casts cut by a hit that will be recast
	Novas          int            // crowd novas (act=nova)
	NovaMonsters   int            // sum of the monsters counted per nova
	Kills          int            // killtime lines (all skills together)
	Branch, Commit string         // d2d build, from any line that carries the keys; "" = not logged

	// docs/logfields.md: `cast result`, `combat kill` last_skill and the minute line armor sample. Absent in old logs.
	Results   map[string]int // cast result lines per skill
	Made      map[string]int // ... that made the effect
	Cut       map[string]int // ... cut before the action frame
	Mana      map[string]int // whole mana spent, summed over the result lines
	Ticks     map[string]int // modelled cast length in ticks (last value seen)
	KilledBy  map[string]int // kills whose last cast was this skill
	WinKills  map[string]int // kills noted inside the cast window of a result line (docs/killcredit.md)
	Targets   map[string]int // monsters in the aimed area, summed over casts that logged `targets`
	TargetN   map[string]int // casts that logged `targets`
	ArmorUp   int            // minute samples with an armor state
	ArmorNone int            // minute samples with no armor state
	ArmorUnk  int            // minute samples where the state array was not readable
}

func newSkillStats() SkillStats {
	return SkillStats{Casts: map[string]int{}, Buffs: map[string]int{}, BuffRetries: map[string]int{},
		Results: map[string]int{}, Made: map[string]int{}, Cut: map[string]int{}, Mana: map[string]int{}, Ticks: map[string]int{}, KilledBy: map[string]int{},
		WinKills: map[string]int{}, Targets: map[string]int{}, TargetN: map[string]int{}}
}

// ParseSkills reads slog JSON or text lines.
func ParseSkills(r io.Reader) (SkillStats, error) {
	s := newSkillStats()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
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
		if s.Branch == "" {
			s.Branch = kv["branch"]
		}
		if s.Commit == "" {
			s.Commit = kv["commit"]
		}
		switch kv["msg"] {
		case "combat decision":
			switch kv["act"] {
			case "cast":
				if kv["skill"] != "" {
					s.Casts[kv["skill"]]++
				}
			case "nova":
				n, err := strconv.Atoi(kv["n"])
				if err == nil && n >= 0 {
					s.Novas++
					s.NovaMonsters += n
				}
			}
		case "combat buff":
			if kv["skill"] != "" {
				s.Buffs[kv["skill"]]++
			}
		case "combat buff cut by a hit, will retry":
			if kv["skill"] != "" {
				s.BuffRetries[kv["skill"]]++
			}
		case "killtime":
			s.Kills++
		case "combat kill":
			if kv["last_skill"] != "" {
				s.KilledBy[kv["last_skill"]]++
			}
		case "cast result":
			k := kv["skill"]
			if k == "" {
				break
			}
			s.Results[k]++
			if kv["made"] == "true" {
				s.Made[k]++
			}
			if kv["cut"] == "true" {
				s.Cut[k]++
			}
			if n, err := strconv.Atoi(kv["cost"]); err == nil && n > 0 {
				s.Mana[k] += n
			}
			if n, err := strconv.Atoi(kv["kills"]); err == nil && n > 0 {
				s.WinKills[k] += n
			}
			if n, err := strconv.Atoi(kv["targets"]); err == nil && n > 0 {
				s.Targets[k] += n
				s.TargetN[k]++
			}
			if n, err := strconv.Atoi(kv["ticks"]); err == nil && n > 0 {
				s.Ticks[k] = n
			}
		case "minute":
			switch a := kv["armor"]; a {
			case "":
			case "?":
				s.ArmorUnk++
			case "none":
				s.ArmorNone++
			default:
				s.ArmorUp++
			}
		}
	}
	return s, sc.Err()
}

const na = "n/a (not logged)"

// Text is the per-skill block of the report. melee is the name of the basic-attack skill in the data ("" = unknown):
// its casts are the staff swings. Fields the log cannot supply print n/a; docs/soakskills.md lists them.
func (s SkillStats) Text(name, melee string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "skills %s: kills %d\n", name, s.Kills)
	names := map[string]bool{}
	for k := range s.Casts {
		names[k] = true
	}
	for k := range s.Buffs {
		names[k] = true
	}
	list := make([]string, 0, len(names))
	for k := range names {
		list = append(list, k)
	}
	sort.Strings(list)
	if len(list) == 0 {
		b.WriteString("  no skill lines in the log (debug level off, or no cast)\n")
	}
	for _, k := range list {
		kind := "attack"
		if s.Buffs[k] > 0 && s.Casts[k] == 0 {
			kind = "buff"
		}
		made, cut, mana, kills := na, na, na, na
		if n := s.Results[k]; n > 0 {
			made = fmt.Sprintf("%d of %d", s.Made[k], n)
			cut = fmt.Sprintf("%d of %d", s.Cut[k], n)
			mana = fmt.Sprintf("%d (%.1f per cast)", s.Mana[k], float64(s.Mana[k])/float64(n))
		}
		if len(s.KilledBy) > 0 {
			kills = strconv.Itoa(s.KilledBy[k]) + " credited (last skill of the kill"
			if len(s.WinKills) > 0 {
				kills += fmt.Sprintf("; %d inside the cast window", s.WinKills[k])
			}
			kills += ")"
		}
		hits := na + ", no missile-hit signal in the bot"
		if n := s.TargetN[k]; n > 0 {
			hits = fmt.Sprintf("%.1f monsters in the area (%d casts, not confirmed hits)", float64(s.Targets[k])/float64(n), n)
		}
		fmt.Fprintf(&b, "  %s (%s): casts started %d, made effect %s, cut before action frame %s, mana spent %s, kills %s, casts/kill %s, hits/cast %s",
			k, kind, s.Casts[k]+s.Buffs[k], made, cut, mana, kills, s.perKillOf(k), hits)
		if s.Buffs[k] > 0 {
			fmt.Fprintf(&b, "; buff recasts after a cut %d, seconds absent %s", s.BuffRetries[k], s.armorAbsent())
		}
		b.WriteString("\n")
	}
	total := 0
	for _, n := range s.Casts {
		total += n
	}
	fmt.Fprintf(&b, "  all attack casts %d, casts/kill %s\n", total, s.perKill(total))
	if s.Novas > 0 {
		fmt.Fprintf(&b, "  novas %d, monsters per nova %.1f\n", s.Novas, float64(s.NovaMonsters)/float64(s.Novas))
	} else {
		b.WriteString("  novas 0\n")
	}
	if melee == "" {
		fmt.Fprintf(&b, "  staff swings %s (melee skill name not given)\n", na)
	} else {
		fmt.Fprintf(&b, "  staff swings %d\n", s.Casts[melee])
	}
	build := na
	if s.Branch != "" || s.Commit != "" {
		build = s.Branch + " " + s.Commit
	}
	fmt.Fprintf(&b, "  d2d build %s; cast length ticks %s\n", build, s.castTicks())
	return b.String()
}

// armorAbsent is the sampled armor-absent share: one sample per minute line, so seconds are a 60 s per sample estimate.
func (s SkillStats) armorAbsent() string {
	if s.ArmorUp+s.ArmorNone == 0 {
		return na
	}
	return fmt.Sprintf("~%d (%d of %d minute samples without armor, %d unreadable)", s.ArmorNone*60, s.ArmorNone, s.ArmorUp+s.ArmorNone, s.ArmorUnk)
}

// castTicks lists the modelled cast length of every skill that logged one (no skill name is hard-coded).
func (s SkillStats) castTicks() string {
	var out []string
	for k, n := range s.Ticks {
		out = append(out, fmt.Sprintf("%s %d", k, n))
	}
	if len(out) == 0 {
		return na
	}
	sort.Strings(out)
	return strings.Join(out, ", ") + " (modelled, castgap.go)"
}

// perKillOf is casts per credited kill of one skill; without credit lines it falls back to all kills.
func (s SkillStats) perKillOf(k string) string {
	if len(s.KilledBy) == 0 {
		return s.perKill(s.Casts[k])
	}
	if s.KilledBy[k] == 0 {
		return "n/a (no kill credited)"
	}
	return fmt.Sprintf("%.1f", float64(s.Casts[k])/float64(s.KilledBy[k]))
}

func (s SkillStats) perKill(casts int) string {
	if s.Kills == 0 {
		return "n/a (no kills)"
	}
	return fmt.Sprintf("%.1f", float64(casts)/float64(s.Kills))
}
