#!/bin/bash
# Offline test of tools/emu/run-on-worker.sh with a stub labrun and agentrun (no ssh, no worker host).
#   bash tools/emu/run-on-worker-test.sh      exit 0 = all pass
set -u
export D2_LABRUN_HOST=testhost
here=$(cd "$(dirname "$0")" && pwd -P)
W=$here/run-on-worker.sh
T=$(mktemp -d); trap 'rm -rf "$T"' EXIT
# a fake worktree with its own tools/emu so the test scripts never touch the real one
mkdir -p "$T/wt/tools/emu"; cp "$W" "$T/wt/tools/emu/"
cat > "$T/labrun" <<'S'
#!/bin/bash
echo "LABRUN $*" >> "$TESTLOG"
S
cat > "$T/agentrun" <<'S'
#!/bin/bash
echo "AGENTRUN $*" >> "$TESTLOG"; shift; [ "$1" = --ttl ] && shift 2; shift; exec "$@"
S
chmod +x "$T/labrun" "$T/agentrun"
export TESTLOG=$T/log WRAP_LABRUN=$T/labrun WRAP_AGENTRUN=$T/agentrun D2D_JOB=tj
echo 'echo hi' > "$T/wt/tools/emu/ok.sh"; echo 'print(1)' > "$T/wt/tools/emu/ok.py"
echo 'sudo ls' > "$T/wt/tools/emu/bad.sh"; echo 'pkill x' > "$T/wt/tools/emu/bad2.sh"; echo 'wine game.exe' > "$T/wt/tools/emu/bad3.sh"
ln -s ok.sh "$T/wt/tools/emu/link.sh"
R=$T/wt/tools/emu/run-on-worker.sh
pass=0 fail=0
ok() { pass=$((pass + 1)); echo "ok   $1"; }
bad() { fail=$((fail + 1)); echo "FAIL $1"; }
accepts() { local d=$1; shift; : > "$TESTLOG"; "$R" "$@" >/dev/null 2>&1 && ok "$d" || bad "$d"; }
refuses() { local d=$1; shift; : > "$TESTLOG"; "$R" "$@" >/dev/null 2>&1; [ $? = 2 ] && [ ! -s "$TESTLOG" ] && ok "$d" || bad "$d"; }

accepts "sh script with args" ok.sh 1-3 "Act 1 H2H A" -v /tmp/d2agent-tj/out.txt /tmp/ug-emu
grep -q 'AGENTRUN tj --ttl 930 -- ' "$TESTLOG" && grep -q 'LABRUN testhost -- env D2_SCRATCH=/tmp/d2agent-tj sh -c .* sh timeout 900 bash tools/emu/ok.sh 1-3 Act 1 H2H A -v' "$TESTLOG" && ok "command line shape (agentrun ttl+30, timeout 900)" || { bad "command line shape"; cat "$TESTLOG"; }
accepts "py script via run-py.sh" ok.py x
grep -q 'bash tools/emu/run-py.sh ok.py x' "$TESTLOG" && ok "py uses run-py.sh" || bad "py uses run-py.sh"
accepts "custom ttl" --ttl 60 ok.sh; grep -q -- '--ttl 90 ' "$TESTLOG" && grep -q 'timeout 60 ' "$TESTLOG" && ok "ttl 60 -> agentrun 90" || bad "ttl mapping"
refuses "ttl too big" --ttl 99999 ok.sh
refuses "ttl too small" --ttl 1 ok.sh
refuses "no script" 
refuses "path as script" ../ok.sh
refuses "subdir script" sub/ok.sh
refuses "wrong extension" ok.txt
refuses "missing script" nope.sh
refuses "symlink script" link.sh
refuses "script naming sudo" bad.sh
refuses "script naming pkill" bad2.sh
refuses "script naming wine" bad3.sh
refuses "arg with semicolon" ok.sh 'a;b'
refuses "arg with dollar" ok.sh '$HOME'
refuses "arg with backtick" ok.sh '`id`'
refuses "arg with .." ok.sh /tmp/ug-emu/../etc
refuses "abs path /etc" ok.sh /etc/passwd
refuses "abs path of another agent scratch" ok.sh /tmp/d2agent-other/x
refuses "abs path under home but not d2d-lab" ok.sh "$HOME/.ssh/id_rsa"
accepts "d2d-lab path" ok.sh "$HOME/d2d-lab/src/tools/emu/x.py"
D2D_JOB='a b' refuses "bad agent name" ok.sh
[ $fail = 0 ] && echo "PASS ($pass)" || echo "FAIL ($fail of $((pass + fail)))"
[ $fail = 0 ]
