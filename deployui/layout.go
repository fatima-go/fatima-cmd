package deployui

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-core/opm/api"
)

var ruleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
var goodStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
var errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("210"))
var activeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("222")).Background(lipgloss.Color("236")).Bold(true)

var stages = []string{"Upload", "Artifact", "Target", "First deploy", "Verify", "Remaining"}
var stageDescriptions = []string{"로컬 FAR 선택·전송", "업로드된 배포본", "첫 패키지 선택", "첫 패키지 배포", "서비스 확인·승인", "나머지 순차 배포"}

type screenContent struct {
	title, description, listTitle, rowHeader, detailTitle string
	rows, details                                         []string
	selected                                              int
}

func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = ansi.Truncate(s, width, "…")
	return s + strings.Repeat(" ", max(0, width-ansi.StringWidth(s)))
}

func wrapLines(lines []string, width int) []string {
	var result []string
	for _, s := range lines {
		result = append(result, strings.Split(ansi.Hardwrap(s, max(1, width), true), "\n")...)
	}
	return result
}

func localTimestamp(seconds int64) string {
	if seconds == 0 {
		return "—"
	}
	return time.Unix(seconds, 0).Local().Format("2006-01-02 15:04:05")
}

func shortID(id string) string {
	if len(id) > 14 {
		return id[:12] + "…"
	}
	return id
}

func field(label, value string) string {
	if strings.TrimSpace(value) == "" {
		value = "—"
	}
	return mutedStyle.Render(fit(label, 12)) + clean(value)
}

func buildFields(raw string, identity bool) []string {
	var b struct {
		User string `json:"user"`
		Time string `json:"time"`
		Git  struct {
			Branch, Commit, Message, Repo string
		} `json:"git"`
	}
	if raw == "" || raw == "null" {
		return []string{mutedStyle.Render("빌드 정보가 없습니다.")}
	}
	if e := json.Unmarshal([]byte(raw), &b); e != nil {
		return []string{mutedStyle.Render("빌드 정보를 해석할 수 없습니다.")}
	}
	commit := b.Git.Commit
	if !identity && len(commit) > 12 {
		commit = commit[:12]
	}
	return []string{field("빌드 시각", b.Time), field("빌드 사용자", b.User), field("브랜치", b.Git.Branch), field("커밋", commit), field("메시지", strings.TrimSpace(b.Git.Message)), field("저장소", displayEndpoint(b.Git.Repo))}
}

func activePackage(p *api.Rollout) int {
	if p == nil || len(p.Targets) == 0 {
		return 0
	}
	for i, t := range p.Targets {
		if t.Operation == nil {
			continue
		}
		switch t.Operation.State {
		case "STAGING", "STAGED", "RUNNING", "FAILED", "INTERRUPTED", "DRIFTED":
			return i
		}
	}
	if p.State == "SUCCEEDED" {
		return len(p.Targets) - 1
	}
	return 0
}

// skipUploadBadge marks the key that deploys an already uploaded artifact
// instead of uploading a new FAR.
var skipUploadBadge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("28"))

func (m *model) stage() int {
	switch m.view {
	case "connection":
		return -2
	case "upload", "legacy":
		return 0
	case "artifacts":
		return 1
	case "targets":
		return 2
	case "watch":
		if m.rollout != nil {
			if m.rollout.State == "WAITING" {
				return 4
			}
			if m.rollout.RemainingApproved || (m.rollout.State == "SUCCEEDED" && len(m.rollout.Targets) > 1) {
				return 5
			}
		}
		return 3
	default:
		return -1
	}
}

func (m *model) connectionLabel() string {
	if m.booting {
		return alertStyle.Render("CONNECTING")
	}
	if m.connection.failure != nil {
		return errorStyle.Render(m.connection.failure.Code)
	}
	if m.opts.Legacy || m.view == "legacy" {
		return alertStyle.Render("HTTP / LEGACY")
	}
	return goodStyle.Render("gRPC / v2")
}

