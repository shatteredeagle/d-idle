package bot

import "time"

// d2bot SIGTERM shutdown budget (cmd/d2bot shutdownSeq). The worst case must stay below the fleet's stop grace, which
// stays below the container grace (TestStopGraceChain in deploy/container). A single-player Save and Exit from the
// field is fine, so SIGTERM does not wait for a long walk home.
const (
	SignalSettleWait = 10 * time.Second // finish-in-town wait on SIGTERM (a Stop from the UI keeps StopTimeoutSec)
	RunnerQuiesce    = 2 * time.Second  // cancelled runner may still send input; wait before the Esc menu
	LeaveTimeout     = 30 * time.Second // Save and Exit
	RunnerDoneWait   = 2 * time.Second
	APIDoneWait      = 2 * time.Second
	GameShutdownWait = 4 * time.Second

	ShutdownWorstCase = SignalSettleWait + RunnerQuiesce + LeaveTimeout + RunnerDoneWait + APIDoneWait + GameShutdownWait
)
