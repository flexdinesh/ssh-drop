package main

import (
	"os"

	"github.com/dineshpandiyan/ssh-drop/internal/app"
)

var version = "dev"

func main() {
	os.Exit(app.Run(os.Args[1:], app.RealDeps(version)))
}
