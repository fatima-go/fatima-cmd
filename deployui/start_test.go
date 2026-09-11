package deployui

import (
	"context"
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
