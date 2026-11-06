// Package access holds the watch screen's access choice: a passcode the tester picked (kept only as a salted
// PBKDF2 hash) or the generated token. d2dsetup writes the file, the watch gate reads it, `d2dsetup reset-access`
// replaces it. The passcode itself is never stored, logged or returned.
//
// Hash choice: the Go standard library has no scrypt or argon2, but since Go 1.24 it has crypto/pbkdf2, so the
// hash is PBKDF2-HMAC-SHA256 with a per-install 16 byte salt and 600000 iterations (the OWASP 2023 figure for
// this PRF, S). PBKDF2 is not memory-hard, so a stolen access.json is easier to attack than an scrypt one would be
// (U: a GPU can try roughly 10^6 guesses a second against this cost); the file is 0600 inside the data volume and the
// online guessing is rate limited, which is the realistic threat for a home-network tester build.
package access

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	KindPasscode = "passcode"
	KindToken    = "token"

	MinPasscode = 6
	MaxPasscode = 128 // bytes; keeps a login request small

	fileSchema = 1
)

// Iterations is the PBKDF2 cost of new passcodes. Tests lower it; the value used is stored in each record.
var Iterations = 600000

// Record is /data/access.json. For the token kind only Kind and SetAt are set: the token lives in watch.token.
type Record struct {
	Schema int    `json:"schema"`
	Kind   string `json:"kind"` // "", KindPasscode or KindToken
	Salt   string `json:"salt,omitempty"`
	Hash   string `json:"hash,omitempty"`
	Iter   int    `json:"iter,omitempty"`
	SetAt  string `json:"set_at,omitempty"`
}

// ValidatePasscode applies the whole rule set: 6 or more characters, no composition rules.
func ValidatePasscode(p string) error {
	if !utf8.ValidString(p) {
		return errors.New("the passcode must be plain text")
	}
	if n := utf8.RuneCountInString(p); n < MinPasscode {
		return fmt.Errorf("the passcode is too short: use %d or more characters (four random words work well)", MinPasscode)
	}
	if len(p) > MaxPasscode {
		return fmt.Errorf("the passcode is too long: use at most %d bytes", MaxPasscode)
	}
	return nil
}

// NewPasscode hashes p with a fresh salt.
func NewPasscode(p string, now time.Time) (Record, error) {
	if err := ValidatePasscode(p); err != nil {
		return Record{}, err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return Record{}, err
	}
	h, err := pbkdf2.Key(sha256.New, p, salt, Iterations, 32)
	if err != nil {
		return Record{}, err
	}
	return Record{Schema: fileSchema, Kind: KindPasscode, Salt: enc(salt), Hash: enc(h), Iter: Iterations, SetAt: now.UTC().Format(time.RFC3339)}, nil
}

// TokenRecord is the record of the token kind.
func TokenRecord(now time.Time) Record {
	return Record{Schema: fileSchema, Kind: KindToken, SetAt: now.UTC().Format(time.RFC3339)}
}

// Verify reports whether p is the passcode of a passcode record (constant time on the hash).
func (r Record) Verify(p string) bool {
	if r.Kind != KindPasscode || r.Iter <= 0 || len(p) > MaxPasscode {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(r.Salt)
	want, err2 := base64.RawStdEncoding.DecodeString(r.Hash)
	if err1 != nil || err2 != nil || len(want) == 0 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, p, salt, r.Iter, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

// Session is the cookie value that proves a passcode login: an HMAC keyed by the stored hash, so a new or reset
// passcode invalidates every old cookie and nothing secret is kept in memory or in a table.
func (r Record) Session() string {
	if r.Kind != KindPasscode || r.Hash == "" {
		return ""
	}
	m := hmac.New(sha256.New, []byte(r.Hash))
	m.Write([]byte("d2d-watch-session-v1"))
	return enc(m.Sum(nil))
}

// SessionOK compares a cookie value with Session in constant time.
func (r Record) SessionOK(v string) bool {
	s := r.Session()
	return s != "" && subtle.ConstantTimeCompare([]byte(v), []byte(s)) == 1
}

// SessionCookie is the passcode login cookie. HttpOnly and SameSite=Strict; Secure is not set because the screen is
// plain HTTP on a home network (a Secure cookie would never be sent back), so anyone who can watch that network's
// traffic can read it: the guide says never to publish the port on the internet.
const SessionCookie = "d2dpass"

const sessionAge = 30 * 24 * time.Hour

// Cookie is the cookie that proves a passcode login for r.
func (r Record) Cookie() *http.Cookie {
	return &http.Cookie{Name: SessionCookie, Value: r.Session(), Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: int(sessionAge / time.Second)}
}

func enc(b []byte) string { return base64.RawStdEncoding.EncodeToString(b) }

// Store reads and writes the access file. Current re-reads it when its modification time or size changed, so a
// reset made with the command line takes effect without restarting the bot.
type Store struct {
	Path string

	mu   sync.Mutex
	rec  Record
	mod  time.Time
	size int64
	seen bool
}

func NewStore(path string) *Store { return &Store{Path: path} }

// Current returns the record; a missing or unreadable file is the empty record (no access chosen yet).
func (s *Store) Current() Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	fi, err := os.Stat(s.Path)
	if err != nil {
		s.rec, s.seen = Record{}, false
		return s.rec
	}
	if s.seen && fi.ModTime().Equal(s.mod) && fi.Size() == s.size {
		return s.rec
	}
	b, err := os.ReadFile(s.Path)
	if err != nil {
		s.rec, s.seen = Record{}, false
		return s.rec
	}
	var r Record
	if json.Unmarshal(b, &r) != nil || (r.Kind != KindPasscode && r.Kind != KindToken) {
		r = Record{} // damaged: fail closed to "not chosen"
	}
	s.rec, s.mod, s.size, s.seen = r, fi.ModTime(), fi.Size(), true
	return r
}

// Save writes the record atomically with mode 0600.
func (s *Store) Save(r Record) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		return err
	}
	s.mu.Lock()
	s.seen = false
	s.mu.Unlock()
	return nil
}

// Remove deletes the file (access is then "not chosen").
func (s *Store) Remove() error {
	err := os.Remove(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	s.mu.Lock()
	s.seen = false
	s.mu.Unlock()
	return err
}
