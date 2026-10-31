// Package party orchestrates a bot party: which member hosts the LAN game, who joins, what happens when
// members die, leave or the host goes away. It knows nothing about the game: every OOG action and every
// in-game behaviour sits behind an interface (Driver, Follower, Assist) so the state machine is tested
// with fakes. Design source: docs/net/multiplayer.md, docs/classes/party.md; protocol in docs/fleet.md.
package party

import "fmt"

// Role is data in the orchestrator (docs/net/design-now.md), not code in the combat package.
type Role string

const (
	Leader     Role = "leader"     // plans the run, creates the game by preference, others follow it
	Killer     Role = "killer"     // fights; follows the leader or the teleporter's portal
	Teleporter Role = "teleporter" // runs ahead (Teleport), lays the town portal, waits for the killer
	Mule       Role = "mule"       // parks in town, holds loot; stable, so a good host
	Follower   Role = "follower"   // stays within a screen of the leader (XP share needs ~2 screens, V party.md)
	Solo       Role = "solo"       // not in a party; the fleet default
)

// Roles in host-preference order: the host's exit ends the game for everybody (V multiplayer.md), so the
// most stable member hosts. Solo can never host a party.
var hostOrder = []Role{Leader, Mule, Teleporter, Killer, Follower}

var all = map[Role]bool{Leader: true, Killer: true, Teleporter: true, Mule: true, Follower: true, Solo: true}

func (r Role) Valid() bool { return all[r] }

// Parse accepts the empty string as Solo.
func Parse(s string) (Role, error) {
	if s == "" {
		return Solo, nil
	}
	if r := Role(s); r.Valid() {
		return r, nil
	}
	return "", fmt.Errorf("party: unknown role %q", s)
}

// Policy is what a role implies for loot and movement. The real bots read it; here it is only data.
type Policy struct {
	Follows    bool // runs follow-leader
	Assists    bool // runs the assist behaviour (attack what the leader attacks)
	Fights     bool
	PickupLoot bool // false: leaves drops alone (LootPolicy, multiplayer.md section on shared loot)
	CanHost    bool
}

// PolicyFor returns the default policy of a role (U: judgement from docs/classes/party.md, tune live).
func PolicyFor(r Role) Policy {
	switch r {
	case Leader:
		return Policy{Fights: true, PickupLoot: true, CanHost: true}
	case Killer:
		return Policy{Follows: true, Assists: true, Fights: true, CanHost: true}
	case Teleporter:
		return Policy{Fights: true, CanHost: true}
	case Mule:
		return Policy{PickupLoot: true, CanHost: true}
	case Follower:
		return Policy{Follows: true, Assists: true, Fights: true, CanHost: true}
	}
	return Policy{Fights: true, PickupLoot: true}
}
