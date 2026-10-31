# Milestones: alpha, beta, the good-bot gate, and the ladder (agent MILESTONES, rescored by MILESTONES2 2026-10-28)

The overall goal is unchanged (PLAN.md): an idle game on top of classic D2 LoD 1.14d, offline single player. The order is the
user's: **first a good, reliable bot, then the game on top of it** (upgrades and progression UI, hardcore as prestige/rebirth,
cheats as rewards, parties). This folder defines what "alpha", "beta" and "good bot" mean for the bot, and the benchmark ladder
used to track progress. It replaces the interim A-H ladder on the hub and refines docs/review/roadmap.md M0-M7 (which stays as
the source of the numeric thresholds).

| file | content |
|---|---|
| [milestones.json](../../milestones/milestones.json) | machine-readable ladder: milestones, criteria, how each is measured, status and evidence, rollup rules. Shown at `/milestones` on the status server |
| [A.md](A.md), [B.md](B.md), [C.md](C.md), [G.md](G.md), [F.md](F.md), [E.md](E.md), [S.md](S.md) | per-milestone detail where useful: work packages still needed, live checks, risks, effort |
| [risks.md](risks.md) | risks and the "what would make us slip" list |
| [next-3-weeks.md](next-3-weeks.md) | recommended sequence of work packages with owners and parallelism |

Tags as elsewhere: **V** verified (live or by a command, says how), **S** secondary, **U** unverified or a guess. All effort
numbers are agent-days and **U**.

## Status rules (also in milestones.json "rollup")

- A criterion is **done** only if verified live (a doc names the run, instance and date) or by CI. Offline tests alone make a
  criterion **partial**, except for criteria that are offline by nature (one: knob metadata). A live run in progress with no
  doc yet is partial.
- Milestone percent = (done + 0.5 x partial) / criteria. Milestone status: done if all done, todo if all todo, else partial.
- Gates are cumulative (beta includes alpha, good-bot includes beta); gate percent is the mean of its milestones.
- A status moves to done only in the commit that adds the evidence note. Whoever verifies a criterion updates milestones.json
  (owner of this folder: MILESTONES, or the main agent).

Honest snapshot today (master abf1195, rescored by MILESTONES2): **alpha 47 %, beta 30 %, good bot 26 %** (was 28 / 19 / 17 at
50b98d2). What moved most: the product path is proven live (managed mode, no setcap, 90+ minute leveling run, docs/launch.md 9)
and the task layer, quest reader, nav and class-agnostic combat exist and have run live in parts. Almost all of the gain is
`partial` (offline code plus a short live run). Only these are `done` beyond the old ones: managed mode, a 60+ minute
unattended run, the task layer (offline by nature), the soak judge, class-agnostic combat (Paladin, live numbers) and the
loopback LAN join. Still missing for alpha: no 2 h or 24 h soak report, the longest campaign run is 31 minutes at clvl 6
(one death, 13 ladder rungs from one nav cause that is fixed), boss scripts and the in-memory quest source are not wired
into the bot and the boss scripts are unverified live, no act hop, no CI runner, saves not wired into start/stop. A new
milestone **S** (stream, scene, public status) sits in the layer phase and gates nothing (56 %, mostly mock-tested tooling).

Criteria added in this pass: A.run90, A.seeds, B.nav, B.questmem_wired, B.boss_live, G1.judge, G1.recycler, G1.rungs
(recovery rungs per hour), H.class_combat, H.class_leveler, H.class_merc, H.class_infra, and the whole S milestone.

## Alpha (bot side)

**One sentence:** a fresh level-1 character, started through the product path, completes Normal Act 1 and reaches Act 2
without help, and the same setup runs 24 h unattended.

Must be true (milestones M0, A, B, G1):
1. **Product path, not the dev path.** One `d2bot --managed` per host is the game's subreaper and memory reader; no setcap, no
   `--attach`, no bash/ssh in the restart path; the sentinel runs in that topology (in process or via memfd).
2. **Levels itself.** clvl 1 to 12 in <= 2 h and to about 15 in Act 1, on two characters with different seeds.
3. **Completes Act 1.** Den of Evil, Blood Raven, Countess, Andariel through the task layer; quest state read and verified;
   Warriv hop to Act 2.
