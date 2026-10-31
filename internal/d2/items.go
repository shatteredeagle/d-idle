package d2

// Item unit modes (UnitAny.dwMode for items). 0 stored, 1 equipped, 2 belt VERIFIED LIVE; 3 ground
// is what the spike assumed (UNVERIFIED, no ground item was available); 4 cursor UNVERIFIED.
const (
	ItemModeStored   = 0
	ItemModeEquipped = 1
	ItemModeBelt     = 2
	ItemModeGround   = 3
	ItemModeCursor   = 4
)

// Quality (ItemData+0x00). 2 = normal VERIFIED LIVE (starting gear); the rest UNVERIFIED (standard D2).
const (
	QualityLow      = 1
	QualityNormal   = 2
	QualitySuperior = 3
	QualityMagic    = 4
	QualitySet      = 5
	QualityRare     = 6
	QualityUnique   = 7
	QualityCrafted  = 8
)

// Item flag bits (ItemData+0x18). 0x10 identified and 0x20000 start item VERIFIED LIVE (starting gear
// reads 0x00822011 / 0x00a22010); the rest are the standard D2 bits, UNVERIFIED.
const (
	FlagIdentified = 0x00000010
	FlagSocketed   = 0x00000800
	FlagStartItem  = 0x00020000
	FlagEthereal   = 0x00400000
	FlagRuneword   = 0x04000000
)

// Container says where an item lives.
type Container int

const (
	ContNone      Container = iota
	ContInventory           // VERIFIED LIVE (NodePage 1, ItemLocation 0)
	ContStash               // ItemLocation 4: UNVERIFIED (no stash content)
	ContCube                // ItemLocation 3: UNVERIFIED
	ContBelt                // VERIFIED LIVE (NodePage 2)
	ContEquipment           // VERIFIED LIVE (NodePage 3, body location)
	ContGround
	ContCursor
)

func (c Container) String() string {
	return [...]string{"none", "inventory", "stash", "cube", "belt", "equipment", "ground", "cursor"}[c]
}

// Body locations (ItemData+0x44), 4 = right hand weapon VERIFIED LIVE (starting staff); others UNVERIFIED.
const (
	BodyNone    = 0
	BodyHead    = 1
	BodyNeck    = 2
	BodyTorso   = 3
	BodyRHand   = 4
	BodyLHand   = 5
	BodyRRing   = 6
	BodyLRing   = 7
	BodyBelt    = 8
	BodyFeet    = 9
	BodyGloves  = 10
	BodyAltR    = 11
	BodyAltL    = 12
	itemNoWhere = 0xFF
)

// Item is the decoded ItemData of an item unit. UnitAny.Class is the base item row (armor/weapons/misc
// txt index; 63 = a staff row, 587 = healing potion row, 529/530 = tomes in the live sample).
type Item struct {
	Quality   int
	Flags     uint32
	ILvl      int
	FileIndex int // unique/set index, UNVERIFIED (+0x28)
	Body      int // BodyLocation (+0x44)
	Page      int // ItemLocation (+0x45)
	NodePage  int // (+0x69): 1 storage, 2 belt, 3 equipment VERIFIED LIVE
	Where     Container
	// GX, GY: top-left grid cell inside inventory/stash/cube, or belt column (GX) and row (GY).
	// VERIFIED LIVE: ItemPath+0x0C/+0x10 hold them (belt potions 0..3,0; tomes 9,2 and 9,3).
	GX, GY int
}

func (i *Item) Identified() bool { return i.Flags&FlagIdentified != 0 }
func (i *Item) Ethereal() bool   { return i.Flags&FlagEthereal != 0 }
func (i *Item) Socketed() bool   { return i.Flags&FlagSocketed != 0 }

// ItemData layout (VERIFIED LIVE by dumping the starting items unless noted).
const (
	idQuality  = 0x00
	idFlags    = 0x18
	idILvl     = 0x2C
	idFileIdx  = 0x28
	idBody     = 0x44
	idPage     = 0x45
	idNodePage = 0x69
	idNextInv  = 0x64
)

