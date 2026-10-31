# NPC greeting skip (internal/town/skip.go, cmd/npcprobe)

## Problem (V from logs, docs/npc.md "Hover misses on first contact")
On a character's first contact with an NPC the greeting holds the menu back. The old bot waited it out: 13-25 s per visit, up
to `greetingWaitS` (90 s); G2 log 61 x `greeting never gave way`, Fresha i5 10:00 Akara: three `npc: interact` clicks at dist 3
opened nothing (hover dead during the speech).

Claims about the fix, all U until the probe ran live: the user (expert) says a left click skips the talking; d2-sme (from
memory) says re-clicking the NPC re-issues the interact, Esc closes menus and skips cinematics, space does nothing.

## Knob `town.greetingSkip` (env.TownTuning.GreetingSkip, default 0 = wait, set after the live probe below)
| value | name | after the first click, menu not open within 1 s |
|---|---|---|
| 0 | wait | old loop: 4 s slices, Esc + re-click, until `greetingWaitS` |
| 1 | npc | re-click the NPC (`interactFresh`: hover-verified; the hand falls back to plain clicks on the NPC tile) |
| 2 | esc | Esc, then re-click as `npc` |
| 3 | ground | `ClickWorld` on the NPC's own tile (then 1 subtile up-left), no hover needed |

Original reasoning for default `npc` (superseded by the live result below): it is what the user and the SME describe for a real player (click the NPC again), it never leaves the NPC (a
click that misses is a walk of at most the 1 to 4 subtiles to him), and the hand's fallback already contains the plain tile
click of `ground`, so it covers both. `esc` risks closing a menu that opened in parallel; `ground` skips the hover check the
hand uses to avoid clicking a wrong unit. Pick another default after the probe.

Sequence per try: first click, wait 1 s for `UINPCMenu`; else up to 2 skip actions, each followed by a 2 s wait (about 5 s per
try); still no menu: Esc and the outer try (walk, click) starts over, `NPCOpenTries` rounds (default 3). Nothing ever clicks
empty ground; no strategy clicks farther than 1 subtile from the NPC. The proof is the menu flag, not a timer.

## Logs (key msg)
- `npc: time-to-panel` ms, first-contact (menu not open 1 s after the first click), clicks (first click + skip actions), strategy, class
- `npc: speech skipped by <strategy>` (only when a skip action preceded the menu)
- `npc: skip click not hovered` (hover dead, the wait still runs), `npc: menu did not open` carries the final error.

## Special cases (kept as waits, bounded)
- `Town.WaitNPCMoves(ctx, class, timeout)`: wait until a unit changes position (Act 2 Harem-gate guard). Wiring is in the quest
  / nav layer (not in town; no code there waits on the guard today, U).
- `Town.SkipCinematic(ctx, fromArea, timeout)`: Esc, then wait for the area to change. Act hops (quest/hops.go) should use it
  instead of a fixed sleep (not wired, outside this package).

## Offline tests (internal/town/skip_test.go, sim fakes)
Menu at once, speech skipped by a click on the NPC tile with hover dead (`SpeechSkipByClick`, nobody walks away), two clicks needed,
bounded when nothing skips, wait mode, guard moves, cinematic.

## Live probe (main agent, fresh character)
1. Fresh character on a free instance: `tools/soak/fresh.sh` (see its header). The bot and soak of that instance must be stopped
   (the probe refuses while a d2bot/d2soak/d2dsoak process of the instance runs).
2. `tools/live.sh <agent> npcprobe` runs the default plan `npc:148,esc:154,ground:147,wait:150` (Akara, Charsi, Gheed, Kashya,
   one strategy per NPC, because first contact happens once per NPC). Output columns: strategy, class, visit, ms, clicks,
   first-contact, result. Add `-again` for a second, non-first visit as the baseline. Plan elements are `strategy:class`.
3. Direct run on the worker host: `D2BOT_INSTANCE=i5 ./npcprobe -selflock` (takes /tmp/d2lod-game-i5.lock without waiting, refuses if held).
4. Repeat on a second fresh character with the strategies rotated (e.g. `-plan ground:148,npc:154,esc:147,wait:150`): one NPC per
   strategy is a single sample each. Needs about 4 NPCs x <= 90 s; stays in the Rogue Encampment, ends with Esc.

