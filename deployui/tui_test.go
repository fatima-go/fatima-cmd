package deployui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-core/opm/api"
)

func key(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }
func TestUnreachableJunoNeverFallsBack(t *testing.T) {
	m := &model{view: "targets", artifact: &api.Artifact{Id: "artifact"}, targets: []*api.Target{{PackageId: "one:default", Group: "group", Reason: "connection refused"}}}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.view == "legacy" || m.draft != nil {
		t.Fatal("network error selected a mutating fallback")
	}
	if !strings.Contains(m.notice, "no deployment submitted") {
		t.Fatal("failure was hidden")
	}
	m.targets[0].Legacy = true
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || m.view != "legacy" || m.draft != nil || m.confirm != "" {
		t.Fatal("explicit legacy detection not routed before submission")
	}
	// Opening the local picker is read-only; legacy deployment still needs
	// file selection and an explicit confirmation.
}
func TestFirstPackageConfirmationPinsEveryTarget(t *testing.T) {
	m := &model{view: "targets", cursor: 1, artifact: &api.Artifact{Id: "artifact"}, targets: []*api.Target{{PackageId: "one:default", Group: "group", Supported: true}, {PackageId: "two:default", Group: "group", Supported: true}, {PackageId: "other:default", Group: "other", Supported: true}}}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.confirm != "create" {
		t.Fatal("first selection submitted immediately")
	}
	if m.draft.FirstPackageId != "two:default" || len(m.draft.PackageIds) != 2 || m.draft.RequestId == "" {
		t.Fatal("plan was not pinned")
	}
}
func TestApprovalUsesReviewedRevisionAndExpiryIsVisible(t *testing.T) {
	m := &model{view: "watch", width: 80, height: 24, rollout: &api.Rollout{Id: "plan", Revision: 5, State: "WAITING", Artifact: &api.Artifact{}}}
	m.Update(key("c"))
	if m.confirmRevision != 5 || m.confirm != "continue" {
		t.Fatal("approval did not pin reviewed revision")
	}
	m.rollout = &api.Rollout{Id: "plan", Revision: 6, State: "WAITING", Artifact: &api.Artifact{}}
	if m.confirmRevision != 5 {
		t.Fatal("background refresh changed approval")
	}
	m.view = "artifacts"
	m.confirm = ""
	m.artifacts = []*api.Artifact{{Id: "expired", ExpiresAt: time.Now().Add(-time.Second).Unix()}}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.busy {
		t.Fatal("expired artifact dispatched")
	}
	if !strings.Contains(m.View(), "EXPIRED") {
		t.Fatal("expiry not displayed")
	}
	for _, line := range strings.Split(m.View(), "\n") {
		if ansi.StringWidth(line) > 78 {
			t.Fatal("render exceeded terminal width")
		}
	}
}
