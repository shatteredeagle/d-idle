package d2dcombat

import (
	"context"
	"sync"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dinput"
)

// Meter is the rig's clock: it steps the game through the Pacer and then reads "state" once, so the smoke tool can
// total damage taken, mana spent and deaths per virtual time without a second reader. One extra verb per Sleep.
type Meter struct {
	P   *d2dinput.Pacer
	C   *d2dctl.Client
	Ctx context.Context

	mu   sync.Mutex
	m    Totals
	last d2dctl.State
	have bool
}

// Totals is what the Meter saw.
type Totals struct {
	Damage    int // sum of life drops between samples (life units as the channel reports them, V hit points)
	ManaSpent int // sum of mana drops between samples (potions and regeneration make this approximate)
	Deaths    int // transitions to life 0
	MinLife   int
	Samples   int
}

func (m *Meter) Now() time.Time { return m.P.Now() }

func (m *Meter) Sleep(d time.Duration) {
	m.P.Sleep(d)
	m.sample()
}

func (m *Meter) sample() {
	ctx := m.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	st, _, err := m.C.State(ctx)
	if err != nil || st.Life.Max == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.have {
		if d := m.last.Life.Cur - st.Life.Cur; d > 0 {
			m.m.Damage += d
		}
		if d := m.last.Mana.Cur - st.Mana.Cur; d > 0 {
			m.m.ManaSpent += d
		}
		if st.Life.Cur <= 0 && m.last.Life.Cur > 0 {
			m.m.Deaths++
		}
	}
	if !m.have || st.Life.Cur < m.m.MinLife {
		m.m.MinLife = st.Life.Cur
	}
	m.m.Samples++
	m.last, m.have = st, true
}

// Totals returns the counters so far.
func (m *Meter) Totals() Totals {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.m
}

// Reset forgets the last sample (call after a cheat that sets life or mana, so the jump is not counted).
func (m *Meter) Reset() {
	m.mu.Lock()
	m.have = false
	m.mu.Unlock()
}
