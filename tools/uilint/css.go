package main

import (
	"fmt"
	"strings"
)

// A tiny CSS reader: enough for hand-written stylesheets (rules, at-rules, declarations), no selectors grammar.
// It tracks brace depth and the at-rules around each rule, and fails loudly on an unbalanced brace.

type decl struct {
	line        int
	prop, value string
	bare        string // value with quoted strings blanked to ""
}

type block struct {
	line    int
	at      string // at-rule name without '@' ("layer", "media", ...); empty for a style rule
	prelude string // selector of a style rule, or the at-rule prelude
	stmt    bool   // an at-rule statement ending in ';' (no block)
	decls   []decl
	parents []*block // enclosing at-rules and style rules, outermost first
}

func (b *block) inside(at string) *block {
	for i := len(b.parents) - 1; i >= 0; i-- {
		if b.parents[i].at == at {
			return b.parents[i]
		}
	}
	return nil
}

// parseCSS returns every block in source order. Braces and semicolons inside quoted strings are not structure;
// comments are dropped.
func parseCSS(src string) ([]*block, error) {
	var (
		out       []*block
		stack     []*block
		buf       strings.Builder
		line      = 1
		bufLine   = 0
		inComment bool
	)
	flushDecl := func() {
		t := strings.TrimSpace(buf.String())
		buf.Reset()
		l := bufLine
		bufLine = 0
		if t == "" {
			return
		}
		if n := len(stack); n > 0 && (stack[n-1].at == "" || stack[n-1].at == "font-face") {
			top := stack[n-1]
			p, v, ok := strings.Cut(t, ":")
			if !ok {
				return
			}
			v = strings.TrimSpace(v)
			top.decls = append(top.decls, decl{line: l, prop: strings.ToLower(strings.TrimSpace(p)), value: v, bare: blankStrings(v)})
			return
		}
		if strings.HasPrefix(t, "@") {
			name, pre, _ := strings.Cut(t[1:], " ")
			b := &block{line: l, at: strings.ToLower(name), prelude: strings.TrimSpace(pre), stmt: true, parents: append([]*block(nil), stack...)}
			out = append(out, b)
		}
	}
	rs := []rune(src)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		if c == '\n' {
			line++
		}
		if inComment {
			if c == '*' && i+1 < len(rs) && rs[i+1] == '/' {
				inComment = false
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(rs) && rs[i+1] == '*' {
			inComment = true
			i++
			continue
		}
		if bufLine == 0 && !isSpace(c) && c != '{' && c != '}' && c != ';' {
			bufLine = line
		}
		switch c {
		case '"', '\'':
			// copy the string whole so braces and semicolons inside it are not structure
			buf.WriteRune(c)
			for i++; i < len(rs) && rs[i] != c; i++ {
				if rs[i] == '\\' && i+1 < len(rs) {
					buf.WriteRune(rs[i])
					i++
				} else if rs[i] == '\n' {
					line++
				}
				buf.WriteRune(rs[i])
			}
			buf.WriteRune(c)
		case '{':
			pre := strings.TrimSpace(buf.String())
			l := bufLine
			if l == 0 {
				l = line
			}
			buf.Reset()
			bufLine = 0
			b := &block{line: l, prelude: pre, parents: append([]*block(nil), stack...)}
			if strings.HasPrefix(pre, "@") {
				name, rest, _ := strings.Cut(pre[1:], " ")
				b.at, b.prelude = strings.ToLower(strings.TrimSpace(name)), strings.TrimSpace(rest)
			}
			out = append(out, b)
			stack = append(stack, b)
		case ';':
			flushDecl()
		case '}':
			flushDecl()
			if len(stack) == 0 {
				return out, fmt.Errorf("line %d: unbalanced }", line)
			}
			stack = stack[:len(stack)-1]
		default:
			buf.WriteRune(c)
		}
	}
	if inComment {
		return out, fmt.Errorf("unterminated comment")
	}
	if len(stack) > 0 {
		return out, fmt.Errorf("line %d: unclosed {", stack[len(stack)-1].line)
	}
	return out, nil
}

func isSpace(c rune) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// selectors splits a selector list on top-level commas (not inside () or []).
func selectors(s string) []string {
	var out []string
	depth, start := 0, 0
	for i, c := range s {
		switch c {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	return append(out, strings.TrimSpace(s[start:]))
}

// blankStrings empties every quoted string ("..." or '...') so a colour word inside content: "..." does not count.
func blankStrings(s string) string {
	var b strings.Builder
	var q rune
	esc := false
	for _, c := range s {
		switch {
		case q != 0 && esc:
			esc = false
		case q != 0 && c == '\\':
			esc = true
		case q != 0 && c == q:
			q = 0
			b.WriteRune(c)
		case q != 0:
		case c == '"' || c == '\'':
			q = c
			b.WriteRune(c)
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}
