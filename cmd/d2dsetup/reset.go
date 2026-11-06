package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/access"
)

// resetAccess is `d2dsetup reset-access`: the way back in when the passcode is lost. It runs inside the container
// (docker compose exec), needs no password because whoever can run it already owns the data volume, and takes effect
// at once because the watch gate re-reads access.json on every request (old login cookies stop working).
//
//	d2dsetup reset-access              switch to the generated token (kept in watch.token) and say where to read it
//	d2dsetup reset-access -passcode    read a new passcode from standard input (one line, 6 or more characters)
func resetAccess(args []string, in io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("reset-access", flag.ContinueOnError)
	fs.SetOutput(out)
	dataDir := fs.String("data", env("D2SETUP_DATA", "/data"), "state directory (access.json, watch.token)")
	tokenFile := fs.String("token-file", env("D2BOT_WATCH_TOKEN_FILE", ""), "watch token file (default <data>/watch.token); created 0600 when missing")
	pass := fs.Bool("passcode", false, "set a new passcode read from standard input instead of switching to the token")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("reset-access takes no arguments")
	}
	if *tokenFile == "" {
		*tokenFile = filepath.Join(*dataDir, "watch.token")
	}
	st := access.NewStore(filepath.Join(*dataDir, accessFile))
	if *pass {
		line, err := bufio.NewReader(io.LimitReader(in, 1024)).ReadString('\n')
		if err != nil && line == "" {
			return errors.New("no passcode on standard input: pipe it in, for example: printf '%s\\n' \"$PASSCODE\" | docker compose exec -T <service> d2dsetup reset-access -passcode")
		}
		rec, err := access.NewPasscode(strings.TrimRight(line, "\r\n"), time.Now())
		if err != nil {
			return err
		}
		if err := st.Save(rec); err != nil {
			return err
		}
		fmt.Fprintln(out, "d2dsetup: the new passcode is set. Old logins stopped working; open the watch screen and enter it.")
		return nil
	}
	if _, _, err := loadOrCreateToken(*tokenFile); err != nil {
		return err
	}
	if err := st.Save(access.TokenRecord(time.Now())); err != nil {
		return err
	}
	fmt.Fprintf(out, "d2dsetup: access is back on the generated token. Old logins stopped working.\nd2dsetup: read the token with: cat %s\nd2dsetup: open the watch screen as /watch?t=<token>. To choose a new passcode later, run this command again with -passcode.\n", *tokenFile)
	return nil
}
