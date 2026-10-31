# Quest completion detection per act and difficulty

Companion to `docs/goals/execution.md` section 3 (mechanics) and `docs/goals/tracking.md` (state inventory). This file answers one question for each quest:
**how does the bot know it is done, started, or failed, and how sure is it?** Tags V/S/U as in the README.

## 1. Evidence tiers

| tier | source | latency | exactness | status |
|---|---|---|---|---|
| T1 save | `.d2s` quest block, offset 0x14F, 3 difficulties x 96 bytes (48 little-endian WORDs each) | only after the game wrote the file (Save and Exit, or an autosave moment we have not characterised) | exact for the moment of writing | layout V (cheats.md: decoded and round-tripped on real saves); **bit meanings U** |
| T2 memory | `PlayerData` quest pointers after the 16-byte name (D2BS: normal, nightmare, hell, then waypoint pointers) | immediate | exact if the offsets hold | U (docs/goals/tracking.md items 1-4; `d2-1.14d-memory.md` has no quest RVA) |
| T3 proxy | reward item in pack (V reading of inventory), quest item in pack, NPC dialog change, kill event (V events), UI flag 0x11 edge, UI flag 0x0F quest panel | immediate | may be wrong (a dropped item, a missed kill) | inventory and flags V; dialog change U |

Rules for the step layer (tasklayer.md):
- A handler step may *skip* work only on T1/T2 evidence ("already done"). T3 alone never skips.
- A handler's last step records `Evidence` T1/T2/T3 with the completion. T3-only completions are re-checked at the next Save and Exit (T1) and corrected.
- Never persist handler progress ("I was at step 4"). After any restart, re-derive from T1/T2 plus the inventory. (execution.md section 5 says the same.)
- **UI flag 0x11 (V town.md):** the red quest-log button flag stays set after a quest update and Esc does not clear it. Use only as a rising edge: 0 -> 1 right after a talk or kill means "the game recorded a quest change". It cannot say which quest. Whether opening the quest log (`q`, flag 0x0F V) clears it: U (V-04 step 6).

## 2. The quest block (save), candidate decoding (indexes mostly V live 2026-10-28, see docs/saves.md; bits partly V)

96 bytes per difficulty = 48 WORDs. D2BS/Kolbot index the WORDs as follows (recollection of `sdk.quest.id`; tracking.md item 3 lists only the first 8).
Indices 0-7 are Act 1 (0 = spoke to Warriv, 1 Den of Evil, 2 Sisters' Burial Grounds, 3 Tools of the Trade, 4 The Search for Cain, 5 The Forgotten Tower,
6 Sisters to the Slaughter, 7 = "able to go to Act 2"), 8-15 Act 2 (8 = spoke to Jerhyn, 9 Radament, 10 Horadric Staff, 11 Tainted Sun, 12 Arcane Sanctuary,
13 Summoner, 14 Seven Tombs, 15 = able to go to Act 3), 16-23 Act 3 (16 = spoke to Hratli, 17 Lam Esen's Tome, 18 Khalim's Will, 19 Blade of the Old Religion,
20 Golden Bird (VERIFIED LIVE 2026-10-28: word 20 lit The Golden Bird; this list had the order wrong, internal/saves was right), 21 Blackened Temple, 22 The Guardian, 23 = able to go to Act 4), 24-28 Act 4 (24 = spoke to Tyrael, 25 Fallen Angel, 26 Hell's Forge,
27 Terror's End, 28 = able to go to Act 5), and Act 5 at 35-40 (VERIFIED LIVE 2026-10-28: words 37 and 40 lit Prison of Ice and Eve of Destruction in the log; internal/saves had 33-38 and is fixed) (35 Siege on Harrogath, 36 Rescue on Mount Arreat, 37 Prison of Ice, 38 Betrayal of Harrogath,
39 Rite of Passage, 40 Eve of Destruction); 29-34 unused padding. The Act 1 intro WORD, the Cow level WORD and the "went to the Worldstone" markers
are not located (U).

LIVE 2026-10-28 (docs/saves.md): 0x0001 and 0x8001 both read "completed in a previous game", 0x000c = started, 0x0010 alone = active, Akara's Den dialog writes 0x0004, leaving town adds 0x0008, the game ORs 0x1000 into completed words after the log is viewed. The rest below stays candidate.

Bit meanings inside a WORD, candidate (from the Phrozen Keep file-format page as remembered, U): bit 0 reward granted, 1 reward pending, 2 started, 3 leave
town, 4 enter area, 5-11 quest-specific (custom), 12 quest log update (the red button), 13 "closed" (quest complete and acknowledged), 14-15 per quest.
Kolbot's `me.getQuest(id, 0)` is "completed" and `getQuest(id, 1)` "requirements met / reward pending" (U). The decode we will code:

