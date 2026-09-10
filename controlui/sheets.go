package controlui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-core/opm/api"
)

var (
	sheetBorder   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sheetHeading  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75"))
	sheetSelected = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "18", Dark: "230"}).
			Background(lipgloss.AdaptiveColor{Light: "153", Dark: "24"})
	sheetAlternate = lipgloss.NewStyle().Background(lipgloss.AdaptiveColor{Light: "254", Dark: "235"})
)

type sheetColumn struct {
	name  string
	width int
}

func cell(value string, width int) string {
	value = line(strings.ReplaceAll(value, "\t", " "), width)
	return value + strings.Repeat(" ", max(0, width-ansi.StringWidth(value)))
}

// Grid cells use terminal display widths, so Korean names and ANSI input cannot
// move the next column. Only the selected row carries the cursor background.
func grid(columns []sheetColumn, rows [][]string, selected int) []string {
	rule := func(left, middle, right string) string {
		parts := make([]string, len(columns))
		for i, c := range columns {
			parts[i] = strings.Repeat("─", c.width+2)
		}
		return sheetBorder.Render(left + strings.Join(parts, middle) + right)
	}
	row := func(values []string, index int) string {
		parts := make([]string, len(columns))
		for i, c := range columns {
			value := ""
			if i < len(values) {
				value = values[i]
			}
			parts[i] = " " + cell(value, c.width) + " "
			if index >= 0 && index != selected {
				switch value {
				case "ALIVE", "SUCCEEDED":
					parts[i] = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(parts[i])
				case "DEAD", "FAILED", "UNREACHABLE", "MISMATCH":
					parts[i] = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(parts[i])
				case "UNKNOWN", "RUNNING", "WAITING":
					parts[i] = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(parts[i])
				}
			}
		}
		text := "│" + strings.Join(parts, "│") + "│"
		if index < 0 {
			return sheetHeading.Render(text)
		}
		if index == selected {
			return sheetSelected.Render(text)
		}
		if index%2 == 1 {
			return sheetAlternate.Render(text)
		}
		return text
	}
	headers := make([]string, len(columns))
	for i, c := range columns {
		headers[i] = c.name
	}
	out := []string{rule("┌", "┬", "┐"), row(headers, -1), rule("├", "┼", "┤")}
	for i, values := range rows {
		out = append(out, row(values, i))
	}
	return append(out, rule("└", "┴", "┘"))
}

func sheetTitle(title string, width int, focused bool) string {
	mark := "○ "
	if focused {
		mark = "● "
	}
	return sheetHeading.Render(cell(mark+title, width))
}

func sheetView(title string, columns []sheetColumn, rows [][]string, cursor, width, height int, focused bool, footer string) string {
	room := max(1, height-6) // title, grid header/borders, footer
	start := min(max(0, cursor-room+1), max(0, len(rows)-room))
	end := min(len(rows), start+room)
	window := append([][]string(nil), rows[start:end]...)
	for len(window) < room {
		window = append(window, nil)
	}
	selected := -1
	if cursor >= start && cursor < end {
		selected = cursor - start
	}
	out := []string{sheetTitle(title, width, focused)}
	out = append(out, grid(columns, window, selected)...)
	position := "0 / 0"
	if len(rows) > 0 {
		position = fmt.Sprintf("%d–%d / %d", start+1, end, len(rows))
	}
	return strings.Join(append(out, line(position+"  "+footer, width)), "\n")
}

func propertySheet(title string, fields [][2]string, width, height, offset int, focused bool) string {
	keyWidth := min(12, max(8, (width-7)/3))
	valueWidth := width - 7 - keyWidth
	columns := []sheetColumn{{"항목", keyWidth}, {"값", valueWidth}}
	var rows [][]string
	for _, field := range fields {
		value := strings.ReplaceAll(clean(field[1]), "\t", " ")
		if value == "" {
			value = "-"
		}
		for i, part := range strings.Split(ansi.Hardwrap(value, valueWidth, true), "\n") {
			key := ""
			if i == 0 {
				key = field[0]
			}
			rows = append(rows, []string{key, part})
		}
	}
	room := max(1, height-6)
	start := min(offset, max(0, len(rows)-room))
	end := min(len(rows), start+room)
	window := append([][]string(nil), rows[start:end]...)
	for len(window) < room {
		window = append(window, nil)
	}
	out := []string{sheetTitle(title, width, focused)}
	out = append(out, grid(columns, window, -1)...)
	foot := "Tab 목록 / 상세"
	if len(rows) > room {
		foot = fmt.Sprintf("%d–%d / %d  Tab 상세 · ↑↓ 스크롤", start+1, end, len(rows))
	}
	return strings.Join(append(out, line(foot, width)), "\n")
}

func processFields(p *api.ProcessEntry) [][2]string {
	return [][2]string{
		{"프로세스", p.Name}, {"그룹", p.Group}, {"상태", p.State},
		{"PID", p.Pid}, {"시작 시간", p.StartedAt}, {"CPU", p.Cpu},
		{"메모리", p.Memory}, {"FD", p.Fd}, {"스레드", p.Threads},
		{"IC", p.Ic}, {"등록 순서", fmt.Sprint(p.Index)},
	}
}

