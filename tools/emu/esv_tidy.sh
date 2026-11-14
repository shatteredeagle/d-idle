#!/bin/bash
# esverify (the worker host): clang-tidy (the repo's .clang-tidy) on the files the head commit adds or changes, and the
# head's own test_energy_shield run. bash tools/emu/esv_tidy.sh
SRC=$(cd "$(dirname "$0")/../../.." && pwd)/verifier2-head     # esverify2; the first run used verifier-eshead
[ -d "$SRC" ] || SRC=$(cd "$(dirname "$0")/../../.." && pwd)/verifier-eshead
B=${D2_SCRATCH:-/tmp/d2agent-esverify}/head/build3
nice cmake -S "$SRC" -B "$B" -DCMAKE_EXPORT_COMPILE_COMMANDS=ON > /dev/null 2>&1 || echo "reconfigure failed"
nice ninja -C "$B" -j3 test_energy_shield test_combat > "$B/../tidy-ninja.log" 2>&1 || { tail -20 "$B/../tidy-ninja.log"; }
cd "$SRC"
for f in tests/game/test_energy_shield.cpp components/game/fight/states.cpp; do
	echo "== clang-tidy $f"
	nice clang-tidy -p "$B" --quiet "$f" 2>&1 | grep -E "warning|error" | grep -E "test_energy_shield|combat.hpp|ai.hpp|states.cpp" | head -20
	echo "exit ${PIPESTATUS[0]}"
done
export D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
for t in test_energy_shield test_combat; do
	exe=$(find "$B/tests" -name "$t" -type f | head -1)
	echo "== $t"; nice "$exe" 2>&1 | grep -v WARNING | tail -3; echo "exit ${PIPESTATUS[0]}"
done
