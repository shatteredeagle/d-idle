package watch

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/unlock"
)

type queueRig struct {
	*funRig
	eng  *unlock.Engine
	halt string
}

func newQueueRig(t *testing.T, class string, level int) *queueRig {
	t.Helper()
	l, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	eng, err := unlock.Open(t.TempDir(), strings.ToLower(class), l, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := &queueRig{eng: eng}
	q.funRig = newFunRigWith(t, class, level, func(r *funRig, fc *FunConfig) {
		fc.Unlock = eng
		fc.OrderQueue = &OrderQueueConfig{Quote: r.strip.Quote, Halt: func() string { return q.halt }}
	})
	return q
}

func (q *queueRig) denCleared() { q.feed(questEv(task.QDen, 2, 0, 1)) }

func (q *queueRig) post(body string) (int, string) {
	resp, b := q.do("POST", q.answer, OrderQueueRoute, funToken, body)
	return resp.StatusCode, string(b)
}

func (q *queueRig) qview() OrderQueueView {
	var v OrderQueueView
	q.getJSON(OrderQueueRoute, &v)
	return v
}

func (q *queueRig) add(t *testing.T, area string, extra string) QueueEntry {
	t.Helper()
	code, b := q.post(`{"action":"add","kind":"go","area":"` + area + `"` + extra + `}`)
	if code != 200 {
		t.Fatalf("add %s: %d %s", area, code, b)
	}
	var rep queueReply
	if err := json.Unmarshal([]byte(b), &rep); err != nil || rep.Entry == nil {
		t.Fatalf("add reply %s", b)
	}
	return *rep.Entry
}

func (q *queueRig) openOrders() []Order {
	var out []Order
	for _, o := range q.hub.OrdersView().Orders {
		if o.State == OrderQueued || o.State == OrderActive {
			out = append(out, o)
		}
	}
	return out
}

func (q *queueRig) finishRunning(state string) {
	for _, o := range q.openOrders() {
		q.hub.OrderState(o.ID, state, "")
	}
}

func TestOrderQueueLockedUntilTheDenAndSlotsOpenInTurn(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			q := newQueueRig(t, class, 5)
			v := q.qview()
			if v.Open || v.Teaser == "" || v.Capacity != 0 || len(v.Slots) != 0 {
				t.Fatalf("locked view: %+v", v)
			}
			if code, b := q.post(`{"action":"add","kind":"go","area":"Blood Moor","minutes":10}`); code != http.StatusForbidden || !strings.Contains(b, `"locked"`) {
				t.Fatalf("locked add: %d %s", code, b)
			}
			q.feed(questEv(task.QDen, 1, 0, 1)) // started is not cleared
			if q.eng.Allowed(OrderQueueID) {
				t.Fatal("the Den quest started opened the queue")
			}
			q.denCleared()
			if c := q.fun.Queue.Capacity(); c != 1 {
				t.Fatalf("capacity after the Den: %d", c)
			}
			q.feed(questEv(task.QBurial, 3, 0, 2)) // Kashya's reward
			if c := q.fun.Queue.Capacity(); c != 2 {
				t.Fatalf("capacity after Kashya: %d", c)
			}
			q.feed(questEv(task.QTools, 3, 0, 3)) // Charsi after Kashya: the same row, no third slot
			if c := q.fun.Queue.Capacity(); c != 2 {
				t.Fatalf("capacity after Charsi too: %d", c)
			}
			q.feed(questEv(task.QCain, 3, 0, 4))
			if c := q.fun.Queue.Capacity(); c != 3 {
				t.Fatalf("capacity after Cain: %d", c)
			}
			// a replay of the same events changes nothing and reveals nothing twice
			q.feed(questEv(task.QDen, 2, 0, 1), questEv(task.QBurial, 3, 0, 2), questEv(task.QCain, 3, 0, 4))
			n := 0
			for _, l := range storyLines(q.funRig) {
				if strings.Contains(l, "Order queue") {
					n++
				}
			}
			if n != 3 || q.fun.Queue.Capacity() != 3 {
				t.Fatalf("replay: %d reveal lines, capacity %d", n, q.fun.Queue.Capacity())
			}
			b, _ := json.MarshalIndent(q.qview(), "", "  ")
			golden(t, "orderqueue-"+strings.ToLower(class), string(b)+"\n")
		})
	}
}

