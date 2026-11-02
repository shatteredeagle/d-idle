# What a zero-damage hit still triggers: game.exe 1.14d vs d2d (agent SKZEROHIT, 2026-11-02)

> status: research, static answer (first result). Question (board item SKZEROHIT, source sk-resist-trace.md section 10 and Q9): a hit that deals no damage (an immune target, a full block, a rounded-down roll) may still wake or alert the monster, fire the on-hit effect of an enchanted monster and knock back. Which of these does game.exe do, where, on what condition, and does d2d match? Tags: V read = read in the disassembly of the read-only game.exe copy on the worker host (`tools/emu/mai2_dis.sh ADDR COUNT`, plus `tools/emu/skzh_table.py` for one data table); D = 1.14d patched MonStats (`mpq-table.sh monstats`); S = an existing proof in this repo; I = inference; U = unverified. NOTHING here is V run: no emulator differential was built (section 6 says how). Offline, no game, no Go changes, nothing pushed.

## 1. Verdict in six lines

1. **Every landed-hit effect runs through one function, FUN_0057cee0, and it is called whether or not the record holds damage.** The missile hit FUN_005ad730 calls the damage function FUN_0057c6c0 only when record flag 1 (damage) is set, then calls FUN_0057cee0 unconditionally (0x5ad8ba..0x5ad8d3). V read.
2. **Knockback does not look at damage.** Record flag 8 is rolled in FUN_005ad730 from the missile row's knockback percent (0x5ad7d4..0x5ad7ef), or set by FUN_005a54f0 for a Sand Leaper (class 78, D) hit by a flag-4 record that is not frozen. FUN_0057cee0 acts on flag 8 (monster: mode 0xd, 0x57d01d..0x57d03a) with no test of the damage total; a monster whose MonStats2 has no knockback bit (FUN_0046c140 bit 0xd) has flag 8 turned into flag 4 (0x57cfcb..0x57cfea). So an immune target is still knocked back (V read; the immune value only zeroes the damage fields in FUN_0057c6c0, the flags were fixed before).
3. **The get-hit does depend on damage**: FUN_0057cb00 answers "no recovery" for a record total under 0x100 (one point), for a frozen unit and for a record that is all poison (S pr-get-hit-rule-proof.md; the three gates re-read here at 0x57cb0b..0x57cb35). Zero damage = no get-hit, with one flag-4 exception: a stunned monster (state 0x15) takes the get-hit anyway (0x57d0b6..0x57d0d1, S).
4. **No wake / aggro in this path.** Neither FUN_0057cee0 nor FUN_005a54f0 writes an aware flag or a target; the wake rule stays a think plus the target search (S mon-wake-trace.md, "being hit never sets the aware flag"). A zero-damage hit wakes nothing that a damaging hit would not. V read for this path, U for the other callers of FUN_0057cee0 (melee `0x56ba73`, `0x57d5c7`, `0x5a6e82`; not read here).
5. **The on-hit effect of enchanted monsters exists for ONE enchant: Lightning Enchanted.** FUN_005a43a0 (the hit event, type 4) walks the monster's nine enchant bytes (monster data +0x1c) through a handler table at 0x73c0b8 (6 slots per enchant id); slot 4 is filled only for id 17 (handler FUN_005a2ba0, D from `skzh_table.py`). Its handler fires the bolt routine FUN_005a29a0 only when the unit's mode is NOT 3 (get-hit) at that moment. FUN_0057cee0 calls the event after setting mode 3 on a real get-hit and WITHOUT setting it on a no-recovery hit (0x57d0d1..0x57d0ed vs 0x57d102..0x57d122). So a zero-damage, no-get-hit hit lets the bolts fire; a hit that puts the monster into get-hit does not. V read (the mode value 3 and the table), I for the in-game consequence (no run).
6. **d2d**: knockback on a landed hit is not gated by damage either (fight.cpp land, then `knock_back`; same Sand Leaper special case, 2 vs 1 reach); the Lightning Enchanted on-hit bolts were not found in d2d (grep of components/ for enchant on-hit logic: only the damage-percent enchants, ai.cpp:2799). U for whether a zero-damage hit reaches `land` at all (immune path not read).

## 2. Questions

| Q | Answer | Address | Tag |
|---|---|---|---|
| Q1 | Is FUN_0057cee0 reached with a zero total? | Yes: FUN_005ad730 tests flag 1 only before FUN_0057c6c0, then calls 0x57cee0 always | 0x5ad8ba..0x5ad8d3 | V read |
| Q2 | Where do the record flags 4 (recovery), 8 (knockback), 0x4000 (no-recovery) come from, and do they depend on damage? | missile hit: flag 4 when the missile row's byte +4 meets mask 0x6ce280, 0x4000 for mask 0x6ce284, flag 8 by a d100 roll against row byte +0xae; all before any damage is computed. A block / avoid result (FUN_0057dfb0 bits 0x1f) with no recovery class and no knockback clears flag 1 and sets 0x4000 | 0x5ad7aa..0x5ad869 | V read |
| Q3 | Monster branch of FUN_0057cee0, in order | dead / mode 0 / mode 0xc: out; flag 8 on a non-knockback type becomes 4; FUN_005a54f0; flag 2 -> FUN_0057ccb0; flag 8 -> mode 0xd; flag 0x10 -> mode 6 or last mode 0x13; flag 4 -> stunned or FUN_0057cb00 == 0 -> mode 3 (get-hit) then event 4; else (and flag 0x4000) the no-recovery tail: FUN_0064c040, unit +0xc4 |= 0x8000, last mode 0x13, event 4; flag 1 alone: if total > 0 and stat 0x160 differs by > 4 from FUN_005a5650, only the +0xc4 bit | 0x57cf8e..0x57d183 | V read |
| Q4 | Which hit-event handlers exist? | slot 4: id 17 only (FUN_005a2ba0 -> FUN_005a29a0). The strike event (slot 5, FUN_005a43b0, attacker side): ids 19, 27, 29. Other slots (0..3) belong to other events (spawn, state change, not traced) | 0x73c0b8 table | D (table), V read (handlers) |
| Q5 | Lightning Enchanted bolt rule | FUN_005a29a0: fires only if the game frame is 10 or more past the unit's last-fire frame (monster data +0x18), then stores the frame; level = stat 0xc / 2 (min 1), skill id 0x5c (the bolt skill; its row not looked up, U); the handler also needs flag byte data+0x16 bit 3 (writer not found, U) | 0x5a29a0..0x5a2a52, 0x5a2ba0 | V read, U (flag, skill row) |
| Q6 | Wake or alert on a hit? | Not in FUN_0057cee0, FUN_005a54f0, FUN_005a43a0 | | V read (this path), S (mon-wake-trace.md) |
| Q7 | A zero total with flag 1 only | does nothing but the +0xc4 test (needs total > 0) | 0x57d130..0x57d17e | V read |

