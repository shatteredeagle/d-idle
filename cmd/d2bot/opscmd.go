package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/logx"
	"d2lod-bot/internal/ops"
	"d2lod-bot/internal/questboard"
	"d2lod-bot/internal/support"
	"d2lod-bot/internal/vaultkeep"
)

const opsUsage = `usage: d2bot ops quest status [--json] [--why]
       d2bot ops quest <start|retry|cancel|handin> <quest> [--chunk ID] --why "reason"
       d2bot ops quest start <quest> [--chunk ID] --try-anyway --why "reason"   (owner override of a readiness refusal, one attempt)
       d2bot ops quest auto <quest> --on|--off --why "reason"
       d2bot ops status --why                                 (the decision ring and standing refusals, read only)
       d2bot ops save check [--char NAME] [--save DIR] [--state DIR]   (live character file validity and its backups, read only)
       d2bot ops hold|home --why "reason"                    (hold: stop and rest in town; home: queue a rest in town)
       d2bot ops go <area> [--minutes N] --why "reason"      (a go order, default 60 minutes)
       d2bot ops park <quest> <hours 1..24> --why "reason"   (stop the quest's chunk, refuse start and retry until then)
       d2bot ops go-difficulty <nightmare|hell> --why "reason"   (your go for the hop; refused with what is missing unless ready)
       d2bot ops log [--tail N] [--grep REGEXP] [--file PATH]   (the bot's own log file, read only, no socket; default N 50)
       d2bot ops vault check --dir DIR [--items FILE] [--json]   (the dry run of a vault repair: read only, no socket; DIR is
                                                              <data>/vaults/<hero>/vault; FILE: the hero's item bytes, one hex a line)
       d2bot ops quest wait <quest> [--chunk ID] [--timeout 600]
flags: --sock PATH (default $D2BOT_OPERATOR_SOCK, else the sandbox container's state directory)`

// defaultOpsSock is where the container keeps the instance state (docs/operator.md).
const defaultOpsSock = "/data/instances/main/state/" + ops.SockName

