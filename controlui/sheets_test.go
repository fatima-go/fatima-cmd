package controlui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-core/opm/api"
)

func inventoryFixture(command string) model {
	m := model{ctx: context.Background(), opts: Options{Command: command}, stage: "select", width: 132, height: 42,
		client: &Client{Target: &api.Target{PackageId: "backend01:default"}}, catalog: &api.ProcessCatalog{}}
	for i := 0; i < 48; i++ {
		m.catalog.Processes = append(m.catalog.Processes, &api.ProcessEntry{
			Name: fmt.Sprintf("worker-%02d", i), Group: "SVC", State: "ALIVE", Pid: fmt.Sprint(2100 + i),
			StartedAt: "2026-09-10 15:01:02", Cpu: "0.3", Memory: "24M", Fd: "16", Threads: "8", Ic: "0", Index: int32(i),
		})
	}
	if command == "ropack" {
		m.packages = &api.PackageCatalog{}
		m.catalog = nil
		for i := 0; i < 48; i++ {
			m.packages.Packages = append(m.packages.Packages, &api.PackageEntry{
				Target: &api.Target{PackageId: fmt.Sprintf("host-%02d:default", i), Group: "backend01", Platform: "linux_arm64", Endpoint: "http://backend-01.internal.example.com:9180/a-long-path/"},
				State:  "ALIVE", Transport: "gRPC", Detail: "한글 상세 정보\t" + strings.Repeat("long endpoint information ", 8),
			})
		}
	}
	return m
}

func assertFits(t *testing.T, m model) {
	t.Helper()
	view := m.View()
	if len(strings.Split(view, "\n")) > m.height {
		t.Fatalf("height overflow %dx%d:\n%s", m.width, m.height, view)
	}
	for _, row := range strings.Split(view, "\n") {
		if ansi.StringWidth(row) > m.width {
			t.Fatalf("width overflow %dx%d: %q", m.width, m.height, row)
		}
	}
}

func TestInventorySheetsFitAndKeepTheSelectedRowVisible(t *testing.T) {
	for _, command := range []string{"rostop", "rostart", "roproc", "ropack"} {
		t.Run(command, func(t *testing.T) {
			m := inventoryFixture(command)
			for _, size := range [][2]int{{60, 19}, {60, 31}, {80, 24}, {80, 40}, {100, 30}, {120, 30}, {132, 42}, {172, 50}} {
				m.width, m.height = size[0], size[1]
				for _, stage := range []string{"select", "detail"} {
					m.stage = stage
					m.cursor = 47
					assertFits(t, m)
					if stage == "select" && !strings.Contains(m.View(), "GROUP") {
						t.Fatal("group column missing")
					}
				}
				m.stage, m.editing, m.input = "select", "filter", "한글\t검색"
				m.err = fmt.Errorf("connection lost: %s", strings.Repeat("long message ", 20))
				assertFits(t, m)
				m.editing, m.err = "", nil
			}
			m.width, m.height, m.stage = 132, 42, "select"
			if !strings.Contains(m.View(), "상세") || !strings.Contains(m.View(), "47") {
				t.Fatal("last selection or detail pane missing")
			}
		})
	}
}

func TestPagingAndDetailNavigationNeverSubmitOrLoseSelection(t *testing.T) {
	m := inventoryFixture("rostop")
	m.opts.Targets = []string{"worker-00", "worker-47"}
	for _, key := range []tea.KeyType{tea.KeyPgDown, tea.KeyEnd, tea.KeyHome, tea.KeyTab, tea.KeyEnd, tea.KeyUp, tea.KeyEsc} {
		updated, cmd := m.Update(tea.KeyMsg{Type: key})
		m = updated.(model)
		if cmd != nil || len(m.opts.Targets) != 2 || m.opts.RequestID != "" {
			t.Fatal("navigation changed the pending operation")
		}
		if m.offset > m.detailScrollLimit() {
			t.Fatal("detail scroll cannot return from its end")
		}
		assertFits(t, m)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = updated.(model)
	if m.cursor != 47 || !strings.Contains(m.View(), "worker-47") {
		t.Fatal("last process is not visible")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if updated.(model).cursor >= 47 {
		t.Fatal("page up did not move the cursor")
	}
}
