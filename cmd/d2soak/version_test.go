package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCollectorVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			w.Write([]byte(`{"version":"d2bot abc1234-dirty"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := &Collector{HTTP: srv.Client(), Base: srv.URL}
	if v := c.Version(context.Background()); v != "abc1234-dirty" {
		t.Fatalf("got %q", v)
	}
	c.Base = srv.URL + "/nope"
	if v := c.Version(context.Background()); v != "unknown" {
		t.Fatalf("got %q", v)
	}
}

func TestReportShowsVersion(t *testing.T) {
	r := Report{Pass: true, BotVersion: "abc1234"}
	if !strings.Contains(r.Text(), "bot version  abc1234") {
		t.Fatal(r.Text())
	}
	dir := t.TempDir()
	os.WriteFile(dir+"/"+versionFile, []byte("abc1234\n"), 0o644)
	if loadVersion(dir) != "abc1234" || loadVersion(t.TempDir()) != "" {
		t.Fatal("loadVersion")
	}
}
