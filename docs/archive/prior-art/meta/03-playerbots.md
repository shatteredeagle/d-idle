# 3. AzerothCore Playerbots + Individual Progression (+ LLM bot-buddy)

- https://github.com/mod-playerbots/mod-playerbots (main fork liyunfan1223/mod-playerbots): GPL-2.0, ~1.2k stars, pushed 2026-10-24 (V, `gh api`; README fetched).
- https://github.com/ZhengPeiRu21/mod-individual-progression: MIT, ~290 stars, pushed 2026-10-20 (V).
- https://github.com/DustinHendrickson/mod-ollama-bot-buddy: AGPL-3.0, 45 stars, "NOT FOR GENERAL USE YET" (V README).
- Lineage: IKE3 mangosbot (V README says "based off IKE3's Playerbots").
- Games: World of Warcraft (WotLK client) on private servers. Not unmodified single-player: you run your own server and patch it.

## What it is (V, README)
- Log in alt characters as bots; form parties; "random bots that wander, quest and otherwise behave like players, simulating the MMO"; bots can run most raids and battlegrounds; "excellent performance even with thousands of bots" (V claim, from README).
- Requires a **custom AzerothCore fork** (`Playerbot` branch), the stock server will not work (V). Supported on Ubuntu, Windows, macOS. => injected-in-server architecture, the strongest of the options, and the most fragile against core updates (forum note: the fork is synced to upstream roughly weekly, S).
- AI engine (S, third-party docs https://wishmaster117.github.io/playerbots-docs/ai-architecture): a context registers strategies, triggers, actions and values; a strategy bundles triggers + default actions + multipliers per state (combat, non-combat, raid); triggers queue actions; engine runs the next action. It is a behavior-tree-ish trigger/action system.
- Player controls bots by chat commands ("follow", "attack", strategy toggles) = the "user sets direction" layer, in-band.
- Individual Progression module (V README): per-player tier state ("simulate progress through expansions"), NPCs/objects aware of each player's progress, content gated by tier, no catch-up mechanics; "works well together with Playerbots". Stored in player settings.
- Ollama bot-buddy (V README): LLM issues commands (`move to`, `attack <guid>`, `spell`, `loot`, `follow`) to playerbots to pursue "high-level AI goals" (level, quests, professions, raids); warns of heavy CPU load and instability.

## Lessons for us
- **Party/fleet at scale is a solved problem only when you own the server.** Thousands of bots is possible because bots are server-side objects, not game clients. Our D2 reality is one client per bot (about one core each on the worker host); multi-bot parties need a different plan (shared game on one host, or separate clients on LAN TCP/IP games, offline Battle.net-less). UNVERIFIED: D2 open TCP/IP games with multiple local clients.
- **Progression tiers as a data object per character** (Individual Progression) is a ready model for "difficulty, act, hardcore/prestige gating"; MIT, so the design is freely reusable (V license).
- **Strategy/trigger/action vocabulary** is a good public language for our goal manager's behavior profiles (Kolbot has the same in script form).
- **LLM-on-top of rule-based bots** (bot-buddy) confirms the layering: LLM picks goals and commands, a deterministic engine executes. Immature (45 stars, caution banner).
- Failure modes (S, forum/Discord chatter, not fetched): server-core drift, bot population affecting economy/auction house, performance tuning knobs. Not verified.

## Reuse
- Code: GPL-2.0, not reusable in our Go tree. Design: yes. Not ban-relevant (private servers), but private-server WoW lives in a legal gray area (U).
