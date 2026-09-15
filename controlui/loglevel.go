package controlui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatima-go/fatima-cmd/share"
	"github.com/fatima-go/fatima-opm/api"
)

// logLevels are the only levels rolog offers; Juno rejects any other value.
var logLevels = []string{"error", "warn", "info", "debug", "trace"}

func normalizeLevel(level string) (string, bool) {
	level = strings.ToLower(strings.TrimSpace(level))
	return level, slices.Contains(logLevels, level)
}
func levelLabel(level string) string {
	if level == "" {
		return "-"
	}
	return strings.ToUpper(level)
}

func (c *Client) LogLevels(ctx context.Context) (*api.LogLevelCatalog, error) {
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	return api.NewLogLevelControlClient(c.backend).List(ctx, &api.Empty{})
}
func (c *Client) SetLogLevel(ctx context.Context, process, level string) (*api.LogLevelEntry, error) {
	if err := c.checkTarget(ctx, "rolog"); err != nil {
		return nil, err
	}
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	return api.NewLogLevelControlClient(c.backend).Set(ctx, &api.LogLevelRequest{Process: process, Level: level})
}

func runLogLevelPlain(ctx context.Context, c *Client, o Options) error {
	q, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if o.Process != "" {
		e, err := c.SetLogLevel(q, o.Process, o.Level)
		if err != nil {
			return err
		}
		if o.JSON {
			return json.NewEncoder(os.Stdout).Encode(e)
		}
		fmt.Printf("[%s] %s\n%s -> %s (the process applies it within a second)\n", c.Name, c.Target.PackageId, e.Process, levelLabel(e.Level))
		return nil
	}
	v, err := c.LogLevels(q)
	if err != nil {
		return err
	}
	if o.JSON {
		return json.NewEncoder(os.Stdout).Encode(v)
	}
	fmt.Printf("[%s] %s\n", c.Name, v.PackageId)
	var rows [][]string
	for _, e := range sortedLogLevels(v, "") {
		rows = append(rows, []string{e.Process, e.Group, levelLabel(e.Level)})
	}
	share.PrintTable([]string{"process", "group", "loglevel"}, rows)
	return nil
}

func sortedLogLevels(c *api.LogLevelCatalog, filter string) []*api.LogLevelEntry {
	if c == nil {
		return nil
	}
	var out []*api.LogLevelEntry
	for _, e := range c.Entries {
		if strings.Contains(strings.ToLower(e.Process+" "+e.Group+" "+e.Level), strings.ToLower(filter)) {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Process < out[j].Process })
	return out
}
func (m model) visibleLogLevels() []*api.LogLevelEntry {
	return sortedLogLevels(m.levels, m.filter)
}

type logLevelApplied struct {
	entry *api.LogLevelEntry
	err   error
}

func (m model) setLogLevel(process, level string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		defer cancel()
		e, err := m.client.SetLogLevel(ctx, process, level)
		return logLevelApplied{e, err}
	}
}
func (m model) packageChoices() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		defer cancel()
		var c *api.PackageCatalog
		var err error
		if m.client.backend == nil {
			c, err = m.client.SelectionPackages(ctx)
		} else {
			c, err = m.client.Packages(ctx)
		}
		return loaded{packages: c, err: err}
	}
}

// loadedLogLevels opens the package list when routing could not pick one,
// otherwise the process list.
func (m model) loadedLogLevels() model {
	if m.levels == nil {
		m.stage = "package"
		m.status = "패키지 선택 · Enter 진입"
		return m
	}
	m.stage = "select"
	m.status = "프로세스 선택 · Enter 로그레벨 변경"
	return m
}
func (m model) appliedLogLevel(v logLevelApplied) (tea.Model, tea.Cmd) {
	m.busy = false
	m.stage = "select"
	if v.err != nil {
		m.err = v.err
		m.status = "변경 실패 · 다시 선택하거나 r 갱신"
		return m, nil
	}
	m.err = nil
	for _, e := range m.levels.Entries {
		if e.Process == v.entry.Process {
			e.Level = v.entry.Level
		}
	}
	m.status = fmt.Sprintf("%s → %s 반영 · 프로세스가 1초 이내 적용", v.entry.Process, levelLabel(v.entry.Level))
	return m, nil
}
func (m model) moveCursor(key string) model {
	last := max(0, m.count()-1)
	switch key {
	case "up", "k":
		m.cursor = max(0, m.cursor-1)
	case "down", "j":
		m.cursor = min(last, m.cursor+1)
	case "pgup":
		m.cursor = max(0, m.cursor-m.pageSize())
	case "pgdown":
		m.cursor = min(last, m.cursor+m.pageSize())
	case "home":
		m.cursor = 0
	case "end":
		m.cursor = last
	}
	return m
}
func (m model) updateLogLevel(key string) (tea.Model, tea.Cmd) {
	switch m.stage {
	case "package":
		return m.choosePackage(key)
	case "level":
		switch key {
		case "up", "k":
			m.levelCursor = max(0, m.levelCursor-1)
		case "down", "j":
			m.levelCursor = min(len(logLevels)-1, m.levelCursor+1)
		case "esc", "n":
			m.stage = "select"
			m.status = "프로세스 선택 · Enter 로그레벨 변경"
		case "enter":
			rows := m.visibleLogLevels()
			if len(rows) == 0 {
				m.stage = "select"
				return m, nil
			}
			p := rows[min(m.cursor, len(rows)-1)]
			m.busy = true
			m.status = p.Process + " 로그레벨 변경 중"
			return m, m.setLogLevel(p.Process, logLevels[m.levelCursor])
		}
	default:
		switch key {
		case "/":
			m.editing, m.input = "filter", m.filter
		case "p":
			m.busy = true
			m.status = "패키지 목록 조회 중"
			return m, m.packageChoices()
		case "enter":
			rows := m.visibleLogLevels()
			if len(rows) == 0 {
				return m, nil
			}
			m.levelCursor = max(0, slices.Index(logLevels, rows[min(m.cursor, len(rows)-1)].Level))
			m.stage, m.err = "level", nil
			m.status = "로그레벨 선택 · Enter 적용 / Esc 취소"
		default:
			return m.moveCursor(key), nil
		}
	}
	return m, nil
}

