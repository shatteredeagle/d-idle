# D.idle

_Say it: die dull. Status page._

A hobby project that turns Diablo II: Lord of Destruction into an idle game. A bot plays the real game by itself, offline, and a layer of goals, routines and progression is being built around it, the way an idle game gives you things to unlock and decisions to check in on.  
It is early days. Right now the work is on the part underneath: a bot that plays reliably for hours with nobody watching.  

_Streams when the bot is running._

## Between runs

No bot is running right now. The last run of each:

- **Loopy**, Sorceress, d2d: level 4, 7,700 XP, 334 kills, 0 deaths, 13 potions drunk, run time 0h 05m. Now: playing through the early game in Rogue Encampment, ended. Last update 2026-10-31 00:10 UTC.
- **Soaky**, Sorceress, d2lod (original game): level 15, 195,140 XP, 7,100 kills, 4 deaths, 203 potions drunk, run time 8h 05m. Now: levelling up toward level 22 in Stony Field, ended. Last update 2026-10-30 20:26 UTC.
- **Fresha**, Sorceress, d2lod (original game): level 11, 75,549 XP, 2,387 kills, 3 deaths, 121 potions drunk, run time 4h 04m. Now: levelling up toward level 20 in Stony Field, ended. Last update 2026-10-30 20:12 UTC.

## Now building on d2d

- d2d is an open-source reimplementation of the Diablo II engine. It lets the bot run headless (no screen) and far faster than real time. Measured with the bot attached: 22 to 33 times faster.
- On 2026-10-30 we decided to make d2d the main way the bot runs. The original-game path is frozen: it keeps working, and it stays as the reference we compare against.
- Progress so far: a fresh Sorceress reaches level 5 in about 3 to 4 hours of game time, which takes minutes of real time. The bot's loop on d2d is still being tuned.
- Goal, not built yet: the bot and d2d packaged in one container, so a friend can start it with a single command.

d2d is the work of the d2decomped project (GPL-3.0). Our fixes to it go upstream as open pull requests. [d2decomped](https://github.com/psi29a/d2decomped), pull requests [#127](https://github.com/psi29a/d2decomped/pull/127) and [#128](https://github.com/psi29a/d2decomped/pull/128).

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
<summary><b>Alpha: a reliable Act 1 bot on the product path</b>: ████░░░░░░ 47%</summary>

| Milestone | Progress | Checks verified |
| --- | --- | --- |
| M0 Foundations: the product path runs live | ██████░░░░ 64% | 2 of 7 |
| A Act 1 self-leveler: clvl 1 to 15, 2 h unattended | █████░░░░░ 50% | 2 of 11 |
| B Normal Act 1 complete: quests, Andariel, hop to Act 2 | ████░░░░░░ 40% | 1 of 11 |
| G1 Day-long soak on the product path (24 h) | ███░░░░░░░ 37% | 1 of 8 |

</details>

<details>
<summary><b>Beta: Normal complete, installable by a friend</b>: ███░░░░░░░ 30%</summary>

| Milestone | Progress | Checks verified |
| --- | --- | --- |
| C1 Normal Acts 2 and 3 | ░░░░░░░░░░ 8% | 0 of 6 |
| C2 Normal Acts 4 and 5, Baal, hop to Nightmare | ░░░░░░░░░░ 0% | 0 of 5 |
| F Headless container and setup wizard, usable by a friend | ████░░░░░░ 41% | 1 of 6 |
| G2 Multi-day soak (72 h) | ░░░░░░░░░░ 0% | 0 of 4 |

</details>

<details>
<summary><b>Good-bot gate: the game layer may start</b>: ██░░░░░░░░ 26%</summary>

| Milestone | Progress | Checks verified |
| --- | --- | --- |
| D1 Nightmare playthrough | ░░░░░░░░░░ 0% | 0 of 4 |
| D2 Hell playthrough and farming runs | █░░░░░░░░░ 12% | 0 of 4 |
| H Good-bot gate | ████░░░░░░ 40% | 2 of 10 |

</details>

## Right now

Between runs.

**Being worked on:**

- Long unattended test runs (working)
- Working on the bot (working)
- Playing in a party of bots (working)
- Running the bot on the d2d engine (blocked)

## What shipped

Nothing merged in this window.
---

Last updated: 2026-10-31 07:45 UTC

_Numbers are conservative._  
_A check only counts as done once it has been verified for real: a live run in the game, or automated tests when that is what the check asks for._  
_Half credit for work that is only partly proven._  
_Anything untried counts for nothing._  

_D.idle is an unofficial fan project. It is not affiliated with or endorsed by Blizzard Entertainment. Diablo is a trademark of its owner._
