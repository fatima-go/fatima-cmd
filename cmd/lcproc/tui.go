package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-cmd/share"
)

type localDone struct{ err error }

// localProc is one row of the process sheet.
type localProc struct {
	name    string
	pid     int
	current Revision
	count   int
	err     error
}

type localModel struct {
	home, name, stage, target, current, notice string
	procs                                      []localProc
	revisions                                  []Revision
	cursor, offset, width, height              int
	chosen                                     Revision
	action                                     string
	busy, detail                               bool
	err                                        error
}

var localActions = [][2]string{
	{"리비전 조회 / 전환", "배포된 리비전을 확인하고 다른 리비전으로 되돌립니다"},
	{"다른 이름으로 복제", "실행 파일·설정 파일을 새 이름의 프로세스로 복사합니다"},
}

func loadProcs(home string) ([]localProc, error) {
	names, err := localProcesses(home)
	if err != nil {
		return nil, err
	}
	procs := make([]localProc, 0, len(names))
	for _, name := range names {
		p := localProc{name: name}
		p.pid, _ = runningPid(home, name)
		revisions, current, err := localRevisions(home, name)
		p.err, p.count = err, len(revisions)
		for _, r := range revisions {
			if same, _ := sameRevision(current, r.dir); same {
				p.current = r
				break
			}
		}
		procs = append(procs, p)
	}
	return procs, nil
}

func newLocalModel(args []string) localModel {
	m := localModel{home: os.Getenv(share.EnvFatimaHome), stage: "process", width: 80, height: 24}
	m.procs, m.err = loadProcs(m.home)
	if m.err == nil && len(args) > 0 {
		if m.procIndex(args[0]) < 0 {
			m.err = fmt.Errorf("프로세스 %s를 찾을 수 없습니다. 목록에서 선택하세요", args[0])
			return m
		}
		m.name = args[0]
		m.stage = "action"
		if len(args) > 1 {
			switch args[1] {
			case "version":
				m.openVersions()
				if len(args) > 2 && m.err == nil {
					r, err := findRevision(m.revisions, args[2])
					if err != nil {
						m.err = err
					} else {
						m.cursor = m.revisionIndex(r)
						m.selectRevision(r)
					}
				}
			case "dup":
				m.stage = "name"
				m.action = "dup"
				if len(args) > 2 {
					m.target = args[2]
				}
			default:
				m.err = fmt.Errorf("작업을 목록에서 선택하세요")
			}
		}
	}
	return m
}
func (m localModel) Init() tea.Cmd { return nil }

func (m localModel) procIndex(name string) int {
	for i, p := range m.procs {
		if p.name == name {
			return i
		}
	}
	return -1
}
func (m localModel) revisionIndex(r Revision) int {
	for i, v := range m.revisions {
		if v.dir == r.dir {
			return i
		}
	}
	return 0
}
func (m localModel) selectedProc() (localProc, bool) {
	if m.stage == "process" {
		if len(m.procs) == 0 {
			return localProc{}, false
		}
		return m.procs[min(m.cursor, len(m.procs)-1)], true
	}
	if i := m.procIndex(m.name); i >= 0 {
		return m.procs[i], true
	}
	return localProc{}, false
}
func (m localModel) isCurrent(r Revision) bool {
	same, _ := sameRevision(m.current, r.dir)
	return same
}
func (m localModel) currentRevision() (Revision, bool) {
	for _, r := range m.revisions {
		if m.isCurrent(r) {
			return r, true
		}
	}
	return Revision{}, false
}
func (m localModel) rowCount() int {
	switch m.stage {
	case "process":
		return len(m.procs)
	case "action":
		return len(localActions)
	case "version":
		return len(m.revisions)
	}
	return 0
}

func (m *localModel) openVersions() {
	m.notice, m.detail, m.offset = "", false, 0
	m.revisions, m.current, m.err = localRevisions(m.home, m.name)
	m.stage = "version"
	m.cursor = 0
}
func (m *localModel) reloadProcs() {
	m.procs, m.err = loadProcs(m.home)
	m.cursor = max(0, m.procIndex(m.name))
}

