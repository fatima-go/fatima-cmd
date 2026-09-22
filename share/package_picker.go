package share

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	filter                string
	filtering             bool
}

func (m packagePicker) Init() tea.Cmd { return nil }

// visible applies the incremental filter. Endpoint text is searchable so an
// operator can narrow by server address as well as by package name.
func (m packagePicker) visible() []PackageChoice {
	if m.filter == "" {
		return m.choices
	}
	needle := strings.ToLower(m.filter)
	var out []PackageChoice
	for _, p := range m.choices {
		if strings.Contains(strings.ToLower(p.ID+" "+p.Group+" "+p.Endpoint+" "+p.State), needle) {
			out = append(out, p)
		}
	}
	return out
}

func (m packagePicker) clamp() packagePicker {
	rows := len(m.visible())
	if m.cursor >= rows {
		m.cursor = max(0, rows-1)
	}
	return m
}

func (m packagePicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case tea.KeyMsg:
		if m.filtering {
			return m.editFilter(v)
		}
		key := v.String()
		switch key {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.cursor = max(0, m.cursor-1)
		case "down", "j":
			m.cursor = min(max(0, len(m.visible())-1), m.cursor+1)
		case "/":
			m.filtering = true
		case "enter":
			return m.choose(m.cursor)
		default:
			// Digits jump straight to a numbered row; the list is short enough
			// that 1-9 covers most endpoints without any cursor movement.
			if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
				return m.choose(int(key[0] - '1'))
			}
		}
	}
	return m.clamp(), nil
}

func (m packagePicker) choose(index int) (tea.Model, tea.Cmd) {
	rows := m.visible()
	if index < 0 || index >= len(rows) {
		return m.clamp(), nil
	}
	m.cursor = index
	m.selected = rows[index].ID
	return m, tea.Quit
}

// editFilter keeps Enter on the filter, matching the inventory screens where
// Enter applies the typed text and a second Enter acts on the selected row.
func (m packagePicker) editFilter(v tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch v.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.filtering, m.filter, m.cursor = false, "", 0
	case "enter":
		m.filtering = false
	case "backspace":
		r := []rune(m.filter)
		if len(r) > 0 {
			m.filter = string(r[:len(r)-1])
		}
		m.cursor = 0
	case "up":
		m.cursor = max(0, m.cursor-1)
	case "down":
		m.cursor = min(max(0, len(m.visible())-1), m.cursor+1)
	default:
		if v.Type == tea.KeyRunes {
			m.filter += string(v.Runes)
			m.cursor = 0
		}
	}
	return m.clamp(), nil
}

// Jupiter's legacy package list abbreviates the state to a single letter while
// the gRPC inventory sends the whole word. One vocabulary keeps the column
// readable and lets the shared renderer color it.
func PackageStateLabel(state string) string {
	switch state = strings.ToUpper(strings.TrimSpace(state)); state {
	case "A":
		return "ALIVE"
	case "D":
		return "DEAD"
	case "":
		return "UNKNOWN"
	}
	return state
}

// PackageHost drops the random juno path token, which never helps a choice.
func PackageHost(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return endpoint
	}
	return u.Host
}

// PackagePickerColumns spends the terminal width on the columns that
// distinguish the candidates, and drops HOST first when the window cannot hold
// it. Both pickers - standalone and in-TUI - lay out through this.
func PackagePickerColumns(width int) []SheetColumn {
	if width >= 88 {
		inner := width - 16 // five columns with cell padding and borders
		state := min(11, max(7, inner/5))
		group := min(14, max(7, inner/5))
		host := min(22, max(12, inner/3))
		return []SheetColumn{{Name: "#", Width: 1}, {Name: "PACKAGE", Width: inner - 1 - group - host - state}, {Name: "GROUP", Width: group}, {Name: "HOST", Width: host}, {Name: "STATE", Width: state}}
	}
	inner := width - 13 // four columns with cell padding and borders
	state := min(11, max(7, inner/5))
	group := min(14, max(7, inner/5))
	return []SheetColumn{{Name: "#", Width: 1}, {Name: "PACKAGE", Width: inner - 1 - group - state}, {Name: "GROUP", Width: group}, {Name: "STATE", Width: state}}
}

