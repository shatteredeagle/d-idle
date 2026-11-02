# SKSTATEEV: which state events fire on a hit and on a miss (FUN_005d3ac0, FUN_005c0c30)

Status: static decomposition, game.exe 1.14d. Tags: **V read** disassembled with upstream's `tools/emu/disasm.py` (addresses given; `tools/emu/sk_stateev_xref.py` finds call sites and the function around an address); **D** Events.txt as the emulator's loader reads it (`tools/emu/mpq-table.sh events`); **S** an earlier trace (named); **I** inference; **U** not verified. Nothing here ran game code: no number is V run, so nothing below is a PR claim. No code or disassembly is pasted.

## Answer in short

1. **FUN_005d3ac0 is not the event dispatcher.** It is an attacker-side hit bonus (Q1). The board text ("walks states and fires events") was an inference from SKMELEE; it is corrected here.
2. **The dispatcher is FUN_005c0c30** (ECX game, EDX event id, stack: unit, other unit, hit record): it walks the unit's event list (+0x90) and calls the function (+0x14) of every node whose id byte equals the event id (Q2).
3. **Event 3 (attackedinmelee, Shiver Armor) fires on a MISS as well as on a hit**, after the damage, from FUN_0057d4f0, as long as the target is within melee reach (Q3).
4. **Event 11 (absorbdamage, Energy Shield) fires inside FUN_0057c1e0, after the %-damage scaling and BEFORE the per-type resistance pass** (Q4).

## Q1. What FUN_005d3ac0 does (V read 0x5d3ac0..0x5d3b92, 0x5d3680..0x5d3782)

| Fact | Detail |
|---|---|
| Signature | ECX game, EDX hit record; returns at once if the record is null or record +4 bit 0 (hit) is clear. So it runs on hits only |
| Loop | over the whole States table (count at game +0x170, ids at +0x16c): for each state the attacker has (FUN_00639df0) it takes the state's stat list (FUN_006256b0), reads stat 0x15e (skill id), checks the id against the Skills count, and the Skills row byte +0x44 must be exactly 1; then FUN_005d3680(list, record) |
| FUN_005d3680 | reads stat 0x15e (skill) and 0x15f (level) from the list, takes the larger of that level and FUN_006442a0 (the unit's level bonus, FUN_006439f0), reads Skills row word +0x54 (a stat id), reads that stat from the list, and adds `FUN_00646ca0(calc at Skills +0x138, level) x stat` into **record +0xc**. Then, for a unit with an anim mode (FUN_00553540), runs FUN_00621e40 with the Skills word +0x108 (an overlay / sound id) |
| Reading | a state-carried skill's flat bonus added to the swing's hit record when the attacker lands a hit (aura-style bonus damage). Which damage row record +0xc is (row table 0x732980, stride 0x2c, 12 rows) and which Skills.txt column is +0x44: **U** (not dumped) |
| Callers | 0x56f163 (the melee swing builder, SKMELEE), 0x5d54bf, 0x5d628a; the first and third set record +4 bit 1 first (V read) |
| Events | none fired here. It fires no event and reads no event list (V read) |

## Q2. The dispatcher FUN_005c0c30 (V read 0x5c0c30..0x5c0cf5)

Walks the singly linked node list at unit +0x90; node byte 0 = event id; on a match sets node flag +2 bit 0, calls the node function (node +0x14) with ECX = game and five stack args (unit, other, record, node +0xc, node +0x10), clears the bit, and returns the last result. A node flagged "once" (+2 bit 1) or with node +4 = 0 is unlinked and freed after the call. Nodes are added by FUN_0056e740 for the aura events of a state (SKARMOR Q1, row FUN_005c9480). Event ids are the Events.txt rows (D): 0 hitbymissile, 1 damagedinmelee, 2 damagedbymissile, 3 attackedinmelee ("melee attack attempt"), 4 doactive, 5 domeleedamage, 6 domissiledamage, 7 domeleeattack, 8 domissileattack, 9 kill, 10 killed, 11 absorbdamage, 12 levelup. SKARMOR's table is the same list (the call sites below use these numbers: 5/1 melee and 6/2 missile in FUN_0057c6c0, 9/10 kill, 11 in FUN_0057c1e0).

The unit that owns the list is the **first stack arg** (the defender for 1, 2, 3, 11; the attacker for 5, 6, 7, 9).

Call sites (V read, by `sk_stateev_xref.py 5c0c30`):

| Event | Site | Function |
|---|---|---|
| 11 absorbdamage | 0x57c2fc | FUN_0057c1e0 (resistance pass) |
| 5 / 6 / 1 / 2 | 0x57c7e3, 0x57c7cc, 0x57c7f3 | FUN_0057c6c0 (damage apply; SKARMOR) |
| 10 / 9 | 0x57c95e / 0x57c970, 0x5a6aeb / 0x5a6aff | FUN_0057c6c0 end; a second kill site |
| 7 then 3 | 0x57d60d, 0x57d61f | FUN_0057d4f0 (melee swing resolution) |
| 3 | 0x57bcad | FUN_0057b7d0 (weapon / state added damage, skipped when record +4 has 0x20) |
| 12 | 0x57e58a, 0x57e979 | level up area (I) |
| 13 | 0x5a72cd | outside the Events.txt rows I know of (U) |
| 0 | 0x5ae088, 0x5ae0fc, 0x5b02ca | missile hits (event 0 hitbymissile, I) |

## Q3. Where event 3 fires, and Shiver Armor on a miss (V read 0x57d4f0..0x57d683)

FUN_0057d4f0 (ECX game, EDX attacker, stack: defender) resolves one queued melee swing:

1. It finds the attacker's queued hit record (unit +0xac list, keyed on both units' type and id) and copies its 0x70 bytes. If no record matches it returns.
2. `FUN_00622c40(attacker, defender, 1 or 3)`: the **reach test** (distance FUN_00641530 against reach FUN_00622870 + extra, then the line check FUN_00622aa0, flag 0x804). If it fails: no damage, no events, only FUN_0057c9f0 (the swing is dropped). 3 when the attacker is a player, 1 otherwise: SKMELEE's reach rule (S sk-melee-trace Q1).
3. **Hit** (record +4 bit 0 set, the to-hit was decided earlier): FUN_0057c6c0 applies the damage (this is where events 5 and 1 and, inside FUN_0057c1e0, 11 fire), kill credit, FUN_0057d3d0.
4. **Both hit and miss now fall through to the same point**: event **7** (domeleeattack) then event **3** (attackedinmelee), both owned by the defender argument (the first stack arg, Q2) with the attacker and the record, then on a hit FUN_005d10c0 (on-hit extras) and FUN_0057cee0 (overlay), then FUN_0057c9f0.

