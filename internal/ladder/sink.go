package ladder

import (
	"fmt"
	"strings"
	"sync"
)

// LogSink writes the ladder's visible events to the bot log as one line each:
//
//	ladder: thought t1.repair key=t1.repair|... choices=repair|skip default=repair
type LogSink struct{ Info func(msg string, kv ...any) }

func (s LogSink) Event(kind string, v ThoughtView, detail string) {
	if s.Info == nil {
		return
	}
	switch kind {
	case "thought":
		s.Info(fmt.Sprintf("ladder: thought %s key=%s choices=%s default=%s", v.Point, v.Key, strings.Join(v.Choices, "|"), v.Default))
	default:
		s.Info(fmt.Sprintf("ladder: %s %s key=%s %s", kind, v.Point, v.Key, detail))
	}
}

// RecordSink keeps the events in memory (tests and the drill record).
type RecordSink struct {
	mu   sync.Mutex
	Rows []string
}

func (s *RecordSink) Event(kind string, v ThoughtView, detail string) {
	s.mu.Lock()
	s.Rows = append(s.Rows, fmt.Sprintf("%s %s key=%s %s", kind, v.Point, v.Key, detail))
	s.mu.Unlock()
}

func (s *RecordSink) Lines() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.Rows...)
}

// Tee sends every event to all sinks.
type Tee []Sink

func (t Tee) Event(kind string, v ThoughtView, detail string) {
	for _, s := range t {
		s.Event(kind, v, detail)
	}
}