// opsMain is `d2bot ops ...`: the client of the operator socket. It returns the process exit code: 0 done, 1 refused
// or not finished, 2 usage or timeout of a wait.
func opsMain(args []string, out io.Writer) (int, error) {
	if len(args) > 0 && args[0] == "log" {
		return opsLog(args[1:], out)
	}
	if len(args) > 0 && args[0] == "vault" {
		return opsVault(args[1:], out)
	}
	if len(args) > 0 && args[0] == "save" {
		return opsSave(args[1:], out)
	}
	if len(args) > 0 && args[0] != "quest" {
		return opsHero(args, out)
	}
	if len(args) < 2 || args[0] != "quest" {
		return 2, errors.New(opsUsage)
	}
	verb, rest := args[1], args[2:]
	showWhy := false
	if verb == "status" { // here --why is a switch (print the Why-not rows), not a reason
		kept := rest[:0:0]
		for _, a := range rest {
			if a == "--why" || a == "-why" {
				showWhy = true
				continue
			}
			kept = append(kept, a)
		}
		rest = kept
	}
	fs := flag.NewFlagSet("ops quest "+verb, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	chunk := fs.String("chunk", "", "chunk id")
	why := fs.String("why", "", "reason (required for every verb that changes something)")
	sock := fs.String("sock", os.Getenv("D2BOT_OPERATOR_SOCK"), "socket path")
	tryAnyway := fs.Bool("try-anyway", false, "start: one attempt past a readiness refusal (owner override, audited, one shot)")
	on := fs.Bool("on", false, "auto on")
	off := fs.Bool("off", false, "auto off")
	asJSON := fs.Bool("json", false, "status: raw rows as JSON lines")
	timeout := fs.Int("timeout", 600, "wait: seconds")
	var pos []string
	for len(rest) > 0 { // flags may follow the quest id
		if err := fs.Parse(rest); err != nil {
			return 2, fmt.Errorf("%w\n%s", err, opsUsage)
		}
		if fs.NArg() == 0 {
			break
		}
		pos, rest = append(pos, fs.Arg(0)), fs.Args()[1:]
	}
	if *sock == "" {
		*sock = defaultOpsSock
	}
	quest := ""
	if verb != "status" {
		if len(pos) != 1 {
			return 2, errors.New(opsUsage)
		}
		quest = pos[0]
	} else if len(pos) != 0 {
		return 2, errors.New(opsUsage)
	}
	switch verb {
	case "status":
		r, err := ops.Call(*sock, ops.Request{Verb: "status"})
		if err != nil {
			return 1, err
		}
		if !r.OK || r.Board == nil {
			fmt.Fprintf(out, "ok=false code=%s reason=%q\n", r.Code, r.Reason)
			return 1, nil
		}
		printBoard(out, r.Board, *asJSON, showWhy)
		return 0, nil
	case "start", "retry", "cancel", "handin", "auto":
		rq := ops.Request{Verb: verb, Quest: quest, Chunk: *chunk, Why: *why, TryAnyway: *tryAnyway}
		if verb == "auto" {
			if *on == *off {
				return 2, errors.New("auto needs exactly one of --on and --off")
			}
			rq.On = on
		}
		r, err := ops.Call(*sock, rq)
		if err != nil {
			return 1, err
		}
		state := ""
		if r.Reply != nil {
			state = r.Reply.State
		}
		fmt.Fprintf(out, "ok=%t code=%s state=%s reason=%q\n", r.OK, r.Code, state, r.Reason)
		if r.OK {
			return 0, nil
		}
		return 1, nil
	case "wait":
		poll := func() (*ops.Board, error) {
			r, err := ops.Call(*sock, ops.Request{Verb: "status"})
			if err != nil {
				return nil, err
			}
			if r.Board == nil {
				return nil, errors.New(r.Reason)
			}
			return r.Board, nil
		}
		res := ops.Wait(poll, quest, *chunk, time.Duration(*timeout)*time.Second, 5*time.Second, time.Sleep, time.Now)
		fmt.Fprintf(out, "RESULT outcome=%s quest=%s chunk=%s state=%s step=%s progress=%d code=%s text=%q\n",
			res.Outcome, res.Quest, res.Chunk, res.State, res.Step, res.Progress, res.Code, res.Text)
		switch res.Outcome {
		case ops.OutFinished:
			return 0, nil
		case ops.OutTimeout:
			return 2, nil
		}
		return 1, nil
	}
	return 2, errors.New(opsUsage)
}

func printBoard(out io.Writer, b *ops.Board, asJSON, showWhy bool) {
	fmt.Fprintf(out, "board enabled=%t difficulty=%s why=%q\n", b.Enabled, b.Difficulty, b.Why)
	for _, r := range b.Rows {
		if asJSON {
			fmt.Fprintf(out, `{"quest":%q,"chunk":%q,"state":%q,"step":%q,"progress":%d,"code":%q,"text":%q}`+"\n", r.Quest, r.Chunk, r.State, r.Step, r.Progress, r.Code, r.Text)
			continue
		}
		fmt.Fprintf(out, "%s/%s\t%s\tauto=%s\tstep=%s\tcode=%s\t%s\n", r.Quest, r.Chunk, r.State, r.Auto, r.Step, r.Code, strings.ReplaceAll(r.Text, "\n", " "))
		if w := r.WhyNot; showWhy && w != nil {
			fmt.Fprintf(out, "  why-not\tword=%s\tfloor=%d\tneed=%s\twake=%q\tadmission=%q\tlast-refused=%s\n", w.Word, w.Floor, w.Need, w.Wake, w.Admission, lastRefused(w))
		}
	}
}

// printWaiting says which ready chunks nothing will start (only the Start tap, the start verb or an auto opt-in do)
// and why an opted-in offer is held back.
func printWaiting(out io.Writer, b *ops.Board) {
	for _, r := range b.Rows {
		if r.WaitingFor != "" {
			fmt.Fprintf(out, "waiting for Start: %s/%s (%s, auto=%s)\n", r.Quest, r.Chunk, r.WaitingFor, r.Auto)
		}
	}
	if b.AutoWhy != "" {
		fmt.Fprintf(out, "auto-start held back: %s\n", b.AutoWhy)
	}
}

func lastRefused(w *questboard.WhyNotView) string {
	if w.LastRefused == nil {
		return "never"
	}
	return w.LastRefused.UTC().Format(time.RFC3339)
}

// opsHero serves the verbs outside the quest board: status --why, hold, home, go and park (ADR 0030 amended 2026-11-25).
func opsHero(args []string, out io.Writer) (int, error) {
	verb, rest := args[0], args[1:]
	fs := flag.NewFlagSet("ops "+verb, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	sock := fs.String("sock", os.Getenv("D2BOT_OPERATOR_SOCK"), "socket path")
	if verb == "status" { // here --why is a switch (read the decision ring), not a reason
		ring := fs.Bool("why", false, "")
		if err := fs.Parse(rest); err != nil || !*ring || fs.NArg() != 0 {
			return 2, errors.New(opsUsage)
		}
		if *sock == "" {
			*sock = defaultOpsSock
		}
		return opsExplain(*sock, out)
	}
	why := fs.String("why", "", "reason (required for every verb that changes something)")
	minutes := fs.Int("minutes", 0, "go: minutes (default 60)")
	var pos []string
	for len(rest) > 0 {
		if err := fs.Parse(rest); err != nil {
			return 2, fmt.Errorf("%w\n%s", err, opsUsage)
		}
		if fs.NArg() == 0 {
			break
		}
		pos, rest = append(pos, fs.Arg(0)), fs.Args()[1:]
	}
	if *sock == "" {
		*sock = defaultOpsSock
	}
	rq := ops.Request{Verb: verb, Why: *why}
	switch verb {
	case "hold", "home":
		if len(pos) != 0 {
			return 2, errors.New(opsUsage)
		}
	case "go":
		if len(pos) < 1 {
			return 2, errors.New(opsUsage)
		}
		rq.Area, rq.Minutes = strings.Join(pos, " "), *minutes
	case ops.DifficultyVerb:
		if len(pos) != 1 {
			return 2, errors.New(opsUsage)
		}
		rq.Target = pos[0]
	case "park":
		h, err := strconv.Atoi(strings.Join(pos[min(1, len(pos)):], ""))
		if len(pos) != 2 || err != nil {
			return 2, errors.New(opsUsage)
		}
		rq.Quest, rq.Hours = pos[0], h
	default:
		return 2, errors.New(opsUsage)
	}
	r, err := ops.Call(*sock, rq)
	if err != nil {
		return 1, err
	}
	fmt.Fprintf(out, "ok=%t code=%s reason=%q\n", r.OK, r.Code, r.Reason)
	if r.OK {
		return 0, nil
	}
	return 1, nil
}

func opsExplain(sock string, out io.Writer) (int, error) {
	r, err := ops.Call(sock, ops.Request{Verb: "why"})
	if err != nil {
		return 1, err
	}
	if !r.OK {
		fmt.Fprintf(out, "ok=false code=%s reason=%q\n", r.Code, r.Reason)
		return 1, nil
	}
	b, _ := json.MarshalIndent(r.Explain, "", " ")
	fmt.Fprintf(out, "%s\n", b)
	if sr, err := ops.Call(sock, ops.Request{Verb: "status"}); err == nil && sr.Board != nil {
		printWaiting(out, sr.Board)
	}
	return 0, nil
}

// botLogName is the bot's operational log inside the instance logs directory (docs/ops/logs.md).
const botLogName = "bot.log"

// defaultBotLog is where the sandbox container keeps it.
const defaultBotLog = "/data/instances/main/logs/" + botLogName

// opsLog is `d2bot ops log`: the last lines of the bot log file and its rotations, optionally filtered. It reads the
// file, so it works when the bot is down and when `docker logs` is empty.
func opsLog(args []string, out io.Writer) (int, error) {
	fs := flag.NewFlagSet("ops log", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	tail := fs.Int("tail", 50, "last N matching lines (0 = all)")
	grep := fs.String("grep", "", "only lines matching this regular expression")
	file := fs.String("file", os.Getenv("D2BOT_LOG"), "log file")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || *tail < 0 {
		return 2, fmt.Errorf("%v\n%s", err, opsUsage)
	}
	if *file == "" {
		*file = defaultBotLog
	}
	var re *regexp.Regexp
	if *grep != "" {
		var err error
		if re, err = regexp.Compile(*grep); err != nil {
			return 2, fmt.Errorf("--grep: %w", err)
		}
	}
	lines, err := logx.Tail(*file, *tail, re)
	if err != nil {
		return 1, err
	}
	// the output goes to a terminal and from there into chats: no credential, no control character (review 4 finding 7)
	for _, l := range lines {
		fmt.Fprintln(out, logx.Printable(support.RedactCredentials(l)))
	}
	return 0, nil
}

// opsVault is `d2bot ops vault check`: what a vault repair would look at (open intents, duplicates, sales that paid for
// records they do not hold), read from the files of one hero's vault directory. It changes nothing and needs no socket.
// Exit 0: clean, 1: findings or unreadable, 2: usage.
func opsVault(args []string, out io.Writer) (int, error) {
	if len(args) == 0 || args[0] != "check" {
		return 2, errors.New(opsUsage)
	}
	fs := flag.NewFlagSet("ops vault check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", "", "the hero's vault directory")
	items := fs.String("items", "", "file with the hero's item bytes (norm hex), one a line")
	asJSON := fs.Bool("json", false, "findings as JSON lines")
	if err := fs.Parse(args[1:]); err != nil || fs.NArg() != 0 || *dir == "" {
		return 2, fmt.Errorf("%v\n%s", err, opsUsage)
	}
	var hero []string
	if *items != "" {
		b, err := os.ReadFile(*items)
		if err != nil {
			return 1, err
		}
		hero = strings.Fields(string(b))
	}
	fnd, err := vaultkeep.Check(*dir, hero)
	if err != nil {
		return 1, err
	}
	for _, f := range fnd {
		if *asJSON {
			b, _ := json.Marshal(f)
			fmt.Fprintln(out, string(b))
			continue
		}
		fmt.Fprintf(out, "%-14s %s  %s\n", f.Kind, f.ID, f.Text)
	}
	if !*asJSON {
		fmt.Fprintf(out, "vault check: %d finding(s), nothing was changed\n", len(fnd))
	}
	if len(fnd) > 0 {
		return 1, nil
	}
	return 0, nil
}
