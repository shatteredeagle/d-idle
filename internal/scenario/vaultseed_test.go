package scenario

import (
	"encoding/json"
	"strings"
	"testing"
)

var twoClasses = []string{"Paladin", "Sorceress"}

func vaultScenario(pre Pre, as ...Assert) Scenario {
	s := Scenario{ID: "vt", Title: "t", Kind: KindBehaviour, Tier: TierFast, Pre: pre, Budget: Budget{VirtualMin: 5},
		Matrix: Matrix{Classes: twoClasses, Seeds: []int64{1}}, Assert: as}
	if len(as) == 0 {
		s.Assert = []Assert{{Kind: ANoWrites}}
	}
	if s.Pre.HeroProfile == "" {
		s.Pre.HeroProfile = HeroBanked
	}
	s.applyDefaults()
	return s
}

func TestBuildSetupSeeds(t *testing.T) {
	pre := Pre{
		Level: 14, Ether: 35,
		Pack:  []Item{{Code: "isc"}},
		Worn:  []WornItem{{Code: "rin", Slot: "lring"}, {Code: "cap", Slot: "head", Quality: 4}},
		Vault: &VaultSeed{Count: 2, Pinned: 1, Quality: "normal", Items: []VaultIt{{Code: "cap", Quality: 4, Unid: true}}},
	}
	s := vaultScenario(pre)
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, class := range twoClasses {
		su, err := BuildSetup(s, class, 11)
		if err != nil {
			t.Fatal(err)
		}
		if !su.VaultVerbs || su.Ether != 35 || len(su.Unsupported) != 0 {
			t.Fatalf("%s: vault verbs %v ether %d unsupported %v", class, su.VaultVerbs, su.Ether, su.Unsupported)
		}
		var gives []string
		for _, sd := range su.Seeds {
			gives = append(gives, sd.Kind+":"+sd.Give)
			if !strings.Contains(sd.Reason, class) {
				t.Errorf("%s: seed reason %q does not name the class", class, sd.Reason)
			}
		}
		want := []string{
			"worn:debug give rin pack", "worn:debug give cap pack q4",
			"vault:debug give cap pack q4 unid", "vault:debug give cap pack q2", "vault:debug give lea pack q2",
		}
		if strings.Join(gives, "|") != strings.Join(want, "|") {
			t.Fatalf("%s: seeds %v\nwant %v", class, gives, want)
		}
		if su.Seeds[0].Slot != BodySlot["lring"] || su.Seeds[1].Slot != 1 {
			t.Fatalf("slots %d %d", su.Seeds[0].Slot, su.Seeds[1].Slot)
		}
		pins := 0
		for _, sd := range su.Seeds {
			if sd.Pin {
				pins++
			}
		}
		if pins != 1 || !su.Seeds[2].Pin {
			t.Fatalf("pinned: %d, first vault record pinned %v", pins, su.Seeds[2].Pin)
		}
		if len(su.Verbs) == 0 || !strings.Contains(su.Verbs[len(su.Verbs)-1].Line, "debug give isc pack") {
			t.Fatalf("pre.pack item must be a give verb: %+v", su.Verbs)
		}
		if su.Tags[0] != "CHEATED" || !contains(su.Tags, "worn") || !contains(su.Tags, "vault-records") || !contains(su.Tags, "ether") {
			t.Fatalf("tags %v", su.Tags)
		}
	}
}

