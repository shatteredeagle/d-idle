package watch

import (
	"fmt"
	"regexp"
	"strconv"

	"d2lod-bot/internal/decide"
)

// cardTexts is the plain-language card of every decision point: a headline that says what happened with the real values
// of the event, what each answer does and what the hero does by itself. Data, not class code (V: TestEveryPointHasACard
// walks the shipped catalogue, so a new point or answer fails the test until it gets its text here).
//
// Placeholders in a headline are {fact} (the value as given), {fact#one|many} (a count with its noun: "no enemies", "1
// enemy", "3 enemies") and {fact:code} where the value is a short code mapped through factWords. A headline whose facts
// are not all known is not shown: the card then says the Generic line, which names no value, so a call site that does not
// report a fact degrades to a general sentence instead of showing a hole.
type cardText struct {
	Headline string
	Generic  string
	Does     map[string]string // answer id -> what the hero does, as a verb phrase ("carry on and enter it anyway")
}

var cardTexts = map[string]cardText{
	"t0.retreat.drink": {
		Headline: "Life fell to {life}% in {area}, with {potions#healing potion|healing potions} in the belt and {hostiles#enemy|enemies} close.",
		Generic:  "Life is low in a fight and the hero has to choose how to survive.",
		Does: map[string]string{"drink": "drink a potion now (a rejuvenation potion first when life is critical)", "run": "run away from the fight",
			"portal": "open a town portal and leave the fight", "leave": "quit the game to be safe", "fight": "keep fighting (its normal potion rules still apply)"},
	},
	"t0.retreat.lethal": {
		Headline: "A {pack} that nearly killed the hero before is close in {area}.",
		Generic:  "A kind of pack that nearly killed the hero before is close.",
		Does:     map[string]string{"avoid": "stay away from the pack", "fight": "fight it anyway"},
	},
	"t0.area.toohard": {
		Headline: "Heading to {dest}: the bot wants hero level {floor} there and the hero is level {herolevel}.",
		Generic:  "The next area may be too hard for the hero.",
		Does: map[string]string{"go": "carry on and enter it anyway", "skip": "give up the area and pick the next one",
			"lower": "skip it and drop to an easier route step"},
	},
	"t0.retreat.repeat": {
		Headline: "The hero has already retreated {retreats#time|times} in a row in {area}.",
		Generic:  "The hero keeps retreating in this area.",
		Does:     map[string]string{"stay": "stay and keep trying", "leave": "give up the area and leave"},
	},
	"t0.unique.fight": {
		Headline: "A {packkind} led by {monster} is {distance#square|squares} away in {area}, and life is {life}%.",
		Generic:  "A champion or unique pack is in reach and the hero is hurt.",
		Does:     map[string]string{"fight": "attack it, drinking potions early", "avoid": "keep away from it"},
	},
	"t0.leave.safe": {
		Headline: "A fast or ranged pack is on the hero in {area} at life {life}%.",
		Generic:  "A fast or ranged pack is attacking.",
		Does:     map[string]string{"portal": "open a town portal and leave", "run": "run away", "stay": "stay and fight"},
	},
	"t0.shrine": {
		Headline: "A shrine is within reach in {area} at life {life}%.",
		Generic:  "A shrine is within reach.",
		Does:     map[string]string{"take": "walk to the shrine and take it", "avoid": "walk past it"},
	},
	"t1.restock": {
		Headline: "In town after a trip with {tripgold} gold to spend; the belt holds {have} of {want} healing potions.",
		Generic:  "In town after a trip, with the belt not full.",
		Does:     map[string]string{"yes": "spend the gold on belt potions", "no": "keep the gold and skip the potions"},
	},
	"t1.return": {
		Headline: "The town chores are done: the belt holds {have} of {want} healing potions and supplies are {kit:kit}.",
		Generic:  "The town chores are done and the hero can head back out.",
		Does:     map[string]string{"yes": "leave town for the field", "no": "stay in town and shop once more first"},
	},
	"t1.keep.kind": {
		Headline: "The hero found {item}.",
		Generic:  "The hero found an item of a kind it has to decide on.",
		Does:     map[string]string{"keep": "keep it", "sell": "sell it"},
	},
	"t1.town.trigger": {
		Headline: "The pack is {packused}% full and the belt holds {have} of {want} healing potions in {area}.",
		Generic:  "The pack or the belt says it may be time for a town trip.",
		Does:     map[string]string{"go": "head to town now", "continue": "keep going in the field"},
	},
	"t1.tp.lost": {
		Headline: "No town portal scroll is left, and the hero has {gold} gold.",
		Generic:  "The hero has no town portal scroll.",
		Does:     map[string]string{"buy": "buy a scroll in town", "walk": "walk back instead"},
	},
	"t1.stash.full": {
		Headline: "The stash is full and the hero holds {gold} gold.",
		Generic:  "The stash is full.",
		Does:     map[string]string{"sell": "sell the surplus", "drop": "drop the surplus", "pause": "pause until you decide"},
	},
	"t1.repair": {
		Headline: "In town with gear worn down to {durability}% and {gold} gold on hand.",
		Generic:  "In town with worn gear.",
		Does:     map[string]string{"repair": "pay the smith to repair", "skip": "leave the gear as it is"},
	},
	"t1.identify": {
		Headline: "The pack holds {item} that is not identified.",
		Generic:  "The pack holds an item that is not identified.",
		Does:     map[string]string{"scroll": "read a scroll of identify", "cain": "ask Deckard Cain", "leave": "leave it unidentified"},
	},
	"t1.potions.buy": {
		Headline: "Buying belt potions with {gold} gold in hand.",
		Generic:  "The hero is about to buy belt potions.",
		Does:     map[string]string{"best": "buy the best potions it can afford", "cheapest": "buy the cheapest that are enough"},
	},
	"t1.gamble": {
		Headline: "The hero holds {gold} gold and could gamble some of it.",
		Generic:  "The hero could gamble some gold.",
		Does:     map[string]string{"gamble": "spend gold on gambling", "save": "save the gold"},
	},
	"t2.repeat": {
		Headline: "The {run:run} run is finished.",
		Generic:  "A run is finished.",
		Does:     map[string]string{"again": "run it again", "stop": "stop and pick something else"},
	},
	"t2.failstreak": {
		Headline: "The hero died {deaths#time|times} during the {run:run} run.",
		Generic:  "The hero keeps dying on this run.",
		Does:     map[string]string{"stop": "stop the run", "continue": "carry on with it"},
	},
	"t2.slowrun": {
		Headline: "The {run:run} run is slower than expected.",
		Generic:  "The run is slower than expected.",
		Does:     map[string]string{"continue": "carry on with it", "switch": "switch to another run"},
	},
	"t2.bossunsafe": {
		Headline: "{boss} is next, but at hero level {herolevel} the gear looks too weak.",
		Generic:  "The next boss looks too strong for the hero's gear.",
		Does:     map[string]string{"skip": "skip the boss for now", "go": "fight the boss anyway"},
	},
	"t2.reset.game": {
		Headline: "The hero is in {area} and could start the game over for a fresh one.",
		Generic:  "The hero could restart the game for a fresh start.",
		Does:     map[string]string{"restart": "restart the game", "stay": "stay and clear what is here"},
	},
	"t2.mercenary": {
		Headline: "The mercenary is {merc:merc} and the hero holds {gold} gold in {area}.",
		Generic:  "The hero has no living mercenary and some gold.",
		Does:     map[string]string{"hire": "hire or revive the mercenary now", "wait": "wait until there is more gold"},
	},
}