```
type QuestBits uint16
func (q QuestBits) Done() bool     // candidate: bit 0 (reward granted) OR bit 13 (closed); resolved by V-01
func (q QuestBits) Started() bool  // candidate: bit 2
func (q QuestBits) Pending() bool  // candidate: bit 1 (reward waiting at an NPC)
```
Do not code the meaning before V-01. Code the *index table* and the raw WORD in `saves.Progress` (the WP-SAVES API) and expose both; the interpretation lives in
one function so V-01 changes one line.

`saves.Progress(d2s)` contract assumed from the WP-SAVES package (parallel-work.md): per difficulty, quests completed (named from a table), waypoints active
(named by level id), difficulty unlocked, map id. This design needs two additions, requests for that package (R2 in packages.md): the raw WORD per index (not
just a bool), and the act of the active difficulty byte (0xA8..0xAA, V layout, "bit 7 active, low bits = act").

## 3. Detection matrix per quest

"Done signal" lists the strongest evidence first. "Loss" lists what makes the proxy lie. Difficulty column: **same** means the quest exists in all three
difficulties with its own bits (V game rule: quest arrays are per difficulty), **N-only** means the repo data says the reward is Normal only.

### Act 1

| quest | index | done signal (T1/T2 bit; then proxy) | started signal | per-difficulty note | loss |
|---|---|---|---|---|---|
| Spoke to Warriv | 0 | bit; NPC Akara reachable | n/a | same | none |
| Den of Evil | 1 | bit; proxy: skill point appeared (stat 5 +1) and Akara re-talk | UI 0x11 after Akara talk | skill point N-only in repo data (quest-points.json); respec flagged U; NM/H: skip (quest-points note) | point spent already; killed-all unknowable from kills |
| Sisters' Burial Grounds | 2 | bit; proxy: Blood Raven kill event (boss flag) + Kashya talk | kill event | same; hire rogue is a separate action | kill by merc still emits kill (V events from unit death) |
| Tools of the Trade | 3 | bit; proxy: Horadric Malus gone from pack + imbued item exists | Malus in pack | same | item sold/dropped |
| Search for Cain | 4 | bit (set when Cain freed and talked); proxy: Cain NPC in town | Scroll of Inifuss in pack | same | Cain also appears in the cage; town presence needs a unit scan |
| Forgotten Tower | 5 | bit; proxy: Countess kill event | n/a | same | none |
| Sisters to the Slaughter | 6 | bit; proxy: Andariel kill event + Warriv "Go East" row visible (U) | n/a | same | none |
| Able to go to Act 2 | 7 | bit; proxy: Warriv hop works | set on Andariel kill | same | none |

### Act 2

| quest | index | done signal | started | note | loss |
|---|---|---|---|---|---|
| Spoke to Jerhyn | 8 | bit | talk | same | none |
| Radament's Lair | 9 | bit; proxy: Book of Skill used (stat 5 +1) | Atma talk | skill point all difficulties (quest-points.json V Maxroll) | book not read (unused book in pack = not done) |
| The Horadric Staff | 10 | bit; proxy: Horadric Staff in pack (or in the orifice: "staff placed") | Cain talk / cube found | same | staff placed -> item gone from pack, bit is the only proof |
| The Tainted Sun | 11 | bit; proxy: Amulet of the Viper in pack | enter Lost City / Claw Viper | same | same as staff |
| Arcane Sanctuary | 12 | bit; proxy: portal to Arcane in Palace Cellar | Jerhyn/Drognan talk | same, optional | none |
| The Summoner | 13 | bit; proxy: Summoner kill event | enter Arcane | same, optional (U) | none |
| The Seven Tombs | 14 | bit; proxy: Duriel kill event + Tyrael talk | Tyrael talk | same | none |
| Able to go to Act 3 | 15 | bit; proxy: Meshif sail option (U) | after Duriel + talks | same | none |

### Act 3

| quest | index | done signal | started | note | loss |
|---|---|---|---|---|---|
| Spoke to Hratli | 16 | bit | talk | same | none |
| The Golden Bird | 20 (V live) | bit; proxy: +20 life potion (not detectable) | Jade Figurine pickup | same, optional; chain is U (execution.md) | none |
| Blade of the Old Religion | 19 | bit | Hratli talk | same, optional | item given |
| Khalim's Will | 18 | bit; proxy: Compelling Orb smashed (object mode change) | Flail pickup | **Req** to reach Mephisto | relics are consumed in the cube |
| Lam Esen's Tome | 17 | bit; proxy: stat 4 +5 | Alkor talk | +5 stat points N-only per repo data, goals open question | tome used |
| Blackened Temple | 21 | bit; proxy: all three Council kills (3 kill events with boss flag) | Ormus talk | same | merc kill counts |
| The Guardian | 22 | bit; proxy: Mephisto kill event + red portal object | n/a | same | none |
| Able to go to Act 4 | 23 | bit | portal | same | none |

