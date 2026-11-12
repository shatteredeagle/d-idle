#!/bin/sh
# Regenerates assets/branding/out/ through Aseprite on the worker host, then packs ico/icns/bmp and checks the result.
# Run from the repo root through labrun on the worker host (the Aseprite binary is a Steam copy used as a command-line tool only).
set -eu
ASE="${ASEPRITE:-$HOME/.local/share/Steam/steamapps/common/Aseprite/aseprite}"
B=assets/branding
rm -rf "$B/out/png"; mkdir -p "$B/out/png" "$B/palettes"
"$ASE" -b --script-param out="$B/out/png" --script-param src="$B/src" --script-param pal="$B/palettes" --script "$B/src/didle.lua"
python3 -I tools/branding/pack.py "$B/out"
python3 -I tools/branding/pack.py --check "$B/out"
