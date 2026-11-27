package watch

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/bulkid"
	item "d2lod-bot/internal/status"
	"d2lod-bot/internal/vault"
)

// The Ethereal Vault page and its routes (docs/vault.md, UI section; design docs/design/loot-vault-2026-11-16.md section 6, slice V4).
// GET /api/vault and GET /api/vault/item/<id> on the viewer; POST /api/vault/{pin,unpin,discard} on the answer listener with the
// same gate as the other answer routes. Read side first: no Keep, Reveal or Sell here (V5, V6), no wallet (V7). The desk is
// late-bound; without a desk, with the knob loot.vault at 0 or with no vault on disk yet the page still opens and says so, and
// every POST answers 409 code "off".

const (
	VaultRoute        = "/api/vault"
	VaultItemRoute    = "/api/vault/item/" // GET /api/vault/item/<id>
	VaultPinRoute     = "/api/vault/pin"
	VaultUnpinRoute   = "/api/vault/unpin"
	VaultDiscardRoute = "/api/vault/discard"
	VaultKeepRoute    = "/api/vault/keep"
	VaultRevealRoute  = "/api/vault/reveal"
	VaultSellRoute    = "/api/vault/sell"
	VaultCancelRoute  = "/api/vault/cancel"
	VaultEquipRoute   = "/api/vault/equip"       // POST {ids:[one id], slot: auto|left|right, now}: one-tap Equip (docs/vault.md, Equip)
	VaultEquipCancel  = "/api/vault/equipcancel" // POST {ids}: cancel an Equip order that has not started
	VaultEquipGet     = "/api/vault/equip/"      // GET /api/vault/equip/<id>: the sheet's data and the order's status (polled)
	VaultMissedRoute  = "/api/vault/missed"      // POST {"dismiss":true}: clear the missed-drops counter and list
	VaultPromptRoute  = "/api/vault/sellprompt"  // POST {"dismiss":true}: mute the "short on gold" prompt
	VaultSellQuote    = "/api/vault/sellquote"   // POST {ids}: what a sale of them pays, the stash after and why some are left out (changes nothing)
	vaultSchema       = "watch.vault/1"

	vaultPageLimit = 60  // rows per page when the client names none
	vaultMaxLimit  = 200 // rows per page at most
	vaultMaxIDs    = 150 // ids in one POST (the answer body is capped at 4 KiB: about 20 bytes an id); the page sends bigger selections in chunks
)

// VaultRoutes are the viewer paths (the page and its files, the two GET routes); VaultPostRoutes the answer ones.
var (
	VaultRoutes     = []string{VaultRoute, VaultItemRoute, VaultEquipGet, "/watch/vault", "/watch/vault.js", "/watch/vault-logic.js", "/watch/vault.css", "/watch/vault.json"}
	VaultPostRoutes = []string{VaultPinRoute, VaultUnpinRoute, VaultDiscardRoute, VaultKeepRoute, VaultRevealRoute, VaultSellRoute, VaultCancelRoute, VaultPromptRoute, VaultMissedRoute, VaultEquipRoute, VaultEquipCancel, VaultRevealGoldRoute, VaultRevealGoldCancelRoute, VaultSellQuote}
)

// VaultDesk is what the routes need from the bot side (cmd/d2bot/vaultwire.go). Every field is a func read per request, so a
// hero switch or a knob change shows at once.
type VaultDesk struct {
	On    func() bool         // the knob loot.vault; nil counts as on
	Store func() *vault.Store // the active hero's vault, nil while there is none
	Tiers *vault.Tiers        // the capacity table, for the tier line
	Owned func() []string     // purchased capacity steps (V7); nil: none
	Keep  VaultKeeper         // the Keep queue (V5); nil: Keep is not part of this bot
	Act   VaultActor          // Reveal, Sell and the sell prompt (V6); nil: not part of this bot
	Equip VaultEquipper       // one-tap Equip (loot.vaultEquip); nil: not part of this bot
	Tip   VaultTipFunc        // the tooltip facts of a row (base type, art, stat lines); nil: rows carry none
	// MassFilter is the identify filter table (data/bulkid); nil: no mass reveal preview. Quality names a quality number; ScrollGold is a scroll's price (information).
	MassFilter *bulkid.Filter
	Quality    func(int) string
	ScrollGold int
	// AutoTrim is the knob loot.vaultAutoTrim (default off): off means a full vault removes and sells nothing by itself.
	AutoTrim func() bool
}

// VaultTip is what the inventory tooltip needs beyond the stored fields: the same fields status.Item carries (base facts, art
// key, kind, sell price, stat lines), so inventory-tip.js draws a vault row unchanged.
type VaultTip struct {
	Base  *item.ItemBase
	Art   string
	Tint  string
	Kind  string
	Sell  int
	Props []string
}

// VaultTipFunc builds a row's tooltip facts. props is "" for an unidentified record: the caller of this func never hands
// out what identify costs a charge for, and vaultRow drops Props of an unidentified row again after the call.
type VaultTipFunc func(r VaultTipIn) VaultTip

// VaultTipIn is what the tip source may read of a record. Props is empty for an unidentified record.
type VaultTipIn struct {
	Code                 string
	Quality              int
	Identified, Ethereal bool
	Props                string
	Pic, FileIndex       int
	Tint                 string
}

// VaultActor is Reveal and Sell as the routes see them (cmd/d2bot/vaultkeep.go adapts internal/vaultkeep).
type VaultActor interface {
	GateReveal(ctx context.Context) (ok bool, code, why string)
	EnqueueReveal(ctx context.Context, ids []string, now bool) []VaultKeepResult
	GateSell(ctx context.Context) (ok bool, code, why string)
	EnqueueSell(ctx context.Context, ids []string, now bool) []VaultKeepResult
	Cancel(ids []string) []VaultKeepResult
	DismissOffer()
	Act(ctx context.Context) VaultActView
}

// VaultEtherActor is the optional part of a VaultActor that pays reveals with Ether, on an explicit tap only.
type VaultEtherActor interface {
	EnqueueRevealEther(ctx context.Context, ids []string, now bool) []VaultKeepResult
}

// VaultActView is the state of Reveal and Sell in GET /api/vault: the price shown before the click, what is queued, the
// sentences under the toolbar and the opt-in prompt.
type VaultActView struct {
	Reveal VaultRevealView `json:"reveal"`
	Sell   VaultSellView   `json:"sell"`
	Prompt *VaultPrompt    `json:"prompt,omitempty"`
}

type VaultRevealView struct {
	Enabled  bool   `json:"enabled"`
	Why      string `json:"why,omitempty"` // when not enabled: shown on the button (code noscroll offers the Buy action)
	Code     string `json:"code,omitempty"`
	Charges  int    `json:"charges"` // identify scrolls and tome charges in the pack, -1 unknown
	Free     bool   `json:"free"`    // Cain stands here: Reveal costs nothing
	PerItem  int    `json:"perItem"`
	EtherPer int    `json:"etherPer,omitempty"` // Ether one tapped reveal costs; 0: Ether is not used
	Balance  int    `json:"balance"`            // Ether in the wallet, -1 none
	Queued   int    `json:"queued"`
	Working  int    `json:"working"`
	Done     int    `json:"done"`
	Line     string `json:"line,omitempty"`
	// Methods are the payments the page offers (knob loot.vaultRevealMethods, default only "gold"): free, ether, gold. The other
	// routes and the keeper still work; the page draws only these and the bar quotes only these. nil: all three (older bots).
	Methods []string `json:"methods,omitempty"`
	// Note says why identify is manual (Cain is not rescued while identify.policy counts on him).
	Note string `json:"note,omitempty"`
	// Mass previews the "Reveal filtered" button: the ids it would queue (to /api/vault/reveal) and the cost sentence.
	Mass *bulkid.Mass `json:"mass,omitempty"`
	// Bar is the one Reveal bar: the payments (free, Ether, gold) with a cost line each, the default and the skipped items.
	Bar  *bulkid.Bar    `json:"bar,omitempty"`
	Gold *VaultGoldInfo `json:"gold,omitempty"`
}