### Act 4

| quest | index | done signal | started | note | loss |
|---|---|---|---|---|---|
| Spoke to Tyrael | 24 | bit | talk | same | none |
| The Fallen Angel | 25 | bit; proxy: Izual kill event + stat 5 +2 | Tyrael talk | 2 skill points all difficulties (V quest-points.json) | points spent |
| Hell's Forge | 26 | bit; proxy: Soulstone gone from pack, forge object used | Mephisto drop picked up | optional; rewards a rune (S) | soulstone is a ground quest item: pickit must take it (V loot has a `quest` field) |
| Terror's End | 27 | bit; proxy: Diablo kill event + Tyrael portal object | n/a | **Req** | none |
| Able to go to Act 5 | 28 | bit | Tyrael talk | needs expansion | none |

### Act 5

| quest | index | done signal | started | note | loss |
|---|---|---|---|---|---|
| Siege on Harrogath | 35 | bit; proxy: Shenk kill event + Larzuk socket dialog | Larzuk talk | socket reward all difficulties (S) | none |
| Rescue on Mount Arreat | 36 | bit (can be *failed*: record separately); proxy: reward runes in pack | barbarian doors broken | quest can fail (V "create a new game" in battle.net text, execution.md) | none |
| Prison of Ice | 37 | bit; proxy: Scroll of Resistance in pack then used | Anya in Frozen River | **Req** to open the Nihlathak portal (U) | scroll used |
| Betrayal of Harrogath | 38 | bit; proxy: Nihlathak kill event | Anya portal | **Req** | revived Tomb Vipers abort the run (prior-art 02) |
| Rite of Passage | 39 | bit; proxy: three Ancients kill events in sequence | Qual-Kehk talk, enter Arreat Summit | **Req**; level gate 20/40/60 by difficulty (V battle.net) | leaving the area resets the fight |
| Eve of Destruction | 40 | bit; proxy: Baal kill event in the Worldstone Chamber | enter Throne | difficulty complete | none |

Special state (no quest index, U): Cow level (Wirt's Leg + kill the King: "killed in this difficulty" flag, location unknown; fallback: remember in
our own per-character store, accept being wrong after a restore), Tristram open (set by Search for Cain).

## 4. Per-difficulty differences the planner needs (tag in brackets)

- Each difficulty has its own arrays; a Normal bit says nothing about Nightmare (V rule). The Act 2/3/4/5 "able to go" bits are *per difficulty*: a new
  Nightmare character starts in Act 1 of Nightmare with Act 2 locked until Andariel dies there. [S: standard game rule; V-03 checks the save bytes]
- Normal-only rewards in repo data: Den skill point, Lam Esen +5 stats. [R in quest-points.json; contradicted by general knowledge that rewards repeat; resolve in V-01 by comparing Nightmare saves.]
- Quest rewards that repeat: Radament point, Izual 2 points, Anya scroll, Larzuk sockets, Malah scroll. [V Maxroll for the points, S for the rest]
- Skippable in NM/Hell for time: Den, Search for Cain/Tristram, Golden Bird, Blade, Tools of the Trade (unless the imbue is wanted), Lam Esen. [policy; planner.md]
- Hell-only dangers on required quests: Council (Mana Burn, Lightning), Mephisto minions, Diablo seals, Nihlathak, Ancients' gate by clvl. [S planner.md]

## 5. What `Observation.Quests` carries

Extends tracking.md's `QuestState{Done, Started, RewardTaken, Failed}`:

```go
type QuestState struct {
    Index    int
    Raw      uint16       // WORD from T1/T2; 0 if only proxies exist
    Done     bool
    Started  bool
    Pending  bool         // reward waiting
    Failed   bool         // Rescue on Mount Arreat
    Evidence Evidence     // EvSave | EvMemory | EvProxy | EvNone
    At       time.Time    // when the evidence was read (T1 is stale mid-game)
}
```
Staleness rule: T1 older than the last quest-changing event (UI 0x11 edge, kill with boss flag, area change into a quest area) is marked `Stale`; the step
layer then trusts T2 or the proxy. When T2 exists T1 is used only as a cross-check and a canary: a mismatch after a Save and Exit means the T2 offset is wrong
(feeds WP-REPLAY's canary list).