## 3. Deviation list (d2d vs game.exe)

| D | game.exe | d2d | Tag | Differential |
|---|---|---|---|---|
| D1 | Lightning Enchanted fires a bolt when hit and not entering get-hit, 10-frame cooldown (Q5) | no on-hit bolt found for the enchant (grep: only damage-percent enchants) | I (grep, no run) | Buildable: run FUN_005a43a0 natively with a unit whose enchant list holds 17 and mode 3 / not 3, hooking FUN_005a29a0 to record the call; d2d side needs the monster hit path read |
| D2 | Knockback independent of damage; converted to a get-hit when the type has no KB mode | `knock_back` after `hurt`, skipped when KB or GH direction counts are 0 (fight.cpp:912) | I | Buildable with `hit_rule.py`'s harness (FUN_0057cee0 natively, record flag 8 with total 0, both MonStats2 bit values); needs d2d's `land` entry fed a zero blow |
| D3 | Immune or fully resisted blow still reaches the flags (Q1) | whether `land` runs for a blow resisted to 0 not read | U | read fight.cpp:850..910 and the immune path |

## 4. Bot findings

None filed: not traced against the bot (the bot's immune check `tg.Immune` is covered in sk-resist-trace.md B-list). B1 (proposed, I): any bot logic that treats "0 damage dealt" as "monster not disturbed / will not knock back" is wrong for knockback and for Lightning Enchanted monsters (they answer a no-recovery hit with bolts). Not located in the bot source here; limit.

## 5. PR candidate

No PR candidate: nothing is V run, so no differential with mismatches before and zero after exists.

## 6. Drill recipe (lab fork verbs, to turn U into V)

1. Differential (emulator, no game): extend `tools/emu/hit_rule.py`'s approach: call FUN_0057cee0 with a monster unit, record flags {1, 4, 8, 0x4000} and total {0, 0xff, 0x100}; hooks as in pr-get-hit-rule-proof.md plus FUN_005a43a0 recorded (and FUN_005a29a0 for D1). Oracle: the set of mode calls (mode 3 / 0xd / none) and whether event 4 reached FUN_005a29a0. Compare with the d2d rule in a small C++ harness as hit_rule_harness.cpp does.
2. Lab (docs/forkdbg.md, docs/d2d-fork.md, docs/forkcheats.md): spawn a Lightning Enchanted monster and a cold- or lightning-immune target of the same kind; hit it with a bolt skill the type is immune to; log the missiles it fires (expect bolts at most every 10 frames, none while it is in get-hit) and whether it flinches or slides. Compare game vs d2d trace.
3. To settle the flag +0x16 bit 3 and the skill row of id 0x5c: read the writer by hardware watch in the lab, or read MonStats / boss-mod init (FUN_005b2a00 chain) with `mai2_dis.sh`.

## 7. Matrix cell updates (sorc-skill-matrix.md; proposed text)

| Row | Cell | New value |
|---|---|---|
| resist, immunity rows (Fire Bolt .. Lightning) | zero-damage hit | no get-hit (S), knockback still applies (V read, sk-zero-hit-trace.md Q3), no wake (V read); Lightning Enchanted bolts fire on a no-recovery hit (I, D1) |
| rank SKRESIST follow-up | status | SKZEROHIT static answer written; differentials D1 / D2 buildable |

## 8. Limits not covered

Melee callers of FUN_0057cee0, the player branch, FUN_0057ccb0 (flag 2) and the 0x10 branch, the event table slots 0..3, the strike event (ids 19 / 27 / 29), Sanctuary / curses, the d2d immune path. Dead end: the data+0x16 bit 3 writer was not found by a text search of the dump (byte-register writes may hide it).

## Reproduce

```sh
tools/agentrun.sh <agent> --ttl 120 -- labrun the worker host -- bash tools/emu/mai2_dis.sh 57cee0 140   # also 57d073, 57cb00, 5a43a0, 5a4270, 5a2ba0, 5a29a0, 5a54f0
tools/emu/run-on-worker.sh skzh_table.py        # the 0x73c0b8 table, rows per enchant id (D)
tools/emu/run-on-worker.sh mpq-table.sh monstats | awk -F'\t' '$2==78 {print $6}'   # SandLeaper
```

## Shared-file lines for the merger

See openspec/changes/skzerohit/merge-lines.txt.
