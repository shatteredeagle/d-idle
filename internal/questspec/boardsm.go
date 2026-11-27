package questspec

import (
	"fmt"
	"sort"

	"d2lod-bot/internal/questboard"
)

// Rules 12 to 14 are the checks of the board state machine (card bsmchk; docs/design/fable-review-2026-11-21
// quest-board-appendix-b-tables.md, "questspec checker rules"). They read the shipped data only.

// CheckOrder is rule 12: in every quest the chunk order of data/questboard is a topological order of `after`. The
// executor picks the head by data order (internal/questboard exec.go), and the loader only rejects cycles, so a chunk
// listed before one it waits for would be a head that can never run.
func CheckOrder(b *questboard.Board) []Finding { return checkOrder(b.Quests()) }

func checkOrder(quests []questboard.Quest) []Finding {
	var out []Finding
	for _, q := range quests {
		idx := map[string]int{}
		for i, c := range q.Chunks {
			idx[c.ID] = i
		}
		for i, c := range q.Chunks {
			for _, a := range c.After {
				j, ok := idx[a]
				switch {
				case !ok:
					out = append(out, Finding{Rule: 12, Step: q.ID + "/" + c.ID, Msg: fmt.Sprintf("after names %q, which is not a chunk of the quest", a)})
				case j >= i:
					out = append(out, Finding{Rule: 12, Step: q.ID + "/" + c.ID, Msg: fmt.Sprintf("chunk %s comes before %s, which it waits for: data order is not a topological order of after", c.ID, a)})
				}
			}
		}
	}
	return out
}

// volatileOnly reports whether a step's posts can only be read from the running game on caps: some post alternative is
// readable, and every readable one is volatile. After a restart nothing durable says the step happened, so the board has
// to latch its chunk.
func volatileOnly(s Step, caps *Caps) bool {
	vol := false
	for _, cl := range s.PostClauses() {
		for _, p := range cl {
			if !caps.readable(p) {
				continue
			}
			if !caps.volatile(p) {
				return false
			}
			vol = true
		}
	}
	return vol
}

// CheckLatched is rule 13: a chunk whose last chain step has only volatile readers for its posts (on any capability
// set) is listed in the chain's `latched`, and `latched` lists nothing else. U: the list the board latches is not data
// yet (the latch is written at run time, store_sm.go), so only the chain side is compared.
func CheckLatched(c *Chain, caps []*Caps) []Finding {
	var out []Finding
	last := map[string]Step{}
	var order []string
	for _, s := range c.Steps {
		if s.Chunk == "" {
			continue
		}
		if _, ok := last[s.Chunk]; !ok {
			order = append(order, s.Chunk)
		}
		last[s.Chunk] = s
	}
	need := map[string]string{}
	for _, ch := range order {
		for _, cp := range caps {
			if volatileOnly(last[ch], cp) {
				need[ch] = cp.Backend
				break
			}
		}
	}
	marked := set(c.Latched...)
	for _, ch := range order {
		if b, ok := need[ch]; ok && !marked[ch] {
			out = append(out, Finding{Rule: 13, Step: last[ch].ID, Msg: fmt.Sprintf("last step of chunk %s has only volatile readers on %s: list the chunk in latched", ch, b)})
		}
	}
	var extra []string
	for ch := range marked {
		if _, ok := last[ch]; !ok {
			out = append(out, Finding{Rule: 13, Step: "*", Msg: fmt.Sprintf("latched names %q, which is no chunk of the chain", ch)})
		} else if _, ok := need[ch]; !ok {
			extra = append(extra, ch)
		}
	}
	sort.Strings(extra)
	for _, ch := range extra {
		out = append(out, Finding{Rule: 13, Step: last[ch].ID, Msg: fmt.Sprintf("chunk %s is listed in latched but its last step has a durable reader on every backend", ch)})
	}
	return out
}

// CheckGuardCodes is rule 14: every reason code a guard can return maps to a code of the closed reason table with a
// remedy. known answers that lookup (questboard.ReasonOf) so the guards' package can be scanned from its own test
// without an import cycle. src names where the codes were found.
func CheckGuardCodes(src string, codes []string, known func(code string) (remedy string, ok bool)) []Finding {
	var out []Finding
	for _, code := range codes {
		r, ok := known(code)
		switch {
		case !ok:
			out = append(out, Finding{Rule: 14, Step: src, Msg: fmt.Sprintf("guard reason %q is not a code of the reason table", code)})
		case r == "":
			out = append(out, Finding{Rule: 14, Step: src, Msg: fmt.Sprintf("guard reason %q has no remedy", code)})
		}
	}
	return out
}
