# Act transitions

What the bot does to move between acts, per hop. Quest mechanics for the *reasons* a hop opens are in `docs/goals/execution.md` section 3; town NPC tables
in `docs/archive/prior-art/05-town.md`. Everything about Acts 2-5 dialogs, portals and tabs is **U** until the checklists V-03 to V-06 in verification.md run.

## 1. The two hop mechanisms

| mechanism | used for | verified |
|---|---|---|
| **NPC travel**: talk to a town NPC, click the travel row, loading screen, arrive at the next act's town | A1->A2 (Warriv), A2->A3 (Meshif), also back west (Warriv A2, Meshif A3) | NPC dialog open V (A1: Gheed, Akara, Charsi, town.md); travel rows U |
| **Quest portal**: a portal that appears after a boss kill and leads to the next act's town | A3->A4 (red portal after Mephisto in Durance 3), A4->A5 (Tyrael/portal after Diablo), plus Arcane/Anya/Baal portals | portal object use: not exercised (town portal reuse U, bot.md) |
| **Town waypoint**: once both town waypoints are active the panel's act tabs move you between acts for free | every later hop, any direction | panel + act-1 tab V; tabs 2-5 U (nav.md NAV2) |

Consequence: the *first* trip to each act is a scripted hop; every later trip is `TakeWaypoint`. The planner should therefore **touch the town waypoint
immediately after arriving in a new act** (step `ActivateTownWP`, 10 s) before anything else. Whether the town WP is already active on arrival: U (V-03 reads
the waypoint bits right after a cheated arrival; kolbot touches it on game start, prior-art 08 section 6, suggesting it is not).

## 2. Per-hop procedure

Each hop is a Plan fragment (tasklayer.md). `NeedActHop(from,to)` is true when the target act's quest-index "able to go" bit is set (T1/T2) and the current
act differs from the goal's act.

| hop | pre-state (must be true) | steps | success check | failure and recovery |
|---|---|---|---|---|
| A1 -> A2 | Andariel dead in this difficulty (index 6 and 7); in Rogue Encampment | 1 town portal home if in the field. 2 `GoToUnitOf(Monster, Warriv)`; Warriv is class 11 in A1 (V nav.md: id 11 collides with the waypoint, so use the typed lookup). 3 `Talk(Warriv)`: dialog flag 0x08, pick the "Go East" row. 4 wait for area change to Lut Gholein (area 40, U). | `Player.Area == 40` (V check method) | row not found after 3 tries: close (Esc only while dialog flag set), re-approach; still failing: T1 re-read (maybe Andariel is not credited), then mark the hop blocked |
| A2 -> A3 | Duriel dead and Tyrael/Jerhyn talks done (index 14, 15); in Lut Gholein | 1 Duriel's Lair portal back, `ToTown`. 2 Talk Tyrael/Jerhyn if the "Seven Tombs" bit is not set (the Tyrael talk ends the quest; Jerhyn in the palace, U). 3 `GoToUnitOf(Monster, Meshif)` (A2 harbour, 5205,5058 Kolbot table, U coordinates: use unit scan). 4 `Talk(Meshif)`, row "Sail East". 5 area change to Kurast Docks (75, U). | area == 75 | as above; if Meshif absent (he is a ship NPC in A2 only after the quest) re-read T1 |
| A3 -> A4 | Mephisto dead (index 22); inside Durance of Hate 3 | 1 after the kill the red portal object appears within the arena (Kolbot (17590,8068) fixed layout, U); `GoToUnitOf(Object, redportal)`; `Interact`. 2 area change to Pandemonium Fortress (103, U). 3 `ActivateTownWP`. | area == 103 | portal not seen in 30 s: rescan objects, then walk to the Kolbot spot; if the player already left the area (TP), use the Durance 3 waypoint-less return: see below |
| A4 -> A5 | Diablo dead in Chaos Sanctuary (index 27) | 1 pick loot, `ToTown` (A4: Pandemonium Fortress). 2 `Talk(Tyrael)` ("Terror's End" completion; Tyrael stands at (5027,5027) in the Kolbot table, U). 3 the Tyrael portal / dialog to Harrogath (U exact UI). 4 area change to Harrogath (109, U). 5 `ActivateTownWP`. | area == 109 | needs the expansion; offline "classic" characters stop here (character expansion flag, status byte 0x24 bit 0x20 V layout) |
| back-hops | any later time | waypoint panel: `TakeWaypoint(<town of act n>)` | area == town | tab geometry U (V-06) |

