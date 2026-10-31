# d2dctl: Go client for the d2d control channel

Package `internal/d2dctl` (client, parsers, script runner), `internal/d2dctl/fake` (test server), `cmd/d2dprobe` (CLI).
Stdlib only, our own code. Protocol facts come from `docs/d2d-spike.md` (observed, **V**) and a read of the upstream
channel notes (**D2D**, not re-verified by us). Nothing of d2d's source or text is copied. d2d is GPL-3.0 and stays a
separate process we talk to over a socket.

Status: written and tested against the fake only. Nothing here has run against a real d2d (**U**); the optional
integration test does that, see below.

## Protocol (our words)

- Transport: AF_UNIX stream socket, path chosen by whoever starts d2d (`--devctl PATH`). Owner-only, no auth. We only ever dial `unix`, never TCP.
- One command per LF-terminated line. A reply is zero or more payload lines, then a final line `ok` / `ok <rest>` or `err <message>`.
- Single client: a second connection receives `err busy` and is dropped. A line over 64 KiB drops the client with `err line too long`.
- Unknown verbs reply `err unknown verb '<name>'`. Payload is mostly tab separated; our spike transcript showed spaces, so parsers accept both.
- Rule of thumb for a payload line that itself starts with `ok` or `err`: not handled (would end the reply early). None seen (U).

## API (`import "d2lod-bot/internal/d2dctl"`)

```go
c := d2dctl.New(sock, d2dctl.Options{Timeout: 5*time.Second})   // lazy; Dial() connects at once
defer c.Close()
rep, err := c.Do(ctx, "debug warps")   // raw: Reply{Lines, Rest, Bad}; err is *CmdError for "err ..."
mons, err := c.Monsters(ctx)           // List[Monster]{Items, *Reply}; unparsable lines go to Reply.Bad
```

- Concurrency: every call holds a mutex for its whole round trip, so replies never interleave. One Client per socket.
- Timeouts: `Options.Timeout` (default 5 s) per command; a shorter context deadline wins. Use a longer context for `Difficulty` (5 to 6 s, V) and `Screenshot` on a slow box.
- Errors: `ErrBusy`, `ErrLineTooLong`, `ErrClosed` (use `errors.Is`); `*CmdError` for any other `err` reply (connection stays open). Any transport error or timeout drops the connection (the stream may be out of step) and the next call redials, unless `NoReconnect`. A command is never resent automatically (click is not idempotent).
- Busy detail: a busy server hangs up right after `err busy`, so our write can fail with EPIPE; the client then reads the pending reply and still reports `ErrBusy`.
- Typed verbs: `Ping Info State Screenshot Quit Save Click RClick Key Text Hover Wheel`, `CmdMove CmdInteract CorpseInteract CmdSkill CmdPickup CmdResurrect CmdStat CmdSkillPt CmdSelect CmdBelt`, lists `NPCs Monsters Ground Menu Items`, and the debug family `Players Level QuestShow QuestDone QuestReset Die Corpses Kill SetStat StatPts SkillPts Equip ClearInv Goto Warp Warps Enter Portals Presets Objects Difficulty Points Passives Attack Blocked Grid`. `Debug(ctx, args...)` sends any other `debug` verb. Not typed (use `Do`): `pad`, `debug roster|net|chat|typing|netsend|rain|day|light|walls|tiles|laid|relevel|tomb|tradestate|operate|itemat|wear|unid|charges|release|boss|critters|collision|automap|alt` (all D2D-only, none used by us yet).
- Tolerance: records keep `Raw`, extra trailing columns land in `Extra` (Monster, Ground), key/value replies are `KV` maps. A changed layout shows up in `Reply.Bad`, never a panic. The integration test fails on `Bad` lines so drift is noticed.
- Units: positions are float **cells**; `SubTile(cell)` = `floor(5*cell)` and `Cell(sub)` convert to our subtiles (V: 5 subtiles per cell). Screen coordinates are the 800x600 frame.
- `CmdSkill(id, x, y, SkillOpts{Unit, Left, Item})` builds `cmd skill <id> <x> <y> [unit] [left] [item]`. Unit 0 means none (U: we only saw unit ids from 1). `Item` requires `Left` (positional tail; the right-button form is unknown, U).

## Quirks, encoded