func TestSeedValidationMustFail(t *testing.T) {
	cases := map[string]Pre{
		"worn slot":         {Worn: []WornItem{{Code: "cap", Slot: "forehead"}}},
		"worn no code":      {Worn: []WornItem{{Slot: "head"}}},
		"vault empty":       {Vault: &VaultSeed{}},
		"vault capacity":    {Vault: &VaultSeed{Count: 1, Capacity: 7}},
		"vault quality":     {Vault: &VaultSeed{Count: 1, Quality: "legendary"}},
		"vault pinned over": {Vault: &VaultSeed{Count: 1, Pinned: 2}},
		"vault item code":   {Vault: &VaultSeed{Items: []VaultIt{{Code: "x"}}}},
		"ether negative":    {Ether: -1},
	}
	for name, pre := range cases {
		if err := vaultScenario(pre).Validate(); err == nil {
			t.Errorf("%s: want a validation error", name)
		}
	}
	if err := vaultScenario(Pre{Vault: &VaultSeed{Count: 400, Capacity: 400}}).Validate(); err != nil {
		t.Errorf("a capacity at the table base is fine: %v", err)
	}
}

func TestResolver(t *testing.T) {
	inv := []byte(`{"backpack":[{"id":7,"code":"isc","name":"Scroll of Identify"},{"id":42,"code":"cap","name":"Cap of Fire"},{"id":43,"code":"cap","name":"Cap of Fire"}],"equipment":{"head":{"id":9,"code":"cap","name":"Cap of Fire"},"neck":null}}`)
	r := Resolver{VaultIDs: []string{"v-aa", "v-bb"}, Inventory: inv}
	body, err := r.ResolveBody(map[string]any{
		"ids":    []any{"{{vault.1.id}}"},
		"id":     "{{pack.item[name=cap].id}}",
		"two":    "{{pack.item[code=cap,nth=1].id}}",
		"worn":   "{{worn.head.id}}",
		"mix":    "item-{{vault.0.id}}!",
		"bycode": "{{pack.item[code=isc].id}}",
	})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(body)
	want := `{"bycode":7,"id":42,"ids":["v-bb"],"mix":"item-v-aa!","two":43,"worn":9}`
	if string(b) != want {
		t.Fatalf("got %s\nwant %s", b, want)
	}
	if p, err := r.ResolveString("/api/vault/equip/{{vault.0.id}}"); err != nil || p != "/api/vault/equip/v-aa" {
		t.Fatalf("path %q %v", p, err)
	}
	bad := []string{"{{vault.2.id}}", "{{vault.x.id}}", "{{pack.item[code=zzz].id}}", "{{pack.item[code=cap,nth=5].id}}", "{{pack.item[kind=cap].id}}",
		"{{pack.item[].id}}", "{{worn.neck.id}}", "{{worn.rring.id}}", "{{nothing}}"}
	for _, expr := range bad {
		if _, err := r.ResolveBody(map[string]any{"x": expr}); err == nil {
			t.Errorf("%s must not resolve", expr)
		}
	}
	if _, err := (Resolver{}).ResolveBody(map[string]any{"x": "{{pack.item[name=cap].id}}"}); err == nil {
		t.Error("an unread inventory must be an error, not a zero id")
	}
	if !NeedsInventory(map[string]any{"a": []any{"{{worn.head.id}}"}}) || NeedsInventory(map[string]any{"a": "{{vault.0.id}}"}) || !HasTemplate("x{{vault.0.id}}") || HasTemplate("plain") {
		t.Error("NeedsInventory / HasTemplate")
	}
}