Rescue from a closed hop: if the player left the quest area before pressing the portal (death, chicken, restart), the portal object is gone with the game.
Mephisto and Diablo are **re-killable only in a new game** (monsters respawn per game, V general rule), but the quest bit stays set; the A3->A4 and A4->A5
"able to go" bits mean the *town* has the travel row (Tyrael at the Fortress, Meshif at the harbour) rather than a portal on the field. U: the Fortress is
reached only through the Durance 3 portal, even in a later game; V-04 checks whether a fresh game after the kill offers another way.

## 3. The Talk primitive (shared by hops, quests, merc, difficulty)

Verified pieces (V town.md): open via `input.Interact`, UI flag 0x08, the dialog is a bottom-anchored text list floating above the NPC, lines 15-16 px
apart, last entry Cancel, drawn where the NPC stood when it opened (use the snapshot taken at the flag edge), tolerance retry offsets 0, +5, -5, +9, -9.
First contact of an NPC may play a greeting (speech box, Akara V): the second contact opens the menu. Unverified: option *text* (cannot be read from memory),
option *order* per NPC, option *count* after a quest changes the menu.

Design:

```go
type Dialog interface {
    Open(ctx, npc UnitRef) (Menu, error)         // approach, Interact, wait 0x08, handle greeting
    Pick(ctx, Menu, Choice) error                 // by Index from the top, or by From-bottom offset
    Close(ctx) error                              // click Cancel row, verify 0x08 cleared
}
type Choice struct{ Label string; FromTop, FromBottom int } // Label for logs; index for clicking
```
Reading rows: the box has N rows; N is unknown in memory, but N is stable per (NPC, quest state), so the table is learned once: V-05 records for each NPC the
count and the row index of each useful option, then ships as `internal/town/dialogtable.json` keyed by NPC class id and a state key (T1 quest bits that change
the menu). Until then `Pick` uses `FromBottom` (Cancel is always the last row; the travel/quest row is usually directly above it, U) and verifies by the expected
effect (area change, flag 0x0C, item appeared) rather than by text. Without a verifiable effect the step fails instead of guessing.

Menu options cause game effects the bot cannot undo (accept travel, imbue an item, choose a socketing target). Every `Pick` that changes the world is a
step of its own with `Need` false once the bit says done.

## 4. Per-act town facts the hops depend on

(U unless noted. Class ids come from `internal/data` monstats at setup; coordinates are Kolbot's and *not used*: units are found by scan, V nav2 seed independence.)

| act | town (area id U) | needed NPCs for progression | waypoint | note |
|---|---|---|---|---|
| 1 | Rogue Encampment (1, V) | Warriv, Akara (V), Kashya (merc), Charsi (V), Cain | V (class 119, V nav.md) | town positions differ per character (V bot.md); Act 1 town layout is one of several presets |
| 2 | Lut Gholein (40) | Jerhyn, Meshif (A2->A3), Greiz (merc), Atma, Drognan, Fara, Elzix | U | Atma's house door blocks (Kolbot side-door nodes, prior-art 05); Warriv here for the way back |
| 3 | Kurast Docks (75) | Ormus, Alkor, Hratli, Asheara (merc, skip), Meshif here for the way back | U | the red portal from Durance 3 does not arrive in A3 |
| 4 | Pandemonium Fortress (103) | Tyrael, Jamella, Halbu | U | no merc NPC, no travel NPC except Tyrael |
| 5 | Harrogath (109) | Qual-Kehk (merc, Rite of Passage), Larzuk, Malah, Anya, Nihlathak portal | U | A* trouble in A5 town (Kolbot hard-codes paths, prior-art 05); Anya's red portal appears after the Prison of Ice |

Waypoint lists by act (V for the nine-per-act count shape in tracking.md; names U): the panel has one tab per act; rows for inactive waypoints are
not shown or not clickable (tracking.md says a new character shows only the act-1 tab and "No Other Waypoints Activated", V nav.md).

## 5. Cross-act travel for goals

`Travel(area)` in the step layer resolves in this order: same act -> existing nav (`ToArea`); other act and both town WPs active (T1/T2 waypoint bits) ->
`ToTown`, `TakeWaypoint(destination act's town or the destination waypoint directly)`; other act, destination act's WP not active, first visit -> the hop
table above; destination act not unlocked -> the goal is blocked (`Ready` returns `KindPrereq`), the planner inserts the missing quest chain.

Config field `acts` (bot.md, default `[1]`, "act changes need quests") becomes the *allowed ceiling*, enforced by the planner (planner.md rule 1), not a
bot-runner mechanism. The runner never decides to change act; a Plan step does.
