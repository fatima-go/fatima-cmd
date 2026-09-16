package deployui

import (
	"context"
	"fmt"
	"github.com/fatima-go/fatima-cmd/share"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-opm/api"
	"github.com/fatima-go/fatima-opm/lifecycle"
	"github.com/fatima-go/fatima-opm/transport"
	"golang.org/x/term"
)

type Options struct {
	Group, First, Endpoint, Command, Value, RequestID, ArtifactID, Action string
	Revision                                                              uint64
	JSON, Legacy, Debug                                                   bool
}
type event struct {
	kind           string
	value          any
	err            error
	current, total int64
	id             string
}
type pulse time.Time
type model struct {
	ctx                           context.Context
	client                        *Client
	opts                          Options
	events                        chan event
	view, input, notice, confirm  string
	artifacts                     []*api.Artifact
	rollouts                      []*api.Rollout
	targets                       []*api.Target
	artifact                      *api.Artifact
	rollout                       *api.Rollout
	draft                         *api.CreateRollout
	cursor, scroll, width, height int
	busy                          bool
	bytes, total                  int64
	watchCancel                   context.CancelFunc
	legacyLog                     string
	legacy                        legacyOutcome
	follow                        bool
	confirmRevision               uint64
	readContext                   contextReader
	connection                    connectionResult
	booting                       bool
	initialView                   string
	startupNotice                 string
	diagnostics, showIdentity     bool
	focusDetail, editingPath      bool
	previousInput                 string
	locals                        localFiles
	preview                       *farPreview
	previewPending                bool
	lastError                     string
	legacyGroup                   string
	legacyTargets                 []string
	menuIndex                     int
	replaceAfterCancel            bool
	connectionLost                bool
}

var titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("222"))
var mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
var selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("222")).Bold(true)
var alertStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

func RunInteractive(o Options) error { return runWithReader(readActiveContext, o) }

func Run(cfg config.JupiterContextRecord, o Options) error {
	return runWithReader(func() (config.JupiterContextRecord, string, error) { return cfg, "", nil }, o)
}

func newModel(ctx context.Context, read contextReader, o Options) *model {
	// A new FAR is the usual start; a switches to already uploaded artifacts.
	view := "upload"
	switch o.Command {
	case "artifacts", "rollouts", "watch":
		view = o.Command
	}
	return &model{ctx: ctx, opts: o, readContext: read, events: make(chan event, 64), width: 100, height: 32, view: view, initialView: view, input: o.Value, diagnostics: o.Debug}
}

