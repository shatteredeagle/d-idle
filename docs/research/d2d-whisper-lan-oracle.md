# d2d whisper / d2proxy as a fidelity oracle

> status: research 2026-10-30, read-only; nothing run, no live game. Tags: V = read in the repo
> (raw.githubusercontent.com/psi29a/d2decomped/main/<path>), S = commit/issue metadata via gh api,
> U = unverified or my inference. Target classic 1.14d.

## 1. What they are

**whisper** (`apps/whisper/`, C++ + Python, V README.md, whisper.cpp 357 lines, whisper.py)
- A remote control INSIDE the original game.exe 1.14d: read state and drive the player over a text TCP socket. Not a game host by itself. (V README)
- Loading: game.exe imports `IJL11.DLL` ordinals 2, 3, 5; whisper is a stand-in `ijl11.dll` forwarding those to `ijl11_real.dll` and starting a socket thread. "No injector and no patched bytes; undoing it is a rename." So it is DLL proxying (side-loading), not injection. (V README "How it loads")
- Needs `whisper.txt` beside game.exe (line 1 token, line 2 port, default 4100); without it nothing listens. Build: mingw-w64 cross-compile (`build.sh`, from the Mac). (V README)
- Verbs (V whisper.cpp header): ping, state, units (every unit the client knows: type id class mode x y life maxlife), click/rclick/move/realclick, key, text, `send HEX` (raw C->S packet, refused if game.exe's own sizer would frame it differently), peek ADDR N (memory read), screenshot, party, follow, town (level id + map seed + loaded rooms diffed against d2d's `drlg-dump`), quests, ping.
- Addresses are hardcoded 1.14d VAs (player unit 0x7a6a70, send 0x52ae50, sizer 0x52bc20, unit tables 0x7a5e70/0x7a5270). Not 1.14d: refuses (`peek 463dd0 2` check). (V whisper.cpp, README "When it breaks")
- Host a game: "start the game, host a TCP/IP game, allow it through the firewall." whisper.py also has a `host(whisper, difficulty)` helper (V whisper.py:202). All commands run on the game's own thread via WM_APP. Token auth only; "keep it on your own network". (V README)

**d2proxy** (`apps/d2proxy/`, C++ `main.cpp` + CMake, V)
- A TCP man-in-the-middle between a client (game.exe or d2d) and a game.exe host: passes every byte unchanged, logs every packet both ways, decoded, in d2d's net-log format. Needs `D2_GAME_EXE` (1.14d table check). Options: `--host`, `--port` (4000), `--listen` (4000), `--log-dir`, `--level debug|trace`, `--allow` CIDRs (private/loopback only by default), `--idle`. One client at a time. (V README)
- Purpose stated: learn what a real client sends, and "see exactly where d2d and a real client differ, by recording both". (V README)

**net-join** (`tools/net-join`, V README) is the headless d2d client: joins a game.exe host, stays N seconds, writes every packet to a log. d2d proper: `d2d --join HOST`. `tools/emu/join_live.py` is its Python twin. (V)

## 2. Protocol, ports, "LAN games"

- Game transport is plain TCP to the host, **port fixed at 4000** ("No option changes it"); host binds INADDR_ANY:4000, backlog 25; this is the same packet layer as Battle.net D2GS, just no BNCS. Host sends 2 bytes `af 01` on accept. (V docs/research/re/net-join.md lines 27-46, 102, 202)
- Joining sends the client's full .d2s; host replies S->C `[03]` load act with the game's **map seed** (game+0x7c), area, etc. (V net-join.md line 310). After the game the host rewrites the joiner's save (act/difficulty/map id). (V net-join.md ~380-390)
- "LAN games" = the in-game "TCP/IP Game" menu: no Battle.net, no login, no CD-key check on the join. Confirmed by LEGAL.md "D2Decomp does not connect to Blizzard's servers" and docs/design/battlenet.md ("never target Blizzard's servers"). (V)
- Docs: docs/research/re/net-join.md (559 lines), net-packets.md, docs/design/net-host.md (d2d as host), net-join-plan.md, multiplayer.md, per-tool README "action first". (V tree)
- Legal posture: LEGAL.md relies on EU Directive 2009/24/EC Art. 1(2), 5(3), 6; no Blizzard files or code in the repo; user's own copy; side-loading a DLL into your own game.exe is local use. Not legal advice; scope "debated". (V LEGAL.md)

