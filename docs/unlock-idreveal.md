# Identify reveal (TASK-157, unlockidreveal)

Fourth of the ten unlockables (`docs/research/unlockables.md`); sits on the ladder (`docs/unlock-ladder.md`). Tags: V verified (code read or offline test named), S secondary, U unverified (needs a live run).

## What the bot does with unidentified items (checked first)

- V (code read): the pickup rules (`internal/loot`) may give a provisional verdict `NeedID` ("identify first"); in town the chore `identify` (`internal/town/identify.go`, plan field `Identify`) identifies pack items with identify scrolls or a tome, buying scrolls when none are held. Cain is not used (his dialog is not mapped, `docs/town.md`). The whole flow is UNVERIFIED live (the file says so).
- V (code read): `loot.emitDrop` and `bot.emitDrop` send `goals.EvDrop` with the item name (the base item name; "item N" when unknown) and the quality read from the item unit (`loot.QualityName`: low, normal, superior, magic, set, rare, unique, crafted). U: that the quality is already set on an unidentified item in the game's item data (it is for magic and rare drops in the game's own display, where the name colour shows before identifying; not read live here).
- No event says "item identified", and no reader gives an item's affixes. Missing reader for the reveal's affix lines: item properties after identify (a Fake serves the test). Until it exists the card says "The properties are not read from the game yet."

## What it is

A face-down card appears for each pickup of a quality with a card. The viewer flips it (button in the sheet that opens, or in the "Cards" tab) and only then sees the quality word, the colour (the page's `--q-*` variables), a short sound (Web Audio, with a checkbox) and a line of text. Ladder rows:

| row | trigger | opens |
|---|---|---|
| `idreveal` | `item_quality` magic or rare (the first such pickup) | cards for magic and rare items |
| `idreveal_later` | `codex_first` `drop` `first_unique` or `first_set` | cards for set, unique and crafted items |

## Rules kept

- Do not reveal rarity colour before the reveal: the face-down payload holds only id, number and time. No quality, no colour, no item name (an identified name such as a unique's would give the rarity away). A test checks the raw JSON for `magic`, `q-magic` and the name. The Chronicle's drop lines (`docs/unlock-chronicle.md`) name no quality either.
- Spoilers: a unique, set or crafted card is stored but not shown, and its id cannot be flipped (404 naming the id), until the Codex has met the first unique or set.
- Never silent: opening writes the ladder's reveal line.

## Pieces

- `data/idreveal/reveal.json`: per quality a label, a colour class, a sound cue and 3 or more texts; the lists of first and later qualities. `ParseRevealTable` errors name the quality.
- `internal/watch/unlockidreveal.go`: `Reveal`, `GET /api/reveal` (schema `watch.reveal/1`), `POST /api/reveal/flip {id}` on the answer listener (same gate as the other unlock routes: 401, 403, 415, 400, 404 unknown or hidden card, 409 already flipped). `FunConfig.Affixes` (an `AffixSource`, nil today) feeds the property lines.
- Engine: trigger kind `item_quality` (arg `qualities`), event `EvItem`; `Unlocks.OnGoalEvent` sends the quality of every drop.
- Page: a "Cards" tab and a modal sheet (`web/unlockui.js`).

## Not done

- Cards live in memory: a restart drops unflipped cards (U). Replays inside the process are ignored by event `Seq`.
- Affix lines need the missing reader; the flip is by the viewer, not by the bot's identify chore (no event for it).

## Verified

V offline: `go test ./internal/watch ./internal/unlock` (golden payloads face down and flipped for Sorceress and Paladin, no leak while face down, uniques wait for the Codex first, probing refused, replay, token and method gates, unknown row and table errors, trigger validation). U: the quality field of unidentified items live; the page and sound in a browser.
