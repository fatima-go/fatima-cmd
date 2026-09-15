package controlui

import (
	"context"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatima-go/fatima-opm/api"
	"time"
)

func (c *Client) Registry(ctx context.Context) (*api.RegistryCatalog, error) {
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	return api.NewProcessRegistryClient(c.backend).Catalog(ctx, &api.RegistryQuery{PackageId: c.Target.PackageId, Timezone: c.Config.Timezone})
}
func (c *Client) PreviewRegistry(ctx context.Context, o Options) (*api.RegistryPlan, error) {
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	return api.NewProcessRegistryClient(c.backend).Preview(ctx, &api.RegistryRequest{PackageId: c.Target.PackageId, RequestId: o.RequestID, Action: o.Action, Process: o.Process, Group: o.RegistryGroup})
}

type registryPreview struct {
	plan *api.RegistryPlan
	err  error
}

func (m model) previewRegistry() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
		defer cancel()
		v, err := m.client.PreviewRegistry(ctx, m.opts)
		return registryPreview{v, err}
	}
}
func (m model) registryReview() []string {
	var out []string
	if m.opts.Action == "remove" {
		out = append(out, warningStyle.Render("⚠ 되돌릴 수 없음 · 프로그램·데이터·로그·리비전 영구 삭제"), "")
	}
	out = append(out, "작업: "+m.opts.Action+" "+m.opts.Process, "패키지: "+m.client.Target.PackageId)
	if m.opts.Action != "remove" {
		out = append(out, "프로세스 그룹: "+m.opts.RegistryGroup)
	}
	if m.opts.Plan != nil {
		out = append(out, m.opts.Plan.Effects...)
	}
	return append(out, "", "요청 ID: "+m.opts.RequestID)
}
func (m model) registryInput() []string {
	out := []string{"새 프로세스 등록", "프로세스: " + m.opts.Process, "프로세스 그룹 ID 또는 이름을 입력합니다."}
	if m.registry != nil {
		for _, g := range m.registry.Groups {
			if g.Id != 1 {
				out = append(out, fmt.Sprintf("%d: %s", g.Id, g.Name))
			}
		}
	}
	return out
}