// PackageRow renders one candidate for a table built from
// PackagePickerColumns, so the two package pickers cannot drift apart.
func PackageRow(columns []SheetColumn, index, cursor int, p PackageChoice) []string {
	mark := "  "
	if index == cursor {
		mark = "▶ "
	}
	number := "·" // only the numbered rows answer a digit key
	if index < 9 {
		number = fmt.Sprint(index + 1)
	}
	row := []string{number, mark + p.ID, p.Group}
	if len(columns) == 5 {
		row = append(row, PackageHost(p.Endpoint))
	}
	return append(row, PackageStateLabel(p.State))
}

func (m packagePicker) View() string {
	if m.width < 60 || m.height < 12 {
		return "터미널을 60열 × 12행 이상으로 늘려 주세요. q 취소"
	}
	width := m.width - 2
	columns := PackagePickerColumns(width)
	wide := len(columns) == 5
	rows := m.visible()
	cells := make([][]string, 0, len(rows))
	for i, p := range rows {
		cells = append(cells, PackageRow(columns, i, m.cursor, p))
	}
	title := fmt.Sprintf("패키지 선택 · %d개", len(m.choices))
	cursor := m.cursor
	if m.filter != "" {
		title = fmt.Sprintf("패키지 선택 · %d / %d개 · 필터 %q", len(rows), len(m.choices), m.filter)
	}
	if len(cells) == 0 {
		cells, cursor = [][]string{{"", "조건에 맞는 패키지가 없습니다"}}, -1
	}
	footer := "↑↓ 이동 · Enter 선택 · 1-9 바로 선택 · / 필터 · Esc 취소"
	if !wide {
		footer = "↑↓ · Enter 선택 · 1-9 · / 필터 · Esc"
	}
	head := SheetBorder.Render("작업할 패키지를 선택하세요.")
	if m.filtering {
		head, footer = SheetHeading.Render("필터> ")+m.filter+"█", "Enter 필터 적용 · Esc 필터 해제"
	}
	// A short list keeps a short table: padding it out to the window would only
	// add empty striped rows under the candidates.
	height := min(m.height-1, len(cells)+6)
	return head + "\n" + SheetView(title, columns, cells, cursor, width, height, true, footer)
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

// PreferClientPackages narrows ambiguous routing to this machine's addresses.
// Wildcard/loopback endpoints are not proof of locality on a remote server.
func PreferClientPackages(choices []PackageChoice) ([]PackageChoice, error) {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("클라이언트 IP 확인 실패: %w", err)
	}
	var ips []net.IP
	for _, address := range addresses {
		ip, _, err := net.ParseCIDR(address.String())
		if err == nil && !ip.IsLoopback() && !ip.IsUnspecified() {
			ips = append(ips, ip)
		}
	}
	return preferPackageIPs(choices, ips), nil
}
func preferPackageIPs(choices []PackageChoice, ips []net.IP) []PackageChoice {
	var matches []PackageChoice
	for _, p := range choices {
		u, err := url.Parse(p.Endpoint)
		if err != nil {
			continue
		}
		ip := net.ParseIP(u.Hostname())
		if ip == nil {
			continue
		}
		for _, local := range ips {
			if ip.Equal(local) {
				matches = append(matches, p)
				break
			}
		}
	}
	if len(matches) > 0 {
		return matches
	}
	return choices
}

// An HTTP server may return the first package matching the peer IP. Keep all
// packages on that IP as candidates instead of silently accepting the first.
func packagesAtEndpointIP(choices []PackageChoice, endpoint string) []PackageChoice {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil
	}
	ip := net.ParseIP(u.Hostname())
	if ip == nil {
		return nil
	}
	var matches []PackageChoice
	for _, p := range choices {
		target, err := url.Parse(p.Endpoint)
		if err == nil && ip.Equal(net.ParseIP(target.Hostname())) {
			matches = append(matches, p)
		}
	}
	return matches
}
