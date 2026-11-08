# Quest hints in town (TASK-197)

> status: built and tested offline (fixtures, no game). Tags: V verified by the tests named, SRC read in the repo, EXP veteran memory (d2-sme answer of 2026-11-08), UNSURE the veteran was not sure, U not run live. The live drill is TASK-198.

## What it does

When the hero enters the Act 1 town (Rogue Encampment, area 1) it looks at the quest givers, like a player glancing around the camp:

- A quest the hero has not been given, that is available (see the table) and whose level for the hero's class family is reached: the hero says a `quest.hint.go` line, once per quest ever (persisted). The quest becomes the "errand" (`QuestHints.Errand()`, read only).
- Below that level: a `quest.hint.weak` line, at most one per town visit over all quests, and a quest speaks again only after the hero's level rose by 2 since its last weak line (persisted: a restart repeats nothing).
- It only speaks. It never changes which quest the campaign plan works on, and it never walks to an NPC (V: no import of the planner; the hint desk has no reference to goals other than the event stream).
- Class family (melee, caster, summoner) comes from the table (`families`); an unknown class is an error naming it (`Fun.Errors()`), the hero says nothing for that visit (V `TestHintUnknownClassNamesIt`).

Where the line appears: the feed (the hero's idle voice, skipped while the voice is muted), the Chronicle as a `thought` line when the Chronicle is open, and the return screen section "What I noticed in town" (lines since the player left, hidden when empty). No new ladder row: hints are available from the start, there is nothing to reveal.

## The policy row `questhints` ("How I handle quests in town")

| Option | Effect |
|---|---|
| Quiet (row default) | nothing is said or kept; today's behaviour |
| Hints | the lines above |
| Hints and ask me | the lines, plus two question cards: `t1.quest.imbue` ("Charsi can imbue one item: which kind?" weapon, body armor, circlet or belt) when Charsi's go line fires, and `t1.quest.cain` ("Search for Cain: do the whole chain or leave it for later?") at clvl 12 or the quest's own level, whichever is higher |

Defaults: every class starts on Hints (`start` in `chores-defaults.json`); a named run (`D2BOT_RUN`) starts no one on a start option (`PolicySheet.SetStarts`, as for stance), so gates and soaks stay Quiet (V `TestFunQuestHintsWiring` for the Paladin start; the named-run rule is the existing stance code, not retested here). The choice is a `policy` line of the session record and a before/after card on the return screen like every row.

Question keys: `t1.quest.imbue|act=1,diff=normal,hc=normal,quest=tools` and `t1.quest.cain|act=1,diff=normal,hc=normal,quest=cain`, built from the catalogue, so asking again while a card is open opens nothing new (V `TestHintQuestionKeyIsStable`). Each is asked once per hero (persisted `asked`). The answers are recorded; NO bot path reads them yet (gap, listed in `default_rules.json`). The catalogue knows only the paladin and sorceress classes, so for other classes the ask fails with an error naming the class (shown in `Fun.Errors()`) and is not retried.

## The table (`data/quests/act1-hints.json`)

Level is the hero's own readiness (the engine has no level requirement for any Act 1 quest, EXP). Fields carry a tag each in the file; the tags below are per row in the order giver / how / level / available / lines.

| Quest | Giver | How given | clvl melee / caster / summoner | Available when | Tags |
|---|---|---|---|---|---|
| den_of_evil | Akara | npc_dialog (first talk; stage word at once, SRC `docs/questmem.md`) | 1 / 2 / 3 | always | SRC SRC EXP SRC EXP |
| sisters_burial | Kashya | npc_dialog | 7 / 8 / 9 | Cold Plains visited | EXP EXP EXP UNSURE EXP |
| search_for_cain | Akara (finishes it) | auto_on_area (the tree in the Dark Wood) | 9 / 11 / 12 | Stony Field visited; ask at 12 | SRC EXP EXP UNSURE EXP |
| tools_of_the_trade | Charsi | npc_dialog | 12 / 14 / 15 | Outer Cloister visited; Imbue asked at go | EXP EXP EXP UNSURE EXP |
| forgotten_tower | none | auto_on_kill (the Countess) | 13 / 15 / 16 | never spoken (no NPC) | EXP EXP EXP SRC SRC |
| sisters_to_slaughter | Warriv | auto_on_kill (Andariel) | 16 / 18 / 18 | Kashya's quest done (stage 3) | EXP EXP EXP UNSURE EXP |

Class families: paladin melee, barbarian melee, amazon melee (UNSURE: a bow Amazon fights at range), sorceress caster, assassin caster (UNSURE: trap and fire builds), druid caster (UNSURE), necromancer summoner. There is no separate class traits table in the repo; the families live in this file so each is data and tagged. Every threshold is a veteran estimate (EXP) and should be checked against soak data.

## UNVERIFIED: the live drill checklist (TASK-198)

Read quest words before and after each step; never write "given" unless the word rose.

1. Does Esc skip every first-talk speech (Akara, Kashya, Charsi, Warriv)?
2. Does Charsi offer Tools of the Trade before the Den is done?
3. Does Kashya's topic list show Burial Grounds before the first talk with her?
4. Does Akara's re-talk after the Den give the skill point and drop the Den topic?
5. Is Treehead Woodfist required to be killed before the Tree of Inifuss can be clicked, or is the tree clickable at once (the repo says yes, the veteran thinks no)?
6. Is the Scroll of Inifuss / Cairn key recoverable if dropped or sold?
7. Does Imbue ask for confirmation before it consumes the Malus and the chosen item?
8. Which areas are really "available" for each giver (the `available` column is UNSURE for four quests)?
9. Do the area events carry `AreaID` on a real run (the hint matches either the id or the name, so one is enough; U which one arrives)?

## Not done / dead ends

- The hint does not queue an errand for the planner: the brief says the campaign keeps choosing quests. `Errand()` is only information.
- The Chronicle line and the voice-mute skip are not covered by a test (the Chronicle needs the ladder rig); the feed and the return section are (V `TestFunQuestHintsWiring`).
- No live run: the quest events (`EvQuest`) are emitted by the bot's quest reader, which has not run against a game (`docs/first-session.md` section C).
