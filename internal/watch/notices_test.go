package watch

import (
	"context"
	"strings"
	"testing"
)

func TestNoticeBannerOnTheViewer(t *testing.T) {
	r := newRig(t, "sorceress")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lines := []string{"save repair: restored a from b"}
	s, err := r.sim.Hub.Serve(ctx, ServeOptions{ViewAddr: "127.0.0.1:0", AnswerAddr: "127.0.0.1:0", Token: "tok", Notices: func() []string { return lines }})
	if err != nil {
		t.Fatal(err)
	}
	view := "http://" + s.View.String()
	if code, _ := req(t, "GET", view+"/api/notices", "", "", nil); code != 401 {
		t.Fatalf("notices without token: %d", code)
	}
	if code, body := req(t, "GET", view+"/api/notices", "tok", "", nil); code != 200 || !strings.Contains(body, "save repair: restored a from b") {
		t.Fatalf("notices: %d %s", code, body)
	}
	lines = nil
	if code, body := req(t, "GET", view+"/api/notices", "tok", "", nil); code != 200 || !strings.Contains(body, `"notices":[]`) {
		t.Fatalf("no notices: %d %s", code, body)
	}
	if code, body := req(t, "GET", view+"/watch/notices.js", "tok", "", nil); code != 200 || !strings.Contains(body, "/api/notices") {
		t.Fatalf("script: %d", code)
	}
	if code, body := req(t, "GET", view+"/watch", "tok", "", nil); code != 200 || strings.Count(body, `<script src="/watch/notices.js"></script>`) != 1 {
		t.Fatalf("watch page: %d", code)
	}
}
