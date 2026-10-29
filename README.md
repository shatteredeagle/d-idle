# D.idle

_Say it: die dull. Status page._

A hobby project that turns Diablo II: Lord of Destruction into an idle game. A bot plays the real game by itself, offline, and a layer of goals, routines and progression is being built around it, the way an idle game gives you things to unlock and decisions to check in on.  
It is early days. Right now the work is on the part underneath: a bot that plays reliably for hours with nobody watching.  

**[LIVE now on Kick](https://kick.com/brognak)**

## Why we are making it

- We like Diablo II and we like idle games. This is what happens when you put them together: the game plays itself, and the fun is in the setup, the progress and watching it go.
- A bot that can be left alone is a satisfying engineering problem. It has to cope with getting stuck, with danger and with dying, without a person stepping in.
- The plan is the bot first and the game second. Once the bot is dependable, the idle-game layer goes on top: things to unlock, runs that reset for a bonus, rewards, and parties of bots.
- If it turns out the bot can be flexible, the game around it can be flexible too. If not, a very good bot that the game layer deliberately limits is still a fine foundation.

## What is different about it

- A game around the bot. Most Diablo II bots are tools for farming. Here the bot is the engine and the game layer (goals, routines, progression, a dashboard) is part of the design.
- Built to be left alone. Several layers of safety catch a stuck bot, a dead character or a crashed game, and a separate watchdog process steps in when the bot cannot help itself.
- Offline and on the original game. It never connects to Blizzard's online service, never changes the game's files, and nothing from Blizzard is included. You use your own copy.
- Quiet on your computer. It reads the game from the outside and sends keys and clicks only to the game window, so you can keep using your machine. Many copies of the game can run at once.
- As far as we have found, nobody has layered a game like this over an unmodified classic PC game. The closest projects, such as bots for mobile games and the World of Warcraft playerbot servers, share parts of the idea.

## How it works

- It looks at what is happening in the game (health, position, items), the way you look at the screen.
- It presses keys and moves the mouse like a person would. The bot never edits the game while it plays.
- Our test tools can edit throwaway characters to set up a situation, and every one of those edits is logged.
- A separate watchdog notices when it is stuck or in danger and steps in.
- The game layer decides what the bot should be working toward, and the bot works out how.

## Where we are

Each stage is a gate made of milestones. Open a stage to see its milestones.

<details open>
<summary><b>Alpha: a reliable Act 1 bot on the product path</b>: ██░░░░░░░░ 28%</summary>

| Milestone | Progress | Checks verified |
| --- | --- | --- |
| M0 Foundations: the product path runs live | █████░░░░░ 50% | 1 of 7 |
| A Act 1 self-leveler: clvl 1 to 15, 2 h unattended | ███░░░░░░░ 33% | 1 of 9 |
| B Normal Act 1 complete: quests, Andariel, hop to Act 2 | █░░░░░░░░░ 12% | 0 of 8 |
| G1 Day-long soak on the product path (24 h) | ██░░░░░░░░ 20% | 0 of 5 |

</details>

<details>
<summary><b>Beta: Normal complete, installable by a friend</b>: █░░░░░░░░░ 19%</summary>

| Milestone | Progress | Checks verified |
| --- | --- | --- |
| C1 Normal Acts 2 and 3 | ░░░░░░░░░░ 0% | 0 of 6 |
| C2 Normal Acts 4 and 5, Baal, hop to Nightmare | ░░░░░░░░░░ 0% | 0 of 5 |
| F Headless container and setup wizard, usable by a friend | ████░░░░░░ 41% | 1 of 6 |
| G2 Multi-day soak (72 h) | ░░░░░░░░░░ 0% | 0 of 4 |

</details>

<details>
<summary><b>Good-bot gate: the game layer may start</b>: █░░░░░░░░░ 17%</summary>

| Milestone | Progress | Checks verified |
| --- | --- | --- |
| D1 Nightmare playthrough | ░░░░░░░░░░ 0% | 0 of 4 |
| D2 Hell playthrough and farming runs | █░░░░░░░░░ 12% | 0 of 4 |
| H Good-bot gate | ██░░░░░░░░ 25% | 1 of 6 |

</details>

## Right now

The bot is paused right now, so there are no live numbers.

**Being worked on:**

- Working on the bot (working)
- Finding its way around the map (working)
- Building this public status page (working)
- Long unattended test runs (working)

## What shipped

<details open>
<summary><b>2026-10-28</b> (63)</summary>

- Tool warm-up and fd plateau, recycler policy hooks.
- Resource policy, task layer and quests connected into the bot; live 31 min campaign.
- Scene tile, grid3, keep-alive, StreamDesk and HA toggles.
- Fd study, game recycler, oog fd tools.
- TV-style stream scene page and generator.
- Shared stream selection, auto scheduler rule, StreamDesk and HA sync.
- Stream overlay, grid layouts, status --json.
- Managed mode live, death/IsDead fixes, long-run test findings.

<details>
<summary>55 more</summary>

- Kick OAuth login and set-title library.
- Home Assistant package and dashboard for the stream.
- Stream control service, scheduler, max-live guard.
- LAN host/join over loopback, party driver, menu TCP/IP screens.
- Live-now link on the public page, sanitizer extension.
- Kick interface research for stream title.
- Desktop tray app for the stream.
- Task layer.
- Pure mana/health resource policy package.
- Quest state reader, NPC dialog primitive, act hops, Act 1 handlers.
- Percent layout, grouped shipped feed, plain work titles, dark-mode check.
- This status page, with a safety check that keeps private details out.
- The roadmap: what alpha, beta and good-bot mean, and how far along each is.
- Checked in the real game that quests, waypoints, acts and difficulty are saved correctly.
- Research on how other projects add goals and progression on top of a game.
- Fixed a crash in the part that presses keys; it now recovers by itself.
- Management for several bots at once, and goals now show up live.
- The bot now runs on the shared server and exposes its settings.
- Run several bots at once and play together as a party.
- The dashboard can show several running bots.
- Counters and a rig for long unattended test runs.
- The bot can start inside a container with no screen, a step toward easy installs.
- A live viewer to watch the bot's game from another device.
- The dashboard and watchdog now handle several bots.
- One web server that serves every running game.
- Record a game session and replay it later for testing.
- More game knowledge: mercenaries, item properties, effects, projectiles and runewords.
- A logged tool to set up test characters at any quest, waypoint or act.
- The bot spends stat and skill points, equips gear and plans its level-up runs.
- Learned what happens on death, skips enemies it cannot reach, and retreats when health is low.
- Automatic checks run on every change now.
- Lets the bot see the game's state without special permissions.
- Buys potions and scrolls, opens portals, and finds its way around town in any game.
- A catalogue of goals the bot can be given, and a way to track them.
- Plans for each character class and ideas for playing as a party.
- Navigation that works in any randomly generated map, finding exits and waypoints.
- A separate watchdog that steps in when the bot is stuck or in danger.
- The first real runs: the bot fights, shops and recovers from trouble.
- A live viewer for watching several bots at once.
- A logged tool to set up test characters quickly.
- Several games can now run side by side, each on its own hidden screen.
- Town chores: shopping, restocking and picking up loot.
- Fighting: picking targets, using skills and noticing danger and death.
- The main bot program with a policy for what to do after dying.
- Finding paths, exploring maps and using waypoints.
- A live status board and first screens for following the bot.
- Research on whether several bots can play together.
- Research on how other Diablo II bots work.
- Reliable key and mouse control that reconnects after hiccups.
- The bot can read the game's state: health, position, items and open screens.
- A launcher that starts the game and its helpers reliably.
- The bot reads the map around it and tracks where it has been.
- A goal manager with a built-in dashboard and a simulator.
- Reads the game's own data files to learn items, monsters and maps.
- Strategy for levelling up quickly.

</details>

</details>

---

Last updated: 2026-10-29 00:34 UTC

_Numbers are conservative._  
_A check only counts as done once it has been verified for real: a live run in the game, or automated tests when that is what the check asks for._  
_Half credit for work that is only partly proven._  
_Anything untried counts for nothing._  

_D.idle is an unofficial fan project. It is not affiliated with or endorsed by Blizzard Entertainment. Diablo is a trademark of its owner._
