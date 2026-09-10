package controlui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-core/opm/api"
)

func TestPackageSelectionIsReadOnly(t *testing.T) {
	catalog := &api.PackageCatalog{Packages: []*api.PackageEntry{{Target: &api.Target{PackageId: "a:default", Group: "one"}, State: "ALIVE"}, {Target: &api.Target{PackageId: "b:default", Group: "two"}, State: "UNREACHABLE"}}}
	o, err := parse("ropack", []string{"-g", "two"})
	if err != nil {
		t.Fatal(err)
	}
	if rows := filteredPackages(catalog, o, ""); len(rows) != 1 || rows[0].Target.PackageId != "b:default" {
		t.Fatal(rows)
	}
	m := model{ctx: context.Background(), opts: Options{Command: "ropack"}, stage: "select", width: 80, height: 24, packages: catalog, client: &Client{Target: &api.Target{PackageId: "전체"}}}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || next.(model).stage != "detail" {
		t.Fatal("Enter must only show details")
	}
	for _, size := range [][2]int{{60, 19}, {80, 24}, {132, 42}} {
		m.width = size[0]
		m.height = size[1]
		view := m.View()
		lines := strings.Split(view, "\n")
		if len(lines) > m.height {
			t.Fatal("height overflow")
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > m.width {
				t.Fatal("width overflow", line)
			}
		}
	}
}

func TestLegacyCannotLoseRequestIdentity(t *testing.T) {
	for _, o := range []Options{{Command: "rostop", RequestID: "stable"}, {Command: "rocron", WatchID: "saved"}, {Command: "ropack", Group: "target"}} {
		if legacyOptionsError(o) == nil {
			t.Fatal("new-only option silently discarded", o)
		}
	}
	if legacyOptionsError(Options{Command: "rostop", Process: "sample"}) != nil {
		t.Fatal("ordinary old syntax must fall back")
	}
}
