package deployui

import (
	"github.com/fatima-go/fatima-opm/api"
	"strings"
	"testing"
	"time"
)

func TestTargetCountdown(t *testing.T) {
	now := time.UnixMilli(10000)
	p := &api.Rollout{State: "RUNNING", NextTargetStartAt: 15000, Targets: []*api.TargetRun{
		{Target: &api.Target{PackageId: "a"}, Operation: &api.Operation{State: "SUCCEEDED"}},
		{Target: &api.Target{PackageId: "b"}, Operation: &api.Operation{State: "SUCCEEDED"}},
		{Target: &api.Target{PackageId: "c"}, Operation: &api.Operation{State: "QUEUED"}},
	}}
	for _, tc := range []struct {
		offset time.Duration
		want   string
	}{{0, "2번 서버 완료 · 3번 서버 (c) 배포까지 5초"}, {1100 * time.Millisecond, "배포까지 4초"}, {5 * time.Second, "배포 시작 확인 중"}} {
		if got := targetCountdown(p, now.Add(tc.offset)); !strings.Contains(got, tc.want) {
			t.Fatal(got)
		}
	}
	p.CancelRequested = true
	if targetCountdown(p, now) != "" {
		t.Fatal("countdown during cancellation")
	}
	p.CancelRequested = false
	p.State = "SUCCEEDED"
	if targetCountdown(p, now) != "" {
		t.Fatal("countdown after completion")
	}
}
