# retreatlive: RT.drink and RT.portal run live on d2d (2026-11-04)

Binary: lab fork build `d2d.lab-cheats3`, slot 3, lockstep, throwaway character, `d2drill -group RT -repeat 2` at branch tip of agent/retreatlive (same code as main c4d4d03b). Recipe: docs/probes/retreat/2026-11-04-RT-run.sh. Logs: docs/probes/retreat/2026-11-04-RT-drink.{txt,json}, 2026-11-04-RT-portal.{txt,json}. This is d2d, not game.exe (V: ran live).

## Results

| drill | verdict | runs | decision | ticks to act / effect | life floor | belt / scrolls |
|---|---|---|---|---|---|---|
| RT.portal | PASS | 2, identical | town-portal | 0 / 1 | 102 (start 102 of 300) | scrolls 1 to 0, portal listed |
| RT.drink | FAIL | 2, identical | chicken (want drink*) | 0 / none | 53 (start 60 of 300) | belt 4 to 4 |

Lockstep is deterministic: both variants gave identical signatures and numbers in both runs (V).

## Findings

1. RT.drink FAIL is a setup/bound error, not a bot bug (V: Safety source, internal/combat/safety.go decideBase). Setup sets life to 20 percent; `ChickenPct` is 25. At or below it, in a fight, with no rejuv in the belt, decideBase returns Chicken (the healing-potion branch needs `!InFight`). The drill's own spec says "decision drink*" so the spec contradicts the tune. Proposed bound: set RT.drink life to about 40 percent (above chicken 25, below DrinkLifePct 55, adjacent monster, heal potion ready gives drink-life; U until run). Not edited here.
2. Because the decision was chicken, Act(Chicken) ran with Fighter.Leave nil: the unverified esc-and-click placeholder was sent in the d2d rig. Effect: none seen, character stayed in game and in town afterwards (V: log "END ... in town: true"). RT.drink needs fixing before reruns, or a rerun sends the placeholder again.
3. RT.portal: the effect appears 1 tick after Act returns (portal object near, scroll count -1). Act itself steps the cast inside lockstep, so "1 tick" counts only the ticks after Act, not the whole cast (S, not measured). The 50 tick bound is very loose: proposal 10 (U). Life floor was the setup value (no hit landed in the window).
4. Life floor for drink shows 53 from 60: the adjacent zombie hit once (V).

## Writes

All writes are setup cheats before the Act, each ledgered with a reason by the drill (drink: goto, kill, stat 7, stat 6 x2, clearinv, spawn zombie1, goto; portal adds belt clear and give tsc). The d2drill path has no runstart marker (no `d2cheat runstart` call in cmd/d2drill, S: grep); writes after start: 0 by construction (retreat_lab.go only calls Tick, Snapshot, PView after the Act, S: code read; not checked against the ledger with --since-run).

## Not done

RT.leave (blocked, Fighter.Leave nil). RT.drink rerun at 40 percent needs a drill edit (owner m1-liveproof/retreatdrill).
