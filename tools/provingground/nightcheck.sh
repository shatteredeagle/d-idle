#!/bin/bash
# Dry preflight of a night run command file (no game, no slot, no lab d2d): run it first, at 22:57.
#   tools/provingground/nightcheck.sh [commands.md] [--include-deferred]
# Reads the fenced sh blocks of the commands file (default docs/probes/lab-night-commands-2026-11-27.md) and checks
#   1. every `-id` scenario id (prefixes ending in '.' too) exists in tools/provingground/scenarios,
#   2. every run.sh flag is a flag of d2scenario run, and `d2scenario plan` accepts each id/class/seed (dry run),
#   3. every D2SCEN_KNOBS / knobs3way knob key is a row of docs/knobs.md,
#   4. each id carries the `no_writes` assertion (the "writes after start: N" line) and the results ledger command is set,
#   5. the lab d2d binary and data dir named by PG_D2D_BIN / PG_DATA (or run.sh's defaults) are present
#      (here; with NC_WORKER=<labrun host> on that host; a missing path is a WARN unless NC_STRICT_PATHS=1).
# Blocks fenced as `sh deferred` hold scenarios that are not on main yet: with --include-deferred their missing ids are
# listed as WARN, without it they are skipped. Exit 0 clean, 1 on any missing id, bad flag, plan error or unknown knob.
set -u
here=$(cd "$(dirname "$0")" && pwd); repo=$(cd "$here/../.." && pwd); cd "$repo" || exit 2
export PATH=$PATH:/opt/go/bin:$HOME/.local/bin
file=docs/probes/lab-night-commands-2026-11-27.md; deferred=
for a in "$@"; do case $a in --include-deferred) deferred=1 ;; *) file=$a ;; esac; done
[ -r "$file" ] || { echo "nightcheck: cannot read $file" >&2; exit 2; }
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
bin=${NC_BIN:-$tmp/d2scenario}
[ -x "$bin" ] || go build -o "$bin" ./cmd/d2scenario || { echo "nightcheck: d2scenario does not build" >&2; exit 2; }
"$bin" list > "$tmp/list" 2>&1 || { echo "nightcheck: d2scenario list failed" >&2; exit 2; }
"$bin" run -h 2>&1 | awk '/^  -[a-z-]+/{print $1}' > "$tmp/runflags"
grep -q 'case "-n"' cmd/d2scenario/detrun.go && printf "%s\n" -n >> "$tmp/runflags"
echo "-dry-run" >> "$tmp/runflags"; echo "-journal" >> "$tmp/runflags"
bad=0; warn() { echo "WARN  $*"; }; fail() { echo "FAIL  $*"; bad=1; }

python3 -I - "$file" "$tmp" "${deferred:-0}" <<'PY' || exit 2
import re, sys
f, tmp, inc = sys.argv[1], sys.argv[2], sys.argv[3] == "1"
blocks, cur, info = [], None, ""
for line in open(f):
    m = re.match(r"^```(\S*)\s*(\S*)", line)
    if m and cur is None:
        cur, info = [], m.group(2)
        if m.group(1) not in ("sh", "bash"): cur = None
        continue
    if line.startswith("```"):
        if cur is not None: blocks.append((info, cur))
        cur = None
        continue
    if cur is not None: cur.append(line.rstrip("\n"))
ids, plans, knobs, feats = [], [], [], []
for info, lines in blocks:
    dfr = info == "deferred"
    if dfr and not inc: continue
    text = "\n".join(lines).replace("\\\n", " ")
    for m in re.finditer(r"for \w+ in ([^;]+); do", text):
        for i in m.group(1).split(): ids.append((i, dfr))
    for cmd in re.split(r"\n|;", text):
        for m in re.finditer(r"D2SCEN_KNOBS=([^\s']+)", cmd):
            knobs += [(kv.split("=")[0], dfr) for kv in m.group(1).split(",")]
        if "knobs3way.sh" in cmd:
            for arm in re.findall(r"\b\w+=(\S+)", cmd.split('"', 2)[-1] if '"' in cmd else cmd):
                knobs += [(kv.split("=")[0], dfr) for kv in arm.split(",") if "=" in kv]
        if "run.sh" in cmd:
            m = re.search(r"-id\s+(\S+)", cmd)
            if not m: continue
            idl = [i for i in m.group(1).split(",")]
            ids += [(i, dfr) for i in idl]
            fl = re.findall(r"(?:^|\s)(-[a-z][a-z-]*)", cmd.split("run.sh", 1)[1])
            c = re.search(r"-class\s+(\w+)", cmd); s = re.search(r"-seed\s+(\d+)", cmd); p = re.search(r"-parallel\s+(\d+)", cmd)
            plans.append((m.group(1), c and c.group(1), s and s.group(1), p and p.group(1), fl, dfr))
        if "featuresoak.py" in cmd and " run " in cmd:
            c = re.search(r"(tools/\S+\.json)", cmd); ft = re.search(r"--features\s+(\S+)", cmd)
            if c and ft: feats.append((c.group(1), ft.group(1), dfr))
