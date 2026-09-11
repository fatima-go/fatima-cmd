package controlui

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-core/opm/api"
	"github.com/fatima-go/fatima-core/opm/operations"
	"github.com/fatima-go/fatima-core/opm/transport"
	"golang.org/x/term"
	"os"
	"strconv"
	"time"
)

const CronNotice = "실제 배치의 시작·완료·성공 여부는 확인할 수 없습니다. 배치 로그나 업무 결과를 확인해 주세요."

type Options struct {
	Action, RegistryGroup                                         string
	Plan                                                          *api.RegistryPlan
	Sort                                                          string
	Command, Package, Process, Job, Arguments, RequestID, WatchID string
	Listing, Plain, JSON, Debug, TUI                              bool
	Group                                                         string
	All                                                           bool
	Targets                                                       []string
	Level                                                         string
	pickPackage                                                   bool // rolog screen lists packages instead of failing
}

func Main(command string, legacyMain func()) error {
	for i, a := range os.Args[1:] {
		if a == "--legacy" {
			os.Args = append(os.Args[:i+1], os.Args[i+2:]...)
			legacyMain()
			return nil
		}
	}
	opts, err := parse(command, os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if !opts.JSON && !opts.Plain && !opts.Listing && term.IsTerminal(int(os.Stdin.Fd())) {
		opts.pickPackage = opts.Command == "rolog"
		m := model{opts: opts, ctx: ctx, stage: "select", width: 100, height: 30, status: "접속 확인 중"}
		final, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
		if err != nil {
			return err
		}
		result := final.(model)
		if result.client != nil {
			result.client.Close()
		}
		if result.fallback {
			if err := legacyOptionsError(opts); err != nil {
				return err
			}
			restoreLegacyArgs(opts)
			legacyMain()
			return nil
		}
		if result.op != nil && (result.op.State == "FAILED" || result.op.State == "INTERRUPTED") {
			return fmt.Errorf("%s: %s", result.op.State, result.op.Message)
		}
		return result.err
	}
	check, done := context.WithTimeout(ctx, 10*time.Second)
	c, err := connect(check, opts)
	done()
	if legacy(err) {
		if err := legacyOptionsError(opts); err != nil {
			return err
		}
		restoreLegacyArgs(opts)
		legacyMain()
		return nil
	}
	if err != nil {
		return err
	}
	defer c.Close()
	return runPlain(ctx, c, opts)
}
func legacyOptionsError(o Options) error {
	if o.JSON || o.WatchID != "" || o.RequestID != "" || o.Job != "" || o.Arguments != "" || (o.Command == "rocron" && o.Process != "") || (o.Command == "ropack" && o.Group != "") {
		return fmt.Errorf("this option requires the new command API; use --legacy for the original command")
	}
	return nil
}
func parse(command string, args []string) (Options, error) {
	o := Options{Command: command}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.StringVar(&o.Package, "p", "", "target host:package (default package may be omitted)")
	f.BoolVar(&o.Debug, "d", false, "connection diagnostics")
	f.BoolVar(&o.Plain, "plain", false, "single invocation without TUI")
	f.BoolVar(&o.JSON, "json", false, "structured output without TUI")
	f.BoolVar(&o.TUI, "tui", false, "interactive terminal UI")
	f.StringVar(&o.RequestID, "request-id", "", "stable ID for a mutating request")
	f.StringVar(&o.WatchID, "watch", "", "reconnect to an existing request ID")
	if command == "rocron" {
		f.BoolVar(&o.Listing, "l", false, "list registered cron jobs and schedules")
		f.StringVar(&o.Process, "process", "", "cron process")
		f.StringVar(&o.Job, "job", "", "cron job to request")
		f.StringVar(&o.Arguments, "arguments", "", "cron argument string")
	} else if command == "rodis" {
		f.StringVar(&o.Sort, "s", "name", "sort by name or index")
	} else if command == "ropack" {
		f.StringVar(&o.Group, "g", "", "package group filter")
	} else if command == "rolog" {
		f.Usage = func() {
			fmt.Fprintln(f.Output(), "usage: rolog [options] [PROCESS LEVEL]\nOmit the arguments to choose a process and level interactively. LEVEL: error, warn, info, debug, trace.")
			f.PrintDefaults()
		}
	} else if command == "roproc" {
		f.Usage = func() {
			fmt.Fprintln(f.Output(), "usage: roproc [options] [add PROCESS [GROUP] | remove PROCESS]\nOmit the action to open the interactive registry. Default group: 4.")
			f.PrintDefaults()
		}
	} else {
		f.StringVar(&o.Group, "g", "", "process group within the selected package")
		f.BoolVar(&o.All, "a", false, "all non-OPM processes in the selected package")
	}
	if err := f.Parse(args); err != nil {
		return o, err
	}
	if command == "roproc" && len(f.Args()) > 0 {
		args := f.Args()
		if len(args) < 2 || len(args) > 3 || (args[0] != "add" && args[0] != "remove") || (args[0] == "remove" && len(args) != 2) {
			return o, fmt.Errorf("use add PROCESS [GROUP] or remove PROCESS")
		}
		o.Action = args[0]
		o.Process = args[1]
		o.RegistryGroup = "4"
		if len(args) == 3 {
			o.RegistryGroup = args[2]
		}
	} else if command == "rolog" && len(f.Args()) > 0 {
		args := f.Args()
		if len(args) != 2 {
			return o, fmt.Errorf("use PROCESS LEVEL to change a log level")
		}
		level, ok := normalizeLevel(args[1])
		if !ok {
			return o, fmt.Errorf("level must be one of error, warn, info, debug, trace")
		}
		// A direct change needs no interactive screen.
		o.Process, o.Level, o.Plain = args[0], level, true
	} else if command != "rocron" && len(f.Args()) == 1 {
		o.Process = f.Args()[0]
	} else if len(f.Args()) > 0 {
		return o, fmt.Errorf("unexpected arguments: %v", f.Args())
	}
	if o.TUI && !term.IsTerminal(int(os.Stdin.Fd())) {
		return o, fmt.Errorf("--tui requires a terminal")
	}
	if o.TUI && (o.Plain || o.JSON) {
		return o, fmt.Errorf("--tui conflicts with --plain/--json")
	}
	if command == "rolog" && (o.RequestID != "" || o.WatchID != "") {
		return o, fmt.Errorf("rolog applies levels directly; operation IDs belong to control commands")
	}
	if (command == "rodis" || command == "ropack") && (o.RequestID != "" || o.WatchID != "" || o.Process != "") {
		return o, fmt.Errorf("%s is read-only; operation IDs belong to control commands", command)
	}
	if o.Sort != "" && o.Sort != "name" && o.Sort != "index" {
		return o, fmt.Errorf("-s must be name or index")
	}
	if o.WatchID != "" && (o.Action != "" || o.Process != "" || o.Group != "" || o.All || o.Job != "" || o.RequestID != "") {
		return o, fmt.Errorf("--watch cannot be combined with a new operation")
	}
	return o, nil
}
func restoreLegacyArgs(o Options) {
	args := []string{os.Args[0]}
	if o.Package != "" {
		args = append(args, "-p", o.Package)
	}
	if o.Debug {
		args = append(args, "-d")
	}
	if o.Listing {
		args = append(args, "-l")
	}
	if o.Command == "rolog" {
		if o.Process != "" {
			args = append(args, o.Process, o.Level)
		}
		os.Args = args
		return
	}
	if o.Command == "rodis" && o.Sort != "" {
		args = append(args, "-s", o.Sort)
	}
	if o.Command != "rocron" {
		if o.Group != "" {
			args = append(args, "-g", o.Group)
		}
		if o.All {
			args = append(args, "-a")
		}
		if o.Command == "roproc" && o.Action != "" {
			args = append(args, o.Action, o.Process)
			if o.Action == "add" {
				args = append(args, o.RegistryGroup)
			}
		} else if o.Process != "" {
			args = append(args, o.Process)
		}
	}
	os.Args = args
}
func runPlain(ctx context.Context, c *Client, o Options) error {
	if o.Command == "rolog" {
		return runLogLevelPlain(ctx, c, o)
	}
	if o.Command == "ropack" {
		q, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		v, err := c.Packages(q)
		if err != nil {
			return err
		}
		v.Packages = filteredPackages(v, o, "")
		if o.JSON {
			return json.NewEncoder(os.Stdout).Encode(v)
		}
		return printPackages(v)
	}
	if o.Command == "rodis" {
		q, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		v, err := c.Processes(q)
		if err != nil {
			return err
		}
		if o.JSON {
			return json.NewEncoder(os.Stdout).Encode(v)
		}
		return printProcesses(v, o.Sort, c.Config.Timezone)
	}
	if o.WatchID != "" {
		op, err := c.Get(ctx, o.Command, o.WatchID)
		if err != nil {
			return err
		}
		return outputOperation(op, o.JSON)
	}
	if o.Command == "roproc" && o.Action == "" {
		q, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		catalog, err := c.Registry(q)
		if err != nil {
			return err
		}
		if o.JSON {
			return json.NewEncoder(os.Stdout).Encode(catalog)
		}
		for _, g := range catalog.Groups {
			fmt.Printf("Group %s: %s\n", strconv.Itoa(int(g.Id)), g.Name)
		}
		return printProcesses(catalog.Catalog, "name", c.Config.Timezone)
	}
	if o.Command != "rocron" && o.Command != "roproc" {
		q, cancel := context.WithTimeout(ctx, 10*time.Second)
		catalog, err := c.Processes(q)
		cancel()
		if err != nil {
			return err
		}
		o.Targets, err = selectProcesses(catalog, o)
		if err != nil {
			return err
		}
		if len(o.Targets) == 0 {
			return fmt.Errorf("select a process, -g process-group or -a; omit --plain in a terminal to choose interactively")
		}
	}
	if o.Command == "rocron" && o.Job == "" {
		q, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		catalog, err := c.Cron(q)
		if err != nil {
			return err
		}
		if o.JSON {
			return json.NewEncoder(os.Stdout).Encode(catalog)
		}
		fmt.Printf("[%s] %s\n", c.Name, c.Target.PackageId)
		if o.Listing {
			for _, h := range catalog.Hours {
				fmt.Printf("\nHOUR %d\n", h.Hour)
				for _, j := range h.Jobs {
					fmt.Printf(" - %s / %s : %s [%s]\n", j.Process, j.Name, j.Spec, j.Description)
				}
			}
			return nil
		}
		for _, j := range catalog.Jobs {
			fmt.Printf("%-20s %-28s %-22s %s\n", j.Process, j.Name, j.Spec, j.Description)
		}
		return nil
	}
	if o.Command == "rocron" && o.Process == "" {
		return fmt.Errorf("--process is required with --job")
	}
	if o.RequestID == "" {
		o.RequestID = transport.ID("c_")
	}
	if o.Command == "roproc" {
		q, cancel := context.WithTimeout(ctx, 10*time.Second)
		var err error
		o.Plan, err = c.PreviewRegistry(q, o)
		cancel()
		if err != nil {
			return err
		}
	}
	fmt.Fprintln(os.Stderr, "request ID:", o.RequestID)
	q, cancel := context.WithTimeout(ctx, 10*time.Second)
	op, err := c.Submit(q, o)
	cancel()
	if err != nil {
		return fmt.Errorf("request outcome may be unknown; inspect --watch %s before a new request: %w", o.RequestID, err)
	}
	if !operations.Terminal(op.State) {
		stream, err := c.Watch(ctx, o.Command, op.Id)
		if err != nil {
			return err
		}
		for !operations.Terminal(op.State) {
			op, err = stream.Recv()
			if err != nil {
				return err
			}
		}
	}
	return outputOperation(op, o.JSON)
}
func outputOperation(op *api.ControlOperation, asJSON bool) error {
	if asJSON {
		if err := json.NewEncoder(os.Stdout).Encode(op); err != nil {
			return err
		}
	} else {
		fmt.Printf("%s %s\n%s\n", op.Id, op.State, op.Message)
		if op.Kind == "rocron" {
			fmt.Println(CronNotice)
		}
	}
	if op.State == "FAILED" || op.State == "INTERRUPTED" {
		return fmt.Errorf("%s: %s", op.State, op.Message)
	}
	return nil
}
