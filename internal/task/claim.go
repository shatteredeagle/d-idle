package task

import (
	"context"
	"errors"
	"time"
)

// QBurialMerc is the follow-up of Sisters' Burial Grounds: the free Rogue Kashya offers once Blood Raven is dead (plan-v2
// 10.1: the user decided to take it). It has no quest word of its own, so the campaign entry is complete when its step has
// nothing left to do.
const QBurialMerc = "sisters_burial_merc"

// ErrClaimUnsupported is what a MercClaimer returns when it cannot claim the Rogue (no verified hire dialog yet). The step
// treats it like the missing capability: skipped, nothing blocked.
var ErrClaimUnsupported = errors.New("merc claim unsupported")

// MercClaimer is the optional capability behind the claim step: talk to Kashya in the Rogue Encampment and take the free
// Rogue. The merc backend (agent/mercpolicy) implements it later; nothing in the task layer depends on that package.
type MercClaimer interface {
	ClaimFreeMerc(ctx context.Context) error
}

func init() { handlers[QBurialMerc] = claimNodes }

// ClaimNeeded is the precondition of the claim step: Blood Raven is dead (the quest word is at least Cleared; the reward
// talk makes it Done), the character is in Act 1 and has no living merc. An unknown merc state (no reader) proceeds.
func ClaimNeeded(w World) bool {
	if w.Quest(QBurial).Stage < StageCleared || w.Act > 1 || !normalDiff(w) {
		return false
	}
	return !w.HasMerc && !(w.MercKnown && w.MercAlive)
}

// KashyaHireAllowed is the soft rule that no gold goes to a Kashya hire before Blood Raven is done: afterwards the
// claim step takes the Rogue for free, so a paid Act 1 hire is never needed and the rule only reads the quest word.
func KashyaHireAllowed(w World) bool { return w.Quest(QBurial).Stage >= StageCleared }

// WithRogueClaim inserts the claim entry right after Sisters' Burial Grounds (a no-op without that entry or when the
// entry exists). The plain templates stay as they were: the bot opts in (internal/bot/runner.go), because the claim is a
// plan-v2 decision and not part of the quest list.
func (c *Campaign) WithRogueClaim() *Campaign {
	if c == nil {
		return nil
	}
	for _, e := range c.Entries {
		if e.Quest == QBurialMerc {
			return c
		}
	}
	for i, e := range c.Entries {
		if e.Quest == QBurial {
			claim := Entry{Quest: QBurialMerc, Title: "Claim the free Rogue", MinLevel: 1, Act: 1}
			c.Entries = append(c.Entries[:i+1], append([]Entry{claim}, c.Entries[i+1:]...)...)
			return c
		}
	}
	return c
}

func claimNodes() []Node {
	q := QBurialMerc
	claim := Step{ID: id(q, "claim"), Title: "Claim the free Rogue", Phase: "turn-in", Budget: 4 * time.Minute, Est: time.Minute,
		Optional: true, // a step that cannot run must never block the campaign
		Need:     func(w World) (bool, Evidence) { return ClaimNeeded(w), EvProxy },
		Ready: func(w World) *Fault {
			if !w.InTown {
				return &Fault{Kind: KindPrereq, Why: "Kashya is in town", Insert: []Step{StepGoHome(id(q, "claim/home"))}}
			}
			return nil
		},
		Do: func(ctx context.Context, x Actions) error {
			c, ok := x.(MercClaimer)
			if !ok {
				return nil
			}
			if err := c.ClaimFreeMerc(ctx); err != nil && !errors.Is(err, ErrClaimUnsupported) {
				return err
			}
			return nil
		}}
	return []Node{{Step: claim}}
}