func packageFields(p *api.PackageEntry) [][2]string {
	return [][2]string{
		{"패키지", p.Target.PackageId}, {"그룹", p.Target.Group},
		{"주소", p.Target.Endpoint}, {"플랫폼", p.Target.Platform},
		{"상태", p.State}, {"통신 방식", p.Transport},
		{"등록 시간", localDate(p.RegisteredAt)}, {"확인 시간", localDate(p.CheckedAt)},
		{"상세", p.Detail}, {"상태 기준", "Juno API 접속 상태 (사용자 프로세스 상태는 rodis에서 확인)"},
	}
}

func (m model) inventoryLayout() bool {
	return m.client != nil && (m.stage == "select" || m.stage == "detail") && (m.catalog != nil || m.packages != nil)
}

func (m model) inventorySize() (width, height int) {
	width, height = m.width, m.height-7
	if m.width >= 128 {
		width -= 19 // existing navigation rail and its gap
	}
	if m.err != nil {
		height--
	}
	if m.editing != "" {
		height--
	}
	return
}

func (m model) listHeight(height, width int) int {
	if width < 92 && height >= 23 {
		return height / 2
	}
	return height
}

func (m model) pageSize() int {
	width, height := m.inventorySize()
	return max(1, m.listHeight(height, width)-6)
}

func (m model) detailScrollLimit() int {
	width, height := m.inventorySize()
	var fields [][2]string
	if rows := m.visiblePackages(); len(rows) > 0 {
		fields = packageFields(rows[min(m.cursor, len(rows)-1)])
	} else if rows := m.visibleProcesses(); len(rows) > 0 {
		fields = processFields(rows[min(m.cursor, len(rows)-1)])
		if width >= 92 {
			width = max(38, width*44/100)
		} else if height >= 23 {
			height -= m.listHeight(height, width) + 1
		}
	}
	valueWidth := width - 7 - min(12, max(8, (width-7)/3))
	lines := 0
	for _, field := range fields {
		value := strings.ReplaceAll(clean(field[1]), "\t", " ")
		lines += len(strings.Split(ansi.Hardwrap(value, valueWidth, true), "\n"))
	}
	return max(0, lines-max(1, height-6))
}

func (m model) inventoryView(width, height int) string {
	listWidth, detailWidth := width, width
	listHeight, detailHeight := m.listHeight(height, width), height
	split := width >= 92
	stack := !split && height >= 23
	if split {
		detailWidth = max(38, width*44/100)
		if m.opts.Command == "ropack" {
			detailWidth = width * 60 / 100
		}
		listWidth = width - detailWidth - 1
	} else if stack {
		detailHeight = height - listHeight - 1
	}
	var title string
	var rows [][]string
	var fields [][2]string
	footer := "↑↓ 이동 · PgUp/PgDn"
	if m.packages != nil {
		title = "패키지 목록"
		for i, p := range m.visiblePackages() {
			mark := "  "
			if i == m.cursor {
				mark = "▶ "
			}
			rows = append(rows, []string{mark + p.Target.PackageId, p.Target.Group, p.State})
		}
		if len(rows) > 0 {
			fields = packageFields(m.visiblePackages()[min(m.cursor, len(rows)-1)])
		}
	} else {
		title = "프로세스 목록"
		for i, p := range m.visibleProcesses() {
			mark := "  "
			if i == m.cursor {
				mark = "▶ "
			}
			if m.opts.Command == "rostart" || m.opts.Command == "rostop" {
				selected := "☐ "
				for _, name := range m.opts.Targets {
					if name == p.Name {
						selected = "☑ "
					}
				}
				mark += selected
			}
			rows = append(rows, []string{mark + p.Name, p.Group, p.State})
		}
		if len(rows) > 0 {
			fields = processFields(m.visibleProcesses()[min(m.cursor, len(rows)-1)])
		}
		if len(m.opts.Targets) > 0 {
			footer = fmt.Sprintf("선택 %d개 · Space 선택/해제", len(m.opts.Targets))
		}
	}
	if m.filter != "" {
		title += " · " + m.filter
	}
	if len(rows) == 0 {
		fields = [][2]string{{"조회 결과", "조건에 맞는 항목이 없습니다."}}
	}
	detailTitle := "프로세스 상세"
	nameHeader := "PROCESS"
	if m.packages != nil {
		detailTitle, nameHeader = "패키지 상세", "PACKAGE"
		// Enter opens a full-width sheet for long endpoints and diagnostics.
		if m.stage == "detail" {
			return propertySheet(detailTitle, fields, width, height, m.offset, true)
		}
	}
	inner := listWidth - 10 // three columns with cell padding and borders
	groupWidth := min(14, max(7, inner/4))
	stateWidth := min(11, max(7, inner/4))
	columns := []sheetColumn{{nameHeader, inner - groupWidth - stateWidth}, {"GROUP", groupWidth}, {"STATE", stateWidth}}
	list := sheetView(title, columns, rows, m.cursor, listWidth, listHeight, m.stage == "select", footer)
	detail := propertySheet(detailTitle, fields, detailWidth, detailHeight, m.offset, m.stage == "detail")
	if split {
		return lipgloss.JoinHorizontal(lipgloss.Top, list, " ", detail)
	}
	if stack {
		return list + "\n\n" + detail
	}
	if m.stage == "detail" {
		return detail
	}
	return list
}
