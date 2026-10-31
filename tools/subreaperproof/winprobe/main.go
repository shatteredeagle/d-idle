// winprobe is the harmless Windows program used by tools/subreaperproof. Cross-compile with
// GOOS=windows GOARCH=386. Without arguments it starts a copy of itself as "sleeper" and exits after
// two seconds (this stands in for a launcher that exits while the real program keeps running);
// "sleeper" just sleeps.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "sleeper" {
		time.Sleep(10 * time.Minute)
		return
	}
	self, _ := os.Executable()
	c := exec.Command(self, "sleeper")
	if err := c.Start(); err != nil {
		fmt.Println("start:", err)
		os.Exit(1)
	}
	fmt.Println("launcher: started sleeper", c.Process.Pid)
	time.Sleep(2 * time.Second)
}
