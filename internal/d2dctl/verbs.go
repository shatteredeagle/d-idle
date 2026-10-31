package d2dctl

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// List is a parsed multi-line reply. Reply keeps the raw lines; unparsable lines are in Reply.Bad.
type List[T any] struct {
	Items []T
	*Reply
}

func parseList[T any](r *Reply, err error, parse func(string) (T, error)) (List[T], error) {
	l := List[T]{Reply: r}
	if err != nil {
		return l, err
	}
	for _, line := range r.Lines {
		v, perr := parse(line)
		if perr != nil {
			r.Bad = append(r.Bad, line)
			continue
		}
		l.Items = append(l.Items, v)
	}
	return l, nil
}

func ff(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// Kick coordinates: the player stands at the screen centre (the camera follows it), so a click there walks
// nowhere. The spike's first choice (400,500) lies below the player and walked it south-east whenever the quirk was
// already cleared (V live, d2dwalk: it pushed the character back over a level border). V that a click near the player
// does not move it (400,285 and 400,300: no displacement; 400,270: 0.2 cell).
const (
	KickX = 400
	KickY = 300
)

// Kick sends the priming click that un-sticks "cmd move" and "cmd interact".
//
// Quirk (V, spike section 4): right after "debug goto", a level change or loading a save, those two cmd
// verbs are silently ignored (no error, the player does not move) until one real "click" was sent. Cause
// unknown (U). The typed Goto, Warp, Enter and Difficulty methods mark the client as needing a kick and
// CmdMove/CmdInteract send it first unless Options.NoAutoKick; after a save load or any other level change
// call Kick yourself.
func (c *Client) Kick(ctx context.Context) error {
	_, err := c.Do(ctx, fmt.Sprintf("click %d %d", KickX, KickY))
	if err == nil {
		c.kickPending.Store(false)
	}
	return err
}

// NeedsKick reports whether a warp-like verb ran since the last Kick.
func (c *Client) NeedsKick() bool { return c.kickPending.Load() }

func (c *Client) autoKick(ctx context.Context) error {
	if c.opts.NoAutoKick || !c.kickPending.Load() {
		return nil
	}
	return c.Kick(ctx)
}

func (c *Client) ok(ctx context.Context, format string, a ...any) error {
	_, err := c.Do(ctx, fmt.Sprintf(format, a...))
	return err
}

// Ping checks liveness.
func (c *Client) Ping(ctx context.Context) error { return c.ok(ctx, "ping") }

// Info returns the framebuffer size and frame counter.
func (c *Client) Info(ctx context.Context) (Info, error) {
	r, err := c.Do(ctx, "info")
	if err != nil {
		return Info{}, err
	}
	return parseInfo(strings.Join(append(r.Lines, r.Rest), " ")), nil
}

// State returns the loop state line.
func (c *Client) State(ctx context.Context) (State, *Reply, error) {
	r, err := c.Do(ctx, "state")
	if err != nil {
		return State{}, r, err
	}
	return parseState(strings.Join(append(r.Lines, r.Rest), " ")), r, nil
}

// Screenshot writes a PNG at an absolute path on the d2d host and returns its size. Relative paths land in
// d2d's user dir and "~" is not expanded (V).
func (c *Client) Screenshot(ctx context.Context, path string) (int, error) {
	r, err := c.Do(ctx, "screenshot "+path)
	if err != nil {
		return 0, err
	}
	n, _ := atoi(r.Rest)
	return n, nil
}

// Quit asks d2d to exit. The connection is closed by the peer afterwards.
func (c *Client) Quit(ctx context.Context) error { return c.ok(ctx, "quit") }

// Save saves the character now.
func (c *Client) Save(ctx context.Context) error { return c.ok(ctx, "save") }

// Click and RClick push a mouse click at 800x600 game coordinates, handled next frame.
func (c *Client) Click(ctx context.Context, x, y int) error  { return c.ok(ctx, "click %d %d", x, y) }
func (c *Client) RClick(ctx context.Context, x, y int) error { return c.ok(ctx, "rclick %d %d", x, y) }

// Key pushes an SDL key name down+up ("Escape", "Return", "i"). Escape toggles the game menu (V): check
// State.GameMenuOpen. Modifier keys are not available (V: "key shift" is "err unknown key").
func (c *Client) Key(ctx context.Context, name string) error { return c.ok(ctx, "key %s", name) }

// Text pushes typed text. Hover moves the mouse without clicking. Wheel scrolls (+up, -down).
func (c *Client) Text(ctx context.Context, s string) error  { return c.ok(ctx, "text %s", s) }
func (c *Client) Hover(ctx context.Context, x, y int) error { return c.ok(ctx, "move %d %d", x, y) }
func (c *Client) Wheel(ctx context.Context, dy int) error   { return c.ok(ctx, "wheel %d", dy) }

// CmdMove walks to a cell position (floating cells, 5 subtiles per cell, V).
func (c *Client) CmdMove(ctx context.Context, x, y float64) error {
	if err := c.autoKick(ctx); err != nil {
		return err
	}
	return c.ok(ctx, "cmd move %s %s", ff(x), ff(y))
}

// CmdInteract walks next to an NPC (index from NPCs or Presets) and opens its menu. A long path may stop
// short: repeat until Menu shows lines (V).
func (c *Client) CmdInteract(ctx context.Context, index int) error {
	if err := c.autoKick(ctx); err != nil {
		return err
	}
	return c.ok(ctx, "cmd interact %d", index)
}

// CorpseInteract takes the player's own corpse number i (V: index -3000-i, "cmd interact -3000").
func (c *Client) CorpseInteract(ctx context.Context, i int) error {
	return c.CmdInteract(ctx, -3000-i)
}

// SkillOpts are the optional tail of "cmd skill <id> <x> <y> [unit] [left] [item]".
type SkillOpts struct {
	Unit int  // target unit id; 0 = none (U: we only saw ids >= 1)
	Left bool // select the skill on the left button (V: "left" was used in the spike)
	Item int  // charged item unit id; 0 = none
}

// CmdSkill casts skill id at a cell position, optionally at a unit.
func (c *Client) CmdSkill(ctx context.Context, id int, x, y float64, o SkillOpts) error {
	if o.Item != 0 && !o.Left {
		return fmt.Errorf("d2dctl: SkillOpts.Item needs Left (positional tail, right button form unknown, U)")
	}
	line := fmt.Sprintf("cmd skill %d %s %s", id, ff(x), ff(y))
	if o.Unit != 0 || o.Left {
		line += fmt.Sprintf(" %d", o.Unit)
	}
	if o.Left {
		line += " left"
	}
	if o.Item != 0 {
		line += fmt.Sprintf(" %d", o.Item)
	}
	return c.ok(ctx, "%s", line)
}

// CmdPickup walks to and picks up a ground unit (V: gold piles too).
func (c *Client) CmdPickup(ctx context.Context, unit int) error {
	return c.ok(ctx, "cmd pickup %d", unit)
}

// CmdResurrect sends "cmd resurrect" (merc/ghost case upstream; not tested by us, U).
func (c *Client) CmdResurrect(ctx context.Context) error { return c.ok(ctx, "cmd resurrect") }

// CmdStat adds a stat point (cmd stat <stat> [n]); CmdSkillPt spends a skill point on the tree index.
func (c *Client) CmdStat(ctx context.Context, stat, n int) error {
	return c.ok(ctx, "cmd stat %d %d", stat, n)
}
func (c *Client) CmdSkillPt(ctx context.Context, index int) error {
	return c.ok(ctx, "cmd skillpt %d", index)
}

// CmdSelect puts a skill on the left or right button.
func (c *Client) CmdSelect(ctx context.Context, skill int, left bool) error {
	l := 0
	if left {
		l = 1
	}
	return c.ok(ctx, "cmd select %d %d", skill, l)
}

// CmdBelt drinks the potion in a belt column; merc=true is the "shift-click" variant (the merc drinks).
func (c *Client) CmdBelt(ctx context.Context, slot int, merc bool) error {
	if merc {
		return c.ok(ctx, "cmd belt %d merc", slot)
	}
	return c.ok(ctx, "cmd belt %d", slot)
}

// BeltPut moves the carried item id into belt box (lab3+ "cmd belt put", V in D2DVERBS: d2d has no shift-click to belt).
// A box that is occupied swaps and leaves the displaced item on the cursor, so the caller picks a free one.
func (c *Client) BeltPut(ctx context.Context, id, box int) error {
	return c.ok(ctx, "cmd belt put %d %d", id, box)
}

// NPCs lists named NPCs/objects on screen.
func (c *Client) NPCs(ctx context.Context) (List[NPC], error) {
	r, err := c.Do(ctx, "npcs")
	return parseList(r, err, ParseNPC)
}

// Monsters lists the monsters of the current level.
func (c *Client) Monsters(ctx context.Context) (List[Monster], error) {
	r, err := c.Do(ctx, "monsters")
	return parseList(r, err, ParseMonster)
}

// Ground lists items on the ground.
func (c *Client) Ground(ctx context.Context) (List[GroundItem], error) {
	r, err := c.Do(ctx, "ground")
	return parseList(r, err, ParseGround)
}

// Menu lists the lines of the open NPC menu, header first.
func (c *Client) Menu(ctx context.Context) (List[MenuLine], error) {
	r, err := c.Do(ctx, "menu")
	return parseList(r, err, ParseMenuLine)
}

// Items lists carried items with their tooltip text.
func (c *Client) Items(ctx context.Context) ([]Item, *Reply, error) {
	r, err := c.Do(ctx, "items")
	if err != nil {
		return nil, r, err
	}
	items, bad := ParseItems(r.Lines)
	r.Bad = append(r.Bad, bad...)
	return items, r, nil
}

// Debug sends "debug <args...>" and returns the raw reply.
func (c *Client) Debug(ctx context.Context, args ...string) (*Reply, error) {
	return c.Do(ctx, strings.TrimSpace("debug "+strings.Join(args, " ")))
}

func (c *Client) dbg(ctx context.Context, format string, a ...any) error {
	_, err := c.Debug(ctx, fmt.Sprintf(format, a...))
	return err
}

// Players lists the game's players.
func (c *Client) Players(ctx context.Context) (List[Player], error) {
	r, err := c.Debug(ctx, "players")
	return parseList(r, err, ParsePlayer)
}

// Level returns the player's level id, position and size.
func (c *Client) Level(ctx context.Context) (LevelInfo, error) {
	r, err := c.Debug(ctx, "level")
	if err != nil {
		return LevelInfo{}, err
	}
	return parseLevel(r.Rest)
}

// QuestShow returns quest q's flag bits and the Act 1 record states.
func (c *Client) QuestShow(ctx context.Context, q int) (QuestInfo, error) {
	r, err := c.Debug(ctx, "quest", strconv.Itoa(q), "show")
	if err != nil {
		return QuestInfo{}, err
	}
	return parseQuest(r.Rest)
}

// QuestDone marks quest q done (and runs its join chain); QuestReset clears its bits.
func (c *Client) QuestDone(ctx context.Context, q int) error  { return c.dbg(ctx, "quest %d", q) }
func (c *Client) QuestReset(ctx context.Context, q int) error { return c.dbg(ctx, "quest %d reset", q) }

// Die kills the player (death penalty, corpse). Corpses lists the player's corpses.
func (c *Client) Die(ctx context.Context) error { return c.dbg(ctx, "die") }
func (c *Client) Corpses(ctx context.Context) (List[Corpse], error) {
	r, err := c.Debug(ctx, "corpses")
	return parseList(r, err, ParseCorpse)
}

// Kill kills the level's monsters except n of them and those of MonStats id spare ("" = none); n < 0 omits it.
func (c *Client) Kill(ctx context.Context, n int, spare string) error {
	args := []string{"kill"}
	if n >= 0 {
		args = append(args, strconv.Itoa(n))
	}
	if spare != "" {
		args = append(args, "spare", spare)
	}
	_, err := c.Debug(ctx, args...)
	return err
}

// SetStat sets a character stat 0..15 (12 level, 13 exp, 14 gold, 15 stash gold, V from the spike).
func (c *Client) SetStat(ctx context.Context, id, v int) error {
	return c.dbg(ctx, "stat %d %d", id, v)
}

// StatPts and SkillPts set the unspent stat / skill points.
func (c *Client) StatPts(ctx context.Context, n int) error  { return c.dbg(ctx, "statpts %d", n) }
func (c *Client) SkillPts(ctx context.Context, n int) error { return c.dbg(ctx, "skillpts %d", n) }

// Equip wears a normal item code in a body slot; the reply rest is "quantity=<n>".
func (c *Client) Equip(ctx context.Context, code string, slot int) (*Reply, error) {
	return c.Debug(ctx, "equip", code, strconv.Itoa(slot))
}

// ClearInv empties the inventory grid.
func (c *Client) ClearInv(ctx context.Context) error { return c.dbg(ctx, "clearinv") }

// Goto arrives on a Levels.txt id. V: outdoor levels work, Burial Grounds (17) works, "debug goto 8"
// (Den of Evil) renders black with no monsters; use Warps + Enter for caves. Marks a kick as needed.
func (c *Client) Goto(ctx context.Context, level int) error {
	err := c.dbg(ctx, "goto %d", level)
	if err == nil {
		c.kickPending.Store(true)
	}
	return err
}

// Warp puts the player at DS1 cell (x, y) of the level (next to another level it crosses into it).
func (c *Client) Warp(ctx context.Context, x, y float64) error {
	err := c.dbg(ctx, "warp %s %s", ff(x), ff(y))
	if err == nil {
		c.kickPending.Store(true)
	}
	return err
}

// Warps lists the level's cave mouths and stairs; Enter takes warp i.
func (c *Client) Warps(ctx context.Context) (List[Warp], error) {
	r, err := c.Debug(ctx, "warps")
	return parseList(r, err, ParseWarp)
}
func (c *Client) Enter(ctx context.Context, i int) error {
	err := c.dbg(ctx, "enter %d", i)
	if err == nil {
		c.kickPending.Store(true)
	}
	return err
}

// Portals, Presets and Objects list the level's town portals, placed objects and shrines/chests.
func (c *Client) Portals(ctx context.Context) (List[Portal], error) {
	r, err := c.Debug(ctx, "portals")
	return parseList(r, err, ParsePortal)
}
func (c *Client) Presets(ctx context.Context) (List[Preset], error) {
	r, err := c.Debug(ctx, "presets")
	return parseList(r, err, ParsePreset)
}
func (c *Client) Objects(ctx context.Context) (List[Object], error) {
	r, err := c.Debug(ctx, "objects")
	return parseList(r, err, ParseObject)
}

// Difficulty restarts the game on difficulty d (0 normal, 1 nightmare, 2 hell); takes 5 to 6 s (V), so
// give the context room. A level-1 character dies at once past Normal (upstream note).
func (c *Client) Difficulty(ctx context.Context, d int) error {
	err := c.dbg(ctx, "difficulty %d", d)
	if err == nil {
		c.kickPending.Store(true)
	}
	return err
}

// Points gives the character's class skill id n points ("err not a class skill" for other classes' ids, V).
func (c *Client) Points(ctx context.Context, skill, n int) error {
	return c.dbg(ctx, "points %d %d", skill, n)
}

// Passives returns the passive-skill stats as stat -> "value[/itype]".
func (c *Client) Passives(ctx context.Context) (KV, error) {
	r, err := c.Debug(ctx, "passives")
	if err != nil {
		return KV{}, err
	}
	return ParseKV(r.Rest), nil
}

// Attack returns the player's combat sheet (dmg=, ar=, def=, res=f/l/c/p, ...) as key/values.
func (c *Client) Attack(ctx context.Context) (KV, error) {
	r, err := c.Debug(ctx, "attack")
	if err != nil {
		return KV{}, err
	}
	return ParseKV(r.Rest), nil
}

// Blocked asks whether a unit cannot stand at cell (x, y). Grid returns the collision rows at n spots per
// cell (1..5, 5 = subtiles); '#' is blocked.
func (c *Client) Blocked(ctx context.Context, x, y float64) (bool, error) {
	r, err := c.Debug(ctx, "blocked", ff(x), ff(y))
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(r.Rest) == "1", nil
}
func (c *Client) Grid(ctx context.Context, n int) ([]string, error) {
	r, err := c.Debug(ctx, "grid", strconv.Itoa(n))
	if err != nil {
		return nil, err
	}
	return r.Lines, nil
}

// SubTile converts a d2d cell coordinate to our integer subtile (V: cells are floats, 5 subtiles each).
func SubTile(cell float64) int { return int(5 * cell) }

// Cell converts our subtile coordinate to d2d cells.
func Cell(sub int) float64 { return float64(sub) / 5 }

// ClockInfo is the reply of the fork's "step" and "clock" verbs (lockstep fork only; V on the private fork,
// docs/d2d-fork.md section 3). GameMS is the virtual clock in milliseconds, Tick the world tick counter.
type ClockInfo struct {
	Lockstep bool
	Frame    int
	GameMS   int
	Tick     int
	Stepped  int
	Pending  int
	KV       KV
}

func parseClock(r *Reply) ClockInfo {
	kv := ParseKV(strings.Join(append(append([]string{}, r.Lines...), r.Rest), " "))
	ci := ClockInfo{KV: kv}
	ci.Frame, _ = kv.Int("frame")
	ci.GameMS, _ = kv.Int("game_ms")
	ci.Tick, _ = kv.Int("tick")
	ci.Stepped, _ = kv.Int("stepped")
	ci.Pending, _ = kv.Int("pending")
	if v, ok := kv.Int("lockstep"); ok {
		ci.Lockstep = v == 1
	}
	return ci
}

// Step advances a lockstep d2d by n world ticks (40 ms each) and returns the clock after the last one. The reply is
// deferred until the frames are done, so use a context with room for large n (about 3000 ticks/s in town, V).
// Stock d2d answers "err unknown verb".
func (c *Client) Step(ctx context.Context, n int) (ClockInfo, error) {
	r, err := c.Do(ctx, fmt.Sprintf("step %d", n))
	if err != nil {
		return ClockInfo{}, err
	}
	return parseClock(r), nil
}

// Clock reads the virtual clock of the lockstep fork.
func (c *Client) Clock(ctx context.Context) (ClockInfo, error) {
	r, err := c.Do(ctx, "clock")
	if err != nil {
		return ClockInfo{}, err
	}
	return parseClock(r), nil
}
