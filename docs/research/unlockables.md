# Unlockable features: a design note (not a decision)

> status: brainstorm of 2026-11-07, recorded at the user's go. Not a decision: `docs/decisions.md` is untouched. The cards are Proposed, milestone m-4, and none of them is on the two-week playable path.

Tags: V verified (read the page), S secondary (search snippet or second-hand), U unverified (from memory). The V and S tags on section 4 come from the brief of 2026-11-07; the pages were not re-fetched for this note.

## 1. Purpose

The user's words: "unlockable... minimap display; further unlocks enemies, enemy health bars, elites, bosses"; "think of other features like this... 10".

The watch UI should grow with the hero. A new viewer sees little; each unlock opens a view or a control, so the first sessions have a sense of progress without the bot being weaker. The minimap and senses ladder is written by another agent and is NOT repeated here (cards `sensesmap`, `sensesenemies`, `senseselites`, `sensesunlock`, `sensesvideo`); this note adds ten other unlockables and the shared framework. As of this note none of the senses cards exists on origin/main, so the dependency of `unlockladder` on `sensesunlock` is stated here in text and must be added to the card when `sensesunlock` lands.

## 2. The ten

Source column: veteran SME (a D2 player's view) or research (section 4).

| # | Feature | What it is | Trigger | Source | Size | D2 notes |
|---|---|---|---|---|---|---|
| 1 | Chronicle | Kill feed first, later the run timeline (expedition cards) | First kill; the timeline after the first expedition card is read | research (Paperclips log, Progress Quest) | S then M | Kills are an observed event already used by the tally. |
| 2 | Hero voice and journal | One-line comments from the hero; journal entries | First level-up, first death, each Codex first | research (A Dark Room) | S | Codex firsts come from the Act 1 firsts card. |
| 3 | Quest log | Quest list with state | Akara hands out the Den of Evil quest | veteran SME | S | Mirrors the D2 quest log; Akara's quest is the natural first. |
| 4 | Identify reveal | Card flip when the first magic or rare item is picked up; later affixes, colour, sound | First magic item; later first unique or set | veteran SME (Cain's identify) | M | Check how the bot handles unidentified items before the reveal. |
| 5 | Bestiary | Cain's notes: card per monster kind (type, resists, attack pattern, drops); silhouettes for unmet kinds | 10 kills of a kind; 50 for resist details | research (Hades codex, Melvor mastery) | M | Do not tie to the Cain rescue (late for a Sorceress). |
| 6 | Risk meter | Ready / marginal / turn back before a boss, from the hero's own numbers | First near-death survived | veteran SME | M | Must use the numbers the bot uses (potions, life ratio, resists vs the zone). |
| 7 | Waypoint atlas | Lit nodes, silhouettes for locked, the hero's proposed next zone | First waypoint taken | veteran SME | S | Atlas is a view only. |
| 8 | Order queue | Queue 2 or 3 orders; one slot per town service reached | Den of Evil cleared once | research (Increlution) | M | Adds agency; see cautions. |
| 9 | Alerts and pings | Phone or desktop notifications: unique, death, level-up, hero returned to town | Tally spend on a charter | research (idle-game check-in timers) | M | Opt-in channels only. |
| 10 | Hall of Fame and replay | Gallery of retired heroes with legacy charms; 30 s replay clip on death | Retire after Andariel or first death | research (Rogue Legacy, Hades House) | M | Depends on the legacyhall card. |

## 3. Draft first-40-minutes schedule

Draft, to be tuned by playing. Times are approximate.

- Minute 0: automap, the first rung of the senses ladder.
- First kill: Chronicle opens (kill feed).
- First level-up: hero voice speaks.
- First magic item: identify reveal.
- Akara's Den quest accepted: quest log.
- About 25 kills: monster dots on the map (senses ladder).
- 10 kills of one kind: first bestiary page.
- Den of Evil cleared: reward chime, quest log entry closes, order queue opens.

## 4. Patterns from research

- Incremental game (https://en.wikipedia.org/wiki/Incremental_game) [V]: unlocks are tiers bought with a spent currency; automation means interacting less; offline earnings.
- 7 idle games, dev.to (article on seven idle games) [S]: first upgrade within 60 s, first automation within 3 min, first prestige at 10 to 15 min, a new phase every 3 to 5 min, show only the next 2 or 3 upgrades. One author's rule of thumb, not a standard.
- Idle game design principles (https://ericguan.substack.com/, idle-game-design-principles) [S]: staggered check-in timers (20 min, 5 h, 2 days); offline caps; multiple currencies need distinct choices.
- Universal Paperclips (https://en.wikipedia.org/wiki/Universal_Paperclips) [V]: one button first; narrator reveals; the log carries the story.
- Progress Quest (https://en.wikipedia.org/wiki/Progress_Quest) [V]: zero-player game.
- Cookie Clicker (https://en.wikipedia.org/wiki/Cookie_Clicker) [V]: building counts unlock upgrades; achievements unlock tiers; shadow achievements are excluded here.
- Hades (https://en.wikipedia.org/wiki/Hades_(video_game)) [V]: the House hub; Pact of Punishment is opt-in difficulty after the first clear.
- Vampire Survivors (https://en.wikipedia.org/wiki/Vampire_Survivors) [S]: challenges unlock content.
- Rogue Legacy (https://en.wikipedia.org/wiki/Rogue_Legacy) [S]: gold buys manor upgrades; blueprints are found.
- Dead Cells (https://en.wikipedia.org/wiki/Dead_Cells) [S]: runes chain; boss cells are opt-in.
- Slay the Spire (https://en.wikipedia.org/wiki/Slay_the_Spire) [S]: unlocks from won and failed runs; ascension.
- A Dark Room [S]: one verb first; system changes tell the story.
- Increlution [S]: queue a loop only after doing it once.
- NGU Idle [S]: quality of life behind a grind is resented.

## 5. Cautions

- Do not gate basics behind grind.
- Never unlock silently: reveal through a silhouette or teaser tied to the Codex plus a log line.
- Control unlocks must add agency, not make the watcher redundant.
- Spoilers: modifiers and uniques only after the first encounter; scouting shows categories.
- Opt-in difficulty never gates the main story.
- Technical checks before building each:
  - Bestiary: Cain's rescue is late for a Sorceress, so do not tie the bestiary to it.
  - Order queue: check the Den of Evil reward conditions against the data.
  - Identify reveal: check how the bot handles unidentified items before the reveal.
  - Risk meter: it must use the same numbers the bot uses.

## 6. Other candidates (not in the ten)

Loot call-outs and sounds (default show-all); policy presets; scouting report; return-screen digest; frame skins by act; secrets sheet; opt-in Hardcore Pact (a policy modifier that touches hero play: needs a user ruling; Hardcore is a D2 mode).

## 7. Open questions

- Cadence: one new unlock per session, or per Act?
- Hardcore Pact: does the user want it at all?

## Cards

Built: the framework of `unlockladder` is in `docs/unlock-ladder.md` (engine `internal/unlock`, table `data/unlocks/ladder.json`).

`unlockladder` (TASK-153; shared framework: per-profile persistence of open unlocks, trigger table as data with kinds codex_first, quest, level, tally_spend, retire_count, kills_of_kind, a reveal moment of log line plus silhouette, API gating; reused by the senses ladder), then TASK-154 to TASK-163 in the order of section 2: `unlockchronicle`, `unlockvoice`, `unlockquestlog`, `unlockidreveal`, `unlockbestiary`, `unlockrisk`, `unlockatlas`, `unlockorderqueue`, `unlockalerts`, `unlockhall`.