So **Shiver Armor's cold nova (event 3, FUN_005cad40, SKARMOR Q3) fires on a miss** (V read): the miss path (record bit 0 clear) jumps straight to the event calls (0x57d595 je 0x57d5fd). It fires **after** the physical damage of a hit, never before, and is skipped when the swing is out of reach (step 2). A second site, FUN_0057b7d0, fires event 3 too unless record +4 carries 0x20; that is the flag Shiver's own 0x4021 record sets, so a Shiver strike does not re-trigger Shiver (V read; whether FUN_0057b7d0 is the missile-hit or the melee add-on path: **U**).
What the unresolved "who is who" labels mean: in the stack order pushed at 0x57d600 (record, EBX, ESI) the first arg is ESI (the defender argument). Tag V read for the order, I for calling ESI the wearer.

## Q4. Where event 11 fires and Energy Shield against resistances (V read 0x57c1e0..0x57c3d0)

FUN_0057c1e0 (ECX game; EDX attacker; stack: defender, record) is the per-hit damage finishing pass that FUN_0057c6c0 calls:

1. Stats 0x22 and 0x23 (damage reduction, magic damage reduction) of the defender, scaled to 256ths, become pre-reductions applied by the later pass.
2. FUN_0057c060 returns a percent (100 when nothing applies; other values 0x11, 0x19, 0x190 / 0x258 and one read from a record field depending on the unit kinds and flags; which case is which: U, a PvP / difficulty style damage scale, I). When it is not 100, every damage field of the 12-row table at 0x732980 that is above 0 is scaled by it (FUN_00483360 mul-div by 100).
3. **Event 11 fires here** (0x57c2fc), unit = defender, other = attacker, record = the hit record, so Energy Shield's FUN_005ca6c0 (SKARMOR Q5: min(field x calc1 %, mana x 16 / calc2) per field, mana paid in 256ths) sees the damage **after step 2 and before any resistance**.
4. Then FUN_0057c140 clears freeze / stun when the defender has state 0x99 or halves them with 0x76, and clears +0x2c / +0x18 fields on states 0x85 / 0x83.
5. Then for each of the 12 rows (rows with a flag at +0x1c skipped for non-players) **FUN_0057bf80** does the resistance: field value, resistance FUN_0057be00, pierce / absorb handling, damage reduction (FUN_0057bf10), through FUN_005347d0 / FUN_005347e0 (the per-row result and the log). The totals go to +0x4c (sum of +8, +0x10, +0x1c, +0x20, +0x24, +0x28; plus +0x38 for a player).

