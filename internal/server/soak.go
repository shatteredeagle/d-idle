package server

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// soakMaxFile bounds what is read from a soak file; both are small and written by another tool.
const soakMaxFile = 64 << 10

type soakAlert struct {
	Level string `json:"level"`
	Code  string `json:"code,omitempty"`
	Msg   string `json:"msg"`
}

// SoakView is the body of /api/v1/soak: the newest soak run's alerts.json and status.txt (written by the soak
// tooling, ~/d2lod-live/soak/<run>/; this server only reads them).
type SoakView struct {
	Run     string      `json:"run"`
	Updated string      `json:"updated,omitempty"` // as written by the soak tool
	AgeSec  int         `json:"ageSec"`            // since the newest file in the run changed
	Level   string      `json:"level"`             // ok, warn, crit, or unknown when alerts.json is missing/unreadable
	Alerts  []soakAlert `json:"alerts"`
	Status  string      `json:"status,omitempty"` // status.txt, or the "status" field of alerts.json
	Error   string      `json:"error,omitempty"`
}

func readSmall(path string) ([]byte, time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}
	b, err := io.ReadAll(io.LimitReader(f, soakMaxFile))
	return b, fi.ModTime(), err
}

// newestRun picks the run directory under root whose alerts.json or status.txt changed last; ties and
// directories without either file fall back to the lexically last name (run names are timestamps).
func newestRun(root string) (string, bool) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return "", false
	}
	var best string
	var bestT time.Time
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		var t time.Time
		for _, n := range []string{"alerts.json", "status.txt"} {
			if fi, err := os.Stat(filepath.Join(root, e.Name(), n)); err == nil && fi.ModTime().After(t) {
				t = fi.ModTime()
			}
		}
		if best == "" || t.After(bestT) || (t.Equal(bestT) && e.Name() > best) {
			best, bestT = e.Name(), t
		}
	}
	return best, best != ""
}

// LoadSoak reads the newest run under root. found is false when there is no run directory at all.
func LoadSoak(root string, now time.Time) (v SoakView, found bool) {
	run, ok := newestRun(root)
	if !ok {
		return SoakView{}, false
	}
	v = SoakView{Run: run, Level: "unknown", Alerts: []soakAlert{}}
	dir := filepath.Join(root, run)
	var newest time.Time
	if b, mt, err := readSmall(filepath.Join(dir, "alerts.json")); err != nil {
		v.Error = "alerts.json: " + err.Error()
	} else {
		newest = mt
		var a struct {
			Updated any         `json:"updated"`
			Level   string      `json:"level"`
			Alerts  []soakAlert `json:"alerts"`
			Status  string      `json:"status"`
		}
		if err := json.Unmarshal(b, &a); err != nil {
			v.Error = "alerts.json: " + err.Error()
		} else {
			if s, ok := a.Updated.(string); ok {
				v.Updated = s
			}
			if a.Level != "" {
				v.Level = a.Level
			}
			if a.Alerts != nil {
				v.Alerts = a.Alerts
			}
			v.Status = a.Status
		}
	}
	if b, mt, err := readSmall(filepath.Join(dir, "status.txt")); err == nil {
		if mt.After(newest) {
			newest = mt
		}
		if s := trimLine(string(b)); s != "" {
			v.Status = s
		}
	}
	if !newest.IsZero() {
		v.AgeSec = max(0, int(now.Sub(newest).Seconds()))
	}
	return v, true
}

// trimLine returns the first non-empty line, trimmed.
func trimLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return ""
}
