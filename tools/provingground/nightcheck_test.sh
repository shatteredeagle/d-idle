#!/bin/bash
# Offline test of nightcheck.sh: a good command file passes, a missing id, an unknown knob and a bad flag each fail,
# and a block fenced `sh deferred` only warns.
here=$(cd "$(dirname "$0")" && pwd); d=$(mktemp -d); trap 'rm -rf "$d"' EXIT; fail=0
export NC_BIN=$d/d2scenario; (cd "$here/../.." && go build -o "$NC_BIN" ./cmd/d2scenario) || exit 2
run() { printf '%s\n' '```sh' "$1" '```' > "$d/c.md"; [ -n "${2:-}" ] && printf '%s\n' '```sh deferred' "$2" '```' >> "$d/c.md"; bash "$here/nightcheck.sh" "$d/c.md" 2>&1; }
t() { out=$(run "$2" "${3:-}"); rc=$?; if [ "$rc" = "$1" ] && echo "$out" | grep -q -- "${4:-nightcheck}"; then echo "ok   $5"; else echo "FAIL $5 (rc $rc)"; echo "$out" | sed 's/^/     /'; fail=1; fi; }
t 0 'tools/provingground/run.sh -id cain.read,town.plan. -class Paladin -seed 11 -parallel 2' '' 'nightcheck: OK' "good ids, prefix, class and seed"
t 1 'tools/provingground/run.sh -id cain.read,no.such.id -class Paladin -seed 11' '' 'MISSING IDS' "missing id fails and is listed"
t 1 'D2SCEN_KNOBS=no.such.knob=1 tools/provingground/run.sh -id cain.read -class Paladin -seed 11' '' 'unknown knob: no.such.knob' "unknown knob fails"
t 1 'tools/provingground/run.sh -id cain.read -bogus 3' '' 'not a d2scenario run flag' "bad flag fails"
t 1 'tools/provingground/run.sh -id cain.read -class Bogus -seed 11' '' 'plan refused' "unknown class fails"
t 0 'tools/provingground/run.sh -id cain.read -class Paladin -seed 11' 'tools/provingground/run.sh -id not.yet.there' 'nightcheck: OK' "deferred block is skipped"
out=$(printf '%s\n' '```sh deferred' 'tools/provingground/run.sh -id not.yet.there' '```' > "$d/c.md"; bash "$here/nightcheck.sh" "$d/c.md" --include-deferred 2>&1); echo "$out" | grep -q "id not on main (deferred): not.yet.there" && echo "ok   --include-deferred lists it as WARN" || { echo "FAIL include-deferred"; fail=1; }
exit $fail
