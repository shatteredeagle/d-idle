# 08 Out-of-game flow (offline single player)

Sources: KB:libs/OOG.js (ControlAction, Starter, login flows), KB:libs/modules/Control.js (800x600 control geometry), KB:libs/oog/Locations.js (location handlers), SP:OOG/SoloEntry.js + OOGOverrides.js (difficulty), KO:internal/helper/game_manager.go (D2R), BT:src/game_recovery.py, TY:docs/features/{character-selection,offline-game-exit,offline-difficulty-selection}.md.
Everything here is for Single Player only (no Battle.net, no passwords, no realms). Kolbot's TCP/IP, B.net login, account creation, channel and mule flows are irrelevant.

## 1. State model
Kolbot asks the host for `getLocation()` = menu screen id (enum `sdk.game.locations`): SplashScreen, MainMenu, (Single Player) CharSelect, CharSelectNoChars, CharSelectConnecting, CharSelectPleaseWait, SelectDifficultySP, OkCenteredErrorPopUp, CreateChar..., plus in-game. A loop runs `while not in game: switch(location) -> handler`, with 100 ms sleeps.
Ours: no getLocation. Need a menu-state detector (UI memory; TY uses image anchors + memory-confirmed phase: `in_game`, `menu`, loading). Options for `internal/launch`: (a) read a "game state / current UI screen" global from memory (candidate: the `ui` control tree / FirstControl is VERIFIED LIVE per PLAN: "FirstControl"); D2BS enumerates controls via `getControl(type,x,y,w,h)` which are the same linked list: a screen can be identified by which controls exist at known geometry. This is exactly how Kolbot's `Controls` module works: every button is defined by (type, x, y, w, h) and located by exact match. => Implementation: walk the control list from FirstControl, match on (type,x,y,w,h), identify screen by presence of signature controls; click at control center. Fully external-compatible, no pixels needed. (memory doc 2.x / `internal/launch` OOG)

## 2. Control geometry at 800x600 (KB `Control.js`)
(type id: Button/TextBox/LabelBox/Image; x,y = top-left; w,h)
| control | x,y,w,h |
|---|---|
| Main menu: Single Player | 264,324,272,35 (type any) |
| Main menu: Other Multiplayer / Battle.net | not needed |
| Char select: Create | 33,528,168,60 |
| Char select: Convert to expansion | 233,528,168,60 |
| Char select: Delete | 433,528,168,60 |
| Char select: first character info box | 37,178,200,92 (list of up to 8 visible; `getNext()` walks entries; text[1] = character name, text[?] class/level) |
| Char select: scroll click zone | 237,457,72,93 (CharSelectChar6) then scroll down |
| Char select: OK (BottomRightOk) | 627,572 (button) |
| Back/Exit (BottomLeftExit) | 33,572,128,35 |
| Create char: name textbox | 318,510,157,16 |
| Create char: expansion/hardcore/ladder checkboxes | 319,540 / 319,560 / 319,580 (15x16) |
| Popup OK centered | 351,337,96,32; popup Yes 421,337; No 281,337 |
| SP difficulty buttons | Normal 264,297,272,35; Nightmare 264,340,272,35; Hell 264,383,272,35 |
Classic ("d2classic") vs expansion create flows differ; ours: not creating characters (AGENTS rule: never create characters), so only select.

## 3. Flow: start a game with an existing character
```
state MainMenu: click "Single Player"
state CharSelect:
    press Home (jump to top of list)
    scan visible entries (<= 8 per page): entry text[1] equals wanted name (case-insensitive)
    if not found on page: click scroll area (237,457) then scroll down (mouse wheel/keys), compare first entry before/after: identical => end of list => not found
    click entry; click OK (627,572)
    wait up to 5 s while still in CharSelect (locationTimeout)
state SelectDifficultySP (single-player difficulty dialog; whether it appears on every start or only for new games is UNVERIFIED):
    click Hell button; if still at the dialog after 1 s (button disabled because the character hasn't unlocked it): click Nightmare; same fallthrough to Normal
    wait up to 5 s for the dialog to go away
state in game
```
Facts: 1.14d offline difficulty dialog lists only unlocked difficulties; clicking a locked one does nothing, so the fall-through "try highest first" (`Starter.LocationEvents.selectDifficultySP`) is the whole logic: default "Highest" = Hell first.
SP records the wanted difficulty in its data file (`setDifficulty`) and the OOG layer selects it on this dialog.
Our `goals` choose difficulty; `launch` clicks its button, verifying via in-game difficulty stat after entering (mem game info).
Timeouts (KB Starter.Config): connecting timeout and please-wait timeout (seconds; defaults in `Starter.Config`; not needed offline). Popup "OK centered" handlers: click OK + BottomLeftExit to recover from error popups.

