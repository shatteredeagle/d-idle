package d2dctl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Town verbs: the channel pieces the town layer needs on top of the raw list verbs. Vendor stock comes from the
// private fork's "dump vendor" section (stock d2d has no verb for it: V 2026-10-30, the store window is invisible to
// "items", "state" only says store=<vendor> store_tab=<n>); every other call here uses stock verbs.

// VendorItem is one entry of the open store window. d2d gives stock items no unit id (V: "id":-1), so an item is
// addressed by Tab and its grid cell.
type VendorItem struct {
	Tab, X, Y, W, H int
	Code            string
	Quality, ILvl   int
	Sockets, Qty    int
	Price           int // buy price in gold at this NPC for this character (V: Wand 205, Tome of Town Portal 250 on Akara)
}

// Vendor is the open store window ("dump vendor").
type Vendor struct {
	Open   bool
	Vendor int // vendor row (V: 0 for Akara); -1 when closed
	NPC    int // "npcs" index of the vendor
	Tab    int // the tab showing now (0 armor, 1 weapons, 2 weapons, 3 misc; V: d2d always draws all four tabs)
	Gamble bool
	Items  []VendorItem
}

var ErrNoVendorDump = errors.New("d2dctl: dump vendor not available (needs the private fork patched with the vendor section)")

// Vendor reads the store window. A stock d2d answers ErrNoVendorDump.
func (c *Client) Vendor(ctx context.Context) (Vendor, error) {
	r, err := c.Do(ctx, "dump vendor")
	if err != nil {
		var ce *CmdError
		if errors.As(err, &ce) && strings.Contains(ce.Msg, "unknown") {
			return Vendor{}, ErrNoVendorDump
		}
		return Vendor{}, err
	}
	line := strings.TrimSpace(r.Rest)
	for _, l := range r.Lines {
		if strings.TrimSpace(l) != "" {
			line = l
			break
		}
	}
	return ParseVendor([]byte(line))
}

// ParseVendor decodes the JSON line of "dump vendor".
func ParseVendor(b []byte) (Vendor, error) {
	var raw struct {
		Vendor struct {
			Open   bool `json:"open"`
			Vendor int  `json:"vendor"`
			NPC    int  `json:"npc"`
			Tab    int  `json:"tab"`
			Gamble bool `json:"gamble"`
			Tabs   [][]struct {
				Code    string `json:"code"`
				X       int    `json:"x"`
				Y       int    `json:"y"`
				W       int    `json:"w"`
				H       int    `json:"h"`
				Quality int    `json:"quality"`
				ILvl    int    `json:"ilvl"`
				Sockets int    `json:"sockets"`
				Qty     int    `json:"qty"`
				Price   int    `json:"price"`
			} `json:"tabs"`
		} `json:"vendor"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return Vendor{}, fmt.Errorf("d2dctl: dump vendor: %w", err)
	}
	v := raw.Vendor
	out := Vendor{Open: v.Open, Vendor: v.Vendor, NPC: v.NPC, Tab: v.Tab, Gamble: v.Gamble}
	for t, tab := range v.Tabs {
		for _, it := range tab {
			out.Items = append(out.Items, VendorItem{Tab: t, X: it.X, Y: it.Y, W: it.W, H: it.H, Code: it.Code,
				Quality: it.Quality, ILvl: it.ILvl, Sockets: it.Sockets, Qty: it.Qty, Price: it.Price})
		}
	}
	return out, nil
}

// UIFlags are the panel flags of "dump ui" (V).
type UIFlags struct {
	Stash, Cube, NPCMenu, Vendor, SkillTree, Waypoint, GameMenu, Mini, Automap bool
	Held                                                                       string // code of the item on the cursor, "" none
	Inventory                                                                  bool   // backpack panel open (lab3+ ui.rects.inv_grid)
}

// UI reads the panel flags.
func (c *Client) UI(ctx context.Context) (UIFlags, error) {
	r, err := c.Do(ctx, "dump ui")
	if err != nil {
		return UIFlags{}, err
	}
	line := strings.TrimSpace(r.Rest)
	for _, l := range r.Lines {
		if strings.TrimSpace(l) != "" {
			line = l
			break
		}
	}
	var raw struct {
		UI struct {
			Stash     bool                       `json:"stash"`
			Cube      bool                       `json:"cube"`
			NPCMenu   bool                       `json:"npc_menu"`
			Vendor    bool                       `json:"vendor"`
			SkillTree bool                       `json:"skill_tree"`
			Waypoint  bool                       `json:"waypoint"`
			GameMenu  bool                       `json:"game_menu"`
			Mini      bool                       `json:"mini_panel"`
			Automap   bool                       `json:"automap"`
			Held      string                     `json:"held"`
			Rects     map[string]json.RawMessage `json:"rects"`
		} `json:"ui"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return UIFlags{}, fmt.Errorf("d2dctl: dump ui: %w", err)
	}
	u := raw.UI
	_, inv := u.Rects["inv_grid"]
	return UIFlags{u.Stash, u.Cube, u.NPCMenu, u.Vendor, u.SkillTree, u.Waypoint, u.GameMenu, u.Mini, u.Automap, u.Held, inv}, nil
}