func (m localModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
		m.offset = min(m.offset, m.detailScrollLimit())
	case localDone:
		m.busy = false
		m.err = v.err
		if errors.Is(v.err, errCurrentRevision) {
			m.openVersions()
			m.notice = errCurrentRevision.Error() + ". 다른 리비전을 선택하세요."
			return m, nil
		}
		if v.err == nil {
			m.stage, m.offset = "result", 0
			if m.action == "version" {
				m.notice = "리비전 전환 완료 · rostart로 프로세스를 시작하세요"
			} else {
				m.notice = "복제 완료 · 아직 프로세스 등록과 시작은 하지 않았습니다. roproc add로 등록하세요"
			}
		}
	case tea.KeyMsg:
		return m.key(v)
	}
	return m, nil
}

func (m localModel) key(v tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := v.String()
	if m.busy {
		return m, nil
	}
	if key == "ctrl+c" || (key == "q" && m.stage != "name") {
		return m, tea.Quit
	}
	if key == "esc" {
		return m.back()
	}
	if m.stage == "name" {
		switch key {
		case "enter":
			if err := validLocalName(m.target); err != nil {
				m.err = err
			} else {
				m.err = nil
				m.stage = "confirm"
			}
		case "backspace":
			r := []rune(m.target)
			if len(r) > 0 {
				m.target = string(r[:len(r)-1])
			}
		default:
			if v.Type == tea.KeyRunes && len(m.target) < 128 {
				m.target += string(v.Runes)
			}
		}
		return m, nil
	}
	switch key {
	case "tab":
		if m.stage == "process" || m.stage == "version" {
			m.detail, m.offset = !m.detail, 0
		}
	case "up", "k":
		m.scroll(-1)
	case "down", "j":
		m.scroll(1)
	case "pgup":
		m.scroll(-m.pageSize())
	case "pgdown":
		m.scroll(m.pageSize())
	case "r":
		m.err, m.notice = nil, ""
		switch m.stage {
		case "process", "action":
			m.reloadProcs()
			if m.stage == "action" {
				m.cursor = 0
			}
		case "version":
			cursor := m.cursor
			m.openVersions()
			m.cursor = min(cursor, max(0, len(m.revisions)-1))
		}
	case "enter":
		switch m.stage {
		case "process":
			if len(m.procs) > 0 {
				m.name = m.procs[m.cursor].name
				m.stage, m.cursor, m.offset, m.detail, m.err = "action", 0, 0, false, nil
			}
		case "action":
			m.err = nil
			if m.cursor == 0 {
				m.openVersions()
			} else {
				m.stage, m.action, m.target = "name", "dup", ""
			}
		case "version":
			if len(m.revisions) > 0 {
				m.selectRevision(m.revisions[m.cursor])
			}
		case "confirm":
			m.busy = true
			m.err = nil
			return m, func() tea.Msg {
				if m.action == "version" {
					return localDone{switchLocalRevision(m.home, m.name, m.chosen.dir)}
				}
				return localDone{duplicateLocal(m.home, m.name, m.target)}
			}
		case "result":
			m.notice, m.stage = "", "process"
			m.reloadProcs()
		}
	}
	return m, nil
}

// scroll moves the list cursor, or the detail rows while the detail has focus.
func (m *localModel) scroll(delta int) {
	if m.detail {
		m.offset = max(0, min(m.offset+delta, m.detailScrollLimit()))
		return
	}
	m.cursor = max(0, min(m.cursor+delta, m.rowCount()-1))
	m.offset = 0
}

