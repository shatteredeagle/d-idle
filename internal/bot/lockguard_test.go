package bot

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// guarded names the locks that must never be held across a caller-supplied callback or a call into the game
// (docs/locks.md). The scan is deliberately small: straight-line Lock ... Unlock regions of these locks, in these
// packages, calling these callees. It has no false positives today; extend the lists with the incident, not by guess.
var (
	guardedLocks = map[string]bool{"r.obMu": true, "n.covMu": true}
	bannedCalls  = map[string]bool{"OnSnapshot": true, "onCover": true, "done": true, "stopWhen": true, "Health.Step": true}
)

func TestNoCallbackUnderObMuOrCovMu(t *testing.T) {
	for _, dir := range []string{".", "../nav"} {
		files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			fs := token.NewFileSet()
			f, err := parser.ParseFile(fs, path, b, 0)
			if err != nil {
				t.Fatal(err)
			}
			text := func(n ast.Node) string { return string(b[fs.Position(n.Pos()).Offset:fs.Position(n.End()).Offset]) }
			ast.Inspect(f, func(n ast.Node) bool {
				fd, ok := n.(*ast.FuncDecl)
				if !ok || fd.Body == nil {
					return true
				}
				held := ""
				for _, s := range fd.Body.List {
					mu, op := lockOp(s, text)
					switch {
					case guardedLocks[mu] && (op == "Lock"):
						held = mu
						continue
					case op == "Unlock" && mu == held:
						if _, deferred := s.(*ast.DeferStmt); !deferred {
							held = ""
						}
						continue
					}
					if held == "" {
						continue
					}
					ast.Inspect(s, func(n ast.Node) bool {
						if _, lit := n.(*ast.FuncLit); lit {
							return false
						}
						if c, ok := n.(*ast.CallExpr); ok {
							callee := text(c.Fun)
							name := callee[strings.LastIndex(callee, ".")+1:]
							if bannedCalls[name] || bannedCalls["Health."+name] && strings.HasSuffix(callee, "Health.Step") {
								t.Errorf("%s:%d %s calls %s while holding %s (docs/locks.md)", path, fs.Position(c.Pos()).Line, fd.Name.Name, callee, held)
							}
						}
						return true
					})
				}
				return false
			})
		}
	}
}

func lockOp(s ast.Stmt, text func(ast.Node) string) (mu, op string) {
	var c *ast.CallExpr
	switch x := s.(type) {
	case *ast.ExprStmt:
		c, _ = x.X.(*ast.CallExpr)
	case *ast.DeferStmt:
		c = x.Call
	}
	if c == nil {
		return "", ""
	}
	if se, ok := c.Fun.(*ast.SelectorExpr); ok && (se.Sel.Name == "Lock" || se.Sel.Name == "Unlock") {
		return text(se.X), se.Sel.Name
	}
	return "", ""
}