// ItemAt returns the screen point of carried item id's box (the inventory panel must be open, V: "debug itemat").
func (c *Client) ItemAt(ctx context.Context, id int) (x, y int, err error) {
	r, err := c.Debug(ctx, "itemat", fmt.Sprint(id))
	if err != nil {
		return 0, 0, err
	}
	if _, err := fmt.Sscanf(r.Rest, "%d %d", &x, &y); err != nil {
		return 0, 0, fmt.Errorf("d2dctl: itemat reply %q", r.Rest)
	}
	return x, y, nil
}

// Pause is the delay HoverClick and friends use between the steps; tests set it to 0.
var Pause = func(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// HoverClick moves the pointer to (x,y) from 3 px away, waits and clicks. V (2026-10-30, Akara's menu): a click
// without a preceding hover is ignored, the same as on Game.exe; two moves (the first 3 px off) were enough.
func (c *Client) HoverClick(ctx context.Context, x, y int, right bool) error {
	if err := c.Hover(ctx, x-3, y); err != nil {
		return err
	}
	if err := Pause(ctx, 150*time.Millisecond); err != nil {
		return err
	}
	if err := c.Hover(ctx, x, y); err != nil {
		return err
	}
	if err := Pause(ctx, 450*time.Millisecond); err != nil {
		return err
	}
	if right {
		return c.RClick(ctx, x, y)
	}
	return c.Click(ctx, x, y)
}

// OpenNPCMenu walks to the NPC (ApproachNPC) and returns its dialog lines.
func (c *Client) OpenNPCMenu(ctx context.Context, index int, timeout time.Duration) ([]MenuLine, error) {
	if err := c.ApproachNPC(ctx, index, timeout); err != nil {
		return nil, err
	}
	m, err := c.Menu(ctx)
	if err != nil {
		return nil, err
	}
	if len(m.Items) == 0 {
		return nil, fmt.Errorf("d2dctl: npc %d: opened a panel, not a dialog", index)
	}
	return m.Items, nil
}

// Screen position of the player's feet (V: the camera follows the player; two screenshots, 2026-10-30) and the
// cell-to-screen scale of "debug presets"/"debug portals"/"npcs" cells (V: Waypoint npc row gave sx 1104 sy 20 for
// a cell offset of (+1.0,-7.8), 80 px per cell of (dx-dy), 40 px per cell of (dx+dy)).
const feetX, feetY = 400, 292

// ScreenOfCell projects cell (cx,cy) seen from a player standing on (px,py); lift raises the point for a tall
// sprite (a town portal is about 50 px high).
func ScreenOfCell(px, py, cx, cy float64, lift int) (int, int) {
	dx, dy := cx-px, cy-py
	return feetX + int(math.Round(80*(dx-dy))), feetY + int(math.Round(40*(dx+dy))) - lift
}

// opened reports whether an interaction took effect: NPC dialog lines or a stash, vendor or waypoint panel.
func (c *Client) opened(ctx context.Context) (bool, error) {
	m, err := c.Menu(ctx)
	if err != nil {
		return false, err
	}
	if len(m.Items) > 0 {
		return true, nil
	}
	ui, err := c.UI(ctx)
	if err != nil { // no dump ui (stock d2d): the dialog lines are the only signal
		return false, nil
	}
	return ui.Stash || ui.Vendor || ui.Waypoint, nil
}

// ApproachNPC makes "cmd interact <index>" work from anywhere in town. V (2026-10-30, Akara's corner, Charsi at
// 22.9,11.5): one interact walks until the straight-line walk meets an object (camp fires, the waypoint, stalls) and
// then stands still for good; repeating it changes nothing (stuck at 27.7,12.7 for 5+ s), but a short side step with
// "cmd move" and a new interact reaches the NPC (Charsi opened after a step to 29.4,11.1). So: interact, wait, and
// when the player did not move, side-step toward the free neighbour cell nearest to the NPC and interact again.
func (c *Client) ApproachNPC(ctx context.Context, index int, timeout time.Duration) error {
	l, err := c.NPCs(ctx)
	if err != nil {
		return err
	}
	var npc *NPC
	for i := range l.Items {
		if l.Items[i].Index == index {
			npc = &l.Items[i]
		}
	}
	deadline := time.Now().Add(timeout)
	prev, perr := c.Level(ctx) // no "debug level" (stock d2d?): walk by repeating the verb only
	if npc == nil {            // not listed (yet): still try the verb, side steps then have no direction
		npc = &NPC{Index: index, Name: "?", CellX: prev.X, CellY: prev.Y}
	}
	for stalls := 0; ; {
		if err := c.CmdInteract(ctx, index); err != nil {
			return err
		}
		for i := 0; i < 10; i++ {
			if err := Pause(ctx, 300*time.Millisecond); err != nil {
				return err
			}
			if ok, err := c.opened(ctx); err != nil || ok {
				return err
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("d2dctl: npc %d (%s): not reached in %v (player at %.1f,%.1f)", index, npc.Name, timeout, prev.X, prev.Y)
		}
		cur, err := c.Level(ctx)
		if err != nil || perr != nil {
			perr, prev = err, cur
			continue
		}
		if math.Hypot(cur.X-prev.X, cur.Y-prev.Y) < 0.3 {
			stalls++
			if _, err := c.sidestep(ctx, cur, npc.CellX, npc.CellY, stalls); err != nil {
				return err
			}
		}
		prev = cur
	}
}

// sidestep moves to a free cell 3.5 away from from (the 8 compass points, nearest to the target first, rotated by
// try so a repeated stall picks another) and reports whether the player moved.
func (c *Client) sidestep(ctx context.Context, from LevelInfo, tx, ty float64, try int) (bool, error) {
	type cand struct{ x, y, d float64 }
	var cs []cand
	for k := 0; k < 8; k++ {
		a := float64(k) * math.Pi / 4
		x, y := from.X+3.5*math.Cos(a), from.Y+3.5*math.Sin(a)
		cs = append(cs, cand{x, y, math.Hypot(x-tx, y-ty)})
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].d < cs[j].d })
	for k := 0; k < 3; k++ {
		cd := cs[(try-1+k)%len(cs)]
		if err := c.CmdMove(ctx, cd.x, cd.y); err != nil {
			return false, err
		}
		if err := Pause(ctx, 1500*time.Millisecond); err != nil {
			return false, err
		}
		cur, err := c.Level(ctx)
		if err != nil {
			return false, err
		}
		if math.Hypot(cur.X-from.X, cur.Y-from.Y) > 1 {
			return true, nil
		}
	}
	return false, nil
}

