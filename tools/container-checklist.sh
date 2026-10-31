#!/bin/bash
# Offline run of the docs/container.md "User test checklist" against the real d2fleet binary in
# `serve -setup -fake` mode: wizard host effects are fakes (internal/setup/setuptest) and no game starts.
# It proves the HTTP flow, the character hold, the slot hand-off, restart, and the no-key-on-disk rules.
# It does NOT prove: Docker flags, Wine prefix, the 1.9 GB copy, the real patch, real character creation,
# memory reads. Those stay on the user (docs/container.md "What needs you").
# Usage: tools/container-checklist.sh   (stdlib Go + curl + python3; needs no network beyond loopback)
set -u
cd "$(dirname "$0")/.."
T=$(mktemp -d); trap 'kill $PID 2>/dev/null; rm -rf "$T"' EXIT
PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
fail=0
ok() { echo "PASS  $*"; }
bad() { echo "FAIL  $*"; fail=1; }
chk() { local what=$1; shift; if "$@"; then ok "$what"; else bad "$what"; fi; }
jq_() { python3 -c "import sys,json;d=json.load(sys.stdin);print($1)"; }
api() { curl -s -H "Authorization: Bearer $TOK" -H 'Content-Type: application/json' -w '\n%{http_code}' "$@"; }
code() { api "$@" | tail -1; }
body() { api "$@" | sed '$d'; }

go build -o "$T/d2fleet" ./cmd/d2fleet || { echo "build failed"; exit 2; }
# 1. volumes: fresh /data, read-only-ish /src with a fake install, /patch
D=$T/data SRC=$T/src
mkdir -p "$D" "$SRC" "$T/patch"
for f in Game.exe "Diablo II.exe" d2data.mpq d2char.mpq d2sfx.mpq d2exp.mpq d2xmusic.mpq d2xtalk.mpq d2xvideo.mpq Patch_D2.mpq; do echo x > "$SRC/$f"; done
echo p > "$T/patch/LODPatch_114d.exe"
TOK=$(head -c 24 /dev/urandom | base64 | tr -dc A-Za-z0-9 | head -c 24)
echo "$TOK" > "$D/token"; chmod 600 "$D/token"
ok "1 volumes prepared (fresh data dir, token file)"

start() { "$T/d2fleet" serve -setup -fake -addr "127.0.0.1:$PORT" -data "$D" -token-file "$D/token" >"$T/out.$1" 2>"$T/err.$1" & PID=$!
  for _ in $(seq 100); do curl -s -o /dev/null "http://127.0.0.1:$PORT/setup.html" && return 0; sleep 0.1; done; return 1; }
U=http://127.0.0.1:$PORT
waitstate() { for _ in $(seq 200); do s=$(body "$U/api/setup/state"); if [ "$(echo "$s" | jq_ "$1")" = "True" ]; then return 0; fi; sleep 0.05; done; return 1; }

# 2. start
chk "2 serve -setup starts, admin API line printed" start 1
chk "2 stdout has admin API line" grep -q "admin API on" "$T/out.1"
chk "2 no slots started" test "$(body "$U/api/v1/fleet-admin/slots" | jq_ "len(d) if isinstance(d,list) else len(d.get('slots',[]))")" = 0
# 2b. token required
chk "2 API refuses without token" test "$(curl -s -o /dev/null -w '%{http_code}' $U/api/setup/state)" = 401

# 3. step list and doctor
ids=$(body "$U/api/setup/state" | jq_ "','.join(s['id'] for s in d['steps'])")
chk "3 step order ($ids)" test "$ids" = "data-dir,sources,prefix,install-d2,install-lod,import-game,patch,extract,character"
chk "3 doctor endpoint answers" test "$(code "$U/api/setup/checks?pre=1&quick=1")" = 200

# 4. sources + start
chk "4 PUT inputs (game-dir)" test "$(code -X PUT "$U/api/setup/inputs" -d "{\"mode\":\"game-dir\",\"gameDirSrc\":\"$SRC\",\"patchFile\":\"$T/patch/LODPatch_114d.exe\"}")" = 200
chk "4 POST start" test "$(code -X POST "$U/api/setup/start")" = 202
# 5. hold at character
chk "5 run stops at character, not failed" waitstate "d['current']=='character' and not d['running'] and not d['done'] and not any(s['status']=='failed' for s in d['steps'])"
chk "5 no slot yet" test "$(body "$U/api/v1/fleet-admin/slots" | jq_ "len(d) if isinstance(d,list) else len(d.get('slots',[]))")" = 0
chk "8 cdKey field is a 400" test "$(code -X PUT "$U/api/setup/inputs" -d '{"cdKey":"AAAA-BBBB"}')" = 400
chk "5 PUT name/class/handicap/goals" test "$(code -X PUT "$U/api/setup/inputs" -d '{"charName":"Wizbot","charClass":"necromancer","profile":"rookie","goals":"levelto:8"}')" = 200
code -X POST "$U/api/setup/start" >/dev/null
chk "5 wizard done" waitstate "d['done']"
chk "5 character file written" test -f "$(find "$D" -name Wizbot.d2s | head -1)"
# 6. bots resume: the wizard's character is a slot
for _ in $(seq 100); do n=$(body "$U/api/v1/fleet-admin/slots" | grep -c Wizbot); [ "$n" -gt 0 ] && break; sleep 0.1; done
chk "6 slot Wizbot with class necromancer, rookie, goals levelto:8" sh -c "curl -s -H 'Authorization: Bearer $TOK' $U/api/v1/fleet-admin/slots | grep -q necromancer"
# 8. no key material on disk
chk "8 setup.json has no key" sh -c "! grep -qi 'cdkey\|AAAA-BBBB' '$D/setup.json'"

# 7. restart: wizard done at once, slot restored without the wizard
kill $PID; wait $PID 2>/dev/null
chk "7 restart" start 2
chk "7 wizard still done" waitstate "d['done']"
for _ in $(seq 100); do n=$(body "$U/api/v1/fleet-admin/slots" | grep -c Wizbot); [ "$n" -gt 0 ] && break; sleep 0.1; done
chk "7 slot restored once" test "$(body "$U/api/v1/fleet-admin/slots" | grep -o Wizbot | wc -l)" = 1
kill $PID; wait $PID 2>/dev/null

# 8b. crash during a step resumes: simulate a persisted 'running' extract
python3 - "$D/setup.json" <<'P'
import json,sys
p=sys.argv[1]; d=json.load(open(p))
st=d.get('steps') or d.get('Steps') or {}
if 'extract' in st: st['extract']['status']='running'
json.dump(d,open(p,'w'))
P
chk "8 restart after simulated crash" start 3
chk "8 wizard back to done (resumed or verified)" sh -c "$(declare -f waitstate body api jq_); TOK=$TOK U=$U; waitstate \"d['done']\""
kill $PID; wait $PID 2>/dev/null
# 8c. bot-owned game model (Yama): d2bot --managed launches Game.exe, the fleet only prepares (fake exec, offline)
chk "8c managed bot launches the game, restart/remove go through d2bot" go test -count=1 -run 'TestManaged' ./internal/fleet
# 9. cleanup is the trap
ok "9 cleanup (temp dir removed on exit)"
[ $fail = 0 ] && echo "ALL OFFLINE CHECKS PASSED" || { echo "SOME CHECKS FAILED"; exit 1; }
