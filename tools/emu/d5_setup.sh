set -u
cd ~/d2d-lab/fp-scratch.git
git worktree add -f --detach ~/d2d-lab/wt-decomp5 FETCH_HEAD 2>&1 | tail -1
T=~/d2d-lab/wt-decomp5; B=~/d2d-lab/build-decomp5; D=$HOME/d2d-lab/src/build/_deps
mkdir -p $B; ln -sfn $B $T/build
nice -n 19 cmake -S $T -B $B -G Ninja -DCMAKE_BUILD_TYPE=Release -DD2D_WERROR=OFF \
 -DCMAKE_CXX_COMPILER_LAUNCHER=$HOME/.local/bin/ccache -DCMAKE_C_COMPILER_LAUNCHER=$HOME/.local/bin/ccache \
 -DFETCHCONTENT_SOURCE_DIR_CLI11=$D/cli11-src -DFETCHCONTENT_SOURCE_DIR_SENTRY_NATIVE=$D/sentry_native-src \
 -DFETCHCONTENT_SOURCE_DIR_STORMLIB=$D/stormlib-src > $B.cmake.log 2>&1 || { tail -20 $B.cmake.log; exit 1; }
nice -n 19 ninja -C $B -j2 test_game mpq-cat > $B.build.log 2>&1
echo build rc=$?; tail -2 $B.build.log
