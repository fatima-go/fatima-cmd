package deployui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-core/opm/api"
)

// A v2 first package in a group that also holds a legacy Juno must explain why
// the whole group falls back instead of silently opening the legacy upload.
func TestMixedGroupNamesTheLegacyPackages(t *testing.T) {
	m := &model{view: "targets", cursor: 1, artifact: &api.Artifact{Id: "artifact"}, targets: []*api.Target{
		{PackageId: "be01.qa:default", Group: "backend", Supported: true},
		{PackageId: "evt01.qa:default", Group: "backend", Supported: true},
		{PackageId: "support.qa:default", Group: "backend", Legacy: true, Reason: "legacy protocol"},
		{PackageId: "other.qa:default", Group: "other", Legacy: true},
	}}
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil || m.confirm != "legacy-group" {
		t.Fatal("mixed group did not ask for confirmation")
	}
	text := strings.Join(m.confirmationLines(), "\n")
	for _, want := range []string{"backend", "support.qa:default", "Enter legacy 방식으로 진행"} {
		if !strings.Contains(text, want) {
			t.Fatalf("confirmation misses %q:\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"evt01.qa:default", "other.qa:default", "Enter 실행"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("confirmation lists %q:\n%s", unwanted, text)
		}
	}
}
