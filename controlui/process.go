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
func (m model) processRows(height, width int) []string {
	processes := m.visibleProcesses()
	content := []string{"프로세스 선택 · Space 복수 선택"}
	if m.opts.Command == "rodis" {
		content = []string{"프로세스 상태 · / 검색 · o 정렬 · Enter 상세"}
	}
	if m.opts.Command == "roproc" {
		content = []string{"프로세스 등록부 · a 등록 / x 삭제 / Enter 상세"}
	}
	room := max(1, height/2-2)
	start := max(0, m.cursor-room+1)
	for i := start; i < min(len(processes), start+room); i++ {
		p := processes[i]
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}
		mark := "[ ]"
		if m.opts.Command == "rodis" || m.opts.Command == "roproc" {
			mark = ""
		}
		for _, n := range m.opts.Targets {
			if n == p.Name {
				mark = "[✓]"
			}
		}
		content = append(content, line(fmt.Sprintf("%s%s %-16s %s  PID %s", cursor, mark, p.Name, p.State, p.Pid), width))
	}
	if len(processes) > 0 {
		p := processes[min(m.cursor, len(processes)-1)]
		content = append(content, "──────── 상세 ────────")
		content = append(content, processDetail(p)...)
	}
	return content
}
