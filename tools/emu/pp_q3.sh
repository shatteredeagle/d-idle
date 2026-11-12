cd $HOME/d2d-lab/wt-pp-light
grep -n "srv_do\|srvdo\|hit_func" components/game/gamedata/gamedata.hpp | head -8
grep -n "SK6_REVERSE" -r tools/emu | head
grep -n "2,584\|2584" -r docs/research/re/missiles.md tools/emu/README.md | head
cd $HOME/d2d-lab/fork/src && git fetch upstream 2>&1 | tail -1; git log upstream/main --oneline -1
