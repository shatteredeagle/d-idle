# First play session guide (TASK-134)

> status: written 2026-11-08 from docs on main only. Nothing here was run for this guide. Tags: V verified live (as the source doc says), S fixtures or code tests only, U unverified. Every claim points at its doc; when a doc and this page disagree, the doc wins.

## Do this first

1. Have your own Diablo II 1.14d folder ready (needs `d2data`, `d2exp`, `d2char`, `Patch_D2` .mpq; V `docs/tester-live.md`).
2. Start the container (section A), open the printed watch URL, run the wizard (class: Paladin is preselected), press Start.
3. Open the run sheet (section D) and follow it minute by minute.

Everything else below is reference. Open the screens you need, ignore the rest.

## A. Start a session

Two ways. Use the container first: it is the path that ran for an hour with a person-less script (V `docs/tester-live.md`, realtime hour, 0 stalls, 0 deaths).

### A1. Container (the way a friend will use it)

1. Build or get the image `d2dbot:tester` (the image already exists on the worker host from earlier runs; a clean build is `deploy/d2d-container/context.sh CTX` then `docker build`, see `docs/d2d-container/README.md`).
2. In `deploy/d2d-container/`: `D2_GAME_DIR=<your game folder> D2_WATCH_ORIGIN=http://<the name you will type in the browser>:8891 docker compose -f compose.tester.yml up -d`. Publish is loopback by default; to open it from another machine set `D2_PUBLISH=0.0.0.0` (V used in the second-machine run, `docs/tester-live.md`).
3. Read the log once: `docker compose -f compose.tester.yml logs`. The first start prints the setup URL and the watch token ONCE. Later: `docker compose -f compose.tester.yml exec d2dbot cat /data/watch.token`. Treat the token like a password.
4. Open the setup URL. Wizard steps: data check (1.14d verdict), go-ahead, name, class, goals, access, Start (V over the API; the page was NOT opened in a browser, U, defect D11).
5. Class pick: the wizard lists only classes that have a leveling table (`docs/wizardclass.md`): Paladin (default), Necromancer, Amazon, Assassin, Barbarian, Druid, Sorceress. Only Paladin and Sorceress were ever played in a container; the others are listed, not proven (U).
6. After Start there is a 1 to 3 minute warm-up, then the hero walks out. Watch URL: `http://<host>:8891/watch?t=<token>`. Answers go to port 8892 (same host, set up for you by the page).
7. Stop: `docker compose -f compose.tester.yml stop` (up to 45 s; the bot saves the character). Same volume keeps the hero (V, recreated 4 times).
8. Getting new builds into your container: a normal image needs a rebuild and a recreate for every bot change. For testing, build a DEV image once and recreate the container once from it; after that `tools/dev-update.sh --to <container>` swaps the bot in about 6 seconds with the same volume, and `--rollback` goes back (`docs/dev-update.md`; the container, volume and hero are kept, the bot restarts like a `docker stop` and start; S/V on a throwaway container, a hero mid-fight U).

### A2. `d2bot play` (on a machine with a d2d backend)

`d2bot play --backend d2d --managed [normal d2bot flags]` (S/U: the live smoke is still a checklist, `docs/session-record.md`). It sets the same environment the container sets, binds the viewer on 0.0.0.0:8891 and the answers on 0.0.0.0:8892 (LAN), creates `<data>/watch.token` (0600) and prints `http://<lan ip>:8891/watch?t=<token>` once. With the d2d backend also set `D2BOT_WATCH_VIDEO_SOURCE=tap`. Stop with Ctrl-C; the summary is written then. `play` ran under lockstep with a Paladin on the worker host (V `docs/paladinsmoke.md`), not by a person yet.

## B. The screens

All are GET pages behind the watch token (`?t=<token>` once, then a cookie). Without the token you get 401. Writes go to port 8892 and need the right origin (`D2_WATCH_ORIGIN`); a wrong origin gives 403.

