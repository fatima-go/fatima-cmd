package deployui

import (
	"context"
	"fmt"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-core/opm/api"
	"github.com/fatima-go/fatima-core/opm/lifecycle"
	"github.com/fatima-go/fatima-core/opm/transport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func rolloutLabel(p *api.Rollout) string {
	if p.CancelRequested && !lifecycle.Terminal(p.State) {
		return "취소 처리 중 · 실행 결과 확인"
	}
	switch p.State {
	case "RUNNING":
		return "배포 진행 중"
	case "WAITING":
		return "첫 서버 완료 · 승인 대기"
	case "ATTENTION":
		return "서버 응답 확인 중"
	case "FAILED":
		return "배포 실패 · 실행 종료"
	case "CANCELLED":
		return "남은 배포 취소 완료"
	case "EXPIRED":
		return "배포 파일 만료"
	case "SUCCEEDED":
		return "배포 완료"
	}
	return p.State
}
func conflictRollout(err error) string {
	for _, detail := range status.Convert(err).Details() {
		if d, ok := detail.(*api.RolloutConflict); ok {
			return d.RolloutId
		}
	}
	return ""
}
func uncertainSubmission(err error) bool {
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled, codes.Unknown, codes.Internal:
		return true
	}
	return false
}
func (m *model) recentRollouts() tea.Cmd {
	return m.task("recent", func(ctx context.Context) (any, error) { return m.client.Rollouts(ctx) })
}
func (m *model) restoreDraft() {
	m.replaceAfterCancel = false
	m.leaveWatch()
	m.confirm = "create"
	m.view = "targets"
	m.menuIndex = 0
	m.scroll = 0
	m.lastError = ""
	// The old submission was explicitly rejected; use a fresh owner for the replacement.
	m.draft.RequestId = transport.ID("cli_")
	m.draft.Management = nil
	m.notice = "기존 작업 종료 확인 · 선택한 파일과 대상으로 새 배포를 확인하세요"
}
func (m *model) actionLabels() []string {
	if m.confirm == "conflict" {
		return []string{"기존 작업 상세 보기", "남은 배포 취소 후 새 배포 준비", "선택한 새 배포로 돌아가기"}
	}
	labels := []string{"새 배포 준비", "배포 목록 보기"}
	if m.rollout != nil && !lifecycle.Terminal(m.rollout.State) {
		labels = append([]string{"남은 배포 취소"}, labels...)
	}
	if m.rollout != nil && m.rollout.State == "WAITING" && !m.rollout.CancelRequested {
		labels = append([]string{"나머지 서버 배포 승인"}, labels...)
	}
	if m.rollout != nil && m.rollout.State == "FAILED" && m.rollout.EndReason == "DEPLOYMENT_FAILED" {
		labels = append([]string{"실패한 서버 다시 배포"}, labels...)
	}
	return labels
}
func (m *model) chooseAction() tea.Cmd {
	labels := m.actionLabels()
	if m.menuIndex < 0 || m.menuIndex >= len(labels) {
		return nil
	}
	action := labels[m.menuIndex]
	m.confirm = ""
	m.menuIndex = 0
	switch action {
	case "기존 작업 상세 보기":
		return nil
	case "선택한 새 배포로 돌아가기":
		m.leaveWatch()
		m.view = "targets"
		m.confirm = "create"
		return nil
	case "남은 배포 취소 후 새 배포 준비":
		if lifecycle.Terminal(m.rollout.State) {
			m.restoreDraft()
			return nil
		}
		m.replaceAfterCancel = true
		m.confirm = "cancel"
	case "남은 배포 취소":
		m.confirm = "cancel"
	case "나머지 서버 배포 승인":
		m.confirm = "continue"
	case "실패한 서버 다시 배포":
		m.confirm = "retry"
	case "새 배포 준비":
		m.leaveWatch()
		m.view = "upload"
		return m.localScan()
	case "배포 목록 보기":
		m.leaveWatch()
		m.view = "rollouts"
		return m.load()
	}
	if m.rollout != nil {
		m.confirmRevision = m.rollout.Revision
	}
	return nil
}
func lifecycleDetails(p *api.Rollout) []string {
	result := []string{rolloutLabel(p)}
	if lifecycle.Terminal(p.State) {
		result = append(result, "실행 중인 작업 없음 · 새 배포 가능")
	} else {
		result = append(result, "새 배포 제한: 기존 작업의 실행·정리 결과 확인 필요")
	}
	if p.EndReason != "" {
		reason := p.EndReason
		switch reason {
		case "OWNER_TIMEOUT":
			reason = "CLI 생존 신호 20초 미수신"
		case "OWNER_DETACHED":
			reason = "배포를 시작한 CLI 종료"
		case "USER_CANCELLED":
			reason = "사용자가 취소 요청"
		case "DEPLOYMENT_FAILED":
			reason = "대상 서버 배포 실패"
		case "ARTIFACT_EXPIRED":
			reason = "배포 파일 만료"
		}
		result = append(result, "종료 사유: "+reason)
	}
	if p.LastCheckedAt > 0 {
		result = append(result, "마지막 서버 확인: "+time.UnixMilli(p.LastCheckedAt).Local().Format("15:04:05"))
	}
	if p.NextCheckAt > 0 {
		result = append(result, "다음 자동 확인: "+time.UnixMilli(p.NextCheckAt).Local().Format("15:04:05"))
	}
	return result
}
func sortActivity(rollouts []*api.Rollout) {
	sort.SliceStable(rollouts, func(i, j int) bool {
		a, b := rollouts[i], rollouts[j]
		if lifecycle.Terminal(a.State) != lifecycle.Terminal(b.State) {
			return !lifecycle.Terminal(a.State)
		}
		return a.CreatedAt > b.CreatedAt
	})
}
func (m *model) activityScreen() screenContent {
	c := screenContent{selected: m.cursor, title: "기존 배포 확인 / 새 배포", description: "실행·확인 중인 작업부터 표시합니다. 방향키와 Enter로 선택하세요.", listTitle: "배포 작업", detailTitle: "선택한 작업", rowHeader: "  프로세스 / 그룹 / 상태"}
	c.rows = append(c.rows, "＋ 새 배포 준비")
	for _, p := range m.rollouts {
		c.rows = append(c.rows, fmt.Sprintf("%s / %s / %s", p.Artifact.Process, p.Group, rolloutLabel(p)))
	}
	if m.cursor > 0 && m.cursor <= len(m.rollouts) {
		p := m.rollouts[m.cursor-1]
		c.details = append(lifecycleDetails(p), "배포자: "+p.CreatedBy, clean(p.Message), "Enter 상세 보기 → Enter 동작 메뉴")
	} else {
		c.details = []string{"로컬 FAR를 선택하거나 기존 업로드 파일로 새 배포를 준비합니다.", "이 화면은 조회만 합니다. 닫아도 다른 CLI가 시작한 배포에는 영향이 없습니다."}
	}
	return c
}
