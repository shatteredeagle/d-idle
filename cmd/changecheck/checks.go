package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// checkOverlap: no two active changes may own overlapping paths or claim the same exclusive resource.
func checkOverlap(cards []*Card) (fs []Finding) {
	var act []*Card
	for _, c := range cards {
		if c.Active() {
			act = append(act, c)
		}
	}
	for i, a := range act {
		for _, b := range act[i+1:] {
			for _, ga := range a.Owns() {
				for _, gb := range b.Owns() {
					if globsOverlap(ga, gb) {
						fs = append(fs, Finding{"overlap", fmt.Sprintf("%s owns %q and %s owns %q overlap", a.ID, ga, b.ID, gb)})
					}
				}
			}
			for _, ea := range a.Exclusive() {
				for _, eb := range b.Exclusive() {
					if ea == eb {
						fs = append(fs, Finding{"overlap", fmt.Sprintf("%s and %s both claim exclusive resource %q", a.ID, b.ID, ea)})
					}
				}
			}
		}
	}
	return fs
}

// Owner is one docs/OWNERS line: path glob, owner, change id or "-".
type Owner struct{ Glob, Owner, Change string }

func loadOwners(root string) (list []Owner, fs []Finding) {
	b, err := readFile(filepath.Join(root, "docs", "OWNERS"))
	if err != nil {
		return nil, []Finding{{"owners", "docs/OWNERS missing"}}
	}
	for n, l := range strings.Split(b, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		f := strings.Fields(l)
		if len(f) < 3 {
			fs = append(fs, Finding{"owners", fmt.Sprintf("docs/OWNERS:%d needs glob, owner, change (or -)", n+1)})
			continue
		}
		list = append(list, Owner{f[0], f[1], f[2]})
	}
	return list, fs
}

// checkOwners keeps docs/OWNERS and the active cards in step.
func checkOwners(root string, cards []*Card) []Finding {
	owners, fs := loadOwners(root)
	if owners == nil && len(fs) > 0 {
		return fs
	}
	byID := map[string]*Card{}
	for _, c := range cards {
		if c.Active() {
			byID[c.ID] = c
		}
	}
	listed := map[string]map[string]bool{}
	for _, o := range owners {
		if o.Change == "-" {
			continue
		}
		c := byID[o.Change]
		if c == nil {
			fs = append(fs, Finding{"owners", fmt.Sprintf("OWNERS lists %s for change %q which is not active", o.Glob, o.Change)})
			continue
		}
		if listed[c.ID] == nil {
			listed[c.ID] = map[string]bool{}
		}
		listed[c.ID][o.Glob] = true
		if !contains(c.Owns(), o.Glob) {
			fs = append(fs, Finding{"owners", fmt.Sprintf("OWNERS lists %s for %s but its card does not own it", o.Glob, c.ID)})
		}
		if o.Owner != c.Agent() {
			fs = append(fs, Finding{"owners", fmt.Sprintf("OWNERS owner of %s is %s, card says %s", o.Glob, o.Owner, c.Agent())})
		}
	}
	for _, c := range byID {
		for _, g := range c.Owns() {
			if !listed[c.ID][g] {
				fs = append(fs, Finding{"owners", fmt.Sprintf("card %s owns %q but docs/OWNERS has no such line", c.ID, g)})
			}
		}
	}
	for i, a := range owners {
		for _, b := range owners[i+1:] {
			if a.Change != "-" && b.Change != "-" && a.Owner != b.Owner && globsOverlap(a.Glob, b.Glob) {
				fs = append(fs, Finding{"owners", fmt.Sprintf("OWNERS %s (%s) overlaps %s (%s)", a.Glob, a.Owner, b.Glob, b.Owner)})
			}
		}
	}
	return fs
}

// Merge is one merge commit on the first-parent history.
type Merge struct{ Hash, Date, Subject, Body string }

var (
	mergeBranchRe = regexp.MustCompile(`Merge (?:branch )?'?agent/([a-z0-9-]+)`)
	trailerRe     = regexp.MustCompile(`(?m)^Change:\s*([a-z0-9-]+)`)
)

// changeID extracts the change a merge names, from its subject or a Change: trailer.
func changeID(m Merge) string {
	if t := trailerRe.FindStringSubmatch(m.Body); t != nil {
		return t[1]
	}
	if t := mergeBranchRe.FindStringSubmatch(m.Subject); t != nil {
		return t[1]
	}
	return ""
}

// gitMerges lists merge commits after rev (or the last n when rev is empty), newest first.
var gitMerges = func(root, rev string, n int) ([]Merge, error) {
	args := []string{"-C", root, "log", "--merges", "--date=format:%m-%d %H:%M", "--format=%h%x1f%ad%x1f%s%x1f%b%x1e"}
	if rev != "" {
		args = append(args, rev+"..HEAD")
	} else {
		args = append(args, fmt.Sprintf("-n%d", n))
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, err
	}
	var ms []Merge
	for _, rec := range strings.Split(string(out), "\x1e") {
		f := strings.SplitN(strings.TrimLeft(rec, "\n"), "\x1f", 4)
		if len(f) == 4 {
			ms = append(ms, Merge{f[0], f[1], f[2], f[3]})
		}
	}
	return ms, nil
}

