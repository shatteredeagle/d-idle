set -e
S=~/labrun/herdr/code/d2lod-wt/prwarp/tools/emu/pw; W=~/d2d-lab/wt-pw
cd $W
git checkout -q fid/warp-arrival-r2
python3 $S/patch_fix.py $W
cp $S/test_warp_arrival.cpp tests/game/test_warp_arrival.cpp
git diff --stat
git add -A
git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com commit -q -m "draft claim message, amended after the run

Agent: Claude Code"
git log --oneline -3
