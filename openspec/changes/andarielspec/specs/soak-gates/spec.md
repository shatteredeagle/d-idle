## MODIFIED Requirements

### Requirement: G2 meaning and floors
The 8 h profile m24 run (Gate 2 on d2d, 24 h Gate 3) SHALL be called a stability and economy soak, not a progression soak, until Andariel and Act 2 work; it SHALL reach clvl 16, meet the Andariel criterion at clvl 14 or higher, hire the merc and keep gold from staying flat. Until boss.andarielHold is lifted, reaching the Andariel gate at clvl 14 or higher meets the Andariel criterion (evidence in the scenarios; planner "why" strings never count); after the lift an actual attempt is required. Source: plan-v2 section 5 and 10; ADR 0013; docs/andarielgate.md; user decision 2026-11-04.

#### Scenario: Cheated run
- **WHEN** a run used cheated gold or cheated levels
- **THEN** it does not count toward any gate

#### Scenario: Interim evidence
- **WHEN** boss.andarielHold is set
- **THEN** the criterion needs a `boss: andariel gate pass` or `boss: held for andariel (...)` line naming the hold, plus `route guard: andariel floor=` lines showing floor and merc state

#### Scenario: Held with merc alive, floor reached
- **WHEN** boss.andarielHold is set, the merc is alive, and the log has `route guard: andariel floor=` lines and a `boss: held for andariel (...)` line naming the hold at clvl >= floor-1 and >= 14
- **THEN** the Andariel criterion is met

#### Scenario: Held with merc dead
- **WHEN** boss.andarielHold is set and the merc is dead so the floor is 17, and the log shows the `route guard: andariel floor=17` lines with the merc state
- **THEN** the evidence shows why the gate was not passed, and the criterion is not met unless the character reached clvl 17 and the gate line

#### Scenario: Planner text only
- **WHEN** the only Andariel mention is a planner "why" string, with no `boss: andariel gate pass` or `boss: held for andariel (...)` line
- **THEN** the criterion is not met

#### Scenario: Hold lifted
- **WHEN** boss.andarielHold is lifted after m3-andariel has won once
- **THEN** only a boss-run start line and an outcome line at clvl 14 or higher meet the criterion
