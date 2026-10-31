# Meta-layer prior art, any game (PRIORART2, 2026-10-28)

Question: does anything like "run an unmodified single-player game, put a goal/schedule/progression meta layer and dashboard over it, and let a bot play unattended, with live view, parties and a fleet" already exist?
Tags: V = fetched and read by me (URL given), S = search snippet or secondary, U = memory/unverified. Licence/stars/dates come from `gh api` on 2026-10-28 (V).
Scope: complements `../index.md` and `../lessons.md` (D2 bots); none of the D2 bots are repeated here. Offline only, so anti-ban notes are brief.

## Ranked 10 closest analogues
| # | project (game) | quadrant | licence | what it teaches | reuse |
|---|---|---|---|---|---|
| 1 | [Alas](01-alas.md) (Azur Lane) | vision on emulator | GPL-3.0 | next_run task scheduler, wait policy, failure budget then human take-over, error bundle, process-per-config fleet + web GUI | design only |
| 2 | [MAA](02-maa.md) (Arknights) | vision on emulator | AGPL-3.0 | task menu = goal menu, JSON schemas as public protocol, CLI/HTTP/bindings, roguelike farm loop, drop stats | design only |
| 3 | [AzerothCore Playerbots + Individual Progression](03-playerbots.md) (WoW) | injected in server | GPL-2.0 / MIT | parties and thousands of bots, strategy/trigger/action vocabulary, per-character progression tiers, LLM on top | tiers design (MIT); rest design |
| 4 | [AutoExile](04-autoexile.md) (Path of Exile) | memory read via ExileApi | none stated | ARPG modes with supply preconditions, phone dashboard, max-deaths as goal config | inspiration only |
| 5 | [Claude Plays Pokemon / Voyager](06-agents-pokemon-voyager.md) | RAM overlay + LLM | Voyager MIT | curriculum = goal manager, skill library, summaries, knowledge base, loop failure modes | ideas; optional planner |
| 6 | [Screeps](05-screeps.md) | game built for scripts | ISC (server) | 24/7 policy-as-code, CPU budget as the scarce resource, mods | design only |
| 7 | ArchiSteamFarm (Steam cards, many accounts) | Steam protocol client | Apache-2.0 | one process manages many accounts: per-account config, IPC, plugin system (V: `gh api`, wiki home) | fleet design; code reusable in spirit |
| 8 | Progress Quest / Godville (zero-player RPGs) | the game is the layer | PQ source released 2011 (licence U) | spectator-first UX, pure idle loop (S: wikipedia.org/wiki/Progress_Quest) | UX tone |
| 9 | Angband Borg (APWBorg, Angband 4.2.5) | in-process, in source tree | Angband licence (U) | full auto player of a real roguelike; `borg.txt` config; `-b` autostart; `cheat_live` option to auto-restart after death; removed pre-4.0 then revived in 4.2.5 (S: angband.readthedocs.io/en/latest/hacking/borg.html) | lesson: maintenance cost kills borgs |
| 10 | Sanderling + Viir/bots (EVE Online) | external memory read | permissive (U) | memory-reading framework separated from bots; "no injection/writes" posture; legality unclear (S: github.com/ranc1/Sanderling, EVE forum thread) | confirms our read-only architecture |

Also checked, ranked lower: Mineflayer (MIT, 7.7k stars) and Baritone (LGPL-3.0, 9.3k) as headless-client plus pathfinder (V metadata); mineflayer fleet managers are small experiments (S). OSRS: DreamBot/OSBot/TRiBot ship script managers and multi-bot per machine, tiered paid, ban-driven economics; Jagex bans tens of thousands per week (S: oldschool.runescape.wiki "Bots, Bans and Appeals"). BALROG, NLE, Factorio Learning Environment, SIMA 2 are research agents (see file 06). Twitch Plays Pokemon is the crowd-direction curiosity (S). Not researched in depth: Dwarf Fortress/RimWorld, Elite Dangerous, Black Desert, Roblox/gacha wrappers, Melvor Idle, Cookie Clicker (no external-layer relevance found).

