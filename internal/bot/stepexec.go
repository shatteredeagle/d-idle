package bot

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	questspecdata "d2lod-bot/data/questspec"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/questspec"
	"d2lod-bot/internal/talk"
	"d2lod-bot/internal/task"
)

// The thin adapter of the generic step executor (internal/questspec/exec.go, knob quests.stepExec, card stepexec1).
// The engine and the step row are data and pure; this file only maps the four verbs onto the bot's capabilities and
// the typed outcome back onto the task layer's errors. First chunk on it: the Cain scroll (take_scroll).

// execObjectNames gives the object adapter's name for a class id (the Objects capability still clicks by name).
var execObjectNames = map[int]string{
	quest.ObjTreeInifuss: TreeObject, quest.ObjOrifice: OrificeObject,
	// the Act 2 quest chests, the altar and the journal: the names task/act2.go's nodes click
	quest.ObjCubeChest: "Horadric Cube chest", quest.ObjScrollChest: "Horadric Scroll chest",
	quest.ObjStaffChest: "Staff of Kings chest", quest.ObjTaintedSun: "taintedsunaltar", quest.ObjTomeJournal: "Summoner's journal",
}

// OrificeObject is the object adapter's name of the Horadric orifice; the adapter finds the true tomb and places the Staff.
const OrificeObject = "tal-orifice"

var (
	execOnce      sync.Once
	execSteps     map[string]questspec.Step
	execTalks     map[string]string // "<npc class>/<pick>" -> "<chain>/<step>" of the talk steps that carry an exec row
	execInteracts map[string]string // object adapter name -> "<chain>/<step>" of the interact steps
	execCubes     map[string]string // sorted input names joined by "+" -> "<chain>/<step>" of the transmute steps
	execKills     map[string]string // normalized monster name -> "<chain>/<step>" of the kill steps (no exec row needed)
	execErr       error
)

// execStep returns the step row "<chain>/<step>" of the embedded act files.
func execStep(chain, step string) (questspec.Step, error) {
	execOnce.Do(func() {
		execSteps, execTalks, execInteracts, execCubes, execKills = map[string]questspec.Step{}, map[string]string{}, map[string]string{}, map[string]string{}, map[string]string{}
		names, err := fs.Glob(questspecdata.FS, "act*.json")
		if err == nil && len(names) == 0 {
			err = errors.New("no act files embedded")
		}
		for _, n := range names {
			if err != nil {
				break
			}
			var raw []byte
			if raw, err = questspecdata.FS.ReadFile(n); err != nil {
				break
			}
			var cs []*questspec.Chain
			if cs, err = questspec.LoadFile(raw); err != nil {
				err = fmt.Errorf("%s: %w", n, err)
				break
			}
			for _, c := range cs {
				for _, s := range c.Steps {
					execIndex(c.ID+"/"+s.ID, s)
				}
			}
		}
		execErr = err
	})
	if execErr != nil {
		return questspec.Step{}, execErr
	}
	s, ok := execSteps[chain+"/"+step]
	if !ok {
		return questspec.Step{}, fmt.Errorf("stepexec: no step %s/%s in the embedded chains", chain, step)
	}
	return s, nil
}

// execIndex files a step under the keys the task layer's calls carry (NPC and line, object name, Cube inputs).
func execIndex(id string, s questspec.Step) {
	execSteps[id] = s
	if s.Act == "kill" && s.Target.Kind == "monster" {
		execKills[normName(s.Target.Name)] = id
	}
	e := s.Exec
	if e == nil {
		return
	}
	on := s.Target
	if e.On != nil {
		on = *e.On
	}
	switch e.Verb {
	case "talk":
		execTalks[fmt.Sprintf("%d/%s", on.Class, e.Pick)] = id
	case "interact":
		if name, ok := execObjectNames[on.Class]; ok {
			execInteracts[name] = id
		}
	case "use": // a pack item is "clicked" through the object adapter as "item:<name>"
		if name, ok := questItemCodes[on.Code]; ok {
			execInteracts["item:"+name] = id
		}
	case "transmute":
		names := make([]string, len(e.Inputs))
		for i, c := range e.Inputs {
			names[i] = questItemCodes[c]
		}
		execCubes[cubeKey(names)] = id
	}
}

