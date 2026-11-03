#!/bin/bash
# sorcdiff (agent SORCDIFF): Sorceress differentials on the real d2d build. Runs INSIDE the Ubuntu-26.04 WSL distro as user d2
# (linux.ps1 -Script sorc-run.sh). Input: C:\d2lab\dl\sorc.bundle (branches local/sorc-base = upstream 005c7cab, local/sorc-lightning-seq,
# whose parent is the harness-only commit). Stages as labels: build | cast | harness | harness2 (the later harnesses with 20,000 cases) (any order, several allowed). Output copy:
# C:\d2lab\logs\sorc-results.txt.
set -u
W=/mnt/c/d2lab; H=$HOME/d2d-lab; R=$H/repo-sorc; S=$H/sorc-scratch; D=$H/src/build/_deps
export D2_MPQ_DIR=$HOME/gamedata D2_PATCH_INSTALLER=$HOME/gamedata/LODPatch_114d.exe
PY=$HOME/venv/bin/python
mkdir -p "$S" "$R"; [ -d "$R/.git" ] || git init -q "$R"
git -C "$R" fetch -q "$W/dl/sorc.bundle" '+refs/heads/*:refs/heads/*' || exit 2
HEADREF=local/sorc-lightning-seq
t0=$SECONDS
mk() { # name ref
  T=$H/wt-sorc-$1; B=$H/build-sorc-$1
  if [ -d "$T" ]; then git -C "$T" checkout -q -f --detach "$2" || return 1; else git -C "$R" worktree add -q --detach "$T" "$2" || return 1; fi
  [ -f "$B/build.ninja" ] || cmake -S "$T" -B "$B" -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF -DD2D_BUNDLED_FFMPEG=OFF \
    -DCMAKE_CXX_COMPILER_LAUNCHER=ccache -DCMAKE_C_COMPILER_LAUNCHER=ccache -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src \
    -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > "$B.cmake.log" 2>&1 || { tail -8 "$B.cmake.log"; return 1; }
  tg="test_cast_seq test_combat"; [ "$1" = A ] && tg="$tg test_game test_missile_hit test_next_hit test_static_field test_armor_hit test_cold_length test_freeze_rules mpq-cat"
  [ "$1" = B ] && tg="$tg mpq-cat"
  ninja -C "$B" -j8 $tg > "$B.build.log" 2>&1; echo "build $1 $(git -C "$T" rev-parse --short HEAD) rc=$? ($((SECONDS-t0))s)"; grep -E "error|FAILED" "$B.build.log" | head -12
}
bin() { ls "$1"/tests/"$2" "$1"/tests/*/"$2" 2>/dev/null | head -1; }
oracle() { # WT script args... ; the emulator reads mpq-cat from the build dir of the same name
  wt=$1; bd=$2; shift 2
  ( cd "$wt/tools/emu" && D2_BUILD=$bd PYTHONPATH=$wt/tools/emu nice "$PY" "$@" )
}
cmpj() { # CASES GAME D2D [skill ids]: per-skill totals of a differential whose lines start with the case id
  "$PY" - "$@" <<'PE'
import sys, collections
cases, game, d2d = sys.argv[1:4]
want = set(sys.argv[4:])
skill = {}
for l in open(cases):
    f = l.split()
    if f: skill[f[0]] = f[1] if len(f) > 1 else "?"
g = {l.split()[0]: l.split(None, 1)[1].strip() for l in open(game) if l.strip() and l.split()[0] in skill}
d = {l.split()[0]: l.split(None, 1)[1].strip() for l in open(d2d) if l.strip()[:1].isdigit() and l.split()[0] in skill}
tot = bad = 0; per = collections.Counter(); perbad = collections.Counter()
for cid, a in g.items():
    b = d.get(cid)
    if b is None: continue
    tot += 1; per[skill[cid]] += 1
    if a != b: bad += 1; perbad[skill[cid]] += 1
print(f"cases {len(skill)} game {len(g)} compared {tot} mismatches {bad}")
for s in sorted(want or per, key=lambda x: int(x) if x.isdigit() else 0):
    print(f"  skill {s}: compared {per[s]} mismatches {perbad[s]}")
PE
}
OUT=$S/results.txt; : > "$OUT"
say() { echo "$@" | tee -a "$OUT"; }
for stage in "$@"; do case $stage in
build) mk A "$HEADREF~1"; mk B "$HEADREF";;
cast)
  A=$H/wt-sorc-A; B=$H/wt-sorc-B; BA=$H/build-sorc-A; BB=$H/build-sorc-B
  say "== cast_rates.py default vs test_combat --cast-rates ($((SECONDS-t0))s)"
  oracle $A $BA cast_rates.py > $S/cr_game.txt 2>$S/cr_game.err; say "game lines $(wc -l < $S/cr_game.txt)"
  for n in A B; do bd=$H/build-sorc-$n; "$(bin $bd test_combat)" --cast-rates > $S/cr_$n.txt; say "$n: differing lines $(diff $S/cr_game.txt $S/cr_$n.txt | grep -c '^[<>]')"; done
  say "== cast_rates.py cast vs test_cast_seq --cast-ticks (FCR 0..400, 9 skills) ($((SECONDS-t0))s)"
  oracle $A $BA cast_rates.py cast > $S/cc_game.txt 2>$S/cc_game.err; say "game lines $(wc -l < $S/cc_game.txt)"
  for n in A B; do bd=$H/build-sorc-$n; "$(bin $bd test_cast_seq)" --cast-ticks > $S/cc_$n.txt 2>$S/cc_$n.err; say "$n (d2d lines $(wc -l < $S/cc_$n.txt)): mismatching lines $(diff $S/cc_game.txt $S/cc_$n.txt | grep -c '^<')"
    awk 'NR==FNR{g[$2" "$3]=$4;next} {k=$2" "$3; if(g[k]!=$4){b[$2]++}} END{for(s in b) printf "   skill %s: %d differ\n", s, b[s]}' $S/cc_game.txt $S/cc_$n.txt | sort | tee -a "$OUT"; done
  for n in A B; do bd=$H/build-sorc-$n; say "$n test_cast_seq (no arg): $("$(bin $bd test_cast_seq)" 2>&1 | tail -2 | tr '\n' ' ') rc=$?"; say "$n test_combat (no arg): $("$(bin $bd test_combat)" 2>&1 | tail -1) rc=$?"; done;;
harness)
  A=$H/wt-sorc-A; BA=$H/build-sorc-A
  say "== skill_damage gen 100000 1 ($((SECONDS-t0))s)"
  oracle $A $BA skill_damage.py gen 100000 1 > $S/sd_cases.tsv; oracle $A $BA skill_damage.py game $S/sd_cases.tsv > $S/sd_game.tsv 2>$S/sd_game.err
  timeout 1500 "$(bin $BA test_game)" --skill-damage $S/sd_cases.tsv | grep -E '^[0-9]+ ' > $S/sd_d2d.tsv; cmpj $S/sd_cases.tsv $S/sd_game.tsv $S/sd_d2d.tsv 36 38 40 42 43 44 49 53 54 57 58 | tee -a "$OUT"
  say "== missile_hit gen 60000 1 ($((SECONDS-t0))s)"
  oracle $A $BA missile_hit.py gen 60000 1 > $S/mh_cases.tsv; oracle $A $BA missile_hit.py game $S/mh_cases.tsv > $S/mh_game.tsv 2>$S/mh_game.err
  timeout 1500 "$(bin $BA test_missile_hit)" --missile-hit $S/mh_cases.tsv | grep -E '^[0-9]+ ' > $S/mh_d2d.tsv; cmpj $S/mh_cases.tsv $S/mh_game.tsv $S/mh_d2d.tsv | tee -a "$OUT"
  for sd in 1 7; do
  say "== static_field.py 100000 $sd ($((SECONDS-t0))s)"
  oracle $A $BA static_field.py 100000 $sd > $S/sf_game_$sd.txt 2>$S/sf_game.err
  timeout 1500 "$(bin $BA test_static_field)" --static-field $sd 100000 | grep -E '^[BR][0-9]' > $S/sf_d2d_$sd.txt; say "game $(wc -l < $S/sf_game_$sd.txt) d2d $(wc -l < $S/sf_d2d_$sd.txt) differing $(diff $S/sf_game_$sd.txt $S/sf_d2d_$sd.txt | grep -c '^<')"
  say "== chill_length.py 100000 $sd ($((SECONDS-t0))s)"
  oracle $A $BA chill_length.py 100000 $sd > $S/cl_game_$sd.txt 2>$S/cl_game.err
  timeout 1500 "$(bin $BA test_cold_length)" --cold-length $sd 100000 | grep -E '^[BR][0-9]' > $S/cl_d2d_$sd.txt; say "game $(wc -l < $S/cl_game_$sd.txt) d2d $(wc -l < $S/cl_d2d_$sd.txt) differing $(diff $S/cl_game_$sd.txt $S/cl_d2d_$sd.txt | grep -c '^<')"
  done
  say "== next_hit.py (Frost Nova / Nova hits), seeds 1 and 7, N 3000 ($((SECONDS-t0))s)"
  for sd in 1 7; do timeout 1500 "$(bin $BA test_next_hit)" --next-hit $sd 3000 > $S/nh_d2d_$sd.txt; oracle $A $BA next_hit.py $S/nh_d2d_$sd.txt > $S/nh_game_$sd.txt 2>$S/nh_game.err
    say "seed $sd: d2d H/T lines $(grep -cE '^[HT]' $S/nh_d2d_$sd.txt) game $(wc -l < $S/nh_game_$sd.txt) differing $(grep -E '^[HT]' $S/nh_d2d_$sd.txt | diff $S/nh_game_$sd.txt - | grep -c '^<')"; done
  say "== armor_hit (Frozen Armor striker) gen 100000 1 ($((SECONDS-t0))s)"
  oracle $A $BA armor_hit.py gen 100000 1 > $S/ah_cases.tsv; oracle $A $BA armor_hit.py game $S/ah_cases.tsv > $S/ah_game.tsv 2>$S/ah_game.err
  timeout 1500 "$(bin $BA test_armor_hit)" --armor-hit $S/ah_cases.tsv > $S/ah_d2d.tsv; oracle $A $BA armor_hit.py compare $S/ah_cases.tsv $S/ah_game.tsv $S/ah_d2d.tsv 2>&1 | tail -6 | tee -a "$OUT"
  say "== freeze_rules gen 100000 1 (Frost Nova / Glacial Spike freeze) ($((SECONDS-t0))s)"
  oracle $A $BA freeze_rules.py gen 100000 1 > $S/fr_cases.tsv; oracle $A $BA freeze_rules.py game $S/fr_cases.tsv > $S/fr_game.tsv 2>$S/fr_game.err
  timeout 1500 "$(bin $BA test_freeze_rules)" --freeze-rules $S/fr_cases.tsv | grep -E '^[0-9]+ ' > $S/fr_d2d.tsv; cmpj $S/fr_cases.tsv $S/fr_game.tsv $S/fr_d2d.tsv | tee -a "$OUT";;
harness2)
  A=$H/wt-sorc-A; BA=$H/build-sorc-A
  for sd in 1 7; do
  say "== static_field.py 20000 $sd ($((SECONDS-t0))s)"
  oracle $A $BA static_field.py 20000 $sd > $S/sf_game_$sd.txt 2>$S/sf_game.err
  timeout 1500 "$(bin $BA test_static_field)" --static-field $sd 20000 | grep -E '^[BR][0-9]' > $S/sf_d2d_$sd.txt; say "game $(wc -l < $S/sf_game_$sd.txt) d2d $(wc -l < $S/sf_d2d_$sd.txt) differing $(diff $S/sf_game_$sd.txt $S/sf_d2d_$sd.txt | grep -c '^<')"
  say "== chill_length.py 20000 $sd ($((SECONDS-t0))s)"
  oracle $A $BA chill_length.py 20000 $sd > $S/cl_game_$sd.txt 2>$S/cl_game.err
  timeout 1500 "$(bin $BA test_cold_length)" --cold-length $sd 20000 | grep -E '^[BR][0-9]' > $S/cl_d2d_$sd.txt; say "game $(wc -l < $S/cl_game_$sd.txt) d2d $(wc -l < $S/cl_d2d_$sd.txt) differing $(diff $S/cl_game_$sd.txt $S/cl_d2d_$sd.txt | grep -c '^<')"
  done
  say "== next_hit.py (Frost Nova / Nova hits), seeds 1 and 7, N 1500 ($((SECONDS-t0))s)"
  for sd in 1 7; do timeout 1500 "$(bin $BA test_next_hit)" --next-hit $sd 1500 > $S/nh_d2d_$sd.txt; oracle $A $BA next_hit.py $S/nh_d2d_$sd.txt > $S/nh_game_$sd.txt 2>$S/nh_game.err
    say "seed $sd: d2d H/T lines $(grep -cE '^[HT]' $S/nh_d2d_$sd.txt) game $(wc -l < $S/nh_game_$sd.txt) differing $(grep -E '^[HT]' $S/nh_d2d_$sd.txt | diff $S/nh_game_$sd.txt - | grep -c '^<')"; done
  say "== armor_hit (Frozen Armor striker) gen 20000 1 ($((SECONDS-t0))s)"
  oracle $A $BA armor_hit.py gen 20000 1 > $S/ah_cases.tsv; oracle $A $BA armor_hit.py game $S/ah_cases.tsv > $S/ah_game.tsv 2>$S/ah_game.err
  timeout 1500 "$(bin $BA test_armor_hit)" --armor-hit $S/ah_cases.tsv > $S/ah_d2d.tsv; oracle $A $BA armor_hit.py compare $S/ah_cases.tsv $S/ah_game.tsv $S/ah_d2d.tsv 2>&1 | tail -6 | tee -a "$OUT"
  say "== freeze_rules gen 20000 1 (Frost Nova / Glacial Spike freeze) ($((SECONDS-t0))s)"
  oracle $A $BA freeze_rules.py gen 20000 1 > $S/fr_cases.tsv; oracle $A $BA freeze_rules.py game $S/fr_cases.tsv > $S/fr_game.tsv 2>$S/fr_game.err
  timeout 1500 "$(bin $BA test_freeze_rules)" --freeze-rules $S/fr_cases.tsv | grep -E '^[0-9]+ ' > $S/fr_d2d.tsv; cmpj $S/fr_cases.tsv $S/fr_game.tsv $S/fr_d2d.tsv | tee -a "$OUT";;
esac; done
cp "$OUT" $W/logs/sorc-results.txt 2>/dev/null
echo "sorc-run done $((SECONDS-t0))s"
