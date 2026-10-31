package launch

import "os"

func selfPID() int { return os.Getpid() }
