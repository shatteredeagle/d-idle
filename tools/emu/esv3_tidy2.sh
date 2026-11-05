#!/bin/bash
# esverify3: the full clang-tidy findings on ai.cpp (base and head) and on a TU per touched header, grouped by file.
S=${D2_SCRATCH:-/tmp/d2agent-esverify3}
for t in base head; do
	SRC=$(cd "$(dirname "$0")/../../.." && pwd)/verifier3-$t
	cd "$SRC"
	for f in components/game/ai.cpp components/game/world/tick.cpp; do
		echo "== $t $f"
		nice -n 10 clang-tidy -p "$S/$t/build3" --quiet --extra-arg=-Wno-unknown-warning-option "$f" > "$S/tidy-$t-$(basename $f).log" 2>&1
		echo "exit $?"
		grep -E "(warning|error):" "$S/tidy-$t-$(basename $f).log" | sed -E 's#^.*/(components|tests|apps)/#\1/#' | sort | uniq | head -20
	done
done
