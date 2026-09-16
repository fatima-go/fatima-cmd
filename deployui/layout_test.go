package deployui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-opm/api"
)

func layoutModel() *model {
	a := &api.Artifact{Id: "a_5c1c72a827a647e43779ee4e4cf775c9", Process: "cheerchart", Filename: "cheerchart.far", UploadedBy: "operator", Size: 16 << 20, UploadedAt: 1788870139, ExpiresAt: time.Now().Add(time.Hour).Unix(), Platforms: []string{"darwin_arm64", "linux_arm64"}, BuildJson: `{"user":"djin.chung","time":"2026-09-03 09:12:35 KST","git":{"branch":"feature/GD-4405","commit":"731ec837369b169f098f50fde1813faf52864296","message":"GD-4405 시간별 인기 플레이리스트 구매 차트 추가\n","repo":"https://github.com/music-flo/be-cheer-chart.git"}}`}
	m := newModel(context.Background(), nil, Options{Endpoint: "http://127.0.0.1:9190"})
	m.connection.name = "local"
	m.connection.config = config.JupiterContextRecord{User: "operator", Timezone: "UTC"}
	m.artifact, m.artifacts = a, []*api.Artifact{a}
	m.targets = []*api.Target{{PackageId: "backend01:default", Group: "backend", Supported: true, Platform: "darwin_arm64"}}
	m.rollout = &api.Rollout{Id: "r_fixture", Artifact: a, State: "WAITING", Group: "backend", Targets: []*api.TargetRun{{Target: m.targets[0], Operation: &api.Operation{Id: "op_fixture", State: "SUCCEEDED", Events: []*api.Event{{At: time.Now().UnixMilli(), Stage: "start", State: "SUCCEEDED", Message: "Process started successfully"}}}}}}
	m.rollouts = []*api.Rollout{m.rollout}
	m.draft = &api.CreateRollout{ArtifactId: a.Id, FirstPackageId: m.targets[0].PackageId, PackageIds: []string{m.targets[0].PackageId}, Group: "backend", RequestId: "cli_fixture"}
	m.locals = localFiles{Root: "/workspace/far", Files: []localFAR{{Path: "/workspace/far/cheerchart/cheerchart.far", Size: a.Size, Modified: time.Now()}}}
	return m
}

func TestCommonFrameFitsAllScreensAndTerminalSizes(t *testing.T) {
	for _, view := range []string{"artifacts", "upload", "targets", "watch", "rollouts", "legacy", "connection", "confirm"} {
		t.Run(view, func(t *testing.T) {
			for _, size := range [][2]int{{60, 19}, {80, 24}, {100, 32}, {132, 42}} {
				m := layoutModel()
				m.view, m.width, m.height = view, size[0], size[1]
				if view == "upload" {
					m.startupNotice = "기존 배포 12건 · 미종료 2건 · l 목록 보기"
				}
				if view == "connection" {
					m.connection.failure = &connectionFailure{Code: "AUTH_FAILED", Message: "Jupiter 인증에 실패했습니다."}
				} else if view == "confirm" {
					m.view, m.confirm = "targets", "create"
				}
				output := ansi.Strip(m.View())
				lines := strings.Split(output, "\n")
				if len(lines) > size[1]-1 {
					t.Fatalf("%v: footer overflowed screen", size)
				}
				for _, line := range lines {
					if ansi.StringWidth(line) > size[0]-2 {
						t.Fatalf("%v: line wrapped unexpectedly: %s", size, line)
					}
				}
				if !strings.Contains(output, "배포 진행 단계") || !strings.Contains(output, "127.0.0.1:9190") || !strings.Contains(strings.Join(lines[len(lines)-2:], "\n"), "q ") {
					t.Fatalf("%v: shared frame or exit help missing", size)
				}
				if view != "connection" && strings.Contains(output, "Connection") {
					t.Fatal("normal flow displays Connection stage")
				}
			}
		})
	}
}

func TestRegistrationUsesPCTimezoneAndNeverTTLOrBuildTime(t *testing.T) {
	previous := time.Local
	time.Local = time.FixedZone("test-local", 5*60*60+30*60)
	t.Cleanup(func() { time.Local = previous })
	m := layoutModel()
	m.view = "artifacts" // the start screen is Upload; registration time lives on the artifact list
	m.artifacts[0].UploadedAt = time.Date(2026, 9, 8, 12, 22, 19, 0, time.UTC).Unix()
	for _, width := range []int{60, 80, 132} {
		m.width, m.height = width, 24
		output := ansi.Strip(m.View())
		if !strings.Contains(output, "2026-09-08 17:52:19") || strings.Contains(output, "TTL") {
			t.Fatalf("width=%d: registration time missing, clipped, or replaced by TTL", width)
		}
	}
	fields := strings.Join(m.screen(100).details, "\n")
	for _, value := range []string{"등록 시간", "2026-09-08 17:52:19", "빌드 시각", "2026-09-03 09:12:35 KST", "빌드 사용자", "djin.chung", "브랜치", "feature/GD-4405", "메시지", "GD-4405 시간별"} {
		if !strings.Contains(fields, value) {
			t.Fatalf("structured metadata missing: %s", value)
		}
	}
}

func TestInvisibleConfirmationCannotSubmitAndCanBeScrolled(t *testing.T) {
	m := layoutModel()
	m.view, m.confirm, m.width, m.height = "targets", "create", 40, 12
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.busy {
		t.Fatal("confirmation submitted while terminal was too small to display it")
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if m.scroll == 0 || m.confirm != "create" {
		t.Fatal("confirmation cannot be reviewed by scrolling")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.confirm != "" || m.busy {
		t.Fatal("cancelling confirmation dispatched it")
	}
}
