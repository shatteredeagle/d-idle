package combat

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// Positive counters of the upkeep drivers' shadow (combat.deliveryUpkeep 1). The shadow logs only disagreements, so
// a quiet log alone cannot tell "the driver agreed" from "no comparison ran"; these count every comparison and every
// agreement per driver kind. They are cumulative for the Fighter's life and atomic: the observer reads them for the
// minute line while the fight loop writes them.
type upkeepKind int

const (
	cmpBuff   upkeepKind = iota // the buff pass: per skill, the curse split and the due answer
	cmpAura                     // keepAura: the aura's up answer, and the recheck between checks
	cmpMinion                   // the minion driver's pick against the old class loops
	cmpMark                     // the mark driver: no old-side pick to compare, agree = the driver would cast nothing
	cmpStance                   // the army stance: the old code has none, agree = the driver would take no step
	cmpKinds
)

var upkeepKindNames = [cmpKinds]string{"buff", "aura", "minion", "mark", "stance"}

type upkeepCmp struct {
	cmp, agree [cmpKinds]atomic.Int64
}

func (u *upkeepCmp) note(k upkeepKind, agree bool) {
	u.cmp[k].Add(1)
	if agree {
		u.agree[k].Add(1)
	}
}

// UpkeepCompare returns the shadow comparison counts per driver kind and whether the knob is on.
func (f *Fighter) UpkeepCompare() (kinds []string, cmp, agree []int64, on bool) {
	if f.upkeepMode() == 0 {
		return nil, nil, nil, false
	}
	for k := range cmpKinds {
		kinds = append(kinds, upkeepKindNames[k])
		cmp = append(cmp, f.upkCmp.cmp[k].Load())
		agree = append(agree, f.upkCmp.agree[k].Load())
	}
	return kinds, cmp, agree, true
}

// UpkeepCompareField is the minute line value, agree/cmp per kind ("buff:12/12,aura:30/30,..."); "" while the knob is 0.
func (f *Fighter) UpkeepCompareField() string {
	kinds, cmp, agree, on := f.UpkeepCompare()
	if !on {
		return ""
	}
	var b strings.Builder
	for i, k := range kinds {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%s:%d/%d", k, agree[i], cmp[i])
	}
	return b.String()
}