func TestVaultAssertions(t *testing.T) {
	api := json.RawMessage(`{"capacity":{"level":"full","count":400},"missed":{"count":2},"autoTrim":false}`)
	st := State{Has: []string{"vault"}, Vault: &VaultFacts{
		Records: []VaultRecFact{{ID: "a", Code: "cap", State: "recorded", Seeded: true}, {ID: "b", Code: "rin", State: "recorded", Seeded: true}, {ID: "c", Code: "cap", State: "trimmed"}},
		Journal: []string{"recorded cap v-1", "revealed rin v-2", "skipped (full)", "equipped cap worn in head"},
		API:     api,
	}}
	one := 1
	three := 3
	f := false
	cases := []struct {
		name string
		a    Assert
		ok   bool
	}{
		{"count in range", Assert{Kind: AVaultCount, Min: 2, Max: &two}, true},
		{"count too many", Assert{Kind: AVaultCount, Max: &one}, false},
		{"seeded present", Assert{Kind: AVaultPresent}, true},
		{"log recorded", Assert{Kind: AVaultLog, Pattern: `^recorded cap`}, true},
		{"log equipped none", Assert{Kind: AVaultLog, Pattern: `^equipped .* head`, Max: new(int)}, false},
		{"log skipped", Assert{Kind: AVaultLog, Pattern: `skipped \(full\)`, Min: 1}, true},
		{"log sold must be zero", Assert{Kind: AVaultLog, Pattern: `^sold`, Max: new(int)}, true},
		{"revealed none", Assert{Kind: AVaultRevealed, Min: 1}, false},
		{"missed min", Assert{Kind: AVaultMissed, Min: 2}, true},
		{"missed too few", Assert{Kind: AVaultMissed, Min: 3}, false},
		{"missed at most", Assert{Kind: AVaultMissed, Max: &three}, true},
		{"api equals string", Assert{Kind: AVaultAPI, Path: "capacity.level", Equals: "full"}, true},
		{"api equals wrong", Assert{Kind: AVaultAPI, Path: "capacity.level", Equals: "ok"}, false},
		{"api equals bool", Assert{Kind: AVaultAPI, Path: "autoTrim", Equals: f}, true},
		{"api number", Assert{Kind: AVaultAPI, Path: "capacity.count", Equals: float64(400)}, true},
		{"api min", Assert{Kind: AVaultAPI, Path: "missed.count", Min: 1}, true},
		{"api no such field", Assert{Kind: AVaultAPI, Path: "nope.x", Equals: 1}, false},
	}
	for _, c := range cases {
		if err := c.a.Validate(); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		res := Evaluate([]Assert{c.a}, st)
		if len(res) != 1 || res[0].OK != c.ok || res[0].Unobserved {
			t.Errorf("%s: %+v, want ok=%v", c.name, res, c.ok)
		}
	}
	st.Vault.Records[0].Unlocked = true
	if r := Evaluate([]Assert{{Kind: AVaultRevealed, Min: 1, Max: &one}}, st); !r[0].OK {
		t.Errorf("one unlocked live record is revealed: %+v", r)
	}
	// a trimmed seeded record fails vault_records_present, an unread vault is UNOBSERVED, not red
	st.Vault.Records[1].State = "trimmed"
	if r := Evaluate([]Assert{{Kind: AVaultPresent}}, st); r[0].OK {
		t.Errorf("a trimmed seeded record must fail: %+v", r)
	}
	if r := Evaluate([]Assert{{Kind: AVaultCount, Min: 1}}, State{}); !r[0].Unobserved {
		t.Errorf("no vault facts must be unobserved: %+v", r)
	}
	for _, bad := range []Assert{{Kind: AVaultLog}, {Kind: AVaultLog, Pattern: "("}, {Kind: AVaultCount}, {Kind: AVaultAPI, Path: "a"}, {Kind: AVaultAPI, Equals: 1}, {Kind: AVaultPresent, Which: "all"}} {
		if bad.Validate() == nil {
			t.Errorf("%+v must not validate", bad)
		}
	}
}

var two = 2