func TestOrderQueueRunsOrdersInTurn(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			q := newQueueRig(t, class, 5)
			q.denCleared()
			q.feed(questEv(task.QBurial, 3, 0, 2), questEv(task.QCain, 3, 0, 3))
			q.add(t, "Blood Moor", `,"minutes":10`)
			q.add(t, "Den of Evil", `,"minutes":20`)
			q.add(t, "Blood Moor", `,"until":"level","level":6`)
			if code, b := q.post(`{"action":"add","kind":"go","area":"Blood Moor","minutes":5}`); code != http.StatusConflict || !strings.Contains(b, "queue-full") {
				t.Fatalf("fourth entry: %d %s", code, b)
			}
			if got := q.openOrders(); len(got) != 0 {
				t.Fatalf("an order was placed before the sequencer ran: %v", got)
			}
			q.fun.Settle()
			open := q.openOrders()
			if len(open) != 1 || open[0].Area != "Blood Moor" || open[0].Minutes != 10 {
				t.Fatalf("first order: %+v", open)
			}
			q.fun.Settle() // still running: nothing else is placed
			if len(q.openOrders()) != 1 {
				t.Fatalf("two open orders: %+v", q.openOrders())
			}
			q.hub.OrderState(open[0].ID, OrderActive, "")
			q.finishRunning(OrderDone)
			q.fun.Settle()
			open = q.openOrders()
			if len(open) != 1 || open[0].Area != "Den of Evil" || open[0].Minutes != 20 {
				t.Fatalf("second order: %+v", open)
			}
			q.finishRunning(OrderDone)
			q.fun.Settle()
			open = q.openOrders()
			if len(open) != 1 || !open[0].Forever {
				t.Fatalf("third order (until level, placed as endless): %+v", open)
			}
			q.fun.Settle() // level 5 < 6
			if len(q.openOrders()) != 1 {
				t.Fatal("the level order ended early")
			}
			q.hub.SetHero(HeroInfo{Class: class, Level: 6, Area: "Blood Moor"})
			q.fun.Settle()
			if len(q.openOrders()) != 0 {
				t.Fatalf("the level order did not end at level 6: %+v", q.openOrders())
			}
			v := q.qview()
			states := []string{}
			for _, e := range v.Entries {
				states = append(states, e.State)
			}
			if strings.Join(states, ",") != "done,done,done" || v.Entries[2].Note != "reached level 6" || v.Stopped != "" {
				t.Fatalf("entries: %+v", v.Entries)
			}
			q.fun.Settle()
			if len(q.openOrders()) != 0 || len(q.hub.OrdersView().Orders) != 3 {
				t.Fatal("extra orders appeared after the queue ran out")
			}
		})
	}
}

func TestOrderQueueStopsOnDeathAndResumes(t *testing.T) {
	q := newQueueRig(t, "Paladin", 5)
	q.denCleared()
	q.feed(questEv(task.QBurial, 3, 0, 2))
	q.add(t, "Blood Moor", `,"minutes":10`)
	q.add(t, "Den of Evil", `,"minutes":10`)
	q.fun.Settle()
	first := q.openOrders()[0]
	q.hub.OrderState(first.ID, OrderActive, "")
	q.feed(goals.Event{Type: goals.EvDeath, Area: "Blood Moor", Seq: 10}, goals.Event{Type: goals.EvDeath, Area: "Blood Moor", Seq: 11})
	q.hub.OrderState(first.ID, OrderFailed, "the hero died")
	q.fun.Settle()
	v := q.qview()
	if v.Stopped != "the hero died" || len(q.openOrders()) != 0 || v.Entries[0].State != QueueFailed || v.Entries[1].State != QueueWaiting {
		t.Fatalf("after death: %+v", v)
	}
	if code, _ := q.post(`{"action":"resume"}`); code != 200 {
		t.Fatal("resume")
	}
	q.fun.Settle()
	if open := q.openOrders(); len(open) != 1 || open[0].Area != "Den of Evil" {
		t.Fatalf("after resume: %+v", open)
	}
}

func TestOrderQueueHaltStopsAndGuardRefusalGoesOn(t *testing.T) {
	q := newQueueRig(t, "Sorceress", 5)
	q.denCleared()
	q.feed(questEv(task.QBurial, 3, 0, 2))
	q.add(t, "Blood Moor", `,"minutes":10`)
	q.add(t, "Den of Evil", `,"minutes":10`)
	q.halt = "a Pact paused the hero"
	q.fun.Settle()
	if v := q.qview(); v.Stopped != "a Pact paused the hero" || len(q.openOrders()) != 0 {
		t.Fatalf("halt: %+v", v)
	}
	q.halt = ""
	q.post(`{"action":"resume"}`)
	q.fun.Settle()
	first := q.openOrders()[0]
	// the bot's route guard refuses the walk: the order ends failed with the guard's reason and the queue goes on to the next
	q.hub.OrderState(first.ID, OrderFailed, "route guard (no merc, clvl 5 < 16)")
	q.fun.Settle()
	if open := q.openOrders(); len(open) != 1 || open[0].Area != "Den of Evil" {
		t.Fatalf("after a guard refusal: %+v", open)
	}
	if e := q.qview().Entries[0]; e.State != QueueFailed || !strings.Contains(e.Note, "route guard") {
		t.Fatalf("entry: %+v", e)
	}
}

