package watch

import (
	"fmt"
	"strings"
)

const mapSchema = "watch.ordersmap/1"

// HeroInfo is what the screen knows about the hero: the bot feeds it from its status poll (GoalsOrders.OnStatus).
type HeroInfo struct {
	Class    string `json:"class,omitempty"`
	Level    int    `json:"clvl,omitempty"`
	XP       int64  `json:"-"`
	Area     string `json:"area,omitempty"`
	AreaID   int    `json:"areaId,omitempty"`
	Activity string `json:"activity,omitempty"`
	Known    bool   `json:"known"`
	Title    string `json:"title,omitempty"` // the best Hall title earned so far (hall.go)
	// Vitals for /api/hero (hero.go); not part of the orders map reply.
	Name       string `json:"-"`
	Difficulty string `json:"-"`
	XPNext     int64  `json:"-"`
	Life       int    `json:"-"`
	MaxLife    int    `json:"-"`
	Mana       int    `json:"-"`
	MaxMana    int    `json:"-"`
}

// SetHero records the hero's state. Only a change a person can see (area, level, what it does) wakes the page; xp alone
// does not.
func (h *Hub) SetHero(in HeroInfo) {
	in.Known = in.Area != "" || in.AreaID != 0
	in.Activity = safeText(in.Activity, 80)
	in.Area = safeText(in.Area, 60)
	h.mu.Lock()
	was := h.hero
	in.Title = was.Title
	h.hero = in
	if in.Area != was.Area || in.AreaID != was.AreaID || in.Level != was.Level || in.Activity != was.Activity || in.Known != was.Known {
		h.notifyLocked()
	}
	fire := h.onClass
	if in.Class == "" || in.Class == was.Class {
		fire = nil
	}
	h.mu.Unlock()
	if fire != nil {
		fire() // outside the lock: the hook reads the hero again
	}
}

// OnClass registers a function run when the hero's class becomes known or changes (the policy sheet settles the
// class's starting options there).
func (h *Hub) OnClass(f func()) {
	h.mu.Lock()
	h.onClass = f
	h.mu.Unlock()
}

// MarkCleared records game areas the bot knows are cleared (a quest milestone). Nothing marks an area cleared by itself:
// "cleared" is shown only where the bot said so (design section 3).
func (h *Hub) MarkCleared(areaIDs ...int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cleared == nil {
		h.cleared = map[int]bool{}
	}
	for _, id := range areaIDs {
		h.cleared[id] = true
	}
	h.notifyLocked()
}

// MapStatus is the per-node state of the hero and the orders.
type MapStatus struct {
	Here    bool   `json:"here"`
	Order   bool   `json:"order"`            // the current order points here
	Queued  int    `json:"queued,omitempty"` // 1-based place in the queue
	Hard    bool   `json:"hard"`             // the hero's own guard closes it now
	HardWhy string `json:"hardWhy,omitempty"`
	Cleared bool   `json:"cleared"`
}

// MapNodeView is one node of GET /api/orders/map.
type MapNodeView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	X         int       `json:"x"`
	Y         int       `json:"y"`
	Waypoint  bool      `json:"waypoint,omitempty"`
	Sub       string    `json:"sub,omitempty"`
	Orderable bool      `json:"orderable"`
	Order     string    `json:"order,omitempty"` // the area name an order carries; "" for the town node (rest in town)
	Range     string    `json:"range,omitempty"` // veteran advice, never a guard
	Why       string    `json:"why,omitempty"`
	Advice    string    `json:"advice,omitempty"` // set when the hero is below the advised level
	Status    MapStatus `json:"status"`
}

// MapEdgeView joins two node ids.
type MapEdgeView struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Dungeon bool   `json:"dungeon,omitempty"`
}

// MapActView lists the acts the screen can draw; Populated is false for acts without data.
type MapActView struct {
	Act       int  `json:"act"`
	Populated bool `json:"populated"`
}

// MapBox is the SVG view box of the act.
type MapBox struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// MapView is GET /api/orders/map.
type MapView struct {
	Schema    string        `json:"schema"`
	Seq       uint64        `json:"seq"`
	Act       int           `json:"act"`
	Acts      []MapActView  `json:"acts"`
	Populated bool          `json:"populated"`
	Town      string        `json:"town,omitempty"`
	Box       MapBox        `json:"box"`
	Hero      HeroInfo      `json:"hero"`
	Nodes     []MapNodeView `json:"nodes"`
	Edges     []MapEdgeView `json:"edges"`
	OpenOrder int           `json:"openOrders"`
	MaxOpen   int           `json:"maxOpenOrders"`
}

const nodeW, nodeH = 104, 44

