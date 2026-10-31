// Package d2dloot adapts d2d's ground reads to what internal/loot expects, so the real pickit, pack-room and
// left-behind policy run unchanged on the d2d backend (docs/d2d-loot.md). Picking up itself is the controller's
// Interact on an item (type 4) = "cmd pickup <unit>", which walks to the unit and takes it (V, docs/d2d-probes.md);
// autoloot is off on our runs, so nothing is ever picked up by walking over it.
//
// What the "ground" verb does not give, and what this package does about it:
//   - Quality and item level: only the label text. Quality is guessed from the label against the base name (below, U).
//   - Gold amount: the label "<n> Gold" (V: "8800 Gold"); it becomes the pile's StatGold so the looter's small-pile
//     policy works.
//   - Position: the trailing cell fields; when they are absent the unit is put at the player (the verb walks anyway).
package d2dloot

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/loot"
)

// Source is the snapshot and stat reader the looter is switched to.
type Source struct {
	// Read is d2dsrc.Source.Read (one snapshot round trip with extras).
	Read func(ctx context.Context) (*d2dsrc.Result, error)
	DB   *data.DB

	mu   sync.Mutex
	gold map[uint32]int32 // pile unit id -> amount, from the last snapshot
}

// Attach switches l to read through a Source built on src and returns it.
func Attach(l *loot.Looter, src *d2dsrc.Source, db *data.DB) *Source {
	s := &Source{Read: src.Read, DB: db}
	l.UseSource(s.Snapshot, s.Stats)
	return s
}

// Snapshot reads one snapshot and enriches its ground units.
func (s *Source) Snapshot() (*d2.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := s.Read(ctx)
	if err != nil {
		return nil, err
	}
	s.Enrich(res)
	return res.Snap, nil
}

// Stats answers the looter's stat read for a ground gold pile (Addr = unit id on d2d, d2denv.Reader).
func (s *Source) Stats(addr uint32) (d2.Stats, error) {
	s.mu.Lock()
	n, ok := s.gold[addr]
	s.mu.Unlock()
	if !ok {
		return nil, d2.ErrUnsupported
	}
	return d2.Stats{d2.StatGold: n}, nil
}

// Enrich fills Addr, position, quality and gold amounts of res.Snap.Ground in place.
func (s *Source) Enrich(res *d2dsrc.Result) {
	snap := res.Snap
	gold := map[uint32]int32{}
	for i := range snap.Ground {
		u := &snap.Ground[i]
		u.Addr = u.ID
		if u.X < 0 && snap.Player != nil {
			u.X, u.Y = snap.Player.X, snap.Player.Y
		}
		ex := res.Extras.Items[u.ID]
		if n, ok := GoldAmount(ex.Label); ok && ex.Code == "gld" {
			gold[u.ID] = int32(n)
			continue
		}
		if u.Item != nil && u.Item.Quality == 0 {
			u.Item.Quality = s.quality(u.Class, ex.Label)
		}
	}
	s.mu.Lock()
	s.gold = gold
	s.mu.Unlock()
}

var goldRe = regexp.MustCompile(`^\s*([0-9]+)\s+gold\b`)

// GoldAmount parses a pile label such as "8800 Gold".
func GoldAmount(label string) (int, bool) {
	m := goldRe.FindStringSubmatch(strings.ToLower(label))
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

var lowPrefixes = []string{"low quality ", "crude ", "damaged ", "cracked "}

// quality guesses the quality of a ground item from its label (U: only the Superior and low-quality prefixes are
// confirmed to show, docs/d2d-fidelity.md section 4; magic and rare labels were not seen yet, so those two rules
// follow the game's naming scheme: magic = affixes around the base name, rare/unique = a name that does not contain
// the base name). Anything that cannot be judged is Normal, which the pickit treats as plain junk.
func (s *Source) quality(class int, label string) int {
	if s.DB == nil || class < 0 || class >= len(s.DB.Items) || s.DB.Items[class] == nil || label == "" {
		return d2.QualityNormal
	}
	b := s.DB.Items[class]
	base := strings.ToLower(b.Display)
	if base == "" {
		base = strings.ToLower(b.Name)
	}
	l := strings.ToLower(strings.TrimSpace(label))
	if base == "" || l == base {
		return d2.QualityNormal
	}
	if strings.HasPrefix(l, "superior ") && strings.TrimPrefix(l, "superior ") == base {
		return d2.QualitySuperior
	}
	for _, p := range lowPrefixes {
		if strings.HasPrefix(l, p) && strings.TrimPrefix(l, p) == base {
			return d2.QualityLow
		}
	}
	if b.Type == "gold" || b.Stackable {
		return d2.QualityNormal // "Arrows (30)", scroll stacks: label carries a count, not an affix
	}
	if strings.Contains(l, base) {
		return d2.QualityMagic
	}
	return d2.QualityRare
}
