# Input: helper, client, transform (agent INPUT)

## API summary
- `cmd/d2input` (Windows, in the prefix, `127.0.0.1:47012`). One line in, one reply out: `ok [data]` or `err CODE msg`
  (codes: usage, badbutton, badkey, nowindow, unknown). Commands: `ping`, `info`, `move X Y`, `down|up l|r X Y`,
  `click l|r X Y [hold=MS]`, `key down|up|press NAME [hold=MS]`, `char X` (WM_CHAR), `hung` (`ok alive|hung`),
  `sleep MS`, `seq A ; B ; C` (one round trip, reply `ok N` or `err CODE step i (cmd): msg`), `quit`.
  Modifier flags on mouse commands: `shift`/`ctrl` (WM_KEYDOWN around the message + MK_ bit), `mkshift`/`mkctrl` (wParam bit only,
  for a key held with `key down`). The window is looked up on every command, so a restarted game is found with no helper restart
  (a missing window gives `err nowindow`).
- `internal/input`: `Client` (reconnecting TCP, per-call deadline = expected work + 2 s, `Seq` batching, `Ping/Info/Alive/TypeChar`;
  errors `ErrNoWindow`, `ErrHelperDown`, `ErrTimeout`, `*CmdError`), `Timing` (the one place for press/settle/gap jitter; 40 ms press floor),
  `Transform` (`Calibrated`), `Reader` (player, mouse, viewport, UI, hover, skills, area over `mem.Source`), `Controller` interface
  with `Game` (live) and `Fake` (tests), `OOG` adapter (`Click(x,y)`, `Key(name)` for internal/launch).
- `Controller`: MoveMouse, Click(btn,x,y,mods...), Key, Hold/Release, Pos, ClickWorld, MoveTo(w, {Tolerance, Timeout}), SelectSkill,
  CastAt(hotkey, w), Interact(Target), CloseAll.
- Measurement tool: `tools/live.sh input inputcal <state|sample N|near|hovertest CLASS|selftest|ui|key NAME|dump RVA LEN|goto X Y TOL|peek ..>`
  (`cmd/inputcal` is a symlink to `internal/input/cmd/inputcal` because live.sh only builds `cmd/<pkg>`). `tools/shot.sh OUT.png`
  grabs the game window (read-only X capture).

## Transform (VERIFIED LIVE, 87 clicks over 3 runs)
- Viewport: for a standing player, `ViewportX = 16(px-py) - 400` and `ViewportY = 8(px+py) - 284` exactly (max error 0.0 over 30 rows),
  where (px,py) = Path xPos/yPos + xOffset/65536 (rests at integer+0.5). Hence the screen position of any world point is
  `sx = 16(wx-wy) - VX`, `sy = 8(wx+wy) - VY`, with the tile centre at wx+0.5. `Transform.WorldToScreenVP` uses this and is what
  `Game` uses (right even when the camera lags a walking player). MEM's "(400,276)" is the same fact for an integer (not +0.5) position: 276+8=284.
- Clicks: click offset (dx,dy) from (400,300) versus Path xTarget/yTarget (integer subtiles, floor of the clicked point):
  regression gives kx = 16.03, ky = 7.96 (model 16, 8), origin offset (+0.2, -16.8) from (400,300), i.e. player feet at (400, ~283.2); the fixed model
  with origin (400,284) has mean error well under 1 subtile (the residual ~0.6 subtile is the floor() rounding of the target).
  The earlier assumption of (400,300) was 16 px (2 subtiles) off vertically: that was the "~2 subtile" error.
- Final position always equals target + 0.5 (click-to-move arrives on the tile centre).
- Clamp rules (`Transform.InView/Clamp`): world clicks accepted at x=400 for y up to 550 and ignored at y>=560 (VERIFIED LIVE, 7 probes).
  Between 520 and 550 at x=400 sits the mini-panel button row (y 530-552), so `PanelTop` stays a conservative 520. Left/right extent of the
  bottom bar not measured (UNVERIFIED); horizontal margin 4 px.
- Dead end: an earlier panel probe that walked back with a blind click at (400,150) drifted the character 100 subtiles; measurement runs now return with MoveTo.

