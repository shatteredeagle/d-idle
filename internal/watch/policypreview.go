package watch

// Policy preview (card policydryrun, docs/policy-preview.md): what a policy option would have done over the recent
// recorded events, shown before the player saves it. Read-only: it reads the away log and the table, writes nothing.

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
)

// PolicyPreviewRoute is the POST route on the answer listener (same gate as PolicyPostRoute; it never changes state).
const PolicyPreviewRoute = "/api/policy/preview"

const (
	previewDefaultMin = 60
	previewMaxMin     = 24 * 60
)

var (
	previewOnce  sync.Once
	previewSpecs *decide.PreviewSpecs
	previewErr   error
	baselineOnce sync.Once
	baselineMap  map[string]float64
)

func shippedPreview() (*decide.PreviewSpecs, error) {
	previewOnce.Do(func() { previewSpecs, previewErr = decide.ShippedPreviewSpecs() })
	return previewSpecs, previewErr
}

// profileKnob is a tuning knob's default value, the value an option without its own knob leaves in force.
func profileKnob(key string) (float64, bool) {
	baselineOnce.Do(func() {
		baselineMap = map[string]float64{}
		for _, k := range env.Knobs() {
			baselineMap[k.Key] = k.Default
		}
	})
	v, ok := baselineMap[key]
	return v, ok
}

// Preview replays the recent events against option optID of row rowID. minutes <= 0 means the last hour. An unknown
// row or option, or a row that is not for the hero's class, is an error naming it; a row without evidence returns a
// result that says so.
func (p *PolicySheet) Preview(rowID, optID string, minutes int) (decide.PreviewResult, error) {
	class, rows, err := p.rows()
	if err != nil {
		return decide.PreviewResult{}, err
	}
	row, err := p.table.Row(rowID)
	if err != nil {
		return decide.PreviewResult{}, fmt.Errorf("%w %q", ErrPolicyRow, safeText(rowID, 40))
	}
	in := false
	for _, r := range rows {
		in = in || r == row
	}
	if !in {
		return decide.PreviewResult{}, fmt.Errorf("%w: %q, class %q", ErrPolicyScope, row.ID, class)
	}
	if _, err := row.Option(optID); err != nil {
		return decide.PreviewResult{}, fmt.Errorf("%w %q of row %q", ErrPolicyOption, safeText(optID, 40), row.ID)
	}
	sp, err := shippedPreview()
	if err != nil {
		return decide.PreviewResult{}, err
	}
	if minutes <= 0 {
		minutes = previewDefaultMin
	}
	if minutes > previewMaxMin {
		minutes = previewMaxMin
	}
	var evs []decide.PreviewEvent
	if p.events != nil {
		for _, e := range p.events.Events() {
			evs = append(evs, decide.PreviewEvent{T: e.T, Kind: e.Kind, LifePct: e.LifePct, Item: e.Item, Quality: e.Quality, Type: e.Type})
		}
	}
	return sp.Preview(decide.PreviewInput{Row: row, Class: class, Current: p.current(row), New: optID, Baseline: profileKnob,
		Events: evs, Now: p.now(), Window: float64(minutes) * 60})
}

type previewBody struct {
	Row     string `json:"row"`
	Option  string `json:"option"`
	Minutes int    `json:"minutes,omitempty"`
}

// PolicyPreviewHandler is POST /api/policy/preview {row, option, minutes?}: the gate of the answer routes (origin list,
// token, JSON only, small body, known fields), then a read-only computation. It cannot change the sheet.
func (p *PolicySheet) PolicyPreviewHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b previewBody
		if !policyGate(o, w, r, &b) {
			return
		}
		fail := func(code int, c, msg string) { writeJSON(w, code, policyReply{Code: c, Error: msg}) }
		res, err := p.Preview(b.Row, b.Option, b.Minutes)
		switch {
		case err == nil:
			writeJSON(w, 200, res)
		case errors.Is(err, ErrPolicyRow):
			fail(http.StatusNotFound, "row", err.Error())
		case errors.Is(err, ErrPolicyOption), errors.Is(err, ErrPolicyScope):
			fail(http.StatusBadRequest, "option", err.Error())
		case errors.Is(err, ErrPolicyNoClass):
			fail(http.StatusServiceUnavailable, "class", err.Error())
		case errors.Is(err, decide.ErrPreviewKnob), strings.HasPrefix(err.Error(), "decide: policy option"):
			fail(http.StatusInternalServerError, "preview", err.Error())
		default:
			fail(http.StatusInternalServerError, "internal", "could not compute the preview")
		}
	})
}
