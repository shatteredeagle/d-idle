# Backend contract specification

## Purpose
The seams that keep decision code backend-agnostic, so the same bot runs on the original Game.exe (Wine, /proc/pid/mem reads, input injection) and on the d2d reimplementation (control socket), and what each backend must provide. Source: docs/decisions.md 2026-10-30 rule 1, docs/d2d-backend.md, docs/d2d-bot.md, docs/d2d-wire.md (when merged).

## Requirements

### Requirement: Decision code stays backend-agnostic
New decision code MUST reach the game only through `env.Env` (`Read d2.StateReader`, `In input.Controller`, `Mem mem.Source`, `World *d2map.Tracker`, `Clock`, `Emit`) and the package seams, and SHALL NOT import `internal/d2dctl`, `internal/d2dsrc` or Game.exe specifics. Source: docs/decisions.md 2026-10-30 rule 1; internal/env/env.go (V read).

#### Scenario: New behaviour package
- **WHEN** a behaviour package needs the player's life
- **THEN** it calls `env.Snapshot()` or `Env.Read.Player()` and compiles without any d2d import

### Requirement: StateReader interface
Every backend MUST implement `d2.StateReader` with exactly the methods `Snapshot`, `Player`, `UI`, `Units`, `UnitStats`, `ReadStatList`, `ReadBaseStatList` and `RawStatList`, and `*d2.Reader` and the d2d snapshot source SHALL both satisfy it (compile-time assertion). Source: internal/d2/statereader.go (V read); merge 0fbb310 (d2diface).

#### Scenario: Interface drift
- **WHEN** a method is added to `StateReader`
- **THEN** both implementations fail to compile until updated

### Requirement: Input client
A backend MUST provide an `input.Controller`; the Game.exe backend uses the `d2input` helper over its per-instance port (47012 + N) and the d2d backend uses `internal/d2dinput` over the control socket, with the same semantic actions. Source: AGENTS.md; docs/d2d-backend.md section 2 (S); merge 3c0f26b (d2dinput).

#### Scenario: Instance selection
- **WHEN** `D2BOT_INSTANCE=i2` is set
- **THEN** `input.NewClient("")` targets the helper of instance i2 and no other

### Requirement: d2d control socket is single-client
At most one party SHALL drive a d2d control socket at a time, there is no authentication, and a bot MUST NOT attach to a d2d it did not start unless `--d2d-attach` is given. The managed start writes `<sock>.owner` and a stranger's d2d is never driven or quit. Source: docs/d2d-wire.md socket-safety section (V offline tests TestBackendFlags, TestManagedRefusesStrangerSocket, on branch d2dwire).

#### Scenario: Stranger socket
- **WHEN** `--managed` finds an answering socket without an owner marker
- **THEN** the start fails with an error and the other d2d is untouched

### Requirement: Per-instance sockets and paths
The default d2d socket MUST be per instance (`<run dir>/d2d-<D2BOT_INSTANCE>.sock`), never the shared lab socket, and each soak SHALL own its bot binary path (`~/d2lod-live/soak24-iN/d2bot`). Source: docs/d2d-wire.md, docs/soak24.md section 2 (V read).

#### Scenario: Two soaks
- **WHEN** soaks run on i4 and i5
- **THEN** they use different binaries, logs, ports and sockets

### Requirement: Backend gaps are named
Behaviour the d2d backend cannot provide yet (loot pickup, stat map, skill list, poison state) SHALL be wired as an explicit "not yet" stub that logs a warning, never as silent success. Source: docs/d2d-wire.md ("Loot NOT wired"), docs/d2d-backend.md section 1.2 (gap column).

#### Scenario: Missing capability
- **WHEN** the leveler needs base stats on d2d and `Mem` is `noMem`
- **THEN** it takes the "unreadable" path and logs it, and does not invent values

### Requirement: Original game remains the acceptance target
Anything verified only on d2d MUST be tagged V-d2d and is not an acceptance result until the fidelity-gate spec passes or the original game confirms it. Source: docs/decisions.md 2026-10-30; plan-v2 section 11.