func (m *model) legacyStyle() lipgloss.Style {
	switch m.legacy.State {
	case "COMPLETED":
		return goodStyle
	case "FAILED":
		return errorStyle
	default:
		return alertStyle
	}
}

func (m *model) legacyHeading() string {
	switch m.legacy.State {
	case "COMPLETED":
		return "✓ 배포 요청 완료"
	case "FAILED":
		return "✕ 배포 요청 실패"
	case "UNCONFIRMED":
		return "! 명령 종료 · 결과 확인 필요"
	default:
		return "● 배포 요청 진행 중"
	}
}

func (m *model) rail(width, height int) []string {
	n := m.stage()
	lines := []string{mutedStyle.Render("DEPLOY FLOW"), ""}
	if n == -2 {
		lines = append(lines, activeStyle.Render(fit("▶ Connection", width)), "")
	}
	spacious := height >= 25 && width >= 22
	for i, name := range stages {
		symbol := fmt.Sprintf("%02d", i+1)
		style := mutedStyle
		if n >= 0 && i < n && !m.booting {
			symbol, style = "✓ ", goodStyle
		}
		if n == i {
			symbol, style = "▶ ", activeStyle
		}
		if m.view == "legacy" && m.legacy.State == "COMPLETED" && i == 0 {
			symbol, style = "✓ ", goodStyle
		}
		lines = append(lines, style.Render(fit(symbol+" "+name, width)))
		if spacious {
			lines = append(lines, mutedStyle.Render(fit("   "+stageDescriptions[i], width)), "")
		}
	}
	if n == -1 {
		lines = append(lines, "", activeStyle.Render("≡ Rollouts"))
	}
	if n == -2 {
		lines = append(lines, "", mutedStyle.Render("배포 시작 전"))
	} else if m.rollout != nil && m.view == "watch" {
		lines = append(lines, "", ruleStyle.Render(strings.Repeat("─", width)), clean(m.rollout.Artifact.Process), mutedStyle.Render(clean(m.rollout.Group)))
	} else if m.artifact != nil {
		lines = append(lines, "", clean(m.artifact.Process))
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = fit(lines[i], width)
	}
	return lines[:height]
}

func drawPanel(title string, content []string, width, height, offset int, focused bool) []string {
	if height < 3 || width < 5 {
		return make([]string, max(0, height))
	}
	style := ruleStyle
	if focused {
		style = titleStyle
	}
	inside := width - 4
	label := " " + clean(title) + " "
	label = ansi.Truncate(label, width-2, "…")
	lines := []string{style.Render("┌" + label + strings.Repeat("─", max(0, width-2-ansi.StringWidth(label))) + "┐")}
	room := height - 2
	offset = min(max(0, offset), max(0, len(content)-room))
	for i := 0; i < room; i++ {
		s := ""
		if offset+i < len(content) {
			s = content[offset+i]
		}
		lines = append(lines, style.Render("│")+" "+fit(s, inside)+" "+style.Render("│"))
	}
	bottom := strings.Repeat("─", width-2)
	if len(content) > room {
		hint := fmt.Sprintf(" %d-%d/%d · PgUp/PgDn ", offset+1, min(offset+room, len(content)), len(content))
		hint = ansi.Truncate(hint, width-2, "")
		bottom = strings.Repeat("─", max(0, width-2-ansi.StringWidth(hint))) + hint
	}
	return append(lines, style.Render("└"+bottom+"┘"))
}