// FindLine returns the first menu line whose text contains want (case-insensitive). The header (first line, the
// NPC name) is skipped.
func FindLine(lines []MenuLine, want string) (MenuLine, bool) {
	want = strings.ToLower(want)
	for i, l := range lines {
		if i == 0 {
			continue
		}
		if strings.Contains(strings.ToLower(l.Text), want) {
			return l, true
		}
	}
	return MenuLine{}, false
}

// PickMenuLine hover-clicks the dialog line containing text.
func (c *Client) PickMenuLine(ctx context.Context, text string) error {
	m, err := c.Menu(ctx)
	if err != nil {
		return err
	}
	l, ok := FindLine(m.Items, text)
	if !ok {
		return fmt.Errorf("d2dctl: no menu line %q", text)
	}
	return c.HoverClick(ctx, l.X, l.Y, false)
}

// Vendor panel geometry at 800x600 (V on d2d 2026-10-30: screenshot of Akara's window, buys landed on the cells
// computed from these; same art as Game.exe, see internal/town/panel for the game-side numbers).
const (
	VendorX0, VendorY0, VendorCell = 97, 125, 29
	TabY, TabX0, TabPitch          = 75, 118, 81
)

// VendorCellPoint is the screen point inside grid cell (x,y) of the vendor panel, for an item of w x h cells.
func VendorCellPoint(x, y, w, h int) (int, int) {
	return VendorX0 + x*VendorCell + w*VendorCell/2, VendorY0 + y*VendorCell + h*VendorCell/2
}

