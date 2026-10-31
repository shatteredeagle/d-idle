# Multiplayer: many bots, party, LAN, private realm (agent NET)

Evidence tags: V = fetched page/code read this session, S = search snippet only, R = recollection / unverified, L = VERIFIED LIVE.
Sections 1-8 were written without running the game (proposed tests in section 7). Section 9 (agent PARTYLIVE, 2026-10-28) holds the live results of T1/T3/T4/T5/T6/T7 on the worker host; where it contradicts an R/S line above, section 9 wins.

## Summary and recommendation

- Possible: yes. Multiple 1.14d clients on one machine, joined in one TCP/IP (LAN) game, is established practice in the Kolbot/D2Bot world (V: 8 profiles on one PC reported; Kolbot has TCP/IP host/join profile types). Our architecture (read memory per pid, post window messages per window) scales to it with modest changes, provided we remove "one game" assumptions now.
- Order of attack: (A) 2 local instances in a loopback TCP/IP game, (B) 4-8, (C) friends reach the worker host over Tailscale and join a TCP/IP game hosted by a bot (this already gives "our own world" for friends, with no server software), (D) humans and bots mixing with chat commands. A PvPGN private realm is a LATER, optional step, only if persistent shared characters on a server are wanted.
- Do NOT use Blizzard's Open Battle.net with bots. Do NOT run a realm before TCP/IP over Tailscale works.
- Biggest risks: (1) not knowing how the 1.14d client behaves with N windows in one prefix (single-instance check, window pausing when unfocused or occluded); (2) D2GS for 1.14d is not documented as stable (pvpgn.pro recommends 1.13c) and our memory offsets are 1.14d only; (3) UI-driven party invite and chat need verification (no external API).
- Cost to keep the door open: small if "instance" is a first-class parameter in wave 2 (see `design-now.md`), large (a cross-package refactor of globals, the helper protocol, and paths) if retrofitted.

## 1. Multiple local game instances

