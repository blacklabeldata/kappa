package main

import (
	"runtime"

	"github.com/blacklabeldata/kappa/commands"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	commands.Execute()
}
