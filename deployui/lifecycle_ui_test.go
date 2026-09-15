package deployui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-core/opm/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLifecycleMenusFitAndExposeActions(t *testing.T) {
	for _, size := range [][2]int{{60, 19}, {80, 24}, {132, 42}} {
		for _, view := range []string{"activity", "conflict", "actions"} {
			m := layoutModel()
			m.width, m.height = size[0], size[1]
			if view == "activity" {
				m.view = view
				m.cursor = 1
			} else {
				m.view = "watch"
				m.confirm = view
			}
			output := ansi.Strip(m.View())
			for _, line := range strings.Split(output, "\n") {
				if ansi.StringWidth(line) > size[0]-2 {
					t.Fatalf("%s wrapped: %s", view, line)
				}
			}
			if !strings.Contains(output, "Enter") {
				t.Fatal("no visible navigation")
			}
		}
	}
}
func TestConflictMenuPreservesDraftUntilCleanupCompletes(t *testing.T) {
	m := layoutModel()
	original := m.draft.ArtifactId
	m.confirm = "conflict"
	m.menuIndex = 1
	if cmd := m.chooseAction(); cmd != nil {
		t.Fatal("cancel submitted without confirmation")
	}
	if !m.replaceAfterCancel || m.confirm != "cancel" || m.draft.ArtifactId != original {
		t.Fatal("draft lost")
	}
	m.rollout.State = "CANCELLED"
	m.restoreDraft()
	if m.confirm != "create" || m.draft.ArtifactId != original || m.replaceAfterCancel {
		t.Fatal("replacement not restored")
	}
}
func TestWatchMenuNeedsNoHiddenShortcut(t *testing.T) {
	m := layoutModel()
	m.view = "watch"
	m.width, m.height = 100, 32
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.confirm != "actions" {
		t.Fatal("Enter did not open menu")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.menuIndex != 1 {
		t.Fatal("arrow navigation missing")
	}
}
func TestRejectedSubmissionDoesNotClaimAcceptance(t *testing.T) {
	st, e := status.New(codes.Aborted, "conflict").WithDetails(&api.RolloutConflict{RolloutId: "existing"})
	if e != nil {
		t.Fatal(e)
	}
	if conflictRollout(st.Err()) != "existing" || uncertainSubmission(st.Err()) {
		t.Fatal("conflict classified as uncertain")
	}
	m := layoutModel()
	m.confirm = "create"
	m.Update(event{kind: "create", err: status.Error(codes.PermissionDenied, "denied")})
	if strings.Contains(m.notice, "may have been accepted") || !strings.Contains(m.notice, "시작되지 않았습니다") {
		t.Fatal(m.notice)
	}
	if !uncertainSubmission(status.Error(codes.Unavailable, "lost")) {
		t.Fatal("network uncertainty lost")
	}
}
func TestStartupShowsExistingWorkWithoutCommandKnowledge(t *testing.T) {
	m := layoutModel()
	m.Update(event{kind: "recent", value: &api.RolloutList{Rollouts: m.rollouts}})
	if m.view != "activity" || m.cursor != 1 {
		t.Fatal("existing work hidden on startup")
	}
	if !strings.Contains(strings.Join(m.activityScreen().rows, " "), "새 배포") {
		t.Fatal("new deployment entry missing")
	}
}