func TestOrderQueueCancelsAndPlayersOrdersGoFirst(t *testing.T) {
	q := newQueueRig(t, "Paladin", 5)
	q.denCleared()
	q.feed(questEv(task.QBurial, 3, 0, 2), questEv(task.QCain, 3, 0, 3))
	// a player's own order is open: the queue waits for it
	if _, err := q.hub.Give(OrderRequest{Kind: OrderGo, Area: "Crypt", Minutes: 10}); err != nil {
		t.Fatal(err)
	}
	a := q.add(t, "Blood Moor", `,"minutes":10`)
	b := q.add(t, "Den of Evil", `,"minutes":10`)
	q.fun.Settle()
	if open := q.openOrders(); len(open) != 1 || open[0].Area != "Crypt" {
		t.Fatalf("queue ran over a player's order: %+v", open)
	}
	q.finishRunning(OrderDone)
	q.fun.Settle()
	if open := q.openOrders(); len(open) != 1 || open[0].Area != "Blood Moor" {
		t.Fatalf("%+v", open)
	}
	// cancelling the waiting entry removes it; cancelling the running one cancels the hero's order and the queue goes on
	if code, _ := q.post(`{"action":"cancel","id":` + qid(b.ID) + `}`); code != 200 {
		t.Fatal("cancel waiting")
	}
	c := q.add(t, "Den of Evil", `,"minutes":5`)
	if code, _ := q.post(`{"action":"cancel","id":` + qid(a.ID) + `}`); code != 200 {
		t.Fatal("cancel running")
	}
	q.fun.Settle()
	v := q.qview()
	if v.Entries[0].State != QueueCancelled || v.Entries[1].State != QueueCancelled || v.Stopped != "" ||
		v.Entries[2].ID != c.ID || v.Entries[2].State != QueueRunning {
		t.Fatalf("after cancelling the running entry: %+v", v)
	}
	if code, _ := q.post(`{"action":"cancel","id":` + qid(c.ID) + `}`); code != 200 {
		t.Fatal("cancel the next running one")
	}
	if code, _ := q.post(`{"action":"cancel","id":99}`); code != http.StatusNotFound {
		t.Fatalf("unknown id: %d", code)
	}
	// the player cancels the running order from the orders box: hands off, the queue stops
	d := q.add(t, "Blood Moor", `,"minutes":10`)
	q.add(t, "Den of Evil", `,"minutes":10`)
	q.fun.Settle()
	if err := q.hub.CancelOrder(q.openOrders()[0].ID); err != nil {
		t.Fatal(err)
	}
	q.fun.Settle()
	if v := q.qview(); !strings.Contains(v.Stopped, "cancelled") || v.Entries[len(v.Entries)-2].ID != d.ID {
		t.Fatalf("%+v", v)
	}
	if code, _ := q.post(`{"action":"clear"}`); code != 200 || len(q.openOrders()) != 0 {
		t.Fatal("clear")
	}
}