What is known:
- Kolbot/D2Bot runs many 1.14d clients per machine: one forum poster ran 8 characters on one PC with 8 profiles and 8 key sets, noting a considerable slowdown with 2 (S: blizzhackers.discourse.group "Multi Botting - How do I go about this?"). Kolbot docs: each instance is a profile, identified by the window title (V: bhdocs.github.io/kolbot/MultiBotting). Two characters of the same account cannot be at the game-select screen together on Battle.net (S); this is a realm issue, not a LAN one.
- The D2R "DiabloII Check For Other Instances" named event is D2R only (S); do not assume anything about 1.14d from it. For 1.14d, a D2Bot note says the game is a unified exe and D2BS absorbed multi-instance features (S); no single-instance mutex was identified. R: classic 1.14d has no single-instance check when launched twice (D2Loader's `-multi` was for older versions). PROPOSED TEST T1 settles this in 1 minute.

Design decision: ONE Wine prefix, ONE install, N Game.exe processes (R, justified below), distinct characters.
- Prefix cost: a prefix is mostly the wineserver plus services.exe/explorer/winedevice; sharing it saves roughly 100-200 MB and 3-5 helper processes per instance versus a prefix each (R, estimate; measure in T2). One install is also 2 GB less disk and one patch level.
- Shared state in one prefix: registry (`HKCU\...\Diablo II`, V launch.md section 5) is shared; this holds video/sound settings and `NewSavePath`, not per-character data. Per-character data is `<name>.d2s/.key/.map` in the save folder (V launch.md). Two instances must use different characters: R, the game keeps its own `.d2s` open/locked while in game, so the same character twice is blocked or corrupts (never do it; enforce in the orchestrator: one live instance per character name).
- Reasons to fall back to one prefix each: the registry fights us (e.g. window position/video flags that differ per instance), or the game's global named objects collide inside one wineserver (T1/T3 will show). The Instance abstraction below hides this: `PrefixDir` is per-instance config, usually equal for all.
- TCP/IP port clash: the host binds TCP 4000 (S: guides list 4000 TCP for hosting). Only ONE instance per machine can host; others join `127.0.0.1`. R: a client joining its own machine over loopback works (Kolbot host/join profiles on one PC do exactly this; test T4).
- Cost per instance (measured/estimated): CPU: the idle live game gains ~11 ticks/s, about 11% of one core (V launch.md); active play with 800x600 software DirectDraw is maybe 20-40% of a core (R). 8 instances fit easily in 16 cores even with other load. RAM: D2 1.14d is small (R: 150-300 MB working set + Wine) so ~2 GB for 8. GPU: DirectDraw to Xwayland, negligible on an RX 7900 XTX; do not use D2DX for the extra instances (launch.md section 6). The Wine overhead is the wineserver round trips, not the GPU.
- Windows: each is a 800x600 top-level Xwayland window. Since input is PostMessage (V), focus is not needed, so windows can sit anywhere, even overlapping. Unknown: whether the game stops rendering/updating when occluded or minimized (R: D2 keeps ticking; rendering may pause), which would not matter for memory reads. Display options: (1) tile on the KDE desktop (fine for <=4; the user sees them); (2) one Xvfb or `Xwayland -rootless`-less nested display per instance via `DISPLAY=:N` (R: works because Wine processes connect to X per process; shared wineserver is fine), which keeps the user's desktop clean and lets the whole fleet run on the headless the head box box or any box later; (3) Wine virtual desktop (`explorer /desktop=d2-1,800x600`) per instance: one window per virtual desktop, trivially addressable, no X server needed beyond one. Recommend trying (3) first for B, (2) if (3) breaks input.
- Kolbot's way: one OS process per bot (D2Bot manager launches a client per profile with its own window title and profile name). Our equivalent is `Instance` (below), supervised by one Go process.

## 2. What breaks in our design with N instances

| Component | Single-game assumption | Change |
|---|---|---|
| `cmd/d2input` | `FindWindowW(0,"Diablo II")` (V main.go:47-49) returns one arbitrary window; listens on a fixed `127.0.0.1:47012` | Every command takes a target; helper enumerates windows with `EnumWindows`, filters by class/title "Diablo II" and records `GetWindowThreadProcessId`; one helper per prefix serves all windows. A `list` command returns `{hwnd, winpid, w, h}` |
| Linux pid vs Windows pid | -- | GOTCHA: `GetWindowThreadProcessId` returns the WINDOWS pid, not the Linux pid in `/proc`. Do not try to match them numerically. Bind by launch order (spawn one game at a time; exactly one new Game.exe in /proc and one new hwnd in `list`), then PROVE the binding: post a `mousemove` to hwnd X at a distinctive point and read `MouseX/MouseY` (V RVA 0x3A6AB0/0x3A6AAC) from that pid; match = bound. Re-run the handshake on helper reconnect |
| `launch` | one prefix/game, one helper addr | `Instance` owns its Game, a shared `Helper` per prefix; `FindImage` already returns all pids; launch serially with a stagger |
| `mem` | one `Source` | already a value; one `Proc` per instance (pid + base). The app is subreaper for all children (V: set once, before the first Wine process) |
| `d2`, `d2map` | take a `Source`; fine as is | no globals/caches keyed by nothing. A global cache (e.g. the static level data) is fine if read-only; per-game caches live in the instance |
| `input` | `Controller` dials one addr | `Controller` carries a window handle; helper addressing is `hwnd=...` per command |
| `bot` | one character, one `Tuning` | `Tuning` and `Policy` per instance; add `Role` |
| `goals` | one `Runner` | `Manager` per bot, plus `Party` orchestrator above (section below) |
| `api`/`web` | one character panel | `/api/instances`, per-instance SSE, party view |
| config/paths | single save dir, single character | per-instance character name, profile dir (goals store, logs, snapshots) |

Concrete `Instance` abstraction (sketch, Go):

```go
type InstanceID string // "i1", stable across restarts; the key for logs, store, API

type Instance struct {
    ID      InstanceID
    Char    string          // character name (unique among live instances)
    Prefix  launch.Config   // usually shared; per-instance override
    Proc    *launch.Game    // pid, base, MemReader (mem.Source)
    Window  input.Window    // hwnd, size, helper addr
    In      input.Controller
    Bot     goals.Runner    // bot.Runner behind it
    Tuning  bot.Tuning
    Role    party.Role      // leader, killer, teleporter, mule, follower, solo
}
type Fleet interface { // orchestrator-facing, goal manager sees only this
    Instances() []InstanceID
    Get(InstanceID) *Instance
    Start(spec InstanceSpec) (InstanceID, error)   // launch + bind + OOG to ready
    Stop(InstanceID) error
    Events() <-chan FleetEvent  // up/down/crash/joined-game/left-game
}
```
Package-by-package interface changes are in `design-now.md`.

## 3. Game modes

### (a) TCP/IP (LAN) games
- Flow (S, Hamachi guide): main menu, Other Multiplayer, TCP/IP Game, then Host Game or Join Game, pick a character; the joiner types the host's IP. Kolbot models exactly this: profile types `TcpIpHost`/`TcpIpJoin` and OOG locations `TcpIp`, `TcpIpEnterIp`, `TcpIpUnableToConnect` (V: kolbot libs/oog/Locations.js:42-66, libs/OOG.js:1080-1121, modules/sdk.js:5075,5139-5143). Our OOG table must add these screens (Verified=false until the checklist is run, T5).
- Port: TCP 4000 on the host, inbound (S). 6112 is Battle.net only (S).
- Characters: R, TCP/IP games use the same single-player (local, `.d2s` on each client's disk) characters; no realm account. Each player's character is saved on their own machine. Open (Battle.net "open") characters are a separate class on the old open realm and are not what we use. R: expansion vs classic characters cannot mix in one game; use LoD chars.
- Player limit: R, 8 players, same as Battle.net; difficulty is chosen by the host in the create-game dialog; each character must have unlocked that difficulty (R). Monster scaling follows the in-game player count automatically (R, `/players` is chat-only for display).
- Loopback: bot instances on the same machine join `127.0.0.1` (R, T4). Humans on other machines join the worker host's address (Tailscale address or LAN name, see below).
- The game's existence: the game lives only while the host client is in it. If the host leaves, the game ends for all. So the host should be the most stable client (a mule/leader bot that never exits), and the others cycle in and out. A human should not host unless the bots can survive it ending; rejoining is cheap for bots (OOG Join).
- Security note: a TCP/IP game has no authentication and single-player characters are trivially editable (R). Acceptable among friends; it matters for any "economy" idea.

### (b) Blizzard Open Battle.net
Classic Open Battle.net is, as far as search shows, still reachable (S, forum threads about "disconnected from Battle.net", no 2026 status confirmed), but botting there violates Blizzard's terms and is actively policed: classic D2 has had mass account and CD key bans, connection restrictions for third-party tools, modified clients and even VPN/proxy/business IPs, and temp bans for rapid relogging (S: Blizzard forum threads, Guru3D/Shacknews ban-wave stories). Our key is on the line and the user's key may not be replaceable. Recommend against, now and later; AGENTS.md already forbids it, and the OOG flow must keep that hard rule (no Battle.net screens in the table, no realm connect code paths enabled by default).

### (c) Private realm: PvPGN + D2GS + D2DBS
- PvPGN: the original project stopped in 2011 (V: github.com/pvpgn/pvpgn-server README says "development stopped in 2011" in the PRO fork's text); the active fork is PvPGN-PRO (GPL-2.0, CMake, needs C++11/GCC 5.1+, runs on Linux, optional Lua and MySQL). Its README lists Diablo II and LoD 1.10 through 1.14d (V). D2GS is NOT part of PvPGN (V).
- D2GS: the game-server piece is a separate project, patches a copy of Blizzard's own server binaries built from the user's game files via "Magic Builder" scripts (V: pvpgn.pro D2GS guide). That guide recommends 1.13c as "latest working stable" with other versions buildable via Magic Builder (V); it does NOT say 1.14d is supported/stable. This is the central risk: our offsets are 1.14d. D2GS wants Windows; Wine is possible but "may hit socket problems" (V), so host D2GS in a Windows VM on Proxmox (PvPGN/D2CS/D2DBS can stay on Linux, example "D2GS on Windows + PvPGN on Linux", V).
- Components and ports (V: pvpgn.pro guide): PvPGN/bnetd 6112, D2CS 6113, D2DBS 6114, D2GS game 4000, D2GS telnet admin 8888. PvPGN and D2GS must be reachable by clients; D2CS and D2DBS can be internal. Set a non-default `D2CSSecrect`/`d2gs_password`; set `EnableWarden=0` (V). For NAT: forward 6112 and 4000 plus `address_translation.conf` (V). With Tailscale none of the forwarding is needed.
- Pointing a client at it: the `BNETIP` registry value under `HKCU\Software\Blizzard Entertainment\Diablo II` selects the Battle.net gateway host (S: Slash Diablo wiki, which also says it predates 1.14), or the "Battle.net Gateway Installer" tool which edits gateway settings (V: pvpgn.pro). In our prefix, write `BNETIP` with `reg.exe` (launch `RegAddArgs`, UNVERIFIED). Server and client versions must match; version check errors "Unable to identify version" are a known failure (S, low quality).
- What players need: their own legal copy of D2 + LoD patched to 1.14d. CD key: R, PvPGN does not validate CD keys against Blizzard (key checks are off/optional by config), so players can use their own key; confirm in PvPGN config docs. Never put keys in the repo (AGENTS.md).
- Hosting: PvPGN on a small LXC (Linux), D2GS on a Windows VM (2 cores / 2 GB is ample for friends), D2DBS beside PvPGN, ports reachable only over Tailscale. Back up the character store. Effort: 2-5 days for someone who has not done it, mostly the D2GS build and 1.14d compatibility (R); plus a new OOG module for realm login/char select/create-game on a realm (the single-player and TCP/IP flows do not cover it).
- Hardening: Tailscale-only exposure, strong D2GS/D2CS secrets, Warden disabled but only trusted friends in the tailnet, no public IP, updated PvPGN-PRO, unprivileged service accounts, offline character backups. R: PvPGN has had old remote bugs; never expose it to the internet.
- Legal/ToS posture (plain): running a private realm emulator for friends with their own licensed copies and no distribution of Blizzard files is low practical risk, but it is not clearly permitted: the EULA bars emulators, and R: Blizzard won its 2002 bnetd case on that basis. Keep it private, non-commercial, no shared game files. The user accepts that residual risk or does not do phase C-realm.
- Recommendation: skip the realm unless persistent shared characters/ladders are wanted. TCP/IP over Tailscale delivers "friends play with the bots" with zero server software.

### Reaching a LAN game safely
Use Tailscale (or a WireGuard tunnel): install on the worker host and each friend's PC, share the worker host's node with the friends (node sharing or a tailnet invite), they join `100.x.y.z` (or the MagicDNS name) on port 4000. Reasons over port forwarding: no inbound port opened to the internet, D2's TCP/IP protocol has no auth or encryption (S/R), and it avoids NAT/firewall tuning. ACL: allow friends only to tcp:4000 on the worker host. Port forwarding is acceptable only with an IP allowlist on UniFi; not recommended. Hamachi-style overlays work too (S) but Tailscale is already in the lab (R, user runs a homelab; check). Latency: D2 is lag-tolerant; WAN jitter is fine.

## 4. Bot-to-bot and bot-to-human interaction

Party mechanics:
- Roster: the `RosterUnit` list at RVA 0x3BB5C0 (docs/d2-1.14d-memory.md 2.5; struct V D2Structs.h:216-235, 0x84 bytes, `pNext @0x80`; has name, unit id, class, level, level id @0x24, x/y @0x28/0x2C). Kolbot reads `partyid` and `partyflag` per roster entry (V: modules/workers/SimpleParty.js:26-102). Constants (V: SimpleParty.js:7-14, sdk.js:49-62): NoParty = 65535 (partyid), flags InParty=1, Accept(able)=2, Invited=4; panel buttons: Hostile=1, Invite/Accept/Cancel=2, Leave=3, Ignore=4, Squelch=5. The offsets of `partyid`/`partyflag` in our RosterUnit are not in our docs: PROPOSED TEST T6 (diff roster memory before/after invite; D2BS D2Structs.h names them `wPartyId` and `dwPartyFlags`, R).
- Invite/accept: D2BS calls `clickParty()` in-process. Externally the equivalent is the party panel (UI index 0x16, hotkey `P`, V docs 2.11): open it, click the player's invite/accept button, close it. Kolbot also does it only after reading flags, with cooldowns (V SimpleParty). So: hotkey `P` (key press works, V), click the per-row button (positions UNVERIFIED, needs a calibration pass T7), confirm via roster flags. Alternative without UI: none known (the engine sends packet 0x5E, V sdk.js:5366, but we do not inject packets; stay UI-only).
- Follow a leader: read the leader's roster x/y and level id (V: roster has them), walk/teleport to a point near it; follower logic is a `nav` goal "go to unit X". Kolbot Follower/Leader scripts rely on the same position info plus chat commands (V: scripts/Follower.js:450-466, ControlBot).
- Shared loot: D2 has no loot rules; items drop per player instance visibility. Convention (our design): a `LootPolicy` per role (leader picks only runes/charms; followers pick nothing) decided by the orchestrator and enforced by each bot's loot filter.
- Mule pattern: V Kolbot AutoMule (libs/systems/automule/Mule.js:259-318: `item.drop()` and a tracking set `droppedGids`). Flow: mule joins the same game, parks in town; bot drops items on the ground near the mule (cursor item drop, UI clicks via inventory layout, which our input already needs for stash); the mule picks up by clicking the ground item. Trading (trade panel 0x17, V docs 2.11) needs both sides' accept clicks and is more UI work; drop and pickup is simpler and enough.
- Chat: sending: Enter, text, Enter via window messages; text needs `WM_CHAR` or an equivalent (open item in launch.md OOG checklist 5: "key typing may need a `char` command"). Receiving: there is no memory API in our docs for the chat buffer; Kolbot uses a hook event `chatmsg` and `gameevent` (V: Follower.js:848-849, Party.js game event modes 0x02 join and 0x06 slain). PROPOSED T8: find the chat/event text buffer by diffing memory while a known string is sent (R: D2 keeps rendered chat lines in the D2Client message list); fallback is a screenshot OCR of the chat area, which is slow but possible. IMPORTANT: bot-to-bot coordination must NOT use game chat; use the orchestrator's in-process channel (same machine) and treat chat as a human-facing channel only (commands in, status out).
- TP/waypoint coordination: Town portals are objects readable in memory (unit table); a leader casting TP then followers clicking it is the Kolbot pattern. Waypoints are per-character; a party can use another player's TP to enter an area without having its waypoint. A "teleporter + killer" role is: teleporter (Sorc, Teleport) walks the level fast to the target and lays a TP; killer follows through the TP and fights, with a 2-way "ready/arrived" handshake through the orchestrator (not chat).
- Role-based farming: roles are data in the orchestrator (`Role`, per-role `Tuning`/`Policy`, per-run plan). Kolbot analogues: MFLeader/MFHelper, BaalAssistant, Rusher/Rushee (V scripts listing).
- Anti-grief and safety with humans present: bots must not be hostile by default (hostile = panel button 1; keep it off and read flags to detect others turning hostile; Kolbot's AntiHostile thread reads hostile flags and reacts, V threads/AntiHostile.js:21-56). Policy: a bot that sees a hostile player quits the game (leaves via OOG) or goes to town and `Stop`s; no PvP, no looting other players' drops unless the owner is a bot in the party (R: D2 protects nothing here, so only pick up items owned by roster members within the same party). `Shitlist` (V oog/ShitList.js) is a ready pattern. Never auto-accept party invites from non-whitelisted names; a humans-allowed list is config. A human kill-switch: a chat command `!stop` from a whitelisted name and a dashboard "pause party" button.

## 5. Server-side bots

D2GS exposes no bot or scripting API (R: it is a binary-patched Blizzard server driven by D2CS/D2DBS packets; V pvpgn.pro describes only config registry values and a telnet admin port 8888 for operations). A server-side bot would mean implementing the client protocol (a headless D2 client in Go) which is a full reverse-engineering project and, on a realm, protocol-level cheating territory. Our client-memory approach scales to this use: it is protocol-agnostic (works on single-player, TCP/IP, and realm), needs nothing from the server, and the extra cost per bot is one game process, which is cheap (section 1). Keep it.

## 6. Staged roadmap

| Phase | Goal | Prerequisites | Verify first | Exit criteria | Effort / risk |
|---|---|---|---|---|---|
| A | 2 local instances in a loopback LAN party | wave 2 with Instance/Fleet abstraction; helper takes a target; OOG TCP/IP screens in table; party panel calibration | T1 (second launch OK), T3 (two windows in one prefix get distinct hwnds), T4 (join over loopback), T5 (TCP/IP menu coordinates) | Two Instances reach the same game, party up (roster flags InParty), one follows the other, both survive 30 min and are driven independently from the dashboard | 1-2 weeks; medium risk (T1, UI calibration) |
| B | 4 to 8 bots | A; staggered launcher; per-instance logs and Tuning; resource limits | T2 (CPU/RAM per instance, does it scale on a loaded worker host), window display option (3) vs (2) | 8 bots up for 4 h, a role-based run (teleporter+killer) works, mule drop and pickup works | 1-2 weeks; low-medium risk (resource, deadlock in orchestrator) |
| C | Friends reach the bot game | B; Tailscale on the worker host and friends' PCs, ACL tcp:4000 only | T9: a friend on another machine joins the worker host's hosted TCP/IP game; version match 1.14d for all | A human joins, is partied by a bot, bots do not misbehave; kill-switch works. (Optional C2: PvPGN + D2GS realm, only if shared persistent characters are wanted) | C: 2-3 days; C2 realm: 1-2 weeks plus the D2GS 1.14d risk and a realm OOG module |
| D | Bots mixing with humans | C; chat read/send works (T8), anti-hostile policy, whitelist | T6 roster flags, T8 chat buffer | Chat-commanded actions (`!tp`, `!follow`, `!stop`) from whitelisted humans; a hostile human triggers retreat; 1-week soak | 1-2 weeks; medium risk (chat reading, UI robustness) |

Do A with only 2 instances before investing in B. Stop at C if friends are happy.

## 7. Proposed tests (NOT run; do when the game can be restarted, all through tools/live.sh and the lock)

- T1: start a second `Game.exe -w` with another character in the same prefix; does it start, is there a single-instance block? Check hwnd count via helper `list` (needs helper change).
- T2: with 2 then 4 instances idle in town: per-process RSS, ticks/s, wineserver CPU.
- T3: two windows: distinct hwnds, EnumWindows pid listing, the mouse-handshake binding (post mousemove to hwnd A, read MouseX/Y from pid A, not pid B).
- T4: host a TCP/IP game in instance 1 (OOG), join `127.0.0.1` from instance 2; is port 4000 bound; what happens when the host leaves.
- T5: dump TCP/IP screens' control lists (type, x, y, w, h) and fill the OOG table (Verified flags).
- T6: roster memory diff before/after party invite to find partyid/partyflag offsets.
- T7: calibrate party panel button rows and the "hostile" checkbox.
- T8: find the chat/event text buffer; test sending text with `WM_CHAR`.
- T9: WAN join over Tailscale from a second machine.
- T10: occluded/minimized window: does the game keep running and responding to PostMessage.

## 8. Sources
V: kolbot clone (Party.js, SimpleParty.js, sdk.js, Locations.js, OOG.js, Mule.js, Follower.js, AntiHostile.js); https://bhdocs.github.io/kolbot/MultiBotting; https://github.com/pvpgn/pvpgn-server (fetched page text is PvPGN-PRO fork); https://pvpgn.pro/d2gs_installation.html.
S: https://blizzhackers.discourse.group/t/multi-botting-how-do-i-go-about-this/1257; Blizzard classic-D2 forum threads on connection restrictions and VPN blocks; https://segmentnext.com/diablo-2-hamachi-guide; Slash Diablo wiki setup (BNETIP); https://github.com/dschu012/pvpgn-d2gs-docker (existence only).

## 9. Live results: two offline 1.14d clients in one LAN game over loopback (agent PARTYLIVE, 2026-10-28)

All L (VERIFIED LIVE on the worker host, i1 Navi = host, i2 Combi = joiner, one Wine prefix each, GE-Proton11-11, 1.14d, both on private Xvfb
displays), method and commands in brackets. Open Battle.net was never opened (its button is not in the OOG table).

Answer to the key risk: YES. Two clients in two Wine prefixes join one TCP/IP game over 127.0.0.1 and both characters are in it.

### Menu path and coordinates (docs/oog.md has the table; control dumps matched against `tools/shot.sh` screenshots)
| screen | controls (type,x,y,w,h bottom-anchored) | click |
|---|---|---|
| Main menu -> "Other Multiplayer" | (6,264,433,272,35) | (400,416) |
| Other Multiplayer: [Open Battle.net], TCP/IP Game, Cancel | (6,264,310..), (6,264,350,272,35), (6,264,568,272,35) | TCP/IP (400,332); Cancel (400,550). Open Battle.net is the top button: never clicked, not in the table |
| TCP/IP Options: label "Your IP Address is:" + the machine's address as a label (265,171) | Host Game (6,265,206,272,35), Join Game (6,265,264,272,35), Cancel (6,39,571,128,35) | Host (401,189), Join (401,247), Cancel (103,553) |
| Join: modal "Enter Host IP Address to Join Game" over TCP/IP Options (the Options controls stay in the list) | edit (1,300,268,213,20), Cancel (6,281,337,96,32), OK (6,421,337,96,32) | box (406,258), OK (469,321), Cancel (329,321) |
| Character select (same controls as single player, so the screen alone does not say it is LAN) | as docs/oog.md | OK (691,555) |
| Failed connect: popup-ok, text "Cannot Connect to ..." | popup-ok button | (399,321) |

Findings that differ from the proposal above:
- There is NO name, password or difficulty dialog for hosting: Host Game -> character select -> OK puts the character straight into the game (a Normal-only character; the difficulty screen is still unseen). `party.Game.Name/Password` therefore only identify the generation in the Directory; nothing is typed into the game.
- The host's own client connects to its own server over the machine's LAN address (ss: Game.exe 192.0.2.226:x <-> 192.0.2.226:4000).
- Port 4000: bound 0.0.0.0:4000 by the HOST prefix's wineserver while the host is in the game; gone when it leaves. The joiner (other prefix) binds nothing: its connection is 127.0.0.1:ephemeral -> 127.0.0.1:4000 (`ss -tanp` on the worker host). Only one host per machine at a time (a second host would need port 4000 too, UNVERIFIED whether it errors or picks another).
- Save and Exit from a LAN game returns to TCP/IP Options, not to the main menu. `Flow.LeaveGame` accepts both.
- The join IP box KEEPS its previous text and appends: typing "127.0.0.1" twice gave "127.0.0.1127.0.0.1" and "Cannot Connect to". The buffer is longer than the 15 visible characters, so the flow now sends backspaces (each key press costs seconds of helper round trips; 100 took about 90 s) before typing. Typing dots with WM_CHAR works.
- Join from the clicked OK to in-game took about 5 s; one join (of about 8) sat on the Loading screen for more than 2 minutes (host fine) and then completed. Cause unknown (U); `FlowDriver` bounds Create/Join at 3 minutes.
- A TCP connect-and-close probe of port 4000 from outside (bash /dev/tcp) did not stop the host: a later join to the same host worked.

### Both players in the same game (T3/T4/T6) [`oog players` on each instance, `cmd/oog/roster.go`]
- Each client's unit hash table (RVA 0x3A5E70, type 0 = player) lists BOTH characters on host and joiner (3 entries: Navi, Combi and a dead Navi corpse unit, mode 17), so the other player's unit is readable by the existing `d2.Units(TypePlayer)` on either side. The "client" table at 0x3A5270 read empty for players.
- Roster (RVA 0x3BB5C0, struct as D2BS, offsets L): name char[16] @0, unit id @0x10, class @0x1C, level WORD @0x20, partyid WORD @0x22 (65535 = none), level id @0x24, partyflag @0x30, next @0x80. Host's list: Navi id 1, Combi id 3; joiner's list: Combi, Navi (order differs). x/y @0x28/0x2C read 0 in town.
- Joining does NOT party the players: both read partyid 65535, partyflag 0.
### Party invite (T6/T7)
Hotkey `p` opens the party panel (screenshot: a row per other player with an INVITE button at about (297,139), level and class). Clicking INVITE on the host: host roster flag of the other = 4 (invited), the joiner's roster flag of the host = 2 (may accept); joiner screen text "Navi invites you to ally against the forces of evil." The joiner's panel shows ACCEPT at the same spot (297,139); after accepting both rosters read partyid 3 and partyflag 1 (in party) for the other player. This matches kolbot's flags (1 in party, 2 acceptable, 4 invited) L. The panel row spacing for more than one other player is UNVERIFIED.
### Host leaves, member rejoins (T4)
- Host Save and Exit: the joiner's client dropped to TCP/IP Options within ~5 s on its own (no popup seen); port 4000 closed. Characters are saved on each side.
- Rejoin: from TCP/IP Options the joiner joins the next game with the same flow (needs the new host to be in the game first). Full orchestrated run (`oog "party i1:Navi:leader i2:Combi:killer" pstatus pplayers "plost i2" prejoin... "pleave i1"`, job log in the worker host `~/d2lod-live/partylive/runs/20261028-190252-oog.log`): Start put host i1 and joiner i2 in one game (both `in`); `MemberLost i2` then `Rejoin i2` brought Combi back into the running host game (new unit id 4 instead of 3, roster of both updated, partyflag back to 0: parties are not restored); `Leave i1` (host) made the Party hand off, with two members it disbanded and both ended on TCP/IP Options. A host swap (i2 hosting, i1 joining over 127.0.0.1) was also done by hand twice and worked.
### Not observed (U)
XP/gold/loot sharing and monster scaling with 2 players (no monsters fought; both characters stood in the Rogue Encampment), Players X, the difficulty screen for a character with Nightmare unlocked, a failed join to a non-hosting address (the junk-IP failure popup is the only error seen), more than 2 players.
