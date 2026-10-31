package d2dsrc

import (
	"context"
	"fmt"
	"strconv"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
)

// items fills Inventory from "dump items" (or the stock "items" verb when nodump).
func (b *builder) items(ctx context.Context, nodump bool) error {
	b.snap.Inventory = &d2.Inventory{}
	if nodump {
		return b.itemsVerb(ctx)
	}
	d, err := b.s.dump(ctx, "items")
	if err != nil {
		if isCmdErr(err) || err == errNoDump {
			b.rp.unknown("inventory", "dump items failed: "+err.Error())
			return nil
		}
		return err
	}
	b.rp.Source["items"] = "dump"
	list := d.list("items")
	if _, ok := d["items"]; !ok {
		b.rp.unknown("inventory", "dump has no items array")
		return nil
	}
	b.rp.have("inventory")
	for _, it := range list {
		u, ok := b.dumpItem(it)
		if !ok {
			continue
		}
		b.addItem(u)
	}
	b.rp.derived("inventory.stash", "dump loc \"stash\" (V that the fork writes it; U whether stash items are listed while the stash is closed)")
	b.rp.unknown("inventory.item_flags", "only identified/ethereal/socketed/runeword; no start-item or other flag bits")
	return nil
}

// addItem files a unit under Inventory (carried) or Ground... carried only: the dump's "ground" loc never
// occurs for items we own; ground items come from the "ground" verb.
func (b *builder) addItem(u d2.Unit) {
	inv := b.snap.Inventory
	if u.Item.Where == d2.ContCursor {
		c := u
		inv.Cursor = &c
	}
	inv.Items = append(inv.Items, u)
	inv.ItemCount++
}

func (b *builder) row(code string) int {
	if b.s.o.Tables != nil {
		if n, ok := b.s.o.Tables.ItemRow(code); ok {
			b.rp.have("item.class")
			return n
		}
		b.rp.unknown("item.class", "item code "+strconv.Quote(code)+" not in the item tables")
		return -1
	}
	b.rp.unknown("item.class", "no Tables configured: item base row unknown (Class = -1)")
	return -1
}

// dumpItem converts one dump item. Socketed gems/runes (loc "socketed") are not units of the inventory
// list in our reader either, so they are skipped.
func (b *builder) dumpItem(it obj) (d2.Unit, bool) {
	rp := b.rp
	loc, _ := it.str("loc")
	code, _ := it.str("code")
	id, hasID := it.int("id")
	if loc == "socketed" {
		return d2.Unit{}, false
	}
	if !hasID || code == "" {
		rp.unknown("item.id", "dump item without id/code")
		return d2.Unit{}, false
	}
	u := d2.Unit{Type: d2.TypeItem, ID: uint32(id), Class: b.row(code)}
	x, _ := it.int("x")
	y, _ := it.int("y")
	slot, _ := it.int("slot")
	q, qok := it.int("quality")
	ilvl, _ := it.int("ilvl")
	m := &d2.Item{Quality: q, ILvl: ilvl, GX: x, GY: y}
	if !qok {
		rp.unknown("item.quality", "missing in dump item")
	}
	var flags uint32
	if f := it.sub("flags"); f != nil {
		if v, _ := f.bool("identified"); v {
			flags |= d2.FlagIdentified
		}
		if v, _ := f.bool("ethereal"); v {
			flags |= d2.FlagEthereal
		}
		if v, _ := f.bool("socketed"); v {
			flags |= d2.FlagSocketed
		}
		if v, _ := f.bool("runeword"); v {
			flags |= d2.FlagRuneword
		}
	}
	m.Flags = flags
	u.Flags = flags
	uniq, _ := it.int("unique")
	set, _ := it.int("set")
	switch q {
	case d2.QualityUnique:
		m.FileIndex = uniq
	case d2.QualitySet:
		m.FileIndex = set
	}
	switch loc {
	case "body":
		m.Where, m.Body, m.NodePage, u.Mode = d2.ContEquipment, slot, 3, d2.ItemModeEquipped
	case "belt":
		m.Where, m.NodePage, u.Mode = d2.ContBelt, 2, d2.ItemModeBelt
	case "inventory":
		m.Where, m.Page, m.NodePage = d2.ContInventory, 0, 1
	case "stash":
		m.Where, m.Page, m.NodePage = d2.ContStash, 4, 1
	case "cube":
		m.Where, m.Page, m.NodePage = d2.ContCube, 3, 1
	case "cursor":
		m.Where, u.Mode = d2.ContCursor, d2.ItemModeCursor
	default:
		rp.unknown("item.loc", fmt.Sprintf("loc %q not understood", loc))
		return d2.Unit{}, false
	}
	u.X, u.Y = x, y
	u.Item = m
	ex := ItemExtra{Code: code, Sockets: -1}
	ex.Name, _ = it.str("name")
	ex.Qty, _ = it.int("qty")
	if n, ok := it.int("sockets"); ok {
		ex.Sockets = n
	}
	ex.Dur, ex.MaxDur, _ = it.pair("dur")
	ex.Unique, ex.Set = uniq, set
	b.res.Extras.Items[u.ID] = ex
	rp.have("item.code", "item.pos", "item.quality", "item.ilvl", "item.flags")
	return u, true
}

