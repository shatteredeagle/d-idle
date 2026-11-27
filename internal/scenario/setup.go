package scenario

import (
	"fmt"
	"sort"
	"strings"

	"d2lod-bot/internal/telemetry"
)

// Verb is one lab verb of the declared SETUP with the reason written to the ledger before it is sent.
type Verb struct {
	Line   string `json:"line"`
	Reason string `json:"reason"`
}

// Setup is what the builder produces from a Pre: lab verbs for the prep d2d (lab verbs on), the board record for the
// hero's plan store, and the provenance tags. It is deterministic: the same scenario, class and seed give the same Setup
// (no clock, no random, no checked-in save blob).
type Setup struct {
	Verbs []Verb `json:"verbs"`
	// Board is the chunk the player accepted; the runner writes it with the real questboard store (<hero>.quests.json).
	Board *QuestRef `json:"board,omitempty"`
	// BoardAlso are further accepted chunks, written after Board in order.
	BoardAlso []QuestRef `json:"board_also,omitempty"`
	// Tuning are knob overrides for the bot run (state/tuning.json), after the scenario's own Trigger.Knobs.
	Tuning map[string]any `json:"tuning,omitempty"`
	// Build is trigger.build: the runner writes it as levelerBuild into the bot's config.json.
	Build string `json:"build,omitempty"`
	// HeroProfile is the profile the hero is prepared with ("played" or "banked", Scenario.HeroProfile). For "played" the
	// runner spends the points after Verbs with the hero library (cmd/d2scenario), which needs the game tables.
	HeroProfile string `json:"hero_profile"`
	// Tags classify the setup for docs/probes/test-characters.md: CHEATED when the state could not be reached by play
	// (a warped position, a given quest item, a hand-set quest word), SETUP-only otherwise.
	Tags []string `json:"tags"`
	// LifePct is the life percentage the runner brings the hero to AFTER the verbs, with `debug hit <dmg>` (V docs/forkcheats.md:
	// the real hurt path), the damage computed from the live life: 0 means leave it alone. It is a gameplay write before the
	// start, a declared SETUP.
	LifePct int `json:"life_pct,omitempty"`
	// Field is true when the hero is warped and the live d2d is handed to the bot (pre.start field).
	Field bool `json:"field"`
	// Seeds are the steps that need a read of the game between verbs (a unit id): worn items and vault records, in order.
	// The runner does them after Verbs (cmd/d2scenario/seed.go).
	Seeds []Seed `json:"seeds,omitempty"`
	// Ether is the wallet balance to credit before the start.
	Ether int `json:"ether,omitempty"`
	// VaultVerbs is true when the d2d of the run needs --vault-verbs (a vault seed, or the scenario's needs list an item verb).
	VaultVerbs bool `json:"vault_verbs,omitempty"`
	// Unsupported lists declared preconditions the builder cannot construct yet (no verb known). A run with any is BLOCKED.
	Unsupported []string `json:"unsupported,omitempty"`
}

