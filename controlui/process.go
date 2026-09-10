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
func (m model) count() int {
	if m.packages != nil {
		return len(m.visiblePackages())
	}
	if m.catalog != nil {
		return len(m.visibleProcesses())
	}
	return len(m.jobs)
}
