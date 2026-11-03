#!/bin/bash
R=~/d2d-lab/verifymonsterpace; P=~/d2d-lab/verifyplayerroll
exec unshare -rm bash -c "mount --bind $R/head $P/base && mount --bind $R/build-head $P/build-base && cd $P/base && while read f; do echo \"=== \$f\"; nice clang-tidy -p build --quiet --extra-arg=-Wno-unknown-warning-option \"\$f\" 2>&1; echo \"rc=\$?\"; done < $R/tidy_files2.txt" > $R/tidy-head.log 2>&1