func TestAPIReplyFieldAssertion(t *testing.T) {
	idx := 0
	st := State{APIReplies: []APIReply{{Index: 0, Path: "/api/vault/equip", Status: 200, OK: true, Body: json.RawMessage(`{"ok":true,"order":{"state":"queued","slot":"head"}}`)}}}
	pass := Assert{Kind: AAPIReply, Index: &idx, Path: "order.state", Equals: "queued"}
	fail := Assert{Kind: AAPIReply, Index: &idx, Path: "order.state", Equals: "done"}
	missing := Assert{Kind: AAPIReply, Index: &idx, Path: "order.nope", Equals: "x"}
	for _, a := range []Assert{pass, fail, missing} {
		if err := a.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	r := Evaluate([]Assert{pass, fail, missing}, st)
	if !r[0].OK || r[1].OK || r[2].OK {
		t.Fatalf("%+v", r)
	}
	if (Assert{Kind: AAPIReply, Index: &idx, Path: "a"}).Validate() == nil {
		t.Error("a path with nothing to compare must not validate")
	}
}

func TestBuildJournal(t *testing.T) {
	storeLog := `{"seq":1,"ev":"add","id":"v-1","item":{"code":"cap","name":"Cap of Fire"}}
{"seq":2,"ev":"add","id":"v-2","item":{"code":"rin"}}
{"seq":3,"ev":"pin","id":"v-2"}
{"seq":4,"ev":"add","id":"v-3","item":{"code":"hax"}}
{"seq":5,"ev":"unseal","id":"v-1"}
{"seq":6,"ev":"rstage","id":"v-1"}
{"seq":7,"ev":"add","id":"v-9","item":{"cod`
	bot := `{"time":"2026-11-21T10:00:00Z","msg":"vault full: new records are stopped until room is made","err":"vault: full"}
{"time":"2026-11-21T10:00:01Z","msg":"equip: Cap worn in head","level":"INFO"}
{"time":"2026-11-21T10:00:02Z","msg":"something else"}`
	got := BuildJournal([]byte(storeLog), 3, []byte(bot))
	want := []string{"recorded hax v-3", "revealed cap v-1", "skipped (full)", "equipped Cap worn in head"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("%q\nwant %q", got, want)
	}
}

func TestShippedScenariosStillLoad(t *testing.T) {
	if _, err := LoadDir("../../tools/provingground/scenarios"); err != nil {
		t.Fatal(err)
	}
}

// vault_sold with an index: the records' gold, the reply's sale.gold and the stash read-back must agree exactly.
func TestVaultSoldExactAgainstTheReply(t *testing.T) {
	two := 2
	mk := func(body string, g1, g2 int) State {
		return State{Has: []string{"vault"}, Vault: &VaultFacts{Records: []VaultRecFact{{ID: "a", Code: "cap", State: "sold", Gold: g1}, {ID: "b", Code: "lea", State: "sold", Gold: g2}, {ID: "c", Code: "hax", State: "recorded"}}},
			APIReplies: []APIReply{{Index: 0, Status: 200, OK: true, Body: json.RawMessage(body)}}}
	}
	idx := 0
	a := Assert{Kind: AVaultSold, Min: 2, Max: &two, Index: &idx}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	good := mk(`{"sale":{"gold":30,"stashBefore":7200,"stashAfter":7230}}`, 10, 20)
	if r := Evaluate([]Assert{a}, good); !r[0].OK {
		t.Fatalf("agreeing numbers: %+v", r)
	}
	for name, st := range map[string]State{
		"records differ from the reply": mk(`{"sale":{"gold":31,"stashBefore":7200,"stashAfter":7231}}`, 10, 20),
		"the stash grew by another sum": mk(`{"sale":{"gold":30,"stashBefore":7200,"stashAfter":7229}}`, 10, 20),
		"no sale in the reply":          mk(`{"ok":true}`, 10, 20),
	} {
		if r := Evaluate([]Assert{a}, st); r[0].OK {
			t.Errorf("%s must fail: %+v", name, r)
		}
	}
	left := mk(`{"sale":{"gold":10,"stashBefore":0,"stashAfter":10}}`, 10, 0)
	left.Vault.Records[1].State = "recorded"
	if r := Evaluate([]Assert{a}, left); r[0].OK {
		t.Errorf("a record still in the vault must fail: %+v", r)
	}
	if r := Evaluate([]Assert{{Kind: AVaultSold, Min: 1}}, good); !r[0].OK {
		t.Errorf("a plain count: %+v", r)
	}
}
