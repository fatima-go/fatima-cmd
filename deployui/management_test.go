package deployui

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-core/opm/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type managementFixture struct {
	openGate    chan struct{}
	openEntered chan struct{}
	api.UnimplementedDeploymentManagementServer
	mu                     sync.Mutex
	opens, beats, detaches int
	beat                   chan struct{}
}

func (f *managementFixture) Open(ctx context.Context, _ *api.Empty) (*api.ManagementSession, error) {
	if f.openGate != nil {
		close(f.openEntered)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-f.openGate:
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.opens++
	return &api.ManagementSession{Id: "owner", Token: "secret", State: "ACTIVE"}, nil
}
func (f *managementFixture) Heartbeat(_ context.Context, q *api.ManagementCredential) (*api.ManagementSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if q.Id != "owner" || q.Token != "secret" {
		return nil, status.Error(codes.PermissionDenied, "wrong credential")
	}
	f.beats++
	if f.beats == 1 {
		return nil, status.Error(codes.Unavailable, "network interruption")
	}
	select {
	case f.beat <- struct{}{}:
	default:
	}
	return &api.ManagementSession{Id: "owner", State: "ACTIVE"}, nil
}
func (f *managementFixture) Detach(_ context.Context, q *api.ManagementCredential) (*api.ManagementSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if q.Id != "owner" || q.Token != "secret" {
		return nil, status.Error(codes.PermissionDenied, "wrong credential")
	}
	f.detaches++
	return &api.ManagementSession{State: "OWNER_DETACHED"}, nil
}

type ownedCreateFixture struct {
	api.UnimplementedDeploymentsServer
	mu       sync.Mutex
	previous *api.CreateRollout
	calls    int
}

func (f *ownedCreateFixture) Create(_ context.Context, q *api.CreateRollout) (*api.Rollout, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if q.Management == nil {
		return nil, status.Error(codes.InvalidArgument, "owner missing")
	}
	if f.previous != nil && !proto.Equal(q, f.previous) {
		return nil, status.Error(codes.AlreadyExists, "request changed after lost response")
	}
	f.previous = proto.Clone(q).(*api.CreateRollout)
	if f.calls == 1 {
		return nil, status.Error(codes.Unavailable, "lost response")
	}
	return &api.Rollout{Id: "rollout", ManagementSessionId: q.Management.Id}, nil
}
func managementClientFixture(t *testing.T) (*Client, *managementFixture) {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	g := grpc.NewServer()
	f := &managementFixture{beat: make(chan struct{}, 1)}
	api.RegisterDeploymentManagementServer(g, f)
	api.RegisterDeploymentsServer(g, &ownedCreateFixture{})
	api.RegisterIdentityServer(g, &loginFixture{t: t})
	go g.Serve(l)
	t.Cleanup(g.Stop)
	cfg := config.NewJupiterContext("test", "http://"+l.Addr().String(), "operator", "b64:literal-password", "UTC").Context
	c, e := NewClient(cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(c.Close)
	return c, f
}
func TestOwnerSurvivesStreamIndependentReconnectAndDetaches(t *testing.T) {
	c, f := managementClientFixture(t)
	q := &api.CreateRollout{RequestId: "request"}
	if _, e := c.Create(context.Background(), q); e != nil {
		t.Fatal(e)
	}
	if _, e := c.Create(context.Background(), q); e != nil {
		t.Fatal(e)
	}
	// The production interval is 10s; a failed heartbeat retries with the same
	// credential in 1s, without opening another owner or a Watch stream.
	select {
	case <-f.beat:
	case <-time.After(15 * time.Second):
		t.Fatal("heartbeat did not recover")
	}
	c.Close()
	c.Close()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.opens != 1 || f.beats < 2 || f.detaches != 1 {
		t.Fatalf("open=%d heartbeat=%d detach=%d", f.opens, f.beats, f.detaches)
	}
}
func TestObserverCloseNeverDetachesDeployment(t *testing.T) {
	c, f := managementClientFixture(t)
	if _, e := c.Context(context.Background()); e != nil {
		t.Fatal(e)
	}
	c.Close()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.opens != 0 || f.detaches != 0 {
		t.Fatal("observer changed ownership")
	}
}

func TestSlowOwnerOpenDoesNotFreezeUIOrExit(t *testing.T) {
	c, f := managementClientFixture(t)
	f.openGate = make(chan struct{})
	f.openEntered = make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = c.Create(context.Background(), &api.CreateRollout{RequestId: "slow"})
	}()
	select {
	case <-f.openEntered:
	case <-time.After(time.Second):
		t.Fatal("Open not reached")
	}
	rendered := make(chan struct{})
	go func() { _ = c.managementNotice(); _ = c.ownsDeployment(); close(rendered) }()
	select {
	case <-rendered:
	case <-time.After(time.Second):
		t.Fatal("network request blocked UI rendering")
	}
	exited := make(chan struct{})
	go func() { c.Close(); close(exited) }()
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("network request blocked Close")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Open was not interrupted")
	}
}
