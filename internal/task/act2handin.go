package task

import (
	"strconv"
	"strings"
	"sync"

	questspecdata "d2lod-bot/data/questspec"
	"d2lod-bot/internal/questspec"
)

// The Act 2 hand-in (kill -> Tyrael -> Jerhyn -> Meshif) is read from the Seven Tombs quest bits. The masks have one
// owner, the questspec rows of data/questspec/act2.json (talk_tyrael, talk_jerhyn; bit order S from the d2d port,
// unverified live): a node's done check is that row's post clause, its ready check the row's quest_bits pre clause.
// The stage alone cannot say it: the kill raises the word to Cleared and it stays there until bit 0 at Meshif.

type specClauses struct {
	tyraelPost, jerhynPre, jerhynPost questspec.Clause
	err                               error
}

var act2Spec = sync.OnceValue(func() specClauses {
	var c specClauses
	data, err := questspecdata.FS.ReadFile("act2.json")
	if err != nil {
		c.err = err
		return c
	}
	chains, err := questspec.LoadFile(data)
	if err != nil {
		c.err = err
		return c
	}
	find := func(chain, step string, post bool) questspec.Clause {
		for _, ch := range chains {
			for _, s := range ch.Steps {
				if ch.ID != chain || s.ID != step {
					continue
				}
				src := s.Pre
				if post {
					src = s.Post
				}
				for _, raw := range src {
					if cl, err := questspec.ParseClause(raw); err == nil && hasBits(cl) {
						return cl
					}
				}
			}
		}
		if c.err == nil {
			c.err = &specMissing{chain + "/" + step}
		}
		return nil
	}
	c.tyraelPost = find("seven_tombs", "talk_tyrael", true)
	c.jerhynPre = find("seven_tombs", "talk_jerhyn", false)
	c.jerhynPost = find("seven_tombs", "talk_jerhyn", true)
	return c
})

type specMissing struct{ step string }

func (e *specMissing) Error() string {
	return "questspec act2: step " + e.step + " has no quest_bits clause"
}

// Act2HandinSpecErr is nil when the hand-in clauses load from the embedded questspec (a test pins it).
func Act2HandinSpecErr() error { return act2Spec().err }

func hasBits(c questspec.Clause) bool {
	for _, p := range c {
		if p.Name == "quest_bits" {
			return true
		}
	}
	return false
}

var specStages = map[string]Stage{"started": StageStarted, "cleared": StageCleared, "done": StageDone}

// holdsClause is a questspec clause read against the World: quest_bits(q,mask) every bit of mask in the reader's
// word, quest_state(q,stage) the quest's stage (with what the world proves). Other facts read false.
func holdsClause(w World, c questspec.Clause) bool {
	for _, p := range c {
		if len(p.Args) != 2 {
			continue
		}
		switch p.Name {
		case "quest_bits":
			m, err := strconv.ParseUint(strings.TrimPrefix(p.Args[1], "0x"), 16, 16)
			if err == nil && m != 0 && w.Quests[p.Args[0]].Raw&uint16(m) == uint16(m) {
				return true
			}
		case "quest_state":
			if s, ok := specStages[p.Args[1]]; ok && a2Stage(w, p.Args[0]) >= s {
				return true
			}
		}
	}
	return false
}

// durielWord: the reader gives the Seven Tombs word (0 is unknown or untouched; after the kill it is never 0).
func durielWord(w World) bool { return w.Quests[QDuriel].Raw != 0 }

// orStage reads the clause when the word is known, else the stage fallback (proxies only, or a reader without the
// raw word): there the bits cannot be seen and the old stage checks apply.
func orStage(get func(specClauses) questspec.Clause, fallback Pred) Pred {
	return func(w World) bool {
		if !durielWord(w) {
			return fallback(w)
		}
		return holdsClause(w, get(act2Spec()))
	}
}

var (
	durielDead = a2At(QDuriel, StageCleared)
	// tyraelTalked: Tyrael's line set its bits (or the quest is done).
	tyraelTalked = orStage(func(c specClauses) questspec.Clause { return c.tyraelPost }, a2At(QDuriel, StageDone))
	// jerhynOpen: what Jerhyn's reward line needs (Tyrael's bits).
	jerhynOpen = orStage(func(c specClauses) questspec.Clause { return c.jerhynPre }, durielDead)
	// jerhynTalked: Jerhyn's reward bit is set (or the quest is done). Without the word: the Act 2 word.
	jerhynTalked = orStage(func(c specClauses) questspec.Clause { return c.jerhynPost }, a2At(QAct2Word, StageDone))
	// meshifOpen: Meshif waits for Jerhyn's bit. Without the word: Duriel dead, as before.
	meshifOpen = orStage(func(c specClauses) questspec.Clause { return c.jerhynPost }, durielDead)
)
