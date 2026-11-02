# verifyprices: Opus verifier of fix/vendor-prices-followon-v2

## Why

The vendor prices follow-on (private fork branch `fix/vendor-prices-followon-v2`, head da80d0ab: repair gate,
divide-first npc mults, oracle) had only a non-Opus verifier (docs/research/verify-price2.md). The standing rule
asks one Opus pass with an own harness before an upstream PR.

## What Changes

Docs and verifier tooling only: docs/research/pr-prices-verifier.md and tools/emu/vpr_* (own game.exe oracle,
own d2d probe include, disassembly helper). No d2d or Go changes; nothing pushed or posted.
