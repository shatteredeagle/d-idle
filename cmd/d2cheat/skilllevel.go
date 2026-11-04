package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/cheat/d2s"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/paths"
)

// skillTables is what skilllevel needs from the game tables, so tests do not need an install.
type skillTables struct {
	skill     func(name string) (id int, classCode string, ok bool) // skills.txt row by key or display name
	classCode func(class int) string                                // d2s class index -> skills.txt charclass code
}

func dataTables(db *data.DB) skillTables {
	return skillTables{
		skill: func(name string) (int, string, bool) {
			s := db.SkillByName(name)
			if s == nil {
				return 0, "", false
			}
			return s.ID, s.Class, true
		},
		classCode: func(class int) string {
			for _, c := range db.Classes {
				if c.ID == class {
					return c.Code
				}
			}
			return ""
		},
	}
}

// skillLevelRequest resolves skill by name against the save's class and builds the saveedit request.
// Unknown skill or class errors come before any ledger write (nothing was attempted); the throwaway
// check and the running-game check stay in cheat.Env.Do, which ledgers the refusal.
func skillLevelRequest(t skillTables, env *cheat.Env, file, skill string, level int, reason, actor string) (cheat.Request, error) {
	if level < 0 || level > 255 {
		return cheat.Request{}, fmt.Errorf("skilllevel: level %d out of range 0-255", level)
	}
	path := file
	if !strings.ContainsAny(file, `/\`) {
		if !strings.HasSuffix(strings.ToLower(file), ".d2s") {
			file += ".d2s"
		}
		path = filepath.Join(env.SaveDir, file)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return cheat.Request{}, fmt.Errorf("skilllevel: %w", err)
	}
	f, err := d2s.Parse(b)
	if err != nil {
		return cheat.Request{}, fmt.Errorf("skilllevel: %w", err)
	}
	id, code, ok := t.skill(skill)
	if !ok {
		return cheat.Request{}, fmt.Errorf("skilllevel: unknown skill %q (use the skills.txt key, e.g. \"Fire Bolt\")", skill)
	}
	want := t.classCode(f.Class())
	if want == "" {
		return cheat.Request{}, fmt.Errorf("skilllevel: unknown class %d in %s", f.Class(), filepath.Base(path))
	}
	if code != want {
		return cheat.Request{}, fmt.Errorf("skilllevel: %q is a %q skill, %s is class %q", skill, code, f.Name(), want)
	}
	return cheat.Request{Kind: cheat.KindSaveEdit, Reason: reason, Actor: actor, Params: map[string]any{
		"file":      path,
		"skills":    fmt.Sprintf("%d:%d", id, level),
		"skillname": skill,
	}}, nil
}

// cmdSkillLevel: d2cheat --reason R skilllevel CHAR|FILE "Skill Name" LEVEL (game stopped; backup and ledger like save).
func cmdSkillLevel(env *cheat.Env, reason, actor string, rest []string) {
	if len(rest) != 3 {
		fatal(fmt.Errorf(`usage: skilllevel CHAR|FILE "Skill Name" LEVEL`))
	}
	lvl, err := strconv.Atoi(rest[2])
	if err != nil {
		fatal(fmt.Errorf("skilllevel: level %q: %w", rest[2], err))
	}
	if strings.TrimSpace(reason) == "" { // before opening the tables: fail fast; Do ledgers the same refusal
		_, err := env.Do(context.Background(), cheat.Request{Kind: cheat.KindSaveEdit, Actor: actor})
		fatal(err)
	}
	l := paths.Layout{}
	if pl, err := paths.FromEnv(); err == nil {
		l = pl
	}
	db, err := data.OpenCached(cheat.DefaultInstallDir(env.Instance), l, filepath.Join(l.HomeDir(), ".cache", "d2lod-cheat-data"))
	if err != nil {
		fatal(fmt.Errorf("skilllevel: game tables: %w", err))
	}
	req, err := skillLevelRequest(dataTables(db), env, rest[0], rest[1], lvl, reason, actor)
	if err != nil {
		fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	res, err := env.Do(ctx, req)
	if err != nil {
		fatal(err)
	}
	fmt.Println(res.Message)
	printDiff(res.Entry.Before, res.Entry.After)
}
