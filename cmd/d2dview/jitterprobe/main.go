// Command jitterprobe measures the frame pacing of a d2dview game video, read-only: one /stream.mjpg client that
// stamps every part on arrival, and (optionally) the frame ring header polled every millisecond, which shows when
// d2d published each frame and with which game time. It prints gap distributions (docs/viewer-fps.md section 7).
//
//	JITTER_TOKEN=... jitterprobe -url http://127.0.0.1:8893 -ring /dev/shm/d2dview/frames.ring -secs 60
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type part struct {
	at     time.Time
	size   int
	gameMS int64 // X-Frame-Game-Ms, -1 when the server does not send it
}

type pub struct {
	at             time.Time
	seq, frame, ms uint64
}

func main() {
	url := flag.String("url", "http://127.0.0.1:8893", "d2dview base URL")
	ring := flag.String("ring", "", "frame ring file to poll (empty: stream only)")
	secs := flag.Int("secs", 60, "measurement length")
	path := flag.String("path", "/stream.mjpg", "stream path (with query)")
	flag.Parse()
	end := time.Now().Add(time.Duration(*secs) * time.Second)
	parts := make(chan []part, 1)
	go func() { parts <- stream(*url+*path, os.Getenv("JITTER_TOKEN"), end) }()
	var pubs []pub
	if *ring != "" {
		time.Sleep(2 * time.Second) // the stream wakes the sampler and the lease
		pubs = poll(*ring, end.Add(-time.Second))
	}
	ps := <-parts
	report(ps, pubs)
}

func stream(url, token string, end time.Time) []part {
	req, _ := http.NewRequest("GET", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "stream:", err)
		return nil
	}
	defer resp.Body.Close()
	rd := bufio.NewReaderSize(resp.Body, 1<<20)
	var out []part
	for time.Now().Before(end) {
		line, err := rd.ReadString('\n')
		if err != nil {
			break
		}
		if !strings.HasPrefix(line, "--") {
			continue
		}
		p := part{gameMS: -1}
		for {
			h, err := rd.ReadString('\n')
			if err != nil || h == "\r\n" {
				break
			}
			k, v, _ := strings.Cut(strings.TrimSpace(h), ": ")
			switch strings.ToLower(k) {
			case "content-length":
				p.size, _ = strconv.Atoi(v)
			case "x-frame-game-ms":
				p.gameMS, _ = strconv.ParseInt(v, 10, 64)
			}
		}
		if _, err := io.CopyN(io.Discard, rd, int64(p.size)); err != nil {
			break
		}
		p.at = time.Now()
		out = append(out, p)
	}
	return out
}

func poll(path string, end time.Time) []pub {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ring:", err)
		return nil
	}
	defer f.Close()
	var out []pub
	var hb [24]byte
	last := uint64(0)
	for time.Now().Before(end) {
		if _, err := f.ReadAt(hb[:], 24); err == nil {
			le := binary.LittleEndian
			if s := le.Uint64(hb[:]); s != last {
				last = s
				out = append(out, pub{time.Now(), s, le.Uint64(hb[8:]), le.Uint64(hb[16:])})
			}
		}
		time.Sleep(time.Millisecond)
	}
	return out
}

func report(ps []part, pubs []pub) {
	if len(pubs) > 2 {
		pubs = pubs[1:]
		var wall, game, frm []float64
		skips := 0
		for i := 1; i < len(pubs); i++ {
			a, b := pubs[i-1], pubs[i]
			wall = append(wall, ms(b.at.Sub(a.at)))
			game = append(game, float64(b.ms)-float64(a.ms))
			frm = append(frm, float64(b.frame-a.frame))
			skips += int(b.seq - a.seq - 1)
		}
		span := pubs[len(pubs)-1].at.Sub(pubs[0].at).Seconds()
		fmt.Printf("ring: %d publishes in %.1fs = %.2f/s, seqs the 1 ms poll missed: %d, game s per wall s %.3f\n",
			len(pubs)-1, span, float64(len(pubs)-1)/span, skips, (float64(pubs[len(pubs)-1].ms)-float64(pubs[0].ms))/1000/span)
		dist("ring publish gap, wall ms", wall)
		dist("ring publish gap, game ms", game)
		dist("ring publish gap, d2d frames", frm)
	}
	if len(ps) > 2 {
		span := ps[len(ps)-1].at.Sub(ps[0].at).Seconds()
		bytes := 0
		var gaps, game []float64
		for i, p := range ps {
			bytes += p.size
			if i > 0 {
				gaps = append(gaps, ms(p.at.Sub(ps[i-1].at)))
				if p.gameMS >= 0 && ps[i-1].gameMS >= 0 {
					game = append(game, float64(p.gameMS-ps[i-1].gameMS))
				}
			}
		}
		fmt.Printf("stream: %d parts in %.1fs = %.2f fps, %.0f KB/part, %.0f KB/s\n", len(ps), span,
			float64(len(ps)-1)/span, float64(bytes)/float64(len(ps))/1024, float64(bytes)/1024/span)
		dist("stream arrival gap, wall ms", gaps)
		if len(game) > 0 {
			dist("stream part gap, game ms", game)
		}
	}
}

func dist(name string, g []float64) {
	sort.Float64s(g)
	n := len(g)
	if n == 0 {
		return
	}
	q := func(p float64) float64 { return g[min(n-1, int(p*float64(n)))] }
	cnt := func(f func(float64) bool) (c int) {
		for _, x := range g {
			if f(x) {
				c++
			}
		}
		return
	}
	mean := 0.0
	for _, x := range g {
		mean += x
	}
	mean /= float64(n)
	sd := 0.0
	for _, x := range g {
		sd += (x - mean) * (x - mean)
	}
	sd = math.Sqrt(sd / float64(n))
	fmt.Printf("  %s: n=%d mean=%.1f sd=%.1f p50=%.1f p90=%.1f p99=%.1f max=%.1f; <20:%d >60:%d >80:%d >120:%d >200:%d\n",
		name, n, mean, sd, q(.5), q(.9), q(.99), g[n-1], cnt(func(x float64) bool { return x < 20 }),
		cnt(func(x float64) bool { return x > 60 }), cnt(func(x float64) bool { return x > 80 }),
		cnt(func(x float64) bool { return x > 120 }), cnt(func(x float64) bool { return x > 200 }))
	h := map[int]int{}
	for _, x := range g {
		h[min(int(x)/10*10, 200)]++
	}
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var sb strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&sb, " %d:%d", k, h[k])
	}
	fmt.Printf("    histogram (10 ms bins):%s\n", sb.String())
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
