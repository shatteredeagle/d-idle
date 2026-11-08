# ptrclick: is d2d's click-targeting a deviation from game.exe, or only our harness? (TASK-179)

> status: plan, nothing run live. Tags: V verified (how stated), S read in code, U unverified. Card: `backlog/tasks/task-179 - ptrclick-*.md`. Found by the Paladin melee fix (`docs/palmelee.md`, `docs/palpace.md`).

## Question
On d2d a melee swing sent as the bare channel verb `click x y` acted on the unit under the pointer of the previous frame. Nothing moved the pointer onto the monster first, so swings missed. In game.exe the pointer moves to the click position on a click and the monster under the cursor is targeted at once (U, see below). Is d2d's behaviour a deviation (candidate upstream fix: refresh the hover unit at click time) or a property of how our input channel delivers the click (harness only)?

## What d2d does (S, read in the private fork at commit c36b77b1, a scratch clone of the private remote)
1. The `click` channel verb (`apps/d2d/devctl_verbs.hpp`, `register_game_verbs`, the `click_verb` lambda) pushes ONE motion event, then button down, then button up, all into the same SDL event queue; they land in the next frame's poll. The pointer teleports and the press happens in the same frame.
2. The left-click decision is `Town::input` (`apps/d2d/town.cpp`, around the `mouse.press_this_frame` branch): it reads `hovered_monster()`, which decodes `hovered_npc`; a live hovered monster gives `cmd::UseSkill{ skillbar.left, x, y, <that monster id>, true, ... }`, a hovered ground item gives `Pickup`, and with nothing hovered it gives `Move` to the clicked ground point (a walk, no attack).
3. `hovered_npc` is NOT computed from the click position. It is written by the renderer (`render_world` fills `hovered`, `ingame.cpp` copies it into `*hovered_npc`, `Town` passes `&hovered_npc` into the draw call) and the code comment at `ingame.cpp` says "last frame's". So the target of a press is the unit under the pointer as of the last drawn frame.
4. Consequence (S): a press in the same frame as the pointer jump uses a stale hover. A real mouse has motion events frames before the press (a human rests the cursor on the monster), so for a person it is the same as "unit under cursor". Only a teleport-and-press, which the `click` verb is, can be wrong. Existing evidence points the same way: `HoverClick` (`internal/d2dctl/town_verbs.go`) needs two moves and a pause before a click works on a vendor or menu (V 2026-10-30, `docs/d2d-fidelity.md` around the vendor flow: "a click without a preceding hover is ignored, the same as on Game.exe", cause U).
5. Our other injection path on game.exe (`apps/whisper/whisper.cpp`, `mouse()`) posts WM_MOUSEMOVE then the button messages to the window, and a "real click" (`SetCursorPos`, 60 ms, `SendInput`) exists because posted clicks do not do everything (an interact after a walk). That is the game.exe side of the channel and is also a harness property.

## Expected deviation class (U until the probe runs)
- Most likely HARNESS ONLY: the stale hover is a one-frame latency of a render-derived hover, invisible to any real mouse; the teleport click is a test-harness verb. If so the right fix is in the verb (queue the motion one frame before the press, or resolve the hover from the click point at press time in the verb path) or the documented rule "address monsters with `cmd skill ... unit`", not an upstream PR.
- DEVIATION (class: timing/latency of target selection, same family as the matrix rows for movement/input) only if game.exe resolves the unit under the cursor at click time from the click position inside a single frame, AND a plausible real input can produce press-in-the-same-frame-as-motion (touch, gamepad cursor, very fast mouse at low frame rate). Then the PR is a differential versus original game.exe code (the cursor-to-unit pick function behind `GetSelectedUnit`, RVA `0x67A10`, `docs/d2-1.14d-memory.md` 2.12) per `docs/process/upstream-pr-standard.md`, queued by the coordinator in `docs/research/pr-bundle-queue.md`.

## Probe plan (same save, needs a live game on the worker host; not run here)
Scenario (define in `internal/fidprobe/scenarios.json` style: id, name, role, class, purpose; suggested name `paladin-pointer-click`, role `pal-click`): a fresh Paladin (or any melee class) loaded into a pinned seed so the same monster stands at a known cell; first bring both runtimes to the same state with the same save (`docs/research/fidelity-probe-design.md`). One stationary monster (a Quill Rat or Fallen asleep, or a townsfolk NPC for the interact case) at 3 to 4 subtiles on screen, hero not moving.

Conditions, per runtime (game.exe and d2d), 5 repeats each:
- A: pointer elsewhere, then ONE `click x y` at the monster's screen position, no earlier motion.
- B: `move x y` (hover) onto the monster, wait 2 frames or 100 ms, then `click x y`.
- C: `move x y` and `click x y` in the same frame (game.exe: posted WM_MOUSEMOVE + button messages back to back; d2d: the `click` verb is exactly this).
- D (control): `cmd skill <id> x y <unit> left` on d2d (the bot's path, `Input.Swing`).

Record per run: (1) the targeted unit id: game.exe from a memory read of the selected/hovered unit after the click (`GetSelectedUnit` RVA `0x67A10` and `SetSelectedUnit_I` `0x66DE0` are named in `docs/d2-1.14d-memory.md` 2.12; the backing variable is not named for 1.14d: U, derive by disassembly, or read the attack target the player's current-action field holds, U) and `CursorHoverX/Y` RVA `0x321E4C/0x321E50` (V in D2BS); d2d from the `dump` verb or the UseSkill command log; (2) whether the monster's life dropped within 3 s; (3) whether the hero moved instead (Move fallback).

Reads: if game.exe hits in A and C and d2d misses in A and C but hits in B, it is the stale hover. Then decide by the class rule above. If game.exe also misses in A or C (cursor must have been seen by a frame), it is harness only on both sides and the answer is "a click needs a prior hover everywhere".

Writes: setup only (place the character, spawn nothing); no gameplay write after the run start; log `writes after start: 0`. Needs a hidden instance for game.exe; the user's go for any hook set.

## Evidence from the Paladin logs
- `docs/palmelee.md` ("Root cause", V): paladinsmoke run `bot-pa3.log` shows "combat no bound attack skill, swinging the basic attack" hundreds of times and "casts 15 life 100% engaged 22s" (15 clicks, the Quill Rat at 4 subtiles never lost life). Same pointer-lag cause named there; realtime had hit only now and then when the pointer happened to rest on a monster.
- `docs/paladinsmoke.md` (the old failure); fix drill table in `docs/palmelee.md`: clvl 9 and 1195 kills in 240 virtual minutes after the unit-addressed swing (V on the lab host).
- Bot test: `internal/combat/swing_test.go` (an input whose bare click is swallowed), `internal/d2dcombat/input_test.go` `TestSwingAddressesTheMonsterByUnit`.
- Unit-addressed casts of the Sorceress always worked for the same reason.

## Answer for the bot
Done, whichever way the probe goes: `Input.Swing` (`internal/d2dcombat/input.go`) sends `cmd skill <id> x y <unit> left`, so no melee swing depends on the pointer. Remaining bare `click` users on d2d are menu and vendor flows, which already hover first (`HoverClick`).

## Proposed matrix row (for the TASK-100 refresh to take over; the matrix is not edited here)
`| IN? | input | Left click picks its target from the previous frame's hover unit (render-derived), so a press in the same frame as a pointer jump acts on a stale unit; real mice are unaffected | click handler (Town::input, hover from render_world) vs game.exe pointer pick (GetSelectedUnit 0x67A10) | U (probe pending: harness only or DEV) | probe ptrclick (TASK-179) | bot: avoided (unit-addressed swing) | S1 | P2 |` (id, columns and priority to match the matrix at refresh time).

## Open questions
1. Does game.exe pick the unit from the click position at press time, or from a hover refreshed once per frame like d2d? (the probe's A and C)
2. Which game.exe variable holds the selected unit (not named for 1.14d in D2BS).
3. Gamepad/touch paths of d2d (SDL) may press without motion: out of scope for the bot, relevant for friend-readiness only if friends play with a controller.
4. Whether the fork's `click` verb should emit motion one frame before the press (then the bare `click` is safe everywhere; a fork change, same-day push rule applies; not done here).
