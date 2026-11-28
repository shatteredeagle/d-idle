package watch

import (
	"errors"
	"fmt"
	"net/http"

	"d2lod-bot/internal/env"
)

// The zone clear settings (docs/farmloop.md section "Dashboard"): when a clear run calls a zone explored
// (bot.exploredPct) and the farm loop's reset share (bot.farmResetPct, bot.farmResetCooldownMin).
// GET /api/zoneclear shows the live values with their range, default and source; POST changes them through the tuner
// (range checks, all or nothing, saved to tuning.json next to every other dashboard edit). The bot reads the knobs at
// its next zone decision, so a change needs no restart.

// ZoneClearRoute is both the GET (viewer) and the POST (answer listener) path.
const ZoneClearRoute = "/api/zoneclear"

const zoneClearSchema = "watch.zoneclear/1"

// zoneClearKeys are the knobs of the surface, in display order.
var zoneClearKeys = []string{"bot.exploredPct", "bot.farmResetPct", "bot.farmResetCooldownMin"}

// ZoneClear connects the surface to the tuner: Value reads a knob and its source ("default", "config", "dashboard"),
// Set is the tuner's all-or-nothing edit (nil value resets to the default).
type ZoneClear struct {
	Value func(key string) (float64, string, error)
	Set   KnobSetter
}

// ZoneClearKnob is one setting.
type ZoneClearKnob struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Unit    string  `json:"unit"`
	Desc    string  `json:"desc"`
	Value   float64 `json:"value"`
	Default float64 `json:"default"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Source  string  `json:"source"` // default, config or dashboard
}

// ZoneClearView is GET /api/zoneclear.
type ZoneClearView struct {
	Schema string          `json:"schema"`
	Knobs  []ZoneClearKnob `json:"knobs"`
	// Line is the one-line explanation the panel shows under the sliders.
	Line string `json:"line"`
}

// ZoneClearBody is the POST body: any subset of the settings, or reset. Unknown fields are refused.
type ZoneClearBody struct {
	ExploredPct          *int `json:"exploredPct,omitempty"`
	FarmResetPct         *int `json:"farmResetPct,omitempty"`
	FarmResetCooldownMin *int `json:"farmResetCooldownMin,omitempty"`
	Reset                bool `json:"reset,omitempty"` // every setting back to its default (the other fields are refused with it)
}

var errZoneClear = errors.New("zoneclear")

type zoneReply struct {
	OK    bool           `json:"ok"`
	Code  string         `json:"code,omitempty"`
	Field string         `json:"field,omitempty"`
	Error string         `json:"error,omitempty"`
	View  *ZoneClearView `json:"view,omitempty"`
}

func (z *ZoneClear) knobs() (map[string]env.Knob, error) {
	have := map[string]env.Knob{}
	for _, k := range env.Knobs() {
		have[k.Key] = k
	}
	for _, k := range zoneClearKeys {
		if _, ok := have[k]; !ok {
			return nil, fmt.Errorf("%w: knob %s is not registered", errZoneClear, k)
		}
	}
	return have, nil
}

// View builds the GET reply.
func (z *ZoneClear) View() (ZoneClearView, error) {
	have, err := z.knobs()
	if err != nil {
		return ZoneClearView{}, err
	}
	v := ZoneClearView{Schema: zoneClearSchema, Knobs: []ZoneClearKnob{},
		Line: "Clear a farm zone to the first share, then leave and rejoin for a fresh one at the second."}
	for _, key := range zoneClearKeys {
		k := have[key]
		val, src, err := z.Value(key)
		if err != nil {
			return ZoneClearView{}, err
		}
		v.Knobs = append(v.Knobs, ZoneClearKnob{Key: key, Label: k.Label, Unit: k.Unit, Desc: k.Desc, Value: val, Default: k.Default, Min: k.Min, Max: k.Max, Source: src})
	}
	return v, nil
}

type zoneRefusal struct{ code, field, msg string }

func (r zoneRefusal) Error() string { return r.msg }

// Apply validates a body and applies it; a zoneRefusal names the field and the code (range, empty, reset).
func (z *ZoneClear) Apply(b ZoneClearBody) error {
	have, err := z.knobs()
	if err != nil {
		return err
	}
	over := map[string]*float64{}
	if b.Reset {
		if b.ExploredPct != nil || b.FarmResetPct != nil || b.FarmResetCooldownMin != nil {
			return zoneRefusal{"reset", "reset", "reset cannot be combined with other fields"}
		}
		for _, k := range zoneClearKeys {
			over[k] = nil
		}
		return z.Set(over)
	}
	n := 0
	set := func(key, field string, v float64) error {
		k := have[key]
		if v < k.Min || v > k.Max {
			return zoneRefusal{"range", field, fmt.Sprintf("%s must be %g to %g %s", field, k.Min, k.Max, k.Unit)}
		}
		x := v
		over[key] = &x
		n++
		return nil
	}
	if b.ExploredPct != nil {
		if err := set("bot.exploredPct", "exploredPct", float64(*b.ExploredPct)); err != nil {
			return err
		}
	}
	if b.FarmResetPct != nil {
		if err := set("bot.farmResetPct", "farmResetPct", float64(*b.FarmResetPct)); err != nil {
			return err
		}
	}
	if b.FarmResetCooldownMin != nil {
		if err := set("bot.farmResetCooldownMin", "farmResetCooldownMin", float64(*b.FarmResetCooldownMin)); err != nil {
			return err
		}
	}
	if n == 0 {
		return zoneRefusal{"empty", "", "nothing to change"}
	}
	return z.Set(over)
}

// Handler serves GET /api/zoneclear (mount on the viewer).
func (z *ZoneClear) Handler() http.Handler {
	return readOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, err := z.View()
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, zoneReply{Code: "unavailable", Error: err.Error()})
			return
		}
		writeJSON(w, 200, v)
	}))
}

// PostHandler serves POST /api/zoneclear (mount on the answer listener; same gate as the policy sheet).
func (z *ZoneClear) PostHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b ZoneClearBody
		if !policyGate(o, w, r, &b) {
			return
		}
		if err := z.Apply(b); err != nil {
			var rf zoneRefusal
			switch {
			case errors.As(err, &rf):
				writeJSON(w, http.StatusBadRequest, zoneReply{Code: rf.code, Field: rf.field, Error: rf.msg})
			default:
				writeJSON(w, http.StatusBadRequest, zoneReply{Code: "refused", Error: err.Error()})
			}
			return
		}
		v, err := z.View()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, zoneReply{Code: "internal", Error: err.Error()})
			return
		}
		writeJSON(w, 200, zoneReply{OK: true, View: &v})
	})
}
