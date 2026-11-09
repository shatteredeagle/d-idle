package main

import (
	"bufio"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// WALKSMOOTH3: the bot pipelines the eight reads of a snapshot in one write; the mux must hand them to d2d in one write
// too (d2d answers every complete line of its buffer in one frame), and keep the order of the replies.
func TestBotBatchGoesUpstreamInOneWrite(t *testing.T) {
	dir := t.TempDir()
	up := filepath.Join(dir, "up.sock")
	ln, err := net.Listen("unix", up)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	var mu sync.Mutex
	var reads []int // complete lines per upstream read
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		buf := make([]byte, 4096)
		var pend string
		for {
			n, err := c.Read(buf)
			if err != nil {
				return
			}
			pend += string(buf[:n])
			lines := strings.Split(pend, "\n")
			pend = lines[len(lines)-1]
			lines = lines[:len(lines)-1]
			mu.Lock()
			reads = append(reads, len(lines))
			mu.Unlock()
			for _, l := range lines {
				c.Write([]byte("payload " + l + "\nok\n"))
			}
		}
	}()
	mux := NewMux(up)
	bot := filepath.Join(dir, "bot.sock")
	bl, err := mux.Serve(bot, true, "/tmp/shots")
	if err != nil {
		t.Fatal(err)
	}
	defer bl.Close()
	c, err := net.Dial("unix", bot)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Write([]byte("state\nmonsters\nnpcs\nground\n")); err != nil {
		t.Fatal(err)
	}
	rd := bufio.NewReader(c)
	for _, want := range []string{"state", "monsters", "npcs", "ground"} {
		a, _ := rd.ReadString('\n')
		b, _ := rd.ReadString('\n')
		if strings.TrimSpace(a) != "payload "+want || strings.TrimSpace(b) != "ok" {
			t.Fatalf("reply for %s: %q %q", want, a, b)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	total := 0
	for _, n := range reads {
		total += n
	}
	// the four lines may arrive split if the bot's write was split by the kernel, but never one line per read
	if total != 4 || len(reads) > 2 {
		t.Fatalf("upstream reads %v, want the 4 lines in at most 2 reads", reads)
	}
}
