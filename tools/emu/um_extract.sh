#!/bin/bash
# SPDX-License-Identifier: GPL-3.0-or-later
# um_extract.sh (card uniqmods): print one 1.14d table to stdout through mpq-cat on the worker host.
#   (on the worker host) bash tools/emu/um_extract.sh monumod|superuniques|monstats|levels|difficultylevels
# Read-only: links the MPQs into a scratch dir (patch layer under the lowercase name, see
# docs/process/mpq-data-reading.md), reads one table, deletes the dir.
set -e
S=$(mktemp -d)
for f in "$HOME"/d2d-lab/data/*; do ln -s "$f" "$S/$(basename "$f")"; done
ln -s "$HOME/d2d-lab/data/Patch_D2.mpq" "$S/patch_d2.mpq"
"$HOME/d2d-lab/src/build/tools/mpq-cat/mpq-cat" "$S" "data\\global\\excel\\$1.txt"
rm -rf "$S"
