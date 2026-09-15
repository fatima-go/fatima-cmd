package controlui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-opm/api"
	"strings"
	"testing"
)

func TestCronConfirmationAndResult(t *testing.T) {
	m := model{opts: Options{Command: "rocron"}, ctx: context.Background(), client: &Client{Config: config.JupiterContextRecord{Jupiter: "http://localhost:9190"}, Name: "local", Target: &api.Target{PackageId: "host:default"}}, jobs: []*api.CronEntry{{Process: "sample", Name: "run", Sample: "date=2026-09-10"}}, stage: "select", width: 80, height: 24}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if cmd != nil || m.stage != "arguments" {
		t.Fatal("selection dispatched")
	}
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if cmd != nil || m.stage != "review" {
		t.Fatal("argument entry dispatched")
	}
	if !strings.Contains(m.View(), "sample") || !strings.Contains(m.View(), "완료") {
		t.Fatal(m.View())
	}
	m.op = &api.ControlOperation{Kind: "rocron", State: "REQUESTED", Message: "delivered"}
	m.stage = "result"
	if !strings.Contains(m.View(), "실행 요청 전달 완료") {
		t.Fatal(m.View())
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("result Enter resubmitted")
	}
	for _, size := range [][2]int{{60, 19}, {80, 24}, {132, 42}} {
		m.width = size[0]
		m.height = size[1]
		v := m.View()
		for _, line := range strings.Split(v, "\n") {
			if ansi.StringWidth(line) > m.width {
				t.Fatalf("line overflows %d: %q", m.width, line)
			}
		}
		if len(strings.Split(v, "\n")) > m.height {
			t.Fatalf("height overflow %v", size)
		}
	}
}
