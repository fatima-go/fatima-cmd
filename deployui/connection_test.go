package deployui

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-opm/api"
	"github.com/fatima-go/fatima-opm/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type connectionFixture struct {
	api.UnimplementedIdentityServer
	calls atomic.Int32
	err   error
}

func (f *connectionFixture) Login(context.Context, *api.LoginRequest) (*api.Session, error) {
	f.calls.Add(1)
	if f.err != nil {
		return nil, f.err
	}
	return &api.Session{Token: "fixture-token", ExpiresAt: time.Now().Add(time.Hour).Unix()}, nil
}

func connectionServer(t *testing.T, err error) (config.JupiterContextRecord, *connectionFixture) {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	f := &connectionFixture{err: err}
	g := grpc.NewServer()
	api.RegisterIdentityServer(g, f)
	s := transport.NewServer(&http.Server{Handler: http.NotFoundHandler()}, g, &api.Capabilities{Server: "jupiter", ApiVersion: 2, Features: []string{"rollouts"}})
	go s.Serve(l)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})
	return config.NewJupiterContext("fixture", "http://"+l.Addr().String(), "operator", "fixture-password", "UTC").Context, f
}

func TestConnectionErrorsNeverSelectLegacyOrAutomaticallyRetry(t *testing.T) {
	for _, tc := range []struct {
		code codes.Code
		want string
	}{{codes.Unauthenticated, "AUTH_FAILED"}, {codes.PermissionDenied, "ACCESS_DENIED"}, {codes.Unavailable, "CONNECT_FAILED"}, {codes.DeadlineExceeded, "CONNECT_FAILED"}} {
		t.Run(tc.want+tc.code.String(), func(t *testing.T) {
			cfg, f := connectionServer(t, status.Error(tc.code, "fixture rejection"))
			m := newModel(context.Background(), func() (config.JupiterContextRecord, string, error) { return cfg, "local", nil }, Options{Command: "upload", Value: "/preset.far"})
			_, cmd := m.Update(m.connect()())
			if cmd != nil || m.view != "connection" || m.connection.failure.Code != tc.want || m.connection.legacy || f.calls.Load() != 1 {
				t.Fatal("initial failure was hidden, retried, or changed to legacy")
			}
			if !strings.Contains(m.View(), "Connection") || !strings.Contains(m.connection.failure.Error(), "context=local") {
				t.Fatal("connection failure missing from TUI or stderr summary")
			}
		})
	}
	for _, code := range []int{401, 403, 500, 404, 405, 501} {
		t.Run(fmt.Sprint("HTTP", code), func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(code) }))
			defer s.Close()
			r := connect(context.Background(), func() (config.JupiterContextRecord, string, error) {
				return config.JupiterContextRecord{Jupiter: s.URL}, "http-fixture", nil
			})
			legacy := code == 404 || code == 405 || code == 501
			if r.legacy != legacy || (r.failure == nil) != legacy {
				t.Fatalf("HTTP %d fallback=%v", code, r.legacy)
			}
		})
	}
}

func TestRetryReloadsContextAndRestoresRequestedScreen(t *testing.T) {
	good, fixture := connectionServer(t, nil)
	reads := 0
	read := func() (config.JupiterContextRecord, string, error) {
		reads++
		if reads == 1 {
			return config.JupiterContextRecord{}, "broken", fmt.Errorf("fixture configuration missing")
		}
		return good, "repaired", nil
	}
	m := newModel(context.Background(), read, Options{Command: "upload", Value: "/preset.far"})
	m.Update(m.connect()())
	if m.view != "connection" || m.connection.failure.Code != "CONFIG_FAILED" {
		t.Fatal("configuration error did not appear in the connection frame")
	}
	_, retry := m.Update(key("r"))
	_, next := m.Update(retry())
	defer m.client.Close()
	if reads != 2 || fixture.calls.Load() != 1 || next == nil || m.view != "upload" || m.input != "/preset.far" || m.connection.name != "repaired" || m.connection.failure != nil {
		t.Fatal("retry lost the requested screen/path or reused old context")
	}
	if strings.Contains(m.View(), "Connection") {
		t.Fatal("normal flow still displays a Connection stage")
	}
}

func TestConnectionDiagnosticsRedactCredentials(t *testing.T) {
	cfg := config.NewJupiterContext("fixture", "http://inline-user:inline-secret@127.0.0.1:9190?token=url-secret", "operator", "fixture-password", "UTC").Context
	message := cfg.Jupiter + " " + cfg.Password + " fixture-password " + base64.StdEncoding.EncodeToString([]byte("fixture-password")) + "\x1b[31m\nrefused"
	got := redactDiagnostic(message, cfg)
	for _, secret := range []string{cfg.Password, "fixture-password", "inline-secret", "url-secret", base64.StdEncoding.EncodeToString([]byte("fixture-password")), "\x1b"} {
		if strings.Contains(got, secret) {
			t.Fatal("diagnostic exposed a credential or terminal control")
		}
	}
	if !strings.Contains(got, "127.0.0.1:9190") || !strings.Contains(got, "refused") {
		t.Fatal("redaction removed useful connection diagnostics")
	}
}

func TestMalformedSavedPasswordProducesConfigurationError(t *testing.T) {
	cfg, fixture := connectionServer(t, nil)
	for _, password := range []string{"", "invalid-base64", "AA=="} {
		cfg.Password = password
		r := connect(context.Background(), func() (config.JupiterContextRecord, string, error) { return cfg, "broken", nil })
		if r.failure == nil || r.failure.Code != "CONFIG_FAILED" || r.legacy || fixture.calls.Load() != 0 {
			t.Fatal("malformed saved password was used for authentication or hidden")
		}
	}
}