w = lambda n, rows: open(f"{tmp}/{n}", "w").write("\n".join(rows) + "\n")
w("ids", sorted({f"{i}\t{int(d)}" for i, d in ids}))
w("knobs", sorted({f"{k}\t{int(d)}" for k, d in knobs}))
w("plans", [f"{a}|{b or ''}|{c or ''}|{d or ''}|{' '.join(e)}|{int(g)}" for a, b, c, d, e, g in plans])
w("feats", [f"{a}\t{b}\t{int(c)}" for a, b, c in feats])
PY

# 1. ids
n=0; while IFS=$'\t' read -r id dfr; do [ -n "$id" ] || continue; n=$((n+1))
  case $id in *.) hit=$(awk -v p="$id" 'index($1,p)==1' "$tmp/list" | head -1) ;; *) hit=$(awk -v p="$id" '$1==p' "$tmp/list") ;; esac
  if [ -z "$hit" ]; then [ "$dfr" = 1 ] && warn "id not on main (deferred): $id" || { fail "missing id: $id"; echo "$id" >> "$tmp/missing"; }; fi
done < "$tmp/ids"
echo "ids: $n checked"
# 2. flags and plan
while IFS='|' read -r idl cls seed par fl dfr; do [ -n "$idl" ] || continue
  for f in $fl; do grep -qx -- "$f" "$tmp/runflags" || fail "flag $f is not a d2scenario run flag ($idl)"; done
  a=(-id "$idl"); [ -n "$cls" ] && a+=(-class "$cls"); [ -n "$seed" ] && a+=(-seed "$seed"); [ -n "$par" ] && a+=(-parallel "$par")
  out=$("$bin" plan "${a[@]}" 2>&1) || { [ "$dfr" = 1 ] && warn "plan refused (deferred): ${a[*]}" || fail "plan refused: ${a[*]}: $(echo "$out" | head -1)"; continue; }
  runs=$(echo "$out" | sed -n 's/^DRY RUN [^:]*: \([0-9]*\) runs.*/\1/p')
  [ "${runs:-0}" -gt 0 ] || { [ "$dfr" = 1 ] && warn "plan has 0 runs (deferred): ${a[*]}" || fail "plan has 0 runs: ${a[*]}"; }
done < "$tmp/plans"
# 3. knobs
while IFS=$'\t' read -r k dfr; do [ -n "$k" ] || continue
  grep -q "^| $k |" docs/knobs.md || { [ "$dfr" = 1 ] && warn "knob not in docs/knobs.md (deferred): $k" || fail "unknown knob: $k"; }
done < "$tmp/knobs"
# 3b. featuresoak features
while IFS=$'\t' read -r c ft dfr; do [ -n "$c" ] || continue
  [ -r "$c" ] && grep -q "\"$ft\"" "$c" || fail "featuresoak: $ft not in $c"
done < "$tmp/feats"
# 4. writes-after-start assertion and ledger
python3 -I - "$tmp" <<'PY'
import glob, json, sys
tmp = sys.argv[1]
have = {}
for f in glob.glob("tools/provingground/scenarios/*.json"):
    for s in json.load(open(f)).get("scenarios", []): have[s["id"]] = "no_writes" in json.dumps(s)
for line in open(f"{tmp}/ids"):
    i = line.split("\t")[0]
    ex = [k for k in have if (k.startswith(i) if i.endswith(".") else k == i)]
    for k in ex:
        if not have[k]: print(f"WARN  {k}: no no_writes assertion (its 'writes after start' line is not asserted)")
PY
( . "$here/pgenv.sh" 2>/dev/null; [ -n "${PG_LEDGER_CMD:-}" ] ) && echo "ledger: PG_LEDGER_CMD set" || warn "results ledger command (PG_LEDGER_CMD) not set on this host: results stay in results.json only"
# 5. lab paths
d2d=${PG_D2D_BIN:-/data/d2d-lab/survhold/bin/d2d.lab}; dat=${PG_DATA:-/data/d2d-lab/data}
for p in "PG_D2D_BIN:$d2d" "PG_DATA:$dat"; do
  if [ -n "${NC_WORKER:-}" ]; then labrun "$NC_WORKER" --light -- ls -d "${p#*:}" >/dev/null 2>&1 && r=present || r=missing; else [ -e "${p#*:}" ] && r=present || r=missing; fi
  echo "path ${p%%:*}: $r"
  if [ $r = missing ]; then [ -n "${NC_STRICT_PATHS:-}" ] && fail "${p%%:*} missing" || warn "${p%%:*} missing here (set NC_WORKER=<host> to test it there)"; fi
done
[ -s "$tmp/missing" ] && { echo "MISSING IDS:"; sort -u "$tmp/missing"; }
[ $bad = 0 ] && echo "nightcheck: OK" || echo "nightcheck: FAILED"
exit $bad
