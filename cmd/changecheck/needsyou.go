package main

import (
	"path/filepath"
	"regexp"
	"strings"
)

// jsonNeed is one open line of docs/needs-you.md (format in that file; tools/needs-you.sh writes it).
type jsonNeed struct {
	ID             string `json:"id"`
	Priority       string `json:"priority"`
	Ask            string `json:"ask"`
	Recommendation string `json:"recommendation"`
	Since          string `json:"since"`
	Link           string `json:"link"`
}

var needRe = regexp.MustCompile(`^- \[ \] (.+)$`)

// parseNeeds reads the open items; malformed lines (fewer than five fields) are skipped.
func parseNeeds(md string) []jsonNeed {
	out := []jsonNeed{}
	for _, line := range strings.Split(md, "\n") {
		m := needRe.FindStringSubmatch(strings.TrimRight(line, " \r"))
		if m == nil {
			continue
		}
		f := strings.Split(m[1], " | ")
		if len(f) < 5 {
			continue
		}
		n := jsonNeed{ID: strings.TrimSpace(f[0]), Priority: strings.TrimSpace(f[1]), Ask: strings.TrimSpace(f[2]),
			Recommendation: strings.TrimSpace(f[3]), Since: strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(f[4]), "since"))}
		if len(f) > 5 {
			n.Link = strings.TrimSpace(f[5])
		}
		switch n.Priority {
		case "now", "soon", "whenever":
		default:
			n.Priority = "whenever"
		}
		out = append(out, n)
	}
	return out
}

func loadNeeds(root string) []jsonNeed {
	md, err := readFile(filepath.Join(root, "docs", "needs-you.md"))
	if err != nil {
		return []jsonNeed{}
	}
	return parseNeeds(md)
}
