package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"d2lod-bot/internal/d2dsoakstat"
)

// cmdReport prints the per-skill section (docs/soakskills.md) for each bot log. The melee skill is named by
// -melee (the skill-data name, never an id); without it the swing line prints n/a.
func cmdReport(args []string) error {
	return reportSkills(args, os.Stdout)
}

func reportSkills(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	melee := fs.String("melee", "", "name of the basic-attack skill in the skill data (its casts are the staff swings)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("no input")
	}
	for _, p := range fs.Args() {
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		s, err := d2dsoakstat.ParseSkills(f)
		f.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		fmt.Fprint(out, s.Text(p, *melee))
		af, err := os.Open(p)
		if err != nil {
			return err
		}
		as, err := d2dsoakstat.ParseAreas(af)
		af.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		fmt.Fprint(out, as.Text(p))
	}
	return nil
}
