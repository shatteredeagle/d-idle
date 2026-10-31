package saves

import (
	"encoding/binary"
	"fmt"
	"os"

	"d2lod-bot/internal/cheat/d2s"
)

// Quest word flags (S: the game's quest-flag list as published by the format docs; the same bits are used
// by every quest). A quest is stored as one little-endian uint16.
const (
	QRewardGranted  = 1 << 0
	QRewardPending  = 1 << 1
	QStarted        = 1 << 2
	QLeftTown       = 1 << 3
	QEnteredArea    = 1 << 4
	QQuestLog       = 1 << 12
	QPrimaryGoal    = 1 << 13
	QCompletedNow   = 1 << 14
	QCompletedEarly = 1 << 15 // "completed before"
)

// questSlot is one 16-bit word of the 96-byte per-difficulty quest block. Word index = byte offset / 2.
// Layout tags (live check 2026-10-28 on Multi, docs/saves.md): V = a hand-set word lit exactly that quest in
// the in-game quest log: words 1 Den, 2 Sisters, 3 Tools of the Trade, 4 Search for Cain, 10 Horadric Staff,
// 20 Golden Bird (so the act III order below), 25 Fallen Angel, 37 Prison of Ice, 40 Eve of Destruction (the game
// also wrote progression 5 for it). The other quest words follow the same order (S). Intro and "act complete"
// words (0, 7, 8, 15, 16, 23, 24, 28, 32) are positions only: U.
type questSlot struct {
	ID   string
	Name string
	Act  int
	Word int
}

var questTable = []questSlot{
	{"act1_intro", "Warriv introduction", 1, 0},
	{"den_of_evil", "Den of Evil", 1, 1},
	{"sisters_burial_grounds", "Sisters' Burial Grounds", 1, 2},
	{"tools_of_the_trade", "Tools of the Trade", 1, 3},
	{"search_for_cain", "The Search for Cain", 1, 4},
	{"forgotten_tower", "The Forgotten Tower", 1, 5},
	{"sisters_to_the_slaughter", "Sisters to the Slaughter", 1, 6},
	{"act1_done", "Act I complete", 1, 7},

	{"act2_intro", "Act II introduction", 2, 8},
	{"radament", "Radament's Lair", 2, 9},
	{"horadric_staff", "The Horadric Staff", 2, 10},
	{"tainted_sun", "The Tainted Sun", 2, 11},
	{"arcane_sanctuary", "The Arcane Sanctuary", 2, 12},
	{"summoner", "The Summoner", 2, 13},
	{"seven_tombs", "The Seven Tombs", 2, 14},
	{"act2_done", "Act II complete", 2, 15},

	{"act3_intro", "Act III introduction", 3, 16},
	{"lam_esen", "Lam Esen's Tome", 3, 17},
	{"khalims_will", "Khalim's Will", 3, 18},
	{"blade_old_religion", "Blade of the Old Religion", 3, 19},
	{"golden_bird", "The Golden Bird", 3, 20},
	{"blackened_temple", "The Blackened Temple", 3, 21},
	{"guardian", "The Guardian", 3, 22},
	{"act3_done", "Act III complete", 3, 23},

	{"act4_intro", "Act IV introduction", 4, 24},
	{"izual", "The Fallen Angel", 4, 25},
	{"hellforge", "Hell's Forge", 4, 26},
	{"terrors_end", "Terror's End", 4, 27},
	{"act4_done", "Act IV complete", 4, 28},

	// Act V quest words are 35-40 (VERIFIED LIVE 2026-10-28, docs/saves.md), not 33-38. The intro word is U.
	{"act5_intro", "Act V introduction", 5, 32},
	{"siege_harrogath", "Siege on Harrogath", 5, 35},
	{"rescue_arreat", "Rescue on Mount Arreat", 5, 36},
	{"prison_of_ice", "Prison of Ice", 5, 37},
	{"betrayal_harrogath", "Betrayal of Harrogath", 5, 38},
	{"rite_of_passage", "Rite of Passage", 5, 39},
	{"eve_of_destruction", "Eve of Destruction", 5, 40},
}