4. **Dies well.** <= 1 death per hour, no death with an unused potion and life above the chicken line; death, respawn,
   corpse policy and back to the goal without help.
5. **Pays its way.** Town loop on earned gold (potions, scrolls, repair, sell, stash), gear and points spent in the loop.
6. **Runs a day.** 24 h soak passes the m3 profile with 0 human or agent interventions, with recovery rungs per hour reported and falling (G1.rungs); a crash or hang loses at most one save
   cycle; saves snapshotted and verified.
7. **Visible.** The dashboard shows state, deaths, failsafe actions and why a bot paused.
8. **CI green** on every push.

Explicitly out of scope for alpha: Acts 2-5, Nightmare/Hell, the mercenary, the container and wizard (the worker host desktop/Xvfb host
is fine), parties, a second class, handicap profiles in play, any game-layer UI, hardcore, phone notifications, teleport.

## Beta

**One sentence:** Normal complete and installable by a friend.

Must be true (alpha plus C1, C2, F, G2): fresh clvl 1 to Nightmare entry unattended within 40 h wall time (merc, cube, all
required quests, act hops, Baal, difficulty screen); a container image plus web setup wizard that a person other than the user
and the agents installs with their own game files; a 72 h soak with 0 manual restarts, log rotation and a disk guard; a
phone notification when the bot needs a human; upgrades of the binary keep state.

Out of scope for beta: Nightmare/Hell playthroughs, farming rates, parties, the second class, game layer.

## Good-bot gate (H): when the game layer may start

**One sentence:** a bot good enough that handicapping it is a game, not a mercy.

Must be true (beta plus D1, D2, H):
1. Nightmare and Hell playthroughs to Baal unattended (deaths <= 2 per hour in Nightmare).
2. Farming goals (Countess, Andariel, Mephisto, Pindleskin, Baal) at runs per hour within 2x of a scripted human baseline,
   with drop logs and a loot filter that keeps what matters.
3. 7-day soak across mixed goals with at most one human intervention.
4. **Handicap profiles proven live**: each built-in profile is measurably slower and still within the death bound. This is the
   raw material of the upgrade game (PLAN "optimal first, handicap by configuration").
5. **Telemetry the game can price**: XP/h, seconds per run, deaths/h, gold/h, drops per run, from measurements.
6. **A second class** (Paladin, docs/classes/ranking.md) passes milestone A, so rebirth can offer a different class.
7. An Opus review of the gate finds no blocker.

