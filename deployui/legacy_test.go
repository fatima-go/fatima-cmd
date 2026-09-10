package deployui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

const legacySuccessLog = "2026-09-10 09:30:00 transfer finished. waiting server response...\n ()\nfar name : example.far (12345 bytes). target : 3 juno enqueued\n"

func TestLegacyCompletionRequiresFinalReceiptAndSuccessfulExit(t *testing.T) {
	for _, tc := range []struct {
		name, output, state string
		err                 error
	}{
		{"receipt", legacySuccessLog, "COMPLETED", nil},
		{"auth error exits zero", "auth fail : invalid response : 401\n", "FAILED", nil},
		{"HTTP error exits zero", "fail to deploy package : invalid response : 500\n", "FAILED", nil},
		{"file error exits zero", "far farArtifactFile doesn't exist : example.far\n", "FAILED", nil},
		{"process failed", legacySuccessLog, "FAILED", errors.New("exit status 1")},
		{"no response", "transfer finished. waiting server response...\n", "UNCONFIRMED", nil},
		{"unknown response", "request finished\n", "UNCONFIRMED", nil},
		{"receipt not final", legacySuccessLog + "connection lost\n", "UNCONFIRMED", nil},
		{"zero targets", "far name : example.far (12345 bytes). target : 0 juno enqueued\n", "UNCONFIRMED", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := legacyResult(tc.output, tc.err)
			if got.State != tc.state {
				t.Fatalf("state=%s, want %s", got.State, tc.state)
			}
			if got.State == "COMPLETED" && got.Targets != 3 {
				t.Fatal("Jupiter target count was lost")
			}
		})
	}
}

func TestLegacyResultRemainsVisibleWhenLogsScroll(t *testing.T) {
	m := newModel(context.Background(), nil, Options{Legacy: true, Command: "upload"})
	m.view, m.busy = "legacy", true
	m.legacy = legacyOutcome{State: "RUNNING", Summary: "FAR 전송 및 Jupiter 응답 대기 중…"}
	m.follow, m.focusDetail = true, true
	// A receipt can arrive across pipe reads; it is not completion until exit.
	log := strings.Repeat("upload progress ...\n", 200) + legacySuccessLog
	m.Update(event{kind: "legacy-output", value: log[:len(log)-8]})
	m.Update(event{kind: "legacy-output", value: log[len(log)-8:]})
	if !m.busy || m.legacy.State != "RUNNING" {
		t.Fatal("receipt alone marked a still-running command complete")
	}
	m.Update(event{kind: "legacy-done"})
	if m.busy || m.legacy.State != "COMPLETED" || m.legacyLog != log {
		t.Fatal("finished request not marked complete, or legacy log changed")
	}
	for _, size := range [][2]int{{60, 19}, {80, 24}, {132, 42}} {
		m.width, m.height = size[0], size[1]
		m.View()
		m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		output := ansi.Strip(m.View())
		if !strings.Contains(output, "✓ 배포 요청 완료") || !strings.Contains(output, "COMPLETED") || !strings.Contains(output, "대상 3개") {
			t.Fatalf("%v: completion hidden by scrolling or terminal size", size)
		}
		if strings.Contains(output, "READY") || strings.Contains(output, "진행 중") || strings.Contains(output, "Enter 업로드") {
			t.Fatal("completed result still looks like an active request")
		}
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.confirm != "" || m.busy {
		t.Fatal("Enter on completed result offered a duplicate deployment")
	}
	_, cmd = m.Update(key("n"))
	if cmd == nil || m.legacy.State != "" || m.legacyLog != "" {
		t.Fatal("new deployment retained the previous result")
	}
}