func (m localModel) back() (tea.Model, tea.Cmd) {
	m.err, m.notice, m.detail, m.offset = nil, "", false, 0
	switch m.stage {
	case "process":
		return m, tea.Quit
	case "action":
		m.stage = "process"
		m.cursor = max(0, m.procIndex(m.name))
	case "version":
		m.stage, m.cursor = "action", 0
	case "name":
		m.stage, m.cursor = "action", 1
	case "confirm":
		if m.action == "version" {
			m.stage, m.cursor = "version", m.revisionIndex(m.chosen)
		} else {
			m.stage = "name"
		}
	case "result":
		m.stage = "process"
		m.reloadProcs()
	}
	return m, nil
}

func (m *localModel) selectRevision(r Revision) {
	m.notice, m.err = "", nil
	// Re-read the link: another command may have switched it since listing.
	_, current, err := localRevisions(m.home, m.name)
	if err != nil {
		m.err = err
		return
	}
	m.current = current
	same, err := sameRevision(current, r.dir)
	if err != nil {
		m.err = err
		return
	}
	if same {
		m.stage = "version"
		m.notice = fmt.Sprintf("이미 %s 리비전을 가리키고 있습니다. 변경이 필요하지 않습니다.", r.revision)
		return
	}
	m.chosen, m.action, m.stage = r, "version", "confirm"
}

// ---- rendering: the same frame as the ro* commands (controlui).

func clean(s string) string {
	s = ansi.Strip(s)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, s)
}
func line(s string, width int) string {
	return ansi.Truncate(strings.ReplaceAll(clean(s), "\n", " "), max(1, width), "…")
}

const (
	railWidth    = 16
	railMinWidth = 128
	splitWidth   = 92
	stackHeight  = 23
)

// banners are the lines between the navigation and the sheets.
func (m localModel) banners() []string {
	var out []string
	if m.err != nil {
		out = append(out, lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("오류: "+clean(m.err.Error())))
	}
	if m.notice != "" && m.stage != "result" {
		out = append(out, lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("안내: "+clean(m.notice)))
	}
	if m.stage == "name" {
		out = append(out, "새 프로세스 이름> "+clean(m.target)+"█")
	}
	return out
}

// paneSize is the area left for sheets: the header box, navigation and footer
// take six rows, and the step rail takes a column on wide terminals.
func (m localModel) paneSize() (width, height int) {
	width, height = m.width, m.height-6-len(m.banners())
	if m.width >= railMinWidth {
		width -= railWidth + 3
	}
	return width, max(1, height)
}

// detailSize mirrors splitPanes for the detail sheet of the current stage.
func (m localModel) detailSize() (width, height int) {
	width, height = m.paneSize()
	if width >= splitWidth {
		return width * 45 / 100, height
	}
	if height >= stackHeight {
		return width, height - height/2 - 1
	}
	return width, height
}

func (m localModel) detailFields() (string, [][2]string) {
	switch m.stage {
	case "process", "action":
		if p, ok := m.selectedProc(); ok {
			return "프로세스 상세", m.procFields(p)
		}
		return "프로세스 상세", [][2]string{{"조회 결과", "표시할 프로세스가 없습니다."}}
	case "version":
		if len(m.revisions) == 0 {
			return "리비전 상세", [][2]string{{"조회 결과", "인식할 수 있는 리비전이 없습니다."}}
		}
		return "리비전 상세", m.revisionFields(m.revisions[min(m.cursor, len(m.revisions)-1)])
	}
	return "", nil
}

func (m localModel) detailScrollLimit() int {
	_, fields := m.detailFields()
	width, height := m.detailSize()
	valueWidth := width - 7 - min(12, max(8, (width-7)/3))
	if valueWidth < 1 {
		return 0
	}
	lines := 0
	for _, field := range fields {
		value := strings.ReplaceAll(clean(field[1]), "\t", " ")
		if value == "" {
			value = "-"
		}
		lines += len(strings.Split(ansi.Hardwrap(value, valueWidth, true), "\n"))
	}
	return max(0, lines-max(1, height-6))
}

func (m localModel) pageSize() int {
	width, height := m.paneSize()
	if width < splitWidth && height >= stackHeight {
		height /= 2
	}
	return max(1, height-6)
}

