# kitstall: baseline `kit` lane i2 stall (Sorceress, clvl 3)

Source: bot log of lane i2 (run 20261102-111844), read-only. Tags: V = read in the log, S = code reading, U = unverified.

## Timeline (V)
- 11:19 to 11:34: normal Blood Moor farming, clvl 3, gold 250 after a sell trip. At 11:34:46 the sell chore finished and the equipment dump shows only the armor: the starting weapon and the gloves are gone from the equipment list (how: U).
- 11:35:57 a low-quality Short Bow is picked up; 11:36:05 "combat weapon problem ... launcher without quiver, weapon sbw" and from then on "no bound attack skill, swinging the basic attack": the Sorceress stands in melee with a bow, kills nothing, chickens at 50 percent (V). How the bow reached the weapon slot is U: the leveler excludes bows and low items (S, leveler/score.go), the looter wears armor only (S, loot/wear.go).
- 11:38:28 and 11:48:19 the ammo chore buys Arrows for 250 and 170 gold, all the gold, then "poverty mode: on" (V). Cause (S): `AmmoPlan` -> `gear.Restock` for any class wearing a launcher. At 11:45:17 the same chore fails "gold did not change" because gold was 0 (V), not a vendor click problem.
- Nav loop (V): from the vendor side of the camp (x about 5313) the nearest Blood Moor gate is {5342 6280}; the walk stalls at {5339 6300} (6 rungs, then the town-portal escape), 11:40, 11:43, 11:48. After the escape the retry uses {5272 6280} and works. Why the planner path to {5342 6280} is blocked (ghost obstacle or a collision-map error) is U.
- Why it repeated (S): the stalled gate was remembered for 2 min only, and `learn(1,2)` after the successful crossing at 5272 deleted all bad-gate marks of that edge.

## Fix
nav: dead gates (`markDeadGate`, 30 min, kept by `learn`, no level binding). gear: caster bow is no weapon problem. town: ammo for Amazon only. Tests: nav `TestDeadGateSurvivesTTLAndLearn`, gear `TestWeaponProblem` cases, town `TestAmmoPlan`.

## Still open (U) and a read-only probe
1. Who equipped the bow and removed the weapon and gloves between 11:34:46 and 11:36:05. Probe: with the lane stopped by its owner, dump the equipment list per snapshot (`inv-tp-dump` lines already carry it) around chore "sell" and "loot" in a fresh run; watch the cursor item after a pickup.
2. The blocked cell near {5341 6300} in the Rogue Encampment: log the collision cells around it at the next stall.
