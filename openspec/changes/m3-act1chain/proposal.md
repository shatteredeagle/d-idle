## Why

Nothing has completed Act 1 on either backend (docs/plan/STATUS.md): the quest chain after Blood Raven (Cain is optional, the Countess is skipped by the tower guard, Andariel is the only quest required for Warriv to open Act 2, d2-sme 2026-10-31) has never run on d2d, and the quest-state reads (docs/progression/quests.md: T1 save words, T2 memory) are V on Game.exe only. This card runs the chain on d2d from a cheat-levelled throwaway up to the Andariel approach and proves the quest evidence.

## What Changes

- internal/task/act1.go, campaign.go: the Act 1 order on d2d (Den, Blood Raven and the claim, optional Cain, Andariel, Warriv), each quest's done evidence read from the save or memory on d2d, no T3-only skips.
- docs/act1-d2d.md: the run log per quest with the evidence tier seen, what differs from Game.exe (U until seen), dead ends.
- Delta: act1-campaign ADDED `Act 1 quest order` and `Quest evidence on d2d`.

## Dependencies

After m3-merc and m3-routeguard are archived and after soak-slot:3 is free.