#### Scenario: Probe result
- **WHEN** a d2d probe says the free Rogue is granted without a window
- **THEN** the doc tags it V-d2d and the original-game probe stays an open item

### Requirement: d2d belt keys drink by belt column
On the d2d combat input the keys "1" to "4" SHALL send `cmd belt <n-1>` (belt column 0 to 3) and count one potion per accepted drink; every other key MUST pass to the d2dinput controller unchanged, because the channel's key verb does not know the belt hotkeys. Source: card d2dcombat; internal/d2dcombat/input.go (Input.Key), internal/d2dctl/combat_verbs.go (Drink); test TestBeltKeysAndCasts.

#### Scenario: Key 2
- **WHEN** combat presses key "2"
- **THEN** the channel receives `cmd belt 1` and Stats().Potions is 1 (TestBeltKeysAndCasts)

### Requirement: A d2d cast targets the monster at the cell
For a hotkey that has a skill id, CastAt SHALL read `monsters` once and send `cmd skill <id> x y <unit>` with the nearest living monster within 1 cell of the target cell (subtile + 0.5, divided by 5), and with none send the cast at the ground cell without a unit. A hotkey with no skill id falls back to a key press then a right click. Source: card d2dcombat; internal/d2dcombat/input.go; tests TestBeltKeysAndCasts, TestUnmappedHotkeyFallsBack.

#### Scenario: Monster at the cell
- **WHEN** hotkey sk36 (skill 36) is cast at subtile (65, 147) where monster unit 5 stands
- **THEN** the lines are `monsters` then `cmd skill 36 13.1 29.5 5` and Stats().Casts[36] counts it

#### Scenario: Nobody there
- **WHEN** the same hotkey is cast at subtile (400, 400)
- **THEN** the lines are `monsters` then `cmd skill 36 80.1 80.1` with no unit

#### Scenario: Unmapped hotkey
- **WHEN** hotkey "f9" has no skill id
- **THEN** the first line sent is `key F9`

### Requirement: Combat skills are bound as pseudo hotkeys on d2d
BindSkills SHALL give every skill of the character that the combat catalog knows the pseudo hotkey `sk<id>`, install the mapping in the input (CastAt becomes `cmd skill <id>`) and in the Fighter, and return how many were bound, so no F-key or popup clicking is needed. UNVERIFIED by test: read from code (no unit test in internal/d2dcombat covers BindSkills). Source: card d2dcombat; internal/d2dcombat/rig.go.

#### Scenario: Two known skills
- **WHEN** the snapshot lists skill ids 36 and 40 and the catalog knows both
- **THEN** BindSkills returns 2 and the hotkeys are `sk36` and `sk40` (UNVERIFIED, no test)

### Requirement: The d2d snapshot gets a viewport for combat
The d2d combat reader SHALL fill Snapshot.Viewport from the player cell as VX = 16 (px - py) - 400 and VY = 8 (px + py) - 284 (px, py = position + 0.5), so a standing player projects to about pixel (400, 284); without it every target counts as off screen. That d2d draws its player at that pixel is UNVERIFIED. Source: card d2dcombat; internal/d2dcombat/viewport.go; test TestViewportPutsPlayerAtScreenAnchor.

#### Scenario: Player at (153, 68)
- **WHEN** the viewport is computed for a player at cell (153, 68)
- **THEN** the player projects to within one pixel of (400, 284)

### Requirement: The combat clock reads state after every sleep
The d2d combat clock (Meter) SHALL step the game through the Pacer on Sleep and then read `state` once, keeping Totals: Damage (sum of life drops), ManaSpent (sum of mana drops), Deaths (transitions to life 0), MinLife and Samples; a read that fails or reports max life 0 is not sampled. Only Samples and MinLife are covered by test; the other totals are UNVERIFIED offline. Source: card d2dcombat; internal/d2dcombat/meter.go; test TestMeterSamples.

#### Scenario: One sleep
- **WHEN** the Meter sleeps 80 ms once against the sample state
- **THEN** Totals shows Samples 1 and MinLife 45

