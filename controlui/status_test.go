package controlui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-core/opm/api"
	"testing"
)

func TestStatusReadOnlySelection(t *testing.T) {
	m := model{ctx: context.Background(), opts: Options{Command: "rodis", Sort: "name"}, catalog: &api.ProcessCatalog{Processes: []*api.ProcessEntry{{Name: "b", State: "DEAD"}, {Name: "a", State: "ALIVE"}}}, stage: "select", width: 80, height: 24}
	rows := m.visibleProcesses()
	if rows[0].Name != "a" {
		t.Fatal(rows)
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || updated.(model).stage != "detail" {
		t.Fatal("status Enter must only show details")
	}
	m.filter = "dead"
	if m.count() != 1 || m.visibleProcesses()[0].Name != "b" {
		t.Fatal("filter failed")
	}
}
