package bot

import (
	"strings"
	"sync"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/resource"
)

// Gear advice (docs/gear-policy.md): with gear.decisions = player the bot never equips, swaps or buys gear. It still
// reads the pack and tells the player what it would have worn, as text only. Nothing here clicks or writes.

// GearHint is one pure-read suggestion: wear pack item ID in Slot (a short slot name).
type GearHint struct {
	ID   uint32
	Text string // "Hand Axe: better than your Short Sword in the weapon slot"
}

// GearAdvisor is the optional read side of Gear (the leveler adapter implements it).
type GearAdvisor interface {
	GearHints(snap *d2.Snapshot) []GearHint
}

// Onboarding is the one line shown while the player has not touched gear (a new hero wears its class starter kit).
const Onboarding = "Your hero wears its starting gear. The bot never changes equipment: equip upgrades yourself on the Inventory page, it only suggests."

// NoWeaponPrompt is shown when a class that needs a weapon has none equipped (for example the player took it off).
const NoWeaponPrompt = "No weapon equipped. Equip one on the Inventory page; the bot will not do it for you."

// gearAdvice is the cache the inventory page reads. It is filled from the runner loop and read from the desk, so the
// desk never calls back into the leveler under its own lock.
type gearAdvice struct {
	mu     sync.Mutex
	byID   map[uint32]string
	prompt string
	last   string // the logged form, to log changes only
}

// adviseGear refreshes the advice from snap. Safe to call every tick; it logs when the prompt or hints change.
func (r *Runner) adviseGear(snap *d2.Snapshot) {
	if snap == nil || snap.Player == nil || snap.Inventory == nil {
		return
	}
	hints := map[uint32]string{}
	var lines []string
	if ga, ok := r.d.Gear.(GearAdvisor); ok {
		for _, h := range ga.GearHints(snap) {
			hints[h.ID] = h.Text
			lines = append(lines, h.Text)
		}
	}
	prompt := Onboarding
	if resource.NeedsWeapon(snap.Player.Class) && snap.Inventory.Equipped(d2.BodyRHand) == nil && snap.Inventory.Equipped(d2.BodyAltR) == nil {
		prompt = NoWeaponPrompt
	}
	key := prompt + "|" + strings.Join(lines, "|")
	g := &r.gearAdv
	g.mu.Lock()
	g.byID, g.prompt = hints, prompt
	changed := key != g.last
	g.last = key
	g.mu.Unlock()
	if changed {
		r.e.Log.Info("gear advice", "prompt", prompt, "hints", lines)
	}
}

// GearAdviceFor is the invdesk.Config.Advice text for a pack item ("" when none).
func (r *Runner) GearAdviceFor(id uint32) string {
	g := &r.gearAdv
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.byID[id]
}

// GearPrompt is the one-line onboarding or no-weapon prompt for the inventory page ("" before the first look).
func (r *Runner) GearPrompt() string {
	g := &r.gearAdv
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.prompt
}
