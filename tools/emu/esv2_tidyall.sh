#!/bin/bash
# esverify2 (the worker host): clang-tidy findings of the head's touched files, unfiltered, as file:check counts.
SRC=$(cd "$(dirname "$0")/../../.." && pwd)/verifier2-head
B=${D2_SCRATCH:-/tmp/d2agent-esverify2}/head/build3
cd "$SRC"
for f in tests/game/test_energy_shield.cpp components/game/fight/states.cpp; do
	echo "== $f"
	nice clang-tidy -p "$B" --quiet "$f" 2>&1 | grep -oE "[A-Za-z0-9_/.]+:[0-9]+:[0-9]+: (warning|error): .*\[[a-z0-9.,-]+\]" | sed -E 's@^(.*/)?([^/]+):[0-9]+:[0-9]+: (warning|error): .*\[([^]]+)\]$@\2 \3 \4@' | sort | uniq -c
done
