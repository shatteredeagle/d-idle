package api

import (
	"errors"
	"net/http"
	"strings"

	denv "d2lod-bot/internal/env"
)

// ErrNoUpgrades is returned by Tuning.Unlock while there is no upgrade data behind the tuning endpoints.
var ErrNoUpgrades = errors.New("api: upgrades are not available on this server")

// Profile is one entry of the profile list.
type Profile struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Desc  string `json:"desc"`
}

// Upgrade is one purchasable step of a tuning track (contract: top of web/panels.js).
type Upgrade struct {
	ID       string          `json:"id"`
	Track    string          `json:"track,omitempty"`
	Tier     int             `json:"tier,omitempty"`
	Label    string          `json:"label"`
	Desc     string          `json:"desc,omitempty"`
	Cost     float64         `json:"cost,omitempty"`
	Requires []string        `json:"requires,omitempty"`
	Effects  []UpgradeEffect `json:"effects,omitempty"`
	State    string          `json:"state"` // locked, available, unlocked
}

type UpgradeEffect struct {
	Knob  string  `json:"knob"`
	Value float64 `json:"value"`
}

// TuneBody is the GET /knobs body.
type TuneBody struct {
	Knobs     []denv.Knob        `json:"knobs"`
	Values    map[string]float64 `json:"values"`
	Profile   string             `json:"profile"`
	Profiles  []Profile          `json:"profiles"`
	Upgrades  []Upgrade          `json:"upgrades"`
	Telemetry map[string]any     `json:"telemetry,omitempty"`
	// ReadOnly tells the dashboard to hide the edit and unlock controls.
	ReadOnly bool `json:"readonly,omitempty"`
}

// TuneEdit is the PUT /knobs body. A nil override value resets that knob to the profile value.
type TuneEdit struct {
	Profile   *string             `json:"profile,omitempty"`
	Overrides map[string]*float64 `json:"overrides,omitempty"`
}

// Tuning is what the knob routes need from the bot (bot.Tuner implements it). Errors wrapping ErrInvalidTune answer
// 400, ErrNoUpgrades 404 (unknown id) or 501 (no upgrade data).
type Tuning interface {
	Snapshot() TuneBody
	Edit(TuneEdit) error
	Unlock(id string) error
}

// ErrInvalidTune marks a rejected edit (unknown knob or profile).
var ErrInvalidTune = errors.New("invalid tuning edit")

// ErrUnknownUpgrade marks an unlock of an id that does not exist.
var ErrUnknownUpgrade = errors.New("unknown upgrade")

// KnobsHandler serves GET and PUT on the path it is mounted at (prefix stripped), for example the instance mount
// "knobs" (/api/v1/instances/{id}/knobs). Both answer with the GET body.
func KnobsHandler(t Tuning) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
		case http.MethodPut:
			var e TuneEdit
			if !decode(w, r, &e) {
				return
			}
			if err := t.Edit(e); err != nil {
				tuneErr(w, err)
				return
			}
		default:
			w.Header().Set("Allow", "GET, HEAD, PUT")
			writeJSON(w, http.StatusMethodNotAllowed, errBody{"method not allowed", "method_not_allowed"})
			return
		}
		writeJSON(w, http.StatusOK, t.Snapshot())
	})
}

// UpgradesHandler serves POST /{id}/unlock on the path it is mounted at (prefix stripped) and answers with the knobs body.
func UpgradesHandler(t Tuning) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, rest, _ := strings.Cut(strings.Trim(r.URL.Path, "/"), "/")
		if id == "" || rest != "unlock" {
			writeJSON(w, http.StatusNotFound, errBody{"not found", "not_found"})
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeJSON(w, http.StatusMethodNotAllowed, errBody{"method not allowed", "method_not_allowed"})
			return
		}
		if err := t.Unlock(id); err != nil {
			tuneErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, t.Snapshot())
	})
}

func tuneErr(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case errors.Is(err, ErrInvalidTune):
		writeJSON(w, http.StatusBadRequest, errBody{msg, "invalid"})
	case errors.Is(err, ErrUnknownUpgrade):
		writeJSON(w, http.StatusNotFound, errBody{msg, "not_found"})
	case errors.Is(err, ErrNoUpgrades):
		writeJSON(w, http.StatusNotImplemented, errBody{msg, "not_implemented"})
	default:
		writeJSON(w, http.StatusInternalServerError, errBody{msg, "internal"})
	}
}