// BuildSetup turns a scenario's Pre into the setup for one class and seed. The seed is only used to name the hero so
// two runs never share a save directory; the map seed is passed to d2d by the runner.
func BuildSetup(s Scenario, class string, seed int64) (Setup, error) {
	if _, err := ClassIndex(class); err != nil {
		return Setup{}, err
	}
	p := s.Pre
	reason := func(what string) string {
		return fmt.Sprintf("SETUP: scenario %s (%s, seed %d): %s", s.ID, class, seed, what)
	}
	su := Setup{HeroProfile: s.HeroProfile()}
	add := func(what, line string) { su.Verbs = append(su.Verbs, Verb{Line: line, Reason: reason(what)}) }
	tags := map[string]bool{}

	if p.Level > 1 {
		exp, ok := telemetry.DefaultExpForLevel(p.Level)
		if !ok {
			return Setup{}, fmt.Errorf("scenario %s: no experience for level %d", s.ID, p.Level)
		}
		add("level", fmt.Sprintf("debug stat 12 %d", p.Level))
		add("experience of that level", fmt.Sprintf("debug stat 13 %d", exp))
		add("stat points a played level gives", fmt.Sprintf("debug statpts %d", 5*(p.Level-1)))
		add("skill points a played level gives", fmt.Sprintf("debug skillpts %d", p.Level-1))
	}
	for _, st := range p.Stats {
		if st.Stat < 0 || st.Stat > 400 {
			return Setup{}, fmt.Errorf("scenario %s: stat id %d", s.ID, st.Stat)
		}
		add(fmt.Sprintf("raw stat %d", st.Stat), fmt.Sprintf("debug stat %d %d", st.Stat, st.Value))
		tags["stat-write"] = true
	}
	if p.Gold > 0 {
		add("purse", fmt.Sprintf("debug stat 14 %d", p.Gold))
	}
	if p.Stash > 0 {
		add("stash gold", fmt.Sprintf("debug stat 15 %d", p.Stash))
	}
	switch p.Difficulty {
	case "normal", "":
	case "nightmare":
		add("difficulty", "debug difficulty 1") // U: argument form of the fork verb
		tags["difficulty"] = true
	case "hell":
		add("difficulty", "debug difficulty 2")
		tags["difficulty"] = true
	default:
		return Setup{}, fmt.Errorf("scenario %s: difficulty %q", s.ID, p.Difficulty)
	}
	for _, w := range p.Waypoints {
		if w < 0 { // -1: every waypoint (the fork's `debug wp all`), for scenarios that start deep in an act
			add("every waypoint", "debug wp all")
		} else {
			add(fmt.Sprintf("waypoint bit %d", w), fmt.Sprintf("debug wp %d", w))
		}
		tags["waypoint"] = true
	}
	for _, q := range p.Quests {
		add(fmt.Sprintf("quest record %d", q.Record), fmt.Sprintf("debug quest set %d 0x%04x", q.Record, q.Word))
		tags["quest-word"] = true
	}
	var floor [][2]string // ground items, dropped after the position verb: a drop before it lands where the hero stood (the town)
	items := append(append([]Item(nil), p.Items...), p.Pack...)
	for _, it := range items {
		n := it.Count
		if n <= 0 {
			n = 1
		}
		flags := ""
		if it.Quality > 0 {
			flags += fmt.Sprintf(" q%d", it.Quality)
		}
		if it.Unid {
			flags += " unid"
		}
		switch it.Where {
		case "", "pack":
			for i := 0; i < n; i++ {
				add("item "+it.Code, fmt.Sprintf("debug give %s pack%s", it.Code, flags))
			}
		case "ground": // the verb without a place drops the item beside the hero and takes the quality as a bare number (V docs/forkcheats.md)
			if it.Quality == 0 {
				su.Unsupported = append(su.Unsupported, fmt.Sprintf("item %s on the ground needs a quality", it.Code))
				break
			}
			for i := 0; i < n; i++ {
				line := fmt.Sprintf("debug give %s %d", it.Code, it.Quality)
				if it.Unid {
					line += " unid"
				}
				floor = append(floor, [2]string{"item " + it.Code + " on the ground", line})
			}
		default:
			su.Unsupported = append(su.Unsupported, fmt.Sprintf("item %s into %s (no setup verb known)", it.Code, it.Where))
		}
		tags["given-item"] = true
	}
	if len(p.Belt) > 0 {
		add("empty the belt", "debug belt clear")
		for i, code := range p.Belt {
			add("belt box "+fmt.Sprint(i), fmt.Sprintf("debug give %s belt %d", code, i))
		}
	}
	if p.Merc != nil {
		add("hired merc", fmt.Sprintf("debug merc hire %d", *p.Merc))
		if p.MercLevel > 0 {
			add("merc level", fmt.Sprintf("debug merc level %d", p.MercLevel))
			add("merc life", "debug merc heal")
		}
		tags["merc"] = true
	}
	if p.LifePct > 0 && p.LifePct < 100 {
		su.LifePct = p.LifePct
	}
	if p.Start == StartField {
		add("position (the save cannot hold a field position)", fmt.Sprintf("debug goto %d", p.Area))
		su.Field = true
		tags["position"] = true
	}
	if p.RoomsUp {
		if p.Start != StartField {
			return Setup{}, fmt.Errorf("scenario %s: pre.rooms_up needs start field", s.ID)
		}
		add("every room of the level up (the NPCs of a town exist only then)", "debug rooms up")
	}
	if p.Warp != nil {
		if p.Start != StartField {
			return Setup{}, fmt.Errorf("scenario %s: pre.warp needs start field", s.ID)
		}
		add(fmt.Sprintf("hero position: cell %d,%d", p.Warp.X, p.Warp.Y), fmt.Sprintf("debug warp %d %d", p.Warp.X, p.Warp.Y))
		tags["position"] = true
	}
	for _, sp := range p.Spawn {
		if p.Start != StartField {
			return Setup{}, fmt.Errorf("scenario %s: pre.spawn needs start field", s.ID)
		}
		if sp.Super != "" {
			if sp.Monster != "" || sp.N > 1 || sp.Mod != "" {
				return Setup{}, fmt.Errorf("scenario %s: a super spawn takes only the name", s.ID)
			}
			add("spawn superunique "+sp.Super, "debug spawn super "+sp.Super)
			tags["spawn"] = true
			continue
		}
		if sp.Monster == "" {
			return Setup{}, fmt.Errorf("scenario %s: spawn needs monster or super", s.ID)
		}
		switch sp.Mod {
		case "", "champion", "unique":
		default:
			return Setup{}, fmt.Errorf("scenario %s: spawn mod %q", s.ID, sp.Mod)
		}
		n := max(sp.N, 1)
		line := fmt.Sprintf("debug spawn %s %d", sp.Monster, n)
		if sp.Mod != "" {
			line += " " + sp.Mod
		}
		add(fmt.Sprintf("spawn %d %s %s", n, sp.Monster, sp.Mod), line)
		tags["spawn"] = true
	}
	for _, o := range p.Objects {
		if p.Start != StartField {
			return Setup{}, fmt.Errorf("scenario %s: pre.objects needs start field", s.ID)
		}
		if o.Class <= 0 {
			return Setup{}, fmt.Errorf("scenario %s: object class %d", s.ID, o.Class)
		}
		line := fmt.Sprintf("debug object add %d", o.Class)
		if o.X != 0 || o.Y != 0 {
			line += fmt.Sprintf(" %d %d", o.X, o.Y)
		}
		add(fmt.Sprintf("object class %d", o.Class), line)
		tags["object"] = true
	}
	for _, w := range p.Worn {
		su.Seeds = append(su.Seeds, Seed{Kind: SeedWorn, Code: w.Code, Slot: BodySlot[w.Slot], SlotName: w.Slot,
			Give: GiveLine(w.Code, w.Quality, w.Unid), Reason: reason("worn " + w.Code + " in " + w.Slot)})
		tags["worn"] = true
	}
	if p.Vault != nil {
		su.Seeds = append(su.Seeds, vaultSeeds(*p.Vault, reason)...)
		tags["vault-records"] = true
		su.VaultVerbs = true
	}
	if p.Ether > 0 {
		su.Ether = p.Ether
		tags["ether"] = true
	}
	for _, n := range s.Needs {
		if strings.HasPrefix(n, "item.") && n != "item.equip" && n != "item.identify" {
			su.VaultVerbs = true
		}
	}
	for _, f := range floor {
		add(f[0], f[1])
	}
	su.Board = p.Board
	su.BoardAlso = p.BoardAlso
	su.Build = s.Trigger.Build
	if p.Board != nil {
		su.Tuning = map[string]any{"quests.board": 1}
	}
	for k, v := range s.Trigger.Knobs {
		if su.Tuning == nil {
			su.Tuning = map[string]any{}
		}
		su.Tuning[k] = v
	}
	cheated := tags["worn"] || tags["vault-records"] || tags["ether"] || tags["position"] || tags["waypoint"] || tags["given-item"] || tags["quest-word"] || tags["stat-write"] || tags["merc"] || tags["difficulty"]
	if cheated {
		su.Tags = append(su.Tags, "CHEATED")
	} else {
		su.Tags = append(su.Tags, "SETUP-only")
	}
	var names []string
	for t := range tags {
		names = append(names, t)
	}
	sort.Strings(names)
	su.Tags = append(su.Tags, names...)
	return su, nil
}

