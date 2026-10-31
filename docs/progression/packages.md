# Proposed package split, owners, order

Owners are the existing agent roles from `docs/archive/review/parallel-work.md` and AGENTS.md (names are roles; the main agent assigns). Sizes S <= 1 day, M 1-3, L 3-5 days of agent time.
No package below edits another's directory; requests (R1...) are changes someone else must make.

## 1. Packages

| package | contains | owner | size | depends on | verified by |
|---|---|---|---|---|---|
| `internal/saves` (exists as WP-SAVES) | `Progress(d2s)`: raw quest WORDs per index, waypoints, difficulty, act, map id | SAVES | M (planned) | WP-STATE | O-1, V-01 |
| `internal/progress` | `Observation` assembler: merges T1 (save), T2 (memory reader), proxies into `Quests/Waypoints/Merc/Difficulty` with `Evidence` and staleness; `QuestBits` decode; quest index table | PROGRESS (new; LEVELER-adjacent) | M | saves, `d2` readers | V-01, V-05 |
| `internal/d2` additions | `quest.go`, `waypoint.go` memory readers (T2), merc unit reader | MEM/D2 | S-M | V-05, V-09 | V-05, V-09 |
| `internal/town/dialog` | `Dialog.Open/Pick/Close`, greeting handling, `dialogtable.json` | TOWN2 | M | V-04 | V-04, V-07 |
| `internal/town/merc` | hire, revive, equip, merc state | TOWN2 | M | dialog, d2 merc reader | V-09 |
| `internal/town/cube` | cube UI, `Cube(recipe)` | TOWN2 | M | none | live, own spike |
| `internal/bot/steps` | `Step`, `Plan`, `Executor`, `Kind`, `Fault`, constraints, `stepworld` fake, primitives | BOT | L | contracts | fakes only |
| `internal/bot/quests` | handlers (one file per act), `Handler(q,d) []Node` | PROGRESS then BOT | L over time | steps, dialog, progress | per handler |
| `internal/bot/acts` | hop steps (A1-A2, A2-A3, A3-A4, A4-A5), `ActivateTownWP` | PROGRESS | S-M | dialog, V-03/V-07 | V-03, V-07 |
| `internal/bot/bosses` | `BossScript` data + shared executor + hazards data | COMBAT2 + BOT | L over time | steps, V-10/11 | V-10, V-11 |
| `internal/launch/difficulty` (inside launch/flow) | difficulty dialog click, byte verify, hop save-and-exit | LAUNCH/OOG | M | V-08 | V-08 |
| `internal/goals` additions | goal types `quest`, `waypoints`, `difficulty`, `campaign`, `Steps`, `Verified` | GOALS | M | tracking.md | tests |
| `internal/planner` | `next(o,h)`, campaigns, gates, requires graph (pure) | PLANNER (GOALS) | M | goals.Observation | unit tests |

Directory placement rule: pure logic (planner, quest tables, progress assembly) lives outside `internal/bot` so it is testable without the Runner; only `bot/steps` knows
`Deps` and the Runner. This also avoids further growth of `internal/bot` (finding F8 layering).

## 2. Requests to other agents or the main agent

| id | request | to | why |
|---|---|---|---|
| R1 | `internal/cheat/d2s`: write the quest block (`quest=INDEX:WORD,...`, per difficulty) via `d2cheat save`; tests on hand-built fixtures | CHEAT (idle: main agent) | prepared states for every handler and hop test; removes hours of playing (largest single saver) |
| R2 | `saves.Progress`: return raw quest WORD per index and the active difficulty/act byte | SAVES | quests.md section 2 |
| R3 | `d2cheat`: confirm `act=`, `difficulty=`, `progression=` change what the client loads (V-03, V-08) | CHEAT | cheats.md says unverified |
| R4 | `tools/live.sh`: forward `D2BOT_AGENT` so the cheat ledger actor is right (cheats.md already notes it is not forwarded) | main | ledger hygiene |
| R5 | COMBAT2: treat a friendly monster owned by the player (merc) as non-hostile everywhere; add `MercDead` advisory cause | COMBAT2 | merc.md |
| R6 | LOOT/TOWN2: pickit must always take quest items and skill/stat books; `Looter` verdict `quest` tier S | TOWN2 | quests.md Hell's Forge |
| R7 | NAV: `Presets(class)` query returning world positions by preset class id; ExitTo for stairs/portal presets verified in Act 2-5 | NAV | boss anchors |
| R8 | GOALS: add `needs_attention` note and `campaignId`, `Steps` | GOALS | tracking.md, tasklayer.md |
| R9 | PLAN data: one copy of plan data (F15) before quests consume `progression-*.json` | main | avoid three copies |
| R10 | WP-REPLAY: record at least one fixture per arena for the boss executor tests | REPLAY | bosses.md section 5 |

## 3. Order (milestones, with cheapest proof)

| step | content | proof | time |
|---|---|---|---|
| P0 | O-1, V-02, O-2 | tags flip, no code | 1 h |
| P1 | R1, R2, `internal/progress` skeleton, V-01, V-03 | Normal Act 1 state reproducible on Multi | 1 day |
| P2 | `bot/steps` + fakes + the 3 simplest handlers (Den, Radament, Fallen Angel: all "kill and read a point") | stepworld tests green; live one handler on i4 | 3 days |
| P3 | dialog layer + V-04 + hop steps A1->A2 | V-07 | 3 days |
| P4 | boss scripts Andariel, Duriel | V-10, V-11 | 4 days |
| P5 | A2->A3 hop, Khalim's Will chain (needs cube), Mephisto | cube spike + V-12 | 5 days |
| P6 | A3->A4, Diablo | seal executor; Diablo is the first large script | 5 days |
| P7 | A4->A5, Ancients (NoTownPortal), Baal | constraints tested; Baal waves | 5 days |
| P8 | merc (V-09), difficulty hop (V-08), `nightmare_playthrough` | first full Normal clear and hop to Nightmare on a throwaway | 5 days |

Roadmap mapping (finding F11): P1-P3 = milestone M2 "Normal Act 1 to Act 2" made explicit; P4-P7 = Normal complete; P8 = first difficulty hop. Each is shippable
alone because plans carry no cursor (tasklayer.md): a bot stuck at P4 still runs Act 1 farming.

## 4. Risks (ranked)

1. **Dialog row blindness** (no text reads): mitigated by recorded indices and effect verification; wrong clicks on irreversible rows (imbue, socket) are blocked by `Step.Need` and a confirm-by-effect rule.
2. **Quest bit semantics** (U): isolated in one decode function (quests.md section 2). If V-01 fails to give clear bits, fall back to proxies plus the 0x11 edge plus T1 at menus.
3. **Expansion flag** off for a test character: Acts 4-5 and mercs absent. V-02 first.
4. **Per-game random layouts** (tombs, seals A/B): preset anchors, never coordinates.
5. **Scope**: handlers multiply; keep each ~50 lines of node data and share primitives. Resist writing a state machine per quest.
