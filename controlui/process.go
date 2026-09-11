package controlui

import (
	"fmt"
	"github.com/fatima-go/fatima-core/opm/api"
	"strings"
)

func selectProcesses(c *api.ProcessCatalog, o Options) ([]string, error) {
	count := 0
	if o.Process != "" {
		count++
	}
	if o.Group != "" {
		count++
	}
	if o.All {
		count++
	}
	if count > 1 {
		return nil, fmt.Errorf("choose one of process, -g or -a")
	}
	if strings.EqualFold(o.Group, "opm") {
		return nil, fmt.Errorf("OPM group operations are not permitted")
	}
	var names []string
	for _, p := range c.Processes {
		if (o.Process != "" && strings.EqualFold(p.Name, o.Process)) || (o.Group != "" && strings.EqualFold(p.Group, o.Group)) || (o.All && !p.Opm) {
			names = append(names, p.Name)
		}
	}
	if count > 0 && len(names) == 0 {
		return nil, fmt.Errorf("no processes match the selection")
	}
	return names, nil
}

// unselectable explains why rostart or rostop would do nothing for p, or be
// refused by Juno; such rows cannot be selected.
func (m model) unselectable(p *api.ProcessEntry) string {
	switch m.opts.Command {
	case "rostart":
		if p.State == "ALIVE" {
			return "이미 실행 중입니다"
		}
	case "rostop":
		if strings.EqualFold(p.Name, "jupiter") || strings.EqualFold(p.Name, "juno") {
			return "원격으로 중지할 수 없습니다"
		}
		if p.State == "DEAD" {
			return "이미 중지되어 있습니다"
		}
	}
	return ""
}
func (m model) count() int {
	if m.packages != nil {
		return len(m.visiblePackages())
	}
	if m.levels != nil {
		return len(m.visibleLogLevels())
	}
	if m.history != nil {
		return len(m.historyProcesses())
	}
	if m.catalog != nil {
		return len(m.visibleProcesses())
	}
	return len(m.jobs)
}
