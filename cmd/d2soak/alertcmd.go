package main

import (
	"context"
	"os"
	"os/exec"
	"time"
)

// alertCmdEnv names the environment variable that holds the command run on every NEW fail alert. Unset (the default)
// means nothing is ever run or contacted: the soak only writes files. The command is a shell line run with `sh -c`; it
// gets D2SOAK_ALERT_LEVEL (fail), _CODE, _MSG, _SINCE (RFC 3339), _RUN (the run directory) and _STATUS (the one-line
// status). It has 30 s, its output goes to the sidecar's stderr, and a failure is only logged.
const alertCmdEnv = "D2SOAK_ALERT_CMD"

const alertCmdTimeout = 30 * time.Second

func (w *liveWriter) runHook(a Alert, status string) {
	if w.cmd == "" {
		return
	}
	w.wait.Add(1)
	go func() {
		defer w.wait.Done()
		ctx, cancel := context.WithTimeout(context.Background(), alertCmdTimeout)
		defer cancel()
		c := exec.CommandContext(ctx, "sh", "-c", w.cmd)
		c.Env = append(os.Environ(), "D2SOAK_ALERT_LEVEL=fail", "D2SOAK_ALERT_CODE="+a.Code, "D2SOAK_ALERT_MSG="+a.Msg,
			"D2SOAK_ALERT_SINCE="+a.Since.Format(time.RFC3339), "D2SOAK_ALERT_RUN="+w.dir, "D2SOAK_ALERT_STATUS="+status)
		c.Stdout, c.Stderr = os.Stderr, os.Stderr
		if err := c.Run(); err != nil {
			w.log("d2soak: alert command: %v", err)
		}
	}()
}
