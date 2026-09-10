package main

import (
	"fmt"
	"github.com/fatima-go/fatima-cmd/controlui"
	"os"
)

func main() {
	if err := controlui.Main("rocron", legacyMain); err != nil {
		fmt.Fprintln(os.Stderr, "rocron:", err)
		os.Exit(1)
	}
}