func cubeKey(names []string) string {
	k := append([]string(nil), names...)
	sort.Strings(k)
	return strings.Join(k, "+")
}

// execRow looks a key up in one of the indexes; false when the data has no exec row for it (the old handler runs).
func execRow(index *map[string]string, key string) (chain, step string, ok bool) {
	execStep("", "") // loads the tables once
	if execErr != nil {
		return "", "", false
	}
	id, found := (*index)[key]
	if !found {
		return "", "", false
	}
	chain, step, _ = strings.Cut(id, "/")
	return chain, step, true
}

// execTalkStep finds the talk step row (chain, step) the data declares for this NPC and dialogue option; false when no
// row carries an exec block for it (the old handler runs).
func execTalkStep(npc, pick string) (chain, step string, ok bool) {
	for _, cls := range talk.ByAct[npc] { // one name, a class per act
		if chain, step, ok = execRow(&execTalks, fmt.Sprintf("%d/%s", cls, pick)); ok {
			return
		}
	}
	return "", "", false
}

// execNPCName is the dialog name of an NPC class id (the Dialog capability talks by name): the inverse of talk.ByAct.
func execNPCName(class int) (string, bool) {
	for name, acts := range talk.ByAct {
		for _, c := range acts {
			if c == class {
				return name, true
			}
		}
	}
	return "", false
}

// execBackend implements questspec.Backend over the bot's capabilities.
type execBackend struct {
	a taskActs
}

// Approach: the object and dialog adapters walk to their target themselves, with one gap: the dialog adapter walks only to
// an NPC the snapshot lists. An NPC of the field that is out of sight (V lab night 2026-11-28, drill.act2.duriel.stepexec,
// 3 of 3 runs: Tyrael in Duriel's Lair right after the kill, "npc: walk failed seen=false" on every try, the row's tries
// and the chunk's three were spent in 0.2 s and the chunk parked) is looked for first: the level is walked until the
// snapshot lists him. On d2d the walk does not reach his chamber (below). A town NPC is the town adapter's job.
func (b *execBackend) Approach(ctx context.Context, t questspec.Target, _ int) error {
	r := b.a.r
	if t.Kind != "npc" || t.Class == 0 || r.d.Mover == nil {
		return nil
	}
	seen := func() bool {
		s, err := r.d.Snapshot()
		if err != nil || s == nil || s.Player == nil || s.Player.InTown() {
			return true // nothing to look for from here
		}
		for i := range s.Monsters {
			if s.Monsters[i].Class == t.Class {
				return true
			}
		}
		return false
	}
	if seen() {
		return nil
	}
	r.e.Log.Info("step executor: npc out of sight, walking the level for him", "class", t.Class)
	if err := r.d.Mover.Explore(ctx, seen); err != nil && ctx.Err() != nil {
		return b.a.tag(err)
	}
	if seen() {
		return nil
	}
	// Not in the part of the level the hero can walk. V offline 2026-11-28 (TASK-1126, the fork's drlg-dump on the game
	// data, level 73): Tyrael's preset is at subtile 78,88, the arena's only way to him is the 1x5 opening at 110,207 that
	// "tyrael's door" (object 153: collision in modes 0 and 1, not selectable, no operate function) closes until Duriel
	// dies; the d2d fork does not open it, so the walk ends in the arena (12 coverage points in every lab run). The talk
	// cannot work from here: say so in words the board keeps apart from a hero's failure (questboard sameTryMarks).
	at, area, objs := [2]int{}, 0, map[int]int{}
	if s := r.latest(); s != nil && s.Player != nil {
		at, area = [2]int{s.Player.X, s.Player.Y}, s.Player.Area
		for i := range s.Objects {
			objs[s.Objects[i].Class]++
		}
	}
	r.e.Log.Warn("step executor: npc not in sight after the walk", "class", t.Class, "area", area, "at", at, "objectsByClass", fmt.Sprint(objs))
	return task.Wrap(task.KindNotFound, fmt.Errorf("%s: npc class %d is not in the part of area %d the hero can walk", npcOutOfReach, t.Class, area))
}

// npcOutOfReach marks the failure of a talk row whose field NPC is not in the walkable part of the level.
const npcOutOfReach = "npc_out_of_reach"

