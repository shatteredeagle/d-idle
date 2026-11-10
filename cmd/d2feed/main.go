// Command d2feed turns a signed release (the output of tools/release/app-assets.sh plus d2sign) into a static
// feed tree the update client can read, and checks that tree with the real client before anything is uploaded
// (docs/friend-feed.md). It is host-agnostic: it only writes and reads directories. Standard library only.
// It never reads a private key; signing is d2sign's job.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"d2lod-bot/internal/selfupdate"
)

const usage = `usage: d2feed <command> [flags]
  build    -release D -pub F[,F] -base-url URL -out D [-repo owner/name]   verify a signed release set and add it to the tree
  withdraw -out D -version V [-repo owner/name]                           hide a release from clients (files stay)
  verify   -out D -pub F[,F] -base-url URL [-repo owner/name]             run the real update client against the tree`

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errw, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "build":
		err = cmdBuild(args[1:], out)
	case "withdraw":
		err = cmdWithdraw(args[1:], out)
	case "verify":
		err = cmdVerify(args[1:], out)
	default:
		fmt.Fprintln(errw, usage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(errw, "d2feed:", args[0]+":", err)
		return 1
	}
	return 0
}

func repoOrDefault(r string) string {
	if r == "" {
		return selfupdate.DefaultRepo
	}
	return strings.Trim(r, "/")
}
