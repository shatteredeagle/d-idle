// Command d2view serves the live display with a simulated game feed so the UI
// can be tried without the game. The same hub mounts in the real bot.
package main

import (
	"context"
	"d2lod-bot/internal/buildinfo"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"d2lod-bot/internal/status"
	"d2lod-bot/internal/telemetry"
)

func main() {
	buildinfo.Exit("d2view")
	addr := flag.String("addr", "127.0.0.1:8765", "listen address")
	speed := flag.Float64("speed", 12, "simulation speed multiplier (sim time per real time)")
	flag.Parse()

	clk := &simClock{start: time.Now(), speed: *speed}
	tr := telemetry.New(clk, telemetry.Config{ExpForLevel: telemetry.DefaultExpForLevel, AreaName: status.AreaName})
	sim := newSim(tr, 7)
	tr.SetGoal(telemetry.Goal{Name: "Reach level 12", Detail: "Sorceress leveling, Blood Moor runs"})

	hub := status.NewHub()
	hub.RegisterTelemetry(tr, status.AreaName)
	hub.Register(status.NewCharacterPanel(sim))
	hub.Register(status.NewInventoryPanel(sim, sim))
	hub.Register(status.NewWorldPanel(sim, status.AreaName))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go hub.Run(ctx)
	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				sim.Step()
				if lvl := tr.Overview().Level; lvl > 0 {
					tr.SetGoal(telemetry.Goal{Name: "Reach level 12", Progress: float64(lvl) / 12, Detail: fmt.Sprintf("level %d of 12", lvl)})
				}
			}
		}
	}()

	mux := http.NewServeMux()
	hub.Mount(mux, "/live/")
	mux.Handle("/", http.RedirectHandler("/live/", http.StatusFound))
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("d2view: simulated feed (speed x%g) at http://%s/live/\n", *speed, ln.Addr())
	srv := &http.Server{Handler: mux}
	go func() { <-ctx.Done(); srv.Shutdown(context.Background()) }()
	if err := srv.Serve(ln); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
