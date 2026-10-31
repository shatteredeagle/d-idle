## ADDED Requirements

### Requirement: d2d belt keys drink by belt column
On the d2d combat input the keys "1" to "4" SHALL send `cmd belt <n-1>` (belt column 0 to 3) and count one potion per accepted drink; every other key MUST pass to the d2dinput controller unchanged, because the channel's key verb does not know the belt hotkeys. Source: card d2dcombat; internal/d2dcombat/input.go (Input.Key), internal/d2dctl/combat_verbs.go (Drink); test TestBeltKeysAndCasts.

#### Scenario: Key 2
- **WHEN** combat presses key "2"
- **THEN** the channel receives `cmd belt 1` and Stats().Potions is 1 (TestBeltKeysAndCasts)

### Requirement: A d2d cast targets the monster at the cell
For a hotkey that has a skill id, CastAt SHALL read `monsters` once and send `cmd skill <id> x y <unit>` with the nearest living monster within 1 cell of the target cell (subtile + 0.5, divided by 5), and with none send the cast at the ground cell without a unit. A hotkey with no skill id falls back to a key press then a right click. Source: card d2dcombat; internal/d2dcombat/input.go; tests TestBeltKeysAndCasts, TestUnmappedHotkeyFallsBack.

#### Scenario: Monster at the cell
- **WHEN** hotkey sk36 (skill 36) is cast at subtile (65, 147) where monster unit 5 stands
- **THEN** the lines are `monsters` then `cmd skill 36 13.1 29.5 5` and Stats().Casts[36] counts it

#### Scenario: Nobody there
- **WHEN** the same hotkey is cast at subtile (400, 400)
- **THEN** the lines are `monsters` then `cmd skill 36 80.1 80.1` with no unit

#### Scenario: Unmapped hotkey
- **WHEN** hotkey "f9" has no skill id
- **THEN** the first line sent is `key F9`

### Requirement: Combat skills are bound as pseudo hotkeys on d2d
BindSkills SHALL give every skill of the character that the combat catalog knows the pseudo hotkey `sk<id>`, install the mapping in the input (CastAt becomes `cmd skill <id>`) and in the Fighter, and return how many were bound, so no F-key or popup clicking is needed. UNVERIFIED by test: read from code (no unit test in internal/d2dcombat covers BindSkills). Source: card d2dcombat; internal/d2dcombat/rig.go.

#### Scenario: Two known skills
- **WHEN** the snapshot lists skill ids 36 and 40 and the catalog knows both
- **THEN** BindSkills returns 2 and the hotkeys are `sk36` and `sk40` (UNVERIFIED, no test)

### Requirement: The d2d snapshot gets a viewport for combat
The d2d combat reader SHALL fill Snapshot.Viewport from the player cell as VX = 16 (px - py) - 400 and VY = 8 (px + py) - 284 (px, py = position + 0.5), so a standing player projects to about pixel (400, 284); without it every target counts as off screen. That d2d draws its player at that pixel is UNVERIFIED. Source: card d2dcombat; internal/d2dcombat/viewport.go; test TestViewportPutsPlayerAtScreenAnchor.

#### Scenario: Player at (153, 68)
- **WHEN** the viewport is computed for a player at cell (153, 68)
- **THEN** the player projects to within one pixel of (400, 284)

### Requirement: The combat clock reads state after every sleep
The d2d combat clock (Meter) SHALL step the game through the Pacer on Sleep and then read `state` once, keeping Totals: Damage (sum of life drops), ManaSpent (sum of mana drops), Deaths (transitions to life 0), MinLife and Samples; a read that fails or reports max life 0 is not sampled. Only Samples and MinLife are covered by test; the other totals are UNVERIFIED offline. Source: card d2dcombat; internal/d2dcombat/meter.go; test TestMeterSamples.

#### Scenario: One sleep
- **WHEN** the Meter sleeps 80 ms once against the sample state
- **THEN** Totals shows Samples 1 and MinLife 45

#### Scenario: Reset after a cheat
- **WHEN** Reset is called and the next sample arrives
- **THEN** the jump from the old sample is not counted as damage (UNVERIFIED, no test)
