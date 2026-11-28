package shadowlog

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"d2lod-bot/internal/agency"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/lootrules"
)

// logTo writes records the way the bot does (slog JSON handler), so the tests read real line shapes.
func logTo() (*slog.Logger, *bytes.Buffer) {
	var b bytes.Buffer
	return slog.New(slog.NewJSONHandler(&b, nil)), &b
}

func digest(t *testing.T, b *bytes.Buffer, fam string) Digest {
	t.Helper()
	rep, err := Read(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range rep.Families {
		if d.Family == fam {
			return d
		}
	}
	t.Fatalf("no family %q", fam)
	return Digest{}
}

func count(rows []Row, key string) int {
	for _, r := range rows {
		if r.Key == key {
			return r.Count
		}
	}
	return 0
}

// plannerLine is the bot's own line (agency.ShadowLine), so a change of its format fails here and not on a night log.
func plannerLine(class, attack, prior, aura, priorAura string, agree bool) string {
	pl := agency.Plan{Attack: &agency.Choice{Skill: attack, Level: 7}, Prior: prior, Best: attack, PriorAura: priorAura, Archetype: "caster"}
	if aura != "" {
		pl.Aura = &agency.Choice{Skill: aura, Level: 3}
	}
	return agency.ShadowLine(agency.View{Class: class, Level: 24, Agrees: &agree}, pl)
}

func TestPlannerPerClass(t *testing.T) {
	log, b := logTo()
	// A Paladin whose attack agrees and whose aura does not, a Necromancer that agrees: skill names with spaces.
	for i := 0; i < 3; i++ {
		log.Info(plannerLine("paladin", "Blessed Hammer", "Blessed Hammer", "Might", "Concentration", false))
	}
	log.Info(plannerLine("necromancer", "Bone Spear", "Bone Spear", "", "", true))
	log.Info(plannerLine("necromancer", "Bone Spear", "Bone Spear", "", "", true))
	d := digest(t, b, Planner)
	if d.Verdict != Review || d.Lines != 5 || d.Differ != 3 || d.Unparsed != 0 {
		t.Fatalf("planner digest: %+v", d)
	}
	if got := count(d.Rows, "class=paladin plan=Blessed Hammer profile=Blessed Hammer aura=Might auraProfile=Concentration"); got != 3 {
		t.Errorf("the Paladin aura difference must be one row of 3, rows %+v", d.Rows)
	}
	if count(d.Totals, "class=necromancer lines") != 2 || count(d.Totals, "class=necromancer agree") != 2 || count(d.Totals, "class=paladin agree") != 0 {
		t.Errorf("per class totals: %+v", d.Totals)
	}

	// Only agreeing lines, for each of the two classes: clean. One disagreeing line more must not stay clean.
	for _, class := range []string{"paladin", "necromancer"} {
		log, b := logTo()
		log.Info(plannerLine(class, "Bone Spear", "Bone Spear", "", "", true))
		if d := digest(t, bytes.NewBuffer(b.Bytes()), Planner); d.Verdict != Clean {
			t.Errorf("%s: an agreeing log is clean, got %s", class, d.Verdict)
		}
		log.Info(plannerLine(class, "Teeth", "Bone Spear", "", "", false))
		if d := digest(t, b, Planner); d.Verdict != Review || d.Differ != 1 {
			t.Errorf("%s: one differing line must ask for a review, got %+v", class, d)
		}
	}
}

func TestPlannerRepeatsAndDrift(t *testing.T) {
	log, b := logTo()
	line := plannerLine("paladin", "Zeal", "Zeal", "Might", "Might", true)
	log.Info(line)
	log.Info("last line repeated", "line", line, "times", 41) // internal/logx.Dedup
	d := digest(t, b, Planner)
	if d.Lines != 42 || d.Verdict != Clean {
		t.Errorf("folded repeats count: %+v", d)
	}

	log, b = logTo()
	log.Info(line)
	log.Info("planner shadow: clvl=9 class=paladin agree=true") // a format this reader does not know
	d = digest(t, b, Planner)
	if d.Unparsed != 1 || d.Verdict != Review || !strings.Contains(d.Why, "did not parse") {
		t.Errorf("an unknown line shape must not pass as clean: %+v", d)
	}
}

func TestAdmission(t *testing.T) {
	log, b := logTo()
	if d := digest(t, bytes.NewBufferString(`{"time":"t","msg":"tick"}`+"\n"), Admission); d.Verdict != NoLines {
		t.Errorf("no line is no-lines, never clean: %+v", d)
	}
	// the attributes of internal/bot/admission.go admitNote
	log.Info("admission shadow", "differ", "new-only", "area", 5, "purpose", "level", "old", "", "new", "kit under the floor", "row", "kit-floor", "need", "8 potions")
	log.Info("admission shadow", "differ", "new-only", "area", 6, "purpose", "level", "old", "", "new", "kit under the floor", "row", "kit-floor", "need", "8 potions")
	log.Info("admission shadow", "differ", "old-only", "area", 3, "purpose", "quest-walk", "old", "too poor", "new", "", "row", "", "need", "")
	d := digest(t, b, Admission)
	if d.Verdict != Review || d.Lines != 3 || d.Differ != 3 {
		t.Fatalf("admission digest: %+v", d)
	}
	if !strings.HasPrefix(d.Why, "new-only 2, old-only 1, words 0") {
		t.Errorf("why %q", d.Why)
	}
	if got := count(d.Rows, `new-only purpose=level row=kit-floor old="" new="kit under the floor"`); got != 2 {
		t.Errorf("two areas of one case are one row: %+v", d.Rows)
	}

	log, b = logTo()
	log.Info("admission shadow", "differ", "words", "area", 5) // keys missing
	if d := digest(t, b, Admission); d.Unparsed != 1 || d.Verdict != Review {
		t.Errorf("a short admission line is unparsed: %+v", d)
	}
}

func TestUpkeepOnceAndAgain(t *testing.T) {
	log, b := logTo()
	// internal/combat/upkeep.go upkeepShadow through Fighter.info: skills of two classes
	log.Info("combat upkeep shadow", "skill", "Frozen Armor", "what", "due", "old", true, "driver", false)
	log.Info("combat upkeep shadow", "skill", "Battle Orders", "what", "due", "old", false, "driver", true)
	log.Info("combat upkeep shadow", "skill", "Battle Orders", "what", "due", "old", false, "driver", true)
	log.Info("last line repeated", "line", "combat upkeep shadow", "times", 2)
	d := digest(t, b, Upkeep)
	if d.Verdict != Review || d.Lines != 5 || len(d.Rows) != 2 {
		t.Fatalf("upkeep digest: %+v", d)
	}
	if d.Rows[0].Key != "skill=Battle Orders what=due old=false driver=true" || d.Rows[0].Count != 4 {
		t.Errorf("the repeat belongs to the line it follows: %+v", d.Rows)
	}
	if !strings.Contains(d.Why, "2 case(s), 1 seen once and 1 seen again") {
		t.Errorf("why %q", d.Why)
	}
	if d := digest(t, bytes.NewBufferString(""), Upkeep); d.Verdict != NoLines {
		t.Errorf("empty log: %+v", d)
	}
}

func TestLootGroundWorse(t *testing.T) {
	if lootGround != loot.ShadowGround || lootPickup != lootrules.Pickup {
		t.Fatal("the ground and pickup words moved in internal/loot or internal/lootrules")
	}
	log, b := logTo()
	log.Info("loot shadow: ground Flawless Skull today=ignore rules=pickup rule=gems reason=a gem worth keeping")
	log.Info("loot shadow: sell Short Sword today=sell rules=keep rule=starter-kit reason=start kit")
	d := digest(t, b, Loot)
	if d.Verdict != Review || d.Lines != 2 || d.Differ != 2 {
		t.Fatalf("extra pickups and a sell difference ask for a review, not worse: %+v", d)
	}
	if d.Rows[0].Example != "Flawless Skull" && d.Rows[1].Example != "Flawless Skull" {
		t.Errorf("an item name with a space is kept whole: %+v", d.Rows)
	}
	// the same log plus one pickup today's code makes and the rules would not: stage 1 is worse
	log.Info("loot shadow: ground Rare Ring today=pickup rules=ignore rule=default reason=no rule")
	d = digest(t, b, Loot)
	if d.Verdict != Worse || count(d.Totals, lootLost) != 1 {
		t.Fatalf("a lost ground pickup is worse: %+v", d)
	}
	// a sell-scope line with the same words is not a ground loss
	log, b = logTo()
	log.Info("loot shadow: tidy Rare Ring today=pickup rules=ignore rule=default reason=no rule")
	if d := digest(t, b, Loot); d.Verdict != Review {
		t.Errorf("only the ground scope decides worse: %+v", d)
	}
	log, b = logTo()
	log.Info("loot shadow: ground Rare Ring") // truncated
	if d := digest(t, b, Loot); d.Unparsed != 1 || d.Verdict != Review {
		t.Errorf("truncated loot line: %+v", d)
	}
}

func TestReportSpanAndFailing(t *testing.T) {
	a := bytes.NewBufferString(`{"time":"2026-11-27T01:00:00Z","msg":"tick"}` + "\nnot json at all\nshadow of a doubt\n")
	b := bytes.NewBufferString(`{"time":"2026-11-27T07:00:00Z","msg":"tick"}`) // no trailing newline
	rep, err := Read(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Lines != 4 || rep.NotJSON != 2 || rep.First != "2026-11-27T01:00:00Z" || rep.Last != "2026-11-27T07:00:00Z" {
		t.Errorf("report head: %+v", rep)
	}
	if len(rep.Families) != 4 || rep.Failing() {
		t.Errorf("a log without shadow lines has four no-lines families and does not fail: %+v", rep.Families)
	}
	log, buf := logTo()
	log.Info("combat upkeep shadow", "skill", "Shout", "what", "due", "old", true, "driver", false)
	if rep, _ := Read(buf); !rep.Failing() {
		t.Error("a review verdict fails the report")
	}
}

func TestFields(t *testing.T) {
	f, ok := fields("a=1 b=two words c=", "a", "b", "c")
	if !ok || f["a"] != "1" || f["b"] != "two words" || f["c"] != "" {
		t.Errorf("fields: %v %v", f, ok)
	}
	if _, ok := fields("b=1 a=2", "a", "b"); ok {
		t.Error("keys out of order must not parse")
	}
}
