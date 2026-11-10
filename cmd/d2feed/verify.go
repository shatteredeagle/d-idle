package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/selfupdate"
)

// cmdVerify serves the tree on a loopback port, rewrites the public base URL to that port on the fly, and runs the
// production client and engine (Check, then Download with its hash checks) with only the pinned keys. It does not
// apply anything. A tree that passes here is what a friend's image will accept.
func cmdVerify(args []string, out io.Writer) error {
	fs := newFlags("verify")
	outDir := fs.String("out", "", "feed tree directory")
	pubs := fs.String("pub", "", "pinned public key file(s), comma separated")
	base := fs.String("base-url", "", "https URL the feed will be served at")
	repo := fs.String("repo", "", "owner/name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outDir == "" || *base == "" {
		return errors.New("-out and -base-url are required")
	}
	keys, err := loadKeys(*pubs)
	if err != nil {
		return err
	}
	baseURL, err := checkBaseURL(*base)
	if err != nil {
		return err
	}
	rp := repoOrDefault(*repo)
	list, err := loadList(listPath(*outDir, rp))
	if err != nil || len(list) == 0 {
		return fmt.Errorf("no releases in %s", listPath(*outDir, rp))
	}
	wantTag := ""
	for _, r := range list {
		if !r.Draft && !r.Pre && (wantTag == "" || selfupdate.CompareVersions(strings.TrimPrefix(r.Tag, selfupdate.TagPrefix), strings.TrimPrefix(wantTag, selfupdate.TagPrefix)) > 0) {
			wantTag = r.Tag
		}
	}
	if wantTag == "" {
		return errors.New("every release in the feed is withdrawn")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	local := "http://" + ln.Addr().String()
	root := filepath.Clean(*outDir)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(filepath.ToSlash(filepath.Clean("/"+r.URL.Path)), "/")))
		b, err := os.ReadFile(p)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/releases") {
			b = bytes.ReplaceAll(b, []byte(baseURL), []byte(local))
			w.Header().Set("Content-Type", "application/json")
		}
		w.Write(b)
	})}
	go srv.Serve(ln)
	defer srv.Close()

	u, _ := url.Parse(local)
	tmp, err := os.MkdirTemp("", "d2feed-verify-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	e := selfupdate.New(selfupdate.Config{
		Layout: selfupdate.Layout{Root: tmp}, Keys: keys, Running: "0.0.0", ImageABI: 1 << 20, Local: selfupdate.BuildCompat, CheckEnabled: true,
		Client: &selfupdate.Client{Repo: rp, BaseURL: local, UserAgent: "d2feed-verify", AllowHost: func(h string) bool { return h == u.Hostname() }},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := e.Check(ctx); err != nil {
		return fmt.Errorf("client check: %w", err)
	}
	a := e.Status().Available
	if a == nil || selfupdate.TagPrefix+a.Version != wantTag {
		return fmt.Errorf("client offers %+v, the tree's newest listed release is %s", a, wantTag)
	}
	if err := e.Download(ctx); err != nil {
		return fmt.Errorf("client download: %w", err)
	}
	if st := e.Status(); st.State != selfupdate.StateReady {
		return fmt.Errorf("client state %s after download: %s", st.State, st.LastError)
	}
	fmt.Fprintf(out, "verified: the update client accepts %s with the pinned key (check and download, nothing applied)\n", wantTag)
	return nil
}
