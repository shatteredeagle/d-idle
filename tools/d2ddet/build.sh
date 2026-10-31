#!/bin/bash
# Build d2bot and d2ddet on the worker (labrun mirrors this worktree there) and put them in ~/d2d-lab/det together with
# pair.sh. usage: tools/d2ddet/build.sh [SUFFIX]   -> ~/d2d-lab/det/d2bot.det[SUFFIX], d2ddet, pair.sh
set -eu
repo="$(cd "$(dirname "$0")/../.." && pwd)"
suf="${1:-}"
. "$repo/tools/env.sh"
cd "$repo"
labrun "${D2_WORKER_HOST%%.*}" --no-pull -- bash -c "mkdir -p ~/d2d-lab/det && export PATH=\$HOME/.local/bin:\$HOME/go/bin:/usr/local/go/bin:\$PATH && go build -o ~/d2d-lab/det/d2bot.det$suf ./cmd/d2bot && go build -o ~/d2d-lab/det/d2ddet ./cmd/d2ddet && mkdir -p ~/d2d-lab/bin && cp tools/agentrun.sh tools/agentreap.sh ~/d2d-lab/bin/ && cp tools/d2ddet/pair.sh ~/d2d-lab/det/pair.sh && chmod +x ~/d2d-lab/det/pair.sh && ls -la ~/d2d-lab/det"
