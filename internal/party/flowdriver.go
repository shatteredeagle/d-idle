package party

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"d2lod-bot/internal/launch"
)

// OOG is the slice of launch.Flow the driver needs; *launch.Flow satisfies it.
type OOG interface {
	HostGame(ctx context.Context, ch launch.CharSpec) error
	JoinGame(ctx context.Context, ch launch.CharSpec, hostIP string) error
	LeaveGame(ctx context.Context, timeout time.Duration) error
}

// LANPort is the TCP port a TCP/IP host binds (VERIFIED LIVE: 0.0.0.0:4000, owned by the host prefix's
// wineserver; a second instance in another prefix joining over 127.0.0.1 did not bind it).
const LANPort = "4000"

// FlowDriver is the real Driver: one member's menus, over launch.Flow, for LAN TCP/IP games only (never
// Battle.net). Game.Name and Game.Password are NOT typed anywhere: a TCP/IP game has no name or password
// dialog (VERIFIED LIVE); they only identify the generation in the Directory so a stale join is refused.
type FlowDriver struct {
	Flow         OOG
	Char         launch.CharSpec // Name (and Difficulty); LAN fields are set per call
	Dir          *Directory      // optional: when set, Join refuses unless the Directory still holds g
	PartyID      string
	OpTimeout    time.Duration // bound on one Create/Join (default 3 min); a join once sat on the loading screen for minutes
	LeaveTimeout time.Duration // default 30 s
	ProbeWait    time.Duration // how long Join waits for the host port (default 10 s)
	// Probe reports whether addr (host:port) accepts a TCP connection. Default dials and closes at once.
	Probe func(ctx context.Context, addr string) error
	Sleep func(time.Duration)
}

var ErrStaleGame = errors.New("party: game is not the Directory's current one")
var ErrHostDown = errors.New("party: host port does not answer")

func (d *FlowDriver) bound(ctx context.Context) (context.Context, context.CancelFunc) {
	t := d.OpTimeout
	if t == 0 {
		t = 3 * time.Minute
	}
	return context.WithTimeout(ctx, t)
}

func (d *FlowDriver) leaveTimeout() time.Duration {
	if d.LeaveTimeout == 0 {
		return 30 * time.Second
	}
	return d.LeaveTimeout
}

// Create leaves any current game first (Start would treat "already in game as this character" as done, and
// that game is not the one to host), then hosts. Returns once the character is in the game.
func (d *FlowDriver) Create(ctx context.Context, g Game) error {
	ctx, cancel := d.bound(ctx)
	defer cancel()
	if err := d.Flow.LeaveGame(ctx, d.leaveTimeout()); err != nil {
		return fmt.Errorf("leave before hosting: %w", err)
	}
	if err := d.Flow.HostGame(ctx, d.Char); err != nil {
		return err
	}
	return nil
}

// Join checks the Directory entry, waits for the host port, leaves any current game, then joins g.Host.
func (d *FlowDriver) Join(ctx context.Context, g Game) error {
	ctx, cancel := d.bound(ctx)
	defer cancel()
	if d.Dir != nil {
		cur, ok := d.Dir.Lookup(d.PartyID)
		if !ok || cur.Name != g.Name {
			return ErrStaleGame
		}
	}
	if err := d.waitHost(ctx, net.JoinHostPort(g.Host, LANPort)); err != nil {
		return err
	}
	if err := d.Flow.LeaveGame(ctx, d.leaveTimeout()); err != nil {
		return fmt.Errorf("leave before joining: %w", err)
	}
	return d.Flow.JoinGame(ctx, d.Char, g.Host)
}

// Leave saves and exits to the main menu (leaving as host ends the game for everybody, V).
func (d *FlowDriver) Leave(ctx context.Context) error {
	return d.Flow.LeaveGame(ctx, d.leaveTimeout())
}

func (d *FlowDriver) waitHost(ctx context.Context, addr string) error {
	probe := d.Probe
	if probe == nil {
		probe = dialProbe
	}
	wait := d.ProbeWait
	if wait == 0 {
		wait = 10 * time.Second
	}
	sleep := d.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	var err error
	for waited := time.Duration(0); ; waited += 500 * time.Millisecond {
		if err = probe(ctx, addr); err == nil {
			return nil
		}
		if waited >= wait || ctx.Err() != nil {
			return fmt.Errorf("%w: %s: %v", ErrHostDown, addr, err)
		}
		sleep(500 * time.Millisecond)
	}
}

func dialProbe(ctx context.Context, addr string) error {
	var nd net.Dialer
	c, err := nd.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return c.Close()
}