// itemsVerb is the stock path: "items" lists header attrs only (loc 0 inventory, 1 body, 2 belt per the
// spike; stash/cube U), no ilvl, no flags.
func (b *builder) itemsVerb(ctx context.Context) error {
	rp := b.rp
	rp.Source["items"] = "verbs"
	items, r, err := b.s.c.Items(ctx)
	if err != nil {
		if isCmdErr(err) {
			rp.unknown("inventory", "items verb failed: "+err.Error())
			return nil
		}
		return err
	}
	_ = r
	rp.have("inventory")
	for _, it := range items {
		loc := it.Loc()
		id, _ := strconv.Atoi(it.Attrs["id"])
		q, _ := strconv.Atoi(it.Attrs["q"])
		slot, _ := strconv.Atoi(it.Attrs["slot"])
		var col, row int
		fmt.Sscanf(it.Attrs["at"], "%d,%d", &col, &row)
		u := d2.Unit{Type: d2.TypeItem, ID: uint32(id), Class: b.row(it.Code), X: col, Y: row}
		m := &d2.Item{Quality: q, GX: col, GY: row}
		switch loc {
		case 0:
			m.Where, m.NodePage = d2.ContInventory, 1
		case 1:
			m.Where, m.Body, m.NodePage, u.Mode = d2.ContEquipment, slot, 3, d2.ItemModeEquipped
		case 2:
			m.Where, m.NodePage, u.Mode = d2.ContBelt, 2, d2.ItemModeBelt
		default:
			rp.unknown("item.loc", fmt.Sprintf("items loc=%d not understood (stash/cube U)", loc))
			continue
		}
		u.Item = m
		b.res.Extras.Items[u.ID] = ItemExtra{Code: it.Code, Sockets: -1}
		b.addItem(u)
	}
	rp.have("item.code", "item.pos", "item.quality")
	for _, f := range []string{"item.ilvl", "item.flags", "item.unique_set_id"} {
		rp.unknown(f, "stock items verb: tooltip text only")
	}
	return nil
}

// world fills Monsters, Objects and Ground from monsters, npcs, ground and portals.
func (b *builder) world(ctx context.Context) error {
	rp := b.rp
	c := b.s.c
	mons, err := c.Monsters(ctx)
	switch {
	case err == nil:
		rp.have("monsters")
		for _, m := range mons.Items {
			b.monster(m)
		}
		rp.unknown("monster.states", "no child state list (curses, minion mark)")
		rp.unknown("monster.resists", "no per-monster resist/immunity")
		rp.unknown("monster.unique_no", "super unique id not in the monsters line (name only)")
		rp.unknown("monster.hostile", "no hostile flag; summons/pets are not marked (state pets= is a count)")
	case isCmdErr(err):
		rp.unknown("monsters", "monsters verb failed: "+err.Error())
	default:
		return err
	}
	npcs, err := c.NPCs(ctx)
	switch {
	case err == nil:
		rp.have("npcs")
		for _, n := range npcs.Items {
			b.npc(n)
		}
	case isCmdErr(err):
		rp.unknown("npcs", "npcs verb failed: "+err.Error())
	default:
		return err
	}
	g, err := c.Ground(ctx)
	switch {
	case err == nil:
		rp.have("ground")
		for _, it := range g.Items {
			b.ground(it)
		}
	case isCmdErr(err):
		rp.unknown("ground", "ground verb failed: "+err.Error())
	default:
		return err
	}
	if b.snap.Player != nil {
		ps, err := c.Portals(ctx)
		switch {
		case err == nil:
			for i, p := range ps.Items {
				if p.Level != b.snap.Player.Area {
					continue
				}
				x, y := b.pos(p.X, p.Y)
				b.snap.Objects = append(b.snap.Objects, d2.Unit{Type: d2.TypeObject, Class: townPortalObj, ID: uint32(PortalIDBase + i), Mode: 1, X: x, Y: y})
			}
			rp.derived("objects.portals", "debug portals rows on this level, class 59 (objects.txt Town Portal, S)")
		case isCmdErr(err):
			rp.unknown("objects.portals", "debug portals failed: "+err.Error())
		default:
			return err
		}
	}
	rp.unknown("objects.other", "waypoints/chests/shrines: npcs lists named ones only; debug presets/objects are static level data")
	return nil
}

// townPortalObj is the objects.txt id of the town portal (S: D2BS Constants.h, 59; U on 1.14d tables).
const townPortalObj = 59