func (b *execBackend) Act(ctx context.Context, verb string, t questspec.Target, args questspec.ActArgs) error {
	a := b.a
	td := a.r.d.Task
	switch verb {
	case "interact":
		name, ok := execObjectNames[t.Class]
		if !ok {
			return &questspec.FatalError{Err: fmt.Errorf("no object name for class %d", t.Class)}
		}
		if td == nil || td.Objects == nil {
			return &questspec.FatalError{Err: errors.New("no object interaction capability")}
		}
		return a.tag(td.Objects.Interact(ctx, name))
	case "pickup":
		if a.r.d.Loot == nil {
			return &questspec.FatalError{Err: errors.New("no looter")}
		}
		name, ok := questItemCodes[t.Code]
		if !ok {
			return &questspec.FatalError{Err: fmt.Errorf("no item name for code %q", t.Code)}
		}
		rep, err := a.r.d.Loot.PickUp(ctx, PickOpts{Radius: 30, Names: []string{name}})
		switch {
		case err != nil && task.Classify(a.tag(err)) != task.KindNotFound:
			return a.tag(err)
		case err != nil || rep.Picked == 0:
			return questspec.ErrNothing
		}
		return nil
	case "talk":
		name, ok := execNPCName(t.Class)
		if !ok {
			return &questspec.FatalError{Err: fmt.Errorf("no npc name for class %d", t.Class)}
		}
		if td == nil || td.Dialog == nil {
			return &questspec.FatalError{Err: errors.New("no dialog capability")}
		}
		return a.tag(td.Dialog.Talk(ctx, name, args.Pick))
	case "use":
		name, ok := questItemCodes[t.Code]
		if !ok {
			return &questspec.FatalError{Err: fmt.Errorf("no item name for code %q", t.Code)}
		}
		// the adapter directly: UseItem goes through Interact, which routes back to this row
		if td == nil || td.Objects == nil {
			return &questspec.FatalError{Err: errors.New("no object interaction capability")}
		}
		return a.tag(td.Objects.Interact(ctx, "item:"+name))
	case "transmute":
		if td == nil || td.Cube == nil {
			return &questspec.FatalError{Err: errors.New("no Horadric Cube capability")}
		}
		names := make([]string, len(args.Inputs))
		for i, c := range args.Inputs {
			if names[i] = questItemCodes[c]; names[i] == "" {
				return &questspec.FatalError{Err: fmt.Errorf("no item name for code %q", c)}
			}
		}
		return a.tag(td.Cube(ctx, names))
	}
	return &questspec.FatalError{Err: fmt.Errorf("verb %q is not an executor verb", verb)}
}

// Holds reads the post from a fresh snapshot: has_item(code) and lacks_item(code) are the pack, quest_state(q,stage) the
// quest reader's word for the hero's difficulty (the reader only, never the task layer's proxies, so the executor's own
// acts cannot satisfy it), quest_bits(q,mask) every bit of mask set in that word, killed(monster) a death of it seen
// in this game (the observer's kill or the dead unit in the snapshot). Any other fact is not known here and reads false.
func (b *execBackend) Holds(c questspec.Clause) bool {
	r := b.a.r
	s, err := r.d.Snapshot()
	if err != nil || s == nil || s.State != d2.StateInGame {
		s = r.latest()
	}
	if s == nil {
		return false
	}
	held := map[string]int{}
	r.scanQuestItems(s, held)
	for _, p := range c {
		switch {
		case p.Name == "has_item" && len(p.Args) == 1:
			if held[questItemCodes[p.Args[0]]] > 0 {
				return true
			}
		case p.Name == "lacks_item" && len(p.Args) == 1:
			if name, ok := questItemCodes[p.Args[0]]; ok && held[name] == 0 && s.Inventory != nil {
				return true
			}
		case p.Name == "quest_state" && len(p.Args) == 2:
			if b.questAt(s, p.Args[0], p.Args[1]) {
				return true
			}
		case p.Name == "quest_bits" && len(p.Args) == 2:
			if b.questBits(s, p.Args[0], p.Args[1]) {
				return true
			}
		case p.Name == "killed" && len(p.Args) == 1:
			if r.killedBoss(p.Args[0]) || deadMonster(r, s, p.Args[0]) {
				return true
			}
		}
	}
	return false
}

var execStages = map[string]task.Stage{"started": task.StageStarted, "cleared": task.StageCleared, "done": task.StageDone}

