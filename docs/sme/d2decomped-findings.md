# d2decomped notes harvested for the bot (read-only study)

Agent D2DHARVEST, 2026-10-30. Source: https://github.com/psi29a/d2decomped (GPL-3.0-or-later), shallow clone read only (nothing built, run or executed; no code or long passages copied into this repo). Everything below is my own summary with file citations relative to the clone: `re/X.md` = `docs/research/re/X.md`. Target: classic LoD 1.14d (the project's ground truth is game.exe 1.14d; D2R is out of scope and not mentioned by the notes).

## 0. How far to trust this

Tags used here:
- **D2D-V**: the notes state it as traced in game.exe 1.14d (function named, often diffed against game.exe's own code run in an emulator, e.g. drops 100000/100000 equal in `re/drops.md` "Checks", hire offers and store fills in `re/npc-menu.md`, `re/store.md`, Act 1-3 monster thinks in `re/monster-ai.md`).
- **D2D-U**: the notes say untraced, approximate ("ponytail"), or it is my inference from them.
- Caveat for every row: this is a reimplementation's research notes, a strong secondary source, not the original. Where they were checked live against a real 1.14d host they say so ("Checked live ..."), otherwise "traced" means read from the decompile. Our own live probe stays the acceptance test. `re/todo.md` row 1 admits nobody has played an act end to end by play; the quest tests set up each quest with debug verbs. `re/unverified.md` lists what they took on trust from other projects (d2s layout, DCC, etc.).
- Function names look like `FUN_005359f0` = game.exe VA (base 0x400000, same as our `mem.Base`).
- I also read a few C++ files (`components/rules/monsters.hpp`, `components/game/world.cpp`, `components/game/npc_menu.hpp`) only to disambiguate a note; those are cited as `code:` and are d2d's behaviour, not game.exe's.

Source of the project itself: `README.md` (all five acts and 27 quests play; "checked against the original"), `LEGAL.md`, `docs/PLAN.md`.

---

## 1. Death

All from `re/combat.md` "Death" (lines ~273-309) unless noted. D2D-V unless marked.

**At the moment of death (mode DT, `FUN_00580ec0` -> `FUN_00535ab0`)**
- XP penalty (`FUN_005359f0`): DifficultyLevels `DeathExpPenalty` = **0 / 5 / 10 %** (Normal / Nightmare / Hell) **of the span of the current level** (`exp_next[level] - exp_next[level-1]`), not of total XP, and never below the level's start + 1. D2D-V. (code: `world.cpp death_penalty` hard-codes {0,5,10}, and skips the penalty at level 1.)
- Gold (`FUN_005357d0`): loss = min(clvl, 20) % of **purse + stash together**; in single player (game type 3) the loss never takes the total below **clvl x 500** and only comes out of the purse. Then the **rest of the purse falls on the ground at the death spot** (`FUN_00535510`, split into piles by `FUN_0055a090`), the purse becomes 0, stat 175 `goldlost` records the loss. D2D-V for the text; my reading of the clamp wording is D2D-U but is confirmed by d2d's own code (`code: world.cpp`: lost = min(clvl,20)% x total; if total-lost < clvl*500 lost is reduced; lost <= purse; purse-lost dropped as a "gld" item; purse = 0).
  - Consequence for the bot: after a death the carried gold is not gone, it is a ground pile at the death place (field state, resets on quit). The stash is only touched by the 'lost' part (and in SP that part comes from the purse only, so effectively the stash is safe in SP). To verify live: purse N, die, read purse (0?) and ground gold at the spot.
- Experience on the corpse: the corpse's experience stat = **75 % of what the death took**; picking the corpse up gives it back. D2D-V.

**Corpse (mode 0x11 played: `FUN_00581020` -> `FUN_0057fca0` -> `FUN_0057f700`)**
- A corpse unit is created, the player's class in mode 0x11, **at most 16 per player** in memory. D2D-V.
- **Contents: everything worn (body locations 1..12) plus the item on the cursor ("in hand")**. Nothing from the inventory grid, belt, stash or cube is mentioned, i.e. those stay with the living character. D2D-V (the note is explicit about what goes on it; absence of the others is an inference, D2D-U; code: `world.cpp make_corpse` moves only `kEquipped` items and the held one).
- Death is **not** a quit: `re/net-packets.md` ~line 975 (answered live 2026-10-05 against a real game.exe host): slain message, death anim, then ~1.5 s later the corpse appears as a new player unit (0x8e add, our worn items 0x9d onto it); respawn is C->S 0x41 which puts the player in the act's town start with full life/mana/stamina (`re/combat.md` "Resurrecting", `FUN_0054c0e0`; hardcore ends the game instead). D2D-V (live-checked wire sequence).

**Picking the corpse up**
- By **clicking it** (interact): C->S 0x13 with unit type 0 and the corpse's unit id -> server `FUN_00548b00` "player (corpse / trade)" (`re/net-packets.md` interaction table ~line 763; `re/net-packets.md` ~978 "0x13 type 0 on its unit -> 0x8e 0, the items back"). Then `FUN_0057fb70`: XP back, each item to its original body slot if that slot is free **and its requirements are met**, otherwise into the inventory; corpse vanishes when empty. D2D-V. Required click distance for the corpse is not stated (objects: < 0x33 subtile units; item pickup: within 5; NPC talk <= 6): D2D-U. Whether plain walking over it works is not stated; the notes only describe the click path (d2d draws corpses lying and clickable, `code: world.cpp`).
- Items going back to free slots first means: if you picked up a replacement weapon/armour after respawn, that slot is occupied and the old piece goes to the inventory (if the inventory has no room: not stated, D2D-U).

**Quit and start a new game with a corpse pending**
- The .d2s keeps a corpse list after the player's items: `"JM"`, u16 count, then per corpse **12 bytes (u32 ?, u32 x, u32 y) and its own item list**; `FUN_00533850` makes them again when the player joins (`re/combat.md` ~298; layout confirmed by `code: components/d2s/d2s_items.hpp parse_corpse` on 19 real saves, `re/unverified.md` ".d2s item bitstream"). D2D-V that it persists and is rebuilt at join.
- **Where it appears in game.exe is NOT traced**: d2d "lays a saved corpse by the town's start" and does not read the saved x/y (`re/combat.md` ponytail, `code: world.cpp lay_save_corpse`). D2D-U. The live notes in our `docs/deathspiral.md` (corpse stays in the field area) are the better evidence; a probe should compare the saved x/y with where the body shows after reload.
- **Multiple corpses / oldest rule**: not documented as a game rule. In memory 16 per player; the d2s writer in d2d writes only the **latest** corpse ("The corpse list holds one", `code: world.cpp:391`, `re/combat.md`). Whether game.exe writes one or all, and which one survives, is D2D-U. This is exactly our open "oldest corpse" item: needs a live probe (die twice, quit, reload, count bodies and read the JM count in the .d2s with our d2s tooling).
- `re/net-packets.md` ~698: S->C 0x74 "assign corpse" with flag 0 does nothing on the client in all 49 logged cases (multiplayer corpse-loot permission). Irrelevant for single player.

---

## 2. Mercenaries

Files: `re/npc-menu.md` (hire/resurrect/identify), `re/hireling.md` (merc panel, wearing), `re/merc-ai.md`, `re/quests-act1.md` "Kashya's reward: a free hireling", `re/combat.md` (potions), `re/act1-gaps.md`.

**Hire window** (`re/npc-menu.md` "Hire", `FUN_004b5c60`; D2D-V, offers diffed against game.exe: 720 offers, 400 lists)
- A 490x350 NPC text window. Header string 0xd24 "Your Gold: %d     Hire which Mercenary?" (gold shown = carried + stash). A list widget 490x280 at ((W-490)/2, H/2-160) with rows 0x23 (35 px) high, **eight rows visible, scrolls (mouse wheel)**, then a "cancel" button. One row per offer: name, " - ", then Lvl, Life, Def, Cost, each ": %u". Click a row = hire (C->S 0x36: u32 NPC id, u16 **offer index**, not merc id, `re/net-packets.md` ~785).
- The server's list (`FUN_00576070`): one entry per name of the seller's first hireling.txt row (NameFirst..NameLast), seeds from the game's proxy seed, then **ten offers** picked at random (with wrap past taken ones). The list is regenerated when no offer is left. Names/seeds are fixed for the game; a quit and new game gives a new list. (D2D-V.)
- Who can hire: Kashya (Act 1 rogue), Greiz (Act 2), Asheara (Act 3), Qual-Kehk (Act 5). Tyrael (Act 4) **only resurrects** (his menu has no hire entry; `code: npc_menu.hpp`, `re/npc-menu.md` "The act hire NPCs").
- **Kashya's hire entry appears only above character level 7** (`FUN_004b6410`; "not quest-gated": it reads only stat 0xc), i.e. clvl >= 8. This matches `docs/sme/mercenary.md` (three sources say 8). The free Blood Raven reward bypasses the menu. D2D-V.

**Offer stats and price** (`FUN_006637f0` from the offer's seed; D2D-V)
- Row set: hireling.txt rows of that act, difficulty and version (100 = LoD) that share the first row's Level; one is picked.
- **level = clvl - 5 + rand(5), at least 2.** So in 1.14d the offered level follows the player (settles the U in `docs/sme/mercenary.md` section on hire level: no 1.10-style fixed ranges). d = level - row.Level.
- life = HP + HP/Lvl x d (min 40); str/dex = base + (per-level x d >> 3) (min 10); defence = Def + Def/Lvl x d; damage min/max + (Dmg/Lvl x d >> 3); exp = (level+1) x Exp/Lvl x level^2.
- **cost = Gold x (15 d + 100) / 100, never below Gold** (Gold = hireling.txt column; our `hireling.tsv` has the table). So each merc level above the row's base level adds 15 % of the base price. Payment source for hire is not stated (revive pays carried first then stash, `FUN_00576d90`): D2D-U for hire.
- Stats come from MonStats **Normal** columns whatever the difficulty; ToBlock/ResDm/ResMa are blank for all five hireling rows (`re/merc-ai.md` "Its gear"). D2D-V.

**Free Blood Raven (Rogue) reward** (`re/quests-act1.md` ~1091-1117; `re/npc-menu.md`; D2D-V)
- Trigger: Kashya's line 92 `A1Q2SuccessfulKashya` heard while quest-2 bit 1 (reward due) is set. The quest sets bit 0 first, then calls `FUN_00579180(player, Kashya)`.
- It binds **the first valid unhired offer in Kashya's current list**, with the same seed/level/stats the hire screen shows (so level = clvl - 5 + rand(5) at the time you claim), **no gold taken**.
- **No merc is granted if the player already has one**: in LoD *any* merc, alive **or dead**; classic only a live one. There is **no deferral**: bit 0 is already set, so the reward is lost for that difficulty.
- The offer list must exist: it is created when Kashya's menu is built (client asks on opening her menu, packet 0x38 kind 3). Talking to her first creates it.
- Also available at Asheara (Act 3 Blade of the Old Religion: Asheara 589, bit 7, "a free mercenary", `re/quests-act3.md`).

**Resurrect** (`re/npc-menu.md` "Resurrect", `FUN_004b1dd0` / `FUN_00579c00`; D2D-V)
- Menu line (only while the client knows of a dead merc) 0x58a8 "Resurrect %s: %d", inserted ahead of "hire" or before "cancel". NPCs: Kashya, Greiz, Asheara, **Tyrael**, Qual-Kehk, and only the NPC currently being talked to. Click = C->S 0x62 (5 bytes, NPC id).
- **Cost = merc level^2 / 2 x 15, at most 50,000** (`FUN_006637b0`). (Operator order as written; our `docs/sme/mercenary.md` already has `min(int(level*level/2)*15, 50000)`.) Paid carried gold first, then stash; refused if both together are short. Result: full life, merc put beside the player, states cleared; items kept (flag 0x10000 cleared, nothing deleted).

**Merc equipment** (`re/hireling.md` "Wearing", C->S 0x61, `FUN_0054d430`; D2D-V)
- Items must be identified (flag 0x10) and not broken (0x100). Allowed: any helm or body armour, plus per merc row: rogue (271) a bow; desert guard (338) a spear or polearm; iron wolf (359) a shield or a one-handed sword; barbarian (560) a one-handed axe, (561) a sword, both also a primal helm.
- **Yes, requirements apply to the merc's own stats**: `FUN_0062eaf0` compares the merc's strength, dexterity and level with the item's requirements (not the player's). Not applied (d2d gap): ethereal / "requirements -x%" subtleties.
- With nothing in hand a click on slot 1..10 takes the merc's item into the hand. Slots: head, torso, left hand (shield), right hand (weapon). A merc can never block unless an Iron Wolf wears a shield (`re/merc-ai.md` "Its gear").
- The merc's damage with a one-handed weapon uses only the weapon (an Act 3 sword merc hits for its own damage only); a two-hander adds the base. AR = stat 19 + dex x 5, scaled. Defence stat 31 + dex/4. D2D-V.

**Potions to the merc**
- Shift + belt key (C->S 0x26 with the shift flag, `FUN_00562390`, `re/combat.md` "Drinking a potion"): hands the potion to the merc. **Only item types 0x4c (hpot), 0x50 (apot, antidote) and 0x51 (wpot, thawing)**; with an item on the cursor, the same three types by clicking the merc panel (`re/hireling.md`). D2D-V as written.
- **Conflict to settle live**: `docs/sme/mercenary.md` says rejuvenation works on the merc; the notes list only hpot/apot/wpot and say "only". Whether `rpot`/`rvs`/`rvl` are in that type chain (type equivalence) is not stated. Probe: Shift+belt-key a rejuv on the merc and read the potion count and merc life. Mana potions are not accepted (matches ours).
- The belt column is addressed by **column, not item id** in 0x26 (`re/net-packets.md` ~803).

**Merc AI** (`re/merc-ai.md`, MonAI 61 "Hireable", think `FUN_005e52d0`; D2D-V, native-checked 30,000 cases)
- Distance to owner = (short + 2 x long)/2 after subtracting size. **> 100: teleports** into the owner's room; **> 24: runs**; > 16 and owner walking: follows at walk; else stands/wanders (6 in 128 per think wander when blocked on a door tile).
- **Never attacks in town**; outside town it takes a target from `FUN_005ddc30` under 25 subtiles away (enemies within 0x31, in line of sight, threat >= 2 first, nearest).
- Skill chance: melee mercs 98; else base +40 + 2 x level (cap 95). Shooters (aip1 = 0) back off 4 when a foe is under 4 away half the time. Skills come from hireling.txt at its level; rogue fires MonStats Skill1; Act 2 guard swings A1 in melee.
- Auras (Act 2 Prayer/Defiance/Blessed Aim/Thorns/Holy Freeze/Might) are **not built** in d2d (count as running, do nothing); the merc has no mana (D2D-U gap).
- **Following through portals, waypoints, acts**: not described. The notes only have: standing in another spawn area than the owner -> mode 0 follow at pace 60; and `FUN_00579aa0` puts a resurrected merc by the player. Keep our live evidence (docs/merc-farm.md).
- Saved in the d2s: after the corpse list, `"jf"` then, with a hired merc, a `"JM"` item list of what it wears (`re/hireling.md`, `code: d2s_items.hpp parse_merc_items`).

---

## 3. Quests: persistence and the quit-safety table

### 3.1 Mechanism (all D2D-V unless marked)
- Quest state **is the save**: each difficulty has 96 bytes = 48 u16 words, 16 flag bits per quest, in the .d2s at block `"Woo!"` (header 0x14F: `Woo!`, u32 6, u16 298, then 3 x 96 bytes: flags start **0x159**); bit `q*16+b` (`re/quests.md` "Flags", `re/npc-talk.md` d2d section). The bits are set in the player's live flag buffer at the event (kill, talk, pickup, operate), so **a Save and Exit right after the event writes them**; nothing else defers them. (The server's buffer is `*(player data +0x10 + difficulty*4)`; a client copy is also kept, see section 8.)
- Common bit meaning (same for most quests): 0 done (reward taken), **1 reward due**, 2 "given" (NPC spoke), 3/4 under way, 13 "took part in this game", 14 "done by someone else", 15 closed (tested only; set on the game's copy when the quest was done in an earlier game).
- **Per-game quest records are rebuilt at join from the bits plus carried items** (`re/quests.md` "Join"; each quest has a join callback `+0xd4`): bit 4 -> state 3 log 2; bit 3 -> state 3 log 1; bit 2 -> state 2 log 1, skipped when bit 0 or 15 set. The record's own data block (kill counters, "stand used", object modes, who is on lists) is **not saved**. Quest items in the pack are re-counted at join (Cain scroll, Malus, staff parts, Hellforge soulstone).
- **What resets on quit** (units, objects, portals live only in the game): monsters respawn (Duriel is alive again), ground items and corpse drops are gone, town portals gone, level objects restart (chests/levers/orb/forge modes; a few are restored from a quest bit at join, noted per row below), "this game only" flags (Den's light, `re/quests.md` "The Den lights up"; the log's done-animation bit 12 is client-only, not saved).
- Reward dialogue works after reload because the NPC offers the reward block while bit 1 is set (`re/quests.md` "Messages": "block 3 while the reward is due").
- **Autosave**: in single player the only forced save the notes find is at **Baal's death** (`FUN_0052e2a0`, every player; `re/quests-act5.md` "Baal's death") and classic Diablo. The 75-second step after Diablo does nothing in single player (`re/quests-act4.md` "The ending"). Everything else needs Save and Exit; a crash or kill loses the events since the last save. U for other autosave points (act change, waypoints): not documented.

### 3.2 Row by row against `docs/plan/plan-v2.md` section 10.5 (class A safe / B unsafe / C unsure)

Verdict key: CONFIRMS, CORRECTS, REFINES, UNSETTLED (needs live probe). Each cites the notes; all are reads of the decompile, none replayed by a human end to end.

**Act 1**
| Plan row | d2d notes | Verdict |
|---|---|---|
| Den of Evil A | Last monster dies: bits 13+1 to participants, state 4; Akara line 76 with bit 1 gives the reward (+1 skill point, quest 41 opened); join rebuilds states (`re/quests.md`) | CONFIRMS A |
| Blood Raven A, free Rogue | Death sets bits 13+1 for players in her room or a neighbouring room (and party in Act 1); Kashya line 92 with bit 1: bit 0, free merc (`re/quests-act1.md` "Blood Raven", "Kashya's reward") | CONFIRMS A. ADD: reward is **forfeited if you already own any merc (even dead)**, no retry. Need list created by opening Kashya's menu first. Kill must be near you (room rule) |
| Cain scroll pickup A | Tree gives scroll (`bks `), state 4; join counts `bks`/`bkd` carried (state 4 / state 5) and **losing the scroll rewinds the quest** (`FUN_00592c80`); bits 2,3 saved (`re/quests-act1.md` "Join", "Inifuss tree") | CONFIRMS A (the scroll must stay in the pack) |
| Cain stones -> Tristram portal B | Stones done sets bit 4 (saved). On join with bit 4 the stone init re-creates the **permanent red portal** when Stony Field's StoneAlpha comes up (stone InitFn 6 `FUN_005935e0`; d2d fixed a bug where a save with bit 4 had no portal) (`re/quests-act1.md` "Tristram portal") | CORRECTS to **A for the portal** (it returns after reload once you walk to Stony Field). Gibbet/Cain: free him in the same session or do it later (gibbet works again while state < 6 and bits 0/1 clear). UNSETTLED live: this is disassembly-read, no one confirmed it on the real game |
| Cain rescued, reward | Rescue sets bit 1 (+13) at the animation end; Akara line 0x76 with bit 1 gives a ring (ilvl 7 magic Normal, 30 rare NM, 60 rare Hell); join does not look at bit 1 | A. Quit before the reward is fine (bit 1 persists); Cain stays out of camp until bit 0 (camp spawn keys on bit 0/15) |
| Forgotten Tower A (no reward) | Countess kill sets bit 0 and 13 **directly at the kill**, no reward talk (`re/quests-act1.md` "Flag bits" quest 5) | CONFIRMS A. Her drops are ground items: loot before quitting |
| Tools of the Trade A | Malus is a pack item; join: bit 2 -> state 2, bit 3 -> state 3; picking it up sets bit 6; Charsi line 163 with the Malus in the pack and **clvl >= 8** gives bit 1 (imbue) (`re/act1-end.md`) | CONFIRMS A. ADD: Charsi's reward line needs clvl >= 8 per the block test (D2D-V), and the imbue is spent on one item pick |
| Sisters to the Slaughter / Andariel A UNSURE | Death sets bits 13+1 (and Act 2 progression unlocked in the client, `FUN_00538680`); join with bit 1 keeps Warriv's line 183 offerable (message block for Warriv, bit 1 -> block 3); hearing 183 sets bit 0. Server `go east` requires **quest-6 bit 0**: so after reload you must still talk to Warriv, then "go east" (`re/act1-end.md` "Move to Act 2") | CONFIRMS A. REFINES: quit is fine but Act 2 needs Warriv's talk first (bit 0), the corpse portal is irrelevant (field state) |

**Act 2**
| Plan row | d2d notes | Verdict |
|---|---|---|
| Radament A only after pick up AND read the book | Death sets bits 13, 1 and 5 and **drops the book `ass` on the ground**; reading the book clears bit 5 and gives +1 skill point (the only reward, `re/quests-act2.md` "Radament's Lair"). Join with bit 0 set, bit 5 still set and **no book in the pack**: bits 0,1,15,5 are cleared and the quest becomes **redoable** | CORRECTS the reason: bit 5 (book due) **is** saved, there is no "read pseudo item". Pick the book up (pack item persists) and read it any time; if you leave it on the ground the reward is lost until the quest is redone in a later game |
| Cube/Amulet/Shaft A | Parts are pack items counted at join; scroll read sets bit 3 etc. (`re/quests-act2.md` "The Horadric Staff") | CONFIRMS A |
| Staff in orifice A/C | Placing sets **quest 10 bits 0 and 13**, staff and parts removed, a timer opens object 100 (stairs) by the orifice; the orifice's room "coming up opens it (mode 2) on data +0xc: the staff placed in an earlier game"; join restores data from quest 10 done (`re/quests-act2.md` "The Seven Tombs") | CONFIRMS A (word advances at placement; the way down is restored after reload). UNSETTLED live (read of code) |
| Summoner C | Death: bits 13+1 for players by him (his room or neighbour); town NPC thanks lines 419..429 with bit 1 give reward. The Canyon portal is made by **reading Horazon's Journal** (record 11, "once"), separate quest (12) | A for the Summoner word. UNSETTLED: notes do not say whether the journal still opens the Canyon portal after a reload (the Sanctuary has its own waypoint, so not a trap) |
| Duriel C (talk to Tyrael first) | Death sets bit 5 for players by him, Tyrael then talks; **join does not consult bit 5** (only bits 2,3,4); Tyrael's line 302 (bits 13+3, Act 3 open) is offered only in state 3, i.e. while Duriel is dead in *this* game; then Jerhyn 442 (bit 3 -> bit 4), Meshif 450 (bit 4 -> done, sail east needs quest 14 bit 0) | CONFIRMS C and the advice: **talk to Tyrael in the Lair before quitting**; if you quit first you must re-kill Duriel (monsters respawn). Bit 3 afterwards persists, so Jerhyn/Meshif can be done in town after any reload |

**Act 3**
| Plan row | d2d notes | Verdict |
|---|---|---|
| Golden Bird / Blade / Lam Esen A | Pack items (figurine, bird, Gidbinn, tome), bits at hand-ins; Lam Esen: 5 stat points at the hand-in; Blade: Asheara 589 free merc, Ormus ring (`re/quests-act3.md`) | CONFIRMS A. Carry the item; Golden Bird's Potion of Life needs bit 5 |
| Khalim's Will C | Orb smashed with the Will: **word 0x12 bits 0,13** set, Will taken; "a3q2: word 0x12 bit 0 breaks the orb at a join" (record table) | CORRECTS toward **A**: the orb is restored broken at join from bit 0, so no need to go down first. Note the Council kill (bits 13+0 / bit 4) is a separate step. UNSETTLED: the table line is terse |
| Mephisto B (Act 4 portal is field state) | Death (players in Durance 3): bits 0xd, 0, 0xb and Act 4 progression; hellgate object (342, level 104 uses record state 2 as its mode) and the bridge; **join (`LAB_005bcc80`): bit 0, 0xb or word 0x17 bit 0 -> record +0xc = 2, "the hellgate open"** (`re/quests-act3.md` "The Guardian"); the notes call the other callbacks "not traced" | CONTRADICTS the plan/user report (B). The notes claim the gate comes back open after reload. **UNSETTLED, high stakes: trust the user's live report until a probe says otherwise** (kill Mephisto on a throwaway, Save and Exit, reload, walk to the hellgate) |

**Act 4**
| Plan row | d2d notes | Verdict |
|---|---|---|
| Izual A | Death: bits 13+1 to players in his room/neighbours; Tyrael 676 with bit 1 -> bits cleared, **2 skill points**; join rebuilds from bits 2..4 (`re/quests-act4.md` "The Fallen Angel") | CONFIRMS A |
| Hellforge B | Hammer `hfh` drops from Hephasto (pack item), soulstone `mss` is consumed when put in the forge (forge mode is record data, **not saved**); the third strike sets bits 13+1 and **drops gems and a rune at the forge** (field) (`re/quests-act4.md` "The Hellforge") | CONFIRMS B. Stay until strike done and rune picked; after bits 13+1 are set, Cain line 680 is the reward |
| Diablo C | LoD: death sets bits 13+0 at once; Tyrael's portal (object next to Tyrael) is made by his talk (line 20000 sets bit 9). **Join clears bit 9 unless Act 5 was entered (word 0x1c bit 0)**, so after reload Tyrael offers the portal again (`re/quests-act4.md` "Terror's End", "The ending") | CONFIRMS A with a refinement: quit before using Tyrael's portal is fine, talk to Tyrael again. Classic (non-LoD) timing irrelevant to us |

**Act 5**
| Plan row | d2d notes | Verdict |
|---|---|---|
| Siege A | Shenk death: bits 1+13 in room/near list; Larzuk 20090 with bit 1: bit 5, state 5; **bit 0 only with Larzuk's socketing** (`re/quests-act5.md`) | CONFIRMS A (the socket reward is a separate pick) |
| Anya A after freed | Thaw at Fana with Malah's potion: bits 13+1; Anya 20136 with bit 1 gives the personalised rare class item, bit 9/10 | CONFIRMS A (freed = the quest bit set; the red portal state is field) |
| Rite of Passage A UNSURE | Last Ancient dead: killer and Summit players of the right level bits 0+13 + XP (1.4M/20M/40M capped at the next level); **stairs (object 547): open when bit 0 or 1 and "won here or the record shut"**, so after reload (record shut) they open (`re/quests-act5.md` "Rite of Passage") | CONFIRMS A (resolves UNSURE; only a level (diff+1) x 20 player is marked) |
| Rescue on Mount Arreat C | Bits 13+1 (and 5/6/7 by the freed count: >= 12 freed) set when **all three cages' prisoners are dead or freed** and the portal opens; fewer freed: bit 4 only; Qual-Kehk 20110 gives runes r07/r08/r09 (3/2/1) | A once the bits are set (the cages and the portal are field state). C only before it |
| Baal A | Bits 0+13, next difficulty opened, gold piles, then **the game is saved** (`FUN_0052e2a0`) | CONFIRMS A (plus a free save) |
| Nightmare/Hell | Quest words are per difficulty; same code; Hellforge rune table rises (r12-r22 NM, r15-r25 Hell) (`re/quests-act4.md`) | CONFIRMS (no split change) |

Bot rule suggestion (unchanged): quit only when the quest word shows the step done and no B row is pending; add three doc-backed checks: pack item still present (Cain scroll, Radament book, parts), Warriv talked before Act 2, Tyrael talked before quitting after Duriel.

---

## 4. NPC menu layout (for click logic)

`re/npc-menu.md` "Building / Layout / Drawing", table at 0x726c48 (48 records), `code: components/game/npc_menu.hpp` for the contents. D2D-V (traced; menu text strings are string.tbl ids).

**Entry order = table order, header first, "cancel" always last** (string 0x102e). Act 1 camp:

| NPC (hcIdx) | Lines after the name header, in order |
|---|---|
| Akara 148 | talk, trade; gets "Reset Stat/Skill Points" (0x2ba0, patched in by `0x4b6da0`) while quest 41 is open (Den reward, or Hell). A special-cased heal string (0xd09, shown by a life check) exists in the menu builder but is not in her record: D2D-U how healing is triggered (our bot already heals by talking/trading, live-proven) |
| Charsi 154 | talk, trade/repair (0xd06); imbue appears as a quest menu item, not a fixed line |
| Gheed 147 | talk, trade, gamble |
| Kashya 150 | talk; above clvl 7 the record is rewritten to talk, **hire**; a dead merc inserts "Resurrect NAME: cost" ahead of hire |
| Warriv 155 (Act 1) | talk; after Andariel (quest 6 bit 0) a second entry "go east" (0xd36) appears (handler 0x4b5140, C->S 0x38 to level 0x28) |
| Cain 265 (camp) | talk, identify items (0xfb4, shown only if you carry something unidentified) |
Other acts, for later: Greiz 198 talk/hire; Elzix 199 talk/trade/gamble; Fara 178 talk/trade-repair; Drognan 177 talk/trade; Lysander 202 talk/trade; Atma 176 and Jerhyn 201 talk only; Meshif 210 (Act 2) talk (plus "sail east" after quest 14 bit 0), Meshif 264 (Act 3) talk/sail west; Asheara 252 talk/hire/trade; Hratli 253 talk/trade-repair; Alkor 254 talk/trade/gamble; Ormus 255 talk/trade; Tyrael (Act 4) 367 talk (+resurrect); Halbu 257 trade-repair; Jamella 405 trade/gamble; Larzuk 511 talk/trade-repair; Malah 513 talk/trade; Drehya 512 talk/trade/gamble; Nihlathak 514 talk/gamble; Qual-Kehk 515 talk.
(Talk is always first when present. The quest "kind 2" topics come from the server and show inside the **talk submenu**: header "talk", "introduction" if never heard, "gossip", quest topics, "about the merchants" (Greiz), "Horadric Cube" (Cain with a cube), cancel: `re/npc-talk.md`.)

**Positions (800x600 default; formulas)**
- Anchor (cx, cy) = the NPC's feet on screen, y - 150 (and y >= 20) (`FUN_004b1c80`).
- Header line height 0x15 (21), each entry and cancel 0xf (15), font16; width = widest line + 20; height = sum of line heights + 15.
- **x = cx - w/2, y = cy - h/3**; clamps in this order: if x + w > W - 10 then x = W - w; if y + h > H - 58 then y = H - h - 48; then x,y >= 10.
- Line i baseline = y + (heights of the lines before it) + its own height; text centred in the box. A click anywhere in a line's 15-px band selects it (the `menu` devctl verb reports "a point inside each line, header first", `docs/control_channel.md`).
- Example Charsi (3 lines + header + cancel = header 21 + 3 x 15 = 66, + 15 = h 81): first entry band starts at y + 21, its baseline at y + 21 + 15. Font metrics for exact width (`FUN_004b83a0` centring) not needed to click the middle of a line: the band is full-width.
- The hover name appears only for MonStats2 `isSel` = 1 units (camp guards and chickens are not selectable) (`re/npc-menu.md` "Hover names").
- Our own click logic should keep using live screenshots (the formulas are for validation: the anchor y-150 and h/3 rule predicts the menu box from a known feet position, D2D-U about our window scale if D2DX stretches the 800x600 frame).

---

## 5. Belt and potion rules

- **Belt layout**: 4 columns of up to N rows; a column's potion is used from the bottom (C->S 0x26 uses a **column**, not an id). When the bottom potion is used the ones above shift down (`code: rules.hpp drink_at`: box -> box - 4). Capacity per belt type = belts.txt rows via armor.txt `belt` index; the values are **not in the notes** (D2D-U). Known community: sash/light belt 2 rows (8), belt/heavy belt 3 (12), larger 4 (16): U, from memory.
- **Auto-placement on pickup** (`re/combat.md` "Picking up" step 5, `FUN_0062bad0`, `FUN_00628ba0`): a 1x1 Beltable item goes to the belt first when it is autobelt (misc.txt autobelt) **or** a column's bottom item matches (same code, or both healing hp1-5, both mana mp1-5, or both rejuvenation rvs/rvl); matching columns 0-3 take the lowest free box; otherwise (autobelt) the first free bottom box. Scrolls (isc, tsc) are excluded from the "bottom match" path (only the autobelt path moves them). Then inventory; no room: message 0x17 (cantcarry). D2D-V.
- **Cursor to belt**: C->S 0x23 {item id, belt slot} (`re/net-packets.md` ~801). Belt-to-cursor 0x24. Auto-belt message 0x63 exists for the shift-click inventory-to-belt shortcut (`re/network.md` ~104 "item to belt"); the shift-click rule itself (which column it picks) is not spelled out beyond the pickup rules above: D2D-U, probe it.
- **Drinking**: potion effect (`FUN_005be3f0`): healing amount = calc, `hpregen` << 8, class bonus (amazon, paladin, assassin x1.5; barbarian x2), a small chance to double from vitality; applied as a regen state over `len` frames that **joins** what is left (not instant). Rejuvenation is a percentage of max (`FUN_005beac0`), instant, no class bonus. Antidote/thawing: +50 poison/cold resist for 750 ticks, clearing poison / freeze. D2D-V.
- **To merc**: Shift + belt key, hpot/apot/wpot only (section 2).

---

## 6. Drops, gold, selling, repair, gamble

**Treasure and item level** (`re/drops.md`, checked 100000/100000 against game.exe's own roll code)
- **ilvl of a monster drop = the monster's level** (min 1); chests/objects use the area level (`FUN_00558200`). TC chosen by the unit type: superunique SuperUniques TC, unique TC3, champion TC2, else TC1 by difficulty; quest bosses (Andariel, Duriel, Mephisto, Diablo) use TC4 until the killer's quest flags show done/reward (flags 15, 1 or TCQuestCP).
- Expansion, past Normal, non-boss: the class is bumped by the monster level (`FUN_0055afa0`).
- NoDrop is rescaled by player count (`nd/(nd+total)`, party members count half beyond the first), picks up to 6 items per kill.
- **Gold**: a gold entry's coins = **ilvl + rand(5 x ilvl), at least 1**, x the TC's multiplier (`mul >> 8`), then **x (100 + gf)/100** where gf = killer's gold find stat 0x4f plus the owner's (`FUN_005589a0`: only for a drop that leaves the pile under the max 6). Gold in chests: "plain gold piles" `rand(5)+5` on quest chests (`re/quests-act2.md` chests).
- **Magic find** (`FUN_00558640`): stat 0x50 (killer plus owner). Unique/set/rare: chance numbers `(base - d/divisor) x 128` with d = ilvl - qlvl; with mf, odds = odds x 100/eff where eff = x < 111 ? x : (x-100) x k/(x-100+k) + 100, x = mf + 100, k = 250 / 500 / 600 (unique / set / rare). **Magic quality uses undiminished mf**. (mf < -99 skips to superior.) That is the standard diminishing-returns formula and now D2D-V against 1.14d.
- Shrines/gold find/mf on gear only count equipment in d2d (not skills), irrelevant to the original.
- **What sells**: sale price = item base cost + quality extras (magic prefix/suffix, set, unique, rare affixes each `mul x base / 1024 + add`) + half of each socketed item's cost; ethereal sells for a quarter; then x the NPC's `buy mult` / 1024 and x quest multipliers for each done quest; capped at the NPC's max buy for the difficulty; x quantity for non-stackables (`re/store.md` "d2d" prices, `FUN_0062efb0`). Live-checked on a real 1.14d host: Akara with Den done and 14 % reduced price: mp1 47, hp1 24, isc 62, tsc 78, vps 19, yps 31.
- **Selling cap**: gold added is capped so carried gold <= clvl x 10000 (`FUN_00622e70`); stash deposit up to 2,500,000.
- **Vendor stock**: store level = clvl + 5, **Normal capped at 12 / 20 / 28 / 36 / 45 by act** (record +0x22); each base item from the vendor's Min..Max columns whose level is reached; quality roll rand(100) (store level < 5: >90 low; < 10: >85 superior; else >74 superior); magic copies per MagicMin/Max if MagicLvl reached; permanent items once; quivers full. In Nightmare above clvl 25 / Hell a chance of upgraded bases. Restock is every **240,000 ms (4 minutes)** (`FUN_00537230`) but Kashya/Greiz/Larzuk lists are not restocked; d2d does not implement the restock. D2D-V for the rules, D2D-U for restock fidelity in d2d.
- **Buy** takes "sell mult" and questsellmult columns; a buy/gamble/repair takes `item_reducedprices` (stat 87, max 99 %) off: price - price x r/100, min 1.
- **Repair** (mode 3, `FUN_0062efb0`): buy base with quality extras (not ethereal /4) x (max - current)/max durability, x NPC rep mult and quest rep mults / 1024, min 1, plus charge recharge cost; free for whole, ethereal or indestructible items. Repair-all at Charsi/Fara/Hratli/Halbu/Larzuk (button frame 18, label 0x276f). Repair works on equipped + inventory (not stash).
- **Identify** (Cain, `FUN_004b2020`): free once the Cain quest is done (bit 0) or reward due (bit 1) at any act's Cain; otherwise **100 gold per item, charged all at once, and nothing is identified if the gold is short** (`re/quests-act1.md` ~1069). Pays carried then stash.
- **Gamble** (Gheed, Elzix, Alkor, Jamella, Nihlathak, Drehya): rings and amulets cost their gamble-cost column; everything else with C = max(clvl,5): price = ((max(qlvl-45,0) - qlvl/2 + C) x 250/3 + mix) x ((2C+1)/3 + 20)/15 where mix weights base/exceptional/elite costs by (clvl - qlvl) probabilities (full formula in `re/store.md` "Gambling"). Item level clvl-5..clvl+4 (`code`, the server roll untraced: D2D-U for the roll, D2D-V for price).
- **Wear** (`re/store.md` "Wear"): a hit that lands takes 1 durability by a roll under 10 (armour) or 4 (other weapons); a defender wears one armour piece chosen by weight (torso 5, head 3, arms 4, belt/feet/gloves 2).

---

## 7. Monster behaviour that matters for the combat loop

Files: `re/monster-ai.md`, `re/monsters.md`, `re/superuniques.md`, thinks in `components/rules/monsters.hpp` (code, cited as comments). Act 1-3 thinks were run natively against game.exe's: 150,000 random scenes per seed x3 all matching (`re/monster-ai.md` "Checked"). D2D-V.

**Noticing you**
- Outdoors and Act 1 outdoor presets: **no line of sight needed**; any monster whose think comes due with a player under its aidist (MonStats; 0 means 35 subtiles, about a screen) takes the player as target; players within 0x37 (55) subtiles only (`re/monster-ai.md` "The target search"). In caves/crypts/tower (Outdoors off) sight is needed until aware; one monster that sees you sets its **area's** flag so the next unaware one in the area also takes you (**no call for help beyond that**). **Being hit does not make a monster aware**: a hit monster that finds nobody wanders (5) (`re/monster-ai.md` "Alerting").
- Town rooms are excluded: monsters never target a player in a town room (levels 1, 40, 75, 103, 109) (`FUN_0061ab00`).
- Thinks come every `aidel` frames (default 15) and at once at the end of a walk/run.

**Per monster (Act 1)**
- **Fallen** (AI Fallen; leader within 15 and shamans can **rally the group into charging**, `FUN_0058f730`): if hit while out of melee it walks in; a leader within 15 rallies with taunt (aip1 %); within aip2 (10 subtiles) it walks in, else 30 % wanders, else stands 10; in melee swings aip3 % else 30 % taunts. **When a neighbour (leader) dies the pack gets scared**: they run off 12 once ("dying" branch, 1 in 20 a scream) (code comment + `re/monster-ai.md`). So "a Fallen standing still at 7 tiles" is normal, not a bug.
- **Zombie / Skeleton melee**: Zombie runs at a foe it was hit by, or aip1 % one within aip2; otherwise wanders 3 (except in the Burial Grounds, where it always runs). Skeleton walks in aip0 %, in melee aip2 % attacks. No ranged threat.
- **Quill Rat / Spike fiend line** (AI QuillRat): melee A1 in melee, **spikes (A2) when hit**, wanders when far (aip0), else aip1 % spikes after a back-off. Hell rows carry thorns 15-40 (MonProp rows 605-607: "thorns" is melee damage back to the striker, `re/monsters.md` MonProp) so ranged attackers or Fire Bolt are safer in Hell.
- **Blood Raven** (AI 59, `FUN_005e6320`): stands if you are > 45 away; heads home if she strays 50+; at > 20 she closes to about half distance; every think adds 3 to a counter and out of melee she raises a **zombie (Nest) 5-19 tiles from the target** while fewer than 8 + 2 x difficulty are up (so up to 8 / 10 / 12) with probability counter %; else past 5 tiles: 5 % repositions, else if not hit 80 % attacks (Quick Strike (difficulty+4) x 10 % else bow), else circles 4; within 5: 30 % backs off 12, else melee. Created with bloodraven unique mods and bound to quest 2; in Hell her MonProp adds a 60 % knockback (`re/superuniques.md`, `re/monsters.md`). It is a **ranged caster-kiter**; do not melee into her zombie pile.
- **Dark Stalker**: not named in the notes (probably a different row name; U). Search the notes by MonStats row if needed.
- Doors: most Act 1 walkers open doors (Fallen, shamans, skeletons, corrupt rogues, Andariel) (`re/monster-ai.md` "Doors").
- Level for drops/XP in Nightmare/Hell in LoD: the area's level, unless noRatio or boss (`re/monsters.md` "Stat init"); HP = MonLvl row x MonStats min..max % (1.14d percentages), plus /players.
- Resist values by difficulty: **not tabulated in the notes** (they read them from MonStats at run time); players' penalty is DifficultyLevels ResistPenalty clamped -100..75 and applies to the merc too (`re/merc-ai.md`, `re/combat.md` ~236). No new data for our `docs/data` tables.
- Poison and fire states: poison state 2 and burn 0x73 hold a negative `hpregen` stat; a weaker new application is ignored, an equal or stronger one replaces (`re/combat.md` "Regeneration"). Healing potions are the same stat in a different state, so poison and potion add per tick.
- NPC heal (Akara etc.): life/mana/stamina to max and curses/poison/freeze off, chill stays (`re/combat.md` "Cures"). Antidote and thawing add +50 resist for 750 ticks.

---

## 8. Memory structures and addresses vs `docs/d2-1.14d-memory.md`

The notes are mostly about game.exe **functions** and their data (decompiled); very few memory addresses of use to an external reader, but several of our open questions are answered. All D2D-V unless marked (they are used by d2d's own test tools against a live 1.14d: `apps/whisper/whisper.py`, `tools/emu/*.py`).

Confirmed (our RVA + 0x400000 = their VA):
| Ours | VA | Their evidence |
|---|---|---|
| PlayerUnit `0x3A6A70` | **0x7a6a70** | `apps/whisper/whisper.py:124` reads it live; `whisper.cpp:45` "FUN_00463dd0 returns DAT_007a6a70, the client's player unit" |
| ExpCharFlag `0x3A04F4` | **0x7a04f4** | `re/quests-act4.md` "The ending": the client's expansion flag, set from S->C 0x01 byte +6 |
| GameInfo `0x3A0438` | **0x7a0438** | `re/tcpip-menu.md` ~124: expansion character = bit 0x20 at +0x1ef of that struct |
| AlwaysRun `0x3A0660` | **0x7a0660** | `re/char-panel.md` ~240: the run toggle |
| bWeapSwitch `0x3BCC4C` | **0x7bcc4c** | `re/weapon-switch.md`, `re/net-packets.md` 0x97 toggles it (the inventory's weapon set) |
| MapId `0x3A0638` (semantics R) | **0x7a0638** | `re/automap.md` ~179, 202: dword at that address = **the map seed** (game +0x7c from the load-act packet); the next dword (+4) is the **object seed**. Useful for map-seed cross-checks |
| GetDifficulty fn RVA `0x4DCD0` | VA 0x44dcd0 | `re/quests.md`: `FUN_0044dcd0() == 2` = Hell (0 N, 1 NM, 2 H) |

New answers to our open questions:
- **Difficulty backing variable (our Q6)**: `tools/emu/skill_tips.py:42` sets `0x7a060c` as "FUN_0044dcd0's difficulty" -> **VA 0x7a060c, RVA 0x3A060C**, a dword 0..2. D2D-U-ish: found through an emulator harness stubbing the variable, not a live read; one 4-byte live read on the worker host confirms it. Related: FUN_0044dcc0 is the expansion-game test.
- **Client quest state** (our questmem reads the server player): the client keeps its own copy: the 96 bytes from S->C 0x28 type 6 are held at `DAT_007c0d43` (a pointer stored **unaligned** at VA 0x7c0d43; the whisper tool reassembles it from the dwords at 0x7c0d40/0x7c0d44, then record +0 points to the 96 bytes = 48 u16 words), and the **game's** quest words (the "done in this game" bits 13/15) at `DAT_007c0d47` (`apps/whisper/whisper.py quests()`, `re/net-packets.md` 0x28/0x29). D2D-U for stability (the pointer may be set per game).
- **Server structs** (the single process holds client + server): confirmed offsets: unit +0x14 data, +0x18 act, +0x1c act ptr, +0x20 unit seed (low, 666), +0x2c path, +0xc4 flags (bit 0x800000 = removed from room), +0xd0 list id, +0xe4 hash-chain next (server hash at `game + DAT_006e10e0[type] + (id & 0x7f) x 4`, 128 buckets), +0xe8 next in room; room->first unit +0x74; PlayerData +0x10..+0x18 normal/nm/hell quest-flag pointers, +0x1c waypoint pointers, +0x60 NPC-intro bitfields, +0x9c client record (`re/quests-act5.md`, `re/quests-act1.md`, `re/act1-end.md`); player path +0x1c Room1, Room1 +0x10 Room2, Room2 +0x58 Level, Level +0x1b4 act, act +0x458 map seed (`apps/whisper/whisper.py town()`); path +0x3c type, +0x58 target unit, +0x10/+0x14/+0x18 = SP1/SP2/SP3 (dest, dest when pathed, computed-to), +0x24/+0x28 point index/count (`re/monster-ai.md` "A move"). These agree with `docs/d2-1.14d-memory.md` sections 2.2 and 2.3 (our Path 0x10/0x12 target and 0x3C type match).
- **Life/mana/stamina are in 256ths** (stats 6/7 stored as value << 8) (`re/monsters.md` "Stat init"). Consistent with our `>> 8`. Gold/XP/stash gold stats 14/15/13 are plain integers (death code adds/subtracts them directly; gold caps `clvl x 10000`): our open Q4 "is gold also x256" -> **no**, D2D-U (inferred from the arithmetic, not a read).
- Monster data: +0x28 AI control (+0x2c owner id), +0x38/+0x34 skill-set target kind/value, +0x50 area; monster stat 0xac = alignment (0 evil, 1 neutral, 2 good); stat 0x4f gold find, 0x50 magic find, 175 goldlost, 87 reduced prices, 73/75 durability.
- Unit modes: player mode 0 and 0x11 are dead, 0x11 corpse; monster 0 and 0xc dead (`re/monster-ai.md` target search), matching our mode tables.
- Not helpful: no Room/CollMap offsets beyond the above; no UI address list (their UI is a reimplementation; the few UI VAs: gold box mode `0x7a279c`, store-open variable `0x7bcbf0` in 1..9, hire list at `0x7c0c85`, resurrect cost `0x7c0dd0`, quest events table `0x7c0ea4`: all D2D-V from the decompile but unvalidated by us).
- Contradictions found: none with our structure tables. Only open: our "stat getter shifts all stats" (we guessed) vs theirs: only life/mana/stamina are fixed-point.

---

## 9. The control channel and driving d2d instead of Game.exe

`docs/control_channel.md` (D2D-V: it is the docs for code they run in tests: `tests/smoke_d2d.py`, `tests/quests_d2d.py`).

**Transport**: AF_UNIX stream socket path given by `--devctl <path>` (or env `D2D_DEVCTL`); line-based LF text; one client at a time (a second gets `err busy`); reply payload lines then `ok`, or `err <msg>`; 64 KiB line cap. `--headless` = SDL dummy driver + software renderer (no window). POSIX only (Windows stub). **Local development only: no authentication**; any local process can drive input and dump state, `save` writes the character file, `screenshot <path>` writes any path.

**Read verbs**: `ping`, `info` (w, h, frame), `state` (one line: screen, difficulty, hardcore, camera, walking, gold, stats, life/mana/stamina cur/max, level, exp, mode, missiles, pets, left/right skill, skill points, waypoint, items count, held item, unidentified count, `merc=x,y:code`, menu lines, automap, speech, store vendor), `items` (each item `[code loc slot q panel size at]` plus tooltip lines), `monsters` (per monster: id, cell, screen x/y, hp/max, mode, boss kind, mods, level, name, unit id, anim row), `npcs` (name, screen x/y, has menu, index, cell), `ground` (code, label, screen x/y, unit id), `menu` (NPC menu lines with a click point each), `debug players/roster/level/blocked/grid/collision/automap/warps/portals/objects/presets/critters/tiles/laid/light/walls/net`, `debug attack` (full combat stat line), `debug corpses` (level, x, y, items per corpse), `debug passives`, `debug quest <q> show` (16 flag bits, quest states).
**Action verbs**: `click x y` / `rclick` (real SDL events, applied next frame), `key <name>`, `text`, `move x y` (hover), `wheel`, virtual gamepad `pad ...`, `save`, `quit`, and **`cmd <what>`** straight to the World: `move x y`, `skill <id> x y [unit] [left] [item]`, `interact <npc index>`, `pickup <unit id>`, `resurrect`, `stat`, `skillpt`, `select <skill> <left>`, **`belt <slot> [merc]`** (merc: shift held, merc drinks).
**Debug/cheat verbs**: `debug stat/statpts/skillpts/equip/wear/unid/clearinv`, `debug die [id]` (player dies where standing: death penalty then corpse), `debug quest <q>` (mark done and run the join/chain), `debug kill [n] [spare <id>]` (kills as real kills: experience, drops, quest deaths), `debug difficulty <d>`, `debug warp x y`, `debug goto <level>`, `debug enter <i>`, `debug itemat <id>`, `debug skill/points/charges/release/boss/operate/day/rain/alt`.
**Not exposed**: no stash/store/trade/hire verbs besides what `click` and `cmd interact` reach (the menu's lines come from `menu`), no unit table with merc state beyond `state merc=`, no reading of the corpse's contents beyond `debug corpses` count, Act 5 content is only partly checked (uber levels).

**What it would take for our Go bot to drive d2d**
- Our layering (`PLAN.md`, `internal/mem` Source, `internal/input` Controller + the d2input in-prefix helper) assumes game.exe's address space and Wine. d2d is a native process with its own C++ memory: our typed readers (`internal/d2`, `internal/d2map`) cannot read it. The seam has to be **above** `internal/mem`: either a `d2` snapshot/state provider fed by `state`, `monsters`, `npcs`, `ground`, `items`, `menu`, `debug level/grid/warps/portals` (our `internal/d2/snapshot.go` already models a recorded game state offline), or a fake `mem.Source` that synthesises game.exe-shaped bytes from those replies (heavier; D2D-U).
- Controller side: `cmd move/skill/interact/pickup/belt` replace iso-transform clicks and hover checks (world coordinates and unit ids go straight in, which is easier than our screen mapping), `click`/`rclick` with `menu` points replace NPC-menu clicking. Latency: line protocol per command, applied at the next tick (25 Hz server tick).
- Out-of-game flow (character creation/selection, difficulty box) is the d2d front end; `state` reports the screen. Our oog/launch package would need a d2d variant.
- Work estimate: a new backend package (`internal/d2d`: socket client, parser for the eight reply formats, mapping to our types) plus switch points in `cmd/d2bot`; the combat/town/nav logic mostly consumes interfaces and could run unchanged. Risky bits: collision data (`debug grid n` gives `.`/`#` rows per subtile, `debug blocked x y`), unit ids vs our stable ids, shop and stash UIs by pixel.
- **Value**: a fast, deterministic, headless rules testbed (`debug die/corpses/quest/kill/difficulty/warp`), seeds (`-seed`, `-act`), no Wine, no Battle.net. A soak or quest-chain regression could run in seconds. But it tests our bot against **d2d's** idea of 1.14d, not the game's.
- **Risks, honestly**: (1) unfinished/fidelity: see section 0, `re/todo.md`, `re/act1-gaps.md` (~365 ponytail notes triaged), 494 ponytail notes by 2026-10-29; merc auras not built, vendor restock not built, corpse x/y not read, one-player quest lists; Act 5 uber levels incomplete; PLAN calls out "play bot not yet written: quest tests use debug verbs" (`docs/PLAN.md` ~1098). (2) The channel is unauthenticated and can write files: keep it on a private socket path under our control, never expose. (3) Need to **build** d2d (C++26, SDL3, OpenAL, StormLib, CMake) on the worker host and supply our own MPQs (the launcher reads ISOs/GOG data; MPQs come from our install; their `bin/` dev-only PEs are not needed by us). Not done here by instruction. (4) **GPL-3.0**: it is a separate program; talking to it over a socket from our stdlib-only Go does not require our code to be GPL, but we must not paste its code, tables or doc passages into this repo (this file paraphrases and cites). `LEGAL.md` argues EU-law interoperability and admits "scope is debated"; it is their position, not advice for us. They state the repo contains no Blizzard files (we must also keep our MPQ-derived data out of git as today). Some assets (icon, wordmark) have other licences. (5) It is a hobby project by a single author's account of the notes; releases/versions may break the verb set (`docs/control_channel.md` is the contract; "Future verbs").

---

## 10. Things we can stop probing live vs what still needs the real game

**Stop probing live (accept the notes, spot check later)** — D2D-V, mostly emulator-diffed or live-checked wire:
1. XP penalty: 0 / 5 / 10 % of the level span, never below level start + 1; 75 % of it on the corpse.
2. Death gold formula and that the remaining purse drops as a ground pile; stash safe in SP.
3. Corpse contents = worn items 1..12 + cursor item; XP return on pickup; pickup by interact click; items re-seat into free slots else inventory. (Keep one probe for walk-over.)
4. Resurrect cost formula and the five NPCs (Tyrael included); pay carried then stash.
5. Hire-offer level rule (clvl-5+rand(5), min 2), cost = Gold x (15d+100)/100, ten offers, eight visible rows, list regenerated; Kashya hire needs clvl >= 8.
6. Free Blood Raven merc: first unhired valid offer, no gold, forfeited if you own any merc (even dead).
7. Merc gear rules (identified, unbroken, slot/type per merc row, merc's own str/dex/level), merc follow/teleport thresholds (24 run, 100 teleport), merc never attacks in town.
8. Sell/buy/repair/identify/gamble price formulas; Cain's identify fee; vendor stock formula (store level = clvl+5, act caps).
9. Gold drop formula, item level = monster level, magic find diminishing formula, GF applied to gold.
10. NPC menu line order and geometry formulas (use for validation of our click targets).
11. Monster noticing rules (35-subtile aidist outdoors, no LOS outdoors, area alert, being hit does not alert) and the Act 1 thinks (Fallen rally/scare, Blood Raven zombie raising and kiting).
12. Quest bit meanings, that quest flags persist in the d2s (0x159 layout) and records are rebuilt from bits and pack items at join; the Act 1/2/4/5 rows marked CONFIRMS in section 3.2.
13. Difficulty variable (one 4-byte read) and the unaligned client quest pointer (one live check each, cheap).

**Still needs a live probe on the real game (notes silent, approximate, or in conflict)**
1. **Where the corpse appears after quit + new game** (saved x/y vs town start vs body near the death area) and **how many corpses survive / which one is oldest** (d2d writes only the latest, game.exe untraced).
2. **Mephisto**: does the hellgate return after Save and Exit? (notes say yes via the join rule; user report says no): decisive, single probe.
3. **Cain stones portal after reload** (notes: yes, returns at Stony Field): confirm before relying on A.
4. **Khalim's Will orb stays broken after reload**; **Staff placed -> stairs open after reload**; **Summoner/journal -> Canyon portal after reload**; **Diablo -> Tyrael portal after reload**.
5. **Rejuvenation to the merc** (notes: only hpot/apot/wpot): conflict with our mercenary doc.
6. Walk-over vs click for corpse pickup, interaction range, and pickup when the slot is already occupied.
7. Merc following through town portals, waypoints and acts (not in the notes); merc XP/level behaviour after resurrect.
8. Belt capacity per belt type (belts.txt rows) and the inventory-to-belt shift-click column choice.
9. Monster resists by difficulty and the exact Act 1 monster table (Dark Stalker); Hell Blood Raven knockback.
10. Autosave points in single player other than Baal (and whether killing the game right after a quest event keeps it).
11. Gold arithmetic on a real death (purse, stash, level cap, ground pile) and the real restock timing of vendors (4 minutes in code; d2d does not implement it).
12. NPC menu click bands at our actual window scale (D2DX/Wine), since formulas assume 800x600 game pixels.

---

## 11. Open questions for the main agent
- Do we want a throwaway-only probe batch (cheat ledger OK) for the "still needs live" list items 1-5? They flip several plan-v2 10.5 rows and our corpse policy.
- Decide whether to spend effort on a d2d backend (section 9). Cheapest first step: build d2d on the worker host with our MPQs, run `--headless --devctl`, and use `debug die` / `debug corpses` / `debug quest` as a rules oracle for the probes above, before any bot port.

Commit trail: see `git log` on branch `agent/d2dharvest`.
