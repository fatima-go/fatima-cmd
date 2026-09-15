package integration

import (
	"archive/zip"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-cmd/deployui"
	"github.com/fatima-go/fatima-core/opm/api"
	"github.com/fatima-go/fatima-core/opm/transport"
	juno "github.com/fatima-go/juno/deployment"
	jupiter "github.com/fatima-go/jupiter/deployment"
	"google.golang.org/grpc"
)

func serve(t *testing.T, g *grpc.Server, caps *api.Capabilities) string {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	server := transport.NewServer(&http.Server{}, g, caps)
	go server.Serve(l)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})
	return "http://" + l.Addr().String()
}
func far(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "example.far")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(f)
	for name, body := range map[string]string{"deployment.json": `{"process":"example"}`, "example": "test executable"} {
		w, e := z.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = w.Write([]byte(body)); e != nil {
			t.Fatal(e)
		}
	}
	if e = z.Close(); e != nil {
		t.Fatal(e)
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
	return path
}
func TestManagedCLIThroughRealJupiterAndJuno(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("failure=%v", fail), func(t *testing.T) {
			var targets []*api.Target
			gateway, e := jupiter.New(t.TempDir(), func(string, string) (string, error) { return "OPERATOR", nil }, func() ([]*api.Target, error) { return targets, nil })
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(gateway.Close)
			g := grpc.NewServer()
			gateway.Register(g)
			endpoint := serve(t, g, gateway.Capabilities())
			authConn, e := transport.Dial(endpoint)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { authConn.Close() })
			var starts atomic.Int32
			entered := make(chan struct{}, 2)
			finish := make(chan struct{})
			for _, name := range []string{"a:default", "b:default"} {
				runner, e := juno.New(t.TempDir(), name, "linux_amd64", func(ctx context.Context, q *api.ValidateRequest) error {
					_, e := api.NewIdentityClient(authConn).Validate(transport.WithToken(ctx, transport.Token(ctx)), q)
					return e
				}, func(ctx context.Context, _ *api.OperationSpec, _ string, _ juno.Emit) (juno.Result, error) {
					starts.Add(1)
					entered <- struct{}{}
					select {
					case <-ctx.Done():
						return juno.Result{}, ctx.Err()
					case <-finish:
					}
					if fail {
						return juno.Result{}, fmt.Errorf("startup failed")
					}
					return juno.Result{}, nil
				})
				if e != nil {
					t.Fatal(e)
				}
				t.Cleanup(runner.Close)
				server := grpc.NewServer()
				runner.Register(server)
				address := serve(t, server, runner.Capabilities())
				targets = append(targets, &api.Target{PackageId: name, Group: "backend", Endpoint: address})
			}
			gateway.StartWorkers()
			cfg := config.NewJupiterContext("integration", endpoint, "operator", "password", "UTC").Context
			owner, e := deployui.NewClient(cfg)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(owner.Close)
			observer, e := deployui.NewClient(cfg)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(observer.Close)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			artifact, e := owner.Upload(ctx, far(t), "upload", nil)
			if e != nil {
				t.Fatal(e)
			}
			plan, e := owner.Create(ctx, &api.CreateRollout{RequestId: "deploy", ArtifactId: artifact.Id, Group: "backend", FirstPackageId: "a:default"})
			if e != nil {
				t.Fatal(e)
			}
			if plan.ManagementSessionId == "" {
				t.Fatal("owner session was not wired")
			}
			select {
			case <-entered:
			case <-ctx.Done():
				t.Fatal("first target not started")
			}
			// Merely opening and closing a second CLI cannot detach the owner.
			if _, e = observer.Get(ctx, plan.Id); e != nil {
				t.Fatal(e)
			}
			observer.Close()
			owner.Close()
			close(finish)
			reader, e := deployui.NewClient(cfg)
			if e != nil {
				t.Fatal(e)
			}
			defer reader.Close()
			for {
				result, e := reader.Get(ctx, plan.Id)
				if e != nil {
					t.Fatal(e)
				}
				if result.State == "CANCELLED" || result.State == "FAILED" {
					if result.BlocksDeployment || result.Targets[1].Operation.State != "CANCELLED" || starts.Load() != 1 {
						t.Fatalf("bad cleanup: %v starts=%d", result, starts.Load())
					}
					want := "SUCCEEDED"
					if fail {
						want = "FAILED"
					}
					if result.Targets[0].Operation.State != want {
						t.Fatal(result)
					}
					// Existing history cannot block the next user's deployment.
					next, e := reader.Create(ctx, &api.CreateRollout{RequestId: "replacement", ArtifactId: artifact.Id, Group: "backend", FirstPackageId: "a:default"})
					if e != nil || next.Id == plan.Id {
						t.Fatal(next, e)
					}
					reader.Close()
					break
				}
				select {
				case <-ctx.Done():
					t.Fatal("rollout not finalized", result)
				case <-time.After(100 * time.Millisecond):
				}
			}
		})
	}
}
