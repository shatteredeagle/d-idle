# b2winworker

## Why

The batch 2 r3 branches (`local/batch2-bundle-r3` 824da884, `local/batch2-mana-teleport` dc6618ae, `local/batch2-monsterdmg-pr` bbd339ad) were checked on the worker host only. The MSVC 14.51 lane, the WSL Linux stages, Teleport seed 9 and the damage-roll sets were left open (b2rebase2 task 1.5) because the worker host was busy with Gate 1. Windows build worker has the owner's go for all of 2026-11-03.

## What Changes

- `docs/research/b2-winworker-rerun.md`: the numbers of those runs (tagged V with the command).
- Tooling lives in the private ops repo (gate rule, job files, WSL script); nothing here.

## Impact

Docs only. Nothing pushed upstream.

Overlap (declared): reads the b2rebase2 card's checklist and results; edits none of its files.