// VaultSale is the instant sale as the page shows it (internal/vaultkeep SaleView).
type VaultSale struct {
	ID          string            `json:"id"`
	State       string            `json:"state"` // queued, credited, done, failed
	Items       int               `json:"items"`
	Done        int               `json:"done"`
	Gold        int               `json:"gold"`
	StashBefore int               `json:"stashBefore"`
	StashAfter  int               `json:"stashAfter"`
	Note        string            `json:"note,omitempty"`
	Line        string            `json:"line"`
	All         bool              `json:"all,omitempty"`     // a Sell all order
	Skipped     []VaultSaleReason `json:"skipped,omitempty"` // Sell all: the records left out and why
}

type VaultSaleLine struct {
	ID   string `json:"id"`
	Gold int    `json:"gold"`
}

// VaultSaleReason is one group of records a sale leaves out and why ("20 pinned (never sold)").
type VaultSaleReason struct {
	Code string `json:"code"`
	N    int    `json:"n"`
	Why  string `json:"why"`
}

// VaultSaleQuote answers POST /api/vault/sellquote: what a sale of the asked ids would do, without doing it.
type VaultSaleQuote struct {
	Asked       int               `json:"asked"`
	Count       int               `json:"count"` // records in the sale
	Lines       []VaultSaleLine   `json:"lines"`
	Total       int               `json:"total"`
	Stash       int               `json:"stash"`
	StashAfter  int               `json:"stashAfter"`
	Room        int               `json:"room"`
	Reasons     []VaultSaleReason `json:"reasons,omitempty"`
	ConfirmAt   int               `json:"confirmAt"`
	NeedConfirm bool              `json:"needConfirm"`
	Why         string            `json:"why,omitempty"` // the sale cannot run: shown as it is
}

// VaultSaleActor is the optional instant sale of a VaultActor (the actor of cmd/d2bot implements it): Sell removes the records
// and credits the stash at once, with the price the engine pays. Without it, or while SaleOn says no, Sell queues as before.
type VaultSaleActor interface {
	SaleOn(ctx context.Context) (bool, string)
	SaleQuote(ctx context.Context, ids []string) VaultSaleQuote
	SellNow(ctx context.Context, ids []string, confirm bool) ([]VaultKeepResult, *VaultSale)
}

// VaultSellAll is the "Sell all (N)" button's number: the records a whole-vault sale would take now (no prices, no stash room: the
// quote of the first tap says those) and the records it leaves out, by reason.
type VaultSellAll struct {
	N    int               `json:"n"`
	Left []VaultSaleReason `json:"left,omitempty"`
}

// VaultSellAllActor is the optional whole-vault sale of a VaultSaleActor (scope all_sellable). The bot lists the records
// itself when it runs; ids are never sent. Quote works the plan out without changing anything; IDs is the list the old
// town-visit queue takes when the instant sale is not available.
type VaultSellAllActor interface {
	SellAllQuote(ctx context.Context) VaultSaleQuote
	SellAll(ctx context.Context, orderID string) (code, why string, sale *VaultSale)
	SellAllIDs() []string
}

type VaultSellView struct {
	Enabled bool          `json:"enabled"`
	All     *VaultSellAll `json:"all,omitempty"` // Sell all: the count of the whole vault (nil: this bot has no Sell all)
	// Instant: Sell is instant now (the stash is credited and the records leave at once); the page asks the quote first and
	// confirms at ConfirmAt gold. InstantWhy says why not (knob 0, an older d2d). Sale is the last sale (shown for ten minutes).
	Instant    bool       `json:"instant"`
	InstantWhy string     `json:"instantWhy,omitempty"`
	ConfirmAt  int        `json:"confirmAt,omitempty"`
	Sale       *VaultSale `json:"sale,omitempty"`
	Why        string     `json:"why,omitempty"`
	Code       string     `json:"code,omitempty"`
	Queued     int        `json:"queued"`
	Working    int        `json:"working"`
	Sold       int        `json:"sold"`
	Gold       int        `json:"gold"` // earned in this process
	Line       string     `json:"line,omitempty"`
}

// VaultPrompt is the opt-in "short on gold: sell these?" offer; it exists only while loot.vaultSellPrompt is on.
type VaultPrompt struct {
	Short int      `json:"short"`
	IDs   []string `json:"ids"`
	Gold  int      `json:"gold"`
	Names []string `json:"names"`
}

// VaultKeeper is the Keep queue as the routes see it (cmd/d2bot/vaultkeep.go adapts internal/vaultkeep).
type VaultKeeper interface {
	// Gate says whether a press can be queued; code is a short word (off, dead, verbs, stopped) the page can branch on.
	Gate(ctx context.Context) (ok bool, code, why string)
	Enqueue(ctx context.Context, ids []string, target string, now bool) []VaultKeepResult
	Progress() VaultKeepView
}

// VaultEquipper is one-tap Equip as the routes see it (cmd/d2bot adapts internal/vaultkeep). Preview returns a JSON-able
// value: the slots, what is worn, whether each fits and why not, plus the order's status.
type VaultEquipper interface {
	Gate(ctx context.Context) (ok bool, code, why string)
	Preview(id string) (view any, found bool)
	Enqueue(ctx context.Context, id, slot string, now bool) VaultKeepResult
	EnqueueSet(ctx context.Context, ids []string, now bool) []VaultKeepResult // "Wear set": many ids, one confirm
	Cancel(id string) VaultKeepResult
}

// VaultEquipFlag says whether the sheet may offer Equip.
type VaultEquipFlag struct {
	Enabled bool   `json:"enabled"`
	Why     string `json:"why,omitempty"`
	Direct  bool   `json:"direct,omitempty"` // Equip v2: one tap wears at once (no order, no town trip)
	Merc    bool   `json:"merc,omitempty"`   // Equip on merc is on offer (the fork has item merc-equip)
}

// VaultMercEquipper is the optional part of a VaultEquipper that gives a vault item to the mercenary at once
// (POST /api/vault/equip with target "merc"; card mercvault).
type VaultMercEquipper interface {
	MercDirect(ctx context.Context) bool
	EquipMerc(ctx context.Context, id string) VaultKeepResult
}

// VaultDirectEquipper is the optional part of a VaultEquipper that knows the direct path is on (probed verbs, knob).
type VaultDirectEquipper interface {
	Direct(ctx context.Context) bool
}

// VaultKeepResult is the outcome of queueing one id.
type VaultKeepResult struct {
	ID      string
	OK      bool
	Already bool
	Code    string
	Why     string

	// Equip v2: the piece was worn at once (the direct verbs); Slot is where, Removed what came off and Where what became of it.
	Direct  bool
	Slot    string
	Removed []string
	Where   string
}

// VaultKeepView is the queue's state in GET /api/vault: counts, the progress sentence, and why nothing moves.
type VaultKeepView struct {
	Enabled  bool   `json:"enabled"`
	Why      string `json:"why,omitempty"` // when not enabled: the reason, shown on the button
	Queued   int    `json:"queued"`
	Creating int    `json:"creating"`
	Done     int    `json:"done"`
	Full     bool   `json:"full,omitempty"`
	Line     string `json:"line,omitempty"`
}

type VaultAPI struct {
	mu sync.Mutex
	d  *VaultDesk
}

func NewVaultAPI() *VaultAPI { return &VaultAPI{} }

func (a *VaultAPI) Bind(d VaultDesk) { a.mu.Lock(); a.d = &d; a.mu.Unlock() }

