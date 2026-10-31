package town_test

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/town"
)

var _ quest.Cube = (*town.Town)(nil)

func cubeCodes(r *rig, c d2.Container) map[string]int {
	m := map[string]int{}
	for _, u := range r.W.Items {
		if u.Item.Where == c {
			m[r.W.DB.Items[u.Class].Code]++
		}
	}
	return m
}

func TestTransmuteStaff(t *testing.T) {
	r := newRig(t)
	r.W.Add("box", d2.ContInventory, 0, 0)
	r.W.Add("msf", d2.ContInventory, 4, 0)
	r.W.Add("vip", d2.ContInventory, 5, 0)
	out, err := r.T.Transmute(context.Background(), []string{"msf", "Top of the Horadric Staff"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "hst" {
		t.Fatalf("out %v", out)
	}
	if m := cubeCodes(r, d2.ContInventory); m["hst"] != 1 || m["msf"] != 0 || m["vip"] != 0 || m["box"] != 1 {
		t.Fatalf("pack %v", m)
	}
	if n := count(r.W, d2.ContCube); n != 0 || r.W.UI[d2.UICube] != 0 {
		t.Fatalf("cube left %d ui %v", n, r.W.UI)
	}
}

func TestTransmuteUnequipsQuestItems(t *testing.T) {
	r := newRig(t)
	r.W.Add("box", d2.ContInventory, 0, 0)
	sh := r.W.Add("msf", d2.ContEquipment, 0, 0)
	sh.Item.Body = d2.BodyRHand
	r.W.Items[len(r.W.Items)-1] = sh
	tp := r.W.Add("vip", d2.ContEquipment, 0, 0)
	tp.Item.Body = d2.BodyNeck
	r.W.Items[len(r.W.Items)-1] = tp
	out, err := r.T.Transmute(context.Background(), []string{"msf", "vip"})
	if err != nil || len(out) != 1 || out[0] != "hst" {
		t.Fatalf("out %v err %v", out, err)
	}
	if count(r.W, d2.ContEquipment) != 0 {
		t.Fatal("still worn")
	}
}

func TestTransmuteFailureReturnsInputs(t *testing.T) {
	r := newRig(t)
	r.W.Add("box", d2.ContInventory, 0, 0)
	r.W.Add("msf", d2.ContInventory, 4, 0)
	r.W.Add("hp1", d2.ContInventory, 5, 0)
	_, err := r.T.Transmute(context.Background(), []string{"msf", "hp1"})
	if !errors.Is(err, town.ErrCubeNoResult) {
		t.Fatalf("err %v", err)
	}
	if m := cubeCodes(r, d2.ContInventory); m["msf"] != 1 || m["hp1"] != 1 || count(r.W, d2.ContCube) != 0 {
		t.Fatalf("pack %v", m)
	}
}

func TestTransmuteMissing(t *testing.T) {
	r := newRig(t)
	r.W.Add("msf", d2.ContInventory, 4, 0)
	if _, err := r.T.Transmute(context.Background(), []string{"msf"}); !errors.Is(err, town.ErrCubeMissing) {
		t.Fatalf("err %v", err)
	}
}
