# NPC interaction (town/npc.go, cmd/d2bot/task.go)

## Flow
1. `openNPC`: find the NPC in the monster table (`findNPC`, nearest of the classes), walk (`approach`: GoToUnit, preset fallback),
   `hand.InteractNPC` (hover-verified click, plain world-click fallback), wait for UI flag `UINPCMenu`, skip a first-contact greeting
   (`town.greetingSkip`, default re-click the NPC, bounded to about 6 s per try, see docs/npc-skip.md; `GreetingWaitS` only in mode 0). Up to `NPCOpenTries` rounds.
2. `pickEntry`: the dialog is a bottom-anchored box at FIXED pixel offsets from the NPC feet (`NPCMenu`, V live 800x600 on
   Gheed/Akara). The line count is not readable from memory (U: no flag), so the click position is a guess corrected by `menuAdj`.
   Proof of success is the target panel flag (`UIShop`), never the click itself.
3. Vendor: sell/buy in shop.go (unchanged). Close: `CloseAll` (Esc).
4. Quest talks (`cmd/d2bot/task.go` dialogAdapter): `quest.PickPath` over `quest.Menus` rows; `Talk` leaf = speech, closes the dialog.

## What went wrong live (G1 attempt 14, i4, log 20261029-235659, V from the log)
- `talk-charsi` (Optional): Talk clicked, dialog closed by the speech, `Do` returned nil, but the quest word did not change, so the
  executor counted "ran but is still needed" -> retry -> same again -> skipped (25 s, two full walk/open/Esc cycles). Logged
  `err:null` because retries carried no error.
- `chores` blocked: NOT the NPC. The belt chore failed ("drag of Minor Healing Potion onto the belt did not land", 3 times, ~25 s each)
  while the character stood still in town (`stuckFor` 75 s); the step then ended "ran but is still needed (evidence 2)" and blocked the group.
  Belt-drag failure root cause is U (not investigated here: not an NPC problem).
- The restart (`save-and-exit` 00:02:17) happened in the Black Marsh during combat (hostileNear 17, life 88/104), not in town; unrelated to NPC code.
- 00:30 `sell` at Charsi (class 154): "dialog entry 1 not reached ... flag 0xc never became true": 8 pixel-nudge clicks. A click one line
  off the real line count closes the dialog (it hits Talk/Cancel/another row), the old loop reopened and re-clicked 5, -5, 9, -9 px away
  (the same wrong line) before reaching -15. This matches the user's "opened and closed it, then picked the same option a few times".

## Changes
- `pickEntry` re-observes before every click (menu open? target window already open? NPC position refreshed) and logs it. After a click
  that CLOSED the dialog it jumps to line-sized adjustments (-15, +15, -30); after one that left it open it nudges (+-5, +-9). More than
  4 closes stops with a clear error. The adjustment that worked is remembered per class (as before).
- Executor: a clean `Do` of an Optional step that is still needed is skipped at once; retries and skips now carry the reason in `err`.
- Logs (key `msg`): `npc: interact` (class, dist, pos), `npc: menu open` (ms), `npc: menu click` (entry, line, lines, adj, at, npc),
  `npc: no window after click` (menuStillOpen, ui), `npc: menu closed by the last click, reopening`, `npc: window open` (adj, tries, ms),
  `npc visit: vendor window open` (ms), `npc visit failed` (stage), `npc: talk` / `npc visit: talk done` (path, ms, err).
- Not added: the per-visit "sold N items for G gold" summary line (shop.go logs `shop: visit` for buys); a sell summary is U/todo.

## Menus per NPC (what the code assumes)
Gheed Talk/Trade/Gamble/Cancel (V), Akara Talk/Trade/Cancel, +Reset line after the Den reward (V), default Talk/Trade/Cancel.
Charsi: Talk, Trade/Repair, Cancel; Imbue line after Tools of the Trade (U). Kashya "Hire" row (U).
The quest Cond `Imbue` window (Started..Done) is U and not used for click placement by town.

## Tests (offline, sim)
`internal/town/npcmenu_test.go`: extra menu line, speech before the menu, bounded closes, wandering NPC. Town tests need /tmp/d2mpq (labrun worker).

## LIVE CHECKLIST (d2-live agent, instance i4 or own, Rogue Encampment, offline char, no cheats needed)
1. Stand near Charsi, run the sell chore. Expect in log: `npc: interact class=154`, `npc: menu open`, `npc: menu click ... adj=0`,
   `npc: window open ... tries=1` and `npc visit: vendor window open`. Note `lines` and `adj`.
