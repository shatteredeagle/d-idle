# OOG: menus, start, leave, crash restart (agent OOG)

Code: `internal/launch/oog.go` (Flow: Start, LeaveGame, CreateCharacter), `oog_table.go` (coordinates), `oog_keeper.go` + `oog_wire.go` (restart after crash). Live tools: `cmd/oog` (explorer/driver, run with `tools/live.sh oog oog 'cmd' 'cmd' ...`), `tools/game.sh status|start|stop`.
All values below VERIFIED LIVE on the worker host 2026-10-28 (1.14d, GE-Proton11-11, 800x600 windowed) unless marked UNVERIFIED.

## Key facts
- Save and Exit does NOT end Game.exe: it returns to the MAIN MENU (not char select). The process stays; "Exit Diablo II" on the main menu ends it (process gone within 3 s; the d2input helper keeps listening and finds the next window by itself).
- Menu screens ARE a D2Win control list at FirstControl (RVA 0x3D55BC). In game FirstControl is 0 and the Esc menu is NOT in the list: found via UI array [9] (RVA 0x3A27C0).
- Control struct (dumped and matched against screenshots): +0 type, +0xC x, +0x10 y (BOTTOM), +0x14 w, +0x18 h, +0x3C next, +0x48 ControlText -> wchar_t*. Click point = (x+w/2, y-h/2). The D2BS "disabled" bits do not hold (+8 was 5 on every control); not used.
- Posted clicks (move, down, up) work on every menu without focus; no real cursor needed. `char X` (WM_CHAR) fills the name box, case preserved; key presses were not tried for the name.
- Entering a game from char select takes <1 s (player pointer set 300 ms after OK). No difficulty dialog appeared for a new or Normal-only character (the difficulty screen stays UNVERIFIED).

