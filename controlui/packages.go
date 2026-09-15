package controlui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-cmd/share"
	"github.com/fatima-go/fatima-opm/api"
	"google.golang.org/grpc"
)

func (c *Client) Packages(ctx context.Context) (*api.PackageCatalog, error) {
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	return api.NewPackageInventoryClient(c.gateway).List(ctx, &api.Empty{})
}
func (c *Client) WatchPackages(ctx context.Context) (grpc.ServerStreamingClient[api.PackageCatalog], error) {
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	return api.NewPackageInventoryClient(c.gateway).Watch(ctx, &api.Empty{})
}

type packageStream struct {
	stream grpc.ServerStreamingClient[api.PackageCatalog]
	epoch  int
	err    error
}
type packageSnapshot struct {
	catalog *api.PackageCatalog
	epoch   int
	err     error
}

func (m model) watchPackages() (model, tea.Cmd) {
	if m.liveCancel != nil {
		m.liveCancel()
	}
	ctx, cancel := context.WithCancel(m.ctx)
	m.liveCancel = cancel
	m.liveEpoch++
	epoch := m.liveEpoch
	return m, func() tea.Msg { s, err := m.client.WatchPackages(ctx); return packageStream{s, epoch, err} }
}
func receivePackages(s grpc.ServerStreamingClient[api.PackageCatalog], epoch int) tea.Cmd {
	return func() tea.Msg { v, err := s.Recv(); return packageSnapshot{v, epoch, err} }
}
func filteredPackages(c *api.PackageCatalog, o Options, filter string) []*api.PackageEntry {
	if c == nil {
		return nil
	}
	var out []*api.PackageEntry
	for _, p := range c.Packages {
		t := p.Target
		if o.Group != "" && !strings.EqualFold(t.Group, o.Group) {
			continue
		}
		id := o.Package
		if id != "" && !strings.Contains(id, ":") {
			id += ":default"
		}
		if id != "" && !strings.EqualFold(id, t.PackageId) {
			continue
		}
		if strings.Contains(strings.ToLower(t.Group+" "+t.PackageId+" "+t.Platform+" "+p.State), strings.ToLower(filter)) {
			out = append(out, p)
		}
	}
	return out
}
func (m model) visiblePackages() []*api.PackageEntry {
	opts := m.opts
	if m.stage == "package" && opts.Command != "ropack" {
		opts.Group = ""   // -g selects a process group, not a package group.
		opts.Package = "" // Reopening the picker must also show other packages.
	}
	return filteredPackages(m.packages, opts, m.filter)
}
func localDate(seconds int64) string {
	if seconds == 0 {
		return "-"
	}
	return time.Unix(seconds, 0).Local().Format("2006-01-02 15:04:05")
}

type processScreenClosed struct{ err error }

func (m model) openProcessScreen() tea.Cmd {
	// A child command uses the same installed CLI set. Refuse a changed context
	// before passing the selected package to it.
	cfg, err := config.GetActiveContext()
	if err != nil {
		return func() tea.Msg { return processScreenClosed{err} }
	}
	if cfg != m.client.Config {
		return func() tea.Msg {
			return processScreenClosed{fmt.Errorf("rocontext changed; reconnect before opening rodis")}
		}
	}
	binary, err := os.Executable()
	if err != nil {
		return func() tea.Msg { return processScreenClosed{err} }
	}
	id := m.visiblePackages()[m.cursor].Target.PackageId
	return tea.Exec(&processReport{cmd: exec.Command(filepath.Join(filepath.Dir(binary), "rodis"), "-p", id)}, func(err error) tea.Msg { return processScreenClosed{err} })
}

// rodis is a single HTTP report again. Keep the report visible while Bubble
// Tea has released the terminal, then resume the package screen on Enter.
type processReport struct {
	cmd    *exec.Cmd
	input  io.Reader
	output io.Writer
}

func (p *processReport) SetStdin(r io.Reader)  { p.input = r; p.cmd.Stdin = r }
func (p *processReport) SetStdout(w io.Writer) { p.output = w; p.cmd.Stdout = w }
func (p *processReport) SetStderr(w io.Writer) { p.cmd.Stderr = w }
func (p *processReport) Run() error {
	err := p.cmd.Run()
	if err != nil {
		fmt.Fprintln(p.output, "rodis:", err)
	}
	fmt.Fprint(p.output, "\nEnter: 패키지 목록으로 돌아가기 ")
	_, readErr := bufio.NewReader(p.input).ReadString('\n')
	if err != nil {
		return err
	}
	return readErr
}
func printPackages(c *api.PackageCatalog) error {
	var rows [][]string
	groups := map[string]bool{}
	hosts := map[string]bool{}
	for _, p := range c.Packages {
		t := p.Target
		groups[t.Group] = true
		hosts[strings.SplitN(t.PackageId, ":", 2)[0]] = true
		rows = append(rows, []string{t.Group, t.PackageId, t.Endpoint, t.Platform, p.State, p.Transport, localDate(p.RegisteredAt)})
	}
	share.PrintTable([]string{"group", "package", "endpoint", "platform", "status", "transport", "registered"}, rows)
	fmt.Printf("Total group:%d, host:%d, package:%d · Juno API availability (application health not checked)\n", len(groups), len(hosts), len(rows))
	return nil
}

func packageChoices(catalog *api.PackageCatalog) []share.PackageChoice {
	var choices []share.PackageChoice
	if catalog == nil {
		return choices
	}
	for _, p := range catalog.Packages {
		if p == nil || p.Target == nil {
			continue
		}
		choices = append(choices, share.PackageChoice{ID: p.Target.PackageId, Group: p.Target.Group, Endpoint: p.Target.Endpoint, State: p.State})
	}
	return choices
}
