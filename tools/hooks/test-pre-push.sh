#!/bin/sh
# Test of tools/hooks/pre-push in a temp repo with a local bare remote (no network, no real go/node).
# A fake `go` records its calls; the hook must run the checks for a push to main and to master, not for other branches,
# and a failing check must block. Usage: sh tools/hooks/test-pre-push.sh
set -eu
src=$(cd "$(dirname "$0")" && pwd)
t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT
mkdir -p "$t/bin" "$t/work/tools/hooks" "$t/work/openspec"
cp "$src/pre-push" "$t/work/tools/hooks/pre-push"
chmod +x "$t/work/tools/hooks/pre-push"
printf 'exit 0\n' >"$t/work/tools/openspec.sh"
cat >"$t/bin/go" <<EOF
#!/bin/sh
echo "\$*" >>"$t/golog"
case "\$*" in *changecheck*) [ -f "$t/fail" ] && exit 1;; esac
exit 0
EOF
chmod +x "$t/bin/go"
: >"$t/golog"
git init -q --bare "$t/remote.git"
cd "$t/work"
git init -q -b main
git config user.name t; git config user.email "$(printf t)@localhost"; git config commit.gpgsign false
git config core.hooksPath tools/hooks
git remote add origin "$t/remote.git"
git add -A; git commit -q -m init
export PATH="$t/bin:$PATH"
n() { grep -c changecheck "$t/golog" || true; }
check() { [ "$2" = "$3" ] || { echo "FAIL: $1 (got $3, want $2)"; exit 1; }; }

git push -q origin main 2>/dev/null
check "push main runs changecheck" 1 "$(n)"
git push -q origin main:master 2>/dev/null
check "push master runs changecheck" 2 "$(n)"
git push -q origin main:agent/x 2>/dev/null
check "push of another branch does not" 2 "$(n)"
touch "$t/fail"
git commit -q --allow-empty -m next
if git push -q origin main:main 2>/dev/null; then echo "FAIL: failing check did not block main"; exit 1; fi
if git push -q origin main:master 2>/dev/null; then echo "FAIL: failing check did not block master"; exit 1; fi
git push -q origin main:agent/y 2>/dev/null || { echo "FAIL: other branch blocked"; exit 1; }
D2_SKIP_HOOKS=1 git push -q origin main:main 2>/dev/null || { echo "FAIL: D2_SKIP_HOOKS did not bypass"; exit 1; }
echo "pre-push hook ok"