func (m *model) renderLayout() string {
	w, h := max(1, m.width-2), max(1, m.height-1)
	if w < 58 || h < 18 {
		lines := []string{"FATIMA / rodeploy", "터미널을 60열 × 19행 이상으로 넓혀주세요.", "q / Ctrl+C 종료"}
		for i := range lines {
			lines[i] = fit(lines[i], w)
		}
		return strings.Join(lines[:min(h, len(lines))], "\n")
	}
	name := m.connection.name
	if name == "" {
		name = "rocontext"
	}
	brand := titleStyle.Render("FATIMA / rodeploy") + "  " + clean(name)
	endpoint := displayEndpoint(m.opts.Endpoint)
	status := m.connectionLabel()
	var header []string
	if w < 100 {
		header = append(header, fit(brand, max(1, w-ansi.StringWidth(status)-1))+" "+status, mutedStyle.Render(fit(endpoint, w)))
	} else {
		header = append(header, fit(brand+"  "+mutedStyle.Render(endpoint), max(1, w-ansi.StringWidth(status)-2))+"  "+status)
	}
	rule := ruleStyle.Render(strings.Repeat("─", w))
	header = append(header, rule)
	n := m.stage()
	stage := "BROWSE / Rollouts · 저장된 배포 조회"
	if n >= 0 {
		stage = fmt.Sprintf("%02d / %s · %s", n+1, stages[n], stageDescriptions[n])
	} else if n == -2 {
		stage = "Connection · 접속·인증 오류"
		if m.booting {
			stage = "Connection · 재시도 중"
		}
	}
	stageStyle := activeStyle
	if m.view == "legacy" && m.legacy.State != "" {
		stage = "LEGACY / " + m.legacyHeading()
		stageStyle = m.legacyStyle().Bold(true).Background(lipgloss.Color("236"))
	}
	header = append(header, stageStyle.Render(fit(stage, w)), rule)
	footer := m.footer(w)
	workHeight := h - len(header) - len(footer)
	railWidth := 17
	if w < 90 {
		railWidth = 15
	} else if w >= 110 {
		railWidth = 24
	}
	bodyWidth := w - railWidth - 3
	body := m.renderBody(bodyWidth, workHeight)
	rail := m.rail(railWidth, workHeight)
	lines := append([]string{}, header...)
	for i := 0; i < workHeight; i++ {
		lines = append(lines, rail[i]+" "+ruleStyle.Render("│")+" "+body[i])
	}
	lines = append(lines, footer...)
	for i := range lines {
		lines[i] = fit(lines[i], w)
	}
	return strings.Join(lines, "\n")
}

func (m *model) renderBody(width, height int) []string {
	if m.view == "connection" {
		lines := []string{field("Context", m.connection.name), field("서버 주소", displayEndpoint(m.opts.Endpoint)), field("사용자", m.connection.config.User), ""}
		if m.connection.reached {
			lines = append(lines, goodStyle.Render("✓ Jupiter 도달 · 기능 확인"))
		}
		if m.booting {
			lines = append(lines, alertStyle.Render("설정을 다시 읽고 접속·인증을 확인하고 있습니다…"))
		} else if e := m.connection.failure; e != nil {
			lines = append(lines, errorStyle.Render("! "+e.Code), clean(e.Message), "", "설정 수정 후 r로 다시 읽고 재시도하세요.", "업로드·배포 요청은 제출되지 않았습니다.")
			if m.diagnostics {
				lines = append(lines, "", titleStyle.Render("DIAGNOSTICS"), clean(e.Detail), "", "종료 시 stderr에 위 오류 요약을 남깁니다.")
			}
		}
		return drawPanel("CONNECTION", wrapLines(lines, width-4), width, height, m.scroll, true)
	}
	if m.confirm != "" {
		lines := wrapLines(m.confirmationLines(), width-4)
		m.scroll = min(m.scroll, max(0, len(lines)-(height-2)))
		return drawPanel("CONFIRM · "+m.confirm, lines, width, height, m.scroll, true)
	}
	c := m.screen(width - 4)
	top := []string{fit(titleStyle.Render(c.title), width), fit(mutedStyle.Render(c.description), width)}
	available := height - len(top)
	if c.rows == nil && c.rowHeader == "" {
		details := wrapLines(c.details, width-4)
		end := max(0, len(details)-(available-2))
		if m.view == "legacy" && m.follow {
			m.scroll = end
		}
		m.scroll = min(m.scroll, end)
		return append(top, drawPanel(c.detailTitle, details, width, available, m.scroll, true)...)
	}
	listHeight := min(max(5, available/3), max(5, len(c.rows)+3))
	artifactList := m.view == "artifacts" || m.view == "upload" || m.view == "legacy"
	if artifactList {
		// Five items plus the column header and the two panel borders.
		listHeight = min(listHeight, 8)
	}
	listHeight = min(listHeight, available-4)
	room := max(1, listHeight-3)
	start := max(0, c.selected-room/2)
	start = min(start, max(0, len(c.rows)-room))
	list := []string{mutedStyle.Render(c.rowHeader)}
	for i := start; i < min(len(c.rows), start+room); i++ {
		line := "  " + c.rows[i]
		if i == c.selected {
			line = activeStyle.Render(fit("› "+c.rows[i], width-4))
		}
		list = append(list, line)
	}
	if len(c.rows) == 0 {
		list = append(list, mutedStyle.Render("목록이 비어 있습니다."))
	}
	listTitle := fmt.Sprintf("%s · %d", c.listTitle, len(c.rows))
	if artifactList && len(c.rows) > room {
		listTitle = fmt.Sprintf("%s · %d–%d / %d", c.listTitle, start+1, min(len(c.rows), start+room), len(c.rows))
	}
	top = append(top, drawPanel(listTitle, list, width, listHeight, 0, !m.focusDetail)...)
	detailHeight := available - listHeight
	details := wrapLines(c.details, width-4)
	if m.view == "watch" && m.follow && m.focusDetail {
		m.scroll = max(0, len(details)-(detailHeight-2))
	}
	m.scroll = min(m.scroll, max(0, len(details)-(detailHeight-2)))
	return append(top, drawPanel(c.detailTitle, details, width, detailHeight, m.scroll, m.focusDetail)...)
}

