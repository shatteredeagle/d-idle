set -e
S=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd); W=~/d2d-lab/wt-pw
cd $W
git branch -f fid/warp-arrival-r2-old fid/warp-arrival-r2
git checkout -q -B fid/warp-arrival-r2 6151f74e
cp $S/test_warp_arrival.v1.cpp tests/game/test_warp_arrival.cpp
git add -A
git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com commit -q --amend -m "tools: oracle for the spot a level warp lands on

sk_tele_spot.py warp runs game.exe's FUN_0064e7b0 and FUN_00554ea0 as FUN_005550b0 calls them; gen --warp makes the cases.
test_warp_arrival --warp-arrival prints where d2d lands with Level::nearest_free: 11,645 of 20,000 differ (seed 1), 11,249 (seed 31).

Agent: Claude Code"
python3 $S/patch_fix.py $W
cp $S/test_warp_arrival.cpp tests/game/test_warp_arrival.cpp
git add -A
git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com commit -q -m "A warp lands on the free subtile nearest its unit, as FUN_005550b0

World::warp_arrival takes Level::warp_spot (rings 1 to 49, least |dx| + |dy|, FUN_0064dea0) instead of nearest_free; no spot refuses the warp (0x5551c3).
Oracle: 20,000 cases, 11,645 and 11,249 differ before (seeds 1 and 31), 0 after; real Act 1 arrivals 25 to 26 of 52 before, 0 after (6 seeds).
Test: tests/game/test_warp_arrival.cpp; drlg.md \"Taking a warp\".

Agent: Claude Code"
git log --format='%h %an <%ae> %s' -3
git diff --stat HEAD~2 | tail -8