// RewardQuests maps the ids of docs/plan/data/quest-points.json to quest ids here (U for the bit meaning).
var RewardQuests = map[string]string{
	"den_of_evil":          "den_of_evil",
	"radament":             "radament",
	"lam_esen":             "lam_esen",
	"izual":                "izual",
	"golden_bird":          "golden_bird",
	"scroll_of_resistance": "prison_of_ice",
}

// QuestState is one quest of one difficulty.
type QuestState struct {
	ID, Name      string
	Act           int
	Word          uint16 // raw
	Started       bool
	RewardGranted bool
	Done          bool // RewardGranted or CompletedEarly (U: "finished" semantics per quest)
}

// Waypoint is one waypoint bit.
type Waypoint struct {
	Bit    int
	Level  int // levels.txt id (S)
	Name   string
	Act    int
	Active bool
}

// waypointTable is in file bit order: act I 9, act II 9, act III 9, act IV 3, act V 9 = 39 (S: PK order;
// V only that bit 0 = Rogue Encampment is set on fresh Testy and Botty). Level ids are levels.txt ids from
// recollection (S).
var waypointTable = [39]struct {
	Level int
	Name  string
	Act   int
}{
	{1, "Rogue Encampment", 1}, {3, "Cold Plains", 1}, {4, "Stony Field", 1}, {5, "Dark Wood", 1}, {6, "Black Marsh", 1},
	{27, "Outer Cloister", 1}, {29, "Jail Level 1", 1}, {32, "Inner Cloister", 1}, {35, "Catacombs Level 2", 1},
	{40, "Lut Gholein", 2}, {48, "Sewers Level 2", 2}, {42, "Dry Hills", 2}, {57, "Halls of the Dead Level 2", 2},
	{43, "Far Oasis", 2}, {44, "Lost City", 2}, {52, "Palace Cellar Level 1", 2}, {74, "Arcane Sanctuary", 2}, {46, "Canyon of the Magi", 2},
	{75, "Kurast Docks", 3}, {76, "Spider Forest", 3}, {77, "Great Marsh", 3}, {78, "Flayer Jungle", 3}, {79, "Lower Kurast", 3},
	{80, "Kurast Bazaar", 3}, {81, "Upper Kurast", 3}, {83, "Travincal", 3}, {101, "Durance of Hate Level 2", 3},
	{103, "The Pandemonium Fortress", 4}, {106, "City of the Damned", 4}, {107, "River of Flame", 4},
	{109, "Harrogath", 5}, {111, "Frigid Highlands", 5}, {112, "Arreat Plateau", 5}, {113, "Crystalline Passage", 5},
	{123, "Halls of Pain", 5}, {115, "Glacial Trail", 5}, {117, "Frozen Tundra", 5}, {118, "The Ancients' Way", 5}, {129, "Worldstone Keep Level 2", 5},
}

// Difficulty is the progression view of one difficulty.
type Difficulty struct {
	Name       string // normal, nightmare, hell
	Unlocked   bool   // U: progression byte >= 5*d, or the previous difficulty's Eve of Destruction done
	LastPlayed bool   // header byte 0xA8+d bit 7 (S)
	Act        int    // 1-5, header low bits + 1 (S)
	Quests     []QuestState
	Waypoints  []Waypoint // all 39, Active set
	WPMask     uint64
}

// Active returns the active waypoints only.
func (d Difficulty) Active() []Waypoint {
	var out []Waypoint
	for _, w := range d.Waypoints {
		if w.Active {
			out = append(out, w)
		}
	}
	return out
}

// Quest returns the state of quest id (zero value if unknown).
func (d Difficulty) Quest(id string) QuestState {
	for _, q := range d.Quests {
		if q.ID == id {
			return q
		}
	}
	return QuestState{}
}

