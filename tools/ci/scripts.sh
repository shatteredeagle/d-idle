#!/usr/bin/env bash
# bash -n over every tracked shell script, then the offline script tests.
# Usage: tools/ci/scripts.sh
set -u
cd "$(dirname "$0")/../.."
fail=0
for f in $(git ls-files '*.sh'); do
  bash -n "$f" || { echo "FAIL: bash -n $f"; fail=1; }
done
bash tools/ci/gamecode_test.sh || { echo "FAIL: gamecode_test.sh"; fail=1; }
bash tools/winereap_test.sh || { echo "FAIL: winereap_test.sh"; fail=1; }
# fake-mode HTTP test: host load is irrelevant, and d2fleet admission would refuse the slot on a busy host
D2FLEET_MAX_LOAD_FACTOR=-1 bash tools/container-checklist.sh || { echo "FAIL: container-checklist.sh"; fail=1; }
bash packaging/release.sh --dry-run ci || { echo "FAIL: release.sh --dry-run"; fail=1; }
python3 -m unittest discover -s tools/ci/tests -t tools/ci || { echo "FAIL: tools/ci python tests"; fail=1; }
python3 -m unittest discover -s tools/discord/tests -t tools/discord -p "test_tick_post.py" || { echo "FAIL: tick_post tests"; fail=1; }
python3 -m unittest discover -s tools/discord/tests -t tools/discord -p "test_board_feed.py" || { echo "FAIL: board_feed tests"; fail=1; }
python3 -m unittest discover -s tools/discord/tests -t tools/discord -p "test_work_channels.py" || { echo "FAIL: work_channels tests"; fail=1; }
python3 -m unittest discover -s tools/contrib/tests -t tools/contrib || { echo "FAIL: tools/contrib python tests"; fail=1; }
python3 tools/contrib/board.py --check || { echo "FAIL: work-board.md is stale or open-work.md is malformed"; fail=1; }
python3 -m unittest discover -s tools/credit -t tools/credit -p "test_credit.py" || { echo "FAIL: tools/credit python tests"; fail=1; }
python3 -m unittest discover -s tools/coverage/tests -t tools/coverage || { echo "FAIL: tools/coverage python tests"; fail=1; }
python3 tools/coverage/coverage.py --check || { echo "FAIL: docs/coverage inputs invalid or rule-coverage.md is stale (python3 tools/coverage/coverage.py --write)"; fail=1; }
exit $fail
