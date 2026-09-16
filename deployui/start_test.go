package deployui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-opm/api"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestUploadIsTheDefaultStart(t *testing.T) {
	for command, want := range map[string]string{"": "upload", "upload": "upload", "artifacts": "artifacts", "rollouts": "rollouts", "watch": "watch"} {
		if m := newModel(context.Background(), nil, Options{Command: command}); m.view != want || m.initialView != want {
			t.Fatalf("command %q starts on %q, want %q", command, m.view, want)
		}
	}
	m := layoutModel()
	m.view, m.width, m.height = "upload", 132, 42
	out := ansi.Strip(m.View())
	if !strings.Contains(out, "업로드 없이 기존 배포본 선택") || !strings.Contains(out, "a 기존 배포본") {
		t.Fatal("upload screen does not point to a for uploaded artifacts:\n" + out)
	}
	m.view = "legacy"
	if strings.Contains(ansi.Strip(m.View()), "기존 배포본") {
		t.Fatal("legacy screen offers a, which it cannot use")
	}
}

func TestStartupHistoryDoesNotReplaceUpload(t *testing.T) {
	for _, command := range []string{"", "upload"} {
		for _, state := range []string{"SUCCEEDED", "RUNNING", "CANCELLED"} {
			m := newModel(context.Background(), nil, Options{Command: command, Value: "sample.far"})
			_, cmd := m.Update(event{kind: "connected", value: connectionResult{client: &Client{}}})
			batch, ok := cmd().(tea.BatchMsg)
			if !ok || len(batch) != 2 {
				t.Fatal("local scan and background history must start independently")
			}
			m.Update(event{kind: "local-files", value: localFiles{}})
			m.Update(event{kind: "startup-rollouts", value: &api.RolloutList{Rollouts: []*api.Rollout{{Id: "old", State: state}}}})
			if m.view != "upload" || m.input != "sample.far" || m.busy {
				t.Fatalf("%s %s: view=%s busy=%t", command, state, m.view, m.busy)
			}
			if !strings.Contains(ansi.Strip(m.View()), "l 목록 보기") {
				t.Fatal("history shortcut is hidden")
			}
		}
	}
}

func TestLateStartupHistoryPreservesForegroundState(t *testing.T) {
	for _, view := range []string{"upload", "artifacts", "rollouts", "watch"} {
		for _, err := range []error{nil, fmt.Errorf("history timeout")} {
			m := newModel(context.Background(), nil, Options{})
			m.view, m.input, m.notice, m.lastError = view, "chosen.far", "foreground status", "foreground error"
			m.busy, m.cursor = true, 3
			m.rollouts = []*api.Rollout{{Id: "fresh"}}
			m.Update(event{kind: "startup-rollouts", value: &api.RolloutList{Rollouts: []*api.Rollout{{Id: "stale", State: "SUCCEEDED"}}}, err: err})
			if m.view != view || !m.busy || m.cursor != 3 || m.input != "chosen.far" || m.notice != "foreground status" || m.lastError != "foreground error" || m.rollouts[0].Id != "fresh" {
				t.Fatal("background result overwrote foreground state")
			}
		}
	}
}

func TestStartupHistoryDoesNotBlockNavigation(t *testing.T) {
	for key, want := range map[string]string{"u": "upload", "a": "artifacts", "l": "rollouts"} {
		m := newModel(context.Background(), nil, Options{})
		m.checkStartupRollouts() // Do not execute: simulate a pending network request.
		if m.busy {
			t.Fatal("history lookup blocked navigation")
		}
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		if cmd == nil || m.view != want {
			t.Fatalf("%s did not navigate to %s", key, want)
		}
	}
}
