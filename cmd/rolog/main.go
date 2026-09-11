package main

import (
	"fmt"
	"github.com/fatima-go/fatima-cmd/controlui"
	"os"
)

func main() {
	if err := controlui.Main("rolog", legacyMain); err != nil {
		fmt.Fprintln(os.Stderr, "rolog:", err)
		os.Exit(1)
	}
}