// questAt: the quest reader says q is at stage want or later.
func (b *execBackend) questAt(s *d2.Snapshot, q, want string) bool {
	td := b.a.r.d.Task
	st, ok := execStages[want]
	if !ok || td == nil || td.Quests == nil || s.Player == nil {
		return false
	}
	v, ok := td.Quests.Read(int(s.Player.Difficulty))
	return ok && v.Quests[q].Stage >= st
}

// questBits: the reader's raw word of q has every bit of mask (hex, "0x0020") set. A malformed mask reads false.
func (b *execBackend) questBits(s *d2.Snapshot, q, mask string) bool {
	m, err := strconv.ParseUint(strings.TrimPrefix(mask, "0x"), 16, 16)
	td := b.a.r.d.Task
	if err != nil || m == 0 || td == nil || td.Quests == nil || s.Player == nil {
		return false
	}
	v, ok := td.Quests.Read(int(s.Player.Difficulty))
	st, known := v.Quests[q]
	return ok && known && st.Raw&uint16(m) == uint16(m)
}

// deadMonster: the snapshot holds a dead unit of that monster (by catalog or unit name, normalized).
func deadMonster(r *Runner, s *d2.Snapshot, name string) bool {
	want := normName(strings.TrimPrefix(name, "The "))
	for i := range s.Monsters {
		u := &s.Monsters[i]
		if u.Alive() {
			continue
		}
		n := ""
		if u.Monster != nil {
			n = u.Monster.Name
		}
		if n == "" && r.d.Catalog != nil {
			if m, ok := r.d.Catalog.Monster(u.Class); ok {
				n = m.Name
			}
		}
		if n = normName(n); n != "" && (n == want || n == "the"+want) {
			return true
		}
	}
	return false
}

// bossDead reads a boss kill step's post from game state: the data row's post (killed, the quest word bit of the kill,
// the reader's stage), or killed(name) alone when no row names the boss. A boss the general fight killed before the
// boss run is then done instead of "not found in the arena".
func (a taskActs) bossDead(name string) bool {
	b := &execBackend{a: a}
	if chain, step, ok := execRow(&execKills, normName(strings.TrimPrefix(name, "The "))); ok {
		if s, err := execStep(chain, step); err == nil {
			for _, c := range s.PostClauses() {
				if !b.Holds(c) {
					return false
				}
			}
			return len(s.Post) > 0
		}
	}
	return b.Holds(questspec.Clause{{Name: "killed", Args: []string{name}}})
}

func (b *execBackend) Sleep(d time.Duration) { b.a.r.e.Clock.Sleep(d) }

// runExecStep runs one data step through the engine and maps the outcome to the task layer's errors: a refusal reads
// "not ready" (the typed refusal path of outcome.go classifies it), a fault is KindNotFound like the old handler.
func (a taskActs) runExecStep(ctx context.Context, chain, step string) error {
	s, err := execStep(chain, step)
	if err != nil {
		return task.Wrap(task.KindFatal, err)
	}
	o := questspec.RunStep(ctx, s, &execBackend{a: a})
	a.r.e.Log.Info("step executor", "step", chain+"/"+step, "outcome", o.Kind.String(), "acts", o.Attempts)
	switch o.Kind {
	case questspec.Done:
		return nil
	case questspec.Refused:
		return task.Errorf(task.KindDanger, "not ready (%s): %s", o.Need, o.Why)
	case questspec.Interrupted:
		return a.tag(o.Err)
	case questspec.Progress:
		return task.Errorf(task.KindNotFound, "step %s/%s in progress", chain, step)
	}
	return a.tag(task.Errorf(task.KindNotFound, "%v", o.Err))
}

// interactViaExecutor runs the interact step the data declares for this object (the Cain tree, the Horadric orifice);
// ok false: no row, the old handler runs. The pseudo item of the click is the task layer's bookkeeping, set after Done.
func (a taskActs) interactViaExecutor(ctx context.Context, object string) (ok bool, err error) {
	chain, step, ok := execRow(&execInteracts, object)
	if !ok {
		return false, nil
	}
	if err := a.runExecStep(ctx, chain, step); err != nil {
		return true, err
	}
	a.setPseudo(interactPseudo[object])
	return true, nil
}
