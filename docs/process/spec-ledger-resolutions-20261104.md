# Spec ledger resolutions, 2026-11-04

Resolutions for the 3 CONFLICT and 5 UNSURE rows of `docs/process/spec-ledger-20261104.md`. Proposals only: nothing under `openspec/specs/` was edited here. Tags: V = read in this worktree (file cited), S = secondary, U = unverified.

Finding (V): the base specs in this worktree are already past the ledger. They carry "Reconciled 2026-11-04" lines and a "user decision 2026-11-04" note (soak-gates:148, fidelity-gate:8, death-rule:6, backend-contract:64, merc-policy:25,36). So for most rows the proposal below is "confirm what the base now says", and only the cited gaps remain.

## CONFLICT

1. fidelity-gate "Death budget" (gatedelta, archived). Proposal: mark DROPPED, superseded by death-rule. (V) `openspec/specs/death-rule/spec.md:6` says it is the only authority on soak deaths and that the gatedelta fixed death budget and Poisson cap were dropped; `openspec/specs/fidelity-gate/spec.md:8` repeats it; ADR 0005 title: deaths are judged by streak, recovery share and rate, not by a fixed count. The deaths/h metrics stay only in the A:B comparison at 20 percent (fidelity-gate:13). No spec edit needed; only the ledger verdict changes from CONFLICT to DROPPED. Residual (U): gatedelta cited a user decision of 2026-10-30; the 2026-11-04 decision overrides it, but the user should confirm the dated override is theirs.

2. fidelity-gate "Death rate cap and Poisson bound" (gatedelta). Proposal: DROPPED, same reason. (V) fidelity-gate:8 names the Poisson cap as dropped; the 0.15/h lower-bound cap has no live reference. The Poisson code in `internal/d2dsoakstat/gatev2_test.go` (S, per the delta text) may be dead; a follow-up could check and delete it (U).

3. soak-gates "Gate 1 deaths and progression" (baselinefree vs specdelta vs base). Proposal: keep the base text as it stands. (V) soak-gates:147-148 now says: death-rule is the only authority; no state longer than 10 virtual minutes without a kill or XP gain; XP/h is never a per-run pass or fail (ADR 0015 item 8); the three band requirements are kept but ADVISORY only (soak-gates:158 "Gate 1 XP per clvl band (advisory)"). That resolves both the baselinefree FAIL clause and the orphaned band rows. Remaining (S): the band rows (rows 25-27) should be marked APPLIED-as-advisory, not NOT APPLIED. The band tolerance (1.96, CV floor 0.14, `internal/d2dsoakstat/bands.go:122-123`, V per the ledger) is only informational now; confirm that the user wants it kept as an advisory number (their decision 2 is cited, not seen by me: U).

## UNSURE

4-6. fidelity-gate "Proxy metrics", "Proxy definitions", "Missing proxy data" (gatedelta, rows 4 to 6). Proposal: DROPPED with the Death budget. (V) fidelity-gate:8 says the three proxy rules were dropped with the death budget. Reason (V, delta text): the 25 percent proxy tolerance and the INCONCLUSIVE-on-missing rule belonged to gate v2, retired by the gate ladder (soak-gates "Gate ladder on d2d"); the delta itself marks `dmgTaken` and `hpMinPct` as not emitted (U then). The code now has `dmgTaken`/`hpMinPct` in `internal/bot/hpfields.go` and `internal/d2dsoakstat/minute.go` (V by grep, file names only), so the data exist; if the user wants a proxy in the A:B report it is a new, separate card, not a revival of these rows.

7. backend-contract "Leaving d2d is break glass" (d2d-default-backend, row 38). Proposal: DROP the delta; the base already has the replacement. (V) backend-contract:63-66 "Leaving d2d is the user's decision": the user decides if the A:B verdict is DIFFERENT and the gap needs an engine rewrite; never automatic; ADR 0015 "Leaving d2d" (line 66-68) replaces the ADR 0009 trigger, so the 16-agent-hour clause is gone on purpose. Note (V): backend-contract:391 says the user dropped and archived d2d-default-backend on 2026-11-04, so the "proposed and unmerged" worry is moot. Open item: the base carries "OPEN QUESTION: confirm break-glass wording" (backend-contract:66); that is the one thing for the user to answer.

8. merc-policy "Revive and stop rule" (mercpolicy MODIFIED vs open m3-merc MODIFIED). Proposal: the archived mercpolicy text is the base; m3-merc's block is stale and should be dropped. (V) merc-policy:36 equals the mercpolicy delta (`reviveGoldMult` 1, reviveMax 2, stopGoldMult 5, stopDeaths 3), the delta says "user decision 2026-11-02 (supersedes the pending m3-merc delta)", and code agrees: `internal/env/tuning_merc.go:34` default 1 (V). m3-merc (`openspec/changes/m3-merc/specs/merc-policy/spec.md:17`) and ADR 0011 still say multiplier 2 on gold minus the reserve. Action: remove that MODIFIED block from the m3-merc card (keep its three ADDED ones) and amend ADR 0011's line 50 (ADR edit: proposed text below, not applied).

Proposed ADR 0011 note: "Superseded 2026-11-02 for the revive multiple: `merc.reviveGoldMult` is 1 on gold of at least price plus the kit reserve (merc-policy spec, user decision 2026-11-02)."

## Rows needing the user's decision

- Row 1/2/4-6 (gatedelta death and proxy rules): confirm DROPPED (the 2026-11-04 decision seems to cover it; one yes/no).
- Row 3 (Gate 1 bands): confirm advisory-only is the intended end state.
- Row 7 (Leaving d2d): confirm the wording that backend-contract:66 still flags as an OPEN QUESTION.
- Row 8 (Revive and stop rule): none for the spec (settled by 2026-11-02); needs go-ahead to strip the m3-merc block and note ADR 0011.
