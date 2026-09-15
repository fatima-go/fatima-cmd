package controlui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-opm/api"
	"testing"
)

func TestRegistryArgumentsAndConfirmation(t *testing.T) {
	for _, args := range [][]string{{"add", "worker"}, {"add", "worker", "SVC"}, {"remove", "worker"}} {
		o, err := parse("roproc", args)
		if err != nil || o.Process != "worker" {
			t.Fatal(o, err)
		}
	}
	for _, args := range [][]string{{"add"}, {"remove", "worker", "4"}, {"--watch", "old", "remove", "worker"}} {
		if _, err := parse("roproc", args); err == nil {
			t.Fatal(args)
		}
	}
	m := model{ctx: context.Background(), opts: Options{Command: "roproc"}, stage: "select", width: 80, height: 24, catalog: &api.ProcessCatalog{Processes: []*api.ProcessEntry{{Name: "sample"}}}}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || next.(model).stage != "select" {
		t.Fatal("Enter left the registry list or mutated registration")
	}
	m.stage = "result"
	m.op = &api.ControlOperation{State: "SUCCEEDED"}
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || next.(model).stage != "result" {
		t.Fatal("result Enter restarted navigation or mutation")
	}
	m.editing = "group"
	m.stage = "arguments"
	m.opts.Action = "add"
	m.opts.Process = "draft"
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || next.(model).opts.Action != "" || next.(model).opts.Process != "" {
		t.Fatal("canceled registration draft survived")
	}
}