## 3. Maturity

- Repo created 2026-09-26, pushed 2026-10-30, 26 stars, 21 open issues, mostly platform/features (none about whisper). (S gh api)
- whisper: 11 commits from ~2026-10-05 to 2026-10-29, "drove a full Act 1 and Act 2 run with d2d" on 2026-10-18 (V README). `town` room diff added 2026-10-27. d2proxy: hardening commits 2026-10-05. No test suite for whisper in tree listing (U: not checked in CI).
- **Wine**: commit eafcb5f2 (2026-10-27) "game.exe under Wine enters games: it needs D2.LNG; run og never takes the front. Live: host Tux, d2d joins, town 35/35 rooms match." and "make wine builds patch_d2.mpq; run og starts hidden; run og host NAME". So the maintainer ran an original host under Wine, with d2d joining it. Whether whisper's DLL was loaded in that Wine run is U (commit does not say; whisper town was in the same period, so likely). (S gh api commits)
- Determinism / seed sharing: the host picks the map seed and sends it; whisper `town` prints `level L seed S` and diffs level tiles vs d2d's DRLG for that seed (V README, whisper.py:147). No mention of a way to force the host's seed or of monster-spawn determinism (U: absent). d2d keeps "one seed per game, as game.exe" (V net-host.md).

## 4. Assessment for our use

(a) **d2d client joins original host under Wine on the worker host**: likely yes. Our Game.exe already hosts TCP/IP games under Wine on the worker host with no Battle.net: VERIFIED LIVE in our docs (docs/oog.md section LAN, docs/fleet.md line 103: Host creates, TCP 4000 bound, no name/password dialog). Maintainer proved d2d-joins-Wine-host (S above). The ijl11 proxy is a plain Windows DLL search-order load; Wine uses native-in-app-dir first for non-builtin DLLs, so it should load (U; may need `WINEDLLOVERRIDES=ijl11=n,b` if Wine has a builtin ijl11, it normally does not). No ptrace needed for whisper (in-process thread), a plus over our memory reader (V design, U Wine behavior). We need a d2d binary that can join: the lab d2d fork in our container? (U: check `--join` exists in our fork vs upstream.)

(b) **Comparisons enabled** (all via whisper `units`/`state`/`peek` and d2proxy logs; V capability, U per item):
- Level layout per map seed: already done (town diff). Spawns per area: `units` lists every unit the host client knows (monsters with class/pos/life) as the original produced them; compare with d2d's spawn for the same area/seed ONLY if seeds match (see c).
- Damage/XP/drops: d2proxy logs the host's S->C packets (damage/hp updates, exp gain 0x1a-ish, item drops) for any actions a d2d client or whisper `follow` makes; diff per packet is real but needs matched scenarios. Raw state to compare: stats via `unit_stat`/peek on the host, as our memory reader already does.
- Packet-shape fidelity: d2proxy "record both" is the cleanest: same action by original client and d2d client, diff the wire. Proven for chat packet byte (V README).
- Not enabled: statistical soak comparison (XP/h etc.); one joined session is one sample on a host-chosen seed.

(c) **Determinism/replay**: weak link. The host's seed is only observable (0x03 packet, whisper town). Per-room/unit seeds are inside the host (unit seed at +0x20, V docs/sme/d2decomped-findings.md:256 quoting d2d notes). Replay (docs/replay.md) records memory pages and input; a joined d2d client as observer adds a second view of the same original session, but cannot replay the host. To get matched seeds we would need to pass the host's map seed into d2d (d2d has `--seed` for layouts, V multiplayer.md), then compare spawns for the same level (U feasibility; monster placement may also use per-room seeds that d2d only approximates).

