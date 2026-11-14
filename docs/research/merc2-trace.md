# MERC2: hire, free Rogue, revive and merc behaviour; game.exe 1.14d vs d2d vs our bot (agent decomp16, 2026-11-14)

> status: research, current (card decomp16, TASK-400; consumer: the gold-priority and hero14 follow-up cards, `docs/probes/hero14-analysis-2026-11-14.md` cards 4 and 5). Offline only: no game instance, no d2d slot, no fork work; nothing pushed or posted. This doc consolidates the finished traces (`docs/research/mercstats-trace.md`, `pets-merc-trace.md`, `palaura-trace.md`, `hire-window-trace.md`, `xp-gold-trace.md`) and adds what they left open: the revive handler and its charge path, the free-Rogue reward path, and the bot's revive gates. Tags: V run = game.exe's own code run under the emulator on the worker host (this card: the two price oracles, 3,856 cases, below); V read = read in the disassembly of the read-only copy (md5 prefix 2b3600c1, address named); D = from the patched hireling.txt / skills.txt (read with `mpq-table.sh`); S = an earlier trace of ours (named); I = inferred; U = unverified. Per the model rule, only V run is a number a PR could rest on; no PR is claimed here. Helper names from the mined repos were used as a read-only cross check of address meaning only (`docs/research/d2-name-map-2026-11-12.md`); nothing was copied.

## Answer in eight lines

