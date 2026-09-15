package controlui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-opm/api"
)

func historyFixture() model {
	return model{ctx: context.Background(), opts: Options{Command: "rohis"}, stage: "select", width: 132, height: 42,
		client: &Client{Target: &api.Target{PackageId: "host:default"}},
		history: &api.HistoryList{Records: []*api.DeploymentRecord{
			{Process: "worker", Group: "SVC", DeployedAt: 1789000000000, BuildUser: "dave", GitBranch: "main", GitCommit: "0123456789abcdef", GitMessage: "fix worker\n\nlong body"},
			{Process: "sample", Group: "SVC", DeployedAt: 1789100000000, BuildUser: "jin", GitBranch: "feature/한글-branch", GitCommit: "abcdef0123456789", GitMessage: "second deploy"},
			{Process: "sample", Group: "SVC", DeployedAt: 1788900000000, BuildUser: "jin", GitBranch: "main", GitCommit: "fedcba9876543210", GitMessage: "first deploy"},
		}}}
}

func TestHistoryArguments(t *testing.T) {
	for _, args := range [][]string{{"sample"}, {"-g", "SVC"}, {"-a"}} {
		if o, err := parse("rohis", args); err != nil || !o.Plain {
			t.Fatal(args, o, err)
		}
	}
	if o, err := parse("rohis", nil); err != nil || o.Plain {
		t.Fatal(o, err)
	}
	if _, err := parse("rohis", []string{"--watch", "id"}); err == nil {
		t.Fatal("rohis accepted an operation ID")
	}
}

func TestHistoryBrowsing(t *testing.T) {
	m := historyFixture()
	if v := m.View(); !strings.Contains(v, "배포 이력 · sample (2)") || !strings.Contains(v, "abcdef01") {
		t.Fatal(v)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd != nil || m.stage != "records" {
		t.Fatal(m.stage)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = next.(model)
	if m.histCursor != 1 || !strings.Contains(m.View(), "first deploy") {
		t.Fatal(m.histCursor, m.View())
	}
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || next.(model).stage != "select" {
		t.Fatal("Esc did not return to the process list")
	}
}

func TestHistoryViewsFit(t *testing.T) {
	m := historyFixture()
	for _, size := range [][2]int{{60, 19}, {60, 31}, {80, 24}, {100, 30}, {132, 42}, {172, 50}} {
		m.width, m.height = size[0], size[1]
		for _, stage := range []string{"select", "records"} {
			m.stage = stage
			assertFits(t, m)
		}
	}
	m.history.Records = nil
	m.stage = "select"
	assertFits(t, m)
}

func TestHistoryGroupFilter(t *testing.T) {
	records := historyFixture().history.Records
	if n := len(filterHistory(records, Options{Group: "svc"})); n != 3 {
		t.Fatal(n)
	}
	if n := len(filterHistory(records, Options{Group: "ENG"})); n != 0 {
		t.Fatal(n)
	}
}
