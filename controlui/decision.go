package controlui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// registryActions puts roproc's own keys above the list, where they are seen
// first. x names the row it would remove, or why it cannot.
func (m model) registryActions(width int) string {
	remove := badgeDanger.Render(" x ") + " 삭제 · "
	rows := m.visibleProcesses()
	switch {
	case len(rows) == 0:
		remove = badgeIdle.Render(" x ") + " 삭제할 프로세스 없음"
	case rows[min(m.cursor, len(rows)-1)].Opm:
		remove = badgeIdle.Render(" x ") + " " + rows[min(m.cursor, len(rows)-1)].Name + " 삭제 불가 (OPM)"
	default:
		remove += rows[min(m.cursor, len(rows)-1)].Name
	}
	return ansi.Truncate(badgeGo.Render(" a ")+" 프로세스 등록     "+remove, width, "…")
}

var (
	badgeDanger  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("160"))
	badgeGo      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("28"))
	badgeIdle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("240"))
	warningStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	promptStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
)

// destructive marks confirmations that delete data or stop running service.
func (m model) destructive() bool {
	return (m.opts.Command == "roproc" && m.opts.Action == "remove") || m.opts.Command == "rostop"
}

// decisionLabel names what Enter will do instead of a generic "execute".
func (m model) decisionLabel() string {
	subject := fmt.Sprintf("%d개 프로세스", len(m.opts.Targets))
	if len(m.opts.Targets) == 1 {
		subject = m.opts.Targets[0]
	}
	switch m.opts.Command {
	case "roproc":
		if m.opts.Action == "remove" {
			return m.opts.Process + " 영구 삭제"
		}
		return fmt.Sprintf("%s 등록 (그룹 %s)", m.opts.Process, m.opts.RegistryGroup)
	case "rostart":
		return subject + " 시작"
	case "rostop":
		return subject + " 중지"
	case "rocron":
		return m.opts.Process + " / " + m.opts.Job + " 실행 요청"
	}
	return "실행 요청"
}

// decisionBar is pinned below the scrolling review so the final keys stay in
// view however long the effects are. Removal waits for the typed process name.
func (m model) decisionBar(width int) []string {
	if m.stage != "review" || m.client == nil {
		return nil
	}
	enter := badgeGo
	if m.destructive() {
		enter = badgeDanger
	}
	out := []string{sheetBorder.Render(strings.Repeat("─", width))}
	if m.editing == "confirm" {
		out = append(out, promptStyle.Render(line(fmt.Sprintf("삭제하려면 %s 입력 › %s█", m.opts.Process, m.input), width)))
		if strings.TrimSpace(m.input) != m.opts.Process {
			enter = badgeIdle
		}
	}
	label := line(m.decisionLabel(), max(1, width-21)) // badges, gaps and "취소" take 21 cells
	return append(out, enter.Render(" Enter ")+" "+label+"   "+badgeIdle.Render(" Esc ")+" 취소")
}
