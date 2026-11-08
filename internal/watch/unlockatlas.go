package watch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	atlasdata "d2lod-bot/data/atlas"
)

// The waypoint atlas (TASK-160, docs/unlock-atlas.md): the Act 1 waypoints as nodes, lit for the ones the hero has taken,
// silhouettes for the rest, and the zone the hero proposes next. The first waypoint taken opens it (ladder row "atlas",
// trigger codex_first waypoint). View only: lit comes from the Codex, the proposal from the zone proposals.

const (
	// AtlasRoute is GET (viewer); AtlasID is the ladder row.
	AtlasRoute = "/api/atlas"
	AtlasID    = "atlas"

	atlasSchema = "watch.atlas/1"
)

// AtlasNode is one row of data/atlas/waypoints.json.
type AtlasNode struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	AreaID  int    `json:"areaId"`
	CodexID string `json:"codexId"`
}

// AtlasTable is the parsed table.
type AtlasTable struct {
	Schema string      `json:"schema"`
	Doc    string      `json:"doc"`
	Note   string      `json:"note"`
	Act    int         `json:"act"`
	Nodes  []AtlasNode `json:"nodes"`
}

// ParseAtlasTable checks the table: unique ids, names, area ids and Codex ids; exactly the shape the view relies on.
// Errors name the node.
func ParseAtlasTable(b []byte) (*AtlasTable, error) {
	var t AtlasTable
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("atlas table: %w", err)
	}
	if t.Schema != "atlas.waypoints/1" {
		return nil, fmt.Errorf("atlas table: schema %q", t.Schema)
	}
	if len(t.Nodes) == 0 {
		return nil, fmt.Errorf("atlas table: no nodes")
	}
	ids, areas, codex := map[string]bool{}, map[int]bool{}, map[string]bool{}
	for _, n := range t.Nodes {
		switch {
		case n.ID == "" || n.Name == "" || n.AreaID <= 0:
			return nil, fmt.Errorf("atlas table: node %q needs an id, a name and an area id", n.ID)
		case ids[n.ID]:
			return nil, fmt.Errorf("atlas table: duplicate node %q", n.ID)
		case areas[n.AreaID]:
			return nil, fmt.Errorf("atlas table: node %q: area id %d is used twice", n.ID, n.AreaID)
		case n.CodexID != "" && codex[n.CodexID]:
			return nil, fmt.Errorf("atlas table: node %q: Codex id %q is used twice", n.ID, n.CodexID)
		}
		ids[n.ID], areas[n.AreaID], codex[n.CodexID] = true, true, true
	}
	return &t, nil
}

// AtlasSlot is one node as the viewer sees it. A node that is not lit is a silhouette: only its place in the order.
type AtlasSlot struct {
	N      int    `json:"n"` // 1 for the first waypoint in the order
	State  string `json:"state"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	AreaID int    `json:"areaId,omitempty"`
	Next   bool   `json:"next,omitempty"` // the hero proposes this zone next
}

// AtlasView is GET /api/atlas. Locked: open false, the teaser, no slots.
type AtlasView struct {
	Schema string      `json:"schema"`
	Open   bool        `json:"open"`
	Title  string      `json:"title"`
	Teaser string      `json:"teaser,omitempty"`
	Act    int         `json:"act"`
	Lit    int         `json:"lit"`
	Total  int         `json:"total"`
	Next   string      `json:"next,omitempty"` // the proposed zone, whether or not it has a waypoint
	Slots  []AtlasSlot `json:"slots"`
}

// Atlas reads the table against two sources. Safe for concurrent use as far as its sources are.
type Atlas struct {
	u     *Unlocks
	table *AtlasTable
	lit   func(codexID string) bool // nil: only the town waypoint is lit
	next  func() string             // nil: no proposal
}

// NewAtlas builds the atlas over the ladder. A ladder without the row "atlas" is an error naming it. table nil reads the
// embedded one. lit says whether a Codex entry exists; next gives the proposed zone name ("" for none).
func NewAtlas(u *Unlocks, table *AtlasTable, lit func(string) bool, next func() string) (*Atlas, error) {
	if _, err := u.Engine.Check(AtlasID); err != nil {
		return nil, err
	}
	if table == nil {
		var err error
		if table, err = ParseAtlasTable(atlasdata.JSON); err != nil {
			return nil, err
		}
	}
	return &Atlas{u: u, table: table, lit: lit, next: next}, nil
}

// View is the payload. A silhouette carries no id, name or area, even when it is the proposed zone: the proposal text
// already names it and the lit nodes are all the atlas states as fact.
func (a *Atlas) View() AtlasView {
	st, _ := a.u.state(AtlasID)
	v := AtlasView{Schema: atlasSchema, Open: st.Open, Title: st.Title, Act: a.table.Act, Total: len(a.table.Nodes), Slots: []AtlasSlot{}}
	if !st.Open {
		v.Teaser = st.Teaser
		return v
	}
	if a.next != nil {
		v.Next = a.next()
	}
	for i, n := range a.table.Nodes {
		s := AtlasSlot{N: i + 1, State: "dark"}
		if n.CodexID == "" || (a.lit != nil && a.lit(n.CodexID)) {
			s = AtlasSlot{N: i + 1, State: "lit", ID: n.ID, Name: n.Name, AreaID: n.AreaID}
			v.Lit++
		}
		s.Next = v.Next != "" && strings.EqualFold(v.Next, n.Name)
		v.Slots = append(v.Slots, s)
	}
	return v
}

// Handler serves GET /api/atlas.
func (a *Atlas) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, a.View())
	})
}