func (b *builder) monster(m d2dctl.Monster) {
	rp := b.rp
	u := d2.Unit{Type: d2.TypeMonster, ID: uint32(m.Unit), Class: -1, Mode: -1}
	u.X, u.Y = b.pos(m.CellX, m.CellY)
	if t := b.s.o.Tables; t != nil {
		if n, ok := t.MonsterClass(m.ClassID); ok {
			u.Class = n
			rp.have("monster.class")
		} else {
			rp.unknown("monster.class", "class key "+strconv.Quote(m.ClassID)+" not in monstats")
		}
	} else {
		rp.unknown("monster.class", "no Tables configured: monster class id unknown (Class = -1)")
	}
	if n, ok := monModes[m.Mode]; ok {
		u.Mode = n
		rp.derived("monster.mode", "two-letter code via MonMode.txt order (S)")
	} else {
		rp.unknown("monster.mode."+m.Mode, "mode code not in the table; Mode = -1")
	}
	if m.Dead() {
		u.Mode = d2.MonModeDead
	}
	info := &d2.MonsterInfo{Life: m.HP.Cur, MaxLife: m.HP.Max, Level: m.Level, Name: m.Name}
	if m.HP.Max > 0 {
		info.LifePct = m.HP.Cur * 100 / m.HP.Max
	}
	info.Flags = 1 << 1
	switch m.Boss {
	case "champion":
		info.Flags = 1 << 2
	case "unique", "superunique":
		info.Flags = 1 << 3
	case "minion":
		info.Flags |= 1 << 4
	case "-", "":
	default:
		rp.unknown("monster.kind."+m.Boss, "boss column value not understood")
	}
	for i, id := range m.Mods {
		if i >= len(info.Enchants) {
			rp.unknown("monster.enchants_overflow", "more than 9 MonUMod ids; extras dropped")
			break
		}
		if id < 0 || id > 255 {
			rp.unknown("monster.enchant_range", "MonUMod id out of byte range")
			continue
		}
		info.Enchants[i] = uint8(id)
	}
	rp.have("monster.pos", "monster.life", "monster.level", "monster.kind", "monster.enchants")
	u.Monster = info
	b.snap.Monsters = append(b.snap.Monsters, u)
}

// npc files an "npcs" row. With Tables a row whose name is a monstats entry is a town NPC (Monsters, like
// the memory reader's unit table); anything else is an Object. Without Tables the menu flag decides (Cain
// has no menu until the Tristram rescue, V docs/d2d-probes.md, so he lands in Objects then).
func (b *builder) npc(n d2dctl.NPC) {
	rp := b.rp
	id := uint32(NPCIDBase + n.Index)
	x, y := b.pos(n.CellX, n.CellY)
	b.res.Extras.Names[id] = n.Name
	class, isNPC := -1, n.HasMenu
	if t := b.s.o.Tables; t != nil {
		if c, ok := t.MonsterClass(n.Name); ok {
			class, isNPC = c, true
		} else {
			isNPC = false
		}
	}
	if isNPC {
		b.snap.Monsters = append(b.snap.Monsters, d2.Unit{Type: d2.TypeMonster, Class: class, ID: id, Mode: d2.MonModeStand, X: x, Y: y,
			Monster: &d2.MonsterInfo{Name: n.Name, Flags: 1 << 1}})
		rp.derived("npc.mode", "assumed stand (npcs carries no mode)")
		return
	}
	b.snap.Objects = append(b.snap.Objects, d2.Unit{Type: d2.TypeObject, Class: -1, ID: id, Mode: -1, X: x, Y: y})
	rp.unknown("object.class", "npcs gives a name, not an objects.txt id (Class = -1; Extras.Names has the name)")
	rp.unknown("object.mode", "open/closed state not in npcs")
}

func (b *builder) ground(g d2dctl.GroundItem) {
	rp := b.rp
	u := d2.Unit{Type: d2.TypeItem, ID: uint32(g.Unit), Class: b.row(g.Code), Mode: d2.ItemModeGround, X: -1, Y: -1}
	if cx, cy, ok := g.Cell(); ok {
		u.X, u.Y = b.pos(cx, cy)
		rp.have("ground.pos")
	} else {
		rp.unknown("ground.pos", "ground line carries only screen x y here (X,Y = -1)")
	}
	u.Item = &d2.Item{Where: d2.ContGround}
	rp.unknown("ground.quality", "only the label text (name prefixes) shows quality; Quality = 0")
	rp.unknown("ground.ilvl", "not in the ground line")
	b.res.Extras.Items[u.ID] = ItemExtra{Code: g.Code, Label: g.Label, Sockets: -1}
	if g.IsGold() {
		b.res.Extras.Gold = append(b.res.Extras.Gold, u.ID)
	}
	rp.have("ground.code", "ground.label")
	b.snap.Ground = append(b.snap.Ground, u)
}
