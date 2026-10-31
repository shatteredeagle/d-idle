//go:build rigs

// Command d2helpersoak drives a d2input helper with realistic traffic for a long time and reports every
// failure (helper death, errors, latency). Usage (via tools/live.sh, instance from the agent):
//
//	d2helpersoak [-d 15m] [-screens 120s]
//
// Traffic: one "bot" client (mouse moves, clicks in town, key presses, seq batches, list, info, hung),
// one "sentinel" client (ping/info every second, reconnecting), short-lived clients that connect, send
// a few commands and drop (like each live.sh tool run), and every -screens a game screen change
// (Esc menu -> Save and Exit -> main menu -> Single Player -> OK), the window-handle change case.
// It never restarts the helper; it only counts how often it is unreachable.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/input"
)

var okN, errN, downN atomic.Int64

func note(who string, err error) {
	if err == nil {
		okN.Add(1)
		return
	}
	errN.Add(1)
	if input.IsHelperDown(err) {
		downN.Add(1)
	}
	fmt.Printf("%s %s ERR %v\n", time.Now().Format("15:04:05"), who, err)
}

func main() {
	d := flag.Duration("d", 15*time.Minute, "duration")
	screens := flag.Duration("screens", 120*time.Second, "screen change interval (0 = off)")
	flag.Parse()
	inst := os.Getenv("D2BOT_INSTANCE")
	end := time.Now().Add(*d)
	var wg sync.WaitGroup
	var mu sync.Mutex // the bot and the screen changer share the one game window; keep sequences whole

	wg.Add(1)
	go func() { // bot
		defer wg.Done()
		c := input.NewClient("")
		for time.Now().Before(end) {
			mu.Lock()
			x, y := 300+rand.Intn(200), 200+rand.Intn(200)
			switch rand.Intn(6) {
			case 0:
				note("bot.move", c.Seq(fmt.Sprintf("move %d %d", x, y)))
			case 1:
				_, err := c.Do(fmt.Sprintf("click l %d %d hold=60", x, y), 150*time.Millisecond)
				note("bot.click", err)
			case 2:
				note("bot.key", c.Seq("key press i hold=60", "sleep 100", "key press i hold=60"))
			case 3:
				note("bot.seq", c.Seq(fmt.Sprintf("move %d %d", x, y), "sleep 30", fmt.Sprintf("down r %d %d", x, y), "sleep 60", fmt.Sprintf("up r %d %d", x, y)))
			case 4:
				_, err := c.Do("list", 0)
				note("bot.list", err)
			case 5:
				_, err := c.Do("hung", 0)
				note("bot.hung", err)
			}
			mu.Unlock()
			time.Sleep(time.Duration(50+rand.Intn(400)) * time.Millisecond)
		}
	}()
	wg.Add(1)
	go func() { // sentinel
		defer wg.Done()
		c := input.NewClient("")
		for time.Now().Before(end) {
			note("sent.ping", c.Ping())
			_, err := c.Info()
			if err != nil && !isNoWin(err) {
				note("sent.info", err)
			}
			time.Sleep(time.Second)
		}
	}()
	wg.Add(1)
	go func() { // short-lived clients
		defer wg.Done()
		for time.Now().Before(end) {
			c := input.NewClient("")
			for i := 0; i < 1+rand.Intn(5); i++ {
				note("short.ping", c.Ping())
			}
			if rand.Intn(3) == 0 { // abrupt drop with an unread reply, like a killed tool
				c.Close()
			}
			c.Close()
			time.Sleep(time.Duration(200+rand.Intn(1500)) * time.Millisecond)
		}
	}()
	if *screens > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o := input.NewOOG(input.NewClient(""))
			for time.Now().Add(*screens).Before(end) {
				time.Sleep(*screens)
				mu.Lock()
				fmt.Println(time.Now().Format("15:04:05"), "screen change: save and exit and back")
				for _, st := range []struct {
					key  string
					x, y int
					wait time.Duration
				}{{"esc", 0, 0, 800 * time.Millisecond}, {"", 400, 261, 6 * time.Second}, {"", 400, 307, 3 * time.Second}, {"", 691, 555, 4 * time.Second}} {
					var err error
					if st.key != "" {
						err = o.Key(st.key)
					} else {
						err = o.Click(st.x, st.y)
					}
					note("screen", err)
					time.Sleep(st.wait)
				}
				mu.Unlock()
			}
		}()
	}
	t := time.NewTicker(30 * time.Second)
	go func() {
		for range t.C {
			fmt.Printf("%s soak %s ok=%d err=%d helperdown=%d\n", time.Now().Format("15:04:05"), inst, okN.Load(), errN.Load(), downN.Load())
		}
	}()
	wg.Wait()
	fmt.Printf("DONE ok=%d err=%d helperdown=%d\n", okN.Load(), errN.Load(), downN.Load())
}

func isNoWin(err error) bool { return err != nil && input.IsNoWindow(err) }