func (a *VaultAPI) desk() *VaultDesk { a.mu.Lock(); defer a.mu.Unlock(); return a.d }

// quality ids of the game (value.json documents them) to the words the inventory page and its tooltip use.
var vaultQuality = map[int]string{1: "low", 2: "normal", 3: "superior", 4: "magic", 5: "set", 6: "rare", 7: "unique", 8: "crafted"}

func qualityWord(q int) string {
	if w, ok := vaultQuality[q]; ok {
		return w
	}
	return "normal"
}

func qualityID(w string) (int, bool) {
	if n, err := strconv.Atoi(w); err == nil {
		_, ok := vaultQuality[n]
		return n, ok
	}
	for id, name := range vaultQuality {
		if name == w {
			return id, true
		}
	}
	return 0, false
}

// vaultKind is one entry of the "Kind" select: a set of quality ids and/or item code prefixes, in web/vault.json (data, not code).
type vaultKind struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	Quality    []int    `json:"quality,omitempty"`
	CodePrefix []string `json:"codePrefix,omitempty"`
}

type vaultTexts struct {
	Kinds []vaultKind       `json:"kinds"`
	Types map[string]string `json:"types"` // item type code to its label; a code without one shows as itself
}

func vaultData() vaultTexts {
	var t vaultTexts
	if b, err := webFS.ReadFile("web/vault.json"); err == nil {
		_ = json.Unmarshal(b, &t)
	}
	return t
}

// VaultItem is a record as the page draws it: the shape of an inventory item where the tooltip needs it (name, code, quality
// word, identified, ethereal, sockets, level), so inventory-tip.js draws it unchanged, plus the vault's own fields. An
// unidentified record shows only what the game shows: the stored name is the as-displayed one and nothing else of the item
// is sent (no property text exists in the store for it).
type VaultItem struct {
	ID         string         `json:"id"`
	Code       string         `json:"code"`
	Name       string         `json:"name"`
	Quality    string         `json:"quality"`
	Q          int            `json:"q"`
	Ilvl       int            `json:"ilvl"`
	Level      int            `json:"level"`
	Identified bool           `json:"identified"`
	Base       *item.ItemBase `json:"base,omitempty"` // tooltip facts, as in status.Item
	Art        string         `json:"art,omitempty"`
	Tint       string         `json:"tint,omitempty"`
	Kind       string         `json:"kind,omitempty"`
	SellPrice  int            `json:"sellPrice,omitempty"` // the vendor price the tooltip shows (the row's own "sell" is the Sell errand's state)
	Props      []string       `json:"props,omitempty"`     // stat lines of an IDENTIFIED row only (docs/vault.md, identify stays a step)
	Ethereal   bool           `json:"ethereal,omitempty"`
	Sockets    int            `json:"sockets,omitempty"`
	Pin        bool           `json:"pin,omitempty"`
	State      string         `json:"state"`
	Status     string         `json:"status"`
	Value      int            `json:"value"`
	Found      string         `json:"found,omitempty"` // RFC 3339
	Area       string         `json:"area,omitempty"`
	Reason     string         `json:"reason,omitempty"`
	Keep       string         `json:"keep,omitempty"` // queued, creating, kept or failed: the Keep queue's word for the row
	Where      string         `json:"where,omitempty"`
	Reveal     string         `json:"reveal,omitempty"`       // queued, working or failed: the Reveal errand's word for the row
	RevealWhy  string         `json:"revealReason,omitempty"` // why a reveal failed
	Sell       string         `json:"sell,omitempty"`         // queued, selling, sold or failed
	SellWhy    string         `json:"sellReason,omitempty"`
}

const (
	keepReasonPrefix   = "keep: " // vaultkeep stores failure reasons with this prefix; the row shows the sentence
	revealReasonPrefix = "reveal: "
	sellReasonPrefix   = "sell: "
)

func vaultRow(r *vault.Rec, tip VaultTipFunc) VaultItem {
	name := r.Name
	if name == "" {
		name = r.Code
	}
	v := VaultItem{ID: r.ID, Code: r.Code, Name: name, Quality: qualityWord(r.Quality), Q: r.Quality, Ilvl: r.Ilvl, Level: r.Ilvl,
		Identified: r.Identified, Ethereal: r.Eth, Sockets: r.Sockets, Pin: r.Pin, State: string(r.State), Status: r.Status(),
		Value: r.Value, Area: r.Found.Area, Reason: r.Reason}
	if !r.Found.Time.IsZero() {
		v.Found = r.Found.Time.UTC().Format(time.RFC3339)
	}
	if tip != nil {
		props := ""
		if r.Identified {
			props = r.Props
		}
		t := tip(VaultTipIn{Code: r.Code, Quality: r.Quality, Identified: r.Identified, Ethereal: r.Eth, Props: props, Pic: r.Pic, FileIndex: r.FileIndex, Tint: r.Tint})
		v.Base, v.Art, v.Tint, v.Kind, v.SellPrice = t.Base, t.Art, t.Tint, t.Kind, t.Sell
		if r.Identified {
			v.Props = t.Props
		}
	}
	switch {
	case r.State == vault.KeepQ && r.InFlight:
		v.Keep = "creating"
	case r.State == vault.KeepQ:
		v.Keep = "queued"
	case r.State == vault.Kept:
		v.Keep = "kept"
	case r.State == vault.Recorded && strings.HasPrefix(r.Reason, keepReasonPrefix):
		v.Keep = "failed"
	}
	if v.Keep != "" {
		v.Reason, v.Where = strings.TrimPrefix(r.Reason, keepReasonPrefix), r.Where
	}
	switch {
	case (r.State == vault.Recorded || r.State == vault.KeepQ) && r.RevQ && r.Stage != "":
		v.Reveal = "working"
	case (r.State == vault.Recorded || r.State == vault.KeepQ) && r.RevQ:
		v.Reveal = "queued"
	case r.State == vault.Recorded && r.Stage == vault.StageRevCleanup:
		v.Reveal = "working"
	case r.State == vault.Recorded && strings.HasPrefix(r.Reason, revealReasonPrefix):
		v.Reveal, v.RevealWhy = "failed", strings.TrimPrefix(r.Reason, revealReasonPrefix)
		v.Reason = v.RevealWhy
	case r.State == vault.KeepQ && strings.HasPrefix(r.Reason, revealReasonPrefix):
		v.Reveal, v.RevealWhy = "failed", strings.TrimPrefix(r.Reason, revealReasonPrefix)
	}
	switch {
	case r.State == vault.SellQ && r.Stage != "":
		v.Sell = "selling"
	case r.State == vault.SellQ:
		v.Sell = "queued"
	case r.State == vault.Sold:
		v.Sell = "sold"
		if strings.HasPrefix(r.Reason, sellReasonPrefix) {
			v.SellWhy, v.Reason = strings.TrimPrefix(r.Reason, sellReasonPrefix), strings.TrimPrefix(r.Reason, sellReasonPrefix)
		}
	case r.State == vault.Recorded && strings.HasPrefix(r.Reason, sellReasonPrefix):
		v.Sell, v.SellWhy = "failed", strings.TrimPrefix(r.Reason, sellReasonPrefix)
		v.Reason = v.SellWhy
	}
	if r.State == vault.Sold {
		v.Value = r.Gold // the row of a sold item shows what it fetched
	}
	return v
}

// VaultCapacity is the meter. Level is ok, warn (80 percent), full (100) or over (a shrink left more than fits).
type VaultCapacity struct {
	Count    int    `json:"count"`
	Capacity int    `json:"capacity"`
	Pins     int    `json:"pins"`
	PinLimit int    `json:"pinLimit"`
	Percent  int    `json:"percent"`
	Level    string `json:"level"`
	Over     int    `json:"over,omitempty"` // how many records to discard before recording resumes
}

