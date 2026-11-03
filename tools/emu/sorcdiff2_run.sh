#!/bin/bash
# sorcdiff2 (agent SORCDIFF2): stages run INSIDE the Ubuntu-26.04 WSL distro (linux.ps1 -Script sorc2-run.sh LABELS...).
# Source: C:\d2lab\dl\sorc2.bundle (refs/heads/*). Branch local/sorc2-work carries the scratch files and tools/emu oracles.
# Stages (labels, ':' separates arguments): dis | emu:SCRIPT[:ARG...] | mana
set -u
W=/mnt/c/d2lab; H=$HOME/d2d-lab; R=$H/repo-sorc2; S=$H/sorc2-scratch
export D2_MPQ_DIR=$HOME/gamedata D2_PATCH_INSTALLER=$HOME/gamedata/LODPatch_114d.exe
PY=$HOME/venv/bin/python
mkdir -p "$S" "$R"; [ -d "$R/.git" ] || git init -q "$R"
git -C "$R" fetch -q "$W/dl/sorc2.bundle" '+refs/heads/*:refs/heads/*' || exit 2
WORK=local/sorc2-work
T=$H/wt-sorc2
if [ -d "$T" ]; then git -C "$T" checkout -q -f --detach "$WORK" || exit 3; else git -C "$R" worktree add -q --detach "$T" "$WORK" || exit 3; fi
t0=$SECONDS
EMU="D2_BUILD=$H/up/build PYTHONPATH=$T/tools/emu"
emu() { (cd "$T/tools/emu" && env $EMU $PY "$@"); }
mk() { # name ref targets...
  n=$1; ref=$2; shift 2; TT=$H/wt-sorc2-$n; B=$H/build-sorc2-$n; D=$H/src/build/_deps
  if [ -d "$TT" ]; then git -C "$TT" checkout -q -f --detach "$ref" || return 1; else git -C "$R" worktree add -q --detach "$TT" "$ref" || return 1; fi
  [ -f "$B/build.ninja" ] || cmake -S "$TT" -B "$B" -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF -DD2D_BUNDLED_FFMPEG=OFF \
    -DCMAKE_CXX_COMPILER_LAUNCHER=ccache -DCMAKE_C_COMPILER_LAUNCHER=ccache -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src \
    -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > "$B.cmake.log" 2>&1 || { tail -8 "$B.cmake.log"; return 1; }
  ninja -C "$B" -j4 "$@" > "$B.build.log" 2>&1; echo "build $n $(git -C "$TT" rev-parse --short HEAD) rc=$? ($((SECONDS-t0))s)"; grep -E "error|FAILED" "$B.build.log" | head -12
}
bin() { ls "$1"/tests/"$2" "$1"/tests/*/"$2" 2>/dev/null | head -1; }
for stage in "$@"; do case $stage in
dis) # every line of tools/emu/sorc2_dis.txt: ADDR [COUNT]
  while read -r a n; do [ -z "$a" ] && continue; echo "=== dis $a $n"; (cd $T/tools/emu && D2_BUILD=$H/up/build $PY disasm.py $a $n 2>&1); done < "$T/tools/emu/sorc2_dis.txt";;
emu:*) s=$(echo "${stage#emu:}" | tr : ' '); echo "=== emu $s ($((SECONDS-t0))s)"; emu $s 2>&1;;
mana) # cast_mana: A = harness only, B = + fix (local/sorc-mana-timing-r2~1 and its tip)
  mk mA local/sorc-mana-timing-r2~2 test_cast_mana mpq-cat; mk mB local/sorc-mana-timing-r2~1 test_cast_mana mpq-cat; mk mC local/sorc-mana-timing-r2 test_cast_mana mpq-cat
  O=$S/mana; mkdir -p $O
  emu cast_mana.py gen 1 > $O/cases.tsv 2>$O/gen.err; echo "cases $(wc -l < $O/cases.tsv) ($((SECONDS-t0))s)"; tail -2 $O/gen.err
  emu cast_mana.py game $O/cases.tsv > $O/game.tsv 2>$O/game.err; echo "game lines $(wc -l < $O/game.tsv) ($((SECONDS-t0))s)"; tail -3 $O/game.err
  for n in mA mB mC; do "$(bin $H/build-sorc2-$n test_cast_mana)" --cast-mana $O/cases.tsv > $O/d2d_$n.raw 2>$O/d2d_$n.err; grep -E '^[0-9]+ ' $O/d2d_$n.raw > $O/d2d_$n.tsv
    echo "$n: d2d lines $(wc -l < $O/d2d_$n.tsv) differing $(diff $O/game.tsv $O/d2d_$n.tsv | grep -c '^<') ($((SECONDS-t0))s)"; tail -2 $O/d2d_$n.err; done
  for n in mA mB mC; do echo "== first differing lines $n (game <, d2d >)"; diff $O/game.tsv $O/d2d_$n.tsv | head -8; done
  echo "== test_cast_mana (no arg)"; for n in mA mB mC; do "$(bin $H/build-sorc2-$n test_cast_mana)" 2>&1 | tail -2; echo "$n rc=$?"; done
  for f in cases game d2d_mA d2d_mB d2d_mC; do cp $O/$f.tsv $W/logs/mana-$f.tsv; done;;
bolts) # cast_bolts: the harness branch tip
  mk bA local/sorc-bolts test_cast_bolts mpq-cat; O=$S/bolts; mkdir -p $O; X=$H/wt-sorc2-bA/tools/emu
  (cd $X && env D2_BUILD=$H/up/build PYTHONPATH=$X $PY cast_bolts.py gen 1 > $O/cases.tsv 2>$O/gen.err; echo "cases $(wc -l < $O/cases.tsv)"
   env D2_BUILD=$H/up/build PYTHONPATH=$X $PY cast_bolts.py game $O/cases.tsv > $O/game.tsv 2>$O/game.err; echo "game lines $(wc -l < $O/game.tsv) ($((SECONDS-t0))s)"; tail -3 $O/game.err)
  "$(bin $H/build-sorc2-bA test_cast_bolts)" --cast-bolts $O/cases.tsv 2>$O/d2d.err | grep -E '^[0-9]+ ' > $O/d2d.tsv
  echo "bolts: d2d lines $(wc -l < $O/d2d.tsv) differing $(diff $O/game.tsv $O/d2d.tsv | grep -c '^<')"; diff $O/game.tsv $O/d2d.tsv | head -10
  "$(bin $H/build-sorc2-bA test_cast_bolts)" 2>&1 | tail -2; for f in cases game d2d; do cp $O/$f.tsv $W/logs/bolts-$f.tsv; done;;
armor) # armor_group: A = harness only, B = + fix
  mk gA local/sorc-armor-group-r2~1 test_armor_group mpq-cat; mk gB local/sorc-armor-group-r2 test_armor_group mpq-cat; O=$S/armor; mkdir -p $O; X=$H/wt-sorc2-gA/tools/emu
  (cd $X && env D2_BUILD=$H/up/build PYTHONPATH=$X $PY armor_group.py game > $O/game.tsv 2>$O/game.err; echo "game lines $(wc -l < $O/game.tsv)"; tail -3 $O/game.err)
  for n in gA gB; do "$(bin $H/build-sorc2-$n test_armor_group)" --armor-group 2>/dev/null | grep -E '^[0-9]+ ' > $O/d2d_$n.tsv; echo "$n: d2d lines $(wc -l < $O/d2d_$n.tsv) differing $(diff $O/game.tsv $O/d2d_$n.tsv | grep -c '^<')"; diff $O/game.tsv $O/d2d_$n.tsv | head -6
    "$(bin $H/build-sorc2-$n test_armor_group)" 2>&1 | tail -2; done; for f in game d2d_gA d2d_gB; do cp $O/$f.tsv $W/logs/armor-$f.tsv; done;;
esac; done
echo "sorc2-run done $((SECONDS-t0))s"
