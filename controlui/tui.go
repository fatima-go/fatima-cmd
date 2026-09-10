package controlui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-core/opm/api"
	"github.com/fatima-go/fatima-core/opm/operations"
	"github.com/fatima-go/fatima-core/opm/transport"
	"google.golang.org/grpc"
	"io"
	"strings"
	"time"
	"unicode"
)

type connected struct {
	client *Client
	err    error
}
type loaded struct {
	registry *api.RegistryCatalog
	packages *api.PackageCatalog
	jobs     []*api.CronEntry
	catalog  *api.ProcessCatalog
	err      error
}
type operationMsg struct {
	op  *api.ControlOperation
	err error
}
type streamMsg struct {
	stream grpc.ServerStreamingClient[api.ControlOperation]
	err    error
}
type model struct {
	registry                      *api.RegistryCatalog
	packages                      *api.PackageCatalog
	packageReceiver               grpc.ServerStreamingClient[api.PackageCatalog]
	statusReceiver                grpc.ServerStreamingClient[api.ProcessCatalog]
	liveEpoch                     int
	liveCancel                    context.CancelFunc
	filter, returnCommand         string
	opts                          Options
	ctx                           context.Context
	client                        *Client
	fallback                      bool
	err                           error
	jobs                          []*api.CronEntry
	catalog                       *api.ProcessCatalog
	cursor, offset, width, height int
	stage, status, input, editing string
	busy                          bool
	op                            *api.ControlOperation
	stream                        grpc.ServerStreamingClient[api.ControlOperation]
}