**Answer:** Energy Shield absorbs **before resistances**: its percent and mana limit apply to the raw (post-percent-scale) damage, and the resistances then work on what ES let through. Bone Armor / Cyclone Armor absorbs (their events) share the same event 11 call, so they run in node-list order (the order the states were created; U which comes first). Poison is not a row of the absorb (SKARMOR Q5) and so never reaches it.

## Deviations d2d vs game.exe

| ID | Rule | Tag | Differential |
|---|---|---|---|
| D1 | Shiver Armor on a miss: game.exe fires event 3 on a miss (Q3). d2d's Shiver is not modelled at all (SKARMOR D2-D4 era); no event 3 on any outcome | V read (game), S (d2d, SKARMOR) | none; to build: a d2d test mode that resolves one swing as a miss and prints the striker's cold damage; oracle would need `sk_armor.py`-style native run of FUN_0057d4f0 with a record whose bit 0 is clear and FUN_005cad40 hooked, U until built |
| D2 | ES vs resistances: game.exe absorbs before resistances (Q4). d2d's ES (SKARMOR D3 / D4) order not read in this trace | V read (game), U (d2d order) | yes, could be built: run FUN_0057c1e0 natively on a record with a state-11 node for a fake ES, oracle = resisted totals (+0x4c); needs the 12-row table dumped (below) |

## Bot findings

None new. SKARMOR B1 / B2 (bot does not model Shiver / ES; internal/combat/necro.go:40) stand; with Q3 the Shiver benefit also arrives on swings the monster misses.

## Not traced (U)

- Skills.txt column at +0x44 and which damage row is record +0xc; which skills carry the byte 1 (Q1). To dump: print Skills rows +0x44, +0x54, +0x138 and the 12 rows of 0x732980 (field offset, flags) with a small emulator script like `sk_armor.py` loaders.
- Order of the event-11 nodes between Energy Shield and Bone / Cyclone Armor; what FUN_0057b7d0's event 3 site is exactly; events 12, 13.
- No game.exe function was run: every claim is V read.

## Drill recipe (lab, U to confirm)

Per `docs/forkdbg.md` / `docs/forkcheats.md`: give a Sorceress Shiver Armor (88) and a monster with a low to-hit; swing it at the wearer until a miss; read the monster's life and the `debug states` / missile list: a cold hit on a miss confirms D1 / Q3. For ES: wear ES, take a fire hit from a monster, vary the wearer's fire resistance with a stat set and compare the mana drain against the formula in Q4; equal drain at different resistances confirms "before resistances".

## Matrix-cell updates (docs/research/sorc-skill-matrix.md, proposed)

- 3.11 Energy Shield (a): add "event 11 fires in FUN_0057c1e0 after the percent scale and before the 12-row resistance pass (FUN_0057bf80): ES absorbs before resistances [V read, sk-state-events-trace Q4]".
- Shiver Armor row (rules): add "event 3 fires on a hit AND on a miss, after the damage, from FUN_0057d4f0, when the target is in reach (FUN_00622c40); not re-triggered by its own record (flag 0x20) [V read, Q3]".
- sk-melee-trace Q1 pre-events row: "FUN_005d3ac0 is the attacker's state bonus into record +0xc on hits only; not an event dispatcher (Q1)".

## Reproduce

`tools/emu/run-on-worker.sh sk_gearar_dis.py 5d3ac0 5d3680 5c0c30 57d4f0 622c40` (disassembly), `sk_stateev_xref.py all` then `sk_stateev_xref.py 5c0c30` and `sk_stateev_xref.py func 57c2ee`, `mpq-table.sh events`.
