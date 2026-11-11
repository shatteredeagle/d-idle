#!/bin/bash
# esverify3 (the worker host): clang-tidy (the fork's .clang-tidy, HeaderFilterRegex covers components/) on TUs that include
# combat.hpp and ai.hpp, on base AND head, warnings in the two headers listed with their line; then the head's tests with the data.
S=${D2_SCRATCH:-/tmp/d2agent-esverify3}
for t in base head; do
	SRC=$(cd "$(dirname "$0")/../../.." && pwd)/verifier3-$t
	B=$S/$t/build3
	nice cmake -S "$SRC" -B "$B" -DCMAKE_EXPORT_COMPILE_COMMANDS=ON > /dev/null 2>&1 || echo "reconfigure failed"
	cd "$SRC"
	for f in tests/test_combat.cpp components/game/ai.cpp; do
		echo "== $t clang-tidy $f"
		nice -n 10 clang-tidy -p "$B" --quiet --extra-arg=-Wno-unknown-warning-option "$f" 2>&1 | grep -E "(warning|error):" | grep -E "combat.hpp|ai.hpp|test_combat.cpp" | sort -u | head -30
		echo "exit ${PIPESTATUS[0]}"
	done
done
clang-tidy --version | head -2
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
for t in base head; do
	B=$S/$t/build3
	for x in test_combat test_energy_shield test_game; do
		exe=$(find "$B/tests" -name "$x" -type f | head -1)
		echo "== $t $x"; [ -n "$exe" ] || { echo "not built"; continue; }; nice -n 10 timeout 600 "$exe" > "$S/$t-$x.log" 2>&1; echo "exit $?"; grep -v WARNING "$S/$t-$x.log" | tail -2
	done
done
