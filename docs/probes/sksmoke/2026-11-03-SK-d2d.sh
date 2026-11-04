#!/bin/bash
# SK d2d run helper (card sksmoke-a1). Runs ON THE WORKER from the synced worktree. Usage: 2026-11-03-SK-d2d.sh SLOT 'probe verbs'
export PATH=$PATH:$HOME/.local/bin
B=$HOME/d2d-lab/fork/bin/${SK_BIN:-d2d.lab-debug-skills}
[ -x /tmp/sksmoke-d2drill ] || go build -o /tmp/sksmoke-d2drill ./cmd/d2drill || exit 1
D2D_AGENT=sksmoke D2D_BIN=$B tools/agentrun.sh sksmoke --ttl "${SK_TTL:-600}" -- tools/d2dslot.sh run -n "$1" -t "${SK_TTL:-600}" -- /tmp/sksmoke-d2drill -group Q -only NONE -bin "$B" -probe "$2"
