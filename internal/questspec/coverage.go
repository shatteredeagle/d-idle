package questspec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"d2lod-bot/internal/questboard"
)

// PendingSchema is the schema of data/questspec/pending.json.
const PendingSchema = "questpending/1"

// Pending is one board quest that has no step chain yet: who owns writing it and what stands in the way. The list is
// the honest remainder of the step executor's coverage (roadmap row R13), not a way to silence the check: a quest
// leaves it only by getting a chain.
type Pending struct {
	Quest string `json:"quest"`
	Owner string `json:"owner"`
	Why   string `json:"why"`
}

// PendingFile is data/questspec/pending.json.
type PendingFile struct {
	Schema  string    `json:"schema"`
	Pending []Pending `json:"pending"`
}

// LoadPending parses the pending list strictly.
func LoadPending(data []byte) (*PendingFile, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var p PendingFile
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("questspec: pending: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("questspec: pending: trailing data")
	}
	if p.Schema != PendingSchema {
		return nil, fmt.Errorf("questspec: pending schema %q, want %q", p.Schema, PendingSchema)
	}
	return &p, nil
}

// ActCoverage counts the board quests of one act by how the step executor knows them.
type ActCoverage struct {
	Act     int
	Chained []string // quests with a chain, board order
	Pending []string // quests with a pending row, board order
}

// Coverage says, per act, which board quests have a chain and which are pending, and returns a finding for every
// quest that is neither (a quest the step executor silently does not know), for a pending row that is stale (the
// quest has a chain), unknown, duplicated or without owner and reason, and for a chain whose quest is not on the board.
func Coverage(b *questboard.Board, chains []*Chain, p *PendingFile) ([]ActCoverage, []string) {
	var bad []string
	chained := map[string]bool{}
	for _, c := range chains {
		if c.Quest == "" {
			continue
		}
		if _, err := b.Quest(c.Quest); err != nil {
			bad = append(bad, fmt.Sprintf("chain %s: %v", c.ID, err))
		}
		chained[c.Quest] = true
	}
	pending := map[string]bool{}
	for i, e := range p.Pending {
		at := fmt.Sprintf("pending row %d (%s)", i, e.Quest)
		switch _, err := b.Quest(e.Quest); {
		case err != nil:
			bad = append(bad, fmt.Sprintf("%s: %v", at, err))
		case pending[e.Quest]:
			bad = append(bad, at+": listed twice")
		case chained[e.Quest]:
			bad = append(bad, at+": the quest has a chain now: remove the row")
		case e.Owner == "" || e.Why == "":
			bad = append(bad, at+": owner and why are required")
		}
		pending[e.Quest] = true
	}
	acts := map[int]*ActCoverage{}
	for _, q := range b.Quests() {
		a := acts[q.Act]
		if a == nil {
			a = &ActCoverage{Act: q.Act}
			acts[q.Act] = a
		}
		switch {
		case chained[q.ID]:
			a.Chained = append(a.Chained, q.ID)
		case pending[q.ID]:
			a.Pending = append(a.Pending, q.ID)
		default:
			bad = append(bad, fmt.Sprintf("act %d quest %s: no chain in data/questspec and no pending row with a reason", q.Act, q.ID))
		}
	}
	var out []ActCoverage
	for _, a := range acts {
		out = append(out, *a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Act < out[j].Act })
	return out, bad
}