// factWords maps the short codes some facts carry to words for a headline.
var factWords = map[string]map[string]string{
	"kit":  {"OK": "in order", "THIN": "thin (potions or a portal are missing)", "NONE": "missing (no potions and no portal)"},
	"merc": {"alive": "alive", "dead": "dead", "none": "not hired yet"},
	"run":  {"farm": "farming", "quest": "quest", "level": "levelling"},
}

var placeholder = regexp.MustCompile(`\{([a-z]+)(?:#([^|}]+)\|([^}]+)|:([a-z]+))?\}`)

// Placeholders lists the fact names a headline uses.
func placeholders(s string) []string {
	var out []string
	for _, m := range placeholder.FindAllStringSubmatch(s, -1) {
		out = append(out, m[1])
	}
	return out
}

// renderHeadline fills the headline from the situation. ok is false when a fact is missing or empty; the caller then
// shows the Generic line.
func renderHeadline(tmpl string, sit decide.Situation) (string, bool) {
	ok := true
	out := placeholder.ReplaceAllStringFunc(tmpl, func(m string) string {
		g := placeholder.FindStringSubmatch(m)
		v, have := sit[g[1]]
		if !have || v == "" {
			ok = false
			return ""
		}
		switch {
		case g[2] != "":
			n, err := strconv.Atoi(v)
			if err != nil {
				ok = false
				return ""
			}
			switch n {
			case 0:
				return "no " + g[3]
			case 1:
				return "1 " + g[2]
			}
			return fmt.Sprintf("%d %s", n, g[3])
		case g[4] != "":
			if w, mapped := factWords[g[4]][v]; mapped {
				return w
			}
		}
		return v
	})
	if !ok {
		return "", false
	}
	return out, true
}