// SelectTab clicks vendor tab t (0..3) unless it is showing already (state store_tab is 1-based, V).
func (c *Client) SelectTab(ctx context.Context, t int) error {
	st, _, err := c.State(ctx)
	if err != nil {
		return err
	}
	if n, err := atoi(st.M["store_tab"]); err == nil && n == t+1 {
		return nil
	}
	if err := c.HoverClick(ctx, TabX0+t*TabPitch, TabY, false); err != nil {
		return err
	}
	return Pause(ctx, 600*time.Millisecond)
}

// BuyItem selects the item's tab and right-clicks it: the game buys and places it in the pack (V: Wand -205 gold,
// Tome of Town Portal -250). Returns the gold paid.
func (c *Client) BuyItem(ctx context.Context, it VendorItem) (int, error) {
	g0, err := c.gold(ctx)
	if err != nil {
		return 0, err
	}
	if err := c.SelectTab(ctx, it.Tab); err != nil {
		return 0, err
	}
	x, y := VendorCellPoint(it.X, it.Y, it.W, it.H)
	if err := c.HoverClick(ctx, x, y, true); err != nil {
		return 0, err
	}
	if err := Pause(ctx, 500*time.Millisecond); err != nil {
		return 0, err
	}
	g1, err := c.gold(ctx)
	if err != nil {
		return 0, err
	}
	if g1 >= g0 {
		return 0, fmt.Errorf("d2dctl: buying %s: gold did not change (%d -> %d): no room, no money or the click missed", it.Code, g0, g1)
	}
	return g0 - g1, nil
}

// SellItem sells carried item id (inventory panel open and a vendor open): picks it up and drops it on the vendor
// grid. V that this sells; V that a right-click on the pack item does NOT sell on d2d. Returns the gold gained
// (the purse is capped at 10000 per clvl, so a cheat purse can make the gain look small).
func (c *Client) SellItem(ctx context.Context, id int) (int, error) {
	g0, err := c.gold(ctx)
	if err != nil {
		return 0, err
	}
	x, y, err := c.ItemAt(ctx, id)
	if err != nil {
		return 0, err
	}
	if err := c.HoverClick(ctx, x, y, false); err != nil {
		return 0, err
	}
	if err := Pause(ctx, 400*time.Millisecond); err != nil {
		return 0, err
	}
	if err := c.HoverClick(ctx, VendorX0+4*VendorCell, VendorY0+6*VendorCell, false); err != nil {
		return 0, err
	}
	if err := Pause(ctx, 600*time.Millisecond); err != nil {
		return 0, err
	}
	g1, err := c.gold(ctx)
	if err != nil {
		return 0, err
	}
	if g1 <= g0 {
		return 0, fmt.Errorf("d2dctl: selling item %d: gold did not rise (%d -> %d)", id, g0, g1)
	}
	return g1 - g0, nil
}

func (c *Client) gold(ctx context.Context) (int, error) {
	st, _, err := c.State(ctx)
	return st.Gold, err
}

// ---- stash, repair, portal, waypoint (V on the private fork, 2026-10-30, Sorceress/Necromancer "Townt") ----

// Panel geometry at 800x600 (V: screenshots of the open stash/vendor windows).
const (
	InvGoldX, InvGoldY     = 493, 461 // coin left of the pack gold: opens "How much gold would you like to deposit?"
	StashGoldX, StashGoldY = 165, 92  // coin of the stash gold bar: opens the withdraw prompt
	RepairAllX, RepairAllY = 368, 461 // right-most of the four buttons under a smith's grid (anvil/hammer)
	WaypointRowX           = 210      // act 1 waypoint list: x of the names, rows 0..8 from y 138, 35 px apart (U: choose)
	WaypointRow0Y, WPPitch = 138, 35
)

