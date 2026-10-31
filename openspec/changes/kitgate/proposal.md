## Why

The SOAKWATCH observer on the livestream soak (seed 5, d2d) found one death at clvl 7 in Burial Grounds in the mode "Recovery: farming gold": belt 0 healing potions, 0 town portal scrolls, 35 gold, 13-16 Hungry Dead, plus 3 near-deaths in the Crypt and Mausoleum and a Cold Plains / Cave Level 1 border loop. The death rule was not violated; it is a policy gap. Root cause in docs/kitgate.md: when recovery found no eligible farm area, the goal fell through to the normal quest ladder, and the recovery area lists still held the Den and the caves.

## What Changes

- internal/bot/kitgate.go: kit missing (no healing potion in the belt or no town portal scroll), the safe list (Blood Moor, Cold Plains), the dungeon list (Den, Caves 1 and 2, Burial Grounds, Crypt, Mausoleum), the nav hand-off.
- internal/bot/broke.go, goal.go: recoveryArea keeps to the safe list while the gate is on; target0 returns no area (wait at home) instead of falling through to the quest ladder.
- internal/nav: SetAvoidLevels prices dungeon levels at a detour in Route and turns NearMouth on while set.
- Knob `recoveryKitGate`, default 1 (on); 0 restores the old choice.
- Scope: d2d-only in effect (the recovery phase and the mouth behaviour are the d2d backend's; the code is shared, Game.exe is frozen).
- Delta: bot-loop ADDED `Recovery without the kit stays near town`.
