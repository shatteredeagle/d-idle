# Gate 1 20261102 BELT window: analysis and fix

Tags: V run = read from the run data of gate1-20261102-1939 (all-minutes.log, logs/bot-0.log on the worker host, read only), V read = read in code, I = inference, U = unverified.

## 1. The window

V run: d2dsoak counts consecutive minute lines outside town with `packHP == 0` and no `Healing` in `beltBoxes` (internal/d2dsoakstat/invariants.go, `hasHealing`). Replaying that rule over all 1182 lines gives two windows: minute lines 439 to 442 (4, harmless) and 446 to 457 = 12 (the failure). Wall 20:16:26 to 20:17:25 (60 s; pace 12x, so 12 virtual minutes), seed 11, clvl 12, 93k XP.

State in the window (V run): area 4 (Stony Field) for lines 446 to 448, then area 2 (Blood Moor); gold 30 to 38; state `recovery` ("Recovery: farming gold"); packHP 0; belt boxes only `Minor_Mana_Potion` (five of them); town portal source empty (shop visits log `tp:0`; the tome is bought but has 0 charges); 80 kills in Blood Moor yielded no gold (gold -35 over 29 minutes of recovery). The window ends when a drop gives a potion (line 458, `Found Light Healing Potion`) and later a 313-gold pile.

## 2. Why the kit logic did not restock

V run, in order:

1. 20:15:29 trip at 388 gold: bought 4 healing and 3 mana potions (300 gold, 180 of it mana at 60 each). 15 s later (Black Marsh) the four healing potions were drunk and an emergency trip followed (gold 88 -> 17, 2 more potions, burns a scroll).
2. 20:16:12 kit trip (30 gold, 1 potion) and 20:16:19 emergency trip (sold a dagger and club: 38 gold, 1 potion): the last two portal charges are gone, gold ends at 8. Each fight drinks the single potion in seconds (`flee: no potion, no portal` 20 times).
3. Gold 30 to 53, no scroll, no potion, Blood Moor (walkable home, V live per town/trip.go). `kitTrip` (internal/bot/kitgate.go, V read) priced the trip at `KitTPGold` (100) whenever the scroll was missing, so with 30 to 99 gold it returned false for 12 virtual minutes. The only trip logged is `[low on healing potions]` at 20:18:10 (after a gold pile), which then failed `no town portal available` because it came from Mausoleum, away from the walkable fields.
4. The ordinary cooldown (`ResupplyCooldownSec` 240 s) also holds the non-emergency kit trip back after a trip (V read, goal.go `tripDue`).

The shop does not need the 100 gold: `buyFirstTP` buys the scroll only when it can pay (V read, town/shop.go), otherwise the potions are bought (the 20:16:12 and 20:16:20 visits bought a potion at 30 and 38 gold).

## 3. Same cause as Gate 2?

I: same family, not the same trigger. Gate 2 (docs/research/gate2-analysis-20261102.md, branch gate2analysis): poverty at clvl 11, 33 gold, belt only mana, an emergency kit trip whose portal was cancelled by a crowd retreat, then a walk. Here: poverty at clvl 12 (gold 30 to 53), no portal source, and no trip at all because of the scroll price. Both end in "0 healing potions in the field with gold for one potion". kittrip/kitstall fixed the trip when the kit is partly there; this window is the kit-empty, scroll-less case they skip. townshop and tomestat (tome count) are not involved (the tome stat read the empty tome correctly: `hastp:false`, V run).

## 4. Artefact check

Not an artefact (V run): `packHP` 0 and `beltBoxes` list only mana potions in every line of the window; the inventory dump at the start of the window shows no healing potion; the bot logged `potion: not drinking because no healing potion in the belt` 20+ times and fled. The checker rule is the spec rule (`beltBoxes` carries every column type, so no column is missed). Pace was 12x (not starved), as in Gate 2.

One judge detail worth noting, no change proposed: the window counts virtual minutes, so at 12x a window of 5 is only 25 wall seconds; one walk home plus a vendor visit costs about that much. The spec number is untouched.

## 5. Fix (bot)

- internal/bot/kitgate.go `kitTrip`: when healing potions are missing, the price that gates the trip is one potion (KitPotionGold), even without a scroll. The scroll price applies only when the potions are complete and just the scroll is missing. The walkable-home condition is unchanged.
- internal/bot/goal.go `choresDue`: a kit trip with no healing potion anywhere is an emergency (cooldown a quarter of 240 s).
- Tests (internal/bot/beltfix_test.go): `TestKitTripNoPortalPotionPrice` replays the window state (clvl 12, Blood Moor, 37 gold, no scroll, no potion): trip wanted, emergency; 29 gold, Stony Field and "potions complete, scroll unaffordable" stay quiet. `TestKitTripNoPotionEmergencyCooldown` checks the 90 s gap. Both fail on the old code (V run: ran them with the change reverted).

I: on the replayed window the trip would start at about line 449 (first 37 gold at 20:16:43, area 2), a short walk, so the window is about 5 to 6 minutes; not safely under the limit. Lines 446 to 448 (Stony Field, 30 gold) are not covered: no walk home there without an activated waypoint (U).

## 6. Second pass (coordinator request)

- Mana reserve (internal/town/shop.go `buyPotions`, `manaReserve`): a mana potion is not bought if it leaves less than KitPotionGold + KitTPGold (30 + 100 = 130). Healing is bought first, so it is unaffected. On the 20:15:29 visit (388 gold) the order HP x4 (120), then mana: 268-60=208 and 148 pass, the third (88) is refused: two mana potions instead of three, 60 gold more in the purse. Test internal/town/manareserve_test.go (600 gold buys the mana fill, 150 gold buys none); fails with the change reverted (V run, worker host: the town tests need the extracted tables and skip on the small box).
- Lines 446-448 (Stony Field, 30 gold, no scroll): options the bot has in game terms. No scroll and a 0-charge tome; the walk home exists only from Blood Moor, the Den and Cold Plains. Stony Field has a waypoint and the run used it (V run: `wp: used 1 -> 4`), and `Town.ToTown` already falls back to `waypointToTown` when there is no scroll (V read, town/trip.go). The only gap was `kitTrip` not counting it as a way home. Now `kitTrip` accepts an activated waypoint in the area (`wpActive`, from the quest/waypoint reader). Gold pickup first or skipping gold hunting is not a smaller change: gold income at clvl 12 in Blood Moor was about zero (-35 gold in 29 minutes), so waiting for 100 gold is the stall itself. Test `TestKitTripWaypointHome` (active: trip; not active: none); fails reverted. U: the waypoint trip with an empty pack has not been run live (waypointToTown is marked U in its comment).
- Replay result (I, not run): line 446 is the first window line and the trip needs only the emergency gap (60 virtual s) since the 20:16:20 trip, with 30 gold = KitPotionGold and an activated Stony Field waypoint; the waypoint trip and the vendor visit take seconds of wall time (about one virtual minute at 12x). Expected window 2 to 3 minutes. The earlier 440s of poverty would also differ (two mana potions fewer at 20:15:29 leaves 60 more gold), so the exact replay is not reproducible; not measured live.
- Still open: syncKitGate lagged behind the window start (`kit gate on` at 20:16:38); not changed.
- Re-run Gate 1 on the merged commit with seed 11 and one more seed.
