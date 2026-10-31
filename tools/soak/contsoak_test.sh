#!/bin/bash
# Offline test of contsoak.sh: bash tools/soak/contsoak_test.sh. A stub "ssh" runs the docker-exec'd command locally with /data
# mapped to a temp dir and d2soak built from this checkout.
set -u
here="$(cd "$(dirname "$0")" && pwd)"; repo="$here/../.."
t="$(mktemp -d)"; trap 'rm -rf "$t"' EXIT
bad=0; fail() { echo "FAIL: $*"; bad=$((bad + 1)); }
(cd "$repo" && go build -o "$t/d2soak" ./cmd/d2soak) || { echo "build failed"; exit 2; }
mkdir -p "$t/data/soak/i1-20260101-000000" "$t/data/soak/i1-20260102-000000" "$t/data/soak/i2-20260101-000000"
for r in i1-20260101-000000 i1-20260102-000000 i2-20260101-000000; do
  for n in 1 2 3; do echo "{\"t\":\"2026-01-02T00:0$n:00Z\",\"http\":200,\"healthOk\":true,\"progress\":$n,\"ladder\":0,\"xp\":$((n * 100)),\"kills\":$n,\"drops\":0,\"deaths\":0,\"errors\":0,\"seq\":0}"; done >"$t/data/soak/$r/samples.jsonl"
done
cat >"$t/ssh" <<EOS
#!/bin/bash
shift 6   # -o BatchMode=yes user@host docker exec NAME
args=()
for a in "\$@"; do args+=("\${a//\/data/$t/data}"); done
if [ "\${args[0]}" = d2soak ]; then "$t/d2soak" "\${args[@]:1}" 2>&1 | sed "s#$t/data#/data#g"; exit \${PIPESTATUS[0]}; fi
"\${args[@]}" | sed "s#$t/data#/data#g"; exit \${PIPESTATUS[0]}
EOS
chmod +x "$t/ssh"
export D2_SOAK_SSH="$t/ssh -o BatchMode=yes"   # the script appends user@host docker exec NAME ...
out="$(bash "$here/contsoak.sh" fakehost status 2>&1)"
echo "$out" | grep -q '^i1-20260102-000000 ' || fail "status shows the newest i1 run: $out"
echo "$out" | grep -q '^i1-20260101-000000 ' && fail "status must not show the older i1 run"
echo "$out" | grep -q '^i2-20260101-000000 ' || fail "status shows i2: $out"
bash "$here/contsoak.sh" fakehost report "$t/rep" >"$t/rep.out" 2>&1
[ -s "$t/rep/i1-20260102-000000/report.txt" ] || fail "report.txt copied: $(cat "$t/rep.out")"
[ -f "$t/rep/i1-20260102-000000/samples.jsonl.gz" ] || fail "samples copied and gzipped"
bash "$here/contsoak.sh" fakehost fault i9 2>&1 | grep -q 'killed 0 Game.exe of i9' || fail "fault runs and reports"
bash "$here/contsoak.sh" fakehost fault x1 >/dev/null 2>&1 && fail "fault rejects a bad slot"
[ "$bad" = 0 ] && echo "contsoak_test: ok"; exit "$bad"