What the live run must confirm: (a) which strategy gets the menu within the 1 s + 5 s bound on first contact, and how many clicks;
(b) that a click skips the speech and the menu still comes (or the menu was there at once, then first-contact=false and the
speech is irrelevant); (c) that `ground` / `npc` never moved the character; (d) then set the winner as the `greetingSkip` default.

## LIVE RESULT (V, 2026-10-30, i5, three fresh Sorceresses Freshb/c/d, no cheats, probe at 733da90)
Run A Freshb `npc:148,esc:154,ground:147,wait:150`; B Freshc `ground:148,npc:154,esc:147,wait:150`; C Freshd `esc:148,ground:154,wait:147,npc:150`
(148 Akara, 154 Charsi, 147 Gheed, 150 Kashya). Result = the single probe call (3 outer tries of about 6 s, then it gives up).
| run | strategy | NPC | result | notes |
|---|---|---|---|---|
| A | npc | Akara | menu on try 2, wall 20.8 s | try 1: click + 2 skip clicks, all "not hovered", no menu in 11 s; try 2 opened at once (speech over) |
| A | esc | Charsi | menu at once (300 ms), wall 12 s | walk took 12 s; speech already over |
| A | ground | Gheed | FAILED, wall 20.3 s | 3 tries, pointer never hovered |
| A | wait | Kashya | menu at once, wall 15.9 s | |
| B | ground | Akara | FAILED, wall 19.4 s | 3 tries, hover dead |
| B | npc | Charsi | menu at 2.7 s, wall 22.2 s | |
| B | esc | Gheed | menu at once, wall 23.8 s | |
| B | wait | Kashya | menu at once, wall 14.6 s | |
| C | esc | Akara | FAILED, wall 22.9 s | Esc + clicks during the speech did nothing |
| C | ground | Charsi | menu on try 2 (312 ms), wall 19 s | try 1 no menu in 6 s |
| C | wait | Gheed | menu after 34.8 s, 2 clicks | first-contact=true; hover dead 3 x 7 s slices, then open |
| C | npc | Kashya | FAILED, wall 18.8 s | 3 tries, hover dead |
Baseline `-again` on Freshb (second visit): 256-316 ms, 1 click, every NPC.
Findings:
- The greeting starts on PROXIMITY, during the walk, not on the click. Whether it is over when the bot clicks depends on the NPC and
  the walk: Charsi/Kashya mostly done (menu at once), Akara and Gheed still talking (their speech plays 20-35 s from arrival).
  The probe's first-contact flag therefore only fires when the click lands mid-speech (4 of 12 rows).
- NO strategy skipped the speech. While it plays the pointer never hovers the NPC; clicks on the NPC tile (npc fallback, ground),
  Esc plus click (esc) all left the menu closed until the speech ended by itself (log "skip click not hovered", no "speech skipped by"
  except the wait row). The user's claim that a click skips it is NOT reproduced here (U: clicks may skip with a working hover, which
  never happened on first contact; the speech text box showed up on screen as a subtitle: `tools/shot.sh i5` showed Akara's
  "Greetings, young Sorceress..." box).
- The character never walked away in any row (the NPC's tile and the character stayed put; npc/ground clicks stay within 1 subtile).
- Failure counts on mid-speech clicks: npc 1 of 3, esc 1 of 3, ground 2 of 3 (all on Akara/Gheed/Kashya), wait 0 of 3 (it waits in
  4 s slices up to greetingWaitS, so it outlasts the 20 s the 3 outer tries give). Sample sizes are 3 per strategy on different NPCs:
  indicative, not statistical.
- Probe quirk: a FAILED row printed the ms/clicks of the previous row (stale time-to-panel); read the result column.
Conclusion: since nothing is faster than the speech, the default is `0` (wait): the only strategy that never failed, and it sends
no pointless clicks. Time to menu on first contact is the speech length: 12-35 s. A real speed-up needs a different lever (arrive
after the speech: approach, wait out of range, or let the speech finish while doing something else: U).
