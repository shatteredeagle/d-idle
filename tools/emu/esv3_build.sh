#!/bin/bash
# esverify3: build the base and head probes from ../verifier3-base and ../verifier3-head (clean build dirs), and test_combat on both.
set -e
here=$(cd "$(dirname "$0")" && pwd)
export ESV_TREE=verifier3
export D2_SCRATCH=${D2_SCRATCH:-/tmp/d2agent-esverify3}
for t in base head; do bash "$here/esv_build.sh" $t test_combat test_energy_shield test_game; done
