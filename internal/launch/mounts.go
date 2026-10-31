package launch

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Wine stalls at startup while a network mount it can see does not answer (V: G1 attempts 2 and 3, the NAS rebooting,
// The worker's /mnt/media 'Host is down': Game.exe started but never made a window). The cause is not in the prefix, so a
// launch must name it instead of reporting an unexplained "no window".

// MountOpts are the injectable parts; the zero value is the real system.
type MountOpts struct {
	MountsFile string                                   // default /proc/mounts
	Stat       func(ctx context.Context, p string) bool // true = answered; default `timeout 3 stat`
}

// NetMounts returns the mount points of cifs/smb3/smbfs/nfs file systems listed in the mounts file.
func NetMounts(file string) []string {
	if file == "" {
		file = "/proc/mounts"
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		f := strings.Fields(l)
		if len(f) < 3 {
			continue
		}
		switch f[2] {
		case "cifs", "smb3", "smbfs", "nfs", "nfs4":
			out = append(out, unescapeMount(f[1]))
		}
	}
	return out
}

// unescapeMount undoes the octal escapes of /proc/mounts (a space is \040).
func unescapeMount(s string) string {
	return strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`).Replace(s)
}

// HungMounts returns the network mounts that do not answer a 3 s stat.
func HungMounts(ctx context.Context, o MountOpts) []string {
	if o.Stat == nil {
		o.Stat = func(ctx context.Context, p string) bool {
			c, cancel := context.WithTimeout(ctx, 5*time.Second) // hard bound if `timeout` itself is stuck in D state
			defer cancel()
			return exec.CommandContext(c, "timeout", "3", "stat", "--", p).Run() == nil
		}
	}
	var bad []string
	for _, m := range NetMounts(o.MountsFile) {
		if !o.Stat(ctx, m) {
			bad = append(bad, m)
		}
	}
	return bad
}

// hungReason is the sentence a failed launch appends, "" when all network mounts answer.
func hungReason(ctx context.Context, o MountOpts) string {
	if bad := HungMounts(ctx, o); len(bad) > 0 {
		return "mount " + strings.Join(bad, ", ") + " is hung (host down): Wine will stall"
	}
	return ""
}