type VaultUpgrade struct {
	Available bool        `json:"available"`
	Currency  string      `json:"currency"`
	Reason    string      `json:"reason"`
	Next      *vault.Step `json:"next,omitempty"`
	Max       int         `json:"max"`
}

func capacityOf(st vault.Status) VaultCapacity {
	c := VaultCapacity{Count: st.Count, Capacity: st.Capacity, Pins: st.Pins, PinLimit: st.PinLimit, Level: "ok"}
	if st.Capacity > 0 {
		c.Percent = st.Count * 100 / st.Capacity
	}
	switch {
	case st.Over:
		c.Level, c.Over = "over", st.Count-st.Capacity
	case st.Full || c.Percent >= 100:
		c.Level = "full"
	case c.Percent >= 80:
		c.Level = "warn"
	}
	return c
}

type vaultFacet struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	N     int    `json:"n"`
}

// VaultView is the reply of GET /api/vault.
type VaultView struct {
	Schema    string          `json:"schema"`
	Mode      string          `json:"mode"`   // "on", or "off" while loot.vault is 0 (read only)
	Exists    bool            `json:"exists"` // a vault is on disk for the hero
	Hero      string          `json:"hero,omitempty"`
	Total     int             `json:"total"`
	Offset    int             `json:"offset"`
	Limit     int             `json:"limit"`
	Items     []VaultItem     `json:"items"`
	Capacity  VaultCapacity   `json:"capacity"`
	Upgrade   VaultUpgrade    `json:"upgrade"`
	Kinds     []vaultKind     `json:"kinds,omitempty"`
	Types     []vaultFacet    `json:"types,omitempty"`
	Anomalies int             `json:"anomalies,omitempty"`
	Equip     *VaultEquipFlag `json:"equip,omitempty"` // one-tap Equip: offered (knob on) or why not; absent when the bot has none
	Keep      *VaultKeepView  `json:"keep,omitempty"`  // schema additive: absent when the bot has no Keep queue
	Act       *VaultActView   `json:"act,omitempty"`   // Reveal and Sell (V6); absent when the bot has neither
	// AutoTrim is loot.vaultAutoTrim: false (default) the vault never removes an item by itself, so a full vault does not record new drops.
	AutoTrim bool `json:"autoTrim"`
	// Missed counts the drops not recorded while the vault was full, with the best few; absent when none.
	Missed *VaultMissed `json:"missed,omitempty"`
	// Marked lists the records flagged for sale that wait for the player's "Sell these N now" (a POST to the sell route with these ids); absent when none.
	Marked *VaultMarked `json:"marked,omitempty"`
}

// VaultMissed is the missed-drops counter of the page.
type VaultMissed struct {
	Count int         `json:"count"`
	Since string      `json:"since,omitempty"`
	Best  []VaultMiss `json:"best,omitempty"`
}

type VaultMiss struct {
	Name    string `json:"name"`
	Quality string `json:"quality"`
	Q       int    `json:"q"`
	Ilvl    int    `json:"ilvl"`
}

// VaultMarked is the "Marked for sale" list.
type VaultMarked struct {
	N     int      `json:"n"`
	IDs   []string `json:"ids"`
	Names []string `json:"names,omitempty"` // the first few, for the sentence
}

type vaultQuery struct {
	f      vault.Filter
	facets bool
}

