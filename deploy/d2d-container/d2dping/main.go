// Command d2dping sends one verb to a d2d control socket and prints the reply; it is the container healthcheck
// (`d2dping -sock /run/d2d/d2d.sock.view ping`) and the smoke test's client (`d2dping -sock ... state`).
// A bare d2d takes ONE client, so the healthcheck must dial the read-only tap socket (the entrypoint always starts the tap).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
)

func run(args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("d2dping", flag.ContinueOnError)
	fs.SetOutput(errw)
	sock := fs.String("sock", os.Getenv("D2D_SOCK")+".view", "d2d control socket (default $D2D_SOCK.view, the read-only tap)")
	wait := fs.Duration("t", 4*time.Second, "timeout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	verb := strings.Join(fs.Args(), " ")
	if verb == "" {
		verb = "ping"
	}
	ctx, cancel := context.WithTimeout(context.Background(), *wait)
	defer cancel()
	c := d2dctl.New(*sock, d2dctl.Options{NoReconnect: true, NoAutoKick: true})
	rep, err := c.Do(ctx, verb)
	if err != nil {
		fmt.Fprintln(errw, "d2dping:", err)
		return 1
	}
	fmt.Fprintln(out, rep.Text())
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
