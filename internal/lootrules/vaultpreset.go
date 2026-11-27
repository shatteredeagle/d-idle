package lootrules

// PresetVault is the id of the Vault preset (docs/vault.md, docs/lootrules.md "Vault action").
const PresetVault = "vault"

// vaultKeepsOwnRule lists Starter ground rules that stay pickups under the Vault preset: gems keep their own rules
// (the cube upgrade needs them in the pack), and so does everything the kit needs (potions, scrolls, keys, tomes, gold).
// Unidentified magic and rare jewelry and charms are NOT in that list: they go to the vault
// unidentified like any other magic item (the gold reveal opens them there); keeping them as pickups sent every ring
// and amulet through the pack to the sell or stash chore and none ever reached the vault (docs/probes/vault-rings-2026-11-24.md).
var vaultKeepsOwnRule = map[string]bool{"g-top-gems": true, "g-mid-gems": true, "g-cube-gems": true}

// vaultUnidJewelry are the Starter's ground `keep` rules for unidentified jewelry and charms.
var vaultUnidJewelry = map[string]bool{"g-jewelry-id": true, "g-small-charm-id": true, "g-big-charm-id": true}

// VaultPreset is the Starter with the ground-stage stash rules turned into `vault` (peek and record, no pickup):
// runes, charms, jewels, jewelry, uniques, sets, runeword bases and the magic, rare and crafted gear that the Starter
// picks up only to sell. Plain, low and superior gear is left on the floor (today's drop-junk behaviour), chipped
// and flawed gems too. The poor-mode junk-for-sale rule stays a pickup so the early gold line is unchanged. With
// recordAll every other non-gold item that is not a gem or a stackable is recorded too, instead of left. The pack
// and sell rules are the Starter's, untouched. Pure data: it never reaches the engine unless loot.vault is 1.
func VaultPreset(recordAll bool) *RuleSet {
	rs := Starter()
	rs.Name, rs.Preset = "Vault", PresetVault
	rs.Comment = "The Starter, with the ground rules that stash or sell turned into vault: the bot records the item and leaves it on the ground. docs/vault.md."
	var rules []Rule
	for _, r := range rs.Rules {
		if len(r.Scope) == 1 && r.Scope[0] == PlaceGround {
			switch {
			case vaultKeepsOwnRule[r.ID] || r.ID == "g-poor-gear":
			case r.ID == "g-magic-rare" || r.Action == Stash || vaultUnidJewelry[r.ID]:
				r.Action, r.Pickup = Vault, nil
				r.Reason = "recorded in the vault: " + r.Reason
			case r.ID == "g-low-gems" || r.ID == "g-junk":
				if recordAll && r.ID == "g-junk" {
					continue // replaced by the catch-all below
				}
				r.Action, r.Pickup = Ignore, nil
			}
		}
		rules = append(rules, r)
	}
	if recordAll {
		rules = append(rules, Rule{
			ID: "g-vault-all", Scope: []string{PlaceGround}, Action: Vault, Reason: "everything else that is not gold, a gem or a stack", Tier: "C",
			When: &Cond{All: []Cond{
				{Not: &Cond{F: "type", Op: "in", V: mustRaw([]string{"gold", "gem"})}},
				{F: "stackable", Op: "eq", V: mustRaw(false)},
			}},
		})
	}
	rs.Rules = rules
	return rs
}
