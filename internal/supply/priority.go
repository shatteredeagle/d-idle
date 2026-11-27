package supply

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
)

//go:embed priority.json
var priorityJSON []byte

// futureLines are lines of table 4.2 that the plan does not model yet; the data lists them, Solve skips them.
var futureLines = map[Line]bool{"repair": true, "ammo": true, "merc_revive": true}

type stepRow struct {
	Prio int    `json:"prio"`
	Line Line   `json:"line"`
	To   string `json:"to"`
}

type priorityFile struct {
	Order    []stepRow            `json:"order"`
	Profiles map[string][]stepRow `json:"profiles"`
}

func toSteps(rows []stepRow, what string) ([]Step, error) {
	var out []Step
	last := 0
	for _, r := range rows {
		if !lines[r.Line] && !futureLines[r.Line] {
			return nil, fmt.Errorf("supply: priority.json: %s names unknown line %q", what, r.Line)
		}
		switch r.To {
		case "one", "min", "target":
		default:
			return nil, fmt.Errorf("supply: priority.json: %s line %q has step %q (want one, min or target)", what, r.Line, r.To)
		}
		if r.Prio < last {
			return nil, fmt.Errorf("supply: priority.json: %s line %q at prio %d is out of order", what, r.Line, r.Prio)
		}
		last = r.Prio
		if lines[r.Line] {
			out = append(out, Step{r.Line, r.To})
		}
	}
	return out, nil
}

func loadPriority() ([]Step, map[string][]Step, error) {
	var f priorityFile
	if err := json.Unmarshal(priorityJSON, &f); err != nil {
		return nil, nil, fmt.Errorf("supply: priority.json: %w", err)
	}
	def, err := toSteps(f.Order, "order")
	if err != nil {
		return nil, nil, err
	}
	per := map[string][]Step{}
	for name, rows := range f.Profiles {
		if per[name], err = toSteps(rows, "profile "+name); err != nil {
			return nil, nil, err
		}
	}
	return def, per, nil
}

// DefaultPriority is table 4.2 of the design restricted to the modelled lines, read from priority.json (user answer
// Q5): potions before the first scroll (decision D3), the chunk's gold carry at position 6, targets last. The file is
// embedded and checked by a test, so a bad row panics at start-up rather than silently reordering the spend.
var DefaultPriority, profilePriority = mustPriority()

func mustPriority() ([]Step, map[string][]Step) {
	d, p, err := loadPriority()
	if err != nil {
		panic(err)
	}
	return d, p
}

// PriorityOf is the spending order of an excursion profile: its own rows when priority.json has them, else the default.
func PriorityOf(profile string) []Step {
	if s, ok := profilePriority[profile]; ok {
		return s
	}
	return DefaultPriority
}

// PriorityProfiles lists the profiles that carry their own order, sorted.
func PriorityProfiles() []string {
	var out []string
	for n := range profilePriority {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