func (m localModel) procFields(p localProc) [][2]string {
	state := "STOPPED"
	if p.pid > 0 {
		state = fmt.Sprintf("ALIVE (PID %d)", p.pid)
	}
	fields := [][2]string{{"프로세스", p.name}, {"상태", state}}
	if p.err != nil {
		return append(fields, [2]string{"리비전", p.err.Error()})
	}
	fields = append(fields, [2]string{"보관 리비전", fmt.Sprintf("%d개", p.count)})
	if p.current.dir == "" {
		return append(fields, [2]string{"현재 리비전", "링크 대상이 인식할 수 있는 리비전이 아닙니다"})
	}
	return append(fields, buildFields(p.current, "현재 리비전")...)
}

func (m localModel) revisionFields(r Revision) [][2]string {
	fields := buildFields(r, "리비전")
	state := "-"
	if m.isCurrent(r) {
		state = "현재 링크 대상"
	}
	return append([][2]string{fields[0], {"상태", state}}, fields[1:]...)
}

func buildFields(r Revision, title string) [][2]string {
	fields := [][2]string{
		{title, r.revision}, {"유형", r.kind}, {"배포 시각", r.createDtime},
		{"빌드 사용자", r.deployment.BuildUser}, {"빌드 시각", r.deployment.BuildTime},
		{"브랜치", r.deployment.Git.Branch}, {"커밋", r.deployment.Git.Commit},
		{"메시지", trimString(r.deployment.Git.Message)},
	}
	if r.kind == revisionKindV2 {
		fields = append(fields, [2]string{"Operation", r.operationID}, [2]string{"SHA256", r.sha256})
	}
	return append(fields, [2]string{"폴더", filepath.Join("app", "revision", filepath.Base(filepath.Dir(r.dir)), r.name)})
}

func markCursor(i, cursor int, s string) string {
	if i == cursor {
		return "▶ " + s
	}
	return "  " + s
}

func (m localModel) processSheet(width, height int) string {
	inner := width - 13 // four columns with cell padding and borders
	revWidth, countWidth, stateWidth := min(14, max(8, inner/4)), 4, 7
	columns := []share.SheetColumn{{Name: "PROCESS", Width: inner - revWidth - countWidth - stateWidth}, {Name: "REVISION", Width: revWidth}, {Name: "REVS", Width: countWidth}, {Name: "STATE", Width: stateWidth}}
	var rows [][]string
	alive := 0
	for i, p := range m.procs {
		state := "STOPPED"
		if p.pid > 0 {
			state = "ALIVE"
			alive++
		}
		rev := p.current.revision
		if rev == "" {
			rev = "-"
		}
		rows = append(rows, []string{markCursor(i, m.cursor, p.name), rev, fmt.Sprint(p.count), state})
	}
	footer := fmt.Sprintf("실행 중 %d개 · Enter 선택", alive)
	return share.SheetView("로컬 프로세스", columns, rows, m.cursor, width, height, !m.detail, footer)
}

func (m localModel) actionSheet(width, height int) string {
	inner := width - 7 // two columns with cell padding and borders
	nameWidth := min(20, inner/2)
	columns := []share.SheetColumn{{Name: "작업", Width: nameWidth}, {Name: "설명", Width: inner - nameWidth}}
	var rows [][]string
	for i, a := range localActions {
		rows = append(rows, []string{markCursor(i, m.cursor, a[0]), a[1]})
	}
	return share.SheetView(m.name+" 작업", columns, rows, m.cursor, width, height, true, "Enter 선택")
}