// Gold reads the purse and the stash gold ("dump player").
func (c *Client) Gold(ctx context.Context) (purse, stash int, err error) {
	r, err := c.Do(ctx, "dump player")
	if err != nil {
		return 0, 0, err
	}
	line := strings.TrimSpace(r.Rest)
	for _, l := range r.Lines {
		if strings.TrimSpace(l) != "" {
			line = l
			break
		}
	}
	var raw struct {
		Player struct {
			Gold      int `json:"gold"`
			StashGold int `json:"stash_gold"`
		} `json:"player"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return 0, 0, fmt.Errorf("d2dctl: dump player: %w", err)
	}
	return raw.Player.Gold, raw.Player.StashGold, nil
}

// OpenStash walks to the stash chest ("npcs" row "Your Private Stash") and waits for the panel.
func (c *Client) OpenStash(ctx context.Context, timeout time.Duration) error {
	l, err := c.NPCs(ctx)
	if err != nil {
		return err
	}
	for _, n := range l.Items {
		if strings.EqualFold(n.Name, "Your Private Stash") {
			return c.ApproachNPC(ctx, n.Index, timeout)
		}
	}
	return errors.New("d2dctl: no stash chest in npcs (not in a town?)")
}

// MoveGold moves n gold through the amount prompt: coin click, the digits typed, Return. V: deposit 5000 took
// 20000/0 to 15000/5000, withdraw 2000 gave 17000/3000. The stash panel must be open. Verified by dump player.
func (c *Client) MoveGold(ctx context.Context, n int, deposit bool) error {
	p0, s0, err := c.Gold(ctx)
	if err != nil {
		return err
	}
	x, y := StashGoldX, StashGoldY
	if deposit {
		x, y = InvGoldX, InvGoldY
	}
	if err := c.HoverClick(ctx, x, y, false); err != nil {
		return err
	}
	if err := Pause(ctx, 400*time.Millisecond); err != nil {
		return err
	}
	if err := c.Text(ctx, strconv.Itoa(n)); err != nil {
		return err
	}
	if err := c.Key(ctx, "Return"); err != nil {
		return err
	}
	if err := Pause(ctx, 600*time.Millisecond); err != nil {
		return err
	}
	p1, s1, err := c.Gold(ctx)
	if err != nil {
		return err
	}
	if deposit && s1 <= s0 || !deposit && p1 <= p0 {
		return fmt.Errorf("d2dctl: gold move of %d did not happen (purse %d->%d, stash %d->%d)", n, p0, p1, s0, s1)
	}
	return nil
}

// Durability sums current and maximum durability of the worn items ("dump items", loc body).
func (c *Client) Durability(ctx context.Context) (cur, max int, err error) {
	r, err := c.Do(ctx, "dump items")
	if err != nil {
		return 0, 0, err
	}
	line := strings.TrimSpace(r.Rest)
	for _, l := range r.Lines {
		if strings.TrimSpace(l) != "" {
			line = l
			break
		}
	}
	var raw struct {
		Items []struct {
			Loc string `json:"loc"`
			Dur [2]int `json:"dur"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return 0, 0, fmt.Errorf("d2dctl: dump items: %w", err)
	}
	for _, it := range raw.Items {
		if it.Loc == "body" && it.Dur[1] > 0 {
			cur += it.Dur[0]
			max += it.Dur[1]
		}
	}
	return cur, max, nil
}

// RepairAll presses the smith window's repair-all button and confirms by the worn durability. The window must be
// open. V: Charsi, wand 7/15 to 15/15 for 13 gold. Nothing to repair (already full) is a success.
func (c *Client) RepairAll(ctx context.Context) error {
	c0, m0, err := c.Durability(ctx)
	if err != nil {
		return err
	}
	if c0 >= m0 {
		return nil
	}
	if err := c.HoverClick(ctx, RepairAllX, RepairAllY, false); err != nil {
		return err
	}
	if err := Pause(ctx, 700*time.Millisecond); err != nil {
		return err
	}
	c1, _, err := c.Durability(ctx)
	if err != nil {
		return err
	}
	if c1 <= c0 {
		return fmt.Errorf("d2dctl: repair not confirmed (worn durability %d/%d unchanged)", c0, m0)
	}
	return nil
}

// UseItem right-clicks carried item id in the open pack (V: a Town Portal scroll opened the portal and was
// consumed). The caller opens the pack ("key i") first; "debug itemat" gives the cell. A Tome of Town Portal is
// the same right-click (U, not run).
func (c *Client) UseItem(ctx context.Context, id int) error {
	x, y, err := c.ItemAt(ctx, id)
	if err != nil {
		return err
	}
	if err := c.Hover(ctx, x-3, y); err != nil {
		return err
	}
	if err := Pause(ctx, 150*time.Millisecond); err != nil {
		return err
	}
	if err := c.Hover(ctx, x, y); err != nil {
		return err
	}
	if err := Pause(ctx, 450*time.Millisecond); err != nil {
		return err
	}
	return c.RClick(ctx, x, y)
}

