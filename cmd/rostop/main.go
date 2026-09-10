package main

import (
	"fmt"
	"github.com/fatima-go/fatima-cmd/controlui"
	"os"
)

func main() {
	if err := controlui.Main("rostop", legacyMain); err != nil {
		fmt.Fprintln(os.Stderr, "rostop:", err)
		os.Exit(1)
	}
}