func (m localModel) revisionSheet(width, height int) string {
	inner := width - 13 // four columns with cell padding and borders
	timeWidth, kindWidth, stateWidth := 16, 4, 4
	columns := []share.SheetColumn{{Name: "REVISION", Width: inner - timeWidth - kindWidth - stateWidth}, {Name: "DEPLOYED", Width: timeWidth}, {Name: "TYPE", Width: kindWidth}, {Name: "현재", Width: stateWidth}}
	var rows [][]string
	for i, r := range m.revisions {
		kind := r.kind
		if kind == revisionKindPrevious {
			kind = "이전"
		}
		state := ""
		if m.isCurrent(r) {
			state = "●"
		}
		deployed := r.createDtime
		if len(deployed) > timeWidth {
			deployed = deployed[:timeWidth] // drop seconds
		}
		rows = append(rows, []string{markCursor(i, m.cursor, r.revision), deployed, kind, state})
	}
	return share.SheetView(m.name+" 리비전 · 최신순", columns, rows, m.cursor, width, height, !m.detail, "Enter 전환")
}

// splitPanes lays the list and its detail side by side, stacked, or one at a
// time with Tab on small terminals.
func (m localModel) splitPanes(width, height int, list func(w, h int) string) string {
	title, fields := m.detailFields()
	if width >= splitWidth {
		dw := width * 45 / 100
		return lipgloss.JoinHorizontal(lipgloss.Top, list(width-dw-1, height), " ", share.PropertySheet(title, fields, dw, height, m.offset, m.detail, true))
	}
	if height >= stackHeight {
		lh := height / 2
		return list(width, lh) + "\n\n" + share.PropertySheet(title, fields, width, height-lh-1, m.offset, m.detail, true)
	}
	if m.detail {
		return share.PropertySheet(title, fields, width, height, m.offset, true, true)
	}
	return list(width, height)
}

func (m localModel) confirmFields() [][2]string {
	if m.action == "version" {
		current := "-"
		if r, ok := m.currentRevision(); ok {
			current = r.revision + " (" + r.kind + ")"
		}
		return [][2]string{
			{"작업", "리비전 전환"}, {"프로세스", m.name},
			{"현재", current}, {"전환 대상", m.chosen.revision + " (" + m.chosen.kind + ")"},
			{"배포 시각", m.chosen.createDtime}, {"커밋", m.chosen.deployment.Git.Commit},
			{"조건", "프로세스가 중지된 경우에만 링크를 전환합니다"},
			{"시작", "전환 후 자동으로 시작하지 않습니다. rostart로 시작하세요"},
		}
	}
	return [][2]string{
		{"작업", "프로세스 복제"}, {"원본", m.name}, {"새 이름", m.target},
		{"복사 범위", "실행 파일과 설정 파일 (하위 디렉터리는 복사하지 않음)"},
		{"이름 변경", "원본 이름으로 시작하는 파일명은 새 이름으로 변경합니다"},
		{"다음 단계", "복제 후 roproc add로 등록해야 사용할 수 있습니다"},
	}
}

func (m localModel) panes(width, height int) string {
	switch m.stage {
	case "process":
		return m.splitPanes(width, height, m.processSheet)
	case "action":
		return m.splitPanes(width, height, m.actionSheet)
	case "version":
		return m.splitPanes(width, height, m.revisionSheet)
	case "name":
		return share.PropertySheet("프로세스 복제", [][2]string{
			{"원본", m.name}, {"새 이름", m.target},
			{"이름 규칙", "영문·숫자·하이픈·밑줄 (운영 프로세스 이름 제외)"},
			{"복사 범위", "실행 파일과 설정 파일 (하위 디렉터리는 복사하지 않음)"},
		}, width, height, 0, true, false)
	case "confirm":
		title := "리비전 전환 확인"
		if m.action != "version" {
			title = "프로세스 복제 확인"
		}
		bar := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render(line("Enter 실행 · Esc 취소", width))
		return share.PropertySheet(title, m.confirmFields(), width, max(1, height-1), 0, true, false) + "\n" + bar
	case "result":
		fields := append([][2]string{{"결과", "✓ 완료"}}, m.confirmFields()[:2]...)
		return share.PropertySheet("결과", append(fields, [2]string{"안내", m.notice}), width, height, 0, true, false)
	}
	return ""
}

