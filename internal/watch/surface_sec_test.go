package watch

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"strings"
	"testing"
)

// Card sec2: the whole surface of a served hub, asked over real sockets. Nothing answers without the token, the
// viewer listener takes no writing verb, the answer listener changes state only by POST with the token and from an
// allowed origin, and both listeners sit on loopback when the caller asks for loopback.
func TestSurfaceNeedsTokenAndGetNeverMutates(t *testing.T) {
	const tok = "surface-test-token"
	r := newRig(t, "sorceress")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, err := r.sim.Hub.Serve(ctx, ServeOptions{ViewAddr: "127.0.0.1:0", AnswerAddr: "127.0.0.1:0", Token: tok})
	if err != nil {
		t.Fatal(err)
	}
	if !Loopback(s.View.String()) || !Loopback(s.Answer.String()) {
		t.Fatalf("bound %v and %v; want loopback", s.View, s.Answer)
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	do := func(method, url, bearer, origin string) int {
		t.Helper()
		req, err := http.NewRequest(method, url, strings.NewReader("{}"))
		if err != nil {
			t.Fatal(err)
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	writes := []string{"POST", "PUT", "PATCH", "DELETE"}
	state := func() string {
		return r.get("/api/orders", nil).Body.String() + r.get("/api/rules", nil).Body.String() + r.get(LayoutRoute, nil).Body.String()
	}
	before := state()

	view := "http://" + s.View.String()
	for _, p := range append(append([]string{}, ViewRoutes...), LogRoutes...) {
		for _, m := range append([]string{"GET"}, writes...) {
			if c := do(m, view+p, "", ""); c != http.StatusUnauthorized {
				t.Errorf("view %s %s without a token: %d, want 401", m, p, c)
			}
			if c := do(m, view+p, tok+"x", ""); c != http.StatusUnauthorized {
				t.Errorf("view %s %s with a wrong token: %d, want 401", m, p, c)
			}
		}
		for _, m := range writes {
			if c := do(m, view+p, tok, ""); c < 400 {
				t.Errorf("view %s %s with the token: %d; the viewer listener must refuse a writing verb", m, p, c)
			}
		}
	}

	ans := "http://" + s.Answer.String()
	posts := []string{"/api/answer", "/api/confirm", "/api/revoke", "/api/revoke-all", "/api/orders", "/api/orders/cancel", LayoutRoute}
	for _, p := range posts {
		if c := do("POST", ans+p, "", ""); c != http.StatusUnauthorized {
			t.Errorf("answer POST %s without a token: %d, want 401", p, c)
		}
		if c := do("POST", ans+p, tok+"x", ""); c != http.StatusUnauthorized {
			t.Errorf("answer POST %s with a wrong token: %d, want 401", p, c)
		}
		// a query token is not a credential on the answer listener: a link cannot carry a write
		if c := do("POST", ans+p+"?t="+tok, "", ""); c != http.StatusUnauthorized {
			t.Errorf("answer POST %s with ?t=: %d, want 401", p, c)
		}
		if c := do("POST", ans+p, tok, "http://evil.example"); c != http.StatusForbidden {
			t.Errorf("answer POST %s from a foreign origin: %d, want 403", p, c)
		}
		for _, m := range []string{"GET", "PUT", "PATCH", "DELETE"} {
			if c := do(m, ans+p, tok, ""); c != http.StatusMethodNotAllowed {
				t.Errorf("answer %s %s: %d, want 405", m, p, c)
			}
		}
	}
	if after := state(); after != before {
		t.Errorf("refused requests changed the hub:\n before %s\n after  %s", before, after)
	}
}

// Every handler that is built from AnswerOptions is a state-changing door. Each one must judge the token itself
// (o.tokenOK) or hand the options on to a function that does; a new POST route that forgets it fails here.
func TestEveryAnswerDoorJudgesTheToken(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatal(err)
	}
	doors := 0
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, d := range f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				name := ""
				for _, p := range fn.Type.Params.List {
					if id, ok := p.Type.(*ast.Ident); ok && id.Name == "AnswerOptions" && len(p.Names) == 1 {
						name = p.Names[0].Name
					}
				}
				if name == "" || name == "_" {
					continue
				}
				doors++
				judged := false
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "tokenOK" {
						if id, ok := sel.X.(*ast.Ident); ok && id.Name == name {
							judged = true
						}
					}
					for _, a := range call.Args {
						if id, ok := a.(*ast.Ident); ok && id.Name == name {
							judged = true
						}
					}
					return true
				})
				if !judged {
					t.Errorf("%s: %s takes AnswerOptions and neither calls %s.tokenOK nor hands %s on", fset.Position(fn.Pos()), fn.Name.Name, name, name)
				}
			}
		}
	}
	if doors < 30 {
		t.Fatalf("found %d functions with AnswerOptions; the walk is broken", doors)
	}
}