// parseVaultQuery turns the query string into a vault.Filter; a bad value is a 400 naming it. kinds come from web/vault.json.
func parseVaultQuery(q map[string][]string, kinds []vaultKind) (vaultQuery, error) {
	var o vaultQuery
	get := func(k string) string {
		if v := q[k]; len(v) > 0 {
			return v[0]
		}
		return ""
	}
	num := func(k string, lo, hi int) (int, error) {
		s := get(k)
		if s == "" {
			return 0, nil
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < lo || n > hi {
			return 0, errors.New("bad " + k)
		}
		return n, nil
	}
	flag := func(k string) (*bool, error) {
		switch get(k) {
		case "":
			return nil, nil
		case "1", "true":
			t := true
			return &t, nil
		case "0", "false":
			f := false
			return &f, nil
		}
		return nil, errors.New("bad " + k)
	}
	f := &o.f
	var err error
	if f.Offset, err = num("offset", 0, 1<<30); err != nil {
		return o, err
	}
	if f.Limit, err = num("limit", 0, vaultMaxLimit); err != nil {
		return o, err
	}
	if f.Limit == 0 {
		f.Limit = vaultPageLimit
	}
	if f.MinIlvl, err = num("ilvlMin", 0, 200); err != nil {
		return o, err
	}
	if f.MaxIlvl, err = num("ilvlMax", 0, 200); err != nil {
		return o, err
	}
	if get("sockets") != "" {
		n, err := num("sockets", 0, 6)
		if err != nil {
			return o, err
		}
		f.Sockets = &n
	}
	if f.Eth, err = flag("eth"); err != nil {
		return o, err
	}
	if f.Identified, err = flag("identified"); err != nil {
		return o, err
	}
	if f.Pinned, err = flag("pinned"); err != nil {
		return o, err
	}
	o.facets = get("facets") == "1"
	if s := get("q"); len(s) > 64 {
		return o, errors.New("q is longer than 64 characters")
	} else {
		f.Name = s
	}
	if s := get("type"); len(s) > 16 {
		return o, errors.New("bad type")
	} else {
		f.Type = s
	}
	for _, w := range strings.Split(get("quality"), ",") {
		if w == "" {
			continue
		}
		id, ok := qualityID(w)
		if !ok {
			return o, errors.New("bad quality " + w)
		}
		f.Quality = append(f.Quality, id)
	}
	if k := get("kind"); k != "" {
		found := false
		for _, kd := range kinds {
			if kd.ID == k {
				f.CodePrefix = kd.CodePrefix
				if len(kd.Quality) > 0 {
					f.Quality = intersect(f.Quality, kd.Quality)
					if len(f.Quality) == 0 {
						f.Quality = []int{-1} // the kind and the quality chips share no quality: nothing matches
					}
				}
				found = true
			}
		}
		if !found {
			return o, errors.New("bad kind " + k)
		}
	}
	for _, s := range strings.Split(get("state"), ",") {
		switch vault.State(s) {
		case "":
		case vault.Recorded, vault.KeepQ, vault.Kept, vault.SellQ, vault.Sold, vault.Discarded, vault.Trimmed:
			f.States = append(f.States, vault.State(s))
		default:
			return o, errors.New("bad state " + s)
		}
	}
	switch s := get("sort"); s {
	case "", "new":
		f.Order = "new"
	case "old", "value", "ilvl", "quality", "name":
		f.Order = s
	default:
		return o, errors.New("bad sort " + s)
	}
	return o, nil
}

func intersect(a, b []int) []int {
	if len(a) == 0 {
		return b
	}
	var out []int
	for _, x := range a {
		for _, y := range b {
			if x == y {
				out = append(out, x)
			}
		}
	}
	return out
}

// keepAction is the card's Keep button: possible for a live recorded item while the queue accepts presses.
func (d *VaultDesk) keepAction(ctx context.Context, r *vault.Rec) VaultAction {
	switch {
	case d.Keep == nil:
		return VaultAction{Reason: "this bot has no Keep queue"}
	case !d.on():
		return VaultAction{Reason: "the vault is read only while loot.vault is 0"}
	case r.State == vault.KeepQ:
		return VaultAction{Reason: "already queued: arrives at your next town visit"}
	case r.State == vault.Kept:
		return VaultAction{Reason: "already in your stash"}
	case r.State != vault.Recorded:
		return VaultAction{Reason: "the item is " + string(r.State)}
	}
	if ok, _, why := d.Keep.Gate(ctx); !ok {
		return VaultAction{Reason: why}
	}
	return VaultAction{OK: true}
}

// revealAction is the card's Reveal button. Cost and the missing-scroll reason come from the gate and the price view.
func (d *VaultDesk) revealAction(ctx context.Context, r *vault.Rec) VaultAction {
	switch {
	case d.Act == nil:
		return VaultAction{Reason: "this bot has no Reveal"}
	case !d.on():
		return VaultAction{Reason: "the vault is read only while loot.vault is 0"}
	case r.Identified:
		return VaultAction{Reason: "it is identified already"}
	case r.RevQ:
		return VaultAction{Reason: "already queued: arrives at your next town visit"}
	case r.State != vault.Recorded && r.State != vault.KeepQ:
		return VaultAction{Reason: "the item is " + string(r.State)}
	}
	if ok, _, why := d.Act.GateReveal(ctx); !ok {
		return VaultAction{Reason: why}
	}
	if av := d.Act.Act(ctx); !av.Reveal.Free && av.Reveal.Charges == 0 {
		return VaultAction{Reason: "no scroll; buy on request"}
	}
	return VaultAction{OK: true}
}

// sellAction is the card's Sell button: a live recorded item that is not pinned.
func (d *VaultDesk) sellAction(ctx context.Context, r *vault.Rec) VaultAction {
	switch {
	case d.Act == nil:
		return VaultAction{Reason: "this bot has no Sell"}
	case !d.on():
		return VaultAction{Reason: "the vault is read only while loot.vault is 0"}
	}
	inst := false
	if sa, ok := saleActor(d); ok {
		inst, _ = sa.SaleOn(ctx)
	}
	switch {
	case r.State == vault.SellQ && !inst:
		return VaultAction{Reason: "already queued: sold at your next town visit"}
	case r.Pin:
		return VaultAction{Reason: "pinned items are never sold: unpin it first"}
	case r.State != vault.Recorded && !(inst && r.State == vault.SellQ):
		return VaultAction{Reason: "the item is " + string(r.State)}
	case inst:
		return VaultAction{OK: true}
	}
	if ok, _, why := d.Act.GateSell(ctx); !ok {
		return VaultAction{Reason: why}
	}
	return VaultAction{OK: true}
}

func (d *VaultDesk) on() bool { return d.On == nil || d.On() }

func (d *VaultDesk) store() *vault.Store {
	if d.Store == nil {
		return nil
	}
	return d.Store()
}

// upgradeOf is the placeholder of the capacity meter: the next tier's gain and price from the table, never available until the
// Ether wallet exists (V7).
func (d *VaultDesk) upgradeOf() VaultUpgrade {
	u := VaultUpgrade{Currency: "Ether", Reason: "needs the vault switched on (loot.vault) for the Ether wallet"}
	if d.Tiers == nil {
		return u
	}
	u.Max = d.Tiers.Max
	var owned []string
	if d.Owned != nil {
		owned = d.Owned()
	}
	if s, ok := d.Tiers.Next(owned); ok {
		u.Next = &s
	} else {
		u.Reason = "every capacity step is owned"
	}
	return u
}

func (d *VaultDesk) view(ctx context.Context, q vaultQuery, kinds vaultTexts) VaultView {
	v := VaultView{Schema: vaultSchema, Mode: "on", Items: []VaultItem{}, Offset: q.f.Offset, Limit: q.f.Limit, Upgrade: d.upgradeOf()}
	if !d.on() {
		v.Mode = "off"
	}
	if q.facets {
		v.Kinds = kinds.Kinds
	}
	if d.Keep != nil {
		kv := d.Keep.Progress()
		if !d.on() {
			kv.Enabled, kv.Why = false, "the vault is read only while loot.vault is 0"
		}
		v.Keep = &kv
	}
	if d.Equip != nil {
		ok, _, why := d.Equip.Gate(ctx)
		if !d.on() {
			ok, why = false, "the vault is read only while loot.vault is 0"
		}
		v.Equip = &VaultEquipFlag{Enabled: ok, Why: why}
		if de, is := d.Equip.(VaultDirectEquipper); is {
			v.Equip.Direct = de.Direct(ctx)
		}
		if me, is := d.Equip.(VaultMercEquipper); is {
			v.Equip.Merc = me.MercDirect(ctx)
		}
	}
	if d.Act != nil {
		av := d.Act.Act(ctx)
		if !d.on() {
			av.Reveal.Enabled, av.Reveal.Why = false, "the vault is read only while loot.vault is 0"
			av.Sell.Enabled, av.Sell.Why = false, "the vault is read only while loot.vault is 0"
			av.Prompt = nil
		}
		v.Act = &av
	}
	st := d.store()
	if st == nil {
		if d.Tiers != nil {
			v.Capacity = capacityOf(vault.Status{Capacity: d.Tiers.Capacity(nil, 0)})
		}
		return v
	}
	v.Exists, v.Hero = true, st.Hero()
	if v.Act != nil && d.MassFilter != nil && d.Quality != nil && d.on() {
		no := false
		recs, _ := st.Query(vault.Filter{Identified: &no})
		r := v.Act.Reveal
		m := bulkid.MassReveal(d.MassFilter, recs, d.Quality, bulkid.Price{Free: r.Free, Charges: r.Charges, ScrollGold: d.ScrollGold, EtherPer: r.EtherPer, EtherBal: r.Balance})
		v.Act.Reveal.Mass = &m
		v.Act.Reveal.Bar = d.bar(ctx, recs, r)
		if ga, ok := d.Act.(VaultGoldActor); ok {
			gi := ga.GoldInfo(ctx)
			gi.Unrevealed = len(bulkid.UnrevealedIDs(recs, gi.Decode))
			v.Act.Reveal.Gold = &gi
		}
	}
	recs, total := st.Query(q.f)
	v.Total = total
	for i := range recs {
		v.Items = append(v.Items, vaultRow(&recs[i], d.Tip))
	}
	s := st.Status()
	v.Capacity, v.Anomalies = capacityOf(s), s.Anomalies
	v.AutoTrim = d.AutoTrim != nil && d.AutoTrim()
	if m := st.MissedDrops(); m.Count > 0 {
		vm := &VaultMissed{Count: m.Count, Since: m.Since}
		for _, it := range m.Best {
			vm.Best = append(vm.Best, VaultMiss{Name: it.Name, Quality: qualityWord(it.Quality), Q: it.Quality, Ilvl: it.Ilvl})
		}
		v.Missed = vm
	}
	if !v.AutoTrim {
		if marked := st.SellMarked(); len(marked) > 0 {
			vk := &VaultMarked{N: len(marked), IDs: make([]string, 0, len(marked))}
			for i, r := range marked {
				vk.IDs = append(vk.IDs, r.ID)
				if i < 5 {
					vk.Names = append(vk.Names, r.Name)
				}
			}
			v.Marked = vk
		}
	}
	if q.facets {
		for id, n := range st.Facets().Types {
			label := kinds.Types[id]
			if label == "" {
				label = id
			}
			v.Types = append(v.Types, vaultFacet{ID: id, Label: label, N: n})
		}
		sortFacets(v.Types)
	}
	return v
}

func sortFacets(l []vaultFacet) {
	for i := 1; i < len(l); i++ {
		for j := i; j > 0 && (l[j].Label < l[j-1].Label); j-- {
			l[j], l[j-1] = l[j-1], l[j]
		}
	}
}

// VaultDetail is the reply of GET /api/vault/item/<id>: the row plus where it was found, the card's lines and what the
// buttons may do. Keep and Reveal are listed as unavailable with their slice, so the page words them from here.
type VaultDetail struct {
	Schema  string                 `json:"schema"`
	Item    VaultItem              `json:"item"`
	Where   map[string]string      `json:"where,omitempty"`
	Lines   []string               `json:"lines"`
	Actions map[string]VaultAction `json:"actions"`
}

type VaultAction struct {
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
}

func detailOf(r *vault.Rec, tip VaultTipFunc, readOnly bool, keep, reveal, sell VaultAction) VaultDetail {
	it := vaultRow(r, tip)
	var lines []string
	if !it.Identified {
		lines = append(lines, "Unidentified: the properties are hidden until it is revealed")
	}
	lines = append(lines, "Item level "+strconv.Itoa(r.Ilvl))
	if r.Sockets > 0 {
		lines = append(lines, "Sockets: "+strconv.Itoa(r.Sockets))
	}
	if r.Eth {
		lines = append(lines, "Ethereal")
	}
	w := map[string]string{}
	if !r.Found.Time.IsZero() {
		w["time"] = r.Found.Time.UTC().Format(time.RFC3339)
	}
	if r.Found.Area != "" {
		w["area"] = r.Found.Area
	}
	if r.Found.Zone != "" {
		w["zone"] = r.Found.Zone
	}
	if r.Found.MonsterClass != "" {
		w["monster"] = r.Found.MonsterClass
	}
	live := r.State == vault.Recorded
	off := func(ok bool, why string) VaultAction {
		switch {
		case readOnly:
			return VaultAction{Reason: "the vault is read only while loot.vault is 0"}
		case !live:
			return VaultAction{Reason: "the item is " + string(r.State)}
		case !ok:
			return VaultAction{Reason: why}
		}
		return VaultAction{OK: true}
	}
	return VaultDetail{Schema: vaultSchema, Item: it, Where: w, Lines: lines, Actions: map[string]VaultAction{
		"pin":     off(!r.Pin, "already pinned"),
		"unpin":   off(r.Pin, "not pinned"),
		"discard": off(true, ""),
		"keep":    keep,
		"reveal":  reveal,
		"sell":    sell,
	}}
}

// Handler serves the page, its files and the two GET routes.
func (a *VaultAPI) Handler() http.Handler {
	m := http.NewServeMux()
	m.Handle(VaultRoute, gzipJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := a.desk()
		if d == nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "the vault is not available on this bot"})
			return
		}
		td := vaultData()
		q, err := parseVaultQuery(r.URL.Query(), td.Kinds)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, policyReply{Code: "query", Error: err.Error()})
			return
		}
		writeJSON(w, 200, d.view(r.Context(), q, td))
	})))
	m.HandleFunc(VaultItemRoute, func(w http.ResponseWriter, r *http.Request) {
		d := a.desk()
		var st *vault.Store
		if d != nil {
			st = d.store()
		}
		id := strings.TrimPrefix(r.URL.Path, VaultItemRoute)
		if st == nil || id == "" {
			writeJSON(w, http.StatusNotFound, policyReply{Code: "missing", Error: "no such item"})
			return
		}
		rec, ok := st.Get(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, policyReply{Code: "missing", Error: "no such item"})
			return
		}
		writeJSON(w, 200, detailOf(&rec, d.Tip, !d.on(), d.keepAction(r.Context(), &rec), d.revealAction(r.Context(), &rec), d.sellAction(r.Context(), &rec)))
	})
	m.HandleFunc(VaultEquipGet, func(w http.ResponseWriter, r *http.Request) {
		d := a.desk()
		if d == nil || d.Equip == nil {
			writeJSON(w, http.StatusConflict, policyReply{Code: "off", Error: "this bot cannot equip from the vault"})
			return
		}
		v, ok := d.Equip.Preview(strings.TrimPrefix(r.URL.Path, VaultEquipGet))
		if !ok {
			writeJSON(w, http.StatusNotFound, policyReply{Code: "missing", Error: "no such item"})
			return
		}
		writeJSON(w, 200, v)
	})
	static := func(path, file, ctype string) {
		m.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			b, err := webFS.ReadFile("web/" + file)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			serveAsset(w, r, file, b, func(w http.ResponseWriter) { pactHdr(w, ctype) })
		})
	}
	static("/watch/vault", "vault.html", "text/html; charset=utf-8")
	static("/watch/vault.js", "vault.js", "text/javascript; charset=utf-8")
	static("/watch/vault-logic.js", "vault-logic.js", "text/javascript; charset=utf-8")
	static("/watch/vault.css", "vault.css", "text/css; charset=utf-8")
	static("/watch/vault.json", "vault.json", "application/json; charset=utf-8")
	return readOnly(m)
}

