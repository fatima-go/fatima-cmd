package deployui

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-cmd/farassemble"
	"github.com/fatima-go/fatima-opm/api"
	"github.com/fatima-go/fatima-opm/artifact"
	"github.com/fatima-go/fatima-opm/transport"
	"google.golang.org/grpc"
)

type Client struct {
	createMu    sync.Mutex
	latestOwner string
	Config      config.JupiterContextRecord
	conn        *grpc.ClientConn
	mu          sync.Mutex
	session     *api.Session
	ownerMu     sync.Mutex
	owners      map[string]*ownedDeployment
	ownerCtx    context.Context
	ownerCancel context.CancelFunc
	ownerWG     sync.WaitGroup
	closeOnce   sync.Once
	closed      bool
	// assemblyNotice 마지막 업로드에서 profile 에 맞게 FAR 를 재조립한 결과 요약
	assemblyNotice string
}

func NewClient(c config.JupiterContextRecord) (*Client, error) {
	conn, e := transport.Dial(c.Jupiter)
	if e != nil {
		return nil, e
	}
	ownerCtx, ownerCancel := context.WithCancel(context.Background())
	return &Client{Config: c, conn: conn, ownerCtx: ownerCtx, ownerCancel: ownerCancel}, nil
}
func (c *Client) Context(ctx context.Context) (context.Context, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == nil || c.session.ExpiresAt < time.Now().Add(time.Minute).Unix() {
		check, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		plain, e := contextPassword(c.Config)
		if e != nil {
			return nil, e
		}
		password := "b64:" + base64.StdEncoding.EncodeToString([]byte(plain))
		s, e := api.NewIdentityClient(c.conn).Login(check, &api.LoginRequest{Username: c.Config.User, Password: password})
		if e != nil {
			return nil, e
		}
		c.session = s
	}
	return transport.WithToken(ctx, c.session.Token), nil
}
func (c *Client) Artifacts(ctx context.Context) (*api.ArtifactList, error) {
	ctx, e := c.Context(ctx)
	if e != nil {
		return nil, e
	}
	return api.NewArtifactsClient(c.conn).List(ctx, &api.ArtifactQuery{})
}
func (c *Client) Rollouts(ctx context.Context) (*api.RolloutList, error) {
	ctx, e := c.Context(ctx)
	if e != nil {
		return nil, e
	}
	return api.NewDeploymentsClient(c.conn).List(ctx, &api.Empty{})
}
func (c *Client) Targets(ctx context.Context, group string) (*api.TargetList, error) {
	ctx, e := c.Context(ctx)
	if e != nil {
		return nil, e
	}
	return api.NewDeploymentsClient(c.conn).Targets(ctx, &api.TargetQuery{Group: group})
}
func (c *Client) Create(ctx context.Context, q *api.CreateRollout) (*api.Rollout, error) {
	if err := c.manage(ctx, q); err != nil {
		return nil, err
	}
	ctx, e := c.Context(ctx)
	if e != nil {
		return nil, e
	}
	var result *api.Rollout
	for attempt := 0; attempt < 3; attempt++ {
		result, e = api.NewDeploymentsClient(c.conn).Create(ctx, q)
		if e == nil || !uncertainSubmission(e) || ctx.Err() != nil {
			return result, e
		}
		if attempt < 2 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
	}
	return result, e
}
func (c *Client) Act(ctx context.Context, q *api.RolloutAction) (*api.Rollout, error) {
	ctx, e := c.Context(ctx)
	if e != nil {
		return nil, e
	}
	return api.NewDeploymentsClient(c.conn).Act(ctx, q)
}
func (c *Client) Get(ctx context.Context, id string) (*api.Rollout, error) {
	ctx, e := c.Context(ctx)
	if e != nil {
		return nil, e
	}
	return api.NewDeploymentsClient(c.conn).Get(ctx, &api.RolloutQuery{Id: id})
}
func (c *Client) Watch(ctx context.Context, id string, revision uint64) (grpc.ServerStreamingClient[api.Rollout], error) {
	ctx, e := c.Context(ctx)
	if e != nil {
		return nil, e
	}
	return api.NewDeploymentsClient(c.conn).Watch(ctx, &api.WatchRequest{Id: id, AfterRevision: revision})
}

// Upload uploads the FAR assembled for the Jupiter profile. The original FAR is
// never changed; a temporary assembled copy is removed once the upload ends.
func (c *Client) Upload(ctx context.Context, path, requestID string, progress func(int64, int64)) (*api.Artifact, error) {
	c.mu.Lock()
	c.assemblyNotice = ""
	c.mu.Unlock()
	ctx, e := c.Context(ctx)
	if e != nil {
		return nil, e
	}
	far, cleanup, notice, e := c.assembleForUpload(ctx, path)
	if e != nil {
		return nil, e
	}
	defer cleanup()
	c.mu.Lock()
	c.assemblyNotice = notice
	c.mu.Unlock()
	sum, size, e := artifact.Digest(far)
	if e != nil {
		return nil, e
	}
	if size <= 0 || size > artifact.MaxSize {
		return nil, fmt.Errorf("FAR must be between 1 byte and 1 GiB")
	}
	f, e := os.Open(far)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	stream, e := api.NewArtifactsClient(c.conn).Upload(ctx)
	if e != nil {
		return nil, e
	}
	if e = stream.Send(&api.UploadChunk{Header: &api.UploadHeader{RequestId: requestID, Filename: filepath.Base(path), Size: size, Sha256: sum}}); e != nil {
		return nil, e
	}
	buf := make([]byte, 256*1024)
	var sent int64
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			if e = stream.Send(&api.UploadChunk{Data: buf[:n]}); e == io.EOF {
				return stream.CloseAndRecv()
			} else if e != nil {
				return nil, e
			}
			sent += int64(n)
			if progress != nil {
				progress(sent, size)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	return stream.CloseAndRecv()
}

// AssemblyNotice summarizes how the last uploaded FAR was assembled for the
// Jupiter profile. It is empty when the FAR has no binary install policy.
func (c *Client) AssemblyNotice() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.assemblyNotice
}

// assembleForUpload removes binaries the Jupiter profile does not install. A
// Jupiter that does not advertise session_profile cannot tell an unset profile
// from a missing one, so the original FAR is uploaded unchanged.
func (c *Client) assembleForUpload(ctx context.Context, path string) (string, func(), string, error) {
	noop := func() {}
	c.mu.Lock()
	profile := c.session.GetProfile()
	c.mu.Unlock()

	check, cancel := context.WithTimeout(ctx, 7*time.Second)
	caps, e := transport.Discover(check, c.Config.Jupiter)
	cancel()
	if e != nil {
		return path, noop, "", fmt.Errorf("capability lookup failed; no upload submitted: %w", e)
	}
	if !transport.Supports(caps, "session_profile") {
		dep, err := farassemble.ReadDeployment(path)
		if err == nil && dep != nil && dep.HasInstallPolicy() {
			return path, noop, "Jupiter가 profile을 알려주지 않아 FAR를 그대로 배포합니다 (profile별 설치 정책 미적용)", nil
		}
		return path, noop, "", nil
	}

	result, cleanup, e := farassemble.ForProfile(path, profile)
	if e != nil {
		return path, noop, "", e
	}
	if !result.HasPolicy {
		return result.Path, cleanup, "", nil
	}
	return result.Path, cleanup, result.Summary(), nil
}
