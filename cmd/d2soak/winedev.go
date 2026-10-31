package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// WineCount is one winedevice.exe sample of an instance's Wine prefix. A winedevice whose parent is not a process
// of the same prefix (the dead wineserver's services.exe is gone, so the parent is init or a subreaper) is an
// orphan: about 35 MB and 15 threads each (docs/leakhunt.md "orphaned winedevice.exe", VERIFIED LIVE on i4).
// Whether a live prefix's winedevice always has an in-prefix parent is U: Total is safe either way.
type WineCount struct {
	Total   int `json:"total"`
	Orphans int `json:"orphans"`
}

// countWineDevices scans root (/proc) for winedevice.exe processes whose WINEPREFIX is one of prefixes (compared
// without trailing slashes, as tools/winereap.sh does). Unreadable processes are skipped, never counted.
func countWineDevices(root string, prefixes []string) WineCount {
	var c WineCount
	want := map[string]bool{}
	for _, p := range prefixes {
		if p = strings.TrimRight(p, "/"); p != "" {
			want[p] = true
		}
	}
	if len(want) == 0 {
		return c
	}
	ents, _ := os.ReadDir(root)
	for _, e := range ents {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || procComm(root, pid) != "winedevice.exe" || !want[procPrefix(root, pid)] {
			continue
		}
		c.Total++
		if pp := procPPID(root, pid); pp <= 1 || !want[procPrefix(root, pp)] {
			c.Orphans++
		}
	}
	return c
}

func procComm(root string, pid int) string {
	b, _ := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "comm"))
	return strings.TrimSpace(string(b))
}

func procPrefix(root string, pid int) string {
	b, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "environ"))
	if err != nil {
		return ""
	}
	for _, kv := range strings.Split(string(b), "\x00") {
		if v, ok := strings.CutPrefix(kv, "WINEPREFIX="); ok {
			return strings.TrimRight(v, "/")
		}
	}
	return ""
}

// procPPID reads field 4 of /proc/PID/stat; the comm field may hold spaces and parentheses, so cut at the last ')'.
func procPPID(root string, pid int) int {
	b, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return 0
	}
	f := strings.Fields(s[i+1:]) // state ppid ...
	if len(f) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(f[1])
	return n
}

// wineSummary is the report view of the winedevice samples.
type wineSummary struct {
	Samples, MaxTotal, MaxOrphans, LastTotal, LastOrphans int
}

func summarizeWine(ss []Sample) (w wineSummary) {
	for _, s := range ss {
		if s.Wine == nil {
			continue
		}
		w.Samples++
		w.MaxTotal, w.MaxOrphans = max(w.MaxTotal, s.Wine.Total), max(w.MaxOrphans, s.Wine.Orphans)
		w.LastTotal, w.LastOrphans = s.Wine.Total, s.Wine.Orphans
	}
	return
}

func (w wineSummary) text() string {
	return fmt.Sprintf("winedevice   %d samples, total max %d (last %d), orphans max %d (last %d)", w.Samples, w.MaxTotal, w.LastTotal, w.MaxOrphans, w.LastOrphans)
}
