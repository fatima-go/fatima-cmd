package deployui

import (
	"context"
	"sync"
	"time"

	"github.com/fatima-go/fatima-opm/api"
	"github.com/fatima-go/fatima-opm/lifecycle"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type ownedDeployment struct {
	credential    *api.ManagementCredential
	state, notice string
}

func (c *Client) manage(ctx context.Context, q *api.CreateRollout) error {
	c.createMu.Lock()
	defer c.createMu.Unlock()
	c.ownerMu.Lock()
	if c.closed {
		c.ownerMu.Unlock()
		return status.Error(codes.Canceled, "client closed")
	}
	if c.owners == nil {
		c.owners = map[string]*ownedDeployment{}
	}
	if owner := c.owners[q.RequestId]; owner != nil {
		q.Management = proto.Clone(owner.credential).(*api.ManagementCredential)
		c.ownerMu.Unlock()
		return nil
	}
	c.ownerMu.Unlock()
	auth, err := c.Context(ctx)
	if err != nil {
		return err
	}
	session, err := api.NewDeploymentManagementClient(c.conn).Open(auth, &api.Empty{})
	if status.Code(err) == codes.Unimplemented {
		return status.Error(codes.FailedPrecondition, "Jupiter 업데이트 필요: CLI 종료 시 배포 정리 기능 미지원")
	}
	if err != nil {
		return err
	}
	c.ownerMu.Lock()
	defer c.ownerMu.Unlock()
	if c.closed {
		return status.Error(codes.Canceled, "client closed")
	}
	owner := &ownedDeployment{credential: &api.ManagementCredential{Id: session.Id, Token: session.Token}, state: session.State}
	c.owners[q.RequestId] = owner
	c.latestOwner = q.RequestId
	q.Management = proto.Clone(owner.credential).(*api.ManagementCredential)
	c.ownerWG.Add(1)
	go func() { defer c.ownerWG.Done(); c.heartbeat(c.ownerCtx, owner) }()
	return nil
}
func (c *Client) heartbeat(ctx context.Context, owner *ownedDeployment) {
	delay := lifecycle.HeartbeatInterval
	for {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		request, cancel := context.WithTimeout(ctx, 3*time.Second)
		auth, err := c.Context(request)
		var session *api.ManagementSession
		if err == nil {
			session, err = api.NewDeploymentManagementClient(c.conn).Heartbeat(auth, owner.credential)
		}
		cancel()
		c.ownerMu.Lock()
		if err != nil {
			owner.notice = "관리 연결 복구 중 · 마지막 신호 후 20초가 지나면 남은 배포 취소"
			delay = time.Second
		} else {
			owner.state = session.State
			owner.notice = ""
			delay = lifecycle.HeartbeatInterval
			if session.State != "ACTIVE" {
				owner.notice = "CLI 관리 세션 종료 · 취소를 되돌리지 않고 결과를 확인합니다"
			}
		}
		done := err == nil && session.State != "ACTIVE"
		c.ownerMu.Unlock()
		if done {
			return
		}
	}
}
func (c *Client) managementNotice() string {
	c.ownerMu.Lock()
	defer c.ownerMu.Unlock()
	if o := c.owners[c.latestOwner]; o != nil {
		return o.notice
	}
	return ""
}
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.ownerMu.Lock()
		c.closed = true
		if c.ownerCancel != nil {
			c.ownerCancel()
		}
		owners := make([]*api.ManagementCredential, 0, len(c.owners))
		for _, o := range c.owners {
			owners = append(owners, o.credential)
		}
		c.ownerMu.Unlock()
		c.ownerWG.Wait()
		// A bounded best-effort detach; lost delivery falls back to the 20s lease.
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var wg sync.WaitGroup
		for _, credential := range owners {
			wg.Add(1)
			go func(q *api.ManagementCredential) {
				defer wg.Done()
				auth, err := c.Context(ctx)
				if err == nil {
					_, _ = api.NewDeploymentManagementClient(c.conn).Detach(auth, q)
				}
			}(credential)
		}
		wg.Wait()
		_ = c.conn.Close()
	})
}

func (c *Client) ownsDeployment() bool {
	c.ownerMu.Lock()
	defer c.ownerMu.Unlock()
	for _, o := range c.owners {
		if o.state == "ACTIVE" {
			return true
		}
	}
	return false
}
