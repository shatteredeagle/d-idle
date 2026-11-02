// lockstepscan: READ-ONLY probe P1 of docs/research/original-game-lockstep.md.
// It looks for 32-bit counters in the game process that grow by about 25 per wall second (the game frame
// counter model), then samples the survivors for a while and prints frames per second and jitter.
// No writes, no hook. Result of a run: docs/research/lockstep-p1p2.md.
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/mem"
)

type region struct{ lo, hi uint32 }

func regions(pid int) []region {
	f, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		fmt.Println("maps:", err)
		os.Exit(1)
	}
	defer f.Close()
	var out []region
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) < 2 || !strings.HasPrefix(fs[1], "rw") {
			continue
		}
		ab := strings.SplitN(fs[0], "-", 2)
		lo, _ := strconv.ParseUint(ab[0], 16, 64)
		hi, _ := strconv.ParseUint(ab[1], 16, 64)
		if lo >= 1<<31 || hi > 1<<31 || hi-lo > 256<<20 {
			continue
		}
		out = append(out, region{uint32(lo), uint32(hi)})
	}
	return out
}

func snap(p *mem.Proc, rs []region) map[uint32][]byte {
	m := map[uint32][]byte{}
	for _, r := range rs {
		b := make([]byte, r.hi-r.lo)
		if p.ReadAt(b, r.lo) == nil {
			m[r.lo] = b
		}
	}
	return m
}

// watch ADDR SECONDS: sample one counter every 5 s and print frames per wall second per window and in total.
func watch(p *mem.Proc, args []string) {
	ad64, _ := strconv.ParseUint(strings.TrimPrefix(args[0], "0x"), 16, 32)
	secs, _ := strconv.Atoi(args[1])
	var buf [4]byte
	rd := func() (uint32, bool) {
		if p.ReadAt(buf[:], uint32(ad64)) != nil {
			return 0, false
		}
		return binary.LittleEndian.Uint32(buf[:]), true
	}
	v0, ok := rd()
	if !ok {
		fmt.Println("read error at start")
		os.Exit(1)
	}
	t0 := time.Now()
	pv, pt := v0, t0
	errs := 0
	for time.Since(t0) < time.Duration(secs)*time.Second {
		time.Sleep(5 * time.Second)
		v, ok := rd()
		if !ok {
			errs++
			continue
		}
		now := time.Now()
		fmt.Printf("t=%.1fs frame=%d window_fps=%.3f\n", now.Sub(t0).Seconds(), v, float64(v-pv)/now.Sub(pt).Seconds())
		pv, pt = v, now
	}
	fmt.Printf("TOTAL frames=%d wall=%.3fs fps=%.4f read_errors=%d\n", pv-v0, pt.Sub(t0).Seconds(), float64(pv-v0)/pt.Sub(t0).Seconds(), errs)
}

func main() {
	p, err := mem.OpenGame()
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer p.Close()
	if len(os.Args) == 4 && os.Args[1] == "watch" {
		watch(p, os.Args[2:])
		return
	}
	rate := 25.0
	if len(os.Args) == 2 && strings.HasPrefix(os.Args[1], "k=") { // expected game speed factor (hook runs)
		rate, _ = strconv.ParseFloat(os.Args[1][2:], 64)
		rate *= 25
	}
	pid, _ := mem.FindGame()
	rs := regions(pid)
	fmt.Printf("regions rw below 2G: %d\n", len(rs))
	a := snap(p, rs)
	t0 := time.Now()
	time.Sleep(2 * time.Second)
	b := snap(p, rs)
	dt := time.Since(t0).Seconds()
	lo, hi := uint32(rate*dt*0.9), uint32(rate*dt*1.1+1)
	var cand []uint32
	for base, x := range a {
		y := b[base]
		if len(y) != len(x) {
			continue
		}
		for i := 0; i+4 <= len(x); i += 4 {
			va, vb := binary.LittleEndian.Uint32(x[i:]), binary.LittleEndian.Uint32(y[i:])
			if vb-va >= lo && vb-va <= hi && va > 0 && va < 1<<28 {
				cand = append(cand, base+uint32(i))
			}
		}
	}
	fmt.Printf("pass1 dt=%.3fs delta in [%d,%d]: %d candidates\n", dt, lo, hi, len(cand))
	// filter by two more rounds of direct reads
	last := map[uint32]uint32{}
	var buf [4]byte
	rd := func(ad uint32) (uint32, bool) {
		if p.ReadAt(buf[:], ad) != nil {
			return 0, false
		}
		return binary.LittleEndian.Uint32(buf[:]), true
	}
	for _, ad := range cand {
		last[ad], _ = rd(ad)
	}
	tl := time.Now()
	for round := 0; round < 2; round++ {
		time.Sleep(2 * time.Second)
		d := time.Since(tl).Seconds()
		tl = time.Now()
		lo, hi := uint32(rate*d*0.9), uint32(rate*d*1.1+1)
		for ad, v := range last {
			n, ok := rd(ad)
			if !ok || n-v < lo || n-v > hi {
				delete(last, ad)
				continue
			}
			last[ad] = n
		}
		fmt.Printf("round %d: %d survivors\n", round+1, len(last))
	}
	// 10 s measurement, one sample per second.
	type ser struct {
		v []uint32
		t []time.Time
	}
	ss := map[uint32]*ser{}
	for ad := range last {
		ss[ad] = &ser{}
	}
	for i := 0; i <= 10; i++ {
		now := time.Now()
		for ad, s := range ss {
			if v, ok := rd(ad); ok {
				s.v = append(s.v, v)
				s.t = append(s.t, now)
			}
		}
		if i < 10 {
			time.Sleep(time.Second)
		}
	}
	for ad, s := range ss {
		if len(s.v) < 11 {
			continue
		}
		span := s.t[10].Sub(s.t[0]).Seconds()
		fps := float64(s.v[10]-s.v[0]) / span
		mn, mx := 1e9, 0.0
		for i := 1; i <= 10; i++ {
			d := float64(s.v[i]-s.v[i-1]) / s.t[i].Sub(s.t[i-1]).Seconds()
			if d < mn {
				mn = d
			}
			if d > mx {
				mx = d
			}
		}
		fmt.Printf("addr=%08x fps=%.3f span=%.3fs persec_min=%.2f persec_max=%.2f first=%d\n", ad, fps, span, mn, mx, s.v[0])
	}
}
