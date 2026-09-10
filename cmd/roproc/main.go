package main

import (
	"fmt"
	"github.com/fatima-go/fatima-cmd/controlui"
	"os"
)

func main() {
	if err := controlui.Main("roproc", legacyMain); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
