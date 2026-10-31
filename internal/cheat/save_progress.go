package cheat

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"d2lod-bot/internal/cheat/d2s"
	"d2lod-bot/internal/saves"
)

// Progress edits of the save editor: quest words, waypoints, difficulty unlock, start act.
// Tags (docs/saves.md): the block positions are VERIFIED; the quest word per quest name comes from
// internal/saves (S for acts 1-3, U for 4-5), the bit meanings behind the state presets are U, and
// whether the game honors these edits at load (unlock, act, quest state) is UNVERIFIED until the live
// checklist in docs/cheats.md has been run. Nothing here claims a quest bit is verified.

// Quest state presets for `quest=...:STATE`. A number (0x8001, 32769) sets the raw word instead.
var questStates = map[string]uint16{
	"none":    0,
	"clear":   0,
	"started": saves.QStarted | saves.QLeftTown, // like Testy's word 1 (0x000c)
	"pending": saves.QStarted | saves.QRewardPending,
	"done":    saves.QRewardGranted | saves.QCompletedEarly, // what saves.Difficulty.Done() tests (U)
}

var diffNames = map[string]int{"0": 0, "1": 1, "2": 2, "n": 0, "normal": 0, "nm": 1, "nightmare": 1, "h": 2, "hell": 2}

// splitDiff parses an optional "DIFF/" prefix (default normal).
func splitDiff(s string) (int, string, error) {
	d, rest, ok := strings.Cut(s, "/")
	if !ok {
		return 0, s, nil
	}
	n, ok := diffNames[strings.ToLower(d)]
	if !ok {
		return 0, "", fmt.Errorf("cheat: difficulty %q (want 0-2, normal, nightmare, hell)", d)
	}
	return n, rest, nil
}

// questWords maps the quest ids of internal/saves to their word index by asking saves itself: set one word at
// a time on a scratch copy and see which named quest reads as done. This reuses the saves table instead of
// copying it (saves does not export the index; request R11 in the report).
func questWords(f *d2s.File) map[string]int {
	c := f.Clone()
	c.ClearQuests(0)
	m := map[string]int{}
	for i := 0; i < d2s.QuestWords; i++ {
		c.SetQuestWord(0, i, saves.QCompletedEarly)
		for _, q := range saves.ProgressOf(c).Difficulty[0].Quests {
			if q.Done {
				m[q.ID] = i
			}
		}
		c.SetQuestWord(0, i, 0)
	}
	return m
}

// questWord resolves "ACT.ID" (e.g. 1.den_of_evil; the act is checked against saves) or "wN" (raw word index).
func questWord(f *d2s.File, ref string) (int, error) {
	if len(ref) > 1 && (ref[0] == 'w' || ref[0] == 'W') {
		if n, err := strconv.Atoi(ref[1:]); err == nil && n >= 0 && n < d2s.QuestWords {
			return n, nil
		}
	}
	act, id, ok := strings.Cut(ref, ".")
	a, err := strconv.Atoi(act)
	if !ok || err != nil || a < 1 || a > 5 {
		return 0, fmt.Errorf("cheat: quest %q (want ACT.QUEST like 1.den_of_evil, or wINDEX)", ref)
	}
	id = strings.ToLower(id)
	w, found := questWords(f)[id]
	if !found {
		return 0, fmt.Errorf("cheat: unknown quest %q (ids: see saves.Progress, e.g. den_of_evil, act2_done)", id)
	}
	for _, q := range saves.ProgressOf(f).Difficulty[0].Quests {
		if q.ID == id && q.Act != a {
			return 0, fmt.Errorf("cheat: quest %s belongs to act %d, not %d", id, q.Act, a)
		}
	}
	return w, nil
}

func parseState(s string) (uint16, error) {
	if s == "" {
		s = "done"
	}
	if w, ok := questStates[strings.ToLower(s)]; ok {
		return w, nil
	}
	n, err := strconv.ParseUint(s, 0, 16)
	if err != nil {
		return 0, fmt.Errorf("cheat: quest state %q (want done, started, pending, none or a 16-bit number)", s)
	}
	return uint16(n), nil
}

// waypointBit resolves a waypoint name (saves' names, lowercase with underscores: cold_plains) or a bit number 0-38.
func waypointBit(f *d2s.File, ref string) (int, error) {
	if n, err := strconv.Atoi(ref); err == nil {
		if n < 0 || n > 38 {
			return 0, fmt.Errorf("cheat: waypoint bit %d out of range 0-38", n)
		}
		return n, nil
	}
	slug := strings.ToLower(ref)
	for _, w := range saves.ProgressOf(f).Difficulty[0].Waypoints {
		if strings.ReplaceAll(strings.ToLower(w.Name), " ", "_") == slug {
			return w.Bit, nil
		}
	}
	return 0, fmt.Errorf("cheat: unknown waypoint %q (bit 0-38 or a name like cold_plains)", ref)
}