(d) **Our rules** (questions for the user, not decided):
- Battle.net: whisper/d2proxy/join never touch it (V). Question: AGENTS.md says "Do not add anything that talks to Battle.net". Is LAN TCP to a local original host (already used by our fleet/OOG TCP/IP, V docs/fleet.md) acceptable for d2d clients too? I read it as yes; user to confirm.
- Hidden instances: host must be i1..i5 on Xvfb, never i0. Port 4000 is fixed and bound INADDR_ANY (V), so only ONE original host per network namespace; fleet.md already hosts via TCP 4000 (V) so this collides with fleet tests (U: which instance holds it). d2d client may run in a container/netns on a different IP.
- No decompiled code/game data in repo: whisper.cpp and the net notes contain decompiled VAs/function names; we must not copy the DLL source or its tables into the repo. Use upstream by reference/binary build from a fetched copy kept outside the repo (private fork location like ~/d2d-lab), same GPL/Patches policy as the d2d fork (AGENTS.md). Our docs already cite VAs (docs/sme/d2decomped-findings.md) so stating them is existing practice, but no copying code.
- Whisper in our game dir changes the original game's files (ijl11 rename). Use a separate copy of the install for oracle runs so the soak baseline binary stays pristine (U policy question).

(e) **Effort** (agent-hours, U estimates):
- First demo (d2d joins original-hosted Wine game on the worker host, log first N packets, via net-join or d2proxy; no whisper needed): 3-5 h (build upstream d2d/net-join in the lab, host from an existing hidden instance via OOG, network/netns, D2_GAME_EXE check).
- Add whisper (mingw build, ijl11 shim in a copy of the install, test under Wine, `units` + `state` from Go or python): +4-6 h.
- Spawn/XP diff harness (same area, matched seed, compare unit lists and kill XP over scripted actions, packet diff report, spec for a new fidelity metric): 15-25 h, with an open feasibility risk on seed control.

(f) **Risks / blockers**
- Seed control: host seed is random per game (U); without forcing it, spawn diffs are not matched. Possible via memory write to the host (forbidden: memory is READ-ONLY except logged cheats) or choose-and-replicate from the observed seed in d2d.
- Our d2d is a private lockstep fork (d2d.lab); upstream `--join` and net-host maturity vs fork divergence (U).
- Player-count scaling: host+joiner = 2 players, so monster HP/XP use the /players 2 tables (V net-host.md section 2: tables [0,0,50,100,...] indexed by player count, so 2 players = +50% HP and +50% exp); fidelity numbers for 1-player soaks are NOT comparable unless d2d has the same scaling (net-host lists it as a still-to-do for d2d, V). Big caveat for XP/damage diffs.
- Joiner character is written back and its act/difficulty changed (V net-join.md): use throwaway chars.
- Whisper address table is 1.14d-only, fine for us; token-only auth, bind to loopback/netns.
- Wine: unproven whisper-under-Wine in our setup; D2.LNG needed (V commit message).

## 5. Experiment card sketch (do not file)

`d2d-oracle-join`: owns docs/d2d-oracle.md (+ tools/oracle/ if code later); exclusive: `instance:i4` (host, hidden), `d2d-lab-binary`; gates: hidden instance only, throwaway chars, no Battle.net, nothing posted upstream.
1. Build upstream tools/net-join and d2proxy (or the lab fork) on the worker host; D2_GAME_EXE from the install.
2. Host on i4 via OOG (TCP/IP Host); `net-join` from a netns or same host, 10 s; log packets.
3. Put d2proxy between them, repeat, check log equals net-join log.
4. Decide go/no-go on whisper from the result; then 5-seed matched-spawn experiment.
Gate: first N=200 packets logged with 0 unknown names for the join sequence.

## 6. Open questions

User: (1) is LAN TCP to a local original host allowed under the Battle.net rule? (2) OK to run oracle runs on a separate game install copy with the ijl11 shim? (3) is the 1-player vs 2-player monster scaling accepted as a known gap?
Maintainer (user to ask): does whisper work under Wine in your `run og` flow, and can the host's map seed be chosen (command line, save header 0xab map id)? Is spawn/XP parity checked per packet anywhere?
