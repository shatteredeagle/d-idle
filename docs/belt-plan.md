# Belt plan knob (TASK-164)

> status: built and tested offline; the live drill is below. Tags: V = proven by a test in this tree, S = from a doc, U = unverified.

The four belt columns (hotkeys 1..4) were code (`resource.PlanFor`, `docs/sme/potion-policy.md`): the policy sheet could move drink lines but no path bought an all-mana or mixed-ratio belt. The player now chooses the plan, including poor choices on purpose (user, 2026-11-07).

## Knobs (`resource.*`, `docs/knobs.md`)

- `resource.beltPlan` template: 0 the build's own plan (today's rule, the default), 1 balanced (healing, healing, mana, rejuvenation), 2 all healing, 3 all mana, 4 three healing to one mana, 5 one to one, 6 rejuvenation only. Data: `internal/resource/beltspec.json` (names, columns, risk lines); an unknown name or id is an error naming it (`resource.TemplateID`, `PlanSpec.Apply`; V `TestSpecErrorsNameTheValue`).
- `resource.beltCol1..4` per-column override over the template or the build plan: 0 keep, 1 healing, 2 mana, 3 rejuvenation, 4 empty (nothing bought or belted for that column).
- Tier is not part of the plan: it follows the level, the town band caps and the potion tier option (`potiontier`, still a record only).
- Class defaults are unchanged and stay in `resource.PlanFor` by `ManaNeed` (casters two mana columns to clvl 30, Paladin/Amazon/Assassin three healing and one mana, Barbarian no mana) and the mana-heavy builds table; U that Paladin "about 4:1" is closer to 3:1 on four columns (a larger belt multiplies each column).

## Where the plan acts (all through one spec)

| place | change |
|---|---|
| `resource.PlanBuys` (shopping list) | spec laid over `PlanForBuild`; a kind is bought only for the columns that hold it (a mana column is filled for any class, V `TestPlanBuysByTemplate` Sorceress, Barbarian, Paladin); healing first, then mana, then rejuvenation; the gold cap and the reserve are unchanged (V `TestPlanBuysGoldShort`) |
| `resource.FieldGate` | wants a share of the PLANNED healing columns; none planned = no demand |
| `town.Town.plan` (shop, belt fill, trips, belt band, `ensureHealing`) | `belt.Band.WithSpec`; the fill and the trips follow the band (V `TestBeltPlanShoppingList`, `TestBeltPlanFillResult`, run on the worker host: the town rig needs the extracted tables) |
| `loot` potion pickup | same band, so ground potions of an excluded kind are not walked to for the belt |
| kit gate (`bot/kitgate.go`) | a plan without a healing or rejuvenation column asks for no healing potions (`k.noHP`); a plan with a mana column asks `kitMinMP` whatever the class (V `TestKitGateAllManaPlan`, Sorceress and Paladin) |
| recovery (`bot/broke.go`) | `resource.KitHealing` waives the healing part of `KitShortfall` and `KitLow` for such a plan, so an all-mana belt is not "low" (V `TestKitHealingWaivesAnAllManaPlan`) |
| town gold reserve | `manaReserve` and the scroll bonus no longer hold back a healing potion's gold for such a plan |

## Safety floor: a plan with no healing potions is "your call, risky"

Not changed and still active: the chicken line (leave the game), the town portal retreat, `kitHolds` (no portal source and no healing potion keeps a recovering character in the safe fields, even for an all-mana plan, V `TestKitGateAllManaPlan` row "all mana, no portal"), and drinking any healing potion the pack happens to hold. The policy sheet shows the risk line on the option.

## Policy sheet

Row `potionmix` options: default, balanced, `h3m1`, `h1m1`, `allmana` (suboptimal), `allhealth` (suboptimal, also lowers the mana drink line), `rejuv` (suboptimal); each sets `resource.beltPlan`. The old option `allmana` no longer moves the healing drink line. Row `belt` (fill share) is unchanged and now applies to the chosen plan. Effect counters are unchanged (V `TestPolicyEffectLineAndCardGolden`, goldens regenerated).

## Not done / U

- `merc` kit check (`cmd/d2bot/mercwire.go kitFull`) still asks `kitMinHP`: an all-mana hero does not hire a mercenary.
- `town.tripMinHPPotions` and other trip knobs are unchanged; they ask for nothing when the plan has no healing column (capacity 0).
- Rejuvenation columns early buy healing in the shop (`buyKind` converts; S, `docs/sme/potion-policy.md`).
- Live drill: see the task result.

## Live drill checklist (not run)

Hidden instance, throwaway Paladin or Sorceress, nice 19, `writes after start: 0`: set `resource.beltPlan` 3 through the tuner (`PUT /api/v1/instances/{id}/knobs`), empty the belt, run a town restock (`NeedsTrip` then `Chores` with potions), then check with `d2cheat` read or the minute line `boxes` that the belt holds mana potions in all four columns and no healing potion was bought; set it back to 0.