1. **`cmd move` / `cmd interact` are silently ignored after `debug goto`, a level change or a save load until one `click` was sent** (V, spike section 4; cause unknown). `Client.Kick(ctx)` sends `click 400 300`. `Goto`, `Warp`, `Enter`, `Difficulty` set a flag and `CmdMove`/`CmdInteract` kick first (disable with `Options.NoAutoKick`). After loading a save (`key Return` at char select) or any change we do not see, call `Kick` yourself. `NeedsKick()` exposes the flag. The fake reproduces the quirk (`Entry.Ignored`).
2. **`key Escape` toggles the in-game menu** (V): two presses reopen it. Check `State.GameMenuOpen()` (the `gamemenu=` field, V as `0:0` when closed; the meaning of the halves is U).
3. **`debug goto 8` (Den of Evil) renders black with no monsters** (V). Use `Warps()` then `Enter(i)`. `debug goto 17` worked (V).
4. `key shift` is `err unknown key`; there is no modifier on `click`. `CmdBelt(slot, merc=true)` is the shift-click equivalent (V).
5. `Screenshot` needs an absolute path on the d2d host (`~` is not expanded, relative goes to d2d's user dir, V). The channel can write any file the d2d user can, so keep the socket in a private dir.
6. `Points`/`debug skill` only accept the character's class skills (`err not a class skill`, V).
7. A corpse is taken with `CorpseInteract(i)` (`cmd interact -3000-i`, V); `click` on the body did not work (V).

## Probe CLI

```
d2dprobe -sock PATH ping|info|state|npcs|monsters|ground|menu        # $D2D_SOCK is the default socket
d2dprobe -sock PATH raw 'debug warps'
d2dprobe -sock PATH -json monsters                                   # {cmd, ok, err, lines, rest, bad, parsed, ms}
d2dprobe -sock PATH [-json] [-total 2m] -script probe.txt            # '-' reads the script from stdin
```

Script lines: `# comment`, `sleep 500ms`, `expect <regex>` (matched against the last reply: payload lines, then the `ok` rest, or `err <msg>` for an error), `-<verb line>` (send, an `err` reply is allowed), any other line is sent and an `err` fails the script. The run stops at the first failed step; exit status 1. JSON output lists every step with reply text and timing. Example:

```
debug warps
expect ^0\t67\.0
debug enter 0
sleep 2s
debug level
expect ^8 
-debug goto 99999
expect err
```

## Running a real d2d headless

Do not duplicate `docs/d2d-spike.md` sections 2 and 3: build flags, data copy, `--headless --devctl PATH --no-video --seed N --start-screen ingame --start-class N --start-name X`, the private `XDG_CONFIG_HOME`. This agent did not start d2d. Ready-to-run checklist for the first real check (on the worker host, nobody else's game involved):

1. Start d2d with `~/d2d-lab/start.sh` (spike wrapper), wait for `ping` to answer.
2. `D2D_SOCK=~/d2d-lab/d2d.sock go test -count=1 -run TestRealD2D -v ./internal/d2dctl` (read-only: ping, info, state, npcs, monsters, ground; fails on unparsed lines).
3. `go run ./cmd/d2dprobe -sock ~/d2d-lab/d2d.sock -json monsters` and compare `parsed` with the spike transcript, especially the ground line's trailing cell columns and the tab versus space separators (both unverified here).
4. Run a script with `debug warps / debug enter 0 / cmd move` to confirm the kick quirk handling (auto kick, then the player moves).

## Plugging into the future adapter

`docs/d2d-spike.md` section 7 seam 2: a `d2dsrc.Reader` that builds a partial `d2.Snapshot` from `State`, `Players`, `Level`, `Monsters`, `NPCs`, `Ground`, `Items`, `Grid`/`Blocked`, and an input shim mapping `input.Controller` calls onto `Click`/`Key`/`CmdMove`/`CmdSkill`/`CmdInteract`/`CmdPickup`/`CmdBelt`. Conversions: `SubTile`/`Cell`; unit ids are d2d's (`Monster.Unit`), not memory addresses. `Env.Read` needs an interface first (not in this package). The fake server plus `fake.Sample()` fixtures let that adapter be tested without a game. Gaps that no verb fills (stats dump, skill levels, missiles, structured affixes) are listed in the spike.

## Not verified

Everything against a real d2d (see checklist); upstream 64 KiB limit behaviour; whether payload lines are tab or space separated on the current upstream commit; `cmd skill` argument forms beyond `<id> x y <unit> left`; kick point (400,500) is harmless in all screens (only V in town/field after warps).
