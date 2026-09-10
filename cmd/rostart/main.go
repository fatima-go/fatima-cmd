package main

import (
	"fmt"
	"github.com/fatima-go/fatima-cmd/controlui"
	"os"
)

func main() {
	if err := controlui.Main("rostart", legacyMain); err != nil {
		fmt.Fprintln(os.Stderr, "rostart:", err)
		os.Exit(1)
	}
}
