package main

import (
	"fmt"
	"github.com/fatima-go/fatima-cmd/controlui"
	"os"
)

func main() {
	if err := controlui.Main("rohis", legacyMain); err != nil {
		fmt.Fprintln(os.Stderr, "rohis:", err)
		os.Exit(1)
	}
}
