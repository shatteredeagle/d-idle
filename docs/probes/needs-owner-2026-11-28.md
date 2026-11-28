# Needs owner (2026-11-28 night)

- fix1 round-2 review (A): `VaultDirectEquip` defaults to 1 and Carry/Stow are exposed; review recommends setting it to 0 and hiding Carry/Stow until N1-N3 (stale unit ids after a game restart; open-order blocks only a second Equip) are fixed. Not done unattended: it removes the Equip you use. A fixer (fix1b, Fable) is fixing N1-N3/N5/B2 tonight; decide in the morning if you want the interim switch-off.
- Questy Cain reward (live1, L4): the game pays a quest reward on first contact with the reward NPC, and the act 1 potion seller counts. Blocking passing claims means no potion buying until the owner claims the reward. Decision needed before the bot (not the user) is allowed to reach Cain's reward on Questy; for now the hold only covers a running board chunk.
- Doofer repair mode (live1, L8): gear hit 0% durability; the hero's repair mode, not the thresholds, decides (docs/repair.md). Mode was not read: check `ops status` on Doofer.
- Tick-line criteria (evid1): ticklog-t4's seconds-based count rule can never hold (ticks come once per goal-chain pass, median gap 16 s). Proposal: ticks at least twice the minute lines, no gap longer than the longest action timeout. Owner row: not edited.

## !! Questy character save lost (2026-11-28 04:08 EST) !!
The worker host root disk filled (go build cache 155 GB, many agents). The d2d fork writes the character save in place, so `Questy.d2s` in the d2play-20261125q container was truncated to 0 bytes while it was being saved. Only the Nov 25 level-1 `.d2s.bak` is older; there was no other backup (not in git, not on the NAS). The sandbox bot has been crash-looping since (`d2d: not in game`, supervisor restarts every 2 min).
- A copy of the broken save directory is at `~/d2play-q-rescue-20261128/save` (0-byte d2s, the .bak, key, map, ma0). The bot state (`/data/instances/main/state`: quests.json etc) and vault (`/data/vaults/Questy`) are intact in the volume.
- I did NOT stop the container or delete anything (a stop was blocked as destructive). Doofer's save is fine.
- Decision needed from you: (a) restore the Nov 25 level-1 `.bak` and let the bot level again, (b) create a fresh Questy via the OOG tooling and replay quest state, or (c) leave it. Your Search for Cain progress on Questy is gone with the character.
- Prevention: fix agent `savesafe` (rolling backups, zero-byte restore, disk guard) running; fork request for atomic save in docs/design/fork-save-atomic.md.
- Owner question (forkreq): atomic-save request A1: if the fork switch is days away, cherry-pick A1 onto the fork round the sandboxes run today? (fork changes need your go)
