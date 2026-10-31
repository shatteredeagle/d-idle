// Command changecheck checks the change cards under openspec/ against each other and the repo, and
// regenerates the generated block of docs/plan/STATUS.md. Warn-only unless -strict. See openspec/README.md.
// "changecheck json" prints the same data as JSON for the board page (tools/agentstatus/changes.html).
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	args := os.Args[1:]
	sub := ""
	if len(args) > 0 && (args[0] == "new" || args[0] == "status" || args[0] == "json") {
		sub, args = args[0], args[1:]
	}
	fl := flag.NewFlagSet("changecheck", flag.ExitOnError)
	root := fl.String("root", "", "repo root (default: git toplevel of the cwd)")
	strict := fl.Bool("strict", false, "exit 1 when there is any warning")
	since := fl.String("since", "", "check merges after this rev (default: openspec/.adopted)")
	nMerges := fl.Int("merges", 10, "merges to look at when there is no -since or .adopted; also the STATUS list length")
	diff := fl.String("diff", "", "check that this branch only touches its card's owns: globs")
	write := fl.Bool("write", false, "status: rewrite the block in docs/plan/STATUS.md instead of printing it")
	cli := fl.Bool("cli", false, "also run tools/openspec.sh validate --all --strict when node is on PATH (needs network the first time)")
	today := fl.String("today", "", "date YYYY-MM-DD used for the strict dates (default: now; for tests)")
	id := fl.String("id", "", "new: change id")
	agent := fl.String("agent", "", "new: agent name (default: id)")
	branch := fl.String("branch", "", "new: branch (default: agent/<id>)")
	owns := fl.String("owns", "", "new: comma separated path globs")
	excl := fl.String("exclusive", "", "new: comma separated exclusive resources")
	_ = fl.Parse(args)

	if *root == "" {
		out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			fatal("not in a git repo and no -root")
		}
		*root = strings.TrimSpace(string(out))
	}
	split := func(s string) (v []string) {
		for _, p := range strings.Split(s, ",") {
			if p = strings.TrimSpace(p); p != "" {
				v = append(v, p)
			}
		}
		return v
	}

	switch sub {
	case "new":
		if err := scaffold(*root, *id, *agent, *branch, split(*owns), split(*excl)); err != nil {
			fatal(err.Error())
		}
		fmt.Println("created openspec/changes/" + *id + " (status proposed); fill proposal.md and tasks.md, add docs/OWNERS lines")
		return
	case "status":
		cards, _ := loadCards(*root)
		if *write {
			if err := writeStatus(*root, cards, *nMerges); err != nil {
				fatal(err.Error())
			}
			fmt.Println("docs/plan/STATUS.md block rewritten")
			return
		}
		ms, _ := gitMerges(*root, "", *nMerges)
		fmt.Print(renderBlock(*root, cards, ms))
		return
	}

	cards, fs := loadCards(*root)
	if sub == "json" {
		ms, _ := gitMergesISO(*root, *nMerges)
		out, err := renderJSON(*root, cards, ms, allFindings(*root, cards, fs, *since, *nMerges), time.Now())
		if err != nil {
			fatal(err.Error())
		}
		os.Stdout.Write(out)
		return
	}
	fs = allFindings(*root, cards, fs, *since, *nMerges)
	if *diff != "" {
		fs = append(fs, checkDiff(*root, cards, *diff)...)
	}
	if *cli {
		fs = append(fs, cliValidate(*root)...)
	}
	sortFindings(fs)
	act := 0
	for _, c := range cards {
		if c.Active() {
			act++
		}
	}
	for _, i := range infos {
		fmt.Println("INFO " + i)
	}
	for _, f := range fs {
		fmt.Printf("WARN %s: %s\n", f.Check, f.Msg)
	}
	fmt.Printf("changecheck: %d cards (%d active), %d warnings\n", len(cards), act, len(fs))
	if *strict && len(fs) > 0 {
		os.Exit(1)
	}
	if *cli {
		now := time.Now()
		if *today != "" {
			if t, err := time.Parse("2006-01-02", *today); err == nil {
				now = t
			}
		}
		bad, notes := loadStrict(*root).fatal(fs, now)
		for _, n := range notes {
			fmt.Println("changecheck: " + n)
		}
		if len(bad) > 0 {
			fmt.Printf("changecheck: %d fatal findings, exiting 1\n", len(bad))
			os.Exit(1)
		}
	}
}

// allFindings runs every card, spec, owner, merge and status check on top of the load findings fs.
func allFindings(root string, cards []*Card, fs []Finding, since string, nMerges int) []Finding {
	fs = append(fs, checkCards(cards)...)
	fs = append(fs, checkOverlap(cards)...)
	fs = append(fs, checkSpecs(root)...)
	fs = append(fs, checkOwners(root, cards)...)
	fs = append(fs, checkMerges(root, cards, since, nMerges)...)
	return append(fs, checkStatus(root, cards)...)
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "changecheck:", msg)
	os.Exit(2)
}

// cliValidate runs the pinned OpenSpec validator as a second opinion; skipped silently without node.
func cliValidate(root string) []Finding {
	if _, err := exec.LookPath("node"); err != nil {
		return nil
	}
	cmd := exec.Command("sh", root+"/tools/openspec.sh", "validate", "--all", "--strict")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	var fs []Finding
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.Contains(l, "✗") || strings.Contains(l, "ERROR") || strings.Contains(l, "WARNING") {
			fs = append(fs, Finding{"openspec-cli", strings.TrimSpace(l)})
		}
	}
	if len(fs) == 0 {
		fs = append(fs, Finding{"openspec-cli", "validate failed: " + strings.TrimSpace(string(out))})
	}
	return fs
}
