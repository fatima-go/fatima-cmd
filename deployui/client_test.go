package deployui

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-core/crypt"
	"github.com/fatima-go/fatima-core/opm/api"
	"github.com/fatima-go/fatima-core/opm/transport"
	"google.golang.org/grpc"
)

type loginFixture struct {
	api.UnimplementedIdentityServer
	calls atomic.Int32
	t     *testing.T
}

func (f *loginFixture) Login(ctx context.Context, q *api.LoginRequest) (*api.Session, error) {
	f.calls.Add(1)
	if crypt.ResolveSecret(q.Password) != "b64:literal-password" {
		f.t.Error("literal password prefix was interpreted as secret encoding")
	}
	return &api.Session{Token: "test-token", Role: "OPERATOR", ExpiresAt: time.Now().Add(time.Hour).Unix()}, nil
}
func TestLoginPreservesLiteralPasswordAndReusesSession(t *testing.T) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	g := grpc.NewServer()
	fixture := &loginFixture{t: t}
	api.RegisterIdentityServer(g, fixture)
	s := transport.NewServer(&http.Server{Handler: http.NotFoundHandler()}, g, &api.Capabilities{Server: "jupiter", ApiVersion: 2})
	go s.Serve(l)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	}()
	cfg := config.NewJupiterContext("test", "http://"+l.Addr().String(), "operator", "b64:literal-password", "UTC").Context
	c, e := NewClient(cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	for i := 0; i < 2; i++ {
		if _, e = c.Context(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
	if fixture.calls.Load() != 1 {
		t.Fatal("session was not reused")
	}
}
