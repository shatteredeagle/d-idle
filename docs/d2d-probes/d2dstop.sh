#!/bin/bash
# stop d2d via its channel, then kill if still alive
python3 ~/d2d-lab/ctl.py save quit >/dev/null 2>&1; sleep 2; pkill -x d2d 2>/dev/null; sleep 1; pgrep -x d2d || echo stopped
