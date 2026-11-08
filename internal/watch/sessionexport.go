package watch

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ExportOptions controls the opt-in anonymised copy of a session's decision dataset (docs/research/decision-dataset.md,
// "Privacy"). Names lists strings that must not survive (profile, account, character names); Rand is the source of
// the new session id (crypto/rand when nil).
type ExportOptions struct {
	Names []string
	Rand  io.Reader
}

// exportKeys is the whitelist of fields that leave the machine; anything else in a line is dropped.
var exportKeys = map[string]bool{"v": true, "kind": true, "schema": true, "session": true, "commit": true, "catalogue": true, "window_min": true,
	"id": true, "t_game": true, "t_off_ms": true, "point": true, "ctx": true, "class": true, "build": true, "options": true,
	"default": true, "answer": true, "by": true, "latency_ms": true, "of": true, "life_pct_min": true, "xp_per_h": true,
	"deaths": true, "partial": true}

var (
	tokenLike = regexp.MustCompile(`[A-Za-z0-9_-]{24,}`)
	hexCommit = regexp.MustCompile(`^([0-9a-f]{7,40}|[0-9A-Za-z.+_-]{1,23})$`)
)

// ExportSession writes an anonymised copy of dir/decisions.jsonl to w: a new random session id, the wall time reduced to
// an offset in milliseconds from the start, only whitelisted fields, and every string value that looks like a name from
// opt.Names, a path, an address or a token replaced by "scrubbed". It returns the number of replaced values.
func ExportSession(dir string, w io.Writer, opt ExportOptions) (scrubbed int, err error) {
	f, err := os.Open(filepath.Join(dir, "decisions.jsonl"))
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if opt.Rand == nil {
		opt.Rand = rand.Reader
	}
	var rb [8]byte
	if _, err := io.ReadFull(opt.Rand, rb[:]); err != nil {
		return 0, err
	}
	sid := hex.EncodeToString(rb[:])
	var start time.Time
	bad := func(s string) bool {
		if strings.ContainsAny(s, `/\@`) || tokenLike.MatchString(s) {
			return true
		}
		l := strings.ToLower(s)
		for _, n := range opt.Names {
			if n != "" && strings.Contains(l, strings.ToLower(n)) {
				return true
			}
		}
		return false
	}
	var clean func(v any) any
	clean = func(v any) any {
		switch x := v.(type) {
		case string:
			if bad(x) {
				scrubbed++
				return "scrubbed"
			}
		case []any:
			for i := range x {
				x[i] = clean(x[i])
			}
		case map[string]any:
			for k, e := range x {
				if bad(k) {
					delete(x, k)
					scrubbed++
					continue
				}
				x[k] = clean(e)
			}
		}
		return v
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	bw := bufio.NewWriter(w)
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			return scrubbed, fmt.Errorf("export: bad line: %w", err)
		}
		if v, _ := m["v"].(float64); v > SessionSchema {
			return scrubbed, fmt.Errorf("export: dataset version %v is newer than this build understands (%d)", v, SessionSchema)
		}
		if st, ok := m["start"].(string); ok {
			start, _ = time.Parse(time.RFC3339, st)
		}
		if tw, ok := m["t_wall"].(string); ok && !start.IsZero() {
			if t, e := time.Parse(time.RFC3339, tw); e == nil {
				m["t_off_ms"] = t.Sub(start).Milliseconds()
			}
		}
		out := map[string]any{}
		for k, v := range m {
			if !exportKeys[k] {
				continue
			}
			if k == "commit" {
				if s, _ := v.(string); !hexCommit.MatchString(s) {
					scrubbed++
					v = "scrubbed"
				}
				out[k] = v
				continue
			}
			out[k] = clean(v)
		}
		if _, ok := out["session"]; ok {
			out["session"] = sid
		}
		b, _ := json.Marshal(out)
		bw.Write(append(b, '\n'))
	}
	if err := sc.Err(); err != nil {
		return scrubbed, err
	}
	return scrubbed, bw.Flush()
}
