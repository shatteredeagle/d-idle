#!/bin/bash
# MERC2: build an emulator data dir in the scratch (the shared /tmp/ug-emu is gone): MPQs linked, game.exe from a read-only copy.
#   tools/emu/run-on-worker.sh mc2_setup.sh
set -e
D=${D2_SCRATCH:?}/ug
mkdir -p "$D/bin"
for f in "$HOME"/d2d-lab/data/*.mpq; do ln -sf "$f" "$D/$(basename "$f" | tr A-Z a-z)"; ln -sf "$f" "$D/$(basename "$f")"; done
src=$HOME/d2d-lab/verifymissile/data/bin
cp -n "$src"/* "$D/bin/" 2>/dev/null || true
ls -la "$D/bin"; md5sum "$D/bin/game.exe"
