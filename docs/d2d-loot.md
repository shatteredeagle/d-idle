# Loot on d2d (agent D2DLOOT, 2026-10-30)

The real `internal/loot` (pickit, pack room, left-behind, poor-mode gold policy) now runs on the d2d backend. Code: `internal/d2dloot`
(ground adapter), `cmd/d2bot/d2dloot.go` (Loot seam; one line in `d2dwire.go`). Tags: V = offline test named or live number, U = unverified.

## How it works

- Pickup: `loot.Looter.pickOne` walks (nav) to within `WalkDist`, calls `In.Interact(item)`; the d2d controller maps a type 4 target to `cmd pickup <unit>`
  (walks to the unit and takes it, gold too: V docs/d2d-probes.md). It then waits for the unit to leave the ground and proves arrival by the pack list or the gold stat
  (`arrived`). Retries: `PickTries` 3 per round, 2 rounds, then "failed before". Autoloot is off, so a pile is never taken by walking over it.
- `d2dloot.Source` replaces the looter's snapshot and stat readers (`Looter.UseSource`): one `d2dsrc.Source.Read` per call, then `Enrich` on the ground units:
  `Addr = ID`; a ground unit without a cell position is put at the player (the verb walks itself); gold amount parsed from the label `"<n> Gold"` (V label shape,
  docs/d2d-fidelity.md) and served as `StatGold` so the "small pile" rule works; quality guessed from the label against the base name (below).
  V `TestEnrich`, `TestGoldAmount`, `TestSnapshotReadError`.
- Seam: `d2dStack.lootSeam` attaches the source to the stack's own Looter (shared with the town judge and boss runs, so `LeftBehind` and recovery state are one)
  and hands the runner a `looterAdapter`. V `TestD2DLootSeam`.
- Everything that already called `Loot.PickUp` now works on d2d: after a clear (`goal.go`), task item pickups by name, and the death-spot gold sweep
  (`goldsweep.go` walks next to each pile, then `PickUp{Radius: 30}`, which is what takes it with autoloot off).
- Pack full: the looter's own room check (`hand.Layout` + occupancy from the dump's inventory items) marks an item "no room"; valuable ones raise `LeftBehind`, which the
  runner turns into the "loot: pack full" town trip (INVSTASH policy unchanged).

## Quality guess (U)

`ground` carries only a label. Rules: label equals the base name = normal; `Superior <base>` = superior; `Low Quality/Crude/Damaged/Cracked <base>` = low (these
prefixes are V visible, docs/d2d-fidelity.md); label contains the base name plus affixes = magic; label without the base name = rare (could be unique or set: U, the
pickit keeps both anyway); stackables and gold are normal. Item level and identified flag are unknown on the ground; rules that need them see 0.
Magic/rare labels were not seen live before this card: see the live section.

## Live results (private fork, d2d.lab2, lockstep, `--no-save`, `autoloot=off,autoparty=off`, throwaway Sorceress "Lootie" seed 7, 2026-10-30)

1. **Looter alone** (`D2D_BIN=... go test -run LiveLoot ./internal/d2dloot`, test gated, holds nothing itself; the caller holds `/tmp/d2d-lab.lock`): clvl 8 via `debug stat 12 8`,
   then Blood Moor, Cold Plains, Stony Field, Blood Moor again with `debug goto L` + `debug kill`, then `Looter.PickUp{Radius: 80}` x3 per level.
   V: 8 pickups, 0 failures, 0 stuck: 7 gold piles (+1..+9 each, purse 0 -> 32 = the sum of the pickup deltas) and 1 Scroll of Identify. Every pile and scroll left the ground and the
   purse rose by the pile amount (`arrived` proof worked). 30 skips, all by the real pickit: 27x "keep [C] potion already in the pack" (belt/pack full of potions), 3x Arrows
   "drop [C] plain or inferior item". Pack 7 -> 11 items, never full, left-behind 0 (pack-full path therefore only offline V, not live).
   Gold label parse and `StatGold` served; the "small pile" rule is skipped when poor (purse < PoorGoldBelow), so all piles were taken.
   No magic/rare/unique drop occurred: the quality guess for those labels is still U.
2. **Whole bot** (`d2bot --backend d2d --managed ... --goals campaign --run-for 9m`, binary with this card): only 4 kills in the Den of Evil (19 hostiles around a clvl 1 character), the
   clear call never returned to the pickup step ("no progress in state clearing for 60 s", area skipped as nav-stuck, then the failsafe restarted the game twice): so the runner never reached
   `Loot.PickUp` and gold stayed 0 (`recovery: loot +0 items, +0 gold`). That is the combat/pace problem of D2DWIRE (clvl 1 Sorceress in Den), not loot: with the Loot seam present the
   runner calls the same `PickUp` the live test proved. Open: re-run the whole bot once a clvl 1 character survives the Den (D2DPACE).