## Hover and UI (VERIFIED LIVE)
- UI-open array: `GetUiVar` (RVA 0x538D0) is `mov eax,[ecx*4+0x7A27C0]`, so UI flags are DWORD[0x26] at RVA 0x3A27C0. Seen: [0]=1, [0x15]=1,
  [0x23]=1 always; [8]=1 NPC menu (Kashya Talk/Cancel), [9]=1 ESC menu, [1]=1 inventory (with [0xA]). One Esc closes the ESC/options menu or the NPC menu.
  `Reader.UIOpen` ignores 0, 0x15, 0x23.
- Hover: `GetSelectedUnit` (RVA 0x67A10) reads id at RVA 0x3A6A78, type at 0x3A6A8C, valid flag 0x3A6A94. Posted WM_MOUSEMOVE over Kashya
  (feet to 50 px above) gave type 1 id 9; the corner over the waypoint gave type 2 id 11; empty ground 0/0. Posted moves drive it without focus.
  NPC hit area spans from the feet up ~50 px; `Interact` probes feet-10, -20, -30.
- `Interact(Kashya)` then `CloseAll` worked live (menu opened, Esc closed it, UIOpen false).

## Other results
- Key press (`i` opens inventory, Esc closes) VERIFIED LIVE. `MoveTo` arrival (tolerance 2, ~1.3 s for 8 subtiles) VERIFIED LIVE.
- Shift: shift-flag click and held-shift click both still walked in town; the game gives no visible difference there. Message delivery
  works, effect (stand-still attack) UNVERIFIED (needs a monster outside town).
- `SelectSkill`/`CastAt`: F1/F2 are unassigned on the test character (right skill stayed Fire Bolt 36), so the hotkey-to-skill path is UNVERIFIED;
  `Reader.Skills` (pInfo -> pLeft/RightSkill -> wSkillId) reads left 0, right 36 and is VERIFIED LIVE for reading.
- `char X` and `hung` helper commands are unit-tested only (UNVERIFIED live).
- Gotchas: `tools/helper.sh` used to leave the helper holding the game flock (fd inherited), blocking every `live.sh`; fixed with `flock -o`.
  The game was found with the ESC/options menu open at the start of the session (not by this agent's clicks); closed with Esc.

## Helper reliability (agent INFRA, VERIFIED LIVE on i5 2026-10-28)
- ROOT CAUSE of the repeated helper deaths: `winSys.list()` called `syscall.NewCallback` on every window enumeration (every
  untargeted command, every `valid`, every `@hwnd` check). Go keeps NewCallback slots in a fixed table of 2000 that is never freed;
  the 2001st call is `runtime.throw("too many callback functions")`: a fatal error, not a panic, so recover() cannot stop it and a GUI-subsystem exe
  prints it nowhere. At bot rates (about 10 commands/s) that is a death every 5-7 minutes per instance. Evidence: soak on i5 died
  at 16:43:59 after 6:47 and about 2300 commands; the crash log's stack is `syscall.NewCallback -> compileCallback -> throw`
  from `winSys.list` (`runtime/syscall_windows.go:333`). Fix: one global callback (`enumCB`), results through a mutex-guarded slice.
  After the fix the same soak ran 20 minutes with zero deaths (numbers in the CHANGELOG/report).
- Exit/crash log: `<prefix>/pfx/drive_c/d2input-PORT.log` (`tools/helper.sh log iN`): start line, heartbeat every minute (goroutines, heap,
  commands, window count), `EXIT code=.. reason` for quit/signals/listen failure, `PANIC` with stack for handler panics (recovered, reply
  `err internal`), and the Go runtime's own fatal-error output (`debug.SetCrashOutput`). A log that ends mid-heartbeat with no EXIT line means
  the process was SIGKILLed or its wineserver was killed from outside.
- Other death sources found by reading (not the soak cause, all fixed): `tools/helper.sh restart|stop` used `pgrep -x d2input.exe` which matches
  EVERY instance's helper (one per prefix); `launch.Managed.Find` without an instance had the same blind spot; `game.sh helper` waited for the
  game lock held by a running bot; `Spawn` truncated the helper's stdout log.
- Self-heal: `input.Client` dials 3 times (150 ms, 500 ms backoff); when all fail it runs `d2game helper iN` (idempotent, no game lock,
  replaces a helper process that does not listen) at most once per 15 s and retries the call once (nothing was sent, so no double click).
  Only works on the worker host (needs `~/d2lod-live/d2game/d2game` or `$D2GAME_BIN`); i0 has none (`tools/helper.sh restart`). Set `Client.Restart` to override.
