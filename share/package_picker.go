package share

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-cmd/domain"
	"golang.org/x/term"
)

type PackageChoice struct{ ID, Group, Endpoint, State string }

func PackagePromptAvailable() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// ChoosePackage never chooses an arbitrary server when more than one is registered.
// Diagnostic output goes to stderr so a report's stdout remains usable.
func ChoosePackage(choices []PackageChoice, interactive bool) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("등록된 패키지가 없습니다. ropack으로 등록 상태를 확인하세요")
	}
	if len(choices) == 1 {
		if interactive {
			fmt.Fprintf(os.Stderr, "선택된 패키지: %s\n", choices[0].ID)
		}
		return choices[0].ID, nil
	}
	if !interactive {
		return "", fmt.Errorf("등록된 패키지가 %d개입니다. ropack으로 목록을 확인한 뒤 -p host:package로 지정하세요", len(choices))
	}
	choices = append([]PackageChoice(nil), choices...)
	sort.Slice(choices, func(i, j int) bool { return choices[i].ID < choices[j].ID })
	m, err := tea.NewProgram(packagePicker{choices: choices, width: 80, height: 24}, tea.WithAltScreen(), tea.WithOutput(os.Stderr)).Run()
	if err != nil {
		return "", err
	}
	selected := m.(packagePicker).selected
	if selected == "" {
		return "", fmt.Errorf("패키지 선택을 취소했습니다. 작업을 실행하지 않았습니다")
	}
	fmt.Fprintf(os.Stderr, "선택된 패키지: %s\n", selected)
	return selected, nil
}

type packagePicker struct {
	choices               []PackageChoice
	cursor, width, height int
	selected              string
}

func (m packagePicker) Init() tea.Cmd { return nil }
func (m packagePicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case tea.KeyMsg:
		switch v.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.cursor = max(0, m.cursor-1)
		case "down", "j":
			m.cursor = min(len(m.choices)-1, m.cursor+1)
		case "enter":
			if len(m.choices) > 0 {
				m.selected = m.choices[m.cursor].ID
				return m, tea.Quit
			}
		}
	}
	return m, nil
}
func (m packagePicker) View() string {
	lines := []string{"패키지 선택", "작업할 패키지를 선택하세요.", ""}
	room := max(1, m.height-6)
	start := max(0, m.cursor-room+1)
	for i := start; i < min(len(m.choices), start+room); i++ {
		p := m.choices[i]
		marker := "  "
		if i == m.cursor {
			marker = "▶ "
		}
		lines = append(lines, marker+p.ID+"  "+p.Group+"  "+p.Endpoint+"  "+p.State)
	}
	lines = append(lines, "", "↑↓ 이동 · Enter 선택 · Esc/q 취소")
	for i, line := range lines {
		lines[i] = ansi.Truncate(strings.ReplaceAll(ansi.Strip(line), "\n", " "), max(1, m.width-2), "…")
	}
	return strings.Join(lines, "\n")
}

func legacyPackageChoices(flags FatimaCmdFlags) ([]PackageChoice, error) {
	_, data, err := CallFatimaApi(flags.BuildJupiterServiceUrl("/pack/v1"), flags, nil)
	if err != nil {
		return nil, err
	}
	var envelope map[string]interface{}
	if err = json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	if _, ok := envelope["summary"]; !ok {
		return nil, fmt.Errorf("패키지 목록 응답에 summary가 없습니다")
	}
	var response domain.RopackResp
	if err = json.Unmarshal(data, &response); err != nil {
		return nil, err
	}
	var choices []PackageChoice
	for _, group := range response.Summary.Deployment {
		for _, p := range group.Deploy {
			choices = append(choices, PackageChoice{ID: p.Host + ":" + p.Name, Group: group.GroupName, Endpoint: p.Endpoint, State: p.Status})
		}
	}
	return choices, nil
}