func (m *model) screen(width int) screenContent {
	if m.view == "activity" {
		return m.activityScreen()
	}
	c := screenContent{selected: m.cursor}
	switch m.view {
	case "artifacts":
		c.title, c.description = "Artifacts", "Jupiter에 업로드된 배포본 · 등록 시간은 PC 로컬 시간"
		c.listTitle, c.detailTitle = "ARTIFACT LIST", "SELECTED ARTIFACT / BUILD"
		columns := func(process, user, size, at string) string {
			if width < 45 {
				return fit(process, width-22) + " " + at
			}
			if width < 80 {
				return fit(process, max(9, width-32)) + " " + fit(size, 9) + " " + at
			}
			return fit(process, width-50) + " " + fit(user, 16) + " " + fit(size, 9) + " " + at
		}
		c.rowHeader = "  " + columns("PROCESS", "UPLOADER", "SIZE", "REGISTERED AT")
		for _, a := range m.artifacts {
			name := clean(a.Process)
			if a.ExpiresAt <= time.Now().Unix() {
				name = "! " + name
			}
			c.rows = append(c.rows, columns(name, clean(a.UploadedBy), bytesText(a.Size), localTimestamp(a.UploadedAt)))
		}
		if m.cursor >= 0 && m.cursor < len(m.artifacts) {
			a := m.artifacts[m.cursor]
			platforms := append([]string{}, a.Platforms...)
			sort.Strings(platforms)
			c.details = []string{field("파일", a.Filename), field("크기", bytesText(a.Size)), field("업로더", a.UploadedBy), field("플랫폼", strings.Join(platforms, ", ")), field("등록 시간", localTimestamp(a.UploadedAt))}
			if a.ExpiresAt <= time.Now().Unix() {
				c.detailTitle += " · EXPIRED"
				c.details = append(c.details, errorStyle.Render("EXPIRED · 새 배포에는 다시 업로드해야 합니다."))
			}
			c.details = append(c.details, "", titleStyle.Render("BUILD"))
			c.details = append(c.details, buildFields(a.BuildJson, m.showIdentity)...)
			id := shortID(a.Id)
			if m.showIdentity {
				id = a.Id
			}
			c.details = append(c.details, "", field("Artifact ID", id))
			if m.showIdentity {
				c.details = append(c.details, field("SHA-256", a.Sha256))
			}
		} else {
			c.details = []string{"u를 눌러 로컬 FAR를 선택하고 업로드하세요."}
		}
	case "upload", "legacy":
		if m.view == "legacy" && m.legacy.State != "" {
			c.title = m.legacyStyle().Bold(true).Render(m.legacyHeading())
			c.description = m.legacy.Summary
			c.detailTitle = "LEGACY LOG / " + m.legacy.State
			c.details = []string{field("FAR", filepath.Base(m.input))}
			if m.opts.Group != "" {
				c.details = append(c.details, field("그룹", m.opts.Group))
			}
			if m.opts.First != "" {
				c.details = append(c.details, field("대상", m.opts.First))
			}
			c.details = append(c.details, "", "설치·기동 결과는 rodis/로그에서 확인하세요.", "", clean(m.legacyLog))
			return c
		}
		c.title, c.description = "Upload FAR", "Path: "+clean(m.input)
		if m.editingPath {
			path := clean(m.input)
			if ansi.StringWidth(path) > width-4 {
				path = "…" + ansi.Cut(path, ansi.StringWidth(path)-(width-5), ansi.StringWidth(path))
			}
			c.description = "Path: " + path + "▏"
		}
		c.listTitle, c.detailTitle = "LOCAL FAR / 최근 수정 순", "SELECTED FAR / BUILD"
		if m.view == "legacy" {
			c.title = "Legacy deployment"
		} else {
			c.title += "   " + skipUploadBadge.Render(" a ") + " 업로드 없이 기존 배포본 선택"
		}
		if m.busy && m.bytes > 0 {
			c.detailTitle = "TRANSFER / SERVER RESPONSE"
		}
		columns := func(name, size, at string) string {
			if width < 45 {
				return fit(name, width-22) + " " + at
			}
			return fit(name, width-32) + " " + fit(size, 9) + " " + at
		}
		c.rowHeader = "  " + columns("FILE", "SIZE", "UPDATED AT")
		for _, f := range m.locals.Files {
			c.rows = append(c.rows, columns(filepath.Base(f.Path), bytesText(f.Size), localTimestamp(f.Modified.Unix())))
		}
		if m.busy {
			c.details = append(c.details, titleStyle.Render("Mac → Jupiter"), bar(m.bytes, m.total), clean(m.notice), "")
		}
		if m.previewPending {
			c.details = append(c.details, "FAR 정보를 읽고 있습니다…")
		} else if p := m.preview; p != nil {
			c.details = append(c.details, field("파일", filepath.Base(p.File.Path)))
			if p.Info != nil {
				c.details = append(c.details, field("프로세스", p.Info.Process))
			}
			if !p.File.Modified.IsZero() {
				c.details = append(c.details, field("크기", bytesText(p.File.Size)), field("파일 수정", localTimestamp(p.File.Modified.Unix())))
			}
			if p.Err != nil {
				c.details = append(c.details, errorStyle.Render(clean(p.Err.Error())))
			}
			if p.Info != nil {
				c.details = append(c.details, field("플랫폼", strings.Join(p.Info.Platforms, ", ")), "", titleStyle.Render("BUILD"))
				c.details = append(c.details, buildFields(p.Info.BuildJSON, m.showIdentity)...)
			}
			c.details = append(c.details, "", field("전체 경로", p.File.Path))
		} else {
			c.details = append(c.details, "p를 눌러 FAR 경로를 직접 입력할 수 있습니다.")
		}
		c.details = append(c.details, "", field("검색 위치", m.locals.Root))
		if m.locals.Warning != "" {
			c.details = append(c.details, clean(m.locals.Warning))
		}
	case "targets":
		c.title, c.description = "Choose the first package", "선택한 패키지의 그룹 전체가 동일 artifact에 고정됩니다."
		c.listTitle, c.detailTitle, c.rowHeader = "PACKAGES", "ROLLOUT PLAN", "  GROUP / PACKAGE / API"
		for _, t := range m.targets {
			support := "v2"
			if !t.Supported {
				support = "UNAVAILABLE"
				if t.Legacy {
					support = "legacy"
				}
			}
			c.rows = append(c.rows, clean(t.Group)+" / "+clean(t.PackageId)+" / "+support)
		}
		if m.cursor < len(m.targets) && m.cursor >= 0 {
			t := m.targets[m.cursor]
			c.details = []string{field("그룹", t.Group), field("첫 패키지", t.PackageId), field("플랫폼", t.Platform)}
			if m.artifact != nil {
				c.details = append(c.details, field("배포본", m.artifact.Filename), field("Artifact ID", m.artifact.Id))
			}
			c.details = append(c.details, "", titleStyle.Render("FIXED TARGETS"))
			for _, target := range m.targets {
				if target.Group == t.Group {
					c.details = append(c.details, "  "+clean(target.PackageId))
				}
			}
			c.details = append(c.details, "", "첫 패키지 배포 → 서비스 확인 → 나머지 순차 배포", clean(t.Reason))
		}
	case "rollouts":
		c.title, c.description = "Rollouts", "저장된 배포를 선택하면 진행 상황에 다시 연결합니다."
		c.listTitle, c.detailTitle, c.rowHeader = "SAVED ROLLOUTS", "SELECTED ROLLOUT", "  STATE / GROUP / PROCESS"
		for _, p := range m.rollouts {
			c.rows = append(c.rows, fit(clean(rolloutLabel(p)), 12)+" "+clean(p.Group)+" / "+clean(p.Artifact.Process))
		}
		if m.cursor >= 0 && m.cursor < len(m.rollouts) {
			p := m.rollouts[m.cursor]
			c.details = []string{field("Rollout", p.Id), field("상태", rolloutLabel(p)), field("그룹", p.Group), field("프로세스", p.Artifact.Process), field("배포자", p.CreatedBy), field("등록 시간", localTimestamp(p.CreatedAt)), "", clean(p.Message)}
		}
	case "watch":
		c.title, c.description = "Deployment progress", "실시간 진행 상황 · Tab 패키지/이벤트 전환"
		c.listTitle, c.detailTitle, c.rowHeader = "PACKAGE STATUS", "CURRENT PACKAGE", "  PACKAGE / STATE"
		if p := m.rollout; p != nil {
			c.title = rolloutLabel(p) + " / " + p.Artifact.Process
			c.description = p.Group + " · " + shortID(p.Id)
			for _, t := range p.Targets {
				c.rows = append(c.rows, clean(t.Target.PackageId)+"  "+clean(t.Operation.State))
			}
			if m.cursor >= 0 && m.cursor < len(p.Targets) {
				t := p.Targets[m.cursor]
				c.detailTitle = t.Target.PackageId
				c.details = append(lifecycleDetails(p), clean(p.Message))
				if t.Operation.Error != "" {
					c.details = append(c.details, "서버 오류: "+clean(t.Operation.Error))
				}
				if t.SentBytes > 0 {
					c.details = append(c.details, "Jupiter → Juno "+bar(t.SentBytes, t.TotalBytes))
				}
				if p.State == "WAITING" {
					c.details = append(c.details, "", alertStyle.Render("첫 배포 완료 · 서비스 상태 확인 후 c로 나머지 승인"))
				}
				for _, event := range t.Operation.Events {
					line := fmt.Sprintf("%s %-9s %s", time.UnixMilli(event.At).Local().Format("15:04:05"), event.Stage, event.State)
					detail := clean(event.Message)
					if event.Total > 0 {
						unit := "bytes"
						if event.Stage == "goaway" || event.Stage == "shutdown" || event.Stage == "start" {
							unit = "s"
						}
						detail += fmt.Sprintf(" (%d/%d %s)", event.Current, event.Total, unit)
					}
					c.details = append(c.details, titleStyle.Render(line), "  "+detail)
				}
				if t.Operation.Error != "" {
					c.details = append(c.details, errorStyle.Render(clean(t.Operation.Error)))
				}
				if m.showIdentity {
					c.details = append(c.details, field("Operation", t.Operation.Id), field("Artifact ID", p.Artifact.Id), field("SHA-256", p.Artifact.Sha256))
				}
			}
		}
	}
	if m.diagnostics && m.lastError != "" {
		c.details = append(c.details, "", errorStyle.Render("ERROR"), clean(m.lastError))
	}
	return c
}

