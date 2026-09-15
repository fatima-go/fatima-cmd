package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-cmd/share"
)

type localDone struct{ err error }
type localModel struct {
	home, name, stage, target, current, notice string
	names                                      []string
	revisions                                  []Revision
	cursor, width, height                      int
	chosen                                     Revision
	action                                     string
	busy                                       bool
	err                                        error
}

func newLocalModel(args []string) localModel {
	m := localModel{home: os.Getenv(share.EnvFatimaHome), stage: "process", width: 80, height: 24}
	m.names, m.err = localProcesses(m.home)
	if m.err == nil && len(args) > 0 {
		found := false
		for _, name := range m.names {
			if name == args[0] {
				found = true
			}
		}
		if !found {
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
					found = false
					for _, r := range m.revisions {
						if strings.EqualFold(r.revision, args[2]) {
							m.selectRevision(r)
							found = true
							break
						}
					}
					if !found {
						m.err = fmt.Errorf("리비전 %s를 찾을 수 없습니다", args[2])
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
func (m *localModel) openVersions() {
	m.notice = ""
	m.revisions, m.current, m.err = localRevisions(m.home, m.name)
	m.stage = "version"
	m.cursor = 0
}
func (m localModel) rows() []string {
	switch m.stage {
	case "process":
		return m.names
	case "action":
		return []string{"리비전 조회 / 전환", "다른 이름으로 복제"}
	case "version":
		var rows []string
		for _, r := range m.revisions {
			mark := ""
			if same, _ := sameRevision(m.current, r.dir); same {
				mark = " [현재]"
			}
			rows = append(rows, r.revision+mark+"  "+r.GetBuildSummary())
		}
		return rows
	}
	return nil
}
func (m localModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case localDone:
		m.busy = false
		m.err = v.err
		if errors.Is(v.err, errCurrentRevision) {
			m.openVersions()
			m.notice = errCurrentRevision.Error() + ". 다른 리비전을 선택하세요."
			return m, nil
		}
		if v.err == nil {
			m.stage = "result"
			if m.action == "version" {
				m.notice = "리비전 전환 완료 · rostart로 프로세스를 시작하세요"
			} else {
				m.notice = "복제 완료 · 아직 프로세스 등록과 시작은 하지 않았습니다. roproc add로 등록하세요"
			}
		}
	case tea.KeyMsg:
		key := v.String()
		if m.busy {
			return m, nil
		}
		if key == "ctrl+c" || (key == "q" && m.stage != "name") {
			return m, tea.Quit
		}
		if key == "esc" {
			m.err = nil
			m.notice = ""
			m.cursor = 0
			if m.stage == "process" {
				return m, tea.Quit
			}
			if m.stage == "action" {
				m.stage = "process"
			} else {
				m.stage = "action"
			}
			return m, nil
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
		case "up", "k":
			m.cursor = max(0, m.cursor-1)
		case "down", "j":
			m.cursor = min(max(0, len(m.rows())-1), m.cursor+1)
		case "r":
			m.err = nil
			if m.stage == "version" {
				m.openVersions()
			} else if m.stage == "process" {
				m.names, m.err = localProcesses(m.home)
				m.cursor = 0
			}
		case "enter":
			switch m.stage {
			case "process":
				if len(m.names) > 0 {
					m.name = m.names[m.cursor]
					m.stage = "action"
					m.cursor = 0
					m.err = nil
				}
			case "action":
				m.err = nil
				if m.cursor == 0 {
					m.openVersions()
				} else {
					m.stage = "name"
					m.action = "dup"
					m.target = ""
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
				m.notice = ""
				m.stage = "process"
				m.cursor = 0
				m.names, m.err = localProcesses(m.home)
			}
		}
	}
	return m, nil
}
func (m localModel) View() string {
	width := max(20, m.width-2)
	lines := []string{"lcproc · 로컬 프로세스 관리", "환경: " + m.home, "프로세스: " + m.name, ""}
	switch m.stage {
	case "process":
		lines = append(lines, "관리할 프로세스를 선택하세요. 운영 프로세스는 제외됩니다.")
	case "action":
		lines = append(lines, "수행할 작업을 선택하세요.")
	case "version":
		lines = append(lines, "현재 리비전을 확인하고 전환할 버전을 선택하세요.")
	case "name":
		lines = append(lines, "복제할 새 프로세스 이름을 입력하세요.", "> "+m.target+"█", "실행 파일과 설정 파일을 복사합니다. 하위 디렉터리는 복사하지 않습니다.")
	case "confirm":
		if m.action == "version" {
			lines = append(lines, "리비전 전환 확인", m.name+" → "+m.chosen.revision, "프로세스가 중지된 경우에만 링크를 전환합니다.", "전환 후 자동으로 시작하지 않습니다.")
		} else {
			lines = append(lines, "프로세스 복제 확인", m.name+" → "+m.target, "실행 파일·설정 파일 복사 및 새 리비전 링크 생성", "원본 이름으로 시작하는 파일명은 새 이름으로 변경합니다.", "복제 후 roproc add로 등록해야 사용할 수 있습니다.")
		}
	case "result":
		lines = append(lines, m.notice)
	}
	rows := m.rows()
	room := max(1, m.height-len(lines)-5)
	start := max(0, m.cursor-room+1)
	for i := start; i < min(len(rows), start+room); i++ {
		mark := "  "
		if i == m.cursor {
			mark = "▶ "
		}
		lines = append(lines, mark+rows[i])
	}
	if len(rows) == 0 && (m.stage == "process" || m.stage == "version") {
		lines = append(lines, "표시할 항목이 없습니다. r 새로고침")
	}
	if m.notice != "" && m.stage != "result" {
		lines = append(lines, "안내: "+m.notice)
	}
	if m.err != nil {
		lines = append(lines, "오류: "+m.err.Error())
	}
	help := "↑↓ 이동 · Enter 선택 · Esc 이전 · r 새로고침 · q 종료"
	if m.stage == "confirm" {
		help = "Enter 실행 · Esc 취소 · q 종료"
	}
	if m.stage == "name" {
		help = "Enter 확인 · Esc 취소 · Ctrl+C 종료"
	}
	if m.busy {
		help = "처리 중입니다. 완료 결과를 기다려 주세요."
	}
	if m.stage == "result" {
		help = "Enter 프로세스 목록 · q 종료"
	}
	lines = append(lines, "", help)
	for i, s := range lines {
		lines[i] = ansi.Truncate(strings.ReplaceAll(ansi.Strip(s), "\n", " "), width, "…")
	}
	return strings.Join(lines, "\n")
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