type vaultBody struct {
	IDs     []string `json:"ids"`
	Scope   string   `json:"scope,omitempty"`   // revealgold: all_unrevealed names every unrevealed record of the whole vault instead of ids; sell and sellquote: all_sellable
	OrderID string   `json:"orderId,omitempty"` // sell, all_sellable: the page's idempotent id of the order (a resend never sells twice)
	Confirm bool     `json:"confirm"`
	Target  string   `json:"target,omitempty"`  // keep: stash (default) or inv; equip: "merc" gives the item to the mercenary
	MaxGold int      `json:"maxGold,omitempty"` // revealgold: the most gold the order may spend (0: the quote)
	Ether   bool     `json:"ether,omitempty"`   // reveal: pay with Ether (an explicit tap; identify.etherPerReveal)
	Now     bool     `json:"now,omitempty"`     // keep, reveal, sell: also ask the bot to walk home for it
	Dismiss bool     `json:"dismiss,omitempty"` // sellprompt: mute the prompt
	Ack     bool     `json:"ack,omitempty"`     // sell: the player saw the gold and the stash after (needed from loot.vaultSellConfirm gold)
	Slot    string   `json:"slot,omitempty"`    // equip: auto (default), left or right (rings)
}

type vaultFail struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Error string `json:"error"`
}

type vaultReply struct {
	OK      bool        `json:"ok"`
	Done    []string    `json:"done"`
	Already []string    `json:"already,omitempty"` // keep: pressed before, nothing changed
	Failed  []vaultFail `json:"failed,omitempty"`
	Sale    *VaultSale  `json:"sale,omitempty"` // sell: the instant sale that ran (gold added to the stash, stash after)
	// Equipped lists, per piece, what an Equip or Wear set did at once (Equip v2): the slot, what came off, where it went.
	Equipped []vaultWorn `json:"equipped,omitempty"`
}

type vaultWorn struct {
	ID      string   `json:"id"`
	Slot    string   `json:"slot"`
	Removed []string `json:"removed,omitempty"`
	Where   string   `json:"where,omitempty"`
}

func (r *vaultReply) worn(res VaultKeepResult) {
	if res.OK && res.Direct {
		r.Equipped = append(r.Equipped, vaultWorn{ID: res.ID, Slot: res.Slot, Removed: res.Removed, Where: res.Where})
	}
}

func vaultCode(err error) string {
	switch {
	case errors.Is(err, vault.ErrNotFound):
		return "missing"
	case errors.Is(err, vault.ErrState):
		return "state"
	case errors.Is(err, vault.ErrPinCap):
		return "pincap"
	}
	return "internal"
}

