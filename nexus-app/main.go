package main

import (
	"os"

	"nexusdesk/internal/app"
)

func main() {
	os.Exit(app.RunWithArgs(os.Args[1:]))
}