// Card is the plain-language face of one question: what happened, what the answers do, what happens on no answer.
type Card struct {
	Headline  string            // what happened, with the real values (or the Generic line)
	Specific  bool              // the headline carries the event's values (false: the generic line)
	IfNothing string            // "If you do nothing: the hero will ..." and the fade time
	WhyNow    string            // urgency in words; "" when calm
	Rule      string            // the learning key in words, a footnote
	Does      map[string]string // answer id -> verb phrase
}

// BuildCard writes the card of a thought. def is the answer id the hero takes meanwhile ("" = the catalogue default).
func (h *Hub) BuildCard(info decide.ThoughtInfo, def string) Card {
	c := Card{Rule: "Rule applies when: " + PlainContext(info.Key) + ".", WhyNow: whyNow(info.Key)}
	t, ok := cardTexts[info.Point]
	if !ok { // a point added to the catalogue without a card: the catalogue's own question
		c.Headline = info.Question
		return c
	}
	c.Does = t.Does
	if s, ok := renderHeadline(t.Headline, info.Sit); ok {
		c.Headline, c.Specific = s, true
	} else {
		c.Headline = t.Generic
	}
	if def == "" {
		def = h.defaultFor(info.Key)
	}
	do := t.Does[def]
	if do == "" {
		do = "decide by its usual rules"
	}
	c.IfNothing = fmt.Sprintf("If you do nothing: the hero will %s. Answer within %d s or the hero decides.", do, h.cfg.Cat.Config.FadeSec)
	return c
}

// whyNow says in words how urgent a situation is, from the urgency band or the low life bands of the key; "" when calm.
func whyNow(key string) string {
	for _, f := range Fields(key) {
		switch f.Name {
		case "urgency":
			switch f.Value {
			case "wary":
				return "Danger is rising, so the answer matters soon."
			case "urgent", "critical":
				return "Danger is " + f.Value + ", so the answer matters right now."
			}
			return ""
		case "life", "drinklife", "lifeengage":
			if f.Value == "lt25" || f.Value == "lt18" || f.Value == "18-25" || f.Value == "25-35" {
				return "Life is low, so the answer matters right now."
			}
		}
	}
	return ""
}

// situationLine is the "When:" text of a journal line: the headline of the event when the call site gave its facts.
func situationLine(info decide.ThoughtInfo) string {
	t, ok := cardTexts[info.Point]
	if !ok {
		return ""
	}
	s, _ := renderHeadline(t.Headline, info.Sit)
	return s
}
