package env

import (
	"os"
	"strings"
)

// ReapEnabled is the knob for killing orphaned winedevice.exe after a managed game stop (internal/launch/reap.go,
// docs/soak.md). Default ON; D2BOT_REAP=0 (or off, false, no) disables it.
func ReapEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("D2BOT_REAP"))) {
	case "0", "off", "false", "no":
		return false
	}
	return true
}