#### Scenario: Reset after a cheat
- **WHEN** Reset is called and the next sample arrives
- **THEN** the jump from the old sample is not counted as damage (UNVERIFIED, no test)

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

### Requirement: The d2d link wait outlives the verb timeout
The d2d world's update bound SHALL be Options.Timeout (per-verb budget, default 10 s) plus Options.LinkWait, so an arrival wait for neighbour levels is not cut short by the timeout; before this nav reported "no exit" while the neighbours were still being linked. Source: card d2dexit; docs/d2d-exit.md (V by failing test); internal/d2dworld/world.go (ctx); test TestLinkWaitOutlivesTimeout.

#### Scenario: Neighbour appears late
- **WHEN** Timeout is 50 ms, LinkWait 5 s and the neighbour appears after 8 polls
- **THEN** pending ends at 0 and the level has at least 2 adjacent entries

### Requirement: The d2d backend waits for links by default
The d2d backend SHALL use a link wait of 90 s (d2denv.DefaultLinkWait) when Config.LinkWait is 0. Stepped lockstep games link neighbours only while stepped, so the wait advances the game. Source: card d2dexit; internal/d2denv/env.go. UNVERIFIED by test (live: Blood Moor reached in about 32 s, docs/d2d-exit.md).

#### Scenario: No link wait configured
- **WHEN** the backend is built with LinkWait 0
- **THEN** the world is created with LinkWait 90 s (UNVERIFIED, no test)

### Requirement: A missing layout verb is a warning
When the d2d binary has no `debug layout` verb (fork patch P3 missing), Backend.Start SHALL log a WARN `d2d has no 'debug layout' verb (fork patch P3 missing): outdoor exits cannot be found, nav will say 'no exit'`, and the arrival line carries the border note `no layout verb: no outdoor borders`. Source: card d2dexit; internal/d2denv/env.go (checkVerbs), internal/d2dworld/world.go (borders). UNVERIFIED by test; seen live with a binary lacking P3.

#### Scenario: Binary without P3
- **WHEN** the layout verb fails at start
- **THEN** the WARN is logged once and the run continues (UNVERIFIED, no test)

### Requirement: Arrival diagnostics are logged
On entering a level the world SHALL log `d2dworld: arrived in level N: <border note> (pending P, linkwait W)` and, when it waited for neighbours, `d2dworld: link wait over after <t>, <n> polls: <note> (pending P, linked [...])`. Source: card d2dexit; internal/d2dworld/world.go (update). UNVERIFIED by test; the text is read from code.

#### Scenario: Arrival with pending neighbours
- **WHEN** the player arrives in a level with pending neighbours and LinkWait above 0
- **THEN** both lines are logged (UNVERIFIED, no test)

### Requirement: The liveness beat grows only with game progress
Beat.Next SHALL return a counter that grows only when game_ms, tick, stepped or frame of `clock`, or the frame of `info`, differs from the last read, and an error only when both verbs fail; it feeds the monitor heartbeat (d2dHealth). Frames counting while lockstep idles is UNVERIFIED. Source: card d2dexit; internal/d2denv/beat.go; tests TestBeatCountsOnlyAdvance, TestBeatLockstepAnyCounter, TestBeatError.

#### Scenario: Wedged but answering
- **WHEN** game_ms rises once and then stays flat
- **THEN** the counter rises once and stays

#### Scenario: Lockstep
- **WHEN** only the info frame advances
- **THEN** the counter rises

#### Scenario: No answer
- **WHEN** clock and info both fail
- **THEN** Next returns the error

### Requirement: Unknown d2d fields are one log line per snapshot
When a snapshot sees fields never reported before, d2dsrc SHALL log ONE line `d2dsrc: N new unknown fields: a, b, ...` (sorted) and nothing further for the same field; reasons stay in Report.Unknown. Source: card d2dexit; internal/d2dsrc/report.go (flush); test TestLogOnce.

#### Scenario: Three reads
- **WHEN** three snapshots with the same unknown fields are read
- **THEN** exactly one summary line is logged
