#!/bin/bash
# decomp8 (worker host): d8_ss.sh over a list of levels. usage: d8_ss_all.sh FIRST-LAST "levels" kind...
HERE=$(cd "$(dirname "$0")" && pwd); R=$1; L=$2; shift 2
for l in $L; do echo "== level $l"; sh $HERE/d8_ss.sh $R $l "$@" 2>&1 | grep -v -E "WARNING" | cut -c1-200 | tail -6; done