func (r *Reader) itemInfo(u *Unit, data uint32) (*Item, error) {
	b, err := r.bytes(data, 0x70)
	if err != nil {
		return nil, err
	}
	it := &Item{
		Quality:   int(le32(b[idQuality:])),
		Flags:     le32(b[idFlags:]),
		FileIndex: int(le32(b[idFileIdx:])),
		ILvl:      int(le32(b[idILvl:])),
		Body:      int(b[idBody]),
		Page:      int(b[idPage]),
		NodePage:  int(b[idNodePage]),
		GX:        u.X,
		GY:        u.Y,
	}
	switch u.Mode {
	case ItemModeGround:
		it.Where = ContGround
	case ItemModeCursor:
		it.Where = ContCursor
	case ItemModeEquipped:
		it.Where = ContEquipment
	case ItemModeBelt:
		it.Where = ContBelt
	case ItemModeStored:
		switch it.Page {
		case 0:
			it.Where = ContInventory
		case 3:
			it.Where = ContCube
		case 4:
			it.Where = ContStash
		}
	}
	return it, nil
}

// Inventory contents of the local player, grouped by container.
type Inventory struct {
	Items     []Unit // every item unit in the player's inventory list
	Cursor    *Unit  // item held on the cursor, if any
	ItemCount int
}

// Container returns the items in one container, in list order.
func (inv *Inventory) Container(c Container) []Unit {
	var out []Unit
	for _, u := range inv.Items {
		if u.Item != nil && u.Item.Where == c {
			out = append(out, u)
		}
	}
	return out
}

// Equipped returns the item worn at a body location, or nil.
func (inv *Inventory) Equipped(body int) *Unit {
	for i := range inv.Items {
		u := &inv.Items[i]
		if u.Item != nil && u.Item.Where == ContEquipment && u.Item.Body == body {
			return u
		}
	}
	return nil
}

// Inventory struct VERIFIED LIVE: +0x08 pOwner, +0x0C first item, +0x10 last item, +0x20 cursor item,
// +0x28 item count.
func (r *Reader) Inventory() (*Inventory, error) {
	pa, err := r.playerAddr()
	if err != nil {
		return nil, err
	}
	inv, err := r.ptr(pa + uoInventory)
	if err != nil {
		return nil, err
	}
	hdr, err := r.bytes(inv, 0x30)
	if err != nil {
		return nil, err
	}
	if le32(hdr[0x08:]) != pa {
		return nil, ErrBadPointer
	}
	out := &Inventory{ItemCount: int(le32(hdr[0x28:]))}
	seen := map[uint32]bool{}
	for it := le32(hdr[0x0C:]); it != 0 && len(out.Items) < 256; {
		if !okPtr(it) || seen[it] {
			return nil, ErrBadPointer
		}
		seen[it] = true
		u, err := r.readUnit(it, TypeItem)
		if err != nil {
			return nil, err
		}
		if u == nil {
			return nil, ErrBadPointer
		}
		out.Items = append(out.Items, *u)
		data, err := r.ptr(it + uoData)
		if err != nil {
			return nil, err
		}
		if it, err = r.u32(data + idNextInv); err != nil {
			return nil, err
		}
	}
	if cur := le32(hdr[0x20:]); cur != 0 {
		u, err := r.readUnit(cur, TypeItem)
		if err != nil {
			return nil, err
		}
		if u != nil {
			u.Item.Where = ContCursor
			out.Cursor = u
		}
	}
	return out, nil
}

// GroundItems returns item units lying on the ground (mode 3), nearest-first is up to the caller.
func (r *Reader) GroundItems() ([]Unit, error) {
	all, err := r.Units(TypeItem)
	if err != nil {
		return nil, err
	}
	var out []Unit
	for _, u := range all {
		if u.Mode == ItemModeGround {
			out = append(out, u)
		}
	}
	return out, nil
}