func (m *model) confirmationLines() []string {
	lines := []string{}
	switch m.confirm {
	case "actions", "conflict":
		if m.confirm == "conflict" {
			lines = append(lines, "기존 배포가 같은 대상을 사용 중입니다.", "이번 새 배포는 시작되지 않았습니다.", "")
		}
		if m.rollout != nil {
			lines = append(lines, lifecycleDetails(m.rollout)...)
			lines = append(lines, "")
		}
		for i, label := range m.actionLabels() {
			prefix := "  "
			if i == m.menuIndex {
				prefix = "› "
			}
			lines = append(lines, prefix+label)
		}
		return append(lines, "", "↑↓ 선택 · Enter 실행 · Esc 상세로")

	case "create":
		if m.draft != nil && m.artifact != nil {
			lines = []string{field("프로세스", m.artifact.Process), field("그룹", m.draft.Group), field("첫 패키지", m.draft.FirstPackageId), field("Artifact ID", m.artifact.Id), "", titleStyle.Render("FIXED TARGETS"), clean(strings.Join(m.draft.PackageIds, "\n")), ""}
			if len(m.draft.PackageIds) > 1 {
				lines = append(lines, "첫 패키지를 배포한 뒤 나머지는 승인을 기다립니다.")
			} else {
				lines = append(lines, "대상이 하나이므로 해당 패키지 배포 후 완료합니다.")
			}
			lines = append(lines, "CLI 종료 시 미실행 대상 취소 · 이미 실행한 대상은 결과까지 확인", field("Request ID", m.draft.RequestId))
		}
	case "legacy-group":
		lines = []string{alertStyle.Render("이 그룹은 신규(v2) 방식으로 배포할 수 없습니다"), "", field("그룹", m.legacyGroup), "legacy Juno 패키지:"}
		for _, id := range m.legacyTargets {
			lines = append(lines, "  - "+clean(id))
		}
		return append(lines, "",
			"배포 계획은 그룹 전체를 포함하므로, legacy 패키지가 하나라도 있으면 그룹 전체를 기존 HTTP 방식으로 배포합니다.",
			"legacy 방식은 FAR 경로를 다시 선택하며 단계별 진행과 나머지 순차 승인이 없습니다.",
			"신규 방식으로 배포하려면 위 패키지의 Juno를 v2로 업데이트한 뒤 다시 시도하세요.",
			"", alertStyle.Render("Enter legacy 방식으로 진행 / Esc 취소하고 대상 선택으로"))
	case "legacy":
		lines = []string{"기존 HTTP 배포를 실행합니다.", field("FAR", m.input), field("그룹", m.opts.Group), field("패키지", m.opts.First), "서버 응답에는 Juno의 상세 설치·기동 상태가 포함되지 않습니다."}
	case "continue":
		lines = []string{"첫 패키지의 실제 서비스 상태를 확인했습니까?", "같은 artifact로 나머지 패키지를 순서대로 배포합니다."}
	case "cancel":
		lines = []string{"이후 패키지에 대한 배포를 중단합니다.", "이미 Juno가 접수한 작업은 별도로 결과를 확인합니다."}
	case "resume":
		lines = []string{"동일 operation ID로 불확실한 결과를 다시 확인합니다."}
	case "retry":
		lines = []string{"이전 실패 기록을 보존하고 실패한 패키지를 다시 시도합니다."}
	}
	return append(lines, "", alertStyle.Render("Enter 실행 / Esc 취소"))
}

