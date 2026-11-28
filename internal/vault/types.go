// Package vault is the offline store of the Ethereal Vault (docs/design/loot-vault-2026-11-16.md, slice V2):
// an append-only event log per hero, content-hash dedupe, a state machine, data-driven capacity and trim,
// an in-memory index rebuilt by replay, an archive for everything that leaves the vault, and export/import.
// Nothing here talks to the game; the bot wiring is slice V3.
package vault

import (
	"errors"
	"sort"
	"time"
)

// State of a record. Pin and identified are flags on the record (Rec.Pin, Item.Identified), not states:
// the design's "pinned" and "revealed" are views of those flags (Rec.Status).
type State string

const (
	Recorded  State = "recorded"
	KeepQ     State = "keepq"     // Keep clicked, waiting for a town visit
	Kept      State = "kept"      // created in the game, final
	SellQ     State = "sellq"     // marked for the sell errand
	Sold      State = "sold"      // final
	Discarded State = "discarded" // final, archived
	Trimmed   State = "trimmed"   // final, archived
)

// Stages of a reveal or sale errand (slice V6). Each is logged BEFORE the game step it names, so a restart can tell what
// may exist in the hero.
const (
	StageRevBegin   = "rb" // reveal: the temporary copy is about to be created (it may or may not exist)
	StageRevMade    = "rc" // reveal: the copy exists in the hero and is not yet identified in the vault
	StageRevCleanup = "rt" // reveal: the identified bytes are stored; the temporary copy still has to leave the hero
	StageSellBegin  = "sb" // sale: the copy is about to be created in the pack
	StageSellMade   = "sc" // sale: the copy exists; the sale is under way
)

// Keep targets.
const (
	WhereStash = "stash"
	WhereInv   = "inv"
)

// Live states count against capacity.
func (s State) Live() bool { return s == Recorded || s == KeepQ || s == SellQ }

// Found says where a drop was seen.
type Found struct {
	Time         time.Time `json:"t"`
	Zone         string    `json:"zone,omitempty"`
	Area         string    `json:"area,omitempty"`
	MonsterClass string    `json:"monster,omitempty"`
}

// Item is what the caller records: the item bytes plus metadata. Raw is the only truth; the rest is derived by
// the caller (item codec, V1) and stored for search and display. d2d items have uid 0 (V0 finding), so the
// content hash of Raw is the identity.
type Item struct {
	ID         string   `json:"id"`
	Hero       string   `json:"hero"`
	Raw        string   `json:"raw"` // hex of the item bytes
	Code       string   `json:"code"`
	Name       string   `json:"name,omitempty"`
	Quality    int      `json:"q"`
	Ilvl       int      `json:"ilvl"`
	Identified bool     `json:"identified"`
	Chain      []string `json:"chain,omitempty"` // base type chain, most specific first, e.g. swor, weap
	Sockets    int      `json:"sockets,omitempty"`
	Eth        bool     `json:"eth,omitempty"`
	Props      string   `json:"props,omitempty"` // "stat:param:value;..." as d2d sent them, kept ONLY for an identified item (the page words them as the stat lines; an unidentified record never carries them)
	// Sealed holds the affixes of an UNIDENTIFIED record exactly as Props would carry them. The server keeps them and
	// serves them to no one until the reveal is paid (Store.Unseal); exports drop them. Empty: nothing held (the fork did
	// not send affixes for the unidentified item, U), so only the in-game identify can reveal it.
	Sealed string `json:"sealed,omitempty"`
	// Unlocked: the record was revealed by Unseal. Raw is still the unidentified bytes the game dropped, so a Keep
	// creates the unidentified item in the game (the fork's item create takes no identified flag, V code read).
	Unlocked  bool   `json:"unlocked,omitempty"`
	Pic       int    `json:"pic,omitempty"`  // picture variant + 1 (jewel, ring, amulet, charm), 0 = base picture
	Tint      string `json:"tint,omitempty"` // icon colour key "t<set>c<colour>"
	FileIndex int    `json:"fidx,omitempty"` // the unique or set id, for the unique and set icon
	Found     Found  `json:"found"`
	Value     int    `json:"value"`
}

// Rec is a record as the index holds it.
type Rec struct {
	Item
	Hash     string `json:"hash"`
	State    State  `json:"state"`
	Pin      bool   `json:"pin,omitempty"`
	InFlight bool   `json:"inflight,omitempty"` // a keep create was started and not confirmed
	Attempts int    `json:"attempts,omitempty"`
	Where    string `json:"where,omitempty"`   // keep target: "" or stash, inv
	Reason   string `json:"reason,omitempty"`  // last failure or trim/discard reason
	Gold     int    `json:"gold,omitempty"`    // sale price once sold
	RevQ     bool   `json:"revq,omitempty"`    // Reveal asked: the item is identified at the next town visit (V6)
	Stage    string `json:"stage,omitempty"`   // in-flight step of a reveal or sale (Stage* constants), "" when idle
	Ordered  bool   `json:"ordered,omitempty"` // sellq: the player ordered the sale; false is a mark that waits for a confirm (loot.vaultAutoTrim off)
	GoldPre  int    `json:"goldpre,omitempty"` // purse before the sale create, for the read-back after a restart
	// Hold is the order id that reserved the record (Store.Reserve): until that order sells it (soldnow) or releases it,
	// every other transition (keep, pin, discard, queue sale, reveal) is refused, so a record is either sold or kept.
	Hold    string    `json:"hold,omitempty"`
	Updated time.Time `json:"updated"`
	seq     int       // position in insertion order
}

// Status is the one-word view the UI shows.
func (r *Rec) Status() string {
	switch {
	case r.State == Recorded && r.Pin:
		return "pinned"
	case r.State == Recorded && r.Identified:
		return "revealed"
	}
	return string(r.State)
}

var (
	ErrNothingSealed = errors.New("vault: no affixes are held for this item")
	ErrDuplicate     = errors.New("vault: identical item already in the vault")
	ErrOverCapacity  = errors.New("vault: over capacity, new records are blocked")
	ErrFull          = errors.New("vault: full: new drops are not recorded until room is made")
	ErrNotFound      = errors.New("vault: no such record")
	ErrState         = errors.New("vault: transition not allowed from this state")
	ErrPinCap        = errors.New("vault: pin limit reached")
	ErrInvalid       = errors.New("vault: invalid item")
	ErrTooLarge      = errors.New("vault: import too large")
	ErrPinned        = errors.New("vault: pinned items are never sold")
	ErrHeld          = errors.New("vault: reserved by an order")
	ErrStale         = errors.New("vault: changed since it was read")
)

// MissedKeep is how many of the best missed drops the vault remembers.
const MissedKeep = 10

// Missed counts the drops that were not recorded because the vault was full, and keeps the best few (highest value)
// so the player sees what they missed. It persists with the vault until the player clears it.
type Missed struct {
	Count int    `json:"count"`
	Since string `json:"since,omitempty"` // RFC3339 time of the first miss since the last clear
	Best  []Item `json:"best,omitempty"`
}

func (m *Missed) add(it Item, t time.Time) {
	if m.Count == 0 {
		m.Since = t.UTC().Format(time.RFC3339)
	}
	m.Count++
	m.Best = append(m.Best, it)
	sort.SliceStable(m.Best, func(i, j int) bool { return m.Best[i].Value > m.Best[j].Value })
	if len(m.Best) > MissedKeep {
		m.Best = m.Best[:MissedKeep]
	}
}
