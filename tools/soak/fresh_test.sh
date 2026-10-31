#!/bin/bash
# Offline test of the refusal logic of fresh.sh and the ledger warning of soak24.sh: bash tools/soak/fresh_test.sh
# Requests that must be refused are run through fresh.sh itself (they exit before any ssh).
set -u
here="$(cd "$(dirname "$0")" && pwd)"
. "$here/freshcheck.sh"
bad=0
fail() { echo "FAIL: $*"; bad=$((bad + 1)); }
names="$here/fresh-names.txt"
ok_args() { r="$(fresh_check_args "$@" "$names")" || fail "should accept: $* ($r)"; }
no_args() { r="$(fresh_check_args "$@" "$names")" && fail "should refuse: $*"; [ -n "$r" ] || fail "refusal without a reason: $*"; return 0; }
ok_args i4 sorceress Fresha
ok_args i1 necromancer freshb       # the list match is case-insensitive
for c in amazon assassin barbarian druid necromancer paladin sorceress; do ok_args i2 $c Freshc; done
no_args i0 sorceress Fresha         # never the user's own game
no_args i10 sorceress Fresha
no_args "" sorceress Fresha
no_args i4 wizard Fresha
no_args i4 sorceress Soaky          # real character, not on the list
no_args i4 sorceress Testy
no_args i4 sorceress Fresh1         # digit
no_args i4 sorceress 'Fresha;ls'
no_args i4 sorceress ../Fresha
no_args i4 sorceress ''
no_args i4 sorceress 'Fresh a'
# every listed name is itself a valid character name and none is a known real character
for n in $(grep -v '^[[:space:]]*#' "$names" | tr -d '\r'); do
  ok_args i3 sorceress "$n"
  case "$n" in Soaky|Testy|Botty|Navi|Combi|Towni|Booti|Multi|Palli*|Necki|Amazi|Druki|Assi|Barby) fail "$n is a real character" ;; esac
done
# busy tokens
fresh_busy $'PROBE_DONE' >/dev/null || true
r="$(fresh_busy $'PROBE_DONE')"; rc=$?; [ $rc = 0 ] && [ -z "$r" ] || fail "clean probe refused: $r"
for t in NOTICE SOAKUNIT BOT LOCKNEW LOCKOLD GAME SAVE; do
  r="$(fresh_busy "$t"$'\nPROBE_DONE')" && fail "token $t not refused"
  [ -n "$r" ] || fail "token $t had no reason"
done
r="$(fresh_busy $'GAME\nBOT\nPROBE_DONE')" || true; [ "$(echo "$r" | wc -l)" = 2 ] || fail "two tokens should give two reasons: $r"
[ "$(fresh_soak_line i4 Fresha)" = "tools/soak/soak24.sh start i4 --gate --char Fresha" ] || fail "soak line"
# fresh.sh itself refuses before touching ssh (a bogus ssh on PATH would be noticed)
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
printf '#!/bin/sh\necho SSH-CALLED >&2; exit 99\n' >"$tmp/ssh"; chmod +x "$tmp/ssh"
for args in "i0 sorceress Fresha" "i4 wizard Fresha" "i4 sorceress Soaky" "i4 sorceress"; do
  out="$(PATH="$tmp:$PATH" "$here/fresh.sh" $args 2>&1 </dev/null)"; rc=$?
  { [ "$rc" = 2 ] && ! echo "$out" | grep -q SSH-CALLED; } || fail "fresh.sh $args: rc=$rc out=$out"
done
# soak24.sh ledger check: ledger_warn prints a WARN when the character is in the ledger
. "$here/ledgercheck.sh"
led="$tmp/cheats.jsonl"
printf '%s\n' '{"time":"2026-10-30T10:00:00Z","character":"Fresha","kind":"gold","success":true}' \
  '{"time":"2026-10-30T10:00:01Z","character":"Fresha","kind":"level","success":false}' \
  '{"time":"2026-10-30T10:00:02Z","character":"Other","kind":"heal","success":true}' >"$led"
[ "$(ledger_count "$led" Fresha)" = "1" ] || fail "ledger_count Fresha counts successful lines only: $(ledger_count "$led" Fresha)"
[ "$(ledger_count "$led" fresha)" = "1" ] || fail "ledger_count is case-insensitive"
[ "$(ledger_count "$led" Nobody)" = "0" ] || fail "ledger_count Nobody"
[ "$(ledger_count "$tmp/missing" Fresha)" = "0" ] || fail "missing ledger is empty"
[ "$(ledger_count "$led" Fresh)" = "0" ] || fail "ledger_count must match the whole name"
[ "$bad" = 0 ] && echo "fresh_test: ok" || { echo "fresh_test: $bad failure(s)"; exit 1; }
