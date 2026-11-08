package watch

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/decide"
)

// The Codex view (TASK-130, docs/codex.md): the Act 1 firsts the hero has met, with what it decided and a veteran tip.
// It is read-only and keeps its own source, so it can be mounted without touching the Hub; the return screen
// (TASK-127) calls CodexSection for its "firsts" tile.

// CodexRoute is the GET route CodexHandler serves; mount it next to ViewRoutes.
const CodexRoute = "/api/codex"

const codexSchema = "watch.codex/1"

// CodexSource is what the view reads. CodexReader is the shipped one.
type CodexSource interface {
	Summary(window time.Duration) decide.CodexSummary
}

// CodexReader reads a Codex against its table on the run's clock.
type CodexReader struct {
	Codex *decide.Codex
	Table *decide.Firsts
	Clock decide.Clock
}

// Summary implements CodexSource.
func (r CodexReader) Summary(window time.Duration) decide.CodexSummary {
	return r.Codex.Summary(r.Table, r.Clock.Now(), window)
}

// CodexView is the body of /api/codex.
type CodexView struct {
	Schema string `json:"schema"`
	decide.CodexSummary
	Line string `json:"line"`
}

// CodexViewOf builds the view: the summary plus the one-line text of the return screen.
func CodexViewOf(src CodexSource, window time.Duration) CodexView {
	s := src.Summary(window)
	return CodexView{Schema: codexSchema, CodexSummary: s, Line: CodexSection(s)}
}

// CodexSection is the return screen's line for the window: "3 new firsts: Fallen Shaman, Corpsefire, Bishibosh
// (12 of 37 Act 1 entries)", or the count alone when nothing is new.
func CodexSection(s decide.CodexSummary) string {
	total := fmt.Sprintf("%d of %d Act 1 entries", s.Filled, s.Total)
	if len(s.Recent) == 0 {
		return total
	}
	titles := make([]string, 0, len(s.Recent))
	for i := len(s.Recent) - 1; i >= 0; i-- { // oldest first reads as the story of the trip
		titles = append(titles, s.Recent[i].Title)
	}
	noun := "new first"
	if len(titles) != 1 {
		noun = "new firsts"
	}
	return fmt.Sprintf("%d %s: %s (%s)", len(titles), noun, strings.Join(titles, ", "), total)
}

// CodexHandler serves GET /api/codex[?window=<minutes>]; window limits "recent" to the last minutes of virtual time
// (absent or 0: every entry). Other methods are refused, as for the rest of the read-only side.
func CodexHandler(src CodexSource) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "read-only", http.StatusMethodNotAllowed)
			return
		}
		var window time.Duration
		if q := r.URL.Query().Get("window"); q != "" {
			m, err := strconv.Atoi(q)
			if err != nil || m < 0 {
				http.Error(w, "bad window", http.StatusBadRequest)
				return
			}
			window = time.Duration(m) * time.Minute
		}
		writeJSON(w, http.StatusOK, CodexViewOf(src, window))
	})
}

// CodexReturnSection is the return screen's "firsts" section (registry id CodexSectionID). The return window's clock
// (journal seconds) is not the Codex's run clock (U), so it shows the three newest entries and the count, not a window.
func CodexReturnSection(src CodexSource) ReturnSection {
	return SectionFunc(func(*ReturnData) *SectionView {
		s := src.Summary(0)
		v := &SectionView{ID: CodexSectionID, Title: "Codex", Placeholder: fmt.Sprintf("%d of %d Act 1 entries", s.Filled, s.Total)}
		for i, e := range s.Recent {
			if i == 3 {
				break
			}
			line := e.Title
			if e.Label != "" {
				line += ": " + e.Label + " (" + e.Source + ")"
			}
			v.Lines = append(v.Lines, line+". "+e.Tip)
		}
		if len(v.Lines) > 0 {
			v.Lines = append(v.Lines, fmt.Sprintf("%d of %d Act 1 entries", s.Filled, s.Total))
		}
		return v
	})
}

// CodexSectionID is the id to pass to SectionRegistry.Register.
const CodexSectionID = "codex"
