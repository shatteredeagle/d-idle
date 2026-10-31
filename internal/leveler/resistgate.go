package leveler

import (
	"fmt"

	"d2lod-bot/internal/d2"
)

// Resist floors for entering a difficulty, measured after that difficulty's penalty (docs/builds-endgame.md W8,
// docs/difficulty.md 2.2 "Resist gate"; U numbers, the usual "cap everything before Hell" advice). Poison only has to
// be non-negative.
const (
	nightmareResistFloor = 20
	hellResistFloor      = 25
	poisonResistFloor    = 0
	maxResist            = 75 // S: the standard cap applies to the final value, after the penalty
)

// ResistGateDecision is the result of ResistGate.
type ResistGateDecision struct {
	OK  bool
	Why string // empty when OK
	// Post is the post-penalty fire, cold, lightning and poison values that were compared.
	Post [4]int
}

// ResistGate says whether a character with the raw (pre-penalty) resists r may enter difficulty target. Normal always
// passes. Raw means the stat as the game stores it without the difficulty penalty; whether the value read from memory
// already contains the penalty is U (docs/difficulty.md live check 2), in which case the caller passes the read value
// minus the current difficulty's penalty. Pure; the wiring point is the entry decision (combat.EntryGate callers).
func ResistGate(target d2.Difficulty, r d2.Resists) ResistGateDecision {
	pen := target.ResistPenalty()
	post := func(v int) int { return min(v+pen, maxResist) }
	d := ResistGateDecision{Post: [4]int{post(r.Fire), post(r.Cold), post(r.Lightning), post(r.Poison)}}
	floor := 0
	switch target {
	case d2.Nightmare:
		floor = nightmareResistFloor
	case d2.Hell:
		floor = hellResistFloor
	default:
		d.OK = true
		return d
	}
	for i, n := range [3]string{"fire", "cold", "lightning"} {
		if d.Post[i] < floor {
			d.Why = fmt.Sprintf("%s resist %d after the %s penalty, need %d", n, d.Post[i], target, floor)
			return d
		}
	}
	if d.Post[3] < poisonResistFloor {
		d.Why = fmt.Sprintf("poison resist %d after the %s penalty, need %d", d.Post[3], target, poisonResistFloor)
		return d
	}
	d.OK = true
	return d
}

// ReadContainsPenalty is the one assumption behind RawResists: whether the resist stats read from the game already
// include the difficulty penalty. U: docs/difficulty.md live check 2. The default is "no" (the stat list holds the
// equipment and skill sum and the game subtracts the penalty when it displays), the same assumption d2.DoctorLine makes.
// Flip it here if the live check shows the opposite.
var ReadContainsPenalty = false

// RawResists returns the pre-penalty resists of p for ResistGate.
func RawResists(p *d2.Player) d2.Resists {
	r := p.Resist
	if !ReadContainsPenalty {
		return r
	}
	pen := p.Difficulty.ResistPenalty() // read values are post-penalty: undo it
	r.Fire, r.Cold, r.Lightning, r.Poison = r.Fire-pen, r.Cold-pen, r.Lightning-pen, r.Poison-pen
	return r
}
