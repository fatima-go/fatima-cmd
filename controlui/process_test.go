package controlui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-core/opm/api"
	"testing"
)

func TestProcessSelectionAndConfirmation(t *testing.T) {
	c := &api.ProcessCatalog{Processes: []*api.ProcessEntry{{Name: "juno", Group: "OPM", Opm: true}, {Name: "a", Group: "SVC"}, {Name: "b", Group: "SVC"}}}
	for _, o := range []Options{{All: true}, {Group: "svc"}} {
		names, err := selectProcesses(c, o)
		if err != nil || len(names) != 2 {
			t.Fatal(names, err)
		}
	}
	if _, err := selectProcesses(c, Options{All: true, Process: "a"}); err == nil {
		t.Fatal("ambiguous selector accepted")
	}
	m := model{ctx: context.Background(), opts: Options{Command: "rostop"}, catalog: c, stage: "select", cursor: 0, width: 80, height: 24}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if cmd != nil || m.stage != "review" || m.opts.Targets[0] != "a" {
		t.Fatal("selection did not require confirmation")
	}
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || updated.(model).stage != "result" {
		t.Fatal("confirmation did not submit")
	}
}
