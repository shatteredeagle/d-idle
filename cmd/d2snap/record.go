package main

import (
	"fmt"
	"os"
	"reflect"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/mem"
)

// record runs d2.Reader.Snapshot against the live game through a mem.Recording, writes the touched
// pages as a mem.Snapshot fixture, then replays the fixture and checks it decodes identically.
// If key is not empty it is pressed first (and pressed again afterwards to close the panel).
func record(src mem.Source, path, key string) {
	var h *helper
	if key != "" {
		var err error
		if h, err = dialHelper(); err != nil {
			fmt.Println("helper:", err)
			return
		}
		defer h.c.Close()
		h.send("key press " + key)
		time.Sleep(600 * time.Millisecond)
	}
	rec := mem.NewRecording(src)
	live, err := d2.New(rec).Snapshot()
	if h != nil {
		h.send("key press " + key)
		time.Sleep(400 * time.Millisecond)
	}
	if err != nil {
		fmt.Println("snapshot:", err)
		return
	}
	snap := rec.Snapshot()
	if err := snap.Save(path); err != nil {
		fmt.Println("save:", err)
		return
	}
	st, _ := os.Stat(path)
	replay, err := d2.New(snap).Snapshot()
	if err != nil {
		fmt.Println("replay:", err)
		return
	}
	fmt.Printf("recorded %s: %d pages, %d bytes, state=%v\n", path, snap.NumPages(), st.Size(), live.State)
	if live.Player != nil {
		fmt.Printf("player %+v\n", *live.Player)
	}
	fmt.Println("replay identical to live read:", reflect.DeepEqual(live, replay))
}
