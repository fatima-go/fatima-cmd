package main

import (
	"fmt"
	"github.com/fatima-go/fatima-cmd/controlui"
	"os"
)

func main() {
	if err := controlui.Main("rodis", legacyMain); err != nil {
		fmt.Fprintln(os.Stderr, "rodis:", err)
		os.Exit(1)
	}
}
