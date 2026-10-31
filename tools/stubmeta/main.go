// stubmeta prints what a Blizzard downloader stub fetches and optionally verifies a downloaded
// installer folder against the embedded piece hashes.
//
//	stubmeta STUB.exe [DIR [-pieces]]
package main

import (
	"fmt"
	"os"

	"d2lod-bot/internal/launch"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: stubmeta STUB.exe [DIR [-pieces]]")
		os.Exit(2)
	}
	m, err := launch.ReadStubMeta(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("name=%s\ndirect download=%s\npiece length=%d total=%d bytes\n", m.Name, m.DirectDownload, m.PieceLength, m.TotalSize())
	for _, f := range m.Files {
		fmt.Printf("  %10d  %s\n", f.Length, f.Path)
	}
	if len(os.Args) > 2 {
		err := m.VerifyDir(os.Args[2], len(os.Args) > 3 && os.Args[3] == "-pieces")
		fmt.Println("verify:", map[bool]string{true: "OK", false: fmt.Sprint(err)}[err == nil])
	}
}
