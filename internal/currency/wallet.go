package currency

import "sync"

// Wallet is what the watch screen reads: the registry, the ledger (account balances) and the per-character counts the
// bot reports (the pack cannot be read from the ledger). Safe for concurrent use.
type Wallet struct {
	Mode   string
	Reg    *Registry
	Ledger *Ledger // nil in vanilla mode

	mu        sync.Mutex
	character string
	carried   map[string]int
}

// NewWallet builds a wallet for a ledger; mode is the overlay stamp.
func NewWallet(mode string, l *Ledger) *Wallet {
	w := &Wallet{Mode: mode, Ledger: l, Reg: Default()}
	if l != nil {
		w.Reg = l.Registry()
	}
	return w
}

// SetCarried records how many units of an item currency the hero carries now. The bot calls it after each inventory read.
func (w *Wallet) SetCarried(character, currency string, n int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.carried == nil || w.character != character {
		w.carried, w.character = map[string]int{}, character
	}
	w.carried[currency] = n
}

// Carried returns the last reported count; known is false until the bot has reported one.
func (w *Wallet) Carried(currency string) (n int, known bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, known = w.carried[currency]
	return
}
