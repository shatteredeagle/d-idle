package cheat

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"d2lod-bot/internal/data/mpq"
)

// Data overlay cheats change the game's excel tables for ONE instance. The game reads loose files from
// <game dir>\Data\global\excel\ instead of the MPQs when it is started with `-direct -txt` (see docs/cheats.md
// for what was verified live). Each instance has its own hardlinked game folder, so the overlay stays with
// that instance; the i0 folder (the source of the clones) is refused. A restart of the game is needed.

var overlayArchives = []string{"patch_d2.mpq", "d2exp.mpq", "d2data.mpq"}

// ReadExcel returns the highest-priority copy of data\global\excel\<name>.txt from the install's MPQs.
func ReadExcel(installDir, name string) ([]byte, error) {
	for _, an := range overlayArchives {
		p := findFold(installDir, an)
		if p == "" {
			continue
		}
		a, err := mpq.Open(p)
		if err != nil {
			return nil, err
		}
		file := `data\global\excel\` + name + ".txt"
		if a.Has(file) {
			b, err := a.ReadFile(file)
			a.Close()
			return b, err
		}
		a.Close()
	}
	return nil, fmt.Errorf("cheat: excel table %s not found in %s", name, installDir)
}

func findFold(dir, name string) string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range ents {
		if strings.EqualFold(e.Name(), name) {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

// editColumns applies f to every cell of the named columns (case-insensitive) of a tab separated table,
// keeping line endings and every other cell untouched. Empty and non-numeric cells are skipped. It returns
// how many cells changed.
func editColumns(table []byte, cols []string, numericFirst bool, f func(int64) int64) ([]byte, int, error) {
	lines := strings.Split(string(table), "\n")
	if len(lines) < 2 {
		return nil, 0, errors.New("cheat: empty table")
	}
	hdr := strings.Split(strings.TrimRight(lines[0], "\r"), "\t")
	var idx []int
	for _, c := range cols {
		found := -1
		for i, h := range hdr {
			if strings.EqualFold(h, c) {
				found = i
				break
			}
		}
		if found < 0 {
			return nil, 0, fmt.Errorf("cheat: column %q not in the table (have %v)", c, hdr)
		}
		idx = append(idx, found)
	}
	n := 0
	for li := 1; li < len(lines); li++ {
		cr := ""
		l := lines[li]
		if strings.HasSuffix(l, "\r") {
			cr, l = "\r", l[:len(l)-1]
		}
		if l == "" {
			continue
		}
		cells := strings.Split(l, "\t")
		if numericFirst {
			if _, err := strconv.Atoi(cells[0]); err != nil {
				continue // e.g. the MaxLvl row of experience.txt
			}
		}
		for _, ci := range idx {
			if ci >= len(cells) {
				continue
			}
			v, err := strconv.ParseInt(cells[ci], 10, 64)
			if err != nil {
				continue
			}
			if nv := f(v); nv != v {
				cells[ci] = strconv.FormatInt(nv, 10)
				n++
			}
		}
		lines[li] = strings.Join(cells, "\t") + cr
	}
	return []byte(strings.Join(lines, "\n")), n, nil
}

// doOverlay supports presets, one per call:
//
//	xprate=N     monstats.txt Exp, Exp(N), Exp(H) multiplied by N (experience per kill)
//	nodrop=0     treasureclassex.txt NoDrop set to 0 (every kill drops, more items)
//	exppct=P     experience.txt level thresholds scaled to P percent (levels come faster)
//	clear=1      delete the instance Data folder (undo everything)
func (e *Env) doOverlay(ctx context.Context, req Request, ent *Entry) (string, error) {
	if e.Instance == "" || e.Instance == "i0" {
		return "", errors.New("cheat: data overlays are refused for i0 (its folder is the source of every clone)")
	}
	var table string
	var cols []string
	var f func(int64) int64
	if has(req, "clear") {
		// The game also writes compiled .bin files next to the .txt overlays, so the whole Data folder goes.
		if filepath.Base(e.DataDir) != "Data" || !strings.Contains(e.DataDir, e.Instance) {
			return "", fmt.Errorf("cheat: refusing to delete %s", e.DataDir)
		}
		ent.Before = map[string]any{"dir": e.DataDir, "existed": fileExists(e.DataDir)}
		if err := os.RemoveAll(e.DataDir); err != nil {
			return "", err
		}
		return "removed " + e.DataDir + "; the next start without -direct -txt uses the MPQ data again", nil
	}
	switch {
	case has(req, "xprate"):
		n, _ := req.Int("xprate")
		if n < 1 || n > 1000 {
			return "", errors.New("cheat: xprate must be 1-1000")
		}
		table, cols, f = "monstats", []string{"Exp", "Exp(N)", "Exp(H)"}, func(v int64) int64 { return v * n }
	case has(req, "nodrop"):
		n, _ := req.Int("nodrop")
		if n != 0 {
			return "", errors.New("cheat: only nodrop=0 is supported")
		}
		table, cols, f = "treasureclassex", []string{"NoDrop"}, func(int64) int64 { return 0 }
	case has(req, "exppct"):
		p, _ := req.Int("exppct")
		if p < 1 || p > 10000 {
			return "", errors.New("cheat: exppct must be 1-10000")
		}
		table = "experience"
		cols = []string{"Amazon", "Sorceress", "Necromancer", "Paladin", "Barbarian", "Druid", "Assassin"}
		f = func(v int64) int64 {
			if v <= 0 {
				return v
			}
			if nv := v * p / 100; nv > 0 {
				return nv
			}
			return 1
		}
	default:
		return "", errors.New("cheat: dataoverlay needs xprate=N, nodrop=0 or exppct=P")
	}
	install := DefaultInstallDir(e.Instance)
	raw, err := ReadExcel(install, table)
	if err != nil {
		return "", err
	}
	out, n, err := editColumns(raw, cols, table == "experience", f)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(e.DataDir, "global", "excel")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dst := filepath.Join(dir, table+".txt")
	ent.Before = map[string]any{"file": dst, "existed": fileExists(dst)}
	if err := os.WriteFile(dst, out, 0o644); err != nil {
		return "", err
	}
	ent.After = map[string]any{"file": dst, "cells changed": n, "bytes": len(out)}
	return fmt.Sprintf("wrote %s (%d cells changed). Restart the instance with -direct -txt for it to apply (docs/cheats.md); remove the Data folder to undo.", dst, n), nil
}

func has(r Request, k string) bool { _, ok := r.Params[k]; return ok }

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }
