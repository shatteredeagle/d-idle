package advisor

import "d2lod-bot/internal/vault"

// qualityWord maps the d2 quality numbers the vault stores (1 low, 2 normal, 3 superior, 4 magic ...) to the words Base uses.
var qualityWord = map[int]string{1: "low", 2: "normal", 3: "superior", 4: "magic", 5: "set", 6: "rare", 7: "unique", 8: "crafted"}

// AddVault adds the live vault records to the stock: runes and gems by code, white bases by their sockets. A record
// in the vault is not in the hero yet; the advice line says where it is so the player can Keep it.
func (s *Stock) AddVault(recs []vault.Rec) {
	for _, r := range recs {
		s.add(r.Code, r.Name, r.Chain, qualityWord[r.Quality], r.Sockets, 0, "vault")
	}
}

// Add puts one owned item into the stock. Runes and gems count by code; everything else is a base candidate.
func (s *Stock) Add(code, name string, chain []string, quality string, sockets, filled int, where string) {
	s.add(code, name, chain, quality, sockets, filled, where)
}

func (s *Stock) add(code, name string, chain []string, quality string, sockets, filled int, where string) {
	switch {
	case isRune(code):
		if s.Runes == nil {
			s.Runes = map[string]int{}
		}
		s.Runes[code]++
	case isGem(code):
		if s.Gems == nil {
			s.Gems = map[string]int{}
		}
		s.Gems[code]++
	case sockets > 0:
		s.Bases = append(s.Bases, Base{Name: name, Code: code, Chain: chain, Quality: quality, Sockets: sockets, Filled: filled, Where: where})
	}
}

func isRune(c string) bool {
	if len(c) != 3 || c[0] != 'r' || c[1] < '0' || c[1] > '3' || c[2] < '0' || c[2] > '9' {
		return false
	}
	n := int(c[1]-'0')*10 + int(c[2]-'0')
	return n >= 1 && n <= 33
}

func isGem(c string) bool { return gemName(c) != c }
