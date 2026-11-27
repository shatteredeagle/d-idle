package town

import (
	"context"

	"d2lod-bot/internal/d2"
)

// PortalByVerbForTest runs the use-item portal cast on tool (input layer I3).
func (t *Town) PortalByVerbForTest(ctx context.Context, tool d2.Unit) (d2.Unit, bool, error) {
	return t.portalByVerb(ctx, tool)
}
