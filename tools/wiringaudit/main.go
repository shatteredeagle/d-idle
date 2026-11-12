// Command wiringaudit lists the places where production wiring can silently turn a feature off.
//
//	go run ./tools/wiringaudit asserts   # comma-ok type assertions x.(T) outside tests (optional interfaces)
//	go run ./tools/wiringaudit knobs     # tuning struct fields (internal/env/tuning*.go) never selected outside them, non-test
//	go run ./tools/wiringaudit orphans   # funcs and methods whose name is never referenced from non-test code
//
// It is syntactic (name based, no type information): a method name shared with another type counts as used, so
// the orphan list is a lower bound, not proof. Stdlib only; run from the repo root.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type decl struct{ name, pos string }

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: wiringaudit asserts|orphans")
		os.Exit(2)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	var tests []*ast.File
	filepath.WalkDir(".", func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasPrefix(p, "tools/wiringaudit") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return nil
		}
		if strings.HasSuffix(p, "_test.go") {
			tests = append(tests, f)
		} else {
			files = append(files, f)
		}
		return nil
	})
	switch os.Args[1] {
	case "asserts":
		asserts(fset, files)
	case "knobs":
		knobs(fset, files, tests)
	case "orphans":
		orphans(fset, files, tests)
	}
}

func asserts(fset *token.FileSet, files []*ast.File) {
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			as, ok := n.(*ast.AssignStmt)
			if !ok || len(as.Lhs) != 2 || len(as.Rhs) != 1 {
				return true
			}
			ta, ok := as.Rhs[0].(*ast.TypeAssertExpr)
			if !ok || ta.Type == nil {
				return true
			}
			fmt.Printf("%s: %s.(%s)\n", fset.Position(as.Pos()), expr(ta.X), expr(ta.Type))
			return true
		})
	}
}

func expr(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return expr(v.X) + "." + v.Sel.Name
	case *ast.StarExpr:
		return "*" + expr(v.X)
	case *ast.CallExpr:
		return expr(v.Fun) + "()"
	case *ast.IndexExpr:
		return expr(v.X) + "[]"
	case *ast.InterfaceType:
		return "interface{...}"
	}
	return "?"
}

func orphans(fset *token.FileSet, files, tests []*ast.File) {
	used := map[string]bool{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.SelectorExpr:
				used[v.Sel.Name] = true
			case *ast.Ident:
				used[v.Name] = true
			}
			return true
		})
	}
	// An identifier use counts once per declaration site too, so count uses, not mentions: subtract the declaration.
	counts := map[string]int{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.SelectorExpr:
				counts[v.Sel.Name]++
			case *ast.Ident:
				counts[v.Name]++
			}
			return true
		})
	}
	inTest := map[string]bool{}
	for _, f := range tests {
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.SelectorExpr:
				inTest[v.Sel.Name] = true
			case *ast.Ident:
				inTest[v.Name] = true
			}
			return true
		})
	}
	var out []decl
	for _, f := range files {
		if f.Name.Name == "main" && false {
			continue
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			n := fd.Name.Name
			if n == "main" || n == "init" || n == "String" || n == "Error" || n == "Len" || n == "Less" || n == "Swap" ||
				strings.HasPrefix(n, "Test") || strings.HasPrefix(n, "Benchmark") || n == "MarshalJSON" || n == "UnmarshalJSON" ||
				n == "Read" || n == "Write" || n == "Close" || n == "Unwrap" || n == "ServeHTTP" || n == "Format" {
				continue
			}
			// The declaration itself is one Ident mention; anything above 1 is a use (or a same-named other symbol).
			if counts[n] <= 1 {
				tag := ""
				if inTest[n] {
					tag = " [tests only]"
				}
				recv := ""
				if fd.Recv != nil && len(fd.Recv.List) == 1 {
					recv = expr(fd.Recv.List[0].Type) + "."
				}
				out = append(out, decl{recv + n + tag, fset.Position(fd.Pos()).String()})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].pos < out[j].pos })
	for _, d := range out {
		fmt.Printf("%s: %s\n", d.pos, d.name)
	}
}

// knobs lists the fields of every struct declared in internal/env/tuning*.go whose name is never selected (x.Field)
// in non-test code outside those files. A knob that only tests or the table of knobs touch changes nothing.
func knobs(fset *token.FileSet, files, tests []*ast.File) {
	isTuning := func(f *ast.File) bool {
		n := filepath.ToSlash(fset.Position(f.Pos()).Filename)
		return strings.HasPrefix(n, "internal/env/tuning")
	}
	fields := map[string]string{} // field -> struct.pos
	used := map[string]bool{}
	for _, f := range files {
		if isTuning(f) {
			for _, d := range f.Decls {
				gd, ok := d.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, sp := range gd.Specs {
					ts, ok := sp.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, isStruct := ts.Type.(*ast.StructType)
					if !isStruct {
						continue
					}
					for _, fl := range st.Fields.List {
						for _, nm := range fl.Names {
							fields[nm.Name] = ts.Name.Name + " " + fset.Position(nm.Pos()).String()
						}
					}
				}
			}
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if se, ok := n.(*ast.SelectorExpr); ok {
				used[se.Sel.Name] = true
			}
			return true
		})
	}
	// the tuning files' own methods and accessors count as reads too when called from elsewhere
	var out []string
	inTest := map[string]bool{}
	for _, f := range tests {
		ast.Inspect(f, func(n ast.Node) bool {
			if se, ok := n.(*ast.SelectorExpr); ok {
				inTest[se.Sel.Name] = true
			}
			return true
		})
	}
	for name, where := range fields {
		if !used[name] {
			tag := ""
			if inTest[name] {
				tag = " [tests only]"
			}
			out = append(out, where+": "+name+tag)
		}
	}
	sort.Strings(out)
	for _, l := range out {
		fmt.Println(l)
	}
}
