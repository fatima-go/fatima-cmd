package controlui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-opm/api"
)

func removalReview() model {
	m := inventoryFixture("roproc")
	m.opts.Action, m.opts.Process, m.opts.RequestID = "remove", "worker-01", "c_test"
	var effects []string
	for i := 0; i < 40; i++ {
		effects = append(effects, fmt.Sprintf("- remove path %02d", i))
	}
	plan := &api.RegistryPlan{Request: &api.RegistryRequest{Action: "remove", Process: "worker-01", PackageId: "backend01:default", RequestId: "c_test"}, Effects: effects}
	next, _ := m.Update(registryPreview{plan, nil})
	return next.(model)
}

func TestRemovalNeedsTypedProcessName(t *testing.T) {
	m := removalReview()
	if m.stage != "review" || m.editing != "confirm" {
		t.Fatal(m.stage, m.editing)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || next.(model).stage != "review" {
		t.Fatal("Enter removed without the process name")
	}
	next, _ = next.(model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("worker-0")})
	next, cmd = next.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || next.(model).stage != "review" {
		t.Fatal("a partial name confirmed removal")
	}
	next, _ = next.(model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	next, cmd = next.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || next.(model).stage != "result" {
		t.Fatal("the typed name did not confirm removal")
	}
	next, cmd = removalReview().Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil || next.(model).stage != "select" || next.(model).opts.Action != "" || next.(model).opts.Plan != nil {
		t.Fatal("Esc did not discard the removal draft")
	}
}

func TestRegistryActionsSitAboveTheList(t *testing.T) {
	m := inventoryFixture("roproc")
	m.cursor = 3
	v := m.View()
	if i, j := strings.Index(v, "프로세스 등록"), strings.Index(v, "PROCESS"); i < 0 || j < 0 || i > j || !strings.Contains(v, "삭제 · worker-03") {
		t.Fatal(v)
	}
	if strings.Contains(v, "a 등록  x 삭제") {
		t.Fatal("a and x are still repeated in the footer")
	}
	m.catalog.Processes[3].Opm = true
	if !strings.Contains(m.View(), "worker-03 삭제 불가 (OPM)") {
		t.Fatal(m.View())
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if cmd != nil || next.(model).opts.Action != "" || !strings.Contains(next.(model).status, "OPM") {
		t.Fatal("x on an OPM process requested a removal preview")
	}
}

func TestDecisionBarStaysVisible(t *testing.T) {
	m := removalReview()
	for _, size := range [][2]int{{60, 19}, {80, 24}, {132, 42}} {
		m.width, m.height = size[0], size[1]
		for _, offset := range []int{0, 1000} {
			m.offset = offset
			assertFits(t, m)
			v := m.View()
			if !strings.Contains(v, "Enter") || !strings.Contains(v, "Esc") || !strings.Contains(v, "worker-01") {
				t.Fatalf("decision bar hidden at %v offset %d:\n%s", size, offset, v)
			}
		}
	}
	add := inventoryFixture("roproc")
	add.opts.Action, add.opts.Process, add.opts.RegistryGroup, add.stage = "add", "newproc", "4", "review"
	if !strings.Contains(add.View(), "newproc 등록 (그룹 4)") || add.destructive() {
		t.Fatal(add.View())
	}
	if _, cmd := add.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
		t.Fatal("registration needs no typed name")
	}
	stop := inventoryFixture("rostop")
	stop.opts.Targets, stop.stage = []string{"worker-01"}, "review"
	if !strings.Contains(stop.View(), "worker-01 중지") || !stop.destructive() {
		t.Fatal(stop.View())
	}
}