// HeroName is the throwaway character name of one run: stable per scenario, class and seed, valid for the board store
// (2 to 24 letters, digits, '_' and '-', starting with a letter) and distinct from any played hero.
func HeroName(scenarioID, class string, seed int64) string {
	var b []byte
	b = append(b, 'P', 'g')
	for _, r := range scenarioID {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b = append(b, byte(r))
		}
		if len(b) >= 10 {
			break
		}
	}
	b = append(b, class[0]|0x20)
	n := fmt.Sprint(seed)
	if len(n) > 5 {
		n = n[len(n)-5:]
	}
	return string(b) + n
}

// Seed kinds.
const (
	SeedWorn  = "worn"
	SeedVault = "vault"
)

// Seed is one ordered step of the seeding: give the item, find its unit id, then wear it (worn) or take it into the vault.
type Seed struct {
	Kind     string `json:"kind"`
	Code     string `json:"code"`
	Give     string `json:"give"` // the `debug give` line
	Slot     int    `json:"slot,omitempty"`
	SlotName string `json:"slot_name,omitempty"`
	Pin      bool   `json:"pin,omitempty"`
	Unid     bool   `json:"unid,omitempty"`
	Bare     bool   `json:"bare,omitempty"` // vault seed: store the record without held affixes
	Sealed   string `json:"sealed,omitempty"`
	// Filler marks a record whose item is any distinct real item (the count of a full vault); Fill is its running number.
	Filler bool   `json:"filler,omitempty"`
	Fill   int    `json:"fill,omitempty"`
	Reason string `json:"reason"`
}

