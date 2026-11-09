package watch

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LayoutManifest is a watch.layout/1 file (docs/ui-design-system.md section 6.4): the grid positions the layout editor
// (docs/mockups/ui/editor.html) exports. The shell does not read it yet (migration PR 14); the type exists so the
// compatibility test can prove that files exported today keep loading.
type LayoutManifest struct {
	Schema      string                      `json:"schema"`
	Cols        int                         `json:"cols"`
	RowHeight   int                         `json:"rowHeight"`
	Density     string                      `json:"density,omitempty"`
	Theme       string                      `json:"theme,omitempty"`
	Breakpoints map[string]LayoutBreakpoint `json:"breakpoints"`
}

// LayoutBreakpoint is one breakpoint of a layout manifest.
type LayoutBreakpoint struct {
	Width  int           `json:"width"`
	Cols   int           `json:"cols"`
	Panels []LayoutPanel `json:"panels"`
}

// LayoutPanel is one placed panel.
type LayoutPanel struct {
	ID        string `json:"id"`
	Band      string `json:"band"`
	Col       int    `json:"col"`
	Row       int    `json:"row"`
	ColSpan   int    `json:"colSpan"`
	RowSpan   int    `json:"rowSpan"`
	Collapsed bool   `json:"collapsed"`
	Order     int    `json:"order"`
}

// ParseLayout reads a layout manifest. Unknown keys are ignored (compatibility promise, section 0); a schema of a
// later major version and a panel outside the grid are errors.
func ParseLayout(b []byte) (LayoutManifest, error) {
	var l LayoutManifest
	if err := json.Unmarshal(b, &l); err != nil {
		return l, err
	}
	if l.Schema != "watch.layout/1" {
		return l, fmt.Errorf("layout schema %q (this version reads watch.layout/1)", l.Schema)
	}
	for name, bp := range l.Breakpoints {
		cols := bp.Cols
		if cols == 0 {
			cols = l.Cols
		}
		for _, p := range bp.Panels {
			if p.ID == "" || strings.ContainsAny(p.ID, " /") || p.Col < 1 || p.ColSpan < 1 || p.Col+p.ColSpan-1 > cols || p.Row < 1 {
				return l, fmt.Errorf("layout %s: panel %q at col %d span %d row %d does not fit %d columns", name, p.ID, p.Col, p.ColSpan, p.Row, cols)
			}
		}
	}
	return l, nil
}
