# Potions: how the bot drinks (card potiondrink, TASK-328)

Tags: V read / V run / D from `docs/probes/fid/fid-potions.md` (game.exe disassembly and misc.txt); I = inference; U = unverified live.

## Game rule (V read, D)
- A healing potion heals over time: hp1..hp5 give 30, 60, 100, 180, 320 life over 192, 160, 171, 192, 256 ticks of 40 ms (7.7 to 10.2 s), times a class factor (Amazon, Paladin, Assassin 1.5; Sorceress, Necromancer, Druid 1.0; Barbarian 2.0; V run).
- A second drink joins the running effect: rate = (old x left + amount) / (left + len), duration left + len. A same-tier drink therefore extends the effect and does not heal faster; only a higher tier raises the rate. Rejuvenation is instant and does not touch the effect.
- No server cooldown (V read), so the limit is the bot's own.

## Bot rule (`internal/combat/potioneffect.go`)
- The Fighter keeps the working healing effect (tier, end, life per second); `Pending` is the life still to come.
- While an effect works another healing potion is drunk only if life is below `combat.potionEmergencyPct` (30), or the belt holds a higher tier (the lowest tier above the working one is taken). Otherwise the bot waits the effect out. Rejuvenation keeps its own lines and is unaffected, `combat.potionGapMs` still applies.
- Belt column selection is unchanged (`Belt.ColumnFor`: lowest tier for a scratch, best below `combat.drinkRejuvPct`); `Belt.ColumnAbove` is the added query.
- Per-class data: the class factor table in potioneffect.go; an unknown class gets the duration but no rate (no guess).
- Not modelled: the vitality double-fill roll (about (vit-1)/4 percent, U for the lab). Tests: `internal/combat/potioneffect_test.go` (Sorceress and Paladin).
