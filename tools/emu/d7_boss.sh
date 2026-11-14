#!/bin/bash
# decomp7 (worker host): TASK-86 boss death/level oracles on upstream main + the four bosses' skill columns (tables D).
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
T=$HOME/d2d-lab/wt-decomp6 B=$HOME/d2d-lab/build-decomp6 O=$HOME/d2d-lab/d7/boss; mkdir -p $O
cd $T/tools/emu; git -C $T log --oneline -1
echo "== boss_deaths"; uv run python boss_deaths.py 2>&1 | grep -v WARN | tail -4
ninja -C $B test_boss_level >/dev/null 2>&1
echo "== boss_level"; uv run python boss_level.py gen 100000 20261109 > $O/bl.tsv && uv run python boss_level.py game $O/bl.tsv > $O/bl.g && $B/tests/test_boss_level --boss-level $O/bl.tsv > $O/bl.d && uv run python boss_level.py compare $O/bl.tsv $O/bl.g $O/bl.d 2>&1 | grep -v WARN | tail -8
echo "== skill columns (tables D)"
python3 - <<'PY'
import csv,os
rows=list(csv.reader(open(os.path.expanduser('~/mmtab/monstats.txt'),encoding='latin1'),delimiter='\t'))
h=rows[0]
want=('andariel','bloodraven','countess','corpsefire')
cols=['Id','hcIdx','AI','Skill1','Sk1mode','Sk1lvl','Skill2','Sk2mode','Sk2lvl','Skill3','Sk3mode','Sk3lvl','Skill4','Sk4mode','Sk4lvl','Skill5','Sk5mode','Sk5lvl','MissA1','MissA2','MissS1']
ix=[h.index(c) for c in cols if c in h]
print('\t'.join(h[i] for i in ix))
for r in rows[1:]:
    if r and r[0].lower() in want: print('\t'.join(r[i] for i in ix))
PY
