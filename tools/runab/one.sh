#!/bin/bash
# usage: one.sh TAG ARM(on|off) SEED HOURS   (ON THE WORKER; change runab)
# Runs inside one d2dslot: fresh uncheated Sorceress "Loopy-ab", lab6, lockstep, binary d2bot.runab, private dir ~/d2d-lab/runab.
# Writes the pid of its own session/process group leader to ~/d2d-lab/runab/pids/TAG.pid; stop the whole run with: kill -TERM -- -$(cat that file)
R=$HOME/d2d-lab/runab; mkdir -p $R/pids; cd $R
export RA_T=$1 RA_ARM=$2 RA_SEED=$3 RA_H=$4
setsid bash -c '
echo $$ > $HOME/d2d-lab/runab/pids/$RA_T.pid
exec $HOME/d2d-lab/d2dslot.sh run -w 3000 -- bash -c "
 cd \$HOME/d2d-lab/runab
 export D2BOT_INSTANCE=ra-\$(echo \$RA_T | tr A-Z a-z) D2DSOAK_HOURS=\$RA_H D2D_BIN=\$HOME/d2d-lab/fork/bin/d2d.lab6 D2D_DATA=\$HOME/d2d-lab/data
 [ \$RA_ARM = off ] && export D2D_RUN=off
 rm -rf data-\$RA_T; mkdir -p data-\$RA_T
 nice -n 10 ./d2bot.runab --backend d2d --managed --d2d-class 4 --char Loopy-ab --seed \$RA_SEED \
  --toggle autoloot=off,autoparty=off --d2d-args --lockstep --d2d-linkwait 60s \
  --gamedir \$HOME/d2d-lab/data --data \$HOME/d2d-lab/runab/data-\$RA_T --d2d-sock \$D2D_SOCK \
  --addr \$D2D_BOT_ADDR --goals campaign --run-for 40m > bot-\$RA_T.log 2>&1
 echo exit \\\$? >> bot-\$RA_T.log"
' > $R/launch-$RA_T.out 2>&1 < /dev/null &
sleep 1; echo launched $RA_T