// OrdersMap builds GET /api/orders/map for an act (1 to 5; anything else is Act 1).
func (h *Hub) OrdersMap(act int) MapView {
	am, ok := Maps[act]
	if !ok {
		am = Maps[1]
	}
	// The guard callback reads the bot: call it before taking the Hub's lock (lock order, hub.go).
	guard := map[string]string{}
	if h.cfg.AreaGuard != nil {
		for _, n := range am.Nodes {
			for _, id := range n.IDs {
				if why := safeText(h.cfg.AreaGuard(id), 160); why != "" {
					guard[n.ID] = why
					break
				}
			}
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	v := MapView{Schema: mapSchema, Seq: h.seq, Act: am.Act, Populated: len(am.Nodes) > 0, Town: am.Town, Hero: h.hero,
		Nodes: []MapNodeView{}, Edges: []MapEdgeView{}, MaxOpen: MaxOpenOrders}
	for a := 1; a <= 5; a++ {
		v.Acts = append(v.Acts, MapActView{Act: a, Populated: len(Maps[a].Nodes) > 0})
	}
	var cur *Order
	var queued []*Order
	for _, o := range h.orders {
		switch o.State {
		case OrderActive:
			cur = o
			v.OpenOrder++
		case OrderQueued:
			queued = append(queued, o)
			v.OpenOrder++
		}
	}
	minX, minY, maxX, maxY := 1<<30, 1<<30, 0, 0
	for _, n := range am.Nodes {
		minX, minY = min(minX, n.X-nodeW/2), min(minY, n.Y-nodeH/2)
		maxX, maxY = max(maxX, n.X+nodeW/2), max(maxY, n.Y+nodeH/2)
		nv := MapNodeView{ID: n.ID, Name: n.Name, Kind: n.Kind, X: n.X, Y: n.Y, Waypoint: n.Waypoint, Sub: n.Sub, Why: n.Why,
			Order: n.Order, Orderable: n.Kind == "town" || (n.Order != "" && containsFold(h.cfg.Areas, n.Order))}
		if n.Lo > 0 {
			nv.Range = rangeText(n.Lo, n.Hi)
			if h.hero.Known && h.hero.Level > 0 && h.hero.Level < n.Lo {
				nv.Advice = fmt.Sprintf("Advice, not a guard: veterans go here from clvl %d.", n.Lo)
			}
		}
		st := &nv.Status
		st.Here = h.hero.Known && nodeHas(n, h.hero.AreaID, h.hero.Area)
		st.Order = cur != nil && orderAt(*cur, n, h.hero)
		for i, q := range queued {
			if orderAt(*q, n, h.hero) {
				st.Queued = i + 1
				break
			}
		}
		if why := guard[n.ID]; why != "" {
			st.Hard, st.HardWhy = true, why
		}
		for _, id := range n.IDs {
			st.Cleared = st.Cleared || h.cleared[id]
		}
		v.Nodes = append(v.Nodes, nv)
	}
	if len(am.Nodes) > 0 {
		v.Box = MapBox{X: minX - 8, Y: minY - 8, W: maxX - minX + 16, H: maxY - minY + 16}
	}
	for _, e := range am.Edges {
		v.Edges = append(v.Edges, MapEdgeView{From: e.From, To: e.To, Dungeon: e.Dungeon})
	}
	return v
}

func rangeText(lo, hi int) string {
	if hi == 0 {
		return fmt.Sprintf("clvl %d+", lo)
	}
	return fmt.Sprintf("clvl %d-%d", lo, hi)
}

func containsFold(list []string, s string) bool {
	for _, x := range list {
		if strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}

// nodeHas says whether the hero's area (id first, then name) belongs to node n.
func nodeHas(n MapNode, id int, name string) bool {
	for _, x := range n.IDs {
		if id != 0 && x == id {
			return true
		}
	}
	return id == 0 && name != "" && (strings.EqualFold(n.Name, name) || strings.EqualFold(n.Order, name))
}

// orderAt says whether order o points at node n; "fight here" points at the hero's own node, "rest in town" at the town.
func orderAt(o Order, n MapNode, hero HeroInfo) bool {
	switch o.Kind {
	case OrderGo:
		return n.Order != "" && strings.EqualFold(n.Order, o.Area)
	case OrderHere:
		return hero.Known && nodeHas(n, hero.AreaID, hero.Area)
	default:
		return n.Kind == "town"
	}
}

// setHeroTitle records the hero's title for the watch header; a change wakes the page.
func (h *Hub) setHeroTitle(t string) {
	t = safeText(t, 60)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hero.Title != t {
		h.hero.Title = t
		h.notifyLocked()
	}
}
