package controlui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-cmd/share"
	"github.com/fatima-go/fatima-core/opm/api"
	"google.golang.org/grpc"
	"sort"
	"strings"
	"time"
)

type statusStream struct {
	stream grpc.ServerStreamingClient[api.ProcessCatalog]
	epoch  int
	err    error
}
type statusSnapshot struct {
	catalog *api.ProcessCatalog
	epoch   int
	err     error
}

func (m model) watchStatus() (model, tea.Cmd) {
	if m.liveCancel != nil {
		m.liveCancel()
	}
	ctx, cancel := context.WithCancel(m.ctx)
	m.liveCancel = cancel
	m.liveEpoch++
	epoch := m.liveEpoch
	return m, func() tea.Msg { s, err := m.client.WatchProcesses(ctx); return statusStream{s, epoch, err} }
}
func receiveStatus(s grpc.ServerStreamingClient[api.ProcessCatalog], epoch int) tea.Cmd {
	return func() tea.Msg { v, err := s.Recv(); return statusSnapshot{v, epoch, err} }
}
func sortedProcesses(c *api.ProcessCatalog, order, filter string) []*api.ProcessEntry {
	if c == nil {
		return nil
	}
	out := make([]*api.ProcessEntry, 0, len(c.Processes))
	for _, p := range c.Processes {
		if strings.Contains(strings.ToLower(p.Name+" "+p.Group+" "+p.State), strings.ToLower(filter)) {
			out = append(out, p)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if order == "index" {
			return out[i].Index < out[j].Index
		}
		return out[i].Name < out[j].Name
	})
	return out
}
func (m model) visibleProcesses() []*api.ProcessEntry {
	return sortedProcesses(m.catalog, m.opts.Sort, m.filter)
}
func printProcesses(c *api.ProcessCatalog, order, timezone string) error {
	loc := time.Local
	if timezone != "" {
		var err error
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return err
		}
	}
	fmt.Printf("%s (%s)\n[%s] %s (%s)\n", time.Now().In(loc).Format("2006-01-02 15:04:05"), loc, c.Group, c.PackageId, c.Platform)
	var rows [][]string
	alive := 0
	for _, p := range sortedProcesses(c, order, "") {
		rows = append(rows, []string{p.Name, p.Pid, p.State, p.Cpu, p.Memory, p.Fd, p.Threads, p.StartedAt, p.Ic, p.Group})
		if p.State == "ALIVE" {
			alive++
		}
	}
	share.PrintTable([]string{"name", "pid", "status", "cpu", "mem", "fd", "thr", "start time", "ic", "group"}, rows)
	fmt.Printf("Total:%d (Alive:%d, Dead:%d), system is %s/%s\n", len(rows), alive, len(rows)-alive, share.AsHaString(int(c.HaStatus)), share.AsPsString(int(c.PsStatus)))
	return nil
}
