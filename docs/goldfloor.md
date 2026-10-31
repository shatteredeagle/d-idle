# Gold floor: where gold comes from and goes (agent GOLDFLOOR, 2026-10-31)

> status: analysis, read-only on the soaks. Card: `openspec/changes/goldfloor` (proposed). Tags: V = read from the logs (worker host: NAVSOAK, NAVSOAK2, NAVSOAK3 at 09:24, m2early 8 seeds x 2 virtual h), I = inferred. Builds differ (m2early old; NAVSOAK3 has BASICSFIX/M1LOOT), seed 5 early game is identical in NAVSOAK2/3.

## Method
Per `minute` line (about 1 per virtual minute; kills/h matches the 190 of navsoak-compare): spend = `shop: bought` prices by kind; piles = `loot: ... gold +N`; sales and other = gold change + spend - piles (V: one sell visit at 4 items paid +89 to +116 gold in NAVSOAK3 08:24-08:25). Death minutes excluded. Wedged minutes (NAVSOAK 11-13 band, NAVSOAK2 8-10) are not counted in the table.

## Ledger, gold per virtual hour (piles / sales / spend = hp + mana + TP)
| run, band | piles | sales | spend | hp | mana | TP |
|---|---|---|---|---|---|---|
| NAVSOAK 1-4 | 101 | 546 | 584 | 284 | 142 | 158 |
| NAVSOAK 5-7 | 86 | 445 | 532 | 152 | 223 | 156 |
| NAVSOAK 8-10 | 119 | 674 | 749 | 67 | 225 | 457 |
| NAVSOAK3 1-4 | 79 | 413 | 427 | 167 | 74 | 186 |
| NAVSOAK3 5-7 | 261 | 1080 | 1271 | 255 | 255 | 760 |
| NAVSOAK3 8-10 | 170 | 973 | 1169 | 221 | 305 | 643 |
| m2early mean 1-4 (8 seeds) | 108 | 632 | 639 | 128 | 195 | 316 |
| m2early mean 5-7 (8 seeds) | 121 | ~700 | 822 | 151 | 191 | 479 |

Share of all spend (V): TP scrolls 48 % NAVSOAK (135 bought, 13.8k), 55 % NAVSOAK2, 55 % NAVSOAK3, 54 % m2early; mana 34 / 22 / 23 / 27 %; healing 18 / 23 / 20 / 19 %. Income: sales 80 to 85 %, piles 15 to 20 % (piles about 0.4 to 1.3 gold per kill). Net gold change is about 0 in every band: income is 430 to 1270 gold/h and the shop spends all of it.

## Findings
1. **The floor is spend-to-zero, not income (V).** Gold is under 100 in 31 to 90 % of minutes because each town visit buys until broke; healing potions are only about 20 % of spend. The SME's 60-200 gold/h is piles only; sales add 400 to 1000/h.
2. **Trips (V).** NAVSOAK: 472 trips, 394 (83 %) spent under 30 gold (avg 11), 334 raised by a `[N healing potions]` count; NAVSOAK2 1278 of 1307 (wedge repeats); NAVSOAK3 49 of 94 (the BASICSFIX/M1LOOT build halves it); m2early 65 %. About 9 trips/h (navsoak-compare). Without a scroll each is a walk; with one, the next shop visit rebuys it: NAVSOAK3 bought two TP scrolls in 25 s (08:24:49 and 08:25:14, 200 gold) at gold 209, then had none again at 08:25:23 (V, scroll-then-rebuy loop still present).
3. **Potions (V).** 16 to 25 potions used/h, 0.08 to 0.12 per kill (Cave Level 1 0.10, Burial Grounds 0.13 to 0.16, Stony Field 0.11 to 0.17, Blood Moor 0.03 to 0.07); healing 150 to 280 gold/h is below income. Mana at 60 gold each is 190 to 305 gold/h in bands 5-10, more than healing there. Belt empty 9 to 45 % of minutes (NAVSOAK3 band 8-10: 45 %).
4. **Loot (V code, I effect).** Outside recovery with gold under 2000 only items at 100+ gold per cell are picked (`MinValuePerCell`, pickup.go:330); in recovery anything at 6+ (`PoorMinSell`). NAVSOAK3 logs about 230 `drop plain or inferior` lines. Average sale is about 25 gold an item (V, 3 visits). Recovery is not worse for kills (202 to 276 kills/h against 163 to 190 overall) and earns about 140 to 195 gold/h at the recovery counter.
5. **Blood Raven gate (V).** Not binding: NAVSOAK2/3 finished the quest at clvl 6 with 139 gold; NAVSOAK at clvl 11 with 259 (first 250 at clvl 7, 353). `boss.brGold` is a refill reserve (tuning_boss.go:19). Death drops gold: m2early lost 244 and 259 gold in two clvl 1-4 deaths.

## Three levers (gold/h saved or earned)
| lever | effect | basis |
|---|---|---|
| A trip purchase floor: a non-emergency trip needs a planned purchase worth 60+ gold (or a belt at 0 healing) and no TP rebuy while gold is under the kit line | +150 to +400 (TP spend is 156 to 760/h, 83 % of trips buy under 30) | V spend, I trips avoided |
| B mana cap: 1 mana potion until clvl 10 (no mana-belt trip trigger) | +100 to +250 (mana 74 to 305/h) | V spend, I risk to casting |
| C loot bar: `MinValuePerCell` 100 -> 20 outside recovery | +50 to +150 | I (skipped item values are not logged) |
| earlier selling / lower kit line 500 -> 150 | about 0 gold/h (kit line only gates areas; sales per item are fixed) | I |

## SME alignment (d2-sme, from play, no sources read)
Agrees: skip early TP buys (walk), sorceress buys few mana potions, gate Blood Raven on potions and clvl not gold, kit line to about 150. Disagrees: says piles give 1-3k gold/h and sales are minor; the logs show piles 80 to 260 and sales 400 to 1080, so the pile-pickup check it asks for is cheap but sales are the lever (UNSURE on its side, V on ours). Unsure of both sides: vendor price formula, unidentified sale price.

## Order
1. Lever A (trip floor plus no TP rebuy when poor), 2. lever B, 3. lever C; leave the BR gate and kit line alone (no gold effect, I). Measure with the paired tooling, 8 seeds x 2 h, on gold/h net, `bought` kinds and trips under 30 gold.