1. **Revive costs `(merc level^2 / 2) x 15`, capped at 50,000, and nothing else.** game.exe computes it from the merc's level stat at the moment of payment (`FUN_006637b0`), takes it from carried gold first and the stash after (`FUN_00576d90`), and requires no portal, no potion and no reserve: the handler `FUN_00579c00` checks only that the NPC is a hirer, that the merc unit exists, and that gold covers the price. Merc level 9 = 600 (the hero14 case), 10 = 750, 12 = 1,080, 13 = 1,260, 15 = 1,680, 20 = 3,000. V run (256 levels, 0 differ from the bot formula) + V read.
2. **There is no free or timed revival.** The only code that puts a dead merc back is `FUN_00579aa0`, and its only caller is the paid handler `FUN_00579c00` (call graph of the linear disassembly, V read; indirect calls not covered: I). A dead merc stays dead across areas, game exit and load; the price is sent to the client at the moment of death and again at load (`FUN_005751a0`, V read).
3. **The hire price is `Gold x (15 d + 100) / 100`** with d = offered level - the row's base level (never under Gold), the offered level `max(2, clvl - 5 + rand(5))`. Act 1 Normal Rogue at clvl 14: offered levels 9..13, Fire Arrow 190..250, Cold Arrow 285..375. Kashya's menu shows "hire" only above clvl 7. V run (3,600 offers this card; 108,000 in MERCSTATS vs d2d, 0 differ).
4. **The free Rogue is `FUN_00579180`**, called from Kashya's talk handler after Sisters' Burial Grounds (`FUN_00590980`, seller 0x96) and from Asheara's talk handler after the Blade of the Old Religion (`FUN_005b9240`, seller 0xfc, an Iron Wolf; d2d has it, our bot's contract does not mention it). It hires the first available entry of that NPC's list for no gold; any merc the player owns, a dead one included, refuses it. V read; the "Normal only" word in our contract is not in the function (U, see D5).
5. **The merc does not drink its own potions and never retreats.** Its think and attack functions read no life stat; potions arrive only through the hand-over (`FUN_0054d230`: healing, rejuvenation, antidote and thawing classes; mana and stamina are refused). V read + V run (S, MERCSTATS Q4, Q6).
6. **Leash:** over 100 subtiles off the owner it teleports, over 24 it runs, over 16 it follows; only then does it fight a foe under 25. V run 100,000 cases, 0 differ from d2d (S, MERCSTATS Q4 / pets-merc-trace).
7. **Prayer heals the merc** (filter 0x12003 passes the owner's merc and pets, not in town, circle `d2 <= r2` with r = 16 + 2 per level subtiles, 50-frame pulse, heal clamped at max life); d2d heals the player only. V run (S, palaura-trace D1 / D2). **A merc's kill gives the hero full XP**, and the merc gets 2 x the award at its own level (86/256 of it unless it struck the blow) while it is under the owner's level; a dead merc earns nothing. V read / V run (S, xp-gold-trace Q3, MERCSTATS Q3).
8. **No d2d deviation proved here; no PR candidate.** Bot findings: the revive gate asks for a portal and potions that the game does not (B1), the reserve is policy not rule (B2), the contract's "Normal only" and the missing Act 3 reward (B3, B4), the revive is written as a recovery of the loop, not of the order (B5).

## Q1 hire (V run, D, V read)

| Item | Rule | Address | Tag |
|---|---|---|---|
| Menu gate | Kashya's record is rewritten to talk / hire / cancel only when character level (stat 0xc) is above 7; the other hirers carry "hire" statically | `FUN_004b66b0` / `FUN_004b6410` (read by upstream) | S (hire-window-trace Q1.2), I for the other hirers |
| The list | seller's first hireling.txt row of the difficulty gives the name range, ten entries; each entry later becomes an offer | `FUN_00576070`, `FUN_00577010` | S (upstream hire.py, 400 lists) |
| The offer | rows of the act and difficulty sharing the first row's Level, one picked; level `max(2, clvl - 5 + rand(5))`; life `max(40, HP + HP/Lvl x d)`; cost `max(Gold, Gold x (15 d + 100) / 100)`; exp `(L + 1) x Exp/Lvl x L^2` | `FUN_006637f0` | V run (this card 3,600 game offers vs the bot formula, 0 differ; MERCSTATS 108,000 vs d2d, 0 differ) |
| The claim | merc created from the entry, re-rolled with the owner's CURRENT level at the claim, experience set to the offer's, `FUN_00572840` run for that level; no equipment is made in the create path | `FUN_00573270` (callees: `FUN_006637f0`, `FUN_00572840`, `FUN_005726c0`, `FUN_005a4850` = queue the first think) | V read (callee list), the "no gear" part I (indirect calls) |
| Payment | carried gold first, then stash | `FUN_00576d90` (shared with the revive) | V read; d2d `rules::hire` same order, S |

Prices from the oracle (this card, 100 seeds per cell, Normal Act 1, `tools/emu/mc2_tab.sh`): clvl 1 level 2 cost 100..150; clvl 9 levels 4..8 cost 115..262; clvl 30 levels 25..29 cost 430..735; Nightmare Rogue 6,000..7,500 and Hell 12,500..14,000 at every level up to the row's own (so a paid Rogue is a Normal-only idea for the bot). Act 2 Normal Desert Mercenary is a flat 350 until level 25, Act 3 Normal Iron Wolf 1,000..1,500. The bot formula and the offered-level rule are in `internal/merc/formulas.go:8..17` and equal game.exe on all 3,600 lines.

### Q1a the free Rogue (V read; the bot's contract rests on it)

`FUN_00579180(game, player, seller id)`:
1. looks up the seller's hire list (`FUN_00535ea0`); no list: nothing.
2. in an expansion game it asks `FUN_00574ec0(game, player, 7, 1)` for the player's hireling slot, WITHOUT skipping the dead flag; a non-null unit means the reward is refused (it only re-rolls the list, `FUN_00577010`). The same lookup is what the revive handler uses to find a dead merc, so a dead but unrevived merc still blocks the reward (V read; the corner "the dead merc's unit has been freed" would make both fail, so I). In a classic game the lookup skips dead mercs (flag 0), so a dead one does not block.
3. picks the hireling row of the active difficulty (`FUN_006564d0`, game byte +0x6d), walks the seller's entries (16 bytes each) and takes the FIRST entry whose +8 is not 1 and +0xc is 1; marks it taken, tells the client (packet 0x50 subtype 2), makes the merc through `FUN_00573270` with the owner's level, with no payment code in the function.
4. callers: `FUN_00590980` at 0x590af6 (Kashya 0x96, in the Kashya quest talk handler, guarded by the quest word of the ACTIVE difficulty: bit 1 set, bit 0xd pending), and `FUN_005b9240` at 0x5b94ce (Asheara 0xfc, quest 0x13 bit 7, "Blade of the Old Religion" in d2d's numbering).

So the first merc hire forfeits nothing only if no merc is owned; "hire first, claim after" is refused (V read). Whether the reward fires in Nightmare and Hell is a quest-state question the function does not answer: the callers index the quest data by difficulty and no "Normal" test is in the read path (V read); whether bit 0xd is ever set for Blood Raven in Nightmare is U (drill R1).

## Q2 revive (V read + V run)

| Step | Rule | Address | Tag |
|---|---|---|---|
| Death | the merc's record is flagged dead, the price is sent to the owner's client (string 0x2ba) and the unit leaves the world via the unit callback; the same handler runs when a dead merc is loaded | `FUN_005751a0` (callers 0x533ed0, 0x56abd2 in the loader, 0x57cd21) | V read |
| Client | "Resurrect %s: %d" (string 0x1507) is added to a hirer's menu while a dead merc's name is known | `FUN_004b6440` | S (hire-window-trace Q1.3) |
| Packet | expansion game only, 5-byte packet, the NPC id at +1 | `FUN_0054bc00` -> `FUN_00579c00` | V read |
| Handler | the NPC must be the player's current interaction target and be one of five hirers (hcIdx 0x96, 0xc6, 0xfc, 0x16f, 0x203: Kashya, Greiz, Asheara, Tyrael, Qual-Kehk); merc unit found by `FUN_00574ec0(.., 7, 1)`; price `FUN_006637b0(merc)`; pay `FUN_00576d90`; on success clear unit flag 0x10000 (+0xc4), set life to the full value, place the merc by the owner (`FUN_00579aa0`) | `FUN_00579c00` | V read |
| Price | `(L x L / 2) x 15`, unsigned minimum 0xc350; L = the merc unit's level stat (stat 0xc) | `FUN_006637b0` | V run, 256 levels, 0 differ from `merc.ReviveCost` and `resource.MercReviveCost` |
| Payment | needs carried + stash >= price (else refusal string 0x994); carried gold is taken first, the rest from the stash | `FUN_00576d90` | V read; d2d `rules::resurrect_merc` equal, S |
| Failures | not a hirer: 0x96c; no merc unit: 0x973; short of gold: 0x994 | `FUN_00579c00` | V read |
| Revival sites | `FUN_00579aa0` (the placement) has exactly one caller, `FUN_00579c00`; no timer or area hook returns a dead merc | linear call graph | V read, indirect calls not covered (I) |

Table of the price by merc level (V run, `mc2_run.sh`, game side): 2: 30, 3: 60, 4: 120, 5: 180, 6: 270, 7: 360, 8: 480, 9: 600, 10: 750, 11: 900, 12: 1,080, 13: 1,260, 14: 1,470, 15: 1,680, 16: 1,920, 18: 2,430, 20: 3,000, 25: 4,680, 30: 6,750, 40: 12,000, 82 and over: 50,000. The merc's level is its own (rises with the hero's kills while it lives and its level is under the hero's), not the hero's: the hero14 merc was level 9 at clvl 14, hence 600.

What the bot must pay for: gold equal to the price in carried plus stash, in town, with the hirer's menu open and the NPC targeted. Nothing else: no portal, no potion (those matter only for the walk back to the field). The price is not a function of difficulty, act or hire cost.

Side finding (D6, no UI path): the handler does not test that the merc is dead; a hand-built packet for a living merc would charge and re-place it. The client menu never offers it (the line needs a dead merc's name), so it is not reachable in play.

## Q3 follow, leash, AI (S + V run, restated)

All from `mercstats-trace.md` Q4 and `pets-merc-trace.md` Q2 (V run 100,000 + 100,000 + 100,000 cases against d2d's real C++, 0 differ): the think `FUN_005e52d0` checks the owner first (gap over 100 teleport to a spot of the owner's room, over 24 run at pace 60, over 16 follow when the owner walks or runs); out of town a foe under 25 hands over to the attack think `FUN_005e5050` (the Rogue backs off 4 from a foe under 4 half the time; skill roll `40 + 2 x level + 10 per miss`, max 95 percent per think for the Rogue, 98 for the Desert merc); the target search `FUN_005ddc30` mode 6 (nearest threat under 49 in sight). No function of the three reads life (V read, callee lists), so a hurt merc fights on and follows; only the leash moves it.

Speed: the merc moves at MonStats Run (11) when running and its walk uses Velocity 7 (D, MonStats rows 271 / 338 / 359); d2d has the known walk-speed deviation (`walk-speed-trace.md`, S, pets-merc D2).

Merc potions: the merc never drinks by itself. The player hands one over (shift-click a belt potion on the hireling screen, packet 0x61 slot 0): `FUN_0054d230` accepts type 0x4c / 0x50 / 0x51 and their children, i.e. healing and rejuvenation, antidote, thawing; mana and stamina are refused (V run over 659 item classes, MERCSTATS Q6). The life the potion gives the merc is U (the use function `FUN_005bf240`, amount not read).

Prayer: the owner's Prayer heals the merc under filter 0x12003 (not in town for other units, in town only the caster), `heal = EMin-table value per 50 frames` (level 5: 6 life per pulse, 1.75 mana, mana is paid and state 85 set only on a pulse where somebody's life rose; below the cost nothing heals), radius `16 + 2 x level` subtiles (V run, palaura-trace section 5, 21,760 aura cases in pal-skill-trace). The skill table: Prayer id 99, srvdofunc 65, aurastat `hitpoints` with calc `edns`, `perdelay` 50, Param1 16 / Param2 2 (D, this card read skills.txt). A merc's own Prayer (Act 2) heals the owner's side by the same do; the merc rolls it once and then leaves it out of its roll (S, MERCSTATS Q4); its heal amount is U.

## Q4 equipment (V read, D, S)

- Slots: head, torso, right (weapon), left (shield), per hireling.txt WType columns: Rogue a bow, Desert a spear or polearm (D: WType1 `pole`, WType2 `spea`), Iron Wolf swords and shields, Barbarian a sword (D). Wearing rules: identified, not broken, the merc's own level, Str and Dex (`FUN_0054d230` / `FUN_0048b290`, S merc-gear / mercstats Q5).
- A dead merc cannot be given or relieved of items: the gear handler returns early for a merc with life 0 (S merc-gear section 2.2, d2d `world.cpp`); a REPLACING hire destroys the old gear (S, sme/mercenary.md).
- Starting equipment: hireling.txt has Head / Torso / Weapon / Shield columns (Rogue 2 / 1 / 2 / 0, Desert 2 / 2 / 2 / 0, Iron Wolf 2 / 2 / 2 / 1, Barbarian 0 / 0 / 0 / 0; D) and MonEquip.txt has no hireling row (D, 46 rows, none). The create path `FUN_00573270` makes no item (callee list, V read). Whether a fresh hire carries items in its slots, or the columns only choose the look, is U (drill G1). d2d gives a hire no items (`rules::hire`, S, read).
- Arrows: the Rogue's attack is a monster skill (MonStats Sk1 `RogueMissile`, Sk1mode A1; D), not a player quiver shot; no ammo count exists on the unit (I; drill G2).
- The bot never decides gear: no caller of `merc.EquipIfBetter` outside tests (S merc-gear section 1).

## Q5 XP (V read, V run, S)

- Every kill by the hero's merc or pets credits the hero's own award in full (control flow read: the award is credited to the owner of whoever landed the blow; no party share in single player): `xp-gold-trace.md` Q3 (V read; the constants are upstream's, S). A merc kill is therefore the same XP as a hero kill.
- The merc earns separately: `FUN_0057e990` gives the merc `FUN_0057e480(victim exp at the merc's level) x 0x56 / 256` unless the merc struck the blow, and `FUN_0057e860` adds 2 x that amount to its experience while the merc's level is under the owner's and under 98 (V read, V run 100,000 cases vs d2d `merc_kill_exp`, 0 differ).
- A dead merc earns nothing and does not level (I: the award code runs for a living unit; drill R5); a merc left dead for hours costs its level growth and its damage output (D1 of this trace is the hero14 consequence, I).

## Q6 differentials: d2d vs game.exe

| What | Game side | d2d side | Result |
|---|---|---|---|
| Hire offer, cost | `FUN_006637f0`, this card 3,600 offers vs the bot formula | `mst_d2d_test hire` (MERCSTATS) | 0 differ vs the bot (this card, V run); d2d 108,000 offers 0 differ (S, V run earlier) |
| Resurrection price | `FUN_006637b0`, 256 levels | `rules::merc_resurrect_cost` | 0 differ (this card vs bot; MERCSTATS vs d2d) |
| Merc think, attack, target, XP | MERCSTATS and pets_merc oracles | d2d's real rules | 0 differ in 100,000 each (S) |
| Revive handler (pay order, hirer list, dead flag, life) | read only | `rules::resurrect_merc`, `kMercNpcs` (read) | not run: needs a populated player and merc; equal by reading except life (D6, I1) |
| Free reward | read only | `World::kashya_merc` (read: first entry of the list, cost 0, a merc in an expansion game blocks even when dead) | equal by reading; difficulty gate not compared |

Not re-run on the d2d side in this card: the mercstats scratch build is not on the worker host any more (the shared emulator data dir was gone; this card rebuilt a data dir in the scratch with `mc2_setup.sh`).

## Deviations d2d vs game.exe

| id | rule on both sides | tag | differential |
|---|---|---|---|
| D1 | friendly auras (Might, Prayer, Defiance, Concentration, Vigor ...) reach the merc, pets and party in game.exe; d2d applies them to the caster only, and Prayer heals the player only, free | V run game side (palaura-trace D1/D2), d2d S | could be built: palaura_apply.py friendly on a merc unit vs a d2d `Fight` test with a merc under the aura; oracle = the table of palaura section 2 |
| D2 | merc life regeneration (stat 0x4a = max life 256ths / 2000, set at level up) exists in game.exe, absent in d2d (S, MERCSTATS D1) | stat V run, effect U | could be built: hurt a merc through `FUN_0057c8d0`, run the regen event `FUN_00580810` N frames |
| D3 | the Act 2 merc's aura starts and counts as running in d2d but does nothing (S, MERCSTATS D2) | V read d2d, effect U | could be built through `FUN_005701b0` |
| D4 | life after a revive: game.exe sets stat 6 from `FUN_00625d10(merc, 0)` (the unit's max-life accessor, I); d2d's comment says "full life" and spawns the merc; equal if the accessor is max life | I | could be built: run `FUN_00579c00` on a hooked merc (payer, stat accessors) and read stat 6 after |
| D5 | free reward in Nightmare / Hell: game.exe's function has no difficulty test and indexes by the player's difficulty (V read); d2d's `kashya_merc` also takes the active difficulty's list and quest word; our contract says Normal only (S, no source line) | V read both, quest-bit behaviour U | could be built: run the `FUN_00590980` talk branch with the quest word at Nightmare; oracle d2d's `burial.said` |
| D6 | the revive handler does not test the merc is dead; d2d's `resurrect_merc` requires `merc_dead` | V read / V read | not reachable from the client UI: no PR value, no differential |
| D7 | classic game: mercs are not revived (the packet answers 3, `FUN_0054bc00`); d2d has `expansion` branches | V read, d2d S | not needed: the bot plays LoD |

All of D1 to D3 are missing features or feature gaps, not proved rule mismatches with a ready harness here; the rule differentials above show 0 mismatches. No PR candidate (see below).

## Bot findings (described, NOT changed)

| id | where | finding | tag |
|---|---|---|---|
| B1 | `internal/merc/policy.go:233` (`!c.HaveTP \|\| !c.HavePotions` -> `ReasonKit`), contract `docs/contracts/merc-policy.md` "Revive and stop rule" | the revive is refused without a town portal and potions; game.exe asks for gold only (Q2, V read). The wiring sets both true on the first pass of a town visit (`cmd/d2bot/mercwire.go:737`), so the gate bites only on later passes; it is not a game rule and it makes the revive depend on spending 1,000 gold (tome 400 + 6 scrolls 600 in hero14) first. Make the kit gate a "can I come back to the field" check, not a precondition of the payment | V read rule, I effect |
| B2 | `internal/merc/policy.go:248..264` (`ReviveNeed`: price x mult + reserve 50 from clvl 13), `internal/env/tuning_merc.go:21..23` | game.exe takes exactly the price (600 at merc level 9); the bar 650 of hero14 was policy. At a purse hovering 600..650 the 50 reserve alone keeps a merc dead, and the hero14 purse spent itself on the kit; the reserve should be 0 when the kit is already complete, or the revive should come ahead of every kit purchase that is not the portal (the contract already says "before potions and repairs") | V read rule, I effect |
| B3 | `docs/contracts/merc-policy.md` line 19 ("claim ... Normal only"), `internal/merc/policy.go` `ShouldClaimFreeRogue` | the "Normal only" is not in `FUN_00579180` or its Kashya caller; at Nightmare and Hell the reward may exist (U, drill R1). The cost is nothing, so the claim step could try at every difficulty and log the result; the contract line is S, not V | V read, U |
| B4 | `internal/merc/` (no code), `docs/act3.md` | game.exe also grants a free Iron Wolf from Asheara after the Blade of the Old Religion (`FUN_005b9240` -> `FUN_00579180`, seller 0xfc); the bot never claims it, and a hire of Rogue or Desert merc first forfeits it. A caster at Act 3 with a dead Rogue would swap for free. Decide in the Act 3 plan; no change here | V read, effect I |
| B5 | `docs/probes/hero14-analysis-2026-11-14.md` card 5 ("merc revive 600 before the tome 400") | the order is right, but the lever is the revive gate (B1/B2) and the sell chore, not the order of two purchases: 600 + 400 + 600 > the hero's gold either way until the 1,121 sale | I |
| B6 | `internal/merc/formulas.go:5..23`, `internal/resource/merc.go:9..14` | the comments call the formulas S / U; they are V run now (3,600 offers, 256 prices, 0 differ, this card) | V run |
| B7 | `docs/merc.md` policy 1 / contract line 19 | "a dead owned merc forfeits the reward" is now V read (expansion lookup includes the dead record); the contract may say V | V read |

## PR candidates

None. The only rule comparisons with a harness (price, offer, think, attack, xp) show 0 differences before any fix; D1 to D3 are missing features whose harness is not built here and whose d2d side is not a one-line rule; D4 / D5 / D6 are read-only. Nothing built in the fork checkout, nothing in `docs/upstream/pr-queue.md` (no draft: a draft needs a proven mismatch first, then "draft: needs verifier").

## Drill recipe (hidden d2d instance, throwaway character, ledgered cheats only; verbs by name: `merc`, `spawn`, `hit`, `give`, `dump merc`, `step`, `debug points`)

| id | what | confirms |
|---|---|---|
| R1 | Nightmare (or Hell) character with Blood Raven's quest word at bit 1 / pending: talk to Kashya with no merc, `dump merc` | D5 / B3 (does the reward appear outside Normal) |
| R2 | hire a merc, hurt it to death (`merc hurt`), leave the area, come back; talk to Kashya (Resurrect line present), gold 599 at merc level 9: refused, gold 600: paid, gold falls by 600 (carried first, then stash when carried is short), merc stands by the player | Q2, price and pay order |
| R3 | with no tome, no scrolls and no potions in the pack and 600 gold: revive in town works | B1 (the game has no kit precondition; this also tells whether d2d has one: it has none in `resurrect_merc`) |
| R4 | a merc at level 12 and 13 (`give` experience): revive price 1,080 / 1,260; read `dump merc` level first | price by merc level |
| R5 | the merc dead, the hero kills ten Fallen: `dump merc` experience unchanged, hero experience as usual | Q5 (dead merc earns nothing) |
| R6 | a Paladin with Prayer 5 and a merc at 50 percent life within 16 subtiles, hero at full life: merc +6 per 50 frames in the original, flat in d2d | D1 |
| R7 | the merc's pack: `dump merc` right after a free claim: any items in the four slots | G1 / the starting-gear U |
| R8 | with the Rogue at the Rogue Encampment waypoint, `give` a Rejuvenation potion to the merc through the belt at 30 percent life: life gain | the potion amount U (Q3) |
Pass rule per step: two identical runs; leave the character alive in town, no panel open.

## Matrix-cell updates (proposed text; docs are the owners' files)

- `docs/contracts/merc-policy.md` "Revive and stop rule": replace "only with a town portal and potions and gold of at least ..." with "gold of at least the price plus the reserve; game.exe (`FUN_00579c00`) asks for gold only, so a portal and potions are a policy choice (B1)". Not edited here (the contract changes only in a task that owns it).
- `docs/research/sorc-skill-matrix.md`, `pal-skill-matrix.md` (D4 row "friendly auras: d2d reaches the caster only"): no change; D1 above repeats it for the merc.
- `docs/merc.md` UNVERIFIED list: item "dead merc keeps gear" stays U; add "revive price V run 256 levels (this trace)".

## Dead ends

- The shared emulator data dir (a volatile scratch) was gone; a data dir was rebuilt in the card's scratch from the lab's MPQs and the lab's read-only game.exe copy (md5 prefix 2b3600c1, the same binary as the earlier traces). No other copy was searched for.
- The d2d side of the differential was not rebuilt (the earlier scratch build is not available; the d2d results cited are MERCSTATS's).
- The revive handler's life value (`FUN_00625d10` argument 0) is read as "max life" from the unit stat table slot, not run: D4 stays I.
- The Head / Torso / Weapon / Shield columns: no consumer found in the create path; meaning U.

## Reproduce (the worker host, from the card's worktree; `D2_LABRUN_HOST` set)

```sh
tools/emu/run-on-worker.sh mc2_setup.sh                    # scratch data dir: MPQ links, game.exe copy
tools/emu/run-on-worker.sh --ttl 600 mc2_dis.sh --all      # linear disassembly to the scratch (then mc2_grep.sh ADDR, mc2_fn.py ADDR)
tools/emu/run-on-worker.sh --ttl 1500 mc2_run.sh           # game.exe price oracles + bot formulas: 3600 offers, 256 prices, 0 differ
tools/emu/run-on-worker.sh mc2_tab.sh                      # the cost windows and the price table of this doc
tools/emu/run-on-worker.sh mpq-table.sh hireling           # D: columns used above (also skills, skillcalc, monequip, monstats)
```
Case counts: 3,600 hire offers (100 seeds x 3 acts x 3 difficulties x 4 character levels) + 256 resurrection prices = 3,856 game-side runs; limits as in MERCSTATS (the oracle hooks the unit plumbing).

## Shared-file lines for the merger

Result line for `docs/decomp-queue.md` (append at the end only):
`- MERC2 (decomp16, 2026-11-14): revive = (merc level^2 / 2) x 15 max 50,000 from carried then stash, gold only (no portal, potions or reserve: FUN_00579c00 / FUN_00576d90, V read; price V run 256 levels + 3,600 hire offers vs the bot formulas, 0 differ); no free or timed revival (FUN_00579aa0 has one caller); free Rogue FUN_00579180 from Kashya and, unnoticed by the bot, an Iron Wolf from Asheara after the Blade of the Old Religion, any owned merc (dead included) refuses it, no Normal test in the read path (U); Prayer reaches the merc in game.exe, not in d2d (S); bot B1-B7 (revive gate demands a portal and potions, reserve 50 is policy); no d2d rule deviation with a harness, no PR candidate; docs/research/merc2-trace.md`
