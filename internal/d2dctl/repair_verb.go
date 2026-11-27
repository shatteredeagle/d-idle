package d2dctl

import (
	"context"
	"fmt"
	"strings"
)

// The devp14 verbs of the private fork (docs/process/sandbox-devp14.md, docs/design/input-i4.md).
// "cmd repair <item|-1>" is the client's repair message (C->S 0x35); -1 repairs everything, what the Repair All button
// sends. "menu choose <text>" picks the open NPC dialog's line containing text. U on a live town: the fork's headless
// test never reached the vendor, and the I4 differential is a fork-side job (the fork change list is in input-i4.md).
// A refusal is a request that the world dropped, so callers read the effect (worn durability, the window flag) back.

// CmdRepair sends "cmd repair <id>"; id -1 is Repair All. The smith window must be open.
func (c *Client) CmdRepair(ctx context.Context, id int) error {
	return c.ok(ctx, "cmd repair %d", id)
}

// MenuChoose sends "menu choose <text>". The text is one line of the dialog, no newline.
func (c *Client) MenuChoose(ctx context.Context, text string) error {
	if text = strings.TrimSpace(text); text == "" || strings.ContainsAny(text, "\r\n") {
		return fmt.Errorf("d2dctl: menu choose: bad text %q", text)
	}
	return c.ok(ctx, "menu choose %s", text)
}
