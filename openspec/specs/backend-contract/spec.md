# Backend contract specification

## Purpose
The seams that keep decision code backend-agnostic, so the same bot runs on the original Game.exe (Wine, /proc/pid/mem reads, input injection) and on the d2d reimplementation (control socket), and what each backend must provide. Source: docs/decisions.md 2026-10-30 rule 1, docs/d2d-backend.md, docs/d2d-bot.md, docs/d2d-wire.md (when merged). Product and oracle (user, 2026-11-04): d2d is the product (native, no Wine); the original Game.exe is the fidelity oracle that the A:B comparison measures d2d against.

> status: reconciled 2026-11-04 against docs/process/spec-ledger-20261104.md; pending deltas: d2d-default-backend (dropped by the user; its rows are not folded, see the end of this file)

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

### Requirement: Game.exe is the fidelity oracle, d2d is the product
d2d (native, no Wine) SHALL be the end product and Game.exe the fidelity oracle: a result verified only on d2d MUST be tagged V-d2d and counts as equal to the original game only when the A:B comparison (goal G-D) says EQUIVALENT. Acceptance: Act 1 cleared for 24 h on d2d AND A:B EQUIVALENT. Source: user 2026-11-04 (replaces the ADR 0015 framing; needs an ADR 0015 amendment). Reconciled 2026-11-04 (source: decision 4).

#### Scenario: Probe result
- **WHEN** a d2d probe says the free Rogue is granted without a window
- **THEN** the doc tags it V-d2d and the oracle (original-game) probe stays an open item until the A:B comparison or an oracle probe covers it

#### Scenario: Accepted
- **WHEN** the bot has cleared Act 1 for 24 h on d2d and the A:B verdict is EQUIVALENT
- **THEN** the backend is accepted; either condition alone is not acceptance

### Requirement: Leaving d2d is the user's decision
If the A:B verdict (G-D) is DIFFERENT and the gap needs a d2d engine rewrite, the user SHALL decide whether to leave d2d, never automatically; toolchain failures are tasks, not an exit trigger. Source: ADR 0009 exit trigger. Reconciled 2026-11-04 (source: decision 4).

> OPEN QUESTION: confirm break-glass wording. The user only discussed this in chat. The old trigger (about 16 agent-hours, metric past 20 percent) is replaced by the G-D verdict.

#### Scenario: Different verdict and an engine rewrite
- **WHEN** the A:B verdict is DIFFERENT and the gap needs an engine rewrite
- **THEN** the coordinator asks the user and nothing switches backend until the user answers

#### Scenario: Toolchain failure
- **WHEN** the container build fails on the C++26 toolchain
- **THEN** it is handled as a task (other compiler or base image), not as a reason to leave d2d

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

### Requirement: The portal verb picks a source with charges
The town portal verb SHALL click a tome only when its quantity (stat 70, absent counts as 0) is above 0, else a scroll, and with neither SHALL return the no-portal error without a click. Combat's picker and the d2d town portal SHALL use the same pick. Source: docs/primitives-review.md F4; docs/fixportal.md (how charges are read per backend).

#### Scenario: Empty tome and two scrolls
- **WHEN** the pack holds an empty tome and two scrolls
- **THEN** the portal verb clicks a scroll

#### Scenario: Empty tome and no scroll
- **WHEN** the pack holds only an empty tome
- **THEN** no click is made and the verb returns the no-portal error

### Requirement: HasTP counts charges
The kit input `HasTP` SHALL be true only when the pack holds a scroll or a tome with charges above 0, so an empty tome makes the bot buy scrolls through the existing refill rule. Source: docs/primitives-review.md F5.

#### Scenario: Empty tome
- **WHEN** the only portal item in the pack is a tome with no charges
- **THEN** `HasTP` is false

