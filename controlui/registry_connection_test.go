package controlui

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/fatima-go/fatima-cmd/cipher"
	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-opm/api"
	"github.com/fatima-go/fatima-opm/transport"
	"google.golang.org/grpc"
)

func registryTestEndpoint(t *testing.T, g *grpc.Server, caps *api.Capabilities) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := transport.NewServer(&http.Server{}, g, caps)
	go s.Serve(l)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})
	return "http://" + l.Addr().String()
}

type directRegistry struct {
	api.UnimplementedProcessRegistryServer
	mu    sync.Mutex
	calls map[string]int
}

func (r *directRegistry) record(ctx context.Context, method, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if transport.Token(ctx) == "test-token" && id == "host:package" {
		r.calls[method]++
	}
}
func (r *directRegistry) Catalog(ctx context.Context, q *api.RegistryQuery) (*api.RegistryCatalog, error) {
	r.record(ctx, "catalog", q.PackageId)
	return &api.RegistryCatalog{}, nil
}
func (r *directRegistry) Preview(ctx context.Context, q *api.RegistryRequest) (*api.RegistryPlan, error) {
	r.record(ctx, "preview", q.PackageId)
	q.ExpectedRevision = "revision"
	return &api.RegistryPlan{Request: q}, nil
}
func (r *directRegistry) Apply(ctx context.Context, q *api.RegistryRequest) (*api.ControlOperation, error) {
	r.record(ctx, "apply", q.PackageId)
	return &api.ControlOperation{Id: "op"}, nil
}
func (r *directRegistry) Get(ctx context.Context, q *api.RegistryOperationQuery) (*api.ControlOperation, error) {
	r.record(ctx, "get", q.PackageId)
	return &api.ControlOperation{Id: q.Id}, nil
}
func (r *directRegistry) Watch(q *api.RegistryOperationQuery, s grpc.ServerStreamingServer[api.ControlOperation]) error {
	r.record(s.Context(), "watch", q.PackageId)
	return s.Send(&api.ControlOperation{Id: q.Id})
}

type registryIdentity struct {
	api.UnimplementedIdentityServer
}

func (*registryIdentity) Login(context.Context, *api.LoginRequest) (*api.Session, error) {
	return &api.Session{Token: "test-token", ExpiresAt: time.Now().Add(time.Hour).Unix()}, nil
}

type registryRouting struct {
	api.UnimplementedRoutingServer
	endpoint string
}

func (r *registryRouting) Resolve(context.Context, *api.PackageQuery) (*api.Target, error) {
	return &api.Target{PackageId: "host:package", Endpoint: r.endpoint}, nil
}

// Jupiter deliberately exposes only identity/routing. All registry calls must
// reach Juno directly, carrying the authenticated session and package identity.
func TestRegistryConnectsDirectlyToJuno(t *testing.T) {
	registry := &directRegistry{calls: map[string]int{}}
	backend := grpc.NewServer()
	api.RegisterProcessRegistryServer(backend, registry)
	endpoint := registryTestEndpoint(t, backend, &api.Capabilities{Server: "juno", ApiVersion: 2, PackageId: "host:package", Features: []string{"roproc"}})
	gateway := grpc.NewServer()
	api.RegisterIdentityServer(gateway, &registryIdentity{})
	api.RegisterRoutingServer(gateway, &registryRouting{endpoint: endpoint})
	gatewayEndpoint := registryTestEndpoint(t, gateway, &api.Capabilities{Server: "jupiter", ApiVersion: 2, Features: []string{"routing"}})
	password, err := cipher.Aes256Encode("password")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, err := Connect(ctx, config.JupiterContextRecord{Jupiter: gatewayEndpoint, Password: password}, "test", Options{Command: "roproc"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.Capabilities.Server != "juno" {
		t.Fatalf("capabilities from %s", c.Capabilities.Server)
	}
	if _, err = c.Registry(ctx); err != nil {
		t.Fatal(err)
	}
	opts := Options{Command: "roproc", RequestID: "request", Action: "add", Process: "sample", RegistryGroup: "4"}
	opts.Plan, err = c.PreviewRegistry(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	op, err := c.Submit(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Get(ctx, "roproc", op.Id); err != nil {
		t.Fatal(err)
	}
	stream, err := c.Watch(ctx, "roproc", op.Id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = stream.Recv(); err != nil {
		t.Fatal(err)
	}
	if _, err = stream.Recv(); err != io.EOF {
		t.Fatalf("watch end: %v", err)
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	for method, want := range map[string]int{"catalog": 2, "preview": 1, "apply": 1, "get": 1, "watch": 1} {
		if got := registry.calls[method]; got != want {
			t.Errorf("authenticated Juno %s calls = %d, want %d", method, got, want)
		}
	}
}
