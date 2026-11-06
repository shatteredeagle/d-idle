export PATH=$PATH:$HOME/.local/bin
B=$HOME/d2d-lab/fork/bin/d2d.lab3-labverbs-f75816c6-linuxbuild
ls -la $B; sha256sum $B | cut -c1-16
go build -o /tmp/shortdrills-d2drill ./cmd/d2drill || exit 1
uptime; tools/d2dslot.sh status
D2D_AGENT=shortdrills D2D_BIN=$B tools/agentrun.sh shortdrills --ttl 260 -- tools/d2dslot.sh run -n 4 -w 30 -t 240 -- /tmp/shortdrills-d2drill -probe "debug help;debug merc;debug merc hire 0;dump merc" -bin $B 2>&1