func (m localModel) steps() (stages, labels []string) {
	third, thirdStage := "리비전 선택", "version"
	if m.action == "dup" && m.stage != "version" {
		third, thirdStage = "이름 입력", "name"
	}
	return []string{"process", "action", thirdStage, "confirm", "result"}, []string{"프로세스 선택", "작업 선택", third, "실행 확인", "결과"}
}

func (m localModel) status() (string, lipgloss.Color) {
	switch {
	case m.busy:
		return "처리 중입니다", lipgloss.Color("214")
	case m.err != nil:
		return "오류를 확인하세요", lipgloss.Color("196")
	case m.stage == "result":
		return "완료", lipgloss.Color("42")
	case m.stage == "version":
		current := "-"
		if r, ok := m.currentRevision(); ok {
			current = r.revision
		}
		return fmt.Sprintf("%s · 리비전 %d개 · 현재 %s", m.name, len(m.revisions), current), lipgloss.Color("75")
	case m.stage == "confirm":
		return "내용을 확인한 뒤 Enter로 실행합니다", lipgloss.Color("214")
	}
	return fmt.Sprintf("로컬 프로세스 %d개", len(m.procs)), lipgloss.Color("75")
}

func (m localModel) help() string {
	if m.busy {
		return "처리 중입니다. 완료 결과를 기다려 주세요."
	}
	switch m.stage {
	case "process":
		return "q 종료  ↑↓ 이동  Enter 선택  Tab 목록/상세  r 새로고침"
	case "action":
		return "q 종료  ↑↓ 이동  Enter 선택  Esc 이전  r 새로고침"
	case "version":
		return "q 종료  ↑↓ 이동  Enter 전환  Tab 목록/상세  r 새로고침  Esc 이전"
	case "name":
		return "Enter 확인  Esc 취소  Ctrl+C 종료"
	case "confirm":
		return "Enter 실행  Esc 취소  q 종료"
	case "result":
		return "Enter 프로세스 목록  q 종료"
	}
	return "q 종료"
}

func (m localModel) View() string {
	if m.width < 60 || m.height < 19 {
		return "터미널을 60열 × 19행 이상으로 늘려 주세요. q 종료"
	}
	blue, grey := lipgloss.Color("75"), lipgloss.Color("240")
	box := func(s string, w, h int, border lipgloss.Color) string {
		return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Width(w).Height(h).MaxWidth(w + 2).MaxHeight(h + 2).Render(s)
	}
	home := m.home
	if home == "" {
		home = "미설정"
	}
	header := "lcproc  |  로컬 프로세스 관리  |  FATIMA_HOME " + home
	if m.name != "" && m.stage != "process" {
		header += "  |  " + m.name
	}
	top := box(line(header, m.width-4), m.width-2, 1, blue)

	stages, labels := m.steps()
	highlight := lipgloss.NewStyle().Bold(true).Foreground(blue)
	var nav, rail []string
	for i, s := range stages {
		if s == m.stage {
			nav = append(nav, highlight.Render(labels[i]))
			rail = append(rail, "▶ "+labels[i])
		} else {
			nav = append(nav, labels[i])
			rail = append(rail, "  "+labels[i])
		}
	}

	width, height := m.paneSize()
	body := strings.Join(append(m.banners(), m.panes(width, height)), "\n")
	if m.width >= railMinWidth {
		body = lipgloss.JoinHorizontal(lipgloss.Top, box(strings.Join(rail, "\n\n"), railWidth, m.height-8, grey), " ", body)
	}
	text, color := m.status()
	footer := []string{lipgloss.NewStyle().Foreground(color).Render(line("● "+text, m.width)), line(m.help(), m.width)}

	lines := strings.Split(top+"\n  "+strings.Join(nav, "  ›  ")+"\n"+body, "\n")
	lines = lines[:min(len(lines), m.height-2)]
	for len(lines) < m.height-2 {
		lines = append(lines, "")
	}
	lines = append(lines, footer...)
	for i, s := range lines {
		lines[i] = ansi.Truncate(s, m.width, "…")
	}
	return strings.Join(lines, "\n")
}
