package main

// fd leak probe (agent LEAKHUNT): samples /proc/PID/fd of every process of one instance (Game.exe, wineserver,
// d2input.exe, the other wine processes, Xvfb) and classifies the descriptors.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type fdProc struct {
	pid  int
	name string
}

// instProcs finds the processes of one instance: wine processes by WINEPREFIX/D2BOT_INSTANCE, Xvfb by its :10N display.
func instProcs(inst string) []fdProc {
	var out []fdProc
	ents, _ := filepath.Glob("/proc/[0-9]*")
	disp := ":10" + strings.TrimPrefix(inst, "i")
	for _, e := range ents {
		pid, _ := strconv.Atoi(filepath.Base(e))
		comm, err := os.ReadFile(e + "/comm")
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(comm))
		env, _ := os.ReadFile(e + "/environ")
		cmd, _ := os.ReadFile(e + "/cmdline")
		mine := strings.Contains(string(env), "D2BOT_INSTANCE="+inst+"\x00") ||
			strings.Contains(string(env), "/d2lod-inst/"+inst+"/")
		if name == "Xvfb" && strings.Contains(strings.ReplaceAll(string(cmd), "\x00", " "), "Xvfb "+disp+" ") {
			mine = true
		}
		if mine {
			out = append(out, fdProc{pid, name})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].pid < out[j].pid })
	return out
}

// classify buckets a readlink target.
func classify(t string) string {
	switch {
	case strings.HasPrefix(t, "socket:"):
		return "socket"
	case strings.HasPrefix(t, "pipe:"):
		return "pipe"
	case strings.HasPrefix(t, "anon_inode:[eventfd]"):
		return "eventfd"
	case strings.HasPrefix(t, "anon_inode:[eventpoll]"):
		return "epoll"
	case strings.HasPrefix(t, "anon_inode:"):
		return "anon"
	case strings.HasPrefix(t, "/dev/shm/"), strings.Contains(t, "/memfd:"), strings.HasPrefix(t, "/memfd:"):
		return "shm"
	case strings.HasPrefix(t, "/proc/") && strings.HasSuffix(t, "/mem"):
		return "procmem"
	case strings.HasPrefix(t, "/dev/"):
		return "dev"
	case strings.HasPrefix(t, "/"):
		return "file"
	}
	return "other"
}

// fdTargets returns fd -> link target for one pid.
func fdTargets(pid int) map[string]string {
	m := map[string]string{}
	ents, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		return m
	}
	for _, e := range ents {
		t, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", pid, e.Name()))
		if err == nil {
			m[e.Name()] = t
		}
	}
	return m
}

// fdLine formats "name:total(cls=n,...)" for every process; returns the line and per-name totals.
func fdLine(procs []fdProc) (string, map[string]int) {
	totals := map[string]int{}
	var parts []string
	for _, p := range procs {
		t := fdTargets(p.pid)
		cls := map[string]int{}
		for _, v := range t {
			cls[classify(v)]++
		}
		totals[p.name] += len(t)
		if p.name == "wineserver" || p.name == "Game.exe" || p.name == "d2input.exe" || p.name == "Xvfb" || p.name == "d2bot" {
			var cs []string
			for k, n := range cls {
				cs = append(cs, fmt.Sprintf("%s=%d", k, n))
			}
			sort.Strings(cs)
			parts = append(parts, fmt.Sprintf("%s[%d]:%d(%s)", p.name, p.pid, len(t), strings.Join(cs, ",")))
		}
	}
	return strings.Join(parts, " "), totals
}

// fdDiff lists targets present in b but not in a (by target string count), for one pid pair snapshot.
func fdDiff(a, b map[string]string) []string {
	have := map[string]int{}
	for _, v := range a {
		have[v]++
	}
	var out []string
	for _, v := range b {
		if have[v] > 0 {
			have[v]--
			continue
		}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