func runWithReader(read contextReader, o Options) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("TUI requires a terminal; use --json for automation or --legacy for the original command")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newModel(ctx, read, o)
	defer func() {
		cancel() // Interrupt in-flight UI requests before detaching the owner.
		if m.client != nil {
			m.client.Close()
		}
	}()
	_, e := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if e != nil {
		return e
	}
	if m.connection.failure != nil {
		return m.connection.failure
	}
	return nil
}
func (m *model) emit(e event) {
	select {
	case m.events <- e:
	case <-m.ctx.Done():
	}
}
func (m *model) next() tea.Cmd {
	return func() tea.Msg {
		select {
		case e := <-m.events:
			return e
		case <-m.ctx.Done():
			return nil
		}
	}
}
func tick() tea.Cmd { return tea.Tick(time.Second, func(t time.Time) tea.Msg { return pulse(t) }) }
func (m *model) task(kind string, fn func(context.Context) (any, error)) tea.Cmd {
	m.busy = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 60*time.Second)
		defer cancel()
		v, e := fn(ctx)
		return event{kind: kind, value: v, err: e}
	}
}
func (m *model) load() tea.Cmd {
	if m.view == "activity" {
		return m.recentRollouts()
	}
	if m.view == "rollouts" {
		return m.task("rollouts", func(ctx context.Context) (any, error) { return m.client.Rollouts(ctx) })
	}
	return m.task("artifacts", func(ctx context.Context) (any, error) { return m.client.Artifacts(ctx) })
}
func (m *model) Init() tea.Cmd {
	return tea.Batch(m.next(), tick(), m.connect())
}
func (m *model) connect() tea.Cmd {
	m.booting, m.busy = true, true
	m.notice = "Jupiter 연결·인증 확인 중…"
	read := m.readContext
	return func() tea.Msg { return event{kind: "connected", value: connect(m.ctx, read)} }
}
func (m *model) localScan() tea.Cmd {
	m.legacy, m.legacyLog = legacyOutcome{}, ""
	m.follow = false
	return m.task("local-files", func(context.Context) (any, error) { return scanLocalFARs(os.Getenv("GOPATH")), nil })
}
func (m *model) inspectInput() tea.Cmd {
	input := m.input
	m.preview, m.previewPending = nil, true
	return func() tea.Msg { return event{kind: "preview", id: input, value: inspectLocalFAR(input)} }
}
func (m *model) selectLocal(index int) tea.Cmd {
	if index < 0 || index >= len(m.locals.Files) {
		return nil
	}
	m.cursor, m.scroll = index, 0
	m.setInput(m.locals.Files[index].Path)
	return m.inspectInput()
}
func (m *model) setInput(input string) {
	if input != m.input {
		m.opts.RequestID = ""
	}
	m.input = input
}
func (m *model) leaveWatch() {
	if m.watchCancel != nil {
		m.watchCancel()
		m.watchCancel = nil
	}
}
func (m *model) startWatch(p *api.Rollout) {
	if m.watchCancel != nil {
		m.watchCancel()
	}
	m.rollout = p
	m.view = "watch"
	m.scroll, m.cursor = 0, activePackage(p)
	m.follow = true
	m.focusDetail = true
	ctx, cancel := context.WithCancel(m.ctx)
	m.watchCancel = cancel
	go func() {
		var revision uint64
		for ctx.Err() == nil {
			session, stop := context.WithTimeout(ctx, 5*time.Minute)
			stream, e := m.client.Watch(session, p.Id, revision)
			if e == nil {
				for {
					snapshot, err := stream.Recv()
					if err != nil {
						e = err
						break
					}
					revision = snapshot.Revision
					m.emit(event{kind: "snapshot", value: snapshot, id: p.Id})
				}
			}
			stop()
			if ctx.Err() != nil {
				return
			}
			m.emit(event{kind: "reconnect", id: p.Id, err: e})
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}()
}
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
	case pulse:
		return m, tick()
	case event:
		v.err = share.UnsupportedRPC(v.err, "Jupiter 또는 대상 Juno")
		if v.kind == "connected" {
			m.booting, m.busy = false, false
			m.connection = v.value.(connectionResult)
			m.opts.Endpoint = m.connection.config.Jupiter
			if m.connection.failure != nil {
				m.view = "connection"
				m.notice = m.connection.failure.Message
				return m, nil
			}
			m.client, m.opts.Legacy = m.connection.client, m.connection.legacy
			m.view, m.scroll = m.initialView, 0
			m.lastError = ""
			m.notice = "Jupiter 인증 완료"
			if m.opts.Legacy {
				m.view = "legacy"
				m.notice = "Legacy HTTP: 기존 업로드·응답 흐름을 사용합니다."
				return m, m.localScan()
			}
			switch m.view {
			case "upload":
				return m, tea.Batch(m.localScan(), m.checkStartupRollouts())
			case "watch":
				return m, m.task("watch", func(ctx context.Context) (any, error) { return m.client.Get(ctx, m.opts.Value) })
			default:
				return m, m.load()
			}
		}
		if v.kind == "startup-rollouts" {
			if v.err != nil {
				m.startupNotice = "기존 배포 조회 실패 · l 목록에서 재확인"
			} else {
				rollouts := v.value.(*api.RolloutList).Rollouts
				active := 0
				for _, rollout := range rollouts {
					if !lifecycle.Terminal(rollout.State) {
						active++
					}
				}
				m.startupNotice = fmt.Sprintf("기존 배포 %d건 · 미종료 %d건 · l 목록 보기", len(rollouts), active)
			}
			// This background result must not change navigation, upload progress,
			// or another request's busy/error state.
			return m, nil
		}
		if v.kind == "preview" {
			if v.id != m.input || m.editingPath || (m.view != "upload" && m.view != "legacy") {
				return m, nil
			}
			p := v.value.(farPreview)
			m.preview, m.previewPending = &p, false
			if p.File.Path != "" {
				m.input = p.File.Path
			}
			if p.Err != nil {
				m.notice = p.Err.Error()
			} else {
				m.notice = "FAR 확인 완료 · Enter 업로드 · p 경로 수정"
			}
			return m, nil
		}
		if v.kind == "snapshot" {
			if m.view == "watch" && m.rollout != nil && m.rollout.Id == v.id {
				m.rollout = v.value.(*api.Rollout)
				if m.follow && m.focusDetail {
					m.cursor = activePackage(m.rollout)
				}
				m.connectionLost = false
				m.notice = "서버 연결됨 · " + rolloutLabel(m.rollout)
				if m.replaceAfterCancel && lifecycle.Terminal(m.rollout.State) && m.draft != nil {
					m.restoreDraft()
				}
			}
			return m, m.next()
		}
		if v.kind == "reconnect" {
			if m.view == "watch" && m.rollout != nil && m.rollout.Id == v.id {
				m.connectionLost = true
				m.notice = "연결 복구 중 · 표시된 상태는 마지막 수신 결과입니다"
			}
			return m, m.next()
		}
		if v.kind == "progress" {
			m.bytes = v.current
			m.total = v.total
			return m, m.next()
		}
		if v.kind == "legacy-output" {
			m.legacyLog += v.value.(string)
			if len(m.legacyLog) > 24000 {
				m.legacyLog = m.legacyLog[len(m.legacyLog)-24000:]
			}
			return m, m.next()
		}
		if v.kind == "legacy-done" {
			m.busy = false
			m.legacy = legacyResult(m.legacyLog, v.err)
			m.notice = m.legacy.Summary
			return m, m.next()
		}
		m.busy = false
		if v.err != nil {
			m.notice = v.err.Error()
			m.lastError = m.notice
			if v.kind == "recent" {
				m.notice = "기존 작업 조회 실패 · l로 다시 확인할 수 있습니다"
				return m, m.localScan()
			}
			if v.kind == "create" {
				if id := conflictRollout(v.err); id != "" {
					m.confirm = ""
					m.lastError = ""
					return m, m.task("conflict", func(ctx context.Context) (any, error) { return m.client.Get(ctx, id) })
				}
				if uncertainSubmission(v.err) {
					m.notice += " · 같은 요청으로 자동 확인했지만 응답이 없습니다. 기존 작업 목록에서 결과를 확인할 수 있습니다"
				} else {
					m.notice += " · 이번 새 배포는 시작되지 않았습니다"
				}
			}
			if v.kind == "upload" {
				return m, m.next()
			}
			return m, nil
		}
		m.lastError = ""
		switch v.kind {
		case "recent":
			m.rollouts = v.value.(*api.RolloutList).Rollouts
			sortActivity(m.rollouts)
			if len(m.rollouts) == 0 {
				return m, m.localScan()
			}
			m.view = "activity"
			m.cursor = 1
			m.focusDetail = false
			m.notice = "이전 배포를 확인하거나 새 배포를 준비하세요"
			return m, m.next()
		case "conflict":
			m.startWatch(v.value.(*api.Rollout))
			m.confirm = "conflict"
			m.menuIndex = 0
			m.notice = "기존 작업과 충돌 · 이번 새 배포는 시작되지 않았습니다"
			return m, m.next()
		case "local-files":
			m.locals = v.value.(localFiles)
			m.notice = m.locals.Warning
			if m.input != "" {
				m.cursor = -1
				path, _ := normalizeFARPath(m.input)
				for i, f := range m.locals.Files {
					if f.Path == path {
						m.cursor = i
					}
				}
				return m, m.inspectInput()
			}
			return m, m.selectLocal(0)
		case "artifacts":
			m.artifacts = v.value.(*api.ArtifactList).Artifacts
			m.cursor = min(max(0, m.cursor), max(0, len(m.artifacts)-1))
			if m.artifact != nil {
				for i, a := range m.artifacts {
					if a.Id == m.artifact.Id {
						m.cursor = i
					}
				}
				m.artifact = nil
			}
			m.notice = "Jupiter authentication passed · artifacts loaded"
		case "rollouts":
			m.rollouts = v.value.(*api.RolloutList).Rollouts
			sortActivity(m.rollouts)
			m.notice = "기존 배포 목록 · Enter 상세 보기 → Enter 동작 메뉴"
		case "targets":
			m.targets = v.value.(*api.TargetList).Targets
			m.view = "targets"
			m.cursor = 0
			m.focusDetail, m.scroll = false, 0
			m.notice = "Select the first package. The plan includes every package in that group."
			for i, t := range m.targets {
				if t.PackageId == m.opts.First {
					m.cursor = i
				}
			}
		case "create", "watch", "action":
			m.startWatch(v.value.(*api.Rollout))
			m.confirm = ""
			if m.replaceAfterCancel && lifecycle.Terminal(m.rollout.State) && m.draft != nil {
				m.restoreDraft()
			}
		case "upload":
			m.artifact = v.value.(*api.Artifact)
			m.view = "artifacts"
			m.opts.Value = m.input
			m.focusDetail, m.scroll = false, 0
			m.opts.RequestID = ""
			m.notice = "Upload finalized: " + m.artifact.Id
			return m, tea.Batch(m.next(), m.load())
		}
	case tea.KeyMsg:
		key := v.String()
		if key == "ctrl+c" || (key == "q" && !m.editingPath) {
			if m.watchCancel != nil {
				m.watchCancel()
			}
			return m, tea.Quit
		}
		// Never act on a confirmation that the terminal is too small to show.
		if (m.width > 0 && m.width < 60) || (m.height > 0 && m.height < 19) {
			return m, nil
		}
		if m.view == "connection" {
			if key == "d" {
				m.diagnostics = !m.diagnostics
				m.scroll = 0
			} else if key == "r" && !m.booting {
				return m, m.connect()
			} else if key == "down" || key == "pgdown" {
				m.scroll++
			} else if key == "up" || key == "pgup" {
				m.scroll = max(0, m.scroll-1)
			}
			return m, nil
		}
		if m.booting {
			return m, nil
		}
		if m.editingPath {
			switch key {
			case "enter":
				m.editingPath = false
				m.cursor, m.scroll = -1, 0
				return m, m.inspectInput()
			case "esc":
				m.editingPath = false
				m.setInput(m.previousInput)
				return m, m.inspectInput()
			case "backspace":
				runes := []rune(m.input)
				if len(runes) > 0 {
					m.setInput(string(runes[:len(runes)-1]))
				}
			case "ctrl+u":
				m.setInput("")
			default:
				if v.Type == tea.KeyRunes {
					m.setInput(m.input + string(v.Runes))
				}
			}
			return m, nil
		}
		if m.view == "legacy" && m.legacy.State != "" && m.confirm == "" {
			switch key {
			case "up", "k", "pgup":
				m.follow = false
				m.scroll = max(0, m.scroll-1)
			case "down", "j", "pgdown":
				m.follow = false
				m.scroll++
			case "f":
				m.follow = true
			case "n", "u", "esc":
				if !m.busy {
					m.focusDetail, m.scroll = false, 0
					return m, m.localScan()
				}
			case "a", "l":
				// A mixed-version group may return to the new Jupiter's lists.
				if !m.opts.Legacy {
					break
				}
				return m, nil
			default:
				return m, nil
			}
			if key != "a" && key != "l" {
				return m, nil
			}
		}
		if key == "esc" {
			if m.busy {
				return m, nil
			}
			if m.confirm != "" {
				m.confirm = ""
				return m, nil
			}
			if m.watchCancel != nil {
				m.watchCancel()
				m.watchCancel = nil
			}
			m.view = "artifacts"
			m.cursor = 0
			m.scroll = 0
			m.focusDetail = false
			if m.opts.Legacy {
				m.view = "legacy"
				return m, nil
			}
			return m, m.load()
		}
		if m.busy {
			return m, nil
		}
		if m.confirm == "actions" || m.confirm == "conflict" {
			labels := m.actionLabels()
			switch key {
			case "up", "k":
				m.menuIndex = max(0, m.menuIndex-1)
			case "down", "j":
				m.menuIndex = min(len(labels)-1, m.menuIndex+1)
			case "enter":
				return m, m.chooseAction()
			}
			return m, nil
		}
		if m.confirm != "" {
			if key == "pgdown" || key == "down" {
				m.scroll++
			} else if key == "pgup" || key == "up" {
				m.scroll = max(0, m.scroll-1)
			}
			if key == "enter" {
				if m.confirm == "create" {
					q := m.draft
					return m, m.task("create", func(ctx context.Context) (any, error) { return m.client.Create(ctx, q) })
				}
				if m.confirm == "legacy-group" {
					m.confirm = ""
					m.view = "legacy"
					m.input = m.opts.Value
					m.opts.Group = m.legacyGroup
					m.opts.First = ""
					m.notice = "Legacy mode selected before deployment. Enter the original FAR path; detailed stages and remaining rollout are unavailable."
					return m, m.localScan()
				}
				if m.confirm == "legacy" {
					m.confirm = ""
					m.startLegacy()
					return m, nil
				}
				action := m.confirm
				p := m.rollout
				revision := m.confirmRevision
				return m, m.task("action", func(ctx context.Context) (any, error) {
					return m.client.Act(ctx, &api.RolloutAction{Id: p.Id, Action: action, ExpectedRevision: revision})
				})
			}
			return m, nil
		}
		if m.view == "upload" || m.view == "legacy" {
			switch key {
			case "enter":
				if m.previewPending {
					return m, nil
				}
				if m.preview == nil || (m.preview.Err != nil && (m.view != "legacy" || m.preview.File.Size <= 0)) {
					m.notice = "유효한 FAR를 선택하거나 p를 눌러 경로를 입력하세요."
					return m, nil
				}
				st, e := os.Stat(m.input)
				if e != nil || st.Size() != m.preview.File.Size || !st.ModTime().Equal(m.preview.File.Modified) {
					m.notice = "FAR가 변경됐습니다. 파일 정보를 다시 확인한 뒤 업로드하세요."
					return m, m.inspectInput()
				}
				if m.view == "legacy" {
					m.confirm = "legacy"
					return m, nil
				}
				m.startUpload()
				return m, nil
			case "p":
				m.editingPath, m.previousInput = true, m.input
				m.focusDetail, m.scroll = false, 0
				return m, nil
			case "r":
				return m, m.localScan()
			case "up", "k":
				if !m.focusDetail {
					return m, m.selectLocal(m.cursor - 1)
				}
			case "down", "j":
				if !m.focusDetail {
					return m, m.selectLocal(m.cursor + 1)
				}
			}
		}
		switch key {
		case "tab":
			m.focusDetail = !m.focusDetail
			m.scroll = 0
		case "i":
			m.showIdentity = !m.showIdentity
		case "d":
			m.diagnostics = !m.diagnostics
		case "up", "k":
			if m.focusDetail {
				m.follow = false
				if m.scroll > 0 {
					m.scroll--
				}
			} else if m.cursor > 0 {
				m.cursor--
				m.scroll = 0
			}
		case "down", "j":
			if m.focusDetail {
				m.follow = false
				m.scroll++
			} else if m.cursor+1 < m.count() {
				m.cursor++
				m.scroll = 0
			}
		case "pgdown":
			m.follow = false
			m.scroll += 10
		case "pgup":
			m.follow = false
			m.scroll = max(0, m.scroll-10)
		case "f":
			if m.view == "watch" {
				m.follow, m.focusDetail = true, true
				m.cursor = activePackage(m.rollout)
			}
		case "u":
			m.leaveWatch()
			m.view = "upload"
			if m.opts.Legacy {
				m.view = "legacy"
			}
			m.input = m.opts.Value
			m.focusDetail, m.scroll = false, 0
			return m, m.localScan()
		case "a":
			if m.opts.Legacy {
				return m, nil
			}
			m.leaveWatch()
			m.view = "artifacts"
			m.cursor = 0
			m.focusDetail, m.scroll = false, 0
			return m, m.load()
		case "l":
			if m.opts.Legacy {
				return m, nil
			}
			m.leaveWatch()
			m.view = "rollouts"
			m.cursor = 0
			m.focusDetail, m.scroll = false, 0
			return m, m.load()
		case "r":
			if m.view == "watch" {
				if m.rollout == nil {
					return m, m.task("watch", func(ctx context.Context) (any, error) { return m.client.Get(ctx, m.opts.Value) })
				}
				m.confirmRevision = m.rollout.Revision
				if m.rollout.State == "ATTENTION" {
					m.confirm = "resume"
				} else if m.rollout.State == "FAILED" {
					m.confirm = "retry"
				}
			} else {
				return m, m.load()
			}
		case "c":
			if m.view == "watch" && m.rollout != nil && m.rollout.State == "WAITING" {
				m.confirmRevision = m.rollout.Revision
				m.confirm = "continue"
			}
		case "x":
			if m.view == "watch" && m.rollout != nil {
				m.confirmRevision = m.rollout.Revision
				m.confirm = "cancel"
			}
		case "enter":
			switch m.view {
			case "activity":
				if m.cursor == 0 {
					m.view = "upload"
					return m, m.localScan()
				}
				if m.cursor <= len(m.rollouts) {
					m.startWatch(m.rollouts[m.cursor-1])
				}
			case "watch":
				m.confirm = "actions"
				m.menuIndex = 0
			case "artifacts":
				if m.cursor >= 0 && m.cursor < len(m.artifacts) {
					m.artifact = m.artifacts[m.cursor]
					if m.artifact.ExpiresAt <= time.Now().Unix() {
						m.notice = "EXPIRED · 새 배포에는 FAR를 다시 업로드하세요."
						return m, nil
					}
					return m, m.task("targets", func(ctx context.Context) (any, error) { return m.client.Targets(ctx, m.opts.Group) })
				}
			case "rollouts":
				if m.cursor >= 0 && m.cursor < len(m.rollouts) {
					m.startWatch(m.rollouts[m.cursor])
				}
			case "targets":
				if m.cursor < 0 || m.cursor >= len(m.targets) {
					return m, nil
				}
				first := m.targets[m.cursor]
				ids, legacyIDs := []string{}, []string{}
				for _, t := range m.targets {
					if t.Group == first.Group {
						ids = append(ids, t.PackageId)
						if !t.Supported {
							if !t.Legacy {
								m.notice = "Capability check failed; no deployment submitted: " + t.PackageId + ": " + t.Reason
								return m, nil
							}
							legacyIDs = append(legacyIDs, t.PackageId)
						}
					}
				}
				if len(legacyIDs) > 0 {
					// Name the packages that force the whole group onto legacy HTTP
					// before switching, so the operator can cancel instead.
					m.legacyGroup, m.legacyTargets = first.Group, legacyIDs
					m.confirm, m.scroll = "legacy-group", 0
					m.notice = "The group includes a legacy Juno; confirm before switching to legacy HTTP."
					return m, nil
				}
				m.draft = &api.CreateRollout{RequestId: transport.ID("cli_"), ArtifactId: m.artifact.Id, Group: first.Group, PackageIds: ids, FirstPackageId: first.PackageId}
				m.confirm = "create"
			}
		}
	}
	return m, nil
}
func (m *model) startUpload() {
	m.busy = true
	m.bytes = 0
	m.total = 0
	m.notice = "Hashing FAR, authenticating, then uploading to Jupiter"
	path := strings.TrimSpace(m.input)
	if strings.HasPrefix(path, "~/") {
		if home, e := os.UserHomeDir(); e == nil {
			path = home + path[1:]
		}
	}
	if m.opts.RequestID == "" {
		m.opts.RequestID = transport.ID("cli_")
	}
	requestID := m.opts.RequestID
	m.opts.Value = path
	go func() {
		ctx, cancel := context.WithTimeout(m.ctx, 30*time.Minute)
		defer cancel()
		a, e := m.client.Upload(ctx, path, requestID, func(n, total int64) { m.emit(event{kind: "progress", current: n, total: total}) })
		m.emit(event{kind: "upload", value: a, err: e})
	}()
}
func (m *model) startLegacy() {
	m.busy = true
	m.legacyLog = ""
	m.lastError = ""
	m.legacy = legacyOutcome{State: "RUNNING", Summary: "FAR 전송 및 Jupiter 응답 대기 중…"}
	m.notice = m.legacy.Summary
	m.follow, m.focusDetail, m.scroll = true, true, 0
	args := []string{"--legacy"}
	if m.opts.Group != "" {
		args = append(args, "-g", m.opts.Group)
	}
	if m.opts.First != "" {
		args = append(args, "-p", m.opts.First)
	}
	args = append(args, strings.TrimSpace(m.input))
	go func() {
		exe, e := os.Executable()
		if e != nil {
			m.emit(event{kind: "legacy-done", err: e})
			return
		}
		cmd := exec.CommandContext(m.ctx, exe, args...)
		pipe, e := cmd.StdoutPipe()
		if e != nil {
			m.emit(event{kind: "legacy-done", err: e})
			return
		}
		cmd.Stderr = cmd.Stdout
		if e = cmd.Start(); e != nil {
			m.emit(event{kind: "legacy-done", err: e})
			return
		}
		buf := make([]byte, 4096)
		var readErr error
		for {
			n, e := pipe.Read(buf)
			if n > 0 {
				m.emit(event{kind: "legacy-output", value: string(buf[:n])})
			}
			if e == io.EOF {
				break
			}
			if e != nil {
				readErr = e
				break
			}
		}
		if e = cmd.Wait(); e == nil {
			e = readErr
		}
		m.emit(event{kind: "legacy-done", err: e})
	}()
}
func (m *model) count() int {
	switch m.view {
	case "upload", "legacy":
		return len(m.locals.Files)
	case "activity":
		return 1 + len(m.rollouts)
	case "watch":
		if m.rollout != nil {
			return len(m.rollout.Targets)
		}
	case "artifacts":
		return len(m.artifacts)
	case "rollouts":
		return len(m.rollouts)
	case "targets":
		return len(m.targets)
	}
	return 0
}
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, ansi.Strip(s))
}
func bytesText(n int64) string { return fmt.Sprintf("%.1f MiB", float64(n)/(1024*1024)) }
func bar(n, total int64) string {
	if total <= 0 {
		return ""
	}
	ratio := min(1, float64(n)/float64(total))
	filled := int(ratio * 24)
	return "[" + strings.Repeat("=", filled) + strings.Repeat("·", 24-filled) + fmt.Sprintf("] %5.1f%%  %s / %s", 100*ratio, bytesText(n), bytesText(total))
}
func (m *model) View() string { return m.renderLayout() }
