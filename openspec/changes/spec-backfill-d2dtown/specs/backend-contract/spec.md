## ADDED Requirements

### Requirement: The d2d town source offers every seam the town reads
Adapter.Source() SHALL return a town.Source with Snapshot, Items, Owner, Stats, MenuEntry and MenuIndex set and FixedTabs true, so internal/town runs on d2d without changing its Game.exe behaviour. Source: card d2dtown; internal/d2dtown/adapter.go; test TestSourceHasSeams.

#### Scenario: Fresh adapter
- **WHEN** a source is built from a new Adapter
- **THEN** none of those functions is nil and FixedTabs is true

### Requirement: Vendor stock becomes stored item units
While the shop window is open Items SHALL list the carried items plus each vendor item as an item unit in mode stored with Class from the item row of its code (-1 without tables), Page = tab, X and Y = grid cell and a minted address from 0x50000000; when the vendor dump is missing only carried items are listed. Source: card d2dtown; internal/d2dtown/adapter.go (vendorUnit, items); test TestVendorUnitsAndOwner.

#### Scenario: Healing potion in tab 3
- **WHEN** the dump lists `hp1` at x 1, y 0 in tab 3 and the table maps hp1 to 587
- **THEN** the unit has Class 587, mode stored, Page 3, X 1, Y 0

### Requirement: Item owner is the vendor NPC or the player
Owner SHALL return NPCIDBase plus the vendor's npc index for an item whose minted address is at or above 0x50000000, and the fixed player address 0x7fffffff for every carried item. Source: card d2dtown; internal/d2dtown/adapter.go (owner); test TestVendorUnitsAndOwner.

#### Scenario: Vendor and carried item
- **WHEN** the dump names npc 4 and Owner is asked for its stock item and for a carried item with address 7
- **THEN** the answers are NPCIDBase+4 and 0x7fffffff

### Requirement: Item stats are synthesised from the dump extras
Stats SHALL give stat 70 (quantity) when the extras quantity is above 0 and stats 72 and 73 (durability, maximum) when the maximum durability is above 0, and MUST return d2.ErrUnsupported before any snapshot was read or for an unknown item. Source: card d2dtown; internal/d2dtown/adapter.go (stats); test TestStatsFromExtras.

#### Scenario: Quantity and durability
- **WHEN** one item has Qty 5 and another Dur 12 of MaxDur 24
- **THEN** stat 70 reads 5 on the first and stats 72 and 73 read 12 and 24 on the second

#### Scenario: Nothing read yet
- **WHEN** Stats is called on a new Adapter
- **THEN** it returns an error

### Requirement: Dialog entries come from the menu verb
MenuEntry SHALL find the line by its text (talk, trade, gamble, cancel, resurrect) in the `menu` reply and return its point, and MenuIndex(i) SHALL return the point of line i+1 (line 0 is the NPC name) or false when i+1 is past the last line. Source: card d2dtown; internal/d2dtown/adapter.go; test TestMenuSeams.

#### Scenario: Akara menu
- **WHEN** the menu lists Akara, talk (580,205), trade (580,220), cancel (580,235)
- **THEN** EntryTrade gives (580,220), MenuIndex(0) gives Y 205, MenuIndex(3) gives false and EntryGamble gives false

### Requirement: Named objects get the class the town searches for
A "npcs" object named "Your Private Stash" SHALL get class town.ObjStash and one named "Waypoint" the waypoint class of the town's act (area 109 and up act 5, 103 act 4, 75 act 3, 40 act 2, else act 1); other objects keep class -1. Source: card d2dtown; internal/d2dtown/adapter.go (classObjects, waypointClass); test TestClassObjects.

#### Scenario: Act 2 town
- **WHEN** the player is in area 40 and the objects are the stash, a waypoint and "Something"
- **THEN** their classes are ObjStash, ObjWPAct2 and -1

### Requirement: Clicks on d2d hover first
The town input SHALL precede every unmodified left or right click by MoveMouse(x-3, y), a 120 ms wait, MoveMouse(x, y) and a 350 ms wait, and MUST NOT hover before a click with a modifier; the waits use the controller's own clock when it has one (lockstep) and wall time otherwise. Source: card d2dtown; docs/d2d-town.md (V-d2d: a click without a prior hover is ignored); internal/d2dtown/adapter.go (HoverFirst); tests TestHoverFirst, TestHoverFirstWaitsOnTheControllerClock.

#### Scenario: Plain and shifted click
- **WHEN** a plain click and a shift click are sent
- **THEN** the events are move, move, click, click

#### Scenario: Controller with a clock
- **WHEN** the controller has Wait
- **THEN** it is asked for 120 ms then 350 ms and no wall time passes

### Requirement: Town interactions walk to the target
In the town input, Interact on an NPC target (ID from NPCIDBase, not the mercenary) SHALL go through ApproachNPC (15 s), a portal target (ID from PortalIDBase) through EnterPortal, and any other target to the wrapped controller. Only the NPC and pass-through routes are covered by test; the portal route is UNVERIFIED offline. Source: card d2dtown; internal/d2dtown/adapter.go (townInput.Interact); test TestInputRoutesInteract.

#### Scenario: NPC and other target
- **WHEN** Interact gets NPCIDBase+7 and then id 99
- **THEN** the first walks via ApproachNPC without calling the wrapped controller and the second reaches it