func qid(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestOrderQueueCharterFlowAndKey(t *testing.T) {
	q := newQueueRig(t, "Sorceress", 5)
	q.denCleared()
	// above range (Cold Plains is for level 6): 402 with the price, nothing queued
	code, b := q.post(`{"action":"add","kind":"go","area":"Cold Plains","minutes":10}`)
	if code != http.StatusPaymentRequired || !strings.Contains(b, "charter-needed") || !strings.Contains(b, `"price":20`) {
		t.Fatalf("%d %s", code, b)
	}
	if len(q.qview().Entries) != 0 {
		t.Fatal("a refused charter queued an entry")
	}
	if _, err := q.strip.Wallet.Ledger.Grant("tally", "grant", "g1", 25, "test"); err != nil {
		t.Fatal(err)
	}
	e := q.add(t, "Cold Plains", `,"minutes":10,"approve":true,"key":"k1"`)
	again := q.add(t, "Cold Plains", `,"minutes":10,"approve":true,"key":"k1"`)
	if again.ID != e.ID || len(q.qview().Entries) != 1 {
		t.Fatalf("a retry with the same key queued twice: %+v", q.qview().Entries)
	}
	q.fun.Settle()
	if open := q.openOrders(); len(open) != 1 || open[0].Area != "Cold Plains" {
		t.Fatalf("%+v", open)
	}
	if bal, _ := q.strip.Wallet.Ledger.Balance("tally"); bal.Spendable != 5 {
		t.Errorf("charter paid at placement: balance %d", bal.Spendable)
	}
}

func TestOrderQueueBuySlotThroughTheLedgerOnce(t *testing.T) {
	q := newQueueRig(t, "Paladin", 5)
	q.denCleared()
	q.feed(funStream()...) // tally 53
	buy := func(id string) (int, string) {
		resp, b := q.do("POST", q.answer, UnlocksBuyRoute, funToken, `{"id":"`+id+`"}`)
		return resp.StatusCode, string(b)
	}
	if v := q.qview(); v.Capacity != 1 || len(v.Slots) != 3 || v.Slots[1].Price != 30 || v.Slots[2].Price != 75 {
		t.Fatalf("slots: %+v", v)
	}
	if code, b := buy("orderqueue3"); code != http.StatusPaymentRequired {
		t.Fatalf("short for slot 3: %d %s", code, b)
	}
	before, _ := q.strip.Wallet.Ledger.Balance("tally")
	if code, b := buy("orderqueue2"); code != 200 {
		t.Fatalf("buy slot 2: %d %s", code, b)
	}
	if code, _ := buy("orderqueue2"); code != http.StatusConflict {
		t.Fatalf("second buy: %d", code)
	}
	after, _ := q.strip.Wallet.Ledger.Balance("tally")
	if before.Spendable-after.Spendable != 30 {
		t.Fatalf("spent %d, want 30 once", before.Spendable-after.Spendable)
	}
	if q.fun.Queue.Capacity() != 2 {
		t.Fatalf("capacity %d", q.fun.Queue.Capacity())
	}
}

func TestOrderQueuePostGatesAndBadBodies(t *testing.T) {
	q := newQueueRig(t, "Sorceress", 5)
	q.denCleared()
	body := `{"action":"add","kind":"go","area":"Blood Moor","minutes":10}`
	if resp, _ := q.do("POST", q.answer, OrderQueueRoute, "", body); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: %d", resp.StatusCode)
	}
	if resp, _ := q.do("GET", q.answer, OrderQueueRoute, funToken, ""); resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET on the answer listener: %d", resp.StatusCode)
	}
	if resp, _ := q.do("POST", q.view, OrderQueueRoute, funToken, body); resp.StatusCode == 200 {
		t.Fatal("the viewer must not take the queue route")
	}
	req, _ := http.NewRequest("POST", q.answer+OrderQueueRoute, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+funToken)
	req.Header.Set("Content-Type", "text/plain")
	if resp, err := q.client.Do(req); err != nil || resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("content type: %v %v", resp, err)
	}
	req, _ = http.NewRequest("POST", q.answer+OrderQueueRoute, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+funToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.example")
	if resp, err := q.client.Do(req); err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("origin: %v %v", resp, err)
	}
	for _, bad := range []string{
		`{"action":"add","kind":"go","area":"Blood Moor","minutes":10,"x":1}`,
		`{"action":"fly"}`,
		`{"action":"add","kind":"go","area":"Moo Moo Farm","minutes":10}`,
		`{"action":"add","kind":"go","area":"Blood Moor","until":"level","level":5}`,
		`{"action":"add","kind":"go","area":"Blood Moor","until":"level"}`,
		`{"action":"add","kind":"go","area":"Blood Moor","until":"level","level":9,"minutes":5}`,
		`{"action":"add","kind":"home","until":"level","level":9}`,
		`{"action":"add","kind":"go","area":"Blood Moor","until":"sometime"}`,
		`{"action":"add","kind":"go","area":"Blood Moor"}`,
	} {
		if code, b := q.post(bad); code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", bad, code, b)
		}
	}
	if n := len(q.qview().Entries); n != 0 {
		t.Fatalf("%d entries from bad bodies", n)
	}
}

func TestOrderQueueTableAndMissingRows(t *testing.T) {
	for _, bad := range []string{
		`{"schema":"x","slots":["orderqueue"],"maxUntilLevel":99}`,
		`{"schema":"watch.orderqueue/1","slots":["orderqueue2"],"maxUntilLevel":99}`,
		`{"schema":"watch.orderqueue/1","slots":["orderqueue","orderqueue"],"maxUntilLevel":99}`,
		`{"schema":"watch.orderqueue/1","slots":["orderqueue"],"maxUntilLevel":1}`,
	} {
		if _, err := ParseQueueTable([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	l, _ := unlock.Shipped()
	eng, _ := unlock.Open("", "p", l, nil)
	tbl := &QueueTable{Schema: orderQueueSchema, Slots: []string{OrderQueueID, "orderqueue9"}, MaxUntilLevel: 99}
	r := newFunRig(t, "Paladin", 1)
	if _, err := NewOrderQueue(&Unlocks{Engine: eng}, r.hub, OrderQueueConfig{Table: tbl}); err == nil || !strings.Contains(err.Error(), `"orderqueue9"`) {
		t.Fatalf("unknown slot row: %v", err)
	}
}
