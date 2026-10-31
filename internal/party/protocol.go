package party

import (
	"fmt"
	"strings"
	"sync"
)

// Game is what members need to meet in one LAN game. The name and password are shared through the
// in-process Directory, NEVER through game chat (multiplayer.md: chat is a human channel only).
//
// Sharing protocol (documented in docs/fleet.md):
//  1. The orchestrator picks the host and makes a Game with a fresh name (NewGame) and password.
//  2. It Publishes the Game to the Directory under the party id; the host's Driver.Create is called with it.
//  3. Only after Create returns nil (the host is in the game, TCP 4000 bound) do members get Driver.Join(Game),
//     in staggered order. A joiner that cannot read the Directory entry does not join.
//  4. On host change or game end the old entry is Revoked and a new Game (new name) is published, so a
//     stale name can never lead a member into a dead or foreign game.
type Game struct {
	Name       string // 1..15 chars (U: client limit; kept short and plain ASCII)
	Password   string // 0..15 chars
	Host       string // address joiners type into the TCP/IP screen, "127.0.0.1" on one machine
	Difficulty string // "normal", "nightmare" or "hell", chosen by the host
	MaxPlayers int    // party size, 2..4 (the game cap is 8, V party.md)
}

// Public is the game without its password, for status output and logs.
func (g Game) Public() Game { g.Password = ""; return g }

// Directory holds the current Game of each party; safe for concurrent use.
type Directory struct {
	mu sync.Mutex
	m  map[string]Game
}

func NewDirectory() *Directory { return &Directory{m: map[string]Game{}} }

func (d *Directory) Publish(party string, g Game) {
	d.mu.Lock()
	d.m[party] = g
	d.mu.Unlock()
}

func (d *Directory) Lookup(party string) (Game, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	g, ok := d.m[party]
	return g, ok
}

func (d *Directory) Revoke(party string) {
	d.mu.Lock()
	delete(d.m, party)
	d.mu.Unlock()
}

// GameName builds a short, unique name: first letters of the party id plus a generation counter ("pty3").
func GameName(party string, gen int) string {
	p := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, party)
	if len(p) > 8 {
		p = p[:8]
	}
	return fmt.Sprintf("%s%d", p, gen)
}
