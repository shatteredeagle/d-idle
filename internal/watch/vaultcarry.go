package watch

import (
	"context"
	"net/http"
)

// Carry and Stow (docs/charms.md, user decision 2026-11-27): the player moves an identified charm from the Vault into the
// backpack, where it works, and back. The bot never does either by itself.
const (
	VaultCarryRoute = "/api/vault/carry" // POST {ids:[one id]}: the charm goes to the backpack under the keep mark
	VaultStowRoute  = "/api/vault/stow"  // POST {ids:[one id of a carried charm]}: it goes back into the vault
)

// VaultCarrier is the optional part of a VaultEquipper that can carry charms (cmd/d2bot adapts internal/vaultkeep).
type VaultCarrier interface {
	CarryReady() (ok bool, why string)
	Carry(ctx context.Context, id string) VaultKeepResult
	Stow(ctx context.Context, id string) VaultKeepResult
	Carried() []VaultCarried
}

// VaultCarried is one charm in the backpack that came out of the Vault (the id is its old record id, the one Stow takes).
type VaultCarried struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Code  string `json:"code"`
	Cells int    `json:"cells"`
}

// VaultCarryView is in GET /api/vault: whether Carry is offered, why not, and what is carried.
type VaultCarryView struct {
	Enabled bool           `json:"enabled"`
	Why     string         `json:"why,omitempty"`
	Carried []VaultCarried `json:"carried"`
	Cells   int            `json:"cells"` // pack cells the carried charms take
}

func (d *VaultDesk) carrier() VaultCarrier {
	if d == nil || d.Equip == nil {
		return nil
	}
	c, _ := d.Equip.(VaultCarrier)
	return c
}

func (d *VaultDesk) carryView() *VaultCarryView {
	c := d.carrier()
	if c == nil {
		return nil
	}
	ok, why := c.CarryReady()
	if !d.on() {
		ok, why = false, "the vault is read only while loot.vault is 0"
	}
	v := &VaultCarryView{Enabled: ok, Why: why, Carried: c.Carried()}
	if v.Carried == nil {
		v.Carried = []VaultCarried{}
	}
	for _, x := range v.Carried {
		v.Cells += x.Cells
	}
	return v
}

// carry takes ONE id: Carry for a vault record, Stow for a carried charm. The reply has the keep shape, so the page's
// toast code is shared; a refusal is a 409 with the code and the sentence of the keeper.
func (a *VaultAPI) carry(ctx context.Context, w http.ResponseWriter, d *VaultDesk, route string, b *vaultBody) {
	c := d.carrier()
	if c == nil {
		writeJSON(w, http.StatusConflict, policyReply{Code: "off", Error: "this bot cannot carry charms"})
		return
	}
	if len(b.IDs) != 1 {
		writeJSON(w, http.StatusBadRequest, policyReply{Code: "ids", Error: "send one id"})
		return
	}
	var res VaultKeepResult
	if route == VaultCarryRoute {
		res = c.Carry(ctx, b.IDs[0])
	} else {
		res = c.Stow(ctx, b.IDs[0])
	}
	rep := vaultReply{Done: []string{}}
	if res.OK {
		rep.Done = append(rep.Done, res.ID)
		rep.OK, rep.Where = true, res.Where
		writeJSON(w, 200, rep)
		return
	}
	rep.Failed = append(rep.Failed, vaultFail{ID: res.ID, Code: res.Code, Error: res.Why})
	writeJSON(w, http.StatusConflict, rep)
}
