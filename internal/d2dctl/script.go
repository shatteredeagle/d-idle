package d2dctl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// Step is the outcome of one script line.
type Step struct {
	N     int    `json:"n"`
	Line  string `json:"line"`
	Kind  string `json:"kind"` // send | sleep | expect
	OK    bool   `json:"ok"`
	Reply string `json:"reply,omitempty"` // payload lines + status text (an error reply ends with "err <msg>")
	Err   string `json:"err,omitempty"`
	MS    int64  `json:"ms"`
}

// RunScript runs a probe script and returns every step run. The script format, one item per line:
//
//	# comment, blank lines ignored
//	sleep 500ms          wait (Go duration)
//	expect <regex>       the last reply's text (payload lines, then "ok <rest>" rest or "err <msg>") must match
//	-<verb line>         send; an "err" reply is allowed (use with "expect err ...")
//	<verb line>          send; an "err" reply or a transport error fails the script
//
// It stops at the first failed step and returns it as the error as well. Transport errors always stop.
func RunScript(ctx context.Context, c *Client, r io.Reader) ([]Step, error) {
	var steps []Step
	var last string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), MaxLine+2)
	n := 0
	for sc.Scan() {
		raw := strings.TrimSpace(sc.Text())
		n++
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		st := Step{N: n, Line: raw}
		t0 := time.Now()
		var fail error
		switch {
		case strings.HasPrefix(raw, "sleep ") || raw == "sleep":
			st.Kind = "sleep"
			d, err := time.ParseDuration(strings.TrimSpace(strings.TrimPrefix(raw, "sleep")))
			if err != nil {
				fail = err
				break
			}
			select {
			case <-time.After(d):
			case <-ctx.Done():
				fail = ctx.Err()
			}
		case strings.HasPrefix(raw, "expect ") || raw == "expect":
			st.Kind = "expect"
			re, err := regexp.Compile(strings.TrimSpace(strings.TrimPrefix(raw, "expect")))
			if err != nil {
				fail = err
				break
			}
			if !re.MatchString(last) {
				fail = fmt.Errorf("expect %q: no match in %q", re, last)
			}
		default:
			st.Kind = "send"
			allowErr := strings.HasPrefix(raw, "-")
			line := strings.TrimSpace(strings.TrimPrefix(raw, "-"))
			rep, err := c.Do(ctx, line)
			var ce *CmdError
			switch {
			case err == nil:
				last = rep.Text()
			case errors.As(err, &ce):
				last = strings.Join(append(append([]string(nil), rep.Lines...), "err "+ce.Msg), "\n")
				if !allowErr {
					fail = err
				}
			default:
				fail = err
				last = ""
			}
			st.Reply = last
		}
		st.MS = time.Since(t0).Milliseconds()
		st.OK = fail == nil
		if fail != nil {
			st.Err = fail.Error()
		}
		steps = append(steps, st)
		if fail != nil {
			return steps, fmt.Errorf("line %d (%s): %w", n, raw, fail)
		}
	}
	return steps, sc.Err()
}
