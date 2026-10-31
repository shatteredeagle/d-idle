package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/party"
)

// partyRun holds the live party between commands of one oog run (all commands run in one process).
type partyRun struct {
	p     *party.Party
	srcs  map[string]mem.Source // instance id -> game memory
	locks []func()
}

var pr partyRun

// partyCmd: "party i1:Navi:leader i2:Combi:killer". Members are driven by party.FlowDriver over launch.Flow,
// each on its own instance (own memory, own d2input helper). The second instance's lock is taken here (live.sh
// holds the first one).
func partyCmd(ctx context.Context, own string, specs []string) error {
	pr.srcs = map[string]mem.Source{}
	dir := party.NewDirectory()
	var members []party.Member
	for _, sp := range specs {
		f := strings.Split(sp, ":")
		if len(f) != 3 {
			return fmt.Errorf("member %q: want id:Character:role", sp)
		}
		id, name := f[0], f[1]
		role, err := party.Parse(f[2])
		if err != nil {
			return err
		}
		if id != own {
			unlock, err := launch.LockInstance(id, 30*time.Second)
			if err != nil {
				return fmt.Errorf("lock %s: %w", id, err)
			}
			pr.locks = append(pr.locks, unlock)
		}
		src, err := mem.OpenInstance(id)
		if err != nil {
			return fmt.Errorf("open %s: %w", id, err)
		}
		pr.srcs[id] = src
		cli := input.NewClientInstance("", id)
		flow := &launch.Flow{Mem: src, In: typer{input.NewOOG(cli), cli},
			Log: func(s string) { fmt.Printf("  flow[%s]: %s\n", id, s) }}
		members = append(members, party.Member{ID: id, Character: name, Role: role,
			Driver: &party.FlowDriver{Flow: flow, Char: launch.CharSpec{Name: name, Difficulty: "normal"},
				Dir: dir, PartyID: "live"}})
	}
	p, err := party.New(party.Config{ID: "live", Dir: dir, JoinStagger: 3 * time.Second}, members)
	if err != nil {
		return err
	}
	pr.p = p
	return p.Start(ctx)
}

func partyOp(ctx context.Context, op, id string) error {
	if pr.p == nil {
		return fmt.Errorf("no party started")
	}
	switch op {
	case "pleave":
		return pr.p.Leave(ctx, id)
	case "plost":
		return pr.p.MemberLost(ctx, id)
	case "prejoin":
		return pr.p.Rejoin(ctx, id)
	case "pdisband":
		return pr.p.Disband(ctx)
	case "pstatus":
		s := pr.p.Status()
		fmt.Printf("  party state=%s host=%s gen=%d game=%+v err=%q\n", s.State, s.Host, s.Gen, s.Game, s.LastErr)
		for _, m := range s.Members {
			fmt.Printf("  member %s %s role=%s state=%s host=%v\n", m.ID, m.Character, m.Role, m.State, m.Host)
		}
		return nil
	case "pplayers":
		for id, src := range pr.srcs {
			fmt.Printf("  instance %s:\n", id)
			if err := players(src); err != nil {
				fmt.Println("   error:", err)
			}
		}
	}
	return nil
}