func (m *model) footer(width int) []string {
	state, message := "READY", m.notice
	if m.client != nil {
		if notice := m.client.managementNotice(); notice != "" {
			message = notice
		}
	}
	if m.connectionLost {
		message = "연결 복구 중 · 현재 화면은 마지막 수신 상태"
	}
	if m.view == "legacy" && m.legacy.State != "" {
		state = m.legacy.State
	}
	if m.view == "watch" && m.rollout != nil {
		state = m.rollout.State
	}
	if m.lastError != "" {
		state = "ERROR"
	}
	if m.editingPath {
		state = "PATH"
	}
	if m.confirm != "" {
		state = "CONFIRM"
	}
	if m.busy {
		state = "WORKING"
		if m.view == "legacy" && m.legacy.State != "" {
			state = m.legacy.State
		}
	}
	if m.booting {
		state = "CONNECTING"
	} else if m.connection.failure != nil {
		state = m.connection.failure.Code
	}
	keys := "↑↓ 선택  Enter 진행  i ID/해시  r 새로고침"
	global := "q 종료  u 새 배포  a 배포본  l 기존 작업  Tab 영역"
	switch {
	case m.view == "connection":
		keys, global = "r 설정 재조회·재시도  d 상세  q 종료", "접속·인증 오류로 업로드·배포 요청은 제출되지 않았습니다."
	case m.booting:
		keys, global = "q 종료", "Jupiter 연결·인증을 확인하고 있습니다."
	case m.editingPath:
		keys, global = "Enter 경로 확인  Esc 입력 취소  Ctrl+U 지우기", "Ctrl+C 종료"
	case m.view == "legacy" && m.legacy.State != "":
		keys, global = "q 종료  n 새 배포  ↑↓ 로그  f 최신", "설치·기동 결과는 rodis/로그에서 확인하세요."
		if m.busy {
			keys = "q 종료  ↑↓ 로그  f 최신"
		}
	case m.busy:
		keys, global = "진행 중…  Ctrl+C 종료", "서버가 이미 접수한 배포는 화면을 닫아도 계속됩니다."
		if m.view == "upload" || m.view == "legacy" {
			global = "업로드 전송 중 종료하면 전송이 중단될 수 있습니다."
		}
	case m.confirm == "actions" || m.confirm == "conflict":
		keys, global = "↑↓ 선택  Enter 진행  Esc 상세로", "q 종료"
	case m.confirm != "":
		keys, global = "Enter 확인 후 실행  Esc 취소", "PgUp/PgDn 스크롤  q 종료"
	case m.view == "upload" || m.view == "legacy":
		keys = "↑↓ FAR 선택  Enter 업로드  a 기존 배포본  p 직접 경로  r 목록 갱신"
		if m.view == "legacy" {
			keys = "↑↓ FAR 선택  p 직접 경로  Enter 업로드  r 목록 갱신"
			global = "Tab 영역  PgUp/PgDn 스크롤  q 종료 · HTTP legacy"
		}
	case m.view == "watch":
		keys = "Enter 동작 메뉴  ↑↓ 스크롤  x 취소  q 종료"
		if m.rollout != nil {
			if m.rollout.State == "WAITING" {
				keys = "Enter 동작 메뉴  c 나머지 승인  x 취소  q 종료"
			} else if m.rollout.State == "FAILED" || m.rollout.State == "ATTENTION" {
				keys = "Enter 동작 메뉴  r 재조회/재시도  q 종료"
			} else if m.rollout.State == "SUCCEEDED" || m.rollout.State == "CANCELLED" {
				keys = "Enter 동작 메뉴  ↑↓ 스크롤  q 종료"
			}
		}
	}
	if m.client != nil && m.client.ownsDeployment() && m.confirm == "" {
		global = "q 종료: 미실행 취소 · 실행 중 대상은 결과 확인"
	}
	if m.focusDetail && m.confirm == "" && !m.busy {
		message = "[상세] " + message
	}
	status := goodStyle.Render(state)
	if m.view == "legacy" && m.legacy.State != "" {
		status = m.legacyStyle().Bold(true).Render(state)
	} else if m.connection.failure != nil || m.lastError != "" {
		status = errorStyle.Render(state)
	} else if state == "WAITING" || state == "CONFIRM" {
		status = alertStyle.Render(state)
	}
	return []string{ruleStyle.Render(strings.Repeat("─", width)), fit(status+"  "+clean(message), width), fit(titleStyle.Render(keys), width), fit(mutedStyle.Render(global), width)}
}
