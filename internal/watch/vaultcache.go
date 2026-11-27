package watch

import (
	"context"
	"fmt"
	"sync"

	"d2lod-bot/internal/bulkid"
	"d2lod-bot/internal/vault"
)

// vaultCache keeps what a poll of /api/vault would otherwise recompute over every record: the mass reveal preview and the
// type facets. Entries are keyed by the store's Version (any write changes it) and, for the preview, by the price inputs it
// was computed from, so a stale answer is never served. One entry each: the page polls with one filter-free query.
type vaultCache struct {
	mu      sync.Mutex
	store   *vault.Store
	massKey string
	mass    bulkid.Mass
	bar     bulkid.Bar
	gold    *VaultGoldInfo
	hasMass bool
	facetV  uint64
	facetS  *vault.Store
	facet   map[string]int
}

func (c *vaultCache) facets(st *vault.Store) map[string]int {
	if c == nil {
		return st.Facets().Types
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	ver := st.Version()
	if c.facet == nil || c.facetS != st || c.facetV != ver {
		c.facet, c.facetS, c.facetV = st.Facets().Types, st, ver
	}
	return c.facet // read only for the caller
}

// massPreview fills the Reveal block's Mass, Bar and Gold.Unrevealed from the cache or, on a miss, from one pass over the
// unidentified records.
func (d *VaultDesk) massPreview(ctx context.Context, st *vault.Store, act *VaultActView) {
	r := act.Reveal
	var gi VaultGoldInfo
	ga, hasGold := d.Act.(VaultGoldActor)
	if hasGold {
		gi = ga.GoldInfo(ctx)
	}
	order := ""
	if gi.Order != nil {
		order = gi.Order.ID + "/" + gi.Order.State
	}
	c := d.cache
	var key string
	if c != nil {
		// Version first, then read the records: a write in between only makes the next poll recompute.
		key = fmt.Sprintf("%d|%v|%d|%d|%d|%v|%d|%v|%d|%d|%d|%v|%s|%s", st.Version(), r.Free, r.Charges, r.EtherPer, r.Balance, r.Methods, d.ScrollGold,
			gi.On, gi.ScrollGold, gi.Stash, gi.ConfirmAt, gi.Decode, gi.Why, order)
		c.mu.Lock()
		if c.hasMass && c.store == st && c.massKey == key {
			m, b := c.mass, c.bar
			var g *VaultGoldInfo
			if c.gold != nil {
				cp := gi
				cp.Unrevealed = c.gold.Unrevealed
				g = &cp
			}
			c.mu.Unlock()
			act.Reveal.Mass, act.Reveal.Bar, act.Reveal.Gold = &m, &b, g
			return
		}
		c.mu.Unlock()
	}
	no := false
	recs, _ := st.Query(vault.Filter{Identified: &no})
	m := bulkid.MassReveal(d.MassFilter, recs, d.Quality, bulkid.Price{Free: r.Free, Charges: r.Charges, ScrollGold: d.ScrollGold, EtherPer: r.EtherPer, EtherBal: r.Balance})
	b := d.bar(ctx, recs, r, gi, hasGold)
	var g *VaultGoldInfo
	if hasGold {
		gi.Unrevealed = len(bulkid.UnrevealedIDs(recs, gi.Decode))
		g = &gi
	}
	act.Reveal.Mass, act.Reveal.Bar, act.Reveal.Gold = &m, b, g
	if c != nil {
		c.mu.Lock()
		c.store, c.massKey, c.mass, c.bar, c.gold, c.hasMass = st, key, m, *b, g, true
		c.mu.Unlock()
	}
}