// choosePackage handles the package list shown when routing could not pick one
// (or after p). Enter reconnects to the chosen package.
func (m model) choosePackage(key string) (tea.Model, tea.Cmd) {
	if m.client == nil {
		return m, nil
	}
	switch key {
	case "/":
		m.editing, m.input = "filter", m.filter
	case "esc":
		if m.client.backend != nil {
			m.filter, m.busy = "", true
			return m, m.load()
		}
	case "enter":
		rows := m.visiblePackages()
		if len(rows) == 0 {
			return m, nil
		}
		m.opts.Package = rows[min(m.cursor, len(rows)-1)].Target.PackageId
		m.client.Close()
		m.client, m.packages, m.filter, m.cursor, m.err = nil, nil, "", 0, nil
		m.status = "접속 확인 중"
		m.busy = true
		return m, m.connect()
	default:
		return m.moveCursor(key), nil
	}
	return m, nil
}
func (m model) packagePickerView(width, height int) string {
	title := "패키지 선택"
	if m.filter != "" {
		title += " · " + m.filter
	}
	var rows [][]string
	for i, p := range m.visiblePackages() {
		mark := "  "
		if i == m.cursor {
			mark = "▶ "
		}
		rows = append(rows, []string{mark + p.Target.PackageId, p.Target.Group, p.State})
	}
	return sheetView(title, threeColumns("PACKAGE", "STATE", width), rows, m.cursor, width, height, true, "Enter 선택")
}

func threeColumns(name, last string, width int) []sheetColumn {
	inner := width - 10 // three columns with cell padding and borders
	groupWidth := min(14, max(7, inner/4))
	lastWidth := min(11, max(7, inner/4))
	return []sheetColumn{{name, inner - groupWidth - lastWidth}, {"GROUP", groupWidth}, {last, lastWidth}}
}
func (m model) logLevelView(width, height int) string {
	suffix := ""
	if m.filter != "" {
		suffix = " · " + m.filter
	}
	if m.stage == "package" {
		return m.packagePickerView(width, height)
	}
	entries := m.visibleLogLevels()
	var rows [][]string
	for i, e := range entries {
		mark := "  "
		if i == m.cursor {
			mark = "▶ "
		}
		rows = append(rows, []string{mark + e.Process, e.Group, levelLabel(e.Level)})
	}
	listWidth, pickWidth := width, width
	listHeight, pickHeight := height, height
	split := width >= 92
	stack := !split && height >= 23
	if split {
		pickWidth = max(38, width*40/100)
		listWidth = width - pickWidth - 1
	} else if stack {
		pickHeight = 11 // five levels plus the sheet frame
		listHeight = height - pickHeight - 1
	}
	list := sheetView("프로세스 로그레벨"+suffix, threeColumns("PROCESS", "LEVEL", listWidth), rows, m.cursor, listWidth, listHeight, m.stage == "select", "Enter 로그레벨 변경")
	picker := m.levelPicker(entries, pickWidth, pickHeight)
	if split {
		return lipgloss.JoinHorizontal(lipgloss.Top, list, " ", picker)
	}
	if stack {
		return list + "\n\n" + picker
	}
	if m.stage == "level" {
		return picker
	}
	return list
}
func (m model) levelPicker(entries []*api.LogLevelEntry, width, height int) string {
	title, current := "로그레벨", ""
	if len(entries) > 0 {
		e := entries[min(m.cursor, len(entries)-1)]
		title += " · " + e.Process
		current = e.Level
	}
	cursor := slices.Index(logLevels, current)
	footer := "Enter 변경"
	if m.stage == "level" {
		cursor = m.levelCursor
		footer = "Enter 적용 · Esc 취소"
	}
	var rows [][]string
	for i, l := range logLevels {
		mark, note := "  ", ""
		if m.stage == "level" && i == m.levelCursor {
			mark = "▶ "
		}
		if l == current {
			note = "현재"
		}
		rows = append(rows, []string{mark + strings.ToUpper(l), note})
	}
	inner := width - 7 // two columns with cell padding and borders
	noteWidth := min(8, max(4, inner/3))
	return sheetView(title, []sheetColumn{{"LEVEL", inner - noteWidth}, {"", noteWidth}}, rows, cursor, width, height, m.stage == "level", footer)
}
