#!/bin/bash
# nsbuild.sh SRC BUILD TARGETS...: run ninja -j1 nice on my BUILD (a copy of verifyplayerroll build-base, a clean build of 6935cf64)
# with my SRC tree bind-mounted at the paths the copy was configured for (private mount namespace, nothing outside changes).
src=$1; bld=$2; shift 2
P=~/d2d-lab/verifyplayerroll
exec unshare -rm bash -c "mount --bind $src $P/base && mount --bind $bld $P/build-base && cd $P/build-base && nice ninja -j1 $*"