### Requirement: The d2d backend enters a town portal with the client's portal command
On d2d, interacting with a town portal object SHALL send the command the client sends for a click on that portal (Interact with id -2000 minus the portal's index), never a screen click computed from a transform, and the portal object's unit id SHALL carry the World's portal index. Source: docs/d2d-portal-entry.md (d2d resolves a world click through the hover pick of the last drawn frame; a lockstep d2d draws none, so the click walked instead, drills D2.t and D2.g).

#### Scenario: Portal interaction on a lockstep d2d
- **WHEN** the town layer interacts with the town portal object in the field
- **THEN** the control socket receives `cmd interact -2000` for portal 0 and no click

#### Scenario: Portal in town
- **WHEN** the portal in the current level is index 1 (the town twin)
- **THEN** the command names -2001

### Requirement: The waypoint unit is found on d2d
On the d2d backend the state reader SHALL give every npcs row named "Waypoint" an object class that `d2map.IsWaypointObject` accepts, so the navigator's nearest-waypoint lookup finds the unit of every act. A named object that is not a waypoint SHALL keep class -1. Source: docs/research/waypoint-trace.md W1 (295 of 295 failures were `unit not found`).

#### Scenario: Waypoint row
- **WHEN** the npcs list holds a row named "Waypoint"
- **THEN** the snapshot has an object of a waypoint class at its position and the navigator returns it as the nearest waypoint

#### Scenario: No waypoint near
- **WHEN** the snapshot has no waypoint object
- **THEN** the waypoint lookup reports unit not found

### Requirement: Waypoint rows are clicked at their panel centres
The waypoint verb SHALL click the tab of the target act and then the row at the centre of game.exe's hit box: tab centres x 112, 176, 240, 304, 368 at y 77, row centres y 135, 171, 207, 243, 280, 316, 352, 388, 424 at x 230, for rows 0 to 8 of the act's Levels.txt waypoint order. Source: docs/research/waypoint-trace.md W2.

#### Scenario: Row 8 of an act
- **WHEN** the verb travels to the ninth waypoint of an act
- **THEN** it clicks that act's tab and then (230, 424)

### Requirement: A bounced step away from a cave mouth changes direction
The navigator SHALL, when a step away from a tile exit ends in another level, step in a different direction from that exit on the next attempt instead of repeating the same step. Source: docs/probes/postfix-soak1.md (6 -> 20, 16 times).

#### Scenario: Bounce then retry
- **WHEN** the character starts a walk inside the trigger of an exit it does not want and the first step away fires that exit
- **THEN** the next step away from the same exit is rotated away from the first direction

#### Scenario: Memory expires
- **WHEN** ten minutes have passed since the bounce
- **THEN** the step away is the straight line away from the exit again

### Requirement: Touched waypoints reach the planner
A waypoint the navigator touched or travelled to SHALL count as active for the level picker and the travel planner of the same character and difficulty. Source: docs/probes/postfix-soak1.md (Black Marsh touched at 15:23, still `no activated waypoint`).

#### Scenario: Touched by the town trip
- **WHEN** the waypoint-to-town trip opened the waypoint of area 6
- **THEN** the next activeWaypoints read includes area 6

### Requirement: The d2d state source lists missiles
The d2d state source SHALL fill `Snapshot.Missiles` from `debug missiles` with one unit per moving missile (speed above 0; Class, OwnerType, OwnerID, subtile X Y) and SHALL exclude speed 0 hit-effect rows. A d2d that answers `err` SHALL be asked once. Source: cards fix-missile, fix-missile2; internal/d2dsrc/missiles.go; tests TestMissilesListed, TestMissilesRealVerb.

#### Scenario: A shooter's missile heads at the player
- **WHEN** the listing holds a missile owned by monster 752 and one owned by the player
- **THEN** the snapshot has two missiles, the first with OwnerType 1 and OwnerID 752, the second with the player's own id, and the combat dodge treats only the first as hostile

#### Scenario: A hit effect is listed
- **WHEN** the listing holds a speed 0 row with owner -1 next to a moving shaman bolt
- **THEN** the snapshot holds only the bolt

#### Scenario: Stock d2d without the verb
- **WHEN** the d2d answers `err` to `debug missiles`
- **THEN** every snapshot has no missiles and the verb is sent once

### Requirement: Field objects are listed and operable on d2d
The d2d state source SHALL list placed objects within 40 cells of the player (chests, shrines, the cage, the Tristram portal) with the objects.txt class of their preset row, and the d2d input SHALL operate one with `cmd interact <index>`, never a walk onto it. Source: drill H.obj. Reconciled 2026-11-04 (source: 2026-10-31-fix-obj).

#### Scenario: Chest near the player
- **WHEN** the level has a chest preset within 40 cells of the player
- **THEN** the snapshot lists an object with the chest's class and Interact on it sends `cmd interact` with the preset index

#### Scenario: Hidden or far preset
- **WHEN** a preset row is hidden or farther than 40 cells
- **THEN** it is not listed

### Requirement: Waits step the game under lockstep
The d2d backend SHALL make the pauses of the hover, menu and NPC-approach verbs advance the game by the requested time when d2d runs under lockstep, and SHALL keep them wall sleeps that end with the context on a free-running d2d. Source: docs/probes/drills/2026-10-31-h.txt (H.pause). Reconciled 2026-11-04 (source: 2026-10-31-fix-obj).

#### Scenario: Lockstep pause
- **WHEN** a pause of 4 seconds is requested under lockstep
- **THEN** the game is stepped by 4 seconds of ticks and almost no wall time passes

#### Scenario: Free-running pause
- **WHEN** a pause is requested on a free-running d2d
- **THEN** no step command is sent and the pause lasts the requested wall time

### Requirement: The d2d portal verb never clicks with a dead context
The d2d town portal verb SHALL check its context before every try and, when it is cancelled, SHALL return an error that names the cancel cause without pressing a key or clicking. Source: docs/research/town-portal-trace.md (134 of 778 failures were three tries in 10 ms with a cancelled context). Reconciled 2026-11-04 (source: 2026-10-31-fix-portal2).

#### Scenario: Cancelled context
- **WHEN** the portal verb is called with an already-cancelled context
- **THEN** no key and no click is made and the error names the cause

### Requirement: The d2d portal verb clicks only when the player is idle
On d2d the portal verb SHALL NOT click while the player is in hit recovery, attack, block, cast, throw, kick, skill, sequence or knockback mode, because d2d drops the click silently. It SHALL wait up to 2 seconds for an idle mode and then fail with a busy error. Source: docs/research/town-portal-trace.md answer 3. Reconciled 2026-11-04 (source: 2026-10-31-fix-portal2).

#### Scenario: Hit recovery ends within the wait
- **WHEN** the player is in hit recovery when the verb starts and becomes idle within 2 seconds
- **THEN** the click happens after the player is idle

#### Scenario: The player stays busy
- **WHEN** the player is still attacking after 2 seconds
- **THEN** no click is made and the error says the player is busy

### Requirement: The d2d portal verb succeeds only on a portal object
The d2d portal verb SHALL report success only when a town portal object (class 59) is within 25 subtiles of the player, never because the scroll left the pack. When the source was spent and no object appeared, it SHALL re-read the pack and retry with the next source; with none left it SHALL return the no-portal error without another click, and failure lines SHALL name the cause. Source: docs/research/town-portal-trace.md. Reconciled 2026-11-04 (source: 2026-10-31-fix-portal2).

#### Scenario: Cast cut by a hit
- **WHEN** the first scroll is spent and no portal appears while a second scroll is in the pack
- **THEN** the second scroll is clicked and its portal counts as success

#### Scenario: Last source spent by a cut cast
- **WHEN** the only scroll is spent and no portal appears
- **THEN** no second click is made and the error is the no-portal error

### Requirement: Deadlines of game waits are game time under lockstep
The d2d verbs that wait for the game up to a timeout (the NPC approach) SHALL measure that timeout on the clock of the run: the Pacer's virtual time under lockstep, the wall clock on a free-running d2d. Source: docs/probes/waits-audit.md. Reconciled 2026-11-04 (source: 2026-10-31-waits-audit).

#### Scenario: Lockstep NPC approach that never opens
- **WHEN** the NPC approach has a 6 second timeout under lockstep and the menu never opens
- **THEN** it gives up after about 6 virtual seconds of stepped pauses, however little wall time that took

#### Scenario: Free-running approach
- **WHEN** the clock is the wall clock
- **THEN** the deadline is the wall time, as before

### Requirement: One call installs the lockstep-aware waits
The d2d backend SHALL offer one call that detects lockstep and installs the pause and the deadline clock of the verbs, and every d2d program that calls those verbs (the bot, the lab shop tool) SHALL use it. Source: docs/probes/waits-audit.md. Reconciled 2026-11-04 (source: 2026-10-31-waits-audit).

#### Scenario: Install under lockstep
- **WHEN** the call runs against a lockstep d2d
- **THEN** a later pause steps the game and the deadline clock advances by the paused time

#### Scenario: Install on a free-running d2d
- **WHEN** the call runs against a free-running d2d
- **THEN** a later pause is a wall sleep that sends no step command and the deadline clock is the wall clock

## Pending deltas (not in force)
- d2d-default-backend (open card, proposed; the user dropped and archived it 2026-11-04): its MODIFIED "Original game remains the acceptance target" and ADDED "Game.exe is frozen" are SUPERSEDED by the oracle wording above and are not folded; its "Leaving d2d is break glass" is replaced by "Leaving d2d is the user's decision".
- The container-deferred delta (d2d-container-soak) targets soak-gates, not this spec; the container is deferred (docs/decisions.md Current truth) and the delta is not folded here.
