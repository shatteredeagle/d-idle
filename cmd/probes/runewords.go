package main

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
)

// Q1 (docs/items.md 3.1): do ladder-flagged runewords and Hel+ rune upgrades work offline? The runewords themselves
// are made by socketing runes by hand (no code does that yet, docs/items.md 3.1), so `check` is the read-only half:
// it inventories bases/runes/runeword items (run it before and after the manual insert) and `cube` tests the other
// ladder-flagged thing, the Dol->Hel recipe, through Town.Transmute.

var runeCode = regexp.MustCompile(`^r(0[1-9]|[12][0-9]|3[0-3])$`)

// Recipe inputs by name. gcg (chipped emerald) is S: the codes gcy/gcv/gcw are V in cubejobs, gcg follows the same
// pattern (g c <colour>). Dol = r14, Hel = r15 (V cubejobs doc: r01 El .. r33 Zod).
var recipes = map[string][]string{
	"hel": {"r14", "r14", "r14", "gcg"},
}

type rwOpts struct {
	cube   bool
	run    bool
	recipe string
	inputs []string
	sec    int
}

func parseRunewords(args []string) (func(context.Context, *Env) error, error) {
	if len(args) == 0 || (args[0] != "check" && args[0] != "cube") {
		return nil, fmt.Errorf("runewords: want 'check' or 'cube'\n%s", usage)
	}
	o := rwOpts{cube: args[0] == "cube"}
	fs, sec := newFlags("runewords " + args[0])
	*sec = 1
	run := fs.Bool("go", false, "cube: really press Transmute")
	recipe := fs.String("recipe", "hel", "cube: recipe name (hel) or custom")
	inputs := fs.String("inputs", "", "cube: comma separated item codes for -recipe custom")
	if err := parseFlags(fs, args[1:], sec); err != nil {
		return nil, err
	}
	o.run, o.recipe, o.sec = *run, *recipe, *sec
	if o.cube {
		if o.recipe == "custom" {
			o.inputs = csv(*inputs)
			if len(o.inputs) == 0 {
				return nil, fmt.Errorf("runewords cube: -recipe custom needs -inputs")
			}
		} else if in, ok := recipes[o.recipe]; ok {
			o.inputs = in
		} else {
			return nil, fmt.Errorf("runewords cube: unknown recipe %q", o.recipe)
		}
	}
	return func(ctx context.Context, e *Env) error { return runRunewords(ctx, e, o) }, nil
}

// packItems lists the items of the player's own containers (pack, stash, cube); worn and belt items are excluded.
func (e *Env) packItems(s *d2.Snapshot) []map[string]any {
	var out []map[string]any
	if s == nil || s.Inventory == nil {
		return out
	}
	for _, u := range s.Inventory.Items {
		it := u.Item
		if it == nil || it.Where == d2.ContEquipment || it.Where == d2.ContBelt {
			continue
		}
		code := ""
		if e.ItemCode != nil {
			code = e.ItemCode(u.Class)
		}
		out = append(out, map[string]any{
			"id": u.ID, "class": u.Class, "code": code, "where": int(it.Where), "gx": it.GX, "gy": it.GY,
			"flags": hex32(it.Flags), "quality": it.Quality, "socketed": it.Socketed(),
			"runeword": it.Flags&d2.FlagRuneword != 0, "ethereal": it.Ethereal(), "file_index": it.FileIndex,
		})
	}
	return out
}

// runeCounts counts rune codes (r01..r33) over the items.
func runeCounts(items []map[string]any) map[string]int {
	c := map[string]int{}
	for _, it := range items {
		if code, _ := it["code"].(string); runeCode.MatchString(code) {
			c[code]++
		}
	}
	return c
}

// candidates lists the runewords whose runes (as a multiset) the pack holds. Order and the "needs a socketed base"
// condition are not checked here; the report carries both facts for the reader.
func candidates(have map[string]int, rows []RuneRow) []map[string]any {
	var out []map[string]any
	for _, r := range rows {
		need := map[string]int{}
		for _, c := range r.Runes {
			need[c]++
		}
		ok := len(r.Runes) > 0
		for c, n := range need {
			if have[c] < n {
				ok = false
			}
		}
		if ok {
			out = append(out, map[string]any{"name": r.Name, "ladder": r.Ladder, "runes": r.Runes})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["name"].(string) < out[j]["name"].(string) })
	return out
}

func rwSummary(e *Env, items []map[string]any) map[string]any {
	socketed, rwItems := 0, 0
	for _, it := range items {
		if it["runeword"] == true {
			rwItems++
		} else if it["socketed"] == true {
			socketed++
		}
	}
	have := runeCounts(items)
	return map[string]any{
		"items": len(items), "socketed_bases": socketed, "runeword_items": rwItems, "runes": have,
		"candidates": candidates(have, e.Runewords),
	}
}

func runRunewords(ctx context.Context, e *Env, o rwOpts) error {
	const p = "runewords"
	s, err := e.Snap()
	if err != nil {
		return e.fail(p, err)
	}
	before := e.packItems(s)
	e.emit(p, "pack", map[string]any{"list": before})
	e.emit(p, "summary", rwSummary(e, before))
	if !o.cube {
		return nil
	}
	have := runeCounts(before)
	need := map[string]int{}
	for _, c := range o.inputs {
		need[c]++
	}
	var missing []string
	codes := map[string]int{}
	for _, it := range before {
		if c, _ := it["code"].(string); c != "" {
			codes[c]++
		}
	}
	for c, n := range need {
		if codes[c] < n {
			missing = append(missing, fmt.Sprintf("%s x%d (have %d)", c, n, codes[c]))
		}
	}
	sort.Strings(missing)
	e.emit(p, "plan", map[string]any{"recipe": o.recipe, "inputs": o.inputs, "missing": missing, "go": o.run, "runes_before": have})
	if len(missing) > 0 {
		return e.fail(p, fmt.Errorf("pack lacks cube inputs: %s", strings.Join(missing, ", ")))
	}
	if !o.run {
		e.emit(p, "verdict", map[string]any{"result": "dry-run", "note": "re-run with -go to press Transmute"})
		return nil
	}
	if e.Transmute == nil {
		return e.fail(p, fmt.Errorf("no transmute wired"))
	}
	t0 := e.Now()
	out, terr := e.Transmute(ctx, o.inputs)
	f := map[string]any{"outputs": out, "ms": e.Now().Sub(t0) / time.Millisecond}
	if terr != nil {
		f["error"] = terr.Error()
	}
	e.emit(p, "transmute", f)
	if e.CloseAll != nil {
		_ = e.CloseAll()
	}
	if s, err = e.Snap(); err == nil {
		after := e.packItems(s)
		e.emit(p, "summary_after", rwSummary(e, after))
	}
	verdict := "no-result"
	switch {
	case terr != nil:
		verdict = "error"
	case o.recipe == "hel" && contains(out, "r15"):
		verdict = "ladder-recipe-works-offline"
	case len(out) > 0:
		verdict = "produced-other"
	}
	e.emit(p, "verdict", map[string]any{"result": verdict, "recipe": o.recipe, "outputs": out})
	return terr
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}
