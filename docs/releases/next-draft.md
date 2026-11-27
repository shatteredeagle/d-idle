# Friends release notes: draft for the next cut

> status: DRAFT. The version number is not chosen (the owner picks it at cut time). Covers what merged on main after the 0.1.3 cut (card cut013, commit `e5b0463ca`). Nothing here is claimed verified live: the merged cards' `result:` lines say what was run, and most of it is offline tests. Check each card before you promise a friend anything. Features that ship switched off are listed apart, as "not in this release".

## What changed for a friend (plain words)

- **Vault is faster.** Opening and scrolling the Vault page no longer recomputes everything on each refresh; the type counts come with every reload. "Sell all" writes a whole batch in one go instead of item by item.
- **Buttons show they are working.** A pressed button on the watch page now shows a busy state until the answer comes back.
- **Mercenary equip from the Vault.** An "Equip on merc" button puts a Vault item on your mercenary; the old piece goes into the Vault.
- **Charms: Carry and Stow.** Move a charm from the Vault into the backpack and back. A charm you marked "keep" stays in the pack when the hero tidies up. Big charms are accepted by the Vault preset.
- **Akara hand-in fix.** The quest hand-in that silently did nothing (for example Search for Cain) now logs the tap, drops a stale request and tells you the wait reason on the page.
- **Return screen.** A tour and Help, plus advice text wiring (the advice itself is off, see below).
- **Tester feedback bundle.** One command writes a scrubbed report file; nothing is sent. See "What to send us".
- **Classes.** Necromancer: a summoner profile, Bone Spear for the leveling table, hotkeys that match the layout, and an early Rogue hire. Druid (tornado), Barbarian (war cry), Amazon (javelin) and Paladin (zeal) got profile and hotkey fixes; Paladin quest points are spent when granted.
- **Town.** A short-gold priority table decides what to buy first when gold is short, with a hand-over to sell, stash and Vault. The staged town plan is still opt-in (below).

## Not in this release (off by default; do not promise these)

- Advice lines on the Return screen (`items.advisor`, 0).
- The per-class ready-for-next-difficulty check (`bot.diffReadyV2`, 0): the old level-only check stays. Difficulty readiness is opt-in.
- The staged town plan (`town.plan`, 0 = old town code) and the buff upkeep driver (`combat.deliveryUpkeep`, 0): on trial only in the lab.
- The hotkey map from the planner (`input.hotkeyMap`, 0).

Defaults are in docs/knobs.md (generated from code).

## Known broken (from docs/first-session.md section H, state as of 2026-11-27)

- Fresh-start leveling is not proven: no uncheated realtime hour has reached the next campaign step (TASK-138).
- Gold recovery is shorter with the tester preset, but the Barbarian was not re-run (TASK-183).
- The minimap was built but never seen on a real level (TASK-148).
- The all-mana belt is tested offline only (TASK-164, TASK-266).
- Unlock alerts show in the page only; no phone or chat pings (TASK-162).
- No published image yet (TASK-120).
- No Pause, Stop or Reset-all buttons or feedback box on some pages (TASK-47, TASK-49): stop with `docker compose stop`.
- Only Paladin and Sorceress ever ran in a real container; a first run of any other class is new ground.

## What to send us

Run the bundle script, `deploy/d2d-container/tester-report.sh` (steps in docs/first-session.md section J): copy it into the running container, run it, copy the `.tar.gz` out, look inside, and send that one file plus three or four sentences (class, how long you played, what felt wrong, the clock time of any problem). It removes tokens, keys, home folders and private addresses and sends nothing itself.

## Still to do at cut time

Choose the version, run the legal checklist, build, sign, drill and stage (the cut card owns that), and re-read this list against main.