## Coordinates (click point; control rect where it exists)
| screen / element | click | control (type,x,y,w,h) | how |
|---|---|---|---|
| Esc menu OPTIONS / SAVE AND EXIT / RETURN TO GAME | (400,210) / (400,261) / (400,310) | none | screenshot, then click (400,261) saved and exited |
| Main menu Single Player | (400,307) | 6,264,324,272,35 | dump + click |
| Main menu Battle.net (never used) / Other Multiplayer / Exit | (400,~349) / (400,416) / (400,551) | 6,264,366 / 433 / 568 ,272,35 | dump; Exit click ended the process |
| Char select Create / Convert / Delete | (117,498) / (317,498) / (517,498) | 6,33/233/433,528,168,60 | dump; Create click opened the create screen |
| Char select OK / Exit | (691,555) / (97,555) | 6,627,572 / 33,572,128,35 | dump; both clicked |
| Char select slots | slot i: col i%2 x=37+272c, row i/2 y=178+93r, 200x92; click centre, e.g. slot0 (137,132), slot1 (409,132) | 4,37/309,178..457,200,92 | dump; clicking slot1 selected Testy |
| Char select selected-name header | read only | 4,85,78,466,42 text = name | dump (slot controls carry NO text) |
| Create: class header (picked class text) | read only | 4,0,180,800,100 | dump |
| Create: name box / expansion box / hardcore box | (396,502) / (326,532) / (326,552) | 1,318,510,157,16 / 6,319,540,15,16 / 6,319,560,15,16 | dump + screenshot (expansion ticked by default, hardcore not). No ladder box offline. Toggling the boxes was NOT tried (UNVERIFIED) |
| Create: OK / Exit | (691,555) / (97,555) | as char select | OK created Botty and entered the game |
| Create: classes amazon, assassin, necromancer, barbarian, paladin, sorceress, druid | (100,245) (200,300) (301,241) (400,238) (521,247) (626,261) (720,278) | portraits are type-2 images, NOT the hit areas | each point clicked, class header read back (assassin (232,272) failed, kolbot's numbers wrong for most) |
| Popup (name taken) | button "Cancel" (399,321) | 6,351,337,96,32 | triggered safely by creating "Testy" again; text "That character name is already taken. Try another." |
| Splash (copyright) | none | 5 controls incl. type-8 (0,0,0,0) and label "(c) Copyright 2000 - 2016..." | watch loop |
| Difficulty buttons, Yes/No popup | kolbot values in the table | | UNVERIFIED (never shown) |

## LAN (TCP/IP) screens (agent PARTYLIVE, VERIFIED LIVE 2026-10-28 on i1/i2; details docs/net/multiplayer.md section 9)
Flow: Main menu "Other Multiplayer" (400,416) -> `multi-menu` (TCP/IP Game (400,332), Cancel (400,550); the top button is Open Battle.net and is deliberately not in the table) -> `tcp-options` (Host (401,189), Join (401,247), Cancel (103,553), label with the machine's IP at (265,171)) -> Host: char select (same controls as single player) -> OK -> in game, no name/password/difficulty dialog. Join: modal `tcp-join-ip` (edit (406,258), OK (469,321), Cancel (329,321); it must be identified before the generic two-button popup) -> clear the box with backspaces (it keeps and appends the previous text) -> type the IP with WM_CHAR -> OK -> char select -> in game. Failure: popup-ok "Cannot Connect to ...".
`Flow.HostGame(ctx, CharSpec)`, `Flow.JoinGame(ctx, CharSpec, ip)` (CharSpec.LAN/HostIP); `LeaveGame` from a LAN game ends on `tcp-options` (not the main menu) and accepts it. `cmd/oog`: `host NAME`, `join NAME IP`, `players` (unit tables + roster of the attached game), `party id:Char:role ...` + `pstatus|pplayers|pleave|plost|prejoin|pdisband` (real `party.Party` over `party.FlowDriver`, two instances in one process).

## Screen detection
`ReadPhase`: PlayerUnit (RVA 0x3A6A70) and FirstControl: in-game / menu / neither (loading). In the menu phase `Identify` matches signature rects (type,x,y,w,h) against the control list: popups first, then difficulty, char-create (name edit box), char-select (Create+Delete buttons), main menu (Single Player+Exit), splash (only the background image). Create and class pick are ONE screen. In-game identity: PlayerName (player +0x14 -> name at +0).

## Timings (measured)
Launch to Game.exe process 2 s; splash 10 s with no input (first controls 0.3 s after the process); main menu at ~10.2 s; Single Player to char select <2 s; OK to in-game <1 s; Save and Exit to menu phase ~2 s. Class click: a click within ~1-2 s of the previous class click is dropped; wait 2.5 s and verify the header, retry (Flow does, 4 tries).

## Flows (internal/launch)
- `Flow.Start(ctx, CharSpec)`: from splash/loading/main menu/char select/create/popup (or in game as another character: leaves first). Select by name = click each of 8 slots and read the header; `Create` makes it when missing; `ForceNew` always creates. Bounded: ScreenTimeout 20 s per unchanged screen (ErrStuck), LoadTimeout 60 s (ErrTimeout), no blind click loops. Returns when PlayerName matches twice, 300 ms apart.
- `Flow.LeaveGame`: reads UI array; Esc only when the Esc menu is not open (Esc with nothing open OPENS it, with a panel open it closes the panel, so it is pressed again), clicks (400,261) once, waits for menu phase then main menu.
- `Flow.CreateCharacter(name, class)`; `ErrNameTaken` dismisses the popup.
- `Keeper` (Monitor + Kill + Launch + Ready + Flow.Start, crash-loop guard 3 per 10 min = ErrCrashLoop); `Wire` builds it for the real system. Offline-tested with fakes; the real crash path was NOT run (no crash was forced). Not verified: what 1.14d shows on a crash (BlizzardError.exe presence is still the assumption).
- Limits: more than 8 characters needs the scrollbar (type 5 at 585,457), not implemented; expansion/hardcore toggling and the difficulty screen unverified; Start expects softcore/expansion defaults.

## Muting (user rule)
Default game args are `-w -ns` (`Config.MuteOff` disables). VERIFIED: with `-ns` the relaunched game (`Game.exe -w -ns`) has NO PipeWire stream at all (`pactl list sink-inputs` contains no "Diablo II" entry), so there was nothing to mute; `tools/mute-d2.sh` ran after launch and muted nothing. Two earlier launches in this session (before the rule arrived) ran without `-ns`; both were closed.

## Restarting the game safely
1. Leave: `tools/live.sh oog oog leave`, or Esc + (400,261). 2. Main menu Exit (400,551), confirm `tools/game.sh status`. 3. `tools/game.sh start` (detached via setsid, sets DISPLAY=:1/WAYLAND_DISPLAY/XAUTHORITY: WITHOUT the desktop session env the game dies within 2 s, found the hard way). 4. `tools/helper.sh status`. 5. `tools/live.sh oog oog 'attach 60000' 'start Testy'`. A game started this way is not our child: readable only with the setcap ptrace tools, as before.

## Dead ends
- kolbot class coordinates (mostly wrong); y of Create/Exit mixed up once (bottom anchored: Create centre y=498, Exit 554); slot labels have no text; D2BS disabled bits; a first game.sh without the session env.