## Patterns that matter (architecture)
1. **Four control quadrants.** Vision on emulator (Alas, MAA), memory read + synthesized input (AutoExile, Sanderling, ours), injected into server/process (Playerbots, Angband Borg), headless protocol client (Mineflayer, ASF). Memory-read is the sweet spot for a frozen offline client: precise state without injection risk, and our D2 1.14d never updates, so the usual breakage (game patch) is absent.
2. **Scheduler over scripts.** Every long-lived tool ended up with "task + next_run + failure count" (Alas) or a mode with a stop condition (AutoExile). Goals need: precondition, stop condition, budget (time, deaths, fails), escalation.
3. **Core + thin frontends.** MAA and Alas: one engine, GUI/CLI/HTTP/bindings. Matches our `internal/goals` + `internal/api`.
4. **Fleet = process per config + one UI.** Alas `ProcessManager.get_manager(config_name)`, ASF multi-account. Resource accounting matters (our instance is about one core).
5. **Data not code for strategies.** MAA JSON schemas, Playerbots strategies, Individual Progression tiers, Voyager skill library. Users and the community extend via data.
6. **Human escalation is a first-class state** (Alas RequestHumanTakeover + notification, error screenshots).
7. **LLM as planner, not pilot.** Claude Plays Pokemon took ~140 h for 3 badges and looped in Mt. Moon; BALROG finds weak long-horizon play. Playerbots' bot-buddy does the sane thing: LLM issues goals to a deterministic engine.
8. **Anti-ban** is the reason the analogues hide (proxies, randomization). Irrelevant offline; do not import their evasion code or culture.

## Verdict (1 page)
**Is it novel?** The ingredients all exist; the exact combination does not, as far as I could find (searches V/S, not exhaustive: no hits for a "meta layer with prestige/rebirth and cheats-as-rewards over an unmodified single-player game").
- Goal queue + schedule + dashboard + unattended bot + fleet: exists in Alas and MAA (mobile gacha games, vision-driven). They are the proven product shape, with large communities (24k and 9.8k stars).
- Parties of bots + per-character progression + LLM direction: exists on WoW private servers (Playerbots, Individual Progression, ollama-bot-buddy), but via a modified server.
- ARPG farming modes + phone dashboard: AutoExile (tiny, online PoE).
- Zero-player game as entertainment: Progress Quest/Godville, but not wrapping a real game.
- **Not found anywhere:** (a) an invented progression layer (rebirth/prestige, hardcore as a resettable season, cheats granted as rewards) sitting on top of a real game's own loop; (b) doing it for a classic offline ARPG on Linux with an external memory-read bot; (c) a live view + parties + fleet in one open tool. MAA's roguelike auto-farm and Angband's `cheat_live` restart are the nearest hints of (a).

**Nearest thing:** Alas for product shape (scheduler + GUI + 24/7 + multi-instance), Playerbots + Individual Progression for the meta/progression idea, AutoExile for the ARPG instance of it. A Playerbots-style "idle MMO" on an unmodified single-player game is what we propose; nobody ships it.

**What to steal (design, not code; Alas/MAA/Playerbots are copyleft, we are not):**
1. Alas-style loop: sort goals by `next_run`, wait policy (park in town / quit game / keep running) as config, per-goal failure budget (3 strikes) then human-required state with notification.
2. Error bundle per failure: timestamped folder with log, last N snapshots/screenshots, memory dump of unit state.
3. One process per instance config, one dashboard, a fleet view with running/waiting/failed.
4. MAA's rule: publish goal, route and strategy JSON schemas and keep CLI/HTTP parity with the GUI; add a headless mode.
5. AutoExile-style goal parameters: stop condition, supply precondition, `max_deaths`, live session stats on phone.
6. Individual Progression's per-character tier record for our difficulty/act gating and prestige state; keep it MIT-style simple data.
7. Claude-plays-Pokemon pieces for an optional planner: an editable "current objectives" text, periodic summary, state overlay from memory. Keep it optional.
8. Record anonymous run statistics (MAA drop uploads) only if the user opts in; useful for tuning.

**Failure modes to plan for (from the analogues):**
- Stuck states and loops (Pokemon agent 78 h in Mt. Moon; Alas stuck/too-many-clicks errors): need watchdogs and the failure budget above.
- UI/game patches break vision bots (Alas, MAA: constant maintenance). We avoid it by pinning 1.14d; keep the offset table fingerprinted.
- Server-core drift for injected designs (Playerbots weekly fork syncs, S).
- Maintainer burnout and removal (Angband Borg dropped pre-4.0, revived at 4.2.5, S).
- Community reaction/legal: botting in online games is banned (Jagex, S); our offline single-player scope sidesteps it. Cheats-as-rewards stay on throwaway offline characters, logged, per AGENTS.md. Do not ship Blizzard files (PLAN.md already says so).
- Copyleft: Alas GPL-3, MAA AGPL-3, Playerbots GPL-2, bot-buddy AGPL. Reading is fine; copying code is not, unless we relicense.

## Open questions
- Does a multi-client D2 1.14d party work offline over local TCP/IP with several Wine instances? (U; test before promising parties.)
- No evidence gathered on Dwarf Fortress/RimWorld automation, Elite Dangerous, or gacha idle wrappers beyond the two vision projects; low expected value.
- Alas multi-instance details and MAA task-queue/multi-instance internals not read in depth (V only for process_manager structure and README).
