package fidprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
)

// Source produces one Input of the character currently loaded on its side. Implementations are
// read-only (cmd/d2fidprobe wires the game.exe memory reader and the d2d control socket).
type Source interface {
	Name() string
	Read(ctx context.Context) (Input, error)
}

// ReadStable reads n times, gap apart, and fails when two sheets differ: the probe only compares a
// character at rest (design section 7, the three-read stability rule). The first sheet is returned.
func ReadStable(ctx context.Context, src Source, n int, gap time.Duration, rules *Rules) (*Sheet, error) {
	var first *Sheet
	for i := 0; i < n; i++ {
		if i > 0 {
			select {
			case <-time.After(gap):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		in, err := src.Read(ctx)
		if err != nil {
			return nil, fmt.Errorf("fidprobe: %s read %d: %w", src.Name(), i+1, err)
		}
		in.Source = src.Name()
		s, err := FromInput(in)
		if err != nil {
			return nil, err
		}
		if first == nil {
			first = s
			continue
		}
		if d := Diff(first, s, []string{"base", "derived", "items", "skills", "position"}, rules); d.Counts[Mismatch] > 0 {
			return nil, fmt.Errorf("fidprobe: %s unstable between read 1 and %d (%d fields differ, first %s): wait until the character is at rest", src.Name(), i+1, d.Counts[Mismatch], firstMismatch(d))
		}
	}
	return first, nil
}

func firstMismatch(r *Report) string {
	for _, c := range r.Cells {
		if c.Class == Mismatch {
			return c.Key
		}
	}
	return ""
}

// ParseInput decodes a recorded snapshot: a plain d2.Snapshot JSON, or the d2dsnap output document
// ({"snapshot":..., "report":{"Unknown":...}, "progress":...}).
func ParseInput(b []byte) (Input, error) {
	var env struct {
		Snapshot *d2.Snapshot `json:"snapshot"`
		Report   *struct{ Unknown map[string]string }
		Progress *d2.PlayerProgress `json:"progress"`
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&env); err != nil {
		return Input{}, fmt.Errorf("fidprobe: snapshot json: %w", err)
	}
	in := Input{Snap: env.Snapshot, Progress: env.Progress}
	if env.Snapshot == nil {
		var s d2.Snapshot
		if err := json.Unmarshal(b, &s); err != nil {
			return Input{}, fmt.Errorf("fidprobe: snapshot json: %w", err)
		}
		in.Snap = &s
	}
	if env.Report != nil {
		in.Unreadable = env.Report.Unknown
	}
	if in.Progress == nil {
		in.ProgressWhy = "recorded document has no progress"
	}
	return in, nil
}

// CheckScenario verifies the character on a sheet is the one the scenario names.
func CheckScenario(sc Scenario, s *Sheet) error {
	if sc.Class == "" {
		return nil // the scenario compares any character
	}
	f, ok := s.Fields["base.class"]
	if !ok || !f.Readable() {
		return fmt.Errorf("fidprobe: %s: class unreadable", s.Source)
	}
	if got := d2.Class(f.I).String(); got != sc.Class {
		return fmt.Errorf("fidprobe: %s: scenario %s needs a %s, the loaded character is a %s", s.Source, sc.Name, sc.Class, got)
	}
	if d := s.Fields["base.difficulty"]; d.Readable() && int(d.I) != sc.Difficulty {
		return fmt.Errorf("fidprobe: %s: scenario %s needs difficulty %d, loaded %d", s.Source, sc.Name, sc.Difficulty, d.I)
	}
	return nil
}

// Run reads both sources and diffs them for the scenario.
func Run(ctx context.Context, sc Scenario, a, b Source, reads int, gap time.Duration, rules *Rules) (*Report, error) {
	rep, _, _, err := RunSheets(ctx, sc, a, b, reads, gap, rules)
	return rep, err
}

// RunSheets is Run that also returns the two sheets (for a third reference such as the file).
func RunSheets(ctx context.Context, sc Scenario, a, b Source, reads int, gap time.Duration, rules *Rules) (*Report, *Sheet, *Sheet, error) {
	sa, err := readSide(ctx, a, reads, gap, rules)
	if err != nil {
		return nil, nil, nil, err
	}
	sb, err := readSide(ctx, b, reads, gap, rules)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, s := range []*Sheet{sa, sb} {
		if err := CheckScenario(sc, s); err != nil {
			return nil, nil, nil, err
		}
	}
	rep := Diff(sa, sb, sc.Groups, rules)
	rep.Scenario = sc.Name
	return rep, sa, sb, nil
}

// readSide is ReadStable, or the direct sheet for a file source (a file does not change between reads).
func readSide(ctx context.Context, src Source, n int, gap time.Duration, rules *Rules) (*Sheet, error) {
	if ss, ok := src.(SheetSource); ok {
		return ss.Sheet()
	}
	return ReadStable(ctx, src, n, gap, rules)
}
