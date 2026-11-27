package d2dctl

import "context"

// CmdUse sends "cmd use <item id>": the client's own use-item message (cmd::UseItem, C->S 0x20), what a right-click on
// the item in the pack sends. For a Scroll or Tome of Town Portal the world opens the portal through the same handler
// as the click (World::read_portal; V: fork source read, apps/d2d/devctl/verbs.hpp and components/game/world/commands.cpp
// at the 2026-11-25 rebase). Like every cmd verb it is a request: the world applies it at its next tick and refuses
// silently (town, dead, busy, an item on the cursor), so callers verify the effect from state. No lab flag (U on a
// build older than the rebase).
func (c *Client) CmdUse(ctx context.Context, id int) error {
	return c.ok(ctx, "cmd use %d", id)
}