// infos collects INFO lines (not warnings) the checks want printed; main prints them.
var infos []string

const noHistoryMsg = "no history before the baseline: merge/diff checks start at the next commit"

func noteNoHistory() {
	if !contains(infos, noHistoryMsg) {
		infos = append(infos, noHistoryMsg)
	}
}

// noHistory reports whether the repo has no usable history to check: shallow, or a single root commit.
func noHistory(root string) bool {
	if out, err := exec.Command("git", "-C", root, "rev-parse", "--is-shallow-repository").Output(); err == nil && strings.TrimSpace(string(out)) == "true" {
		return true
	}
	out, err := exec.Command("git", "-C", root, "rev-list", "--count", "HEAD").Output()
	return err == nil && strings.TrimSpace(string(out)) == "1"
}

func adoptedRev(root string) string {
	b, err := readFile(filepath.Join(root, "openspec", ".adopted"))
	if err != nil {
		return ""
	}
	return strings.Fields(b + " ")[0]
}

// checkMerges: every merge since adoption names a change that exists (active or archived).
func checkMerges(root string, cards []*Card, since string, n int) []Finding {
	strict := loadStrict(root)
	if since == "" {
		since = strict.HistoryBaseline
	}
	if since == "" {
		since = adoptedRev(root)
	}
	if noHistory(root) {
		noteNoHistory()
		return nil
	}
	ms, err := gitMerges(root, since, n)
	if err != nil && since == "" {
		noteNoHistory()
		return nil
	}
	if err != nil {
		return []Finding{{"merge", "git log failed: " + err.Error()}}
	}
	ids := map[string]bool{}
	for _, c := range cards {
		ids[c.ID] = true
	}
	var fs []Finding
	for _, m := range ms {
		if strict.allowed(m.Hash) {
			continue
		}
		low := strings.ToLower(m.Subject)
		if strings.HasPrefix(low, "merge master") || strings.HasPrefix(low, "merge branch 'master'") || strings.HasPrefix(low, "merge branch 'main'") {
			continue
		}
		id := changeID(m)
		switch {
		case id == "":
			fs = append(fs, Finding{"merge", fmt.Sprintf("%s %q names no change (use 'Merge agent/<id>:' or a Change: trailer)", m.Hash, m.Subject)})
		case !ids[id]:
			fs = append(fs, Finding{"merge", fmt.Sprintf("%s names change %q which has no card in openspec/changes", m.Hash, id)})
		}
	}
	return fs
}

// defaultBranch names the integration branch: origin/HEAD when set, else the first of main, master that exists
// locally or as origin/<name>, else "main". The private working repo uses master, the public one main.
func defaultBranch(root string) string {
	if out, err := exec.Command("git", "-C", root, "symbolic-ref", "--short", "refs/remotes/origin/HEAD").Output(); err == nil {
		if b := strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/"); b != "" {
			return b
		}
	}
	for _, b := range []string{"main", "master"} {
		for _, ref := range []string{"refs/heads/" + b, "refs/remotes/origin/" + b} {
			if exec.Command("git", "-C", root, "show-ref", "--verify", "--quiet", ref).Run() == nil {
				return b
			}
		}
	}
	return "main"
}

// checkDiff compares the files a branch changed against the owns: of its card (the merge checklist).
func checkDiff(root string, cards []*Card, branch string) []Finding {
	id := strings.TrimPrefix(branch, "agent/")
	var c *Card
	for _, x := range cards {
		if x.ID == id || x.Branch() == branch {
			c = x
		}
	}
	if c == nil {
		return []Finding{{"diff", "no card for " + branch}}
	}
	out, err := exec.Command("git", "-C", root, "diff", "--name-only", defaultBranch(root)+"..."+branch).Output()
	if err != nil && noHistory(root) {
		noteNoHistory()
		return nil
	}
	if err != nil {
		return []Finding{{"diff", "git diff failed: " + err.Error()}}
	}
	globs := append(append([]string{}, c.Owns()...), "openspec/changes/"+c.ID+"/", "openspec/changes/archive/*-"+c.ID+"/")
	var fs []Finding
	for _, p := range strings.Fields(string(out)) {
		if !matchAny(globs, p) {
			fs = append(fs, Finding{"diff", fmt.Sprintf("%s changed %s outside owns", branch, p)})
		}
	}
	return fs
}

func readFile(p string) (string, error) {
	b, err := os.ReadFile(p)
	return string(b), err
}

func contains(v []string, s string) bool {
	for _, x := range v {
		if x == s {
			return true
		}
	}
	return false
}
