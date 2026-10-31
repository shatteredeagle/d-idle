# Mercenary: hire, revive, equip

Policy data: `docs/plan/data/mercs.json` (R). Rule sources: Kolbot SoloPlay and Koolo in `docs/archive/prior-art/01-leveling.md` section 7 and `05-town.md` section 6.
Today there is no merc code anywhere under `internal/` (V, findings F2); the plan relies on an Act 2 Might merc against cold immunes (V leveling.md).
Class: Act 1 Rogue (Kashya), Act 2 Desert Mercenary (Greiz), Act 3 Iron Wolf (Asheara), Act 5 Barbarian (Qual-Kehk); Act 4 has no hire NPC but Tyrael can revive (S Hive Workshop/Maxroll, class 367 U, dialog line U). Mercenaries need the
expansion flag (status byte bit 0x20, V layout); a classic character has none (U for our test chars: V-02 reads the flag).

## 1. State the bot needs (Observation.Merc)

```go
type Merc struct {
    Present bool        // a friendly monster unit owned by the player exists
    Alive   bool
    Class   int         // monstats class id (Rogue 271, Desert Merc 338, Iron Wolf 359, Barb 561: Kolbot sdk values, U)
    Level   int         // stat 12 on the unit (U)
    Life    int; MaxLife int
    Aura    string      // from the hire dialog text or the skill stat list (U)
    Gear    [6]ItemRef  // helm, armor, weapon, shield/off-hand(by class), gloves/boots do not exist for mercs: helm, armor, weapon only (S)
    Evidence Evidence
}
```
How to read it:
- **Unit** (implemented: `d2.FindMerc`, internal/d2/merc.go): found by CLASS in {271, 338, 359, 561}, not by owner. The owner words read 0/0 for the hired Rogue (V docs/friendly.md), which refutes the D2BS `getMercenary` owner rule. Old text: the merc is a monster unit whose owner is the player (D2BS, refuted). We have unit tables of
  monsters (V). The owner field offset in the monster struct is U (mem.md does not list it). Fallback: a friendly monster of a merc class id within ~12 subtiles
  that follows the player (V nav.md: "mercenaries are friendly monsters" are penalised cells; so the tables see them). V-07 step 3.
- **Dead/alive**: dead merc stays as a corpse/absent; Kolbot treats "no unit for 3 reads 100 ms apart" as dead (S prior-art 05). The PlayerData has a
  merc-dead flag in D2 (`pMercDead`?, U): search for it in V-07.
- **Aura/level**: the hire list shows aura and level per candidate (text in the NPC window); not readable from memory (U). Read from skills on the unit's
  skill list if the structure matches the player's (V Player skill list at `Info` +4/+8/+0xC; merc: U).

## 2. UI flow (all U unless noted)

The dialog layer (acts.md section 3) gives `Open/Pick/Close`; the merc screens add:

| screen | how it opens | evidence | geometry |
|---|---|---|---|
| NPC dialog | `Interact` on Kashya/Greiz/Qual-Kehk | UI 0x08 (V for A1 NPCs) | list above the NPC; rows "Hire", "Talk", "Cancel" (U: also "Resurrect"/"Heal" rows when dead, per Kolbot dialog lines containing ":" are the entries, prior-art 05) |
| Hire list | pick the hire row | UI flag unknown (new?) (U) | up to ~6 candidates per page with name, level, aura, cost; "accept"/"cancel" buttons (U) |
| Merc equipment | in 1.14d: clicking the merc portrait/inventory tab on the character screen (U; V-07 step 6 finds the hotkey or panel flag) | a UI flag distinct from 0x01/0x02 (U) | slots: head, torso, weapon (and shield for Iron Wolf? no: helm, armor, weapon, shield for some, S) |

