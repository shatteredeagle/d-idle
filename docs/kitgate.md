# KITGATE: recovery without the kit stays near town

> status: shipped on branch agent/kitgate; knob `recoveryKitGate` (default 1). d2d-only in effect, shared code.

Finding (SOAKWATCH, livestream soak seed 5, d2d): one death at clvl 7 in Burial Grounds in "Recovery: farming gold", belt 0 potions, 0 portal scrolls, 35 gold; 49 x "no town portal available"; near-deaths in the Crypt and Mausoleum; a Cold Plains / Cave 1 border loop. Death rule not violated: a policy gap.

## Why the bot was there (code path)
- VERIFIED by reading: `Runner.target0` (goal.go) for a ReachLevel goal calls `recoveryArea` while recovery runs. When that returned no area (both safe fields cleared or banned after deaths, or `starveStep` found its first field cleared) the code FELL THROUGH to the normal planner, which picks from the quest ladder at the character's level: Burial Grounds, then the Crypt and Mausoleum. The recovery slice (`recoveryStep`, 40 min) then walked there with no scroll to get home. This is the probable route of the death and the near-deaths.
- VERIFIED by reading: `safeTier` and `recoveryXPTier` (broke.go, recoveryxp.go) include the Den, Cave 1 and Cave 2 (and the XP tier Dark Wood and Black Marsh); the only potion rule is "Den needs 2 potions". An empty belt with a portal scroll therefore still farmed the caves.
- VERIFIED by reading: nav `Route` priced every level the same, and `NearMouth` (mouth guard) is off by default, so a walk Blood Moor to Cold Plains or Cold Plains to Stony Field could pass a mouth; a seen crossing through Cave 1 beats the direct hint.
- UNVERIFIED (inferred from the logs, not replayed): that the 91 "stepping away from a cave mouth" and the 20 "left level 2 for 3 but ended in 8" lines are the same mouths; and that the first Burial Grounds entry was the fall-through rather than the quest step. The fix covers both causes.

## Fix
- `kitMissing`: no healing or rejuvenation potion in the belt (`Belt.HP+RV`, as `updateRecovery`) or no town portal scroll (`resInput.HasTP`); unknown belt is not missing. Same parts `resource.KitShortfall` prices.
- Gate on (knob, recovery running, Normal act 1, kit missing): `recoveryArea` keeps to `kitSafeTier` (Blood Moor, Cold Plains), `target0` returns no area instead of falling through (the leveling slice waits at home), and `syncKitGate` gives the navigator `dungeonMouths` (Den, Caves 1 and 2, Burial Grounds, Crypt, Mausoleum): `Route` adds a 30-edge cost to entering them and `NearMouth` answers while set.
- With the kit, or knob 0, nothing changes. Tests: TestKitGate*, TestRouteAvoidLevels.
- Not done (follow-up): the exit-gate choice inside one level (`findGate`) is unchanged; `penalizeExits` already prices mouth tiles. The border-loop breaker still fires for a mouth the walk cannot leave.