// GiveLine is the verb that puts an item in the pack (V docs/forkcheats.md: `give <code> pack q<n> [unid]`).
func GiveLine(code string, quality int, unid bool) string {
	l := "debug give " + code + " pack"
	if quality > 0 {
		l += fmt.Sprintf(" q%d", quality)
	}
	if unid {
		l += " unid"
	}
	return l
}

// FillerCodes are base items of every class that fit a pack: the first fillers are one normal item of each, later ones
// are magic items (their affixes differ), so the vault never sees two identical records.
var FillerCodes = []string{"cap", "lea", "buc", "ssd", "hax", "clb", "sst", "bsd", "lgl", "lbt", "lbl", "spr", "jav", "xsh", "scl", "stu", "wnd", "hbt", "gts", "cst"}

func vaultSeeds(v VaultSeed, reason func(string) string) []Seed {
	var out []Seed
	pin := func(i int) bool { return i < v.Pinned }
	for _, it := range v.Items {
		for k := 0; k < max(it.N, 1); k++ {
			out = append(out, Seed{Kind: SeedVault, Code: it.Code, Give: GiveLine(it.Code, it.Quality, it.Unid), Unid: it.Unid, Sealed: it.Sealed, Bare: it.Bare,
				Pin: it.Pin || pin(len(out)), Reason: reason("vault record " + it.Code)})
		}
	}
	q, _ := v.FillerQuality()
	for f := 0; f < v.Count; f++ {
		code, quality := FillerCodes[f%len(FillerCodes)], q
		if f >= len(FillerCodes) {
			quality = 4
		}
		out = append(out, Seed{Kind: SeedVault, Code: code, Give: GiveLine(code, quality, false), Filler: true, Fill: f,
			Pin: pin(len(out)), Reason: reason(fmt.Sprintf("vault filler record %d", f))})
	}
	return out
}
