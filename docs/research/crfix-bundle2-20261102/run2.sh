O=~/d2d-lab/crfix2; B=~/d2d-lab/crfix/build-fix/tests
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
export PYTHONSAFEPATH=1 PYTHONPATH=$HOME/d2d-lab/src/tools/emu; R=$HOME/d2d-lab/fork/wt-crfix4/tools/emu/sk_ground_roll.py; cd ~/d2d-lab/src/tools/emu
for n in fireball glacialspike; do
  nice -n 19 $HOME/d2d-lab/pr/walk2-venv/bin/python $R game $n 400 > $O/roll-$n.game 2>$O/roll-$n.gerr; echo "game $n rc $?"
  ( cd $B && nice -n 19 ./test_sk_ground --burst-roll $n 400 > $O/roll-$n.d2d 2>/dev/null ); echo "d2d $n rc $?"
  python3 -I $R cmp $O/roll-$n.game $O/roll-$n.d2d; echo "cmp $n rc $?"
done
bash $HOME/d2d-lab/crfix/suites.sh
echo ALLDONE