// RewardsEarned lists the quest-points.json ids whose quest has its reward granted (U).
func (d Difficulty) RewardsEarned() []string {
	var out []string
	for _, k := range []string{"den_of_evil", "radament", "lam_esen", "golden_bird", "izual", "scroll_of_resistance"} {
		if d.Quest(RewardQuests[k]).RewardGranted {
			out = append(out, k)
		}
	}
	return out
}

// Progress is everything the bot needs from the save.
type Progress struct {
	Name        string
	Class       int
	Level       int
	Hardcore    bool
	Expansion   bool
	Progression byte
	MapID       uint32 // header 0xAB (V: Botty and Testy, and equals Botty.map offset 8)
	Difficulty  [3]Difficulty
}

var diffNames = [3]string{"normal", "nightmare", "hell"}

// ProgressOf decodes a parsed file.
func ProgressOf(f *d2s.File) Progress {
	p := Progress{
		Name: f.Name(), Class: f.Class(), Level: f.Level(),
		Hardcore:    f.Status()&d2s.FlagHardcore != 0,
		Expansion:   f.Status()&d2s.FlagExpansion != 0,
		Progression: f.Progression(),
		MapID:       f.MapID(),
	}
	for d := 0; d < 3; d++ {
		raw := f.Quests(d)
		dp := Difficulty{Name: diffNames[d], WPMask: f.WaypointMask(d)}
		b := f.Difficulty(d)
		dp.LastPlayed = b&0x80 != 0
		dp.Act = int(b&0x7) + 1
		for _, s := range questTable {
			w := binary.LittleEndian.Uint16(raw[s.Word*2:])
			dp.Quests = append(dp.Quests, QuestState{
				ID: s.ID, Name: s.Name, Act: s.Act, Word: w,
				Started:       w&QStarted != 0,
				RewardGranted: w&QRewardGranted != 0,
				Done:          w&(QRewardGranted|QCompletedEarly) != 0,
			})
		}
		for i, w := range waypointTable {
			dp.Waypoints = append(dp.Waypoints, Waypoint{Bit: i, Level: w.Level, Name: w.Name, Act: w.Act, Active: dp.WPMask>>uint(i)&1 == 1})
		}
		p.Difficulty[d] = dp
	}
	for d := 0; d < 3; d++ {
		u := d == 0 || int(p.Progression) >= 5*d
		if d > 0 && p.Difficulty[d-1].Quest("eve_of_destruction").Done {
			u = true
		}
		p.Difficulty[d].Unlocked = u
	}
	return p
}

// Decode parses and decodes save bytes (Verify first, so a corrupt file is an error, not zeros).
func Decode(b []byte) (Progress, error) {
	f, err := Verify(b)
	if err != nil {
		return Progress{}, err
	}
	return ProgressOf(f), nil
}

// ProgressFile reads a .d2s from disk.
func ProgressFile(path string) (Progress, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Progress{}, err
	}
	return Decode(b)
}

// MapFileSeed reads the map seed from a <name>.map file. V on Botty.map (24 bytes): uint32 at offset 8
// equals the d2s header map id; the other 20 bytes (0c000000 01000000 ... zeros) are not understood (U).
func MapFileSeed(b []byte) (uint32, error) {
	if len(b) < 12 {
		return 0, fmt.Errorf("saves: .map file too short (%d bytes)", len(b))
	}
	return binary.LittleEndian.Uint32(b[8:]), nil
}

// ExpOf is the character level and total experience stored in a save (the stat the game writes on Save and Exit; V: it
// is what the in-game experience was at that moment, docs/soak-analysis.md section 6). Verify first, like Decode.
func ExpOf(b []byte) (level int, exp uint32, err error) {
	f, err := Verify(b)
	if err != nil {
		return 0, 0, err
	}
	exp, _ = f.Stat(d2s.StatExp)
	return f.Level(), exp, nil
}

// ExpFile reads ExpOf from a .d2s on disk.
func ExpFile(path string) (level int, exp uint32, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	return ExpOf(b)
}
