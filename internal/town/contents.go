package town

import (
	"fmt"
	"sort"
	"strings"

	"d2lod-bot/internal/d2"
)

// contentsOf summarizes a container as "code xN" pairs, biggest first, so the inventory log line names what fills
// the pack and the stash (the log used to show only percentages: INVREJUV found the rejuvenation hoard in a save file).
func (t *Town) contentsOf(s *d2.Snapshot, c d2.Container) string {
	n := map[string]int{}
	for _, u := range s.Inventory.Container(c) {
		code := "?"
		if b := t.base(u); b != nil {
			code = b.Code
		}
		n[code]++
	}
	codes := make([]string, 0, len(n))
	for k := range n {
		codes = append(codes, k)
	}
	sort.Slice(codes, func(i, j int) bool {
		if n[codes[i]] != n[codes[j]] {
			return n[codes[i]] > n[codes[j]]
		}
		return codes[i] < codes[j]
	})
	parts := make([]string, len(codes))
	for i, k := range codes {
		parts[i] = fmt.Sprintf("%s x%d", k, n[k])
	}
	return strings.Join(parts, ", ")
}
