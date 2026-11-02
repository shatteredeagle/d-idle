# proof for CodeRabbit round 2 (worker host)
O=~/d2d-lab/crfix2; C=~/d2d-lab/crfix
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe PYTHONSAFEPATH=1
cat > $O/spawnfail.py <<'PY'
import importlib.util, sys
sys.path.insert(0, sys.argv[1] and "/home/jesse/d2d-lab/src/tools/emu")
import missiles
orig = missiles.Game.unit; n = [0]
def unit(self, cls, *a):
    if cls == 19:
        n[0] += 1
        if n[0] == 2: return None   # the second monster fails to spawn
    return orig(self, cls, *a)
missiles.Game.unit = unit
spec = importlib.util.spec_from_file_location("sgr", sys.argv[1]); m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
m.game("fireball", int(sys.argv[2]))
PY
PY=$HOME/d2d-lab/pr/walk2-venv/bin/python
head -n 30 <(grep -v '^R' $C/roll-fireball.d2d) > /dev/null
grep '^R' $C/roll-fireball.d2d | head -30 > $O/d2d30.txt
echo "== comment 1: second monster fails to spawn (3 -> 2 units), 30 trials, then cmp vs the 30-trial d2d file"
for v in old new; do
  nice -n 19 $PY $O/spawnfail.py $O/$v.py 30 > $O/sf-$v.game 2> $O/sf-$v.err; echo "$v game rc $? stdout-lines $(wc -l < $O/sf-$v.game) stderr: $(tail -1 $O/sf-$v.err | cut -c1-110)"
  head -1 $O/sf-$v.game | cut -c1-60
  python3 -I $O/$v.py cmp $O/sf-$v.game $O/d2d30.txt; echo "$v cmp rc $?"
done
echo "== comment 2: game side with no equal trial (every S line ends 0), 30 trials vs d2d with 30 unequal trials"
awk 'BEGIN{for(i=0;i<30;i++) print "S", i, 0, "", 0}' > $O/zero.game
awk 'BEGIN{for(i=0;i<30;i++) print "R", i, 10, 20, 30}' > $O/unequal.d2d
for v in old new; do python3 -I $O/$v.py cmp $O/zero.game $O/unequal.d2d; echo "$v cmp rc $?"; done
echo "== control: real 400 vs 400 (fireball, glacialspike) from the bundle-3 run files, new cmp"
for n in fireball glacialspike; do python3 -I $O/new.py cmp $C/roll-$n.game $C/roll-$n.d2d; echo "$n new cmp rc $?"; done