### Hire procedure (`Step: HireMerc`)
1. Preconditions: no living merc (or policy says replace); gold >= quoted cost (cost visible on the list; formula U, repo heuristic `((clvl-1)^2/2)*7.5`
   is Kolbot's affordability guess); correct act town; Normal/NM/Hell expectations from mercs.json.
2. `Talk(NPC)`, pick "Hire". Read candidates: screenshot-free approach is memory-only, which we cannot do for text; so the first version hires **the first
   candidate whose cost is within budget** and verifies the result by reading the merc unit class id (Act 2 class => right NPC; aura verified later from
   the skills on the unit). Aura selection by text/OCR is out of scope until a font anchor exists; mercs.json lists the auras we *want*, used only as an
   advisory check after the fact ("got Defiance, wanted Might: rehire next difficulty").
3. Confirm: click the accept button; verify by merc unit appearing within ~3 s and gold dropping by the cost.
4. Strip: if replacing, take the old merc's gear first (merc equipment panel) or it is lost with the dismissal.

### Revive procedure (`Step: ReviveMerc`, a standard chore in `town.Chores`, order: after repair, before stash, prior-art 05)
1. Need: merc existed (T1/T2: merc-dead flag or last-known present) and is not currently alive; gold >= revive cost; the act must have a reviver: ANY act's hireling NPC revives any merc type, Act 4 via Tyrael (S Hive Workshop/Maxroll; corrects the old "Act 4 has none"; U not live-checked). Revive price `min(int(lvl^2/2)*15, 50000)` (S, D2R page; `resource.MercReviveCost`). Pre-flight: `town.ReviveReady`; flow: `Town.Revive` (gated behind `SetReviveUnverified` until a live run sees the dialog).
2. `Talk(NPC)`, click the "Resurrect" row (a dialog line containing ":" in Kolbot terms; up to 3 attempts, wait max(750 ms, 2 x ping) after each click); abort
   on the "not enough gold" speech (U: how detected, probably the unit does not appear + gold unchanged).
3. Verify: merc unit exists within 2 s of the last click; gold dropped.
4. Cap: revival cost never exceeds 50,000 gold (V Arreat Summit hirelings page, classes/ranking.md). If gold is short, the planner inserts a gold farm.
Revive trigger in-fight: the Fighter's Safety already has TownPortal; add a `MercDead` advisory cause that the Runner turns into a town trip only between
fights (Kolbot `MercWatch`), never mid-pack (bot.md guard rules: drinks are left to the Fighter while Clearing).

### Equip procedure (`Step: EquipMerc`)
- Source: `loot` verdict with a `Merc` tier list (separate from the player's, SP `Merctier`, prior-art 01 section 6): helm/armor/weapon by sockets and
  runeword (Insight polearm for the Act 2 merc, mercs.json), only when requirements (str/dex/level) are met by the merc (merc level from the unit, U).
- Mechanics: with the merc equipment panel open, `hand.Transfer` from pack to slot (the verified cursor primitives in `internal/town/hand`: Take/Put/Transfer, V for the
  player's own slots; merc slots are U, the slot rectangles must be calibrated per screen like the player's, V town.md "equipment slot rects match screenshots").
- Verification: read back the merc's items by container owner, as vendor stock is read (V: ItemData pOwnerInventory -> Inventory pOwner == NPC unit). For the merc,
  owner is the merc unit address (U: same mechanism).
- Guard rails: never equip the merc with an item the player needs (compare tier lists), never unequip a runeword the merc already wears without runes in hand.

### Potions for the merc
Shift+click of a belt potion or shift+hotkey (Koolo) heals the merc (S prior-art 01 section 7: thresholds heal 75-80%, rejuv optional). Whether our `input`
helper can send shift+key is U (V-07 step 8). First version: no merc potions; revive when dead.

## 3. Policy summary

| difficulty | hire | when | gear |
|---|---|---|---|
| Normal before Act 2 | Act 1 Rogue from Kashya, only after Sisters' Burial Grounds | clvl 1-18 stopgap, cheap | buy a bow, helm, light armor |
| Normal Act 2+ | Desert Mercenary: Defiance/Prayer/Blessed Aim (Normal auras, SP) | as soon as Lut Gholein is reachable (~clvl 17) | Insight polearm clvl 27 (leveling.md V) |
| Nightmare | Desert Mercenary: Might or Holy Freeze (the plan prefers Might vs cold immunes) | at the first Lut Gholein visit | Insight / Fortitude armor |
| Hell | KEEP the Nightmare Might merc and revive it. Hell hire rows are Prayer/Defiance/Blessed Aim only: Might, Holy Freeze, Thorns exist only in Nightmare (V real 1.14d hireling.tsv; docs/merc-farm.md 1.1). Hire in Hell only when no merc lives (Defiance) | as above | Infinity polearm at clvl 63, Andariel's Visage, Fortitude |

Replace a merc when dead and `Cost <= gold * 0.5` (mercs.json hiring rule). A merc of the previous difficulty is *kept* until the new difficulty's Act 2
is reachable (a Normal merc is weak in Hell but better than none).

## 4. Dependencies and risks

- Needs the dialog layer and the unit-owner read: nothing else in the progression plan depends on the merc being present, so merc work can run in parallel with the quest handlers.
- The Fighter must treat the merc as friendly in all target selection (V nav.md already penalises merc cells in pathing; combat is COMBAT2's area, request R5).
- Offline accounting: a dead merc has no "death" event in our log; add one (`merc_died`) when Alive flips false inside a fight, for dashboards and for the revive-cost budget.
