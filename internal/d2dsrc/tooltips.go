package d2dsrc

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Item tooltips (docs/inventory-tooltip.md). The fork's "dump items" has no affix list, but the stock "items" verb
// prints the tooltip text the game itself builds, so the inventory page can show the game's own words. Off by default
// (Options.Tooltips): it is one more round trip, so the lines are cached by item id and refetched only when the set of
// carried items changes or tipMaxAge has passed (durability and quantity lines change slowly). S: the verb and the
// "name first" layout come from docs/d2d-spike.md section 7; U live: that the ids of "items" equal the ids of "dump items"
// and whether stash and cube items are listed (the spike says they may not be).
const tipMaxAge = 30 * time.Second

type tipCache struct {
	mu    sync.Mutex
	key   string
	at    time.Time
	lines map[uint32][]string
	off   bool // the binary has no items verb: stop asking
}

// attachTips fills ItemExtra.Tooltip of the items just read. Errors never fail the snapshot.
func (b *builder) attachTips(ctx context.Context) {
	c := &b.s.tips
	ids := make([]int, 0, len(b.res.Extras.Items))
	for id := range b.res.Extras.Items {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	var sb strings.Builder
	for _, id := range ids {
		sb.WriteString(strconv.Itoa(id))
		sb.WriteByte(',')
	}
	key := sb.String()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.off {
		return
	}
	if c.lines == nil || c.key != key || time.Since(c.at) > tipMaxAge {
		items, _, err := b.s.c.Items(ctx)
		if err != nil {
			if isCmdErr(err) {
				c.off = true
				b.s.warnOnce("tooltips", "the items verb failed, item tooltips are off: %v", err)
			}
			return
		}
		c.lines = map[uint32][]string{}
		for _, it := range items {
			if id, e := strconv.Atoi(it.Attrs["id"]); e == nil && len(it.Tooltip) > 0 {
				c.lines[uint32(id)] = it.Tooltip
			}
		}
		c.key, c.at = key, time.Now()
	}
	for id, ex := range b.res.Extras.Items {
		if l, ok := c.lines[id]; ok {
			ex.Tooltip = l
			b.res.Extras.Items[id] = ex
		}
	}
}