func (e *Env) applyProgress(f *d2s.File, req Request) ([]string, error) {
	var did []string
	if v, ok := req.Int("questclear"); ok {
		for d := 0; d < 3; d++ {
			if v&(1<<d) != 0 {
				if err := f.ClearQuests(d); err != nil {
					return nil, err
				}
			}
		}
		did = append(did, fmt.Sprintf("quests cleared (difficulty mask %d)", v))
	}
	if s := req.Str("quest"); s != "" { // "1.den_of_evil:done,1/w9:started"
		for _, item := range strings.Split(s, ",") {
			d, rest, err := splitDiff(strings.TrimSpace(item))
			if err != nil {
				return nil, err
			}
			ref, state, _ := strings.Cut(rest, ":")
			w, err := questWord(f, ref)
			if err != nil {
				return nil, err
			}
			word, err := parseState(state)
			if err != nil {
				return nil, err
			}
			if err := f.SetQuestWord(d, w, word); err != nil {
				return nil, err
			}
		}
		did = append(did, "quest "+s)
	}
	if s := req.Str("wp"); s != "" { // "0/all,1/none,2/cold_plains+7"
		for _, item := range strings.Split(s, ",") {
			d, rest, err := splitDiff(strings.TrimSpace(item))
			if err != nil {
				return nil, err
			}
			switch rest {
			case "all":
				err = f.SetWaypointMask(d, d2s.AllWaypoints)
			case "none":
				err = f.SetWaypointMask(d, 0)
			default:
				for _, ref := range strings.Split(rest, "+") {
					var b int
					if b, err = waypointBit(f, ref); err != nil {
						break
					}
					err = f.SetWaypoints(d, 1<<b)
				}
			}
			if err != nil {
				return nil, err
			}
			openTabs(f, d)
		}
		did = append(did, "wp "+s)
	}
	if v, ok := req.Int("unlock"); ok { // difficulty 1 or 2
		if v < 1 || v > 2 {
			return nil, errors.New("cheat: unlock wants 1 (nightmare) or 2 (hell)")
		}
		unlock(f, int(v))
		did = append(did, fmt.Sprintf("progression %d (unlock difficulty %d)", f.Progression(), v))
	}
	if s := req.Str("startat"); s != "" { // "2" or "1/3": act 1-5 in difficulty
		d, rest, err := splitDiff(s)
		a, aerr := strconv.Atoi(rest)
		if err != nil || aerr != nil || a < 1 || a > 5 {
			return nil, fmt.Errorf("cheat: startat %q (want [DIFF/]ACT, act 1-5)", s)
		}
		if err := startAt(f, d, a); err != nil {
			return nil, err
		}
		did = append(did, "startat "+s)
	}
	return did, nil
}

// unlock raises the progression byte to what a character that finished difficulty d-1 would have
// (5 per finished difficulty). The meaning of the byte is U (docs/saves.md); this only moves it.
func unlock(f *d2s.File, d int) {
	if want := byte(5 * d); f.Progression() < want {
		f.SetProgression(want)
	}
}

// openTabs marks "act N done" for every act below the highest act that has a waypoint bit set in difficulty d.
// The waypoint panel only draws the tab of an act the character has reached (V live 2026-10-29: with all 39 bits
// set but no act-done words the panel showed tabs I and II only, and the act II rows all worked); the exact quest
// condition for the tab is U, this uses the same words as startAt.
func openTabs(f *d2s.File, d int) {
	top := 1
	for _, w := range saves.ProgressOf(f).Difficulty[d].Waypoints {
		if w.Active && w.Act > top {
			top = w.Act
		}
	}
	words := questWords(f)
	for n := 1; n < top; n++ {
		if w, ok := words[fmt.Sprintf("act%d_done", n)]; ok {
			f.SetQuestWord(d, w, questStates["done"])
		}
	}
	if top == 5 { // tab V needs more than act IV done: V live 2026-10-29, terrors_end alone left it hidden; with these it shows
		for _, id := range []string{"izual", "hellforge", "terrors_end", "act5_intro"} {
			if w, ok := words[id]; ok {
				f.SetQuestWord(d, w, questStates["done"])
			}
		}
	}
}

// startAt makes the character load in the town of act a of difficulty d: active difficulty byte, the town
// waypoint, the "act done" word of every earlier act and the intro word of act a, and the difficulty unlock.
// All of it is U until the live check (the game may recompute some of it on load).
func startAt(f *d2s.File, d, a int) error {
	if err := f.SetActive(d, a-1); err != nil {
		return err
	}
	if d > 0 {
		unlock(f, d)
	}
	words := questWords(f)
	for n := 1; n < a; n++ {
		if w, ok := words[fmt.Sprintf("act%d_done", n)]; ok {
			f.SetQuestWord(d, w, questStates["done"])
		}
	}
	if w, ok := words[fmt.Sprintf("act%d_intro", a)]; ok {
		f.SetQuestWord(d, w, questStates["done"])
	}
	for _, w := range saves.ProgressOf(f).Difficulty[d].Waypoints {
		if w.Act == a {
			return f.SetWaypoints(d, 1<<w.Bit) // first waypoint of the act = its town
		}
	}
	return nil
}