| URL | What it shows | State |
|---|---|---|
| `/watch` | Live screen: video, Hero now (class, level, title), question cards with choices and a fade bar, journal, rules, orders map, tally strip in the header. If you were away over 20 minutes it sends you ONCE to the return screen; `/watch?live=1` skips that | V video and questions in the container; page in a browser U |
| `/watch/return` (`/api/return`) | What happened while you were away: headline, closest call, deaths, drops, expedition cards, questions missed with "answer once for all" and "make this a rule", tally, Codex line, "Unlocked since you left", policy before/after cards, proposals, notes box | wired (TASK-140); live smoke of the feed V, page S |
| `/api/senses` + Game tab panel | Fog-of-war minimap beside the video, hero cross, exits, waypoints. Tier 1 is on by default (`senses.tier` 1) | S, never seen in a browser or on a live level |
| `/watch/policy` (`/api/policy`) | Chores sheet: repair, restock, town trips, retreat, potion mix, belt fill, whites, sell... One row per chore, change = one click. Poor choices are allowed and flagged "your call" | S; bot reading the rules live is U |
| `/api/proposals` and the return screen section | "I am level N and ready for <Zone>. Go there?" go / stay / later. Default stay, never auto-orders | S, wired with fixtures |
| `/watch/pacts` (`/api/pacts`) | Pact offer page. Pacts are locked silhouettes until the Andariel (or Baal Normal) kill | S; not reachable in 30 min |
| `/watch/hall` (`/api/hall`) | Hall of Fame, titles, and the retire question | S; opens only on an Andariel kill |
| `/api/tally` | Tally balance, recent credits, next buys | wired; works without the overlay ("events mode"); amounts U |
| `/api/codex` | Act 1 firsts (37 rows) | wired, V live smoke 2026-11-07: 3 firsts in 10 minutes |
| `/api/unlocks`, POST `/api/unlocks/buy` | Unlock ladder: open entries, locked teasers with prices | S |

## C. What is real and what is not

Read this before you judge the game. "Wired" means the code path is connected in `d2bot`; "verified live" means a run against a real game saw it.

| Feature | Wired in a real run? | Verified live? | Known gaps (from the doc) |
|---|---|---|---|
| Live screen, video (tap), questions, answers, orders | yes | V container hour (3.4 frames/s, 33 questions, 27 answered) | page in a browser U; video reconnected twice (D12) |
| Teach-the-hero questions | yes | V (list below) | same card reopened about every minute (D7 / TASK-137, fixed offline); fight-time cards rarely fire at low level |
| Return screen | yes (TASK-140) | partly: data feed V, 20 min landing not exercised live | notes sink needs the session record; gold, repairs, purchases "not reported" |
| Session record | yes (`d2bot play`, container) | V in lockstep Paladin run; token absent from files | `stayed_in_tab_min` counts only minutes with the video open (U) |
| Tally strip, charter 402 | yes | V over the API (events mode); charter 402 V (`docs/paladinsmoke.md`) | prices and ranges are U; pace 60 to 75 tally in hour one not measured |
| Policy sheet | yes | no (offline only) | identify, potion buy, full stash, keep-kind points are not asked by the bot yet: those rows only store the choice; potion tier is a record |
| All-mana belt | yes (knob `resource.beltPlan`, TASK-164) | NO: live drill skipped (host busy), AC 5 open | risky by design; healing safety floor stays |
| Zone proposals | yes (fixtures) | no | ranges and class lead are U; Blood Moor to Cold Plains is the first pair |
| Codex firsts | yes | V smoke | champion, shrine, difficulty and true "first sight" rows never fire |
| Minimap (`/api/senses`) | yes (TASK-148) | no | hero cross on a real d2d level and the exits are U |
| Unlock ladder, reveals, "Unlocked since you left" | yes (TASK-154, 185) | no | `retire` has no emitter; waypoint events only from this runner; senses tiers 2 to 7 carry no data yet |
| Bestiary, risk meter, order queue | yes (TASK-185) | no | quest reader never run against a game; resist gap always 0; queue only in memory |
| Alerts | in-page centre only, senders off | no | Telegram or push senders are OFF |
| Hall of Fame / retire | yes | no | needs Andariel; permanent death has no signal |
| Pacts | yes | no | locked until Andariel; away-consent rules offline only |
| Wizard class pick | yes | Sorceress V; Paladin hand-seeded V | the first real non-Paladin pick in a container not done |