// CastTP opens a town portal with scroll or tome id: pack open, right-click, pack closed, then waits for
// "debug portals" to list one.
func (c *Client) CastTP(ctx context.Context, id int) error {
	n0, err := c.Portals(ctx)
	if err != nil {
		return err
	}
	if err := c.Key(ctx, "i"); err != nil {
		return err
	}
	if err := Pause(ctx, 700*time.Millisecond); err != nil {
		return err
	}
	uerr := c.UseItem(ctx, id)
	if err := c.Key(ctx, "i"); err != nil && uerr == nil {
		uerr = err
	}
	if uerr != nil {
		return uerr
	}
	for i := 0; i < 10; i++ {
		if err := Pause(ctx, 300*time.Millisecond); err != nil {
			return err
		}
		if p, err := c.Portals(ctx); err != nil {
			return err
		} else if len(p.Items) > len(n0.Items) {
			return nil
		}
	}
	return errors.New("d2dctl: town portal did not open")
}

// EnterPortal walks to the portal that stands in the current level and clicks it. V: a portal is not an npc or a
// unit, walking onto it does nothing, a left click on its sprite takes it (Blood Moor to the camp, then back is
// the same click in the camp). Verified by the level id changing.
func (c *Client) EnterPortal(ctx context.Context) error {
	here, err := c.Level(ctx)
	if err != nil {
		return err
	}
	ps, err := c.Portals(ctx)
	if err != nil {
		return err
	}
	var p *Portal
	for i := range ps.Items {
		if ps.Items[i].Level == here.ID {
			p = &ps.Items[i]
		}
	}
	if p == nil {
		return fmt.Errorf("d2dctl: no portal in level %d", here.ID)
	}
	for try := 0; try < 3; try++ {
		cur, err := c.Level(ctx)
		if err != nil {
			return err
		}
		if math.Hypot(cur.X-p.X, cur.Y-p.Y) > 1.5 {
			if err := c.CmdMove(ctx, p.X, p.Y); err != nil {
				return err
			}
			if err := Pause(ctx, 2*time.Second); err != nil {
				return err
			}
			if cur, err = c.Level(ctx); err != nil {
				return err
			}
		}
		x, y := ScreenOfCell(cur.X, cur.Y, p.X, p.Y, 50)
		if err := c.HoverClick(ctx, x, y, false); err != nil {
			return err
		}
		for i := 0; i < 10; i++ {
			if err := Pause(ctx, 400*time.Millisecond); err != nil {
				return err
			}
			if l, err := c.Level(ctx); err != nil {
				return err
			} else if l.ID != here.ID {
				return nil
			}
		}
	}
	return fmt.Errorf("d2dctl: portal click did not change the level (still %d)", here.ID)
}

// OpenWaypoint walks to the town waypoint ("npcs" row "Waypoint") until its panel is up. V: panel flag true,
// the list shows Rogue Encampment plus the activated waypoints.
func (c *Client) OpenWaypoint(ctx context.Context, timeout time.Duration) error {
	l, err := c.NPCs(ctx)
	if err != nil {
		return err
	}
	for _, n := range l.Items {
		if strings.EqualFold(n.Name, "Waypoint") {
			return c.ApproachNPC(ctx, n.Index, timeout)
		}
	}
	return errors.New("d2dctl: no waypoint in npcs")
}

// ChooseWaypoint clicks row (0 = the first list entry) of the open act 1 waypoint panel and waits for the level id
// to change. U: only the open panel and its row geometry were seen (no second waypoint was active).
func (c *Client) ChooseWaypoint(ctx context.Context, row int) error {
	here, err := c.Level(ctx)
	if err != nil {
		return err
	}
	if err := c.HoverClick(ctx, WaypointRowX, WaypointRow0Y+row*WPPitch, false); err != nil {
		return err
	}
	for i := 0; i < 20; i++ {
		if err := Pause(ctx, 400*time.Millisecond); err != nil {
			return err
		}
		if l, err := c.Level(ctx); err == nil && l.ID != here.ID {
			return nil
		}
	}
	return fmt.Errorf("d2dctl: waypoint row %d did not change the level (still %d)", row, here.ID)
}
