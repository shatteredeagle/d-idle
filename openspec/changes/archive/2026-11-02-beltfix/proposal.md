## Why

Gate 1 run gate1-20261102-1939 failed on one criterion, BELT: 12 virtual minutes outside town without a healing potion (limit 5). The cause is real (not a judge artefact) and is the same family as the Gate 2 BELT failure (poverty plus a kit trip that never fires). See docs/research/beltfix-analysis.md.

## What Changes

- internal/bot/kitgate.go: kitTrip prices a character that lacks healing potions at one potion (KitPotionGold), not at a portal scroll (KitTPGold), because the shop buys the scroll first only when it can pay for it. A poor character in Blood Moor with 30 to 99 gold and no scroll now walks home to buy a potion.
- internal/bot/goal.go (choresDue): a kit trip with no healing potion anywhere counts as an emergency, so the cooldown is a quarter of ResupplyCooldownSec.
- internal/bot/beltfix_test.go: replay of the window state.

## Impact

Bot behaviour only, no spec or knob change (skip_specs). Overlap (declared): internal/bot/goal.go and kitgate.go were last changed by kittrip, kitstall and townshop (merged); no open card owns them.

Second pass: internal/town/shop.go buyPotions leaves 130 gold (one healing potion plus one scroll) before any mana potion; internal/bot/kitgate.go kitTrip counts an activated waypoint as a way home.
