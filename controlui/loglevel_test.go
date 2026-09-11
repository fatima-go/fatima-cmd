package controlui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-core/opm/api"
)

func TestLogLevelArguments(t *testing.T) {
	o, err := parse("rolog", []string{"worker", "DEBUG"})
	if err != nil || o.Process != "worker" || o.Level != "debug" || !o.Plain {
		t.Fatal(o, err)
	}
	for _, args := range [][]string{{"worker"}, {"worker", "fatal"}, {"worker", "info", "extra"}, {"--watch", "id"}} {
		if _, err := parse("rolog", args); err == nil {
			t.Fatal(args)
		}
	}
	if o, err = parse("rolog", nil); err != nil || o.Plain || o.Process != "" {
		t.Fatal(o, err)
	}
}

func logLevelFixture() model {
	return model{ctx: context.Background(), opts: Options{Command: "rolog"}, stage: "select", width: 132, height: 42,
		client: &Client{Target: &api.Target{PackageId: "host:default"}},
		levels: &api.LogLevelCatalog{Entries: []*api.LogLevelEntry{{Process: "worker", Group: "SVC", Level: "info"}, {Process: "juno", Group: "OPM", Level: "debug", Opm: true}}}}
}

func TestLogLevelSelectionAppliesChosenLevel(t *testing.T) {
	m := logLevelFixture()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // juno, worker
	next, cmd := next.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd != nil || m.stage != "level" || logLevels[m.levelCursor] != "info" {
		t.Fatal("Enter must open the picker on the current level", m.stage, m.levelCursor)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	next, cmd = next.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd == nil || !m.busy {
		t.Fatal("choosing a level must apply it")
	}
	next, _ = m.Update(logLevelApplied{entry: &api.LogLevelEntry{Process: "worker", Level: "debug"}})
	m = next.(model)
	if m.stage != "select" || m.visibleLogLevels()[1].Level != "debug" || !strings.Contains(m.status, "DEBUG") {
		t.Fatal(m.stage, m.status)
	}
	m.stage = "level"
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || next.(model).stage != "select" {
		t.Fatal("Esc must cancel the picker without applying")
	}
}

func TestLogLevelPackageChoiceReconnects(t *testing.T) {
	m := model{ctx: context.Background(), opts: Options{Command: "rolog"}, stage: "package", width: 100, height: 30,
		client:   &Client{Target: &api.Target{PackageId: "패키지 선택"}},
		packages: &api.PackageCatalog{Packages: []*api.PackageEntry{{Target: &api.Target{PackageId: "a:default"}}, {Target: &api.Target{PackageId: "b:default"}}}}}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	next, cmd := next.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(model)
	if cmd == nil || m.client != nil || m.opts.Package != "b:default" {
		t.Fatal("package choice must reconnect to the chosen package", m.opts.Package)
	}
}

func TestLogLevelViewsFit(t *testing.T) {
	m := logLevelFixture()
	for _, size := range [][2]int{{60, 19}, {60, 31}, {80, 24}, {100, 30}, {132, 42}} {
		m.width, m.height = size[0], size[1]
		for _, stage := range []string{"select", "level"} {
			m.stage = stage
			assertFits(t, m)
		}
	}
	m.width, m.height, m.stage = 132, 42, "level"
	if v := m.View(); !strings.Contains(v, "TRACE") || !strings.Contains(v, "현재") {
		t.Fatal(v)
	}
}

func TestControlTargetsSkipProcessesWithNothingToDo(t *testing.T) {
	catalog := &api.ProcessCatalog{Processes: []*api.ProcessEntry{{Name: "alive", State: "ALIVE"}, {Name: "dead", State: "DEAD"}, {Name: "juno", State: "ALIVE"}}}
	for _, c := range []struct {
		command string
		cursor  int
		allowed bool
	}{{"rostart", 0, false}, {"rostart", 1, true}, {"rostop", 0, true}, {"rostop", 1, false}, {"rostop", 2, false}} {
		m := model{ctx: context.Background(), opts: Options{Command: c.command}, stage: "select", width: 100, height: 30, cursor: c.cursor,
			client: &Client{Target: &api.Target{PackageId: "host:default"}}, catalog: catalog}
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
		if selected := len(next.(model).opts.Targets) == 1; selected != c.allowed {
			t.Fatal(c, "space selection")
		}
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if review := next.(model).stage == "review"; review != c.allowed {
			t.Fatal(c, "enter review", next.(model).status)
		}
		assertFits(t, m)
	}
}

func TestRegistryListKeepsFocus(t *testing.T) {
	m := inventoryFixture("roproc")
	for _, key := range []tea.KeyType{tea.KeyTab, tea.KeyEnter} {
		next, cmd := m.Update(tea.KeyMsg{Type: key})
		if cmd != nil || next.(model).stage != "select" {
			t.Fatal("roproc must stay on the list", key)
		}
	}
}