## D. The 30 minute run sheet

Pace first (do not be surprised). Realtime means game time equals your clock.

- Paladin realtime: 62 kills, clvl 2 at 20 minutes, clvl 3 expected at about 22 minutes (V `docs/palpace.md`). It is NOT clvl 5 to 6 in 20 minutes. Lockstep (about 20x) is a test mode, not what you play.
- The hero stays in Blood Moor and town for the whole half hour. The Blood Moor gold recovery loop is still being fixed (TASK-183 in progress); you may see "Recovery: farming gold" and long idle walking in a dry field (TASK-138, TASK-177).
- The Might aura is fixed and V (`docs/palready.md`). The equip fix (#427, TASK-182) is NOT landed: a bought bow or armor may not reach its slot; ignore "did not reach slot" warnings.

Minute by minute:

1. Min 0 to 3: wizard done, warm-up. Open `/watch`. Look for: video moving, "Hero now" card, journal lines. Nothing to do.
2. Min 3 to 6: the hero leaves the Rogue Encampment and fights in Blood Moor. Look at the Game tab minimap: it should show the area around the hero only (fog). If the panel stays empty, note it (U).
3. Min 5 to 10: first questions. Expect, in this order of likelihood (V `docs/probes/t0drill.md`, `docs/tester-live.md`):
   - `t0.area.toohard`: "Forgotten Tower is too hard for you right now: go, skip, lower?" Answer: skip. It is route planning; the hero is not near it. If it comes back within a minute, write down the time (known defect, D7).
   - `t0.retreat.drink`: a potion question during a fight; `t1.repair`, `t1.restock`, `t1.return` in town.
   - `t0.first`: "first time I meet X" cards from the Codex (Fallen, Blood Moor, Quill Rat, Corpsefire). Answer yes or no; the Codex line is on the return screen later.
   How to answer: tap a choice on the card. The "if you do nothing" default applies after the fade (about 45 s). Tap the same answer several times and the hero offers to make it a rule; accept once to see a rule appear in Rules.
4. Min 8 to 12: change a policy row. Open `/watch/policy`, row "Potion mix", pick another option, read the one-line risk. One click applies it (needs the tuner; if you get 501 nothing was changed). Then try row "Repair" or "Restock": those cards should stop opening (U live).
5. Min 12 to 15: try the all-mana belt. Policy sheet, "Potion mix", option "all mana" (flagged your call). Watch the next town trip: it should buy only mana potions. This was never run live (TASK-164 AC 5), so report what you see; expect the hero to go to town for potions and the kit gate to stay in the safe field.
6. Min 15 to 18: order a zone above range. On the orders map give "go" to Stony Field or Cold Plains while you are clvl 1 to 2. Without confirming, the answer is 402 `charter-needed`; the page asks with a confirm box showing the price (20, 40 or 80 tally by how far below you are; V over the API, `docs/paladinsmoke.md`). The hero also writes one journal line: "That is above my range ... it costs N tally; go?". Cancel on the confirm box for no charge. If you accept with too little tally you get 402 `insufficient`; nothing is booked.
7. Min 18 to 20: look at the tally strip in the page header. It shows balance, recent credits (kills give 1 per 25, firsts 5, trips 2) and the next buy. If it says "off" it also says why ("tally off" hints, section G).
8. Min 20 to 25: Paladin should reach clvl 3. Open the minimap again, then `/api/senses` in a tab for the raw map (token needed). Check `/watch/pacts`: all Pacts are `???` silhouettes. That is correct.
9. Min 25 to 28: if a proposal appears (the hero ready for Cold Plains, level and potions ok), answer go, stay or later. go places one order and the charter rules apply. At clvl 3 this is unlikely; if none came, say so.
10. Min 28 to 30: to see the return screen, close the tab, wait over 20 minutes, then open `/watch` once. Otherwise stop with `docker compose stop` and wait for it to finish (the bot saves).

Hall of Fame: the retire question opens only when Andariel dies. That will not happen in 30 minutes; do not expect it. `/watch/hall` shows an empty Hall with your title so far (S).

Ladder reveals: when something unlocks (first kill, level 2, first magic item, first waypoint) the Chronicle gets a line "Unlocked: ..." and nothing asks you. They are only listed on the return screen under "Unlocked since you left". To see that section you must leave for over 20 minutes and then open `/watch` once (it redirects to `/watch/return`).

## E. The baseline sessions (TASK-107; hypotheses H4 and H1)

Why: H4 is watching only (boredom baseline); H1 is "questions feel good". Source: `docs/research/direction-review.md` section 4. Do them on two different days or at least one hour apart.

1. H4, one hour, video only. Open `/watch`, do not answer cards, do not give orders, do not change policies. Note the clock time when you close the tab. That minus the start is your boredom time.
2. H1, 30 minutes with cards. Start from the clvl 6 Cold Plains save if one exists (TASK-107 names it; U whether it is on hand, check the task). Otherwise a fresh hero: say so in your report, because a fresh hero rarely gets fight-time questions (V `docs/probes/t0drill.md`: none opened in 40 minutes).
3. Answer every card you can. At the end read `summary.md`: the number to quote is `questions_per_10min` (index = ten-minute bucket of the GAME clock) plus rules learned. Goal from the review: 3 or more questions per 10 minutes in context, 1 or more rule learned, and "did you feel you taught it" at 4 of 5 or better.

## F. Where the session record lands, and what to send back

1. Files: `<instance state>/sessions/<utc>-<profile>/` (for example `20261108T100000Z-human`) holds `session.jsonl`, `decisions.jsonl`, `summary.json`, `summary.md`. In the container the state is under `/data/instances/main/state/` in the volume; `docs/paladinsmoke.md` found it under `ladder/sessions/` (V). Run `find` inside the container volume if unsure; the summary is written when the container or `d2bot play` stops cleanly.
2. Export (opt-in, never sends anything): `d2bot sessions export -o out.jsonl <SESSION_DIR>`. It scrubs names, paths and token-like strings and prints how many values it replaced. Read the file before you send it.
3. Report three sentences per hypothesis (`docs/research/direction-review.md` section 4):
   - H1 (teaching feels good with context): questions per 10 min, rules learned, did it feel like teaching?
   - H3 (orders are the game): number of deliberate orders, whether you wished for "until level N" or a queue, whether you stayed in the tab. Counted by the record (`orders`, `mean_sec_between_orders`).
   - H4 (watching is enough): the minute you closed the tab.
4. Also list anything from section G that fired.

## G. Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Wizard refuses the class | It lists only classes with a leveling table. Choosing one outside the list is an error naming it (`unknown class "x"`). Pick from the list; Paladin is the default. |
| "No game folder" or d2d exits at once | Missing `d2char.mpq` or `Patch_D2.mpq` in `D2_GAME_DIR`. All four of `d2data`, `d2exp`, `d2char`, `Patch_D2` are required; sound and music are optional (V `docs/tester-live.md`). The folder must be a 1.14d install: the wizard verdict shows the version. |
| 401 on the watch page | Add `?t=<token>` once. Token: see A1 step 3. Never paste it in chat. |
| 403 on answering a card | The page origin differs from `D2_WATCH_ORIGIN`. Set it to exactly the URL you type in the browser, including the host name, and recreate the container. |
| Cannot reach the page from another machine | Compose publishes on loopback by default. Set `D2_PUBLISH=0.0.0.0`. Ports: 8891 watch, 8892 answers. `d2bot play` already binds the LAN. |
| Port in use | Another container or a second `d2bot play`. Stop it first. |
| No cards for ten minutes | Normal at clvl 1 to 5 (V `docs/research/direction-review.md` section 5). Not broken. |
| Tally strip says "tally off" or "off in vanilla mode" | The ledger is missing: the ladder or watch is off, or the profile has no ledger. The reason is in the strip text (`watch.TallyOffText`). With the container it should run in events mode. If it stays off, copy the text into your report. |
| Policy change answers 501 | No tuner is connected; nothing changed. Report it. |
| Hero never leaves Blood Moor | TASK-138 and TASK-183 (see section H). Wait for the stall rule (ends at 20 to 25 min) or report it. |
| Video freezes | Reload the tab. The tap source reconnected twice in the V hour. |

## H. Known broken (open cards on main)

| Card | State | What it means for you |
|---|---|---|
| TASK-138 freshstart | AC 1 done, AC 2 NOT met | No uncheated realtime hour has yet reached the next campaign step. Stall rule verified live but XP per hour did not improve. |
| TASK-148 sensesmap | built, live check not run | The minimap has never been seen on a real level. Part of this session IS that check (checklist in `docs/senses-map.md`). |
| TASK-164 beltplan | built, AC 5 live drill open | All-mana belt is untested live. |
| TASK-183 recoverypace | in progress | Hero camps Blood Moor on the gold farm for much of the first 90 minutes; class pace is not measured. |
| TASK-184 cpnav | open | The Assassin cannot leave Blood Moor for Cold Plains. Do not pick the Assassin. |
| TASK-182 equipfail | open (#427 not landed) | Equip click leaves the cursor empty; items may not reach slots. The Amazon loops buying and selling a bow. Avoid Amazon and Barbarian for now. |
| TASK-162 unlockalerts | in-page only | No phone pings; senders off. |
| TASK-120 ghcrpublish | needs your go | No published image yet. |
| TASK-107 h1baseline | open | This session is its input. |

## I. For a friend

1. What to send: nothing yet. The registry image is NOT published (TASK-120 needs your go; the workflow is inert, dispatch only, and the environment `tester-publish` must exist first, `docs/tester-registry.md` section 3).
2. Until then the friend builds the image from public inputs on a machine with Docker: `deploy/d2d-container/context.sh CTX` then `docker build -t d2dbot:tester CTX` (about 6 to 8 minutes on the worker host, V `docs/tester-imgrepro.md`; a clean machine needs only network). The d2d source is the public fork tag `didle-gate2-lab3` (commit `f75816c616c8`); the vendored archives are sha256-pinned.
3. The friend needs their own Diablo II 1.14d files. No game data is in the image; it is checked in the publish workflow.
4. Then they use section A1, with `D2_GAME_DIR` pointing at their folder, and send you the exported session (section F) if they agree.
5. The wizard asks them for a go-ahead, name, class, goals and a passcode/token.
6. When the image is published (later): `D2_IMAGE=ghcr.io/shatteredeagle/didle-tester:latest docker compose pull && up -d`; the character and token survive updates.

GPL and licence open items before any outside copy (`docs/tester-licence.md` section 6):

1. GPL reviewer: named or waived (open).
2. Freepik icon: whether its terms allow redistribution inside the d2d binary is unverified (section 5); options are confirm or patch the icon out.
3. Your explicit go on the day of the first copy of an image to anyone else.
4. The in-image `SOURCE.txt` still says "NOT FOR DISTRIBUTION"; the contact placeholder for the source offer is empty. Both must be resolved at the cut (`docs/tester-registry.md` section 5).
5. Rewrite history or not (session-URL trailers): `docs/research/fork-publication-scan.md`, D1.

Sending a hand-built image to a friend is a "first outside copy" and needs your go; nothing here does that for you.