// PostHandler serves pin, unpin and discard for one or more ids. Discard needs "confirm": true (the page asks first); the
// store archives the record before it leaves (docs/vault.md). A partly refused batch answers 200 with the failures listed;
// a batch where nothing worked answers 409.
func (a *VaultAPI) PostHandler(o AnswerOptions, route string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b vaultBody
		if !policyGate(o, w, r, &b) {
			return
		}
		d := a.desk()
		var st *vault.Store
		if d != nil && d.on() {
			st = d.store()
		}
		if st == nil {
			writeJSON(w, http.StatusConflict, policyReply{Code: "off", Error: "the vault is read only (loot.vault is 0 or there is no vault yet)"})
			return
		}
		if route == VaultRevealGoldCancelRoute {
			a.goldRoute(w, r, d, route, &b)
			return
		}
		if route == VaultMissedRoute {
			if !b.Dismiss {
				writeJSON(w, http.StatusBadRequest, policyReply{Code: "dismiss", Error: "send dismiss: true"})
				return
			}
			if err := st.ClearMissed(); err != nil {
				writeJSON(w, http.StatusInternalServerError, policyReply{Code: "internal", Error: err.Error()})
				return
			}
			writeJSON(w, 200, vaultReply{OK: true, Done: []string{}})
			return
		}
		if route == VaultPromptRoute {
			if d.Act == nil || !b.Dismiss {
				writeJSON(w, http.StatusBadRequest, policyReply{Code: "dismiss", Error: "send dismiss: true"})
				return
			}
			d.Act.DismissOffer()
			writeJSON(w, 200, vaultReply{OK: true, Done: []string{}})
			return
		}
		if route == VaultRevealGoldRoute && b.Scope != "" {
			if b.Scope != bulkid.ScopeAllUnrevealed || len(b.IDs) > 0 {
				writeJSON(w, http.StatusBadRequest, policyReply{Code: "scope", Error: "scope is all_unrevealed, without ids"})
				return
			}
			a.goldRoute(w, r, d, route, &b)
			return
		}
		if b.Scope != "" && (route == VaultSellRoute || route == VaultSellQuote) {
			a.sellAll(w, r, d, route, &b)
			return
		}
		if len(b.IDs) == 0 || len(b.IDs) > vaultMaxIDs {
			writeJSON(w, http.StatusBadRequest, policyReply{Code: "ids", Error: "send 1 to " + strconv.Itoa(vaultMaxIDs) + " ids"})
			return
		}
		if route == VaultRevealGoldRoute {
			a.goldRoute(w, r, d, route, &b)
			return
		}
		if route == VaultSellQuote {
			if sa, ok := saleActor(d); ok {
				writeJSON(w, 200, sa.SaleQuote(r.Context(), b.IDs))
			} else {
				writeJSON(w, http.StatusConflict, policyReply{Code: "off", Error: "this bot has no instant sale"})
			}
			return
		}
		if route == VaultDiscardRoute && !b.Confirm {
			writeJSON(w, http.StatusBadRequest, policyReply{Code: "confirm", Error: "discard needs confirm: true"})
			return
		}
		if route == VaultEquipRoute || route == VaultEquipCancel {
			a.equip(w, r, d, route, &b)
			return
		}
		if route == VaultKeepRoute {
			a.keep(w, r, d, &b)
			return
		}
		if route == VaultSellRoute {
			if sa, ok := saleActor(d); ok {
				if on, _ := sa.SaleOn(r.Context()); on {
					a.sellNow(w, r, d, sa, &b)
					return
				}
			}
		}
		if route == VaultRevealRoute || route == VaultSellRoute || route == VaultCancelRoute {
			a.act(w, r, d, route, &b)
			return
		}
		rep := vaultReply{Done: []string{}}
		for _, id := range b.IDs {
			var err error
			switch route {
			case VaultPinRoute:
				err = st.Pin(id)
			case VaultUnpinRoute:
				err = st.Unpin(id)
			default:
				err = st.Discard(id, "player, vault page")
			}
			if err != nil {
				rep.Failed = append(rep.Failed, vaultFail{ID: id, Code: vaultCode(err), Error: err.Error()})
				continue
			}
			rep.Done = append(rep.Done, id)
		}
		rep.OK = len(rep.Failed) == 0
		code := 200
		if len(rep.Done) == 0 {
			code = http.StatusConflict
		}
		writeJSON(w, code, rep)
	})
}

// sellAll is the whole-vault sale (scope all_sellable, no ids): the bot lists the records when the order runs, the page only
// carries its own order id. The quote route answers what the sale would do. Without the instant sale the same list goes to
// the old town-visit queue, except in a measured run, where nothing is sold.
func (a *VaultAPI) sellAll(w http.ResponseWriter, r *http.Request, d *VaultDesk, route string, b *vaultBody) {
	sa, ok := saleActor(d)
	var all VaultSellAllActor
	if ok {
		all, ok = sa.(VaultSellAllActor)
	}
	if b.Scope != "all_sellable" || len(b.IDs) > 0 || !ok {
		writeJSON(w, http.StatusBadRequest, policyReply{Code: "scope", Error: "scope is all_sellable, without ids (and this bot must have Sell all)"})
		return
	}
	on, why := sa.SaleOn(r.Context())
	if route == VaultSellQuote {
		q := all.SellAllQuote(r.Context())
		if !on && q.Why == "" {
			q.Why = why
		}
		writeJSON(w, 200, q)
		return
	}
	if !b.Confirm {
		writeJSON(w, http.StatusBadRequest, policyReply{Code: "confirm", Error: "sell needs confirm: true"})
		return
	}
	if on {
		code, msg, sale := all.SellAll(r.Context(), b.OrderID)
		rep := vaultReply{Done: []string{}, Sale: sale}
		if code != "" {
			writeJSON(w, http.StatusConflict, policyReply{Code: code, Error: msg})
			return
		}
		rep.OK = sale != nil && sale.State == "done" && sale.Done > 0
		status := 200
		if sale == nil || (sale.State == "failed") {
			status = http.StatusConflict
		}
		writeJSON(w, status, rep)
		return
	}
	if strings.Contains(why, "measured run") {
		writeJSON(w, http.StatusConflict, policyReply{Code: "measured", Error: why})
		return
	}
	if d.Act == nil {
		writeJSON(w, http.StatusConflict, policyReply{Code: "off", Error: "this bot has no Sell"})
		return
	}
	if ok, code, why := d.Act.GateSell(r.Context()); !ok {
		writeJSON(w, http.StatusConflict, policyReply{Code: code, Error: why})
		return
	}
	ids := all.SellAllIDs()
	if len(ids) == 0 {
		writeJSON(w, http.StatusConflict, policyReply{Code: "state", Error: "nothing in the vault can be queued for sale now"})
		return
	}
	rep := vaultReply{Done: []string{}}
	for _, x := range d.Act.EnqueueSell(r.Context(), ids, false) {
		if x.OK {
			rep.Done = append(rep.Done, x.ID)
		} else {
			rep.Failed = append(rep.Failed, vaultFail{ID: x.ID, Code: x.Code, Error: x.Why})
		}
	}
	rep.OK = len(rep.Failed) == 0
	status := 200
	if len(rep.Done) == 0 {
		status = http.StatusConflict
	}
	writeJSON(w, status, rep)
}

func saleActor(d *VaultDesk) (VaultSaleActor, bool) {
	if d.Act == nil || !d.on() {
		return nil, false
	}
	sa, ok := d.Act.(VaultSaleActor)
	return sa, ok
}

// sellNow is the instant sale: the records leave and the stash is credited before the reply. Needs confirm (an order), and ack
// from loot.vaultSellConfirm gold (the page showed the gold and the stash after).
func (a *VaultAPI) sellNow(w http.ResponseWriter, r *http.Request, d *VaultDesk, sa VaultSaleActor, b *vaultBody) {
	if !b.Confirm {
		writeJSON(w, http.StatusBadRequest, policyReply{Code: "confirm", Error: "sell needs confirm: true"})
		return
	}
	res, sale := sa.SellNow(r.Context(), b.IDs, b.Ack)
	rep := vaultReply{Done: []string{}, Sale: sale}
	for _, x := range res {
		if x.OK {
			rep.Done = append(rep.Done, x.ID)
		} else {
			rep.Failed = append(rep.Failed, vaultFail{ID: x.ID, Code: x.Code, Error: x.Why})
		}
	}
	rep.OK = len(rep.Failed) == 0
	code := 200
	if len(rep.Done) == 0 {
		code = http.StatusConflict
	}
	writeJSON(w, code, rep)
}

