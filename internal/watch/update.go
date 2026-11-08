package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"d2lod-bot/internal/selfupdate"
)

// The update API of the watch screen (TASK-122; docs/self-update-release.md, docs/research/self-update-design.md
// 3.6). Reads are on the viewer, every write is on the answer listener behind the token and the origin check.
// Nothing is silent: a download and an apply each need a click, and apply names the version it was shown.

// UpdateRoutes are the GET routes on the viewer. /api/update is the same body as /api/update/status.
var UpdateRoutes = []string{"/api/update", "/api/update/status", "/watch/update.js"}

// UpdatePostRoutes are the POST routes on the answer listener.
var UpdatePostRoutes = []string{
	"/api/update/check", "/api/update/download", "/api/update/apply", "/api/update/cancel",
	"/api/update/rollback", "/api/update/inbox", "/api/update/seen",
}

// Updater is what the screen needs from the engine (selfupdate.Engine implements it).
type Updater interface {
	Status() selfupdate.Status
	Check(ctx context.Context) error
	Download(ctx context.Context) error
	Apply(confirm string, force bool) error
	Cancel()
	Rollback() error
	Inbox() error
	MarkSeen() error
}

var _ Updater = (*selfupdate.Engine)(nil)

type updateReply struct {
	OK     bool               `json:"ok"`
	Code   string             `json:"code,omitempty"`
	Error  string             `json:"error,omitempty"`
	Status *selfupdate.Status `json:"status,omitempty"`
}

type updateBody struct {
	Confirm string `json:"confirm"`
	Force   bool   `json:"force"`
}

// UpdateHandler serves the GET routes (read only).
func UpdateHandler(u Updater) http.Handler {
	m := http.NewServeMux()
	status := func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, u.Status()) }
	m.HandleFunc("/api/update", status)
	m.HandleFunc("/api/update/status", status)
	m.HandleFunc("/watch/update.js", func(w http.ResponseWriter, r *http.Request) {
		b, err := webFS.ReadFile("web/update.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write(b)
	})
	return readOnly(m)
}

// UpdateAnswerHandler serves the POST routes with the same token and origin rules as the other writes.
func UpdateAnswerHandler(u Updater, o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && containsStr(o.Origins, origin)
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		fail := func(code int, c, msg string) { writeJSON(w, code, updateReply{Code: c, Error: msg}) }
		switch {
		case r.Method == http.MethodOptions && allowed:
			w.Header().Set("Access-Control-Allow-Methods", "POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusNoContent)
			return
		case r.Method != http.MethodPost:
			w.Header().Set("Allow", "POST, OPTIONS")
			fail(http.StatusMethodNotAllowed, "method", "POST only")
			return
		case origin != "" && !allowed:
			fail(http.StatusForbidden, "origin", "origin not allowed")
			return
		case !o.tokenOK(r):
			fail(http.StatusUnauthorized, "token", "unauthorized")
			return
		case !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json"):
			fail(http.StatusUnsupportedMediaType, "content-type", "send application/json")
			return
		}
		var b updateBody
		dec := json.NewDecoder(io.LimitReader(r.Body, maxBody+1))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&b); err != nil || dec.More() {
			fail(http.StatusBadRequest, "body", "body must be one small JSON object with known fields")
			return
		}
		var err error
		switch r.URL.Path {
		case "/api/update/check":
			err = u.Check(r.Context())
		case "/api/update/download":
			// A download runs on past the request: the page polls the status. The request context ends with the response.
			go u.Download(context.Background())
		case "/api/update/apply":
			if b.Confirm == "" {
				fail(http.StatusBadRequest, "confirm", "apply needs confirm=<version>")
				return
			}
			err = u.Apply(b.Confirm, b.Force)
		case "/api/update/cancel":
			u.Cancel()
		case "/api/update/rollback":
			err = u.Rollback()
		case "/api/update/inbox":
			err = u.Inbox()
		case "/api/update/seen":
			err = u.MarkSeen()
		default:
			fail(http.StatusNotFound, "route", "no such route")
			return
		}
		st := u.Status()
		if err != nil {
			code := http.StatusConflict
			c := "update"
			switch {
			case errors.Is(err, selfupdate.ErrNotSafe):
				c = "not-safe"
			case errors.Is(err, selfupdate.ErrConfirm), errors.Is(err, selfupdate.ErrNotReady):
				c = "state"
			case errors.Is(err, selfupdate.ErrOffline):
				c = "offline"
			case errors.Is(err, selfupdate.ErrBadSignature), errors.Is(err, selfupdate.ErrHashMismatch), errors.Is(err, selfupdate.ErrRevoked), errors.Is(err, selfupdate.ErrDowngrade):
				c = "refused"
			}
			writeJSON(w, code, updateReply{Code: c, Error: err.Error(), Status: &st})
			return
		}
		writeJSON(w, 200, updateReply{OK: true, Status: &st})
	})
}

// withUpdateBanner adds the update script to the screens' HTML without touching the pages themselves.
func withUpdateBanner(next http.Handler) http.Handler {
	const tag = `<script src="/watch/update.js"></script>`
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/watch") || strings.Contains(r.URL.Path[1:], ".") {
			next.ServeHTTP(w, r)
			return
		}
		rec := &bufWriter{h: http.Header{}, code: 200}
		next.ServeHTTP(rec, r)
		body := rec.buf.Bytes()
		if strings.HasPrefix(rec.h.Get("Content-Type"), "text/html") && bytes.Contains(body, []byte("</body>")) {
			body = bytes.Replace(body, []byte("</body>"), []byte(tag+"</body>"), 1)
		}
		for k, v := range rec.h {
			w.Header()[k] = v
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(rec.code)
		w.Write(body)
	})
}

type bufWriter struct {
	h    http.Header
	buf  bytes.Buffer
	code int
}

func (b *bufWriter) Header() http.Header         { return b.h }
func (b *bufWriter) Write(p []byte) (int, error) { return b.buf.Write(p) }
func (b *bufWriter) WriteHeader(c int)           { b.code = c }
