package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-cmd/deployui"
	"github.com/fatima-go/fatima-core/opm/transport"
	"golang.org/x/term"
)

func main() {
	// The legacy entry point still receives its original arguments and flags.
	// No v2 request has been sent when this explicit path is chosen.
	for i, a := range os.Args[1:] {
		if a == "--legacy" {
			os.Args = append(os.Args[:i+1], os.Args[i+2:]...)
			legacyMain()
			return
		}
	}
	if e := runV2(); e != nil {
		fmt.Fprintln(os.Stderr, "rodeploy:", e)
		os.Exit(1)
	}
}
func runV2() error {
	fs := flag.NewFlagSet("rodeploy", flag.ContinueOnError)
	opts := deployui.Options{}
	fs.StringVar(&opts.Group, "g", "", "package group")
	fs.StringVar(&opts.First, "p", "", "first package (host:name)")
	fs.BoolVar(&opts.Debug, "d", false, "show connection diagnostics (credentials are never logged)")
	fs.BoolVar(&opts.JSON, "json", false, "JSON output for automation")
	fs.StringVar(&opts.RequestID, "request-id", "", "idempotency key for upload/create")
	fs.StringVar(&opts.ArtifactID, "artifact", "", "artifact ID for create")
	fs.StringVar(&opts.Action, "action", "", "continue, resume, retry, cancel")
	fs.Uint64Var(&opts.Revision, "revision", 0, "reviewed rollout revision for action")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "rodeploy [flags] [upload FAR | artifacts | rollouts | watch ID | create | act ID]\nNo arguments opens the deployment TUI. A bare FAR path opens the upload screen.\n--legacy [original flags] FAR runs the unchanged HTTP command.")
		fs.PrintDefaults()
	}
	if e := fs.Parse(os.Args[1:]); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	args := fs.Args()
	if len(args) > 0 {
		opts.Command = args[0]
		if len(args) > 1 {
			opts.Value = args[1]
		}
		if opts.Command != "upload" && opts.Command != "artifacts" && opts.Command != "rollouts" && opts.Command != "watch" && opts.Command != "create" && opts.Command != "act" {
			opts.Value = opts.Command
			opts.Command = "upload"
		}
	}
	// Interactive discovery belongs inside the TUI so initial connection and
	// authentication failures use the same frame as every other screen.
	if !opts.JSON && term.IsTerminal(int(os.Stdin.Fd())) {
		return deployui.RunInteractive(opts)
	}
	cfg, e := config.GetActiveContext()
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	caps, e := transport.Discover(ctx, cfg.Jupiter)
	cancel()
	if errors.Is(e, transport.ErrLegacy) {
		opts.Legacy = true
	} else if e != nil {
		return fmt.Errorf("capability lookup failed; no deployment submitted: %w", e)
	} else if caps.Server != "jupiter" || !transport.Supports(caps, "rollouts") {
		return fmt.Errorf("endpoint does not advertise Jupiter rollout support")
	}
	opts.Endpoint = cfg.Jupiter
	if opts.Legacy && opts.JSON {
		return fmt.Errorf("server supports legacy HTTP only; use --legacy with a FAR path")
	}
	if opts.Legacy && !term.IsTerminal(int(os.Stdin.Fd())) && opts.Command == "upload" && opts.Value != "" {
		os.Args = []string{os.Args[0]}
		if opts.Group != "" {
			os.Args = append(os.Args, "-g", opts.Group)
		}
		if opts.First != "" {
			os.Args = append(os.Args, "-p", opts.First)
		}
		os.Args = append(os.Args, opts.Value)
		legacyMain()
		return nil
	}
	if opts.JSON {
		c, e := deployui.NewClient(cfg)
		if e != nil {
			return e
		}
		defer c.Close()
		return deployui.RunJSON(c, opts)
	}
	return deployui.Run(cfg, opts)
}