// act queues Reveal or Sell, or cancels queued requests of any kind. Same shape as keep: the gate first, then the real reason
// of every refusal; the page reads the state back from GET. Sell needs confirm (it turns an item into gold).
func (a *VaultAPI) act(w http.ResponseWriter, r *http.Request, d *VaultDesk, route string, b *vaultBody) {
	if d.Act == nil {
		writeJSON(w, http.StatusConflict, policyReply{Code: "off", Error: "this bot has no Reveal or Sell"})
		return
	}
	var res []VaultKeepResult
	switch route {
	case VaultRevealRoute:
		// An Ether reveal is an unlock of data the vault holds: it needs no game and no gate.
		if ea, ok := d.Act.(VaultEtherActor); ok && b.Ether {
			res = ea.EnqueueRevealEther(r.Context(), b.IDs, b.Now)
		} else {
			if ok, code, why := d.Act.GateReveal(r.Context()); !ok {
				writeJSON(w, http.StatusConflict, policyReply{Code: code, Error: why})
				return
			}
			res = d.Act.EnqueueReveal(r.Context(), b.IDs, b.Now)
		}
	case VaultSellRoute:
		if !b.Confirm {
			writeJSON(w, http.StatusBadRequest, policyReply{Code: "confirm", Error: "sell needs confirm: true"})
			return
		}
		if ok, code, why := d.Act.GateSell(r.Context()); !ok {
			writeJSON(w, http.StatusConflict, policyReply{Code: code, Error: why})
			return
		}
		res = d.Act.EnqueueSell(r.Context(), b.IDs, b.Now)
	default:
		res = d.Act.Cancel(b.IDs)
	}
	rep := vaultReply{Done: []string{}}
	for _, x := range res {
		switch {
		case x.OK && x.Already:
			rep.Already = append(rep.Already, x.ID)
			rep.Done = append(rep.Done, x.ID)
		case x.OK:
			rep.Done = append(rep.Done, x.ID)
		default:
			rep.Failed = append(rep.Failed, vaultFail{ID: x.ID, Code: x.Code, Error: x.Why})
		}
	}
	rep.OK = len(rep.Failed) == 0
	code := 200
	if len(rep.Done) == 0 {
		code = http.StatusConflict
	}
	writeJSON(w, code, rep)
}

// keep queues the ids (idempotent: a second press answers "already"), after the gate. The reply names the real reason of every
// failure; the page reads the state back from GET, so a toast never says more than what the store holds.
func (a *VaultAPI) keep(w http.ResponseWriter, r *http.Request, d *VaultDesk, b *vaultBody) {
	if d.Keep == nil {
		writeJSON(w, http.StatusConflict, policyReply{Code: "off", Error: "this bot has no Keep queue"})
		return
	}
	if b.Target != "" && b.Target != vault.WhereStash && b.Target != vault.WhereInv {
		writeJSON(w, http.StatusBadRequest, policyReply{Code: "target", Error: "target is stash or inv"})
		return
	}
	if ok, code, why := d.Keep.Gate(r.Context()); !ok {
		writeJSON(w, http.StatusConflict, policyReply{Code: code, Error: why})
		return
	}
	rep := vaultReply{Done: []string{}}
	for _, res := range d.Keep.Enqueue(r.Context(), b.IDs, b.Target, b.Now) {
		switch {
		case res.OK && res.Already:
			rep.Already = append(rep.Already, res.ID)
			rep.Done = append(rep.Done, res.ID)
		case res.OK:
			rep.Done = append(rep.Done, res.ID)
		default:
			rep.Failed = append(rep.Failed, vaultFail{ID: res.ID, Code: res.Code, Error: res.Why})
		}
	}
	rep.OK = len(rep.Failed) == 0
	code := 200
	if len(rep.Done) == 0 {
		code = http.StatusConflict
	}
	writeJSON(w, code, rep)
}

// equip takes ONE order (the sheet sends the id and the slot) or cancels it. The reply has the same shape as keep; the page
// reads the order back from GET /api/vault/equip/<id>, so a toast never says more than what the bot holds.
func (a *VaultAPI) equip(w http.ResponseWriter, r *http.Request, d *VaultDesk, route string, b *vaultBody) {
	if d.Equip == nil {
		writeJSON(w, http.StatusConflict, policyReply{Code: "off", Error: "this bot cannot equip from the vault"})
		return
	}
	if len(b.IDs) == 0 || len(b.IDs) > 1 && route == VaultEquipCancel || len(b.IDs) > maxEquipSet {
		writeJSON(w, http.StatusBadRequest, policyReply{Code: "ids", Error: "send one id, or up to 12 for a set"})
		return
	}
	if len(b.IDs) > 1 {
		a.equipSet(w, r, d, b)
		return
	}
	var res VaultKeepResult
	if route == VaultEquipCancel {
		res = d.Equip.Cancel(b.IDs[0])
	} else if b.Target == "merc" {
		me, ok := d.Equip.(VaultMercEquipper)
		if !ok || !me.MercDirect(r.Context()) {
			writeJSON(w, http.StatusConflict, policyReply{Code: "off", Error: "equipping the mercenary from the Vault needs the instant equip and the fork's mercenary verbs"})
			return
		}
		if ok, code, why := d.Equip.Gate(r.Context()); !ok {
			writeJSON(w, http.StatusConflict, policyReply{Code: code, Error: why})
			return
		}
		res = me.EquipMerc(r.Context(), b.IDs[0])
	} else {
		if b.Slot != "" && b.Slot != "auto" && b.Slot != "left" && b.Slot != "right" {
			writeJSON(w, http.StatusBadRequest, policyReply{Code: "slot", Error: "slot is auto, left or right"})
			return
		}
		if ok, code, why := d.Equip.Gate(r.Context()); !ok {
			writeJSON(w, http.StatusConflict, policyReply{Code: code, Error: why})
			return
		}
		res = d.Equip.Enqueue(r.Context(), b.IDs[0], b.Slot, b.Now)
	}
	rep := vaultReply{Done: []string{}}
	switch {
	case res.OK && res.Already:
		rep.Already = append(rep.Already, res.ID)
		rep.Done = append(rep.Done, res.ID)
	case res.OK:
		rep.Done = append(rep.Done, res.ID)
		rep.worn(res)
	default:
		rep.Failed = append(rep.Failed, vaultFail{ID: res.ID, Code: res.Code, Error: res.Why})
	}
	rep.OK = len(rep.Failed) == 0
	code := 200
	if len(rep.Done) == 0 {
		code = http.StatusConflict
	}
	writeJSON(w, code, rep)
}

// maxEquipSet is the most pieces one Wear set takes (a full body: ten slots, two spare weapons).
const maxEquipSet = 12

// equipSet is "Wear set": every id is ordered in one request; the reply lists each piece as done or failed with its own
// reason, so the page can say which pieces will be worn and which stay in the vault.
func (a *VaultAPI) equipSet(w http.ResponseWriter, r *http.Request, d *VaultDesk, b *vaultBody) {
	if ok, code, why := d.Equip.Gate(r.Context()); !ok {
		writeJSON(w, http.StatusConflict, policyReply{Code: code, Error: why})
		return
	}
	rep := vaultReply{Done: []string{}}
	for _, res := range d.Equip.EnqueueSet(r.Context(), b.IDs, b.Now) {
		switch {
		case res.OK && res.Already:
			rep.Already = append(rep.Already, res.ID)
			rep.Done = append(rep.Done, res.ID)
		case res.OK:
			rep.Done = append(rep.Done, res.ID)
			rep.worn(res)
		default:
			rep.Failed = append(rep.Failed, vaultFail{ID: res.ID, Code: res.Code, Error: res.Why})
		}
	}
	rep.OK = len(rep.Failed) == 0
	code := 200
	if len(rep.Done) == 0 {
		code = http.StatusConflict
	}
	writeJSON(w, code, rep)
}
