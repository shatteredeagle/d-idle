package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// scaffold writes openspec/changes/<id>/ with a proposed card; the coordinator fills Why, tasks and gates.
func scaffold(root, id, agent, branch string, owns, excl []string) error {
	if !idRe.MatchString(id) {
		return fmt.Errorf("id %q must be lowercase kebab-case", id)
	}
	dir := filepath.Join(root, "openspec", "changes", id)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("%s exists", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if agent == "" {
		agent = id
	}
	if branch == "" {
		branch = "agent/" + id
	}
	list := func(v []string) string {
		if len(v) == 0 {
			return " []"
		}
		return "\n  - " + strings.Join(v, "\n  - ")
	}
	files := map[string]string{
		"proposal.md": "## Why\n\nTODO: why this job exists, 50 to 1000 characters, with the evidence (doc, log line, user decision).\n\n## What Changes\n\n- TODO\n",
		"tasks.md": fmt.Sprintf("# Tasks\n\n```yaml\nchange: %s\nstatus: proposed\nagent: %s\nbranch: %s\nworktree: ~/code/d2lod-wt/%s\nowns:%s\nexclusive:%s\ngates:\n  - \"TODO: a number or a named test\"\nmerge:\nresult:\n```\n\n## 1. Work\n\n- [ ] 1.1 TODO task and how it is verified\n",
			id, agent, branch, id, list(owns), list(excl)),
		".openspec.yaml": fmt.Sprintf("schema: spec-driven\ncreated: %s\nskip_specs: true\n", time.Now().Format("2006-01-02")),
	}
	for n, c := range files {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(c), 0o644); err != nil {
			return err
		}
	}
	return nil
}
