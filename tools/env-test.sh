#!/bin/sh
# Offline test of tools/env.sh d2_role_env. Run: sh tools/env-test.sh
here=$(cd "$(dirname "$0")" && pwd)
out=$(D2_BOT_DIR=/nonexistent D2_COLD_DIR=/c D2_WORKER_HOST=w.test D2_NAS_HOST=n.test sh -c ". '$here/env.sh'; d2_role_env; d2_role_env --setenv=")
want="D2_COLD_DIR='/c' D2_WORKER_HOST='w.test' D2_NAS_HOST='n.test' --setenv=D2_COLD_DIR='/c' --setenv=D2_WORKER_HOST='w.test' --setenv=D2_NAS_HOST='n.test' "
[ "$out" = "$want" ] && echo "ALL OK" || { echo "FAIL: $out"; exit 1; }
