package task

import (
	"context"
	"errors"
	"fmt"
)

// Kind classifies a failure (tasklayer.md section 6). It replaces per-package sentinel matching for steps.
type Kind int

const (
	KindNone Kind = iota
	KindPrereq
	KindResource
	KindNotFound
	KindDanger
	KindUI
	KindTimeout
	KindDeath
	KindFatal
)

func (k Kind) String() string {
	return [...]string{"none", "prereq", "resource", "not_found", "danger", "ui", "timeout", "death", "fatal"}[k]
}

// Fault says why a step cannot or did not run. Insert holds prerequisite steps the executor may splice in front.
type Fault struct {
	Kind   Kind
	Why    string
	Insert []Step
}

func (f *Fault) Error() string { return f.Kind.String() + ": " + f.Why }

// kindErr tags an error with a Kind.
type kindErr struct {
	kind Kind
	err  error
}

func (e *kindErr) Error() string { return e.err.Error() }
func (e *kindErr) Unwrap() error { return e.err }

// Wrap tags err with a kind; adapters in the bot use it to map their sentinels (nav.ErrNoPath -> KindNotFound ...).
func Wrap(k Kind, err error) error {
	if err == nil {
		return nil
	}
	return &kindErr{k, err}
}

// Errorf is Wrap(k, fmt.Errorf(...)).
func Errorf(k Kind, format string, a ...any) error { return &kindErr{k, fmt.Errorf(format, a...)} }

// Classify maps any error to a Kind. Untagged errors are KindUI (retry, then skip) except context deadlines.
func Classify(err error) Kind {
	var ke *kindErr
	var f *Fault
	switch {
	case err == nil:
		return KindNone
	case errors.As(err, &ke):
		return ke.kind
	case errors.As(err, &f):
		return f.Kind
	case errors.Is(err, context.DeadlineExceeded):
		return KindTimeout
	case errors.Is(err, context.Canceled):
		return KindFatal
	}
	return KindUI
}
