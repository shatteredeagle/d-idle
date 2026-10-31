package metrics

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"

	"d2lod-bot/internal/event"
)

// Ident is the correlation set stamped on every log line and event (gotcha 65): inst is the instance id,
// run the process session id (event.Event.Run), game the in-game session counter (event.Event.Game).
type Ident struct {
	Inst string
	Run  string
	Game int
}

// NewRunID returns a short random session id.
func NewRunID() string {
	var b [4]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// NewLogger returns a JSON-lines slog logger writing to w with inst, run and game on every line.
func NewLogger(w io.Writer, id Ident, level slog.Level) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	l := slog.New(h)
	if id.Inst != "" {
		l = l.With("inst", id.Inst)
	}
	if id.Run != "" {
		l = l.With("run", id.Run)
	}
	if id.Game > 0 {
		l = l.With("game", id.Game)
	}
	return l
}

// WithGame returns l with the game counter. Call it on a logger built with Game 0 (slog cannot replace an
// attribute), once per EnterGame.
func WithGame(l *slog.Logger, game int) *slog.Logger { return l.With("game", game) }

// LogEvent writes an event as one line: msg is the kind, level follows the kind. The envelope ids are
// attributes, so the event stream and the log stream join on (inst, run, game, seq).
func LogEvent(l *slog.Logger, e event.Event) {
	lvl := slog.LevelInfo
	switch e.Kind {
	case event.KindError, event.KindDeath:
		lvl = slog.LevelWarn
	case event.KindFailsafe:
		lvl = slog.LevelWarn
	case event.KindKill, event.KindPotion, event.KindDrop, event.KindActivity:
		lvl = slog.LevelDebug
	}
	attrs := []any{"seq", e.Seq, "source", e.Source}
	if e.Goal != "" {
		attrs = append(attrs, "goal", e.Goal)
	}
	if e.Data != nil {
		attrs = append(attrs, "data", e.Data)
	}
	l.Log(context.Background(), lvl, string(e.Kind), attrs...)
}
