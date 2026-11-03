## Why

`tests/monsters/test_monster_ai.cpp` failed to compile on the Windows build worker with MSVC C2026 (string too big) at upstream main 3e073888. We need to know whether that is an upstream problem or an artefact of our toolchain, and what the smallest fix is.

## What Changes

Docs only: a trace (`docs/research/msvcc2026-trace.md`) and three scripts under `tools/emu/` (literal sizes, the split, a byte-identity probe). No Go, no game, no push.

Result: a real latent upstream problem (the Windows CI job builds the test targets; four raw-string literals in two included files exceed 16380 bytes). A byte-identical split is proven by a g++ probe; the private branch and the MSVC rebuild are not done (see the trace).
