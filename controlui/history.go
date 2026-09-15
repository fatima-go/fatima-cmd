package controlui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatima-go/fatima-cmd/share"
	"github.com/fatima-go/fatima-opm/api"
)

func (c *Client) History(ctx context.Context, process string) (*api.HistoryList, error) {
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	return api.NewDeploymentHistoryClient(c.backend).List(ctx, &api.HistoryQuery{Process: process})
}

func deployedAt(ms int64, layout string) string {
	if ms <= 0 {
		return "-"
	}
	return time.UnixMilli(ms).Local().Format(layout)
}
func shortCommit(commit string) string {
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// filterHistory applies the -g selection of the original command; a process
// is narrowed by the server and -a keeps everything.
func filterHistory(records []*api.DeploymentRecord, o Options) []*api.DeploymentRecord {
	var out []*api.DeploymentRecord
	for _, r := range records {
		if o.Group == "" || strings.EqualFold(r.Group, o.Group) {
			out = append(out, r)
		}
	}
	return out
}

func runHistoryPlain(ctx context.Context, c *Client, o Options) error {
	q, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	v, err := c.History(q, o.Process)
	if err != nil {
		return err
	}
	v.Records = filterHistory(v.Records, o)
	if o.JSON {
		return json.NewEncoder(os.Stdout).Encode(v)
	}
	fmt.Printf("[%s] %s\n", c.Name, v.PackageId)
	var rows [][]string
	for _, r := range v.Records {
		rows = append(rows, []string{r.Process, deployedAt(r.DeployedAt, "2006-01-02 15:04:05"), r.BuildUser, r.GitBranch, shortCommit(r.GitCommit), firstLine(r.GitMessage)})
	}
	share.PrintTable([]string{"process", "deployed", "build user", "branch", "commit", "message"}, rows)
	fmt.Printf("total %d deployment records\n", len(rows))
	return nil
}

type historyProcess struct {
	name, group string
	records     []*api.DeploymentRecord
}

// historyProcesses groups records by process, newest record first.
func (m model) historyProcesses() []historyProcess {
	if m.history == nil {
		return nil
	}
	index := map[string]int{}
	var out []historyProcess
	for _, r := range m.history.Records {
		if !strings.Contains(strings.ToLower(r.Process+" "+r.Group), strings.ToLower(m.filter)) {
			continue
		}
		i, ok := index[r.Process]
		if !ok {
			i = len(out)
			index[r.Process] = i
			out = append(out, historyProcess{name: r.Process, group: r.Group})
		}
		out[i].records = append(out[i].records, r)
	}
	for i := range out {
		records := out[i].records
		sort.SliceStable(records, func(a, b int) bool { return records[a].DeployedAt > records[b].DeployedAt })
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}
func (m model) currentHistory() (historyProcess, bool) {
	rows := m.historyProcesses()
	if len(rows) == 0 {
		return historyProcess{}, false
	}
	return rows[min(m.cursor, len(rows)-1)], true
}

func (m model) loadedHistory() model {
	if m.history == nil {
		m.stage = "package"
		m.status = "패키지 선택 · Enter 진입"
		return m
	}
	m.stage, m.histCursor = "select", 0
	m.status = fmt.Sprintf("배포 이력 %d건 · Enter 이력 보기", len(m.history.Records))
	return m
}
func (m model) updateHistory(key string) (tea.Model, tea.Cmd) {
	switch m.stage {
	case "package":
		return m.choosePackage(key)
	case "records":
		p, _ := m.currentHistory()
		last := max(0, len(p.records)-1)
		switch key {
		case "up", "k":
			m.histCursor = max(0, m.histCursor-1)
		case "down", "j":
			m.histCursor = min(last, m.histCursor+1)
		case "home":
			m.histCursor = 0
		case "end":
			m.histCursor = last
		case "esc", "n":
			m.stage = "select"
			m.status = "프로세스 선택 · Enter 이력 보기"
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
			if p, ok := m.currentHistory(); ok && len(p.records) > 0 {
				m.stage, m.histCursor = "records", 0
				m.status = p.name + " 배포 이력 · Esc 목록"
			}
		default:
			return m.moveCursor(key), nil
		}
	}
	return m, nil
}

func (m model) historyView(width, height int) string {
	if m.stage == "package" {
		return m.packagePickerView(width, height)
	}
	title := "배포 이력 프로세스"
	if m.filter != "" {
		title += " · " + m.filter
	}
	var rows [][]string
	for i, p := range m.historyProcesses() {
		mark := "  "
		if i == m.cursor {
			mark = "▶ "
		}
		rows = append(rows, []string{mark + p.name, p.group, deployedAt(p.records[0].DeployedAt, "01-02 15:04")})
	}
	listWidth, recordWidth := width, width
	listHeight, recordHeight := height, height
	split := width >= 92
	stack := !split && height >= 23
	if split {
		recordWidth = width * 62 / 100
		listWidth = width - recordWidth - 1
	} else if stack {
		listHeight = height / 2
		recordHeight = height - listHeight - 1
	}
	list := sheetView(title, threeColumns("PROCESS", "LATEST", listWidth), rows, m.cursor, listWidth, listHeight, m.stage == "select", "Enter 이력 보기")
	records := m.recordsPane(recordWidth, recordHeight)
	if split {
		return lipgloss.JoinHorizontal(lipgloss.Top, list, " ", records)
	}
	if stack {
		return list + "\n\n" + records
	}
	if m.stage == "records" {
		return records
	}
	return list
}

// recordsPane lists the deployments of the selected process; while browsing
// them it adds the full detail, or shows only the detail when space is short.
func (m model) recordsPane(width, height int) string {
	p, ok := m.currentHistory()
	if !ok {
		return propertySheet("배포 이력", [][2]string{{"조회 결과", "배포 이력이 있는 프로세스가 없습니다."}}, width, height, 0, false, false)
	}
	title := fmt.Sprintf("배포 이력 · %s (%d)", p.name, len(p.records))
	inner := width - 13 // four columns with cell padding and borders
	timeWidth, commitWidth := 18, 8
	userWidth := min(12, max(6, inner/5))
	columns := []sheetColumn{{"DEPLOYED", timeWidth}, {"USER", userWidth}, {"BRANCH", max(4, inner-timeWidth-commitWidth-userWidth)}, {"COMMIT", commitWidth}}
	var rows [][]string
	for i, r := range p.records {
		mark := "  "
		if m.stage == "records" && i == m.histCursor {
			mark = "▶ "
		}
		rows = append(rows, []string{mark + deployedAt(r.DeployedAt, "2006-01-02 15:04"), r.BuildUser, r.GitBranch, shortCommit(r.GitCommit)})
	}
	if m.stage != "records" {
		return sheetView(title, columns, rows, 0, width, height, false, "Enter 이력 보기")
	}
	r := p.records[min(m.histCursor, len(p.records)-1)]
	fields := [][2]string{
		{"배포 시각", deployedAt(r.DeployedAt, "2006-01-02 15:04:05")}, {"빌드 사용자", r.BuildUser}, {"빌드 시각", r.BuildTime},
		{"브랜치", r.GitBranch}, {"커밋", r.GitCommit}, {"커밋 메시지", r.GitMessage},
	}
	if height < 15 { // both sheets need seven rows each
		return propertySheet(fmt.Sprintf("배포 상세 · %d/%d", m.histCursor+1, len(p.records)), fields, width, height, 0, true, false)
	}
	tableHeight := min(height/2, max(7, len(rows)+6))
	table := sheetView(title, columns, rows, m.histCursor, width, tableHeight, true, "↑↓ 이력 이동 · Esc 목록")
	return table + "\n\n" + propertySheet("배포 상세", fields, width, height-tableHeight-1, 0, false, false)
}