func (m model) Init() tea.Cmd { return m.connect() }
func (m model) connect() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		defer cancel()
		c, err := connect(ctx, m.opts)
		return connected{c, err}
	}
}
func (m model) load() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		defer cancel()
		if m.opts.Command == "roproc" {
			c, err := m.client.Registry(ctx)
			return loaded{registry: c, err: err}
		}
		if m.opts.Command == "ropack" {
			c, err := m.client.Packages(ctx)
			return loaded{packages: c, err: err}
		}
		if m.opts.Command != "rocron" {
			c, err := m.client.Processes(ctx)
			return loaded{catalog: c, err: err}
		}
		v, err := m.client.Cron(ctx)
		if err != nil {
			return loaded{err: err}
		}
		return loaded{jobs: v.Jobs}
	}
}
func (m model) watch() tea.Cmd {
	return func() tea.Msg {
		s, err := m.client.Watch(m.ctx, m.opts.Command, m.opts.RequestID)
		return streamMsg{s, err}
	}
}
func receive(s grpc.ServerStreamingClient[api.ControlOperation]) tea.Cmd {
	return func() tea.Msg { o, err := s.Recv(); return operationMsg{o, err} }
}
func (m model) submit() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		defer cancel()
		o, err := m.client.Submit(ctx, m.opts)
		return operationMsg{o, err}
	}
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
	case connected:
		if legacy(v.err) {
			m.fallback = true
			return m, tea.Quit
		}
		if v.err != nil {
			m.err = v.err
			m.status = "접속 실패 · r 재시도"
			return m, nil
		}
		m.client = v.client
		m.err = nil
		m.status = "목록 조회 중"
		m.busy = true
		if m.opts.WatchID != "" {
			m.opts.RequestID = m.opts.WatchID
			m.stage = "result"
			return m, m.watch()
		}
		return m, m.load()
	case loaded:
		m.busy = false
		if v.err != nil {
			m.err = v.err
			m.status = "조회 실패 · r 재시도"
			return m, nil
		}
		m.err = nil
		m.jobs = v.jobs
		m.catalog = v.catalog
		m.registry = v.registry
		if v.registry != nil {
			m.catalog = v.registry.Catalog
		}
		m.packages = v.packages
		m.cursor = 0
		m.status = "작업 선택"
		if m.opts.Command == "roproc" {
			if m.opts.Action != "" {
				if m.opts.RequestID == "" {
					m.opts.RequestID = transport.ID("c_")
				}
				m.busy = true
				m.status = "변경 범위 확인 중"
				return m, m.previewRegistry()
			}
			return m, nil
		}
		if m.opts.Command == "ropack" {
			return m.watchPackages()
		}
		if m.opts.Command == "rodis" {
			m.status = "실시간 상태 구독"
			next, cmd := m.watchStatus()
			return next, cmd
		}
		if m.catalog != nil && (m.opts.Process != "" || m.opts.Group != "" || m.opts.All) {
			m.opts.Targets, m.err = selectProcesses(m.catalog, m.opts)
			if m.err == nil {
				m.stage = "review"
				if m.opts.RequestID == "" {
					m.opts.RequestID = transport.ID("c_")
				}
				m.status = "대상 확인 · Enter 실행"
			}
		}
		return m, nil
	case streamMsg:
		if v.err != nil {
			m.err = v.err
			m.busy = false
			m.status = "결과 확인 불가 · r 같은 요청 재조회"
			return m, nil
		}
		m.stream = v.stream
		return m, receive(v.stream)
	case registryPreview:
		m.busy = false
		m.err = v.err
		if v.err != nil {
			m.status = "미리보기 실패 · n 목록 / r 재조회"
			return m, nil
		}
		m.opts.Plan = v.plan
		m.opts.Process = v.plan.Request.Process
		m.opts.RegistryGroup = v.plan.Request.Group
		m.stage = "review"
		m.offset = 0
		m.status = "변경 범위 확인 · Enter 실행 요청"
		return m, nil
	case packageStream:
		if v.epoch != m.liveEpoch {
			return m, nil
		}
		if v.err != nil {
			m.err = v.err
			m.status = "패키지 구독 실패 · r 재연결"
			return m, nil
		}
		m.packageReceiver = v.stream
		return m, receivePackages(v.stream, v.epoch)
	case packageSnapshot:
		if v.epoch != m.liveEpoch {
			return m, nil
		}
		if v.err != nil {
			m.err = v.err
			m.status = "연결 끊김 · 마지막 조회 자료 · r 재연결"
			return m, nil
		}
		id := ""
		rows := m.visiblePackages()
		if m.cursor < len(rows) {
			id = rows[m.cursor].Target.PackageId
		}
		m.packages = v.catalog
		m.cursor = 0
		for i, p := range m.visiblePackages() {
			if p.Target.PackageId == id {
				m.cursor = i
			}
		}
		m.err = nil
		m.status = "패키지 상태 조회 " + time.Unix(v.catalog.ObservedAt, 0).Local().Format("15:04:05")
		return m, receivePackages(m.packageReceiver, v.epoch)
	case processScreenClosed:
		m.err = v.err
		if v.err != nil {
			m.status = "rodis 실행 실패 · r 목록 갱신"
			return m, nil
		}
		return m, m.load()
	case statusStream:
		if v.epoch != m.liveEpoch {
			return m, nil
		}
		if v.err != nil {
			m.err = v.err
			m.status = "상태 구독 실패 · r 재연결"
			return m, nil
		}
		m.statusReceiver = v.stream
		return m, receiveStatus(v.stream, v.epoch)
	case statusSnapshot:
		if v.epoch != m.liveEpoch || m.client == nil {
			return m, nil
		}
		if v.err != nil {
			m.err = v.err
			m.status = "연결 끊김 · 마지막 조회 자료 · r 재연결"
			return m, nil
		}
		name := ""
		rows := m.visibleProcesses()
		if len(rows) > 0 && m.cursor < len(rows) {
			name = rows[m.cursor].Name
		}
		m.catalog = v.catalog
		m.cursor = 0
		for i, p := range m.visibleProcesses() {
			if p.Name == name {
				m.cursor = i
			}
		}
		m.err = nil
		m.status = "실시간 조회 " + time.Unix(v.catalog.ObservedAt, 0).Local().Format("15:04:05")
		// Reopen the receive command on the same stream (stored below).
		return m, receiveStatus(m.statusReceiver, v.epoch)
	case operationMsg:
		if v.err != nil {
			if v.err == io.EOF && m.op != nil && operations.Terminal(m.op.State) {
				return m, nil
			}
			m.err = v.err
			m.busy = false
			m.status = "결과 확인 불가 · r 같은 요청 재조회"
			return m, nil
		}
		m.err = nil
		m.op = v.op
		m.stage = "result"
		m.status = v.op.State
		m.offset = 0
		if operations.Terminal(v.op.State) {
			m.busy = false
			return m, nil
		}
		m.busy = true
		if m.stream != nil {
			return m, receive(m.stream)
		}
		return m, m.watch()
	case tea.KeyMsg:
		key := v.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.width < 60 || m.height < 19 {
			if key == "q" {
				return m, tea.Quit
			}
			return m, nil
		}
		if m.editing != "" {
			switch key {
			case "enter":
				if m.editing == "process" {
					m.opts.Process = strings.TrimSpace(m.input)
					m.editing = "group"
					m.input = "4"
					return m, nil
				}
				if m.editing == "group" {
					m.opts.RegistryGroup = strings.TrimSpace(m.input)
					m.editing = ""
					m.opts.RequestID = transport.ID("c_")
					m.busy = true
					m.err = nil
					return m, m.previewRegistry()
				}
				if m.editing == "filter" {
					m.filter = m.input
					m.editing = ""
					m.cursor = 0
					return m, nil
				}
				if m.editing == "package" {
					if m.liveCancel != nil {
						m.liveCancel()
					}
					m.liveEpoch++
					m.opts.Targets = nil
					m.opts.Package = m.input
					m.editing = ""
					if m.client != nil {
						m.client.Close()
						m.client = nil
					}
					m.err = nil
					return m, m.connect()
				}
				m.opts.Arguments = m.input
				m.editing = ""
				m.stage = "review"
				m.opts.RequestID = transport.ID("c_")
				m.status = "내용 확인 / Enter 실행 요청"
			case "esc":
				if m.opts.Command == "roproc" && (m.editing == "process" || m.editing == "group") {
					m.opts.Action = ""
					m.opts.Process = ""
					m.opts.Plan = nil
					m.opts.RequestID = ""
				}
				m.editing = ""
				m.stage = "select"
			case "backspace":
				r := []rune(m.input)
				if len(r) > 0 {
					m.input = string(r[:len(r)-1])
				}
			case "ctrl+u":
				m.input = ""
			default:
				if v.Type == tea.KeyRunes && len(m.input) < 8192 {
					m.input += string(v.Runes)
				}
			}
			return m, nil
		}
		if key == "q" {
			return m, tea.Quit
		}
		if key == "d" {
			m.opts.Debug = !m.opts.Debug
			return m, nil
		}
		if key == "r" && !m.busy {
			if m.opts.Command == "ropack" && m.client != nil {
				m.err = nil
				return m.watchPackages()
			}
			if m.opts.Command == "rodis" && m.client != nil {
				m.err = nil
				next, cmd := m.watchStatus()
				return next, cmd
			}
			if m.stage == "result" && m.opts.RequestID != "" {
				m.err = nil
				return m, m.watch()
			}
			if m.client == nil {
				return m, m.connect()
			}
			m.err = nil
			m.busy = true
			return m, m.load()
		}
		if m.busy {
			if m.stage == "result" {
				if key == "up" {
					m.offset = max(0, m.offset-1)
				}
				if key == "down" {
					m.offset++
				}
			}
			return m, nil
		}
		switch key {
		case "a":
			if m.opts.Command == "roproc" && (m.stage == "select" || m.stage == "detail") && m.client != nil {
				m.opts.Action = "add"
				m.opts.Plan = nil
				m.opts.Process = ""
				m.editing = "process"
				m.input = ""
				m.stage = "arguments"
				m.offset = 0
				m.err = nil
			}
		case "/":
			if m.stage == "select" && (m.catalog != nil || m.packages != nil) {
				m.editing = "filter"
				m.input = m.filter
			}
		case "o":
			if m.catalog != nil {
				if m.opts.Sort == "index" {
					m.opts.Sort = "name"
				} else {
					m.opts.Sort = "index"
				}
				m.cursor = 0
			}
		case "s", "x":
			if m.opts.Command == "roproc" && key == "x" && m.err == nil && m.count() > 0 && (m.stage == "select" || m.stage == "detail") {
				m.opts.Action = "remove"
				m.opts.Process = m.visibleProcesses()[m.cursor].Name
				m.opts.RegistryGroup = ""
				m.opts.RequestID = transport.ID("c_")
				m.opts.Plan = nil
				m.busy = true
				return m, m.previewRegistry()
			}
			if m.opts.Command == "ropack" && key == "s" && m.err == nil && m.count() > 0 {
				if m.liveCancel != nil {
					m.liveCancel()
				}
				m.liveEpoch++
				return m, m.openProcessScreen()
			}
			if m.err == nil && m.opts.Command == "rodis" && m.count() > 0 {
				command := "rostart"
				if key == "x" {
					command = "rostop"
				}
				if !transport.Supports(m.client.Capabilities, command) {
					m.err = fmt.Errorf("%s is not supported by this Juno", command)
					return m, nil
				}
				if m.liveCancel != nil {
					m.liveCancel()
				}
				m.liveEpoch++
				m.returnCommand = "rodis"
				m.opts.Command = command
				m.opts.Targets = []string{m.visibleProcesses()[m.cursor].Name}
				m.opts.RequestID = transport.ID("c_")
				m.stage = "review"
				m.status = "대상 확인 · Enter 실행"
			}
		case "p":
			if m.stage == "select" && m.opts.Command != "ropack" {
				m.editing = "package"
				m.input = m.opts.Package
			}
		case "up", "k":
			if m.stage == "select" {
				m.cursor = max(0, m.cursor-1)
			} else {
				m.offset = max(0, m.offset-1)
			}
		case "down", "j":
			if m.stage == "select" {
				m.cursor = min(max(0, m.count()-1), m.cursor+1)
			} else {
				m.offset++
			}
		case "esc", "n":
			if m.op == nil || operations.Terminal(m.op.State) {
				m.stage = "select"
				m.op = nil
				m.stream = nil
				m.opts.RequestID = ""
				m.opts.Action = ""
				m.opts.Plan = nil
				m.err = nil
				m.status = "작업 선택"
				if m.returnCommand != "" {
					m.opts.Command = m.returnCommand
					m.returnCommand = ""
				}
				if m.opts.Command != "rocron" {
					m.opts.Targets = nil
					m.opts.Process = ""
					m.opts.Group = ""
					m.opts.All = false
					return m, m.load()
				}
			}
		case "enter":
			if m.err != nil {
				return m, nil
			}
			if m.opts.Command == "rodis" || m.opts.Command == "ropack" || (m.opts.Command == "roproc" && (m.stage == "select" || m.stage == "detail")) {
				if m.count() > 0 {
					m.stage = "detail"
				}
				return m, nil
			}
			if m.stage == "select" && m.catalog != nil && m.count() > 0 {
				if len(m.opts.Targets) == 0 {
					m.opts.Targets = []string{m.visibleProcesses()[m.cursor].Name}
				}
				m.stage = "review"
				if m.opts.RequestID == "" {
					m.opts.RequestID = transport.ID("c_")
				}
				m.status = "대상 확인 · Enter 실행"
			} else if m.stage == "select" && len(m.jobs) > 0 {
				j := m.jobs[m.cursor]
				m.opts.Process = j.Process
				m.opts.Job = j.Name
				m.input = j.Sample
				m.editing = "arguments"
				m.stage = "arguments"
				m.status = "인자 입력 · Enter 확인"
			} else if m.stage == "review" {
				m.stage = "result"
				m.busy = true
				m.status = "실행 요청 전달 중"
				return m, m.submit()
			}
		case " ":
			if m.opts.Command != "rodis" && m.opts.Command != "roproc" && m.stage == "select" && m.catalog != nil && m.count() > 0 {
				name := m.visibleProcesses()[m.cursor].Name
				found := -1
				for i, n := range m.opts.Targets {
					if n == name {
						found = i
					}
				}
				if found < 0 {
					m.opts.Targets = append(m.opts.Targets, name)
				} else {
					m.opts.Targets = append(m.opts.Targets[:found], m.opts.Targets[found+1:]...)
				}
			}
		}
	}
	return m, nil
}
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
func (m model) View() string {
	if m.width < 60 || m.height < 19 {
		return "터미널을 60열 × 19행 이상으로 늘려 주세요. q 종료"
	}
	width := m.width - 2
	bodyHeight := m.height - 9
	sideWidth := 16
	mainWidth := width - sideWidth - 3
	blue := lipgloss.Color("75")
	grey := lipgloss.Color("240")
	color := lipgloss.Color("214")
	if m.err != nil || (m.op != nil && (m.op.State == "FAILED" || m.op.State == "INTERRUPTED")) {
		color = lipgloss.Color("196")
	} else if m.op != nil && operations.Terminal(m.op.State) {
		color = lipgloss.Color("42")
	}
	header := m.opts.Command + " · rocontext 연결 확인"
	if m.client != nil {
		header = fmt.Sprintf("%s  |  %s  %s  |  %s  |  gRPC", m.opts.Command, m.client.Name, m.client.Config.Jupiter, m.client.Target.PackageId)
	}
	box := func(s string, w, h int, border lipgloss.Color) string {
		return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Width(w).Height(h).MaxWidth(w + 2).MaxHeight(h + 2).Render(s)
	}
	stages := []string{"select", "arguments", "review", "result"}
	labels := []string{"작업 선택", "인자 입력", "실행 확인", "전달 결과"}
	if m.opts.Command != "rocron" {
		stages = []string{"select", "review", "result"}
		labels = []string{"대상 선택", "실행 확인", "진행 / 결과"}
	}
	if m.opts.Command == "rodis" {
		stages = []string{"select", "detail"}
		labels = []string{"프로세스 상태", "상세 정보"}
	}
	if m.opts.Command == "ropack" {
		stages = []string{"select", "detail"}
		labels = []string{"패키지 목록", "상세 정보"}
	}
	if m.opts.Command == "roproc" {
		stages = []string{"select", "arguments", "review", "result"}
		labels = []string{"등록부 / 상세", "등록 입력", "변경 확인", "진행 / 결과"}
	}
	var rail []string
	for i, s := range stages {
		prefix := "  "
		if s == m.stage || (m.opts.Command == "roproc" && m.stage == "detail" && s == "select") {
			prefix = "▶ "
		}
		rail = append(rail, prefix+labels[i])
	}
	var content []string
	if m.client == nil {
		if m.err != nil {
			content = append(content, "Connection · 접속 실패", clean(m.err.Error()), "", "r 재시도 · p 옵션으로 대상 지정 · q 종료")
		} else {
			content = append(content, "서버 기능과 인증을 확인하고 있습니다.")
		}
	} else if m.opts.Command == "ropack" && m.packages != nil {
		content = m.packageRows(bodyHeight, mainWidth-2)
	} else if m.stage == "select" && m.catalog != nil {
		content = m.processRows(bodyHeight, mainWidth-2)
	} else if m.stage == "select" {
		content = append(content, "등록된 배치 작업")
		room := max(1, bodyHeight/2-2)
		start := 0
		if m.cursor >= room {
			start = m.cursor - room + 1
		}
		for i := start; i < min(len(m.jobs), start+room); i++ {
			j := m.jobs[i]
			p := "  "
			if i == m.cursor {
				p = "▶ "
			}
			content = append(content, line(fmt.Sprintf("%s%s / %s", p, j.Process, j.Name), mainWidth-2))
		}
		if len(m.jobs) == 0 {
			content = append(content, "등록된 배치 작업이 없습니다.")
		} else {
			j := m.jobs[m.cursor]
			content = append(content, "──────── 상세 ────────", line(j.Description, mainWidth-2), "스케줄: "+j.Spec, "인자 예시: "+j.Sample)
		}
	} else if m.stage == "detail" && m.catalog != nil && m.count() > 0 {
		content = processDetail(m.visibleProcesses()[m.cursor])
	} else if m.opts.Command == "roproc" && m.stage == "review" {
		content = m.registryReview()
	} else if m.opts.Command == "roproc" && m.stage == "arguments" {
		content = m.registryInput()
	} else if m.stage == "arguments" || m.stage == "review" {
		if m.opts.Command == "rocron" {
			content = append(content, "프로세스: "+m.opts.Process, "배치 작업: "+m.opts.Job, "인자: "+m.opts.Arguments, "", CronNotice)
		} else {
			content = append(content, "작업: "+m.opts.Command, "대상 패키지: "+m.client.Target.PackageId, "선택 프로세스:")
			content = append(content, m.opts.Targets...)
		}
		if m.stage == "review" {
			content = append(content, "", "요청 ID: "+m.opts.RequestID, "Enter: 위 대상으로 실행 요청 / Esc: 돌아가기")
		}
	} else {
		content = append(content, "요청 ID: "+m.opts.RequestID)
		if m.op != nil {
			title := m.op.State
			if m.op.State == "REQUESTED" {
				title = "✓ 실행 요청 전달 완료"
			}
			if m.op.State == "SUCCEEDED" {
				title = "✓ 선택 대상 모두 완료"
			}
			content = append(content, title)
			if m.op.Kind == "rocron" {
				content = append(content, CronNotice)
			}
			content = append(content, "", clean(m.op.Message))
			for _, e := range m.op.Events {
				content = append(content, fmt.Sprintf("%s [%s] %s", time.Unix(e.At, 0).Local().Format("15:04:05"), e.Stage, e.Message))
			}
		}

	}
	if m.err != nil && m.client != nil {
		content = append([]string{"오류: " + clean(m.err.Error()), ""}, content...)
	}
	if m.editing != "" {
		content = append([]string{m.editing + "> " + m.input + "█", ""}, content...)
	}
	wrapped := strings.Split(lipgloss.NewStyle().Width(mainWidth-2).Render(strings.Join(content, "\n")), "\n")
	maxOffset := max(0, len(wrapped)-bodyHeight)
	start := min(m.offset, maxOffset)
	wrapped = wrapped[start:min(len(wrapped), start+bodyHeight)]
	top := box(line(header, width-2), width, 1, blue)
	nav := "  " + strings.Join(labels, "  ›  ") + "   [" + m.stage + "]"
	body := lipgloss.JoinHorizontal(lipgloss.Top, box(strings.Join(rail, "\n\n"), sideWidth, bodyHeight, grey), " ", box(strings.Join(wrapped, "\n"), mainWidth-2, bodyHeight, blue))
	help := "q 종료  ↑↓ 이동  Enter 선택/확인  p 패키지  r 갱신  n 목록"
	if m.opts.Command == "rodis" {
		help = "q 종료  ↑↓ 이동  / 검색  o 정렬  Enter 상세  s 시작  x 중단  r 재연결"
	}
	if m.opts.Command == "ropack" {
		help = "q 종료  ↑↓ 이동  / 검색  Enter 상세  s 프로세스 상태  r 재연결  n 목록"
	}
	if m.opts.Command == "roproc" {
		help = "q 종료  ↑↓ 이동/스크롤  a 등록  x 삭제  Enter 상세/확인  n 목록  r 재조회"
	}
	footer := lipgloss.NewStyle().Foreground(color).Render(line("● "+m.status, width)) + "\n" + line(help, width)
	return top + "\n" + line(nav, width) + "\n" + body + "\n" + footer
}
