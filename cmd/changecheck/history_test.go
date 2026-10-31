package main

import (
	"os/exec"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	a := append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=noreply@anthropic.com", "-c", "commit.gpgsign=false"}, args...)
	out, err := exec.Command("git", a...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func newRepo(t *testing.T) string {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main")
	write(t, root, "a.txt", "1")
	git(t, root, "add", ".")
	git(t, root, "commit", "-q", "-m", "init")
	return root
}

func TestSingleCommitNoHistory(t *testing.T) {
	root := newRepo(t)
	// a stale .adopted pointing at a commit that does not exist must not fail a one-commit repo
	write(t, root, "openspec/.adopted", "deadbeef\n")
	infos = nil
	if fs := checkMerges(root, nil, "", 10); len(fs) != 0 || len(infos) != 1 {
		t.Fatalf("fs=%v infos=%v", fs, infos)
	}
	if fs := checkDiff(root, []*Card{}, "agent/x"); !has(fs, "diff", "no card") {
		t.Fatalf("diff without a card still reports: %v", fs)
	}
}

func TestRealHistoryStaysFatal(t *testing.T) {
	root := newRepo(t)
	git(t, root, "checkout", "-q", "-b", "agent/x")
	write(t, root, "b.txt", "2")
	git(t, root, "add", ".")
	git(t, root, "commit", "-q", "-m", "b")
	git(t, root, "checkout", "-q", "main")
	git(t, root, "merge", "-q", "--no-ff", "-m", "Merge agent/ghost: x", "agent/x")
	infos = nil
	fs := checkMerges(root, nil, "", 10)
	if !has(fs, "merge", "ghost") || len(infos) != 0 {
		t.Fatalf("fs=%v infos=%v", fs, infos)
	}
	// a baseline that does not resolve is a real failure when history exists
	fs = checkMerges(root, nil, "deadbeef", 10)
	if !has(fs, "merge", "git log failed") {
		t.Fatalf("fs=%v", fs)
	}
	// a baseline at the first commit still sees the merge; at HEAD it sees nothing
	first := git(t, root, "rev-list", "--max-parents=0", "HEAD")
	if fs := checkMerges(root, nil, first[:7], 10); !has(fs, "merge", "ghost") {
		t.Fatalf("fs=%v", fs)
	}
	if fs := checkMerges(root, nil, "HEAD", 10); len(fs) != 0 {
		t.Fatalf("fs=%v", fs)
	}
}

func TestHistoryBaselineFromStrict(t *testing.T) {
	root := newRepo(t)
	git(t, root, "checkout", "-q", "-b", "agent/x")
	write(t, root, "b.txt", "2")
	git(t, root, "add", ".")
	git(t, root, "commit", "-q", "-m", "b")
	git(t, root, "checkout", "-q", "main")
	git(t, root, "merge", "-q", "--no-ff", "-m", "Merge agent/ghost: x", "agent/x")
	write(t, root, "openspec/.strict.yaml", "history_baseline: HEAD\n")
	if fs := checkMerges(root, nil, "", 10); len(fs) != 0 {
		t.Fatalf("baseline at HEAD should hide the old merge: %v", fs)
	}
	if loadStrict(root).HistoryBaseline != "HEAD" {
		t.Fatal("baseline not parsed")
	}
}

func TestShallowCloneNoHistory(t *testing.T) {
	src := newRepo(t)
	write(t, src, "c.txt", "3")
	git(t, src, "add", ".")
	git(t, src, "commit", "-q", "-m", "c")
	dst := t.TempDir() + "/s"
	if out, err := exec.Command("git", "clone", "-q", "--depth", "1", "file://"+src, dst).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	write(t, dst, "openspec/.adopted", "deadbeef\n")
	infos = nil
	if fs := checkMerges(dst, nil, "", 10); len(fs) != 0 || len(infos) != 1 {
		t.Fatalf("fs=%v infos=%v", fs, infos)
	}
}

func TestDefaultBranchAndDiffOnMainAndMaster(t *testing.T) {
	for _, name := range []string{"main", "master"} {
		root := t.TempDir()
		git(t, root, "init", "-q", "-b", name)
		write(t, root, "a.txt", "1")
		git(t, root, "add", ".")
		git(t, root, "commit", "-q", "-m", "init")
		write(t, root, "b.txt", "x")
		git(t, root, "add", ".")
		git(t, root, "commit", "-q", "-m", "second")
		if got := defaultBranch(root); got != name {
			t.Fatalf("%s: defaultBranch = %q", name, got)
		}
		git(t, root, "checkout", "-q", "-b", "agent/x")
		write(t, root, "c.txt", "2")
		git(t, root, "add", ".")
		git(t, root, "commit", "-q", "-m", "c")
		card := &Card{ID: "x", Fields: map[string][]string{"owns": {"c.txt"}}}
		if fs := checkDiff(root, []*Card{card}, "agent/x"); len(fs) != 0 {
			t.Fatalf("%s: diff findings: %v", name, fs)
		}
		card.Fields["owns"] = []string{"other.txt"}
		if fs := checkDiff(root, []*Card{card}, "agent/x"); !has(fs, "diff", "outside owns") {
			t.Fatalf("%s: expected outside-owns: %v", name, fs)
		}
	}
}

func TestDefaultBranchPrefersOriginHEAD(t *testing.T) {
	root := newRepo(t) // branch main
	git(t, root, "branch", "trunk")
	git(t, root, "update-ref", "refs/remotes/origin/trunk", "trunk")
	git(t, root, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
	if got := defaultBranch(root); got != "trunk" {
		t.Fatalf("got %q", got)
	}
}
