## ADDED Requirements

### Requirement: Replicated soak is a fidelity check
The replicated soak SHALL NOT gate making d2d the primary backend; it MUST be used to find d2d gaps to fix and to fire the exit trigger, and only the four gated metrics (XP/h, kills/h, deaths/h, deaths per 1000 kills) are held to the 20 percent tolerance. Source: ADR 0009 items 2 and 3.

#### Scenario: Price gap outside the four metrics
- **WHEN** a vendor price differs on d2d but the four metrics are in tolerance
- **THEN** it is an explained known gap, fixed on the fork when cheap

### Requirement: Soak FAIL attribution
A replicated-soak FAIL SHALL count against d2d only when the same bot build's original-game baseline is inside the gate and the gap is traced to an engine difference; otherwise it MUST be treated as a bot bug and fixed in the bot. Source: ADR 0009 attribution rule.

#### Scenario: Baseline outside the gate
- **WHEN** the original-game baseline for the bot build is itself outside the gate
- **THEN** the FAIL is a bot bug, not an exit-trigger input
