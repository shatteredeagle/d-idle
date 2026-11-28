package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/scenario"
)

// cutReason is the BLOCKED reason of a bot run that did not play its virtual budget to the end (killed by the wall
// limit or exited early without an error): its state is not what the scenario asked about, so it is not judged.
func cutReason(trace string) string {
	b, err := os.ReadFile(trace)
	if err != nil || scenario.RunLengthReached(b) {
		return ""
	}
	return "the bot run was cut before its virtual budget (no '" + scenario.RunLengthLine + "' in its log): not judged"
}

// fileSHA256 is the hex sha256 of a file, "" when unreadable.
func fileSHA256(p string) string {
	f, err := os.Open(p)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// collectProv gathers the provenance of a run: the bot's VCS stamp (go version -m; D2LOD_BOT_COMMIT when the binary
// carries none), the binary hashes, the host ROLE (D2D_HOST_ROLE, never a machine name) and the load.
func collectProv(bot string) *scenario.Provenance {
	p := &scenario.Provenance{BotSHA256: fileSHA256(bot), D2DSHA256: fileSHA256(os.Getenv("D2D_BIN")), Host: os.Getenv("D2D_HOST_ROLE")}
	if p.Host == "" {
		p.Host = "unknown"
	}
	if out, err := exec.Command("go", "version", "-m", bot).Output(); err == nil {
		p.BotCommit, p.BotDirty = scenario.ParseBuildInfo(string(out))
	}
	if p.BotCommit == "" { // a binary built without a VCS stamp (run.sh, for a reproducible hash): the runner says what it built
		p.BotCommit, p.BotDirty = os.Getenv("D2LOD_BOT_COMMIT"), os.Getenv("D2LOD_BOT_DIRTY") == "1"
	}
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		if f := strings.Fields(string(b)); len(f) > 0 {
			p.Load1, _ = strconv.ParseFloat(f[0], 64)
		}
	}
	return p
}

// The runner logs the start and end of every exec-one in <out>/slots.log and tells the run where the log is and which
// line is its own, so the run can record its peak slot load (the stability rule's "parallel 2", design section 2.7).
const (
	slotLogName = "slots.log"
	slotLogEnv  = "PG_SLOT_LOG"
	slotIDEnv   = "PG_SLOT_ID"
)

// slotParallel is the peak number of busy runs during this run, 0 when the run was not started by the runner.
func slotParallel() int {
	path, id := os.Getenv(slotLogEnv), os.Getenv(slotIDEnv)
	if path == "" || id == "" {
		return 0
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return scenario.PeakParallel(b, id)
}

// logSlotStart writes this run's start line into the runner's slot log (nothing when not started by the runner).
func logSlotStart(t time.Time) {
	if path, id := os.Getenv(slotLogEnv), os.Getenv(slotIDEnv); path != "" && id != "" {
		if err := appendRaw(path, scenario.SlotLogLine("start", id, t.UnixNano())); err != nil {
			fmt.Fprintf(os.Stderr, "d2scenario: slot log: %v\n", err)
		}
	}
}

// appendRaw appends text in one O_APPEND write.
func appendRaw(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(text)
	return err
}