2. Repeat with Tools of the Trade started/done (4th line): expect `closed by the last click` once then `window open adj=-15`. Record the real line list with a `tools/shot.sh` shot of the open box.
3. Akara after the Den reward (4 lines): same pattern, adj remembered.
4. `talk-charsi` step: expect `npc: talk`, `npc visit: talk done`, then the step `skipped` after ONE run with `err` "ran but is still needed".
5. Walk Charsi away is not needed; confirm she stays within a subtile or two (the click follows `npc:` position).
6. Unverified: Charsi's real line count and Trade/Repair label, whether Talk to Charsi starts the quest word, belt drag failure cause.

## Hover misses on first contact (NPCHOVER, V from the i5 Fresha log 20261030, G2 log for frequency)
- i5 10:00 Akara (148) and 09:59 Charsi (154): try 1 hovered and clicked (log `npc: interact`), the greeting held the menu back, `greeting never gave way` after 13 s, then every further click failed `never hovered` (3 tries x ~4 s). NOT a stale position: the unit position was unchanged across the failed tries; hover is dead while the first-contact speech plays. Both NPCs worked on a later trip.
- G2 log (294 NPC clicks): 61 `greeting never gave way`, only 1 NPC hover failure (Gheed, i4 14:46), so the miss is first-contact specific.
- `panel open before interact` at 10:07:12 with ui[12] (UIShop) set: Charsi's trade window was still open when the next chore moved to Akara; closing it is correct.
- Fix: `interactFresh` re-reads the unit position before every click and, after a hover miss, retries with a changed position (`npc: hover missed, re-reading position old=.. new=..`); `InteractNPC` gives up after 3 s; a hover miss inside the greeting loop no longer ends the try (`npc: greeting still blocks hover, waiting`); greeting slice 6 s -> 4 s. U live: whether the visit now ends with the menu once the speech is over.

## Trade entry failures, 2026-10-30 (NPCTRADE; V from G2 logs soak24 20261030-* and i5 Fresha 20261030-095519/105216, read-only copies)
- Counts: 223 first menu clicks (Akara/Charsi/Gheed). 198 opened the window at the modelled spot (Akara 93, Charsi 104 all at adj 0, p95 303 ms, max 3.1 s). So the Game.exe `entryPoint` model (bottom-anchored, Cancel -115, step 31/2) is right; the 41 px d2d offset does not apply.
- `npc visit failed` stage trade: G2 15 (Akara 8, Charsi 7, Gheed 1 deadline), Fresha 3 (+1 Gheed deadline). Stage open: G2 12, Fresha 5 (hover dead during greeting, covered by NPCHOVER; walk errors).
- Every failed visit has the same shape: first click at adj 0 either leaves the dialog open or closes it, and then ALL later clicks (adj +5 -5 +9 -9 -15 +15 -30, i.e. 45 px, every line) leave flag 0x8 set with the UI array unchanged for ~2.8 s each. No click ever hit Talk (which would close the dialog), so it is not a line-count or quest-line problem: lines=3 and the model is right; the dialog ignores clicks (U why: input not registering, or the flag is a ghost). first-contact was false in all Fresha failures, so it is not the greeting. Failures cluster per trip (10:57:58, then 11:03:47 to 11:04:48 across Charsi, Gheed, Akara).
- Gheed (4 lines): 7 of 9 first clicks at adj 0 missed; adj +5 worked 7 of 7 afterwards. Now the first adjustment.
- Cost was 8 clicks x 2.8 s = 30 to 45 s per failed visit.
- ui flag 0xc is UIShop (the proof flag), not a wrong panel; flag 0x8 = NPC menu; index 35 (0x23) is always set.

### Fix (town/npc.go pickEntry)
- Proof wait per click 1.5 s (was the 2.8 s ui timeout). A click after which the dialog is open and the UI array is unchanged counts as ignored; the 2nd and 3rd click hover slowly (move 3 px off, 150 ms, onto the line, 450 ms, click, as PickLine). After 3 ignored clicks: Esc and fail. Closes bounded to 2, whole pick to 9 s (checked between clicks).
- Log: `npc: trade entry not reached because <reason>` with class, ui, lines, clicks, closes, ignored, ms; also `slow` and `ignored` on `npc: menu click` / `npc: no window after click`.
- Sim: `World.MenuInert` (clicks ignored). Tests in npcmenu_test.go (need /tmp/d2mpq: run via labrun worker).

### Watch live
`npc: trade entry not reached because the dialog ignored 3 clicks` (expect visit cost under 10 s), `npc: menu click ... slow=true` followed by `npc: window open ... tries=2` (slow hover recovers it: then the cause is the click timing). If failures persist with ignored clicks, capture `tools/shot.sh` at the failure to see whether the dialog is drawn at all.
