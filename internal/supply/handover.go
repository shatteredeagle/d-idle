package supply

import (
	"fmt"
	"strings"
)

// Source is one place the gold can still come from when the plan is short (town-planner section 5.4), in the order the
// step hands over to them.
type Source string

const (
	SrcSell        Source = "sell"         // the sell chore of the visit
	SrcStash       Source = "stash"        // gold in the stash the plan's budget did not count
	SrcVaultPrompt Source = "vault_prompt" // the opt-in "sell these vault items?" Needs-you row (never sells by itself)
	SrcRecovery    Source = "recovery"     // recovery or the player's gold farm: always the last one
)

// Gold is what the caller knows about the gold sources; the package reads none of it itself.
type Gold struct {
	Sellable    bool // the pack holds something the sell policy would sell and the plan's sale estimate did not count
	StashGold   int  // stash gold NOT already in the plan's budget (0 when the stash was read: Wealth counted it)
	VaultPrompt bool // loot.vault and loot.vaultSellPrompt are on
}

// Hand is the hand-over of a short-gold plan: the sources in order and the sentence that names the amount and the line.
type Hand struct {
	Sources []Source
	Reason  string
}

// Handover answers what a step does when the plan is short of gold, instead of ending as needs_attention. It is empty
// for any other verdict. The last source is always recovery, so a short-gold plan never has no way forward.
func (p Plan) Handover(g Gold) Hand {
	if p.Verdict != ShortGold {
		return Hand{}
	}
	var src []Source
	if g.Sellable {
		src = append(src, SrcSell)
	}
	if g.StashGold > 0 {
		src = append(src, SrcStash)
	}
	if g.VaultPrompt {
		src = append(src, SrcVaultPrompt)
	}
	src = append(src, SrcRecovery)
	names := make([]string, len(src))
	for i, s := range src {
		names[i] = string(s)
	}
	return Hand{Sources: src, Reason: fmt.Sprintf("needs %d more gold for %s: %s", p.Short, p.Open, strings.Join(names, ", then "))}
}