## 4. Flow: leave game (and what comes after)
- Kolbot: host `quit()` (no UI). After leaving the game the screen returns to the char select (SP) automatically.
- External: Esc opens game menu (UI flag QuitMenu in memory; TY reads `QuitMenuOpen`), click "Save and Exit Game" (button geometry at 800x600 classic: UNVERIFIED; Tyniann's 1280x720 D2R coordinates do not transfer; calibrate live via control list if the menu is a control tree, else image anchor). TY protocol (D2R): require 3 consecutive ticks of: valid in-game, safe town area, identity confirmed, inventory/stash closed, no quit menu open; wait <= 500 ms; send one Esc; wait <= 5 s for 3 ticks of QuitMenuOpen; click Save&Exit once; confirm 3 stable ticks of `menu` phase without a readable player. Then reload character select. No retries that could double-click through menus.
- Memory facts needed: `inGame` flag and `GameInfo` (VERIFIED LIVE per PLAN), UI-open flags (mem 2.11).
- Between games (SP): KB `Starter.setNextGame` only matters online; `MinGameTime` (SP 400 s) stalls in town so game creation rate stays humane: irrelevant offline, but keep a minimum 2-3 s settle after returning to char select to avoid rapid restart bugs.

## 5. Error and recovery in OOG
| situation | KB response | ours |
|---|---|---|
| unexpected popup (OkCenteredErrorPopUp) | click OK + BottomLeftExit | click OK, then Esc |
| char select shows "no chars" / connecting | exit (BottomLeftExit); after >3 "connecting" retries call D2Bot restart | restart client (PLAN LAUNCH: crash detection) |
| character not found | log, stop profile | operator alert (goal fails) |
| stuck on loading/black screen | BT waits through loading while `is_loading`; 30 s limit then restarts D2R | same: poll area/state 30 s then kill+relaunch (only `launch` may do it; agents never kill the live game) |
| game crash | D2Bot restarts | `launch` subreaper detects child exit |
| realm down, ladder, passwords | n/a | n/a |
Clicks in OOG should block user mouse (`me.blockMouse = true` in KB). Ours: PostMessage to the game window, no host cursor use.

## 6. Delays that matter
- After clicking a character: wait up to 5 s for screen change (`locationTimeout`).
- After difficulty click: 1 s then fall through, final wait 5 s.
- After entering a game: wait until `gameReady` (player unit valid, area known); Kolbot waits `me.gameReady` and 500 ms extra before first actions; first-action sequence: heal, retrieve corpse, `Pather.init` (touch WP once to cache WP bits, 500 ms listener wait).
- Between the loops: 100 ms.

## 7. D2S header (for character list without UI; UNVERIFIED for 1.14d, standard format)
TY reads a 315-byte prefix of D2R `.d2s` files: magic 0xAA55AA55, version 105 (D2R), class byte at 0x18, name at 0x12B. For 1.14d classic files: magic same, version 0x60 (96), name at 0x14 (16 bytes), class byte at 0x28, level at 0x2B, status byte 0x24 (hardcore/expansion/ladder flags). Treat as UNVERIFIED until a live save is inspected; read-only use only (AGENTS: no save edits).

## 8. Mapping
| need | package | data |
|---|---|---|
| screen detection | `internal/launch/oog` | control list walk from FirstControl (type,x,y,w,h,text) (VERIFIED FirstControl LIVE per PLAN; the control struct layout in memory doc) |
| clicks | `internal/input` | PostMessage mouse at control center |
| character selection | `launch` | text of char list entries; optional d2s header reader |
| difficulty | `launch` + `goals` | requested difficulty; verify via `GameInfo` difficulty |
| exit | `launch` + `input` | QuitMenu flag, menu button location |