After the gate, in this order: L1 upgrades and progression UI, then L2 hardcore as prestige and L3 cheats as rewards (both on
L1's currency), and parties (E) as a feature. Out of scope forever: Battle.net or any realm, human-bot mixed online play.

## The ladder

Changes against the interim hub ladder, with reasons:
- **M0 added** (foundations): every live number so far was taken on the `--attach` + setcap dev path (findings F1); alpha must
  be measured on the shipped path, with CI and save protection.
- **A kept**, thresholds made measurable (clvl 12 in 2 h from roadmap M1; 15 as the Act 1 ceiling).
- **B now ends with the Act 2 hop**, so the whole progression machinery (task layer, dialog, boss script, act travel) is proven
  once on the cheapest act.
- **G split and moved up**: G1 (24 h) gates alpha, G2 (72 h) gates beta, and a 7-day soak sits in H. Reliability is the first
  thing the user asked for, and every failure so far came from running longer.
- **C split** into C1 (Acts 2-3: merc, cube, Duriel, Mephisto) and C2 (Acts 4-5, Baal, Nightmare entry): different mechanisms
  and about 30 agent-days in total, too big for one rung.
- **D split** into D1 Nightmare and D2 Hell + farming.
- **F stays in beta**, in parallel with C (different owners); it fixes topology and file layout but does not make the bot play
  better, so it does not gate alpha.
- **E moved after the gate** as a feature (the user lists parties with the game layer), but its risk is retired early: the
  loopback TCP/IP join spike is in week 2 of the plan.
- **H kept** as the gate, with criteria; **L1-L3** added so the game layer is visible on the board, all todo.

| id | milestone | phase | % now | rationale |
|---|---|---|---|---|
| M0 | Foundations: the product path runs live | alpha | 64 | managed mode with the real game is proven live without setcap (docs/launch.md 9), memfd and the container subreaper are proven. Missing: CI runner, saves wired into start/stop, restart through Life, kill -9 loss. |
| A | Act 1 self-leveler, clvl 1-15, 2 h unattended | alpha | 50 | A 90+ minute managed leveling run and a 31 minute campaign (7,600 XP/h, 1 death, corpse retrieved) exist; nav is seed independent; fd growth is a plateau. Missing: any 2 h soak report, clvl 12 inside 2 h, earned-gold economy, deaths and ladder rungs under their bounds over a long run. |
| B | Normal Act 1 complete incl. Andariel, hop to Act 2 | alpha | 40 | Task layer, quest reader, in-memory quest source, Akara claim, 20 dungeon transitions and four boss scripts exist; the Den was cleared live. Missing: boss scripts and questmem wired and verified live, Blood Raven/Countess/Andariel kills, Warriv hop, a full unattended run. |
| G1 | Day-long soak on the product path (24 h) | alpha | 37 | Soak judge understands warm-up and plateau, recycler policy exists, drills passed in the dev topology. Missing: rung 4 restart wiring, recycler wiring, save cycle, the 24 h run, the managed-topology sentinel run. |
| C1 | Normal Acts 2-3 | beta | 8 | Only the merc hire dialog is live-probed; no merc reader, cube, or Act 2-3 handlers. |
| C2 | Normal Acts 4-5, Baal, Nightmare entry | beta | 0 | Largest scripts (seals, Ancients without portal, Baal waves) plus the difficulty screen; ends with the 40 h Normal campaign. |
| F | Headless container + setup wizard, usable by a friend | beta | 41 | The image and its no-caps topology are verified with a probe; wizard backend and d2doctor exist offline. The real game in the container, the installer viewport and a real outside user are missing. |
| G2 | Multi-day soak (72 h) | beta | 0 | Finds slow leaks, log and disk growth and rare states; adds the phone notification and mixed goals. |
| D1 | Nightmare playthrough | good bot | 0 | Same scripts, harder numbers: immunities, resist penalty, teleport. Shows whether Normal scripts generalise. |
| D2 | Hell playthrough + farming runs | good bot | 12 | Farming is what an idle player leaves running; rates against a human baseline make "good" a number. |
| H | Good-bot gate | good bot | 40 | Knob metadata done; class-agnostic combat done live on a Paladin (22.8 kills/min); Paladin builds, merc hire and class infra partial. Missing: 7-day soak, handicap live, telemetry per run, a second class passing A, review. |
| E | Party of 2-4 bots on one host | layer | 37 | The loopback TCP/IP join is proven live (docs/net/multiplayer.md 9); fleet and party code are offline. Joining does not form a party; roles and 4 bots are untested. |
| S | Showcase: stream, scene, public status | layer | 56 | Kick stream, streamctl, scene page, public page generator, HA package, StreamDesk, title updater. Mostly mock-tested; gates nothing. |
| L1 | Upgrades and progression UI | layer | 0 | The idle game itself, built on measured telemetry and the knob data contract. |
| L2 | Hardcore as prestige (rebirth) | layer | 0 | Death resets the run and pays permanent bonuses; character mode already kept in bot.Policy. |
| L3 | Cheats as rewards | layer | 0 | The logged cheat tool becomes an earned reward; cheated sessions excluded from stats. |

Effort to alpha from here: about 17 agent-days of work (M0 1.5, A 4, B 7, G1 4; U, down from 24) plus soak wall time; with 3-4
agents in parallel and the worker host's capacity (about 4 instances) that is roughly 2 calendar weeks if nothing slips (risks.md). Beta adds
about 40 agent-days (C1 14, C2 15, F 10, G2 5), the good-bot gate about 40 more (all U; the first soaks will recalibrate).
Class support (a second class passing A) is in the gate (H.class_*), not in alpha.
