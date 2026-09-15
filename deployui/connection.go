package deployui

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/fatima-go/fatima-cmd/cipher"
	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-opm/transport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contextReader func() (config.JupiterContextRecord, string, error)

var errContextPassword = errors.New("rocontext password is missing or invalid; update the saved context")

// The historical decoder can panic on truncated ciphertext. Contain malformed
// local settings in the new client without changing the legacy decoder.
func contextPassword(cfg config.JupiterContextRecord) (plain string, err error) {
	defer func() {
		if recover() != nil {
			plain, err = "", errContextPassword
		}
	}()
	plain, err = cipher.Aes256Decode(cfg.Password)
	if err != nil {
		return "", errContextPassword
	}
	return plain, nil
}

type connectionResult struct {
	client  *Client
	config  config.JupiterContextRecord
	name    string
	legacy  bool
	reached bool
	failure *connectionFailure
}

type connectionFailure struct {
	Code, Message, Detail, Endpoint, ContextName, User string
}

func (e *connectionFailure) Error() string {
	return fmt.Sprintf("%s context=%s endpoint=%s user=%s: %s", e.Code, clean(e.ContextName), e.Endpoint, clean(e.User), e.Detail)
}

func readActiveContext() (config.JupiterContextRecord, string, error) {
	items, e := config.NewJupiterConfigList()
	if e != nil {
		return config.JupiterContextRecord{}, "", e
	}
	for _, item := range items {
		if item.Active {
			return item.Context, item.Name, nil
		}
	}
	return config.JupiterContextRecord{}, "", fmt.Errorf("no active rocontext")
}

func displayEndpoint(endpoint string) string {
	u, e := url.Parse(endpoint)
	if e != nil {
		return "(invalid endpoint)"
	}
	u.User, u.RawQuery, u.Fragment = nil, "", ""
	return clean(u.String())
}

func redactDiagnostic(message string, cfg config.JupiterContextRecord) string {
	plain, _ := contextPassword(cfg)
	for _, secret := range []string{cfg.Password, plain, base64.StdEncoding.EncodeToString([]byte(plain))} {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[redacted]")
		}
	}
	if cfg.Jupiter != "" {
		message = strings.ReplaceAll(message, cfg.Jupiter, displayEndpoint(cfg.Jupiter))
	}
	return strings.Join(strings.Fields(clean(message)), " ")
}

// This preflight performs read-only discovery and authentication. It never
// uploads, dispatches, or interprets a network/authentication error as legacy.
func connect(ctx context.Context, read contextReader) connectionResult {
	cfg, name, e := read()
	r := connectionResult{config: cfg, name: name}
	fail := func(code, message string, cause error) connectionResult {
		r.failure = &connectionFailure{Code: code, Message: message, Detail: redactDiagnostic(cause.Error(), cfg), Endpoint: displayEndpoint(cfg.Jupiter), ContextName: name, User: cfg.User}
		return r
	}
	if e != nil {
		return fail("CONFIG_FAILED", "rocontext 설정을 읽을 수 없습니다.", e)
	}
	check, cancel := context.WithTimeout(ctx, 7*time.Second)
	caps, e := transport.Discover(check, cfg.Jupiter)
	cancel()
	if errors.Is(e, transport.ErrLegacy) {
		r.legacy, r.reached = true, true
		return r
	}
	if e != nil {
		code, message := "CONNECT_FAILED", "Jupiter에 연결할 수 없습니다. 서버 주소·포트, 실행 상태와 VPN을 확인하세요."
		if strings.Contains(e.Error(), "HTTP 401") || strings.Contains(e.Error(), "HTTP 403") || status.Code(e) == codes.Unauthenticated {
			code, message = "AUTH_FAILED", "접속 인증이 거부됐습니다. rocontext와 게이트웨이 인증 설정을 확인하세요."
		} else if strings.Contains(e.Error(), "capabilit") {
			var network *url.Error
			if !errors.As(e, &network) {
				code, message = "API_UNAVAILABLE", "Jupiter 기능 확인에 실패했습니다. 서버 상태를 확인하세요."
			}
		}
		return fail(code, message, e)
	}
	r.reached = true
	if caps.Server != "jupiter" || !transport.Supports(caps, "rollouts") {
		return fail("API_UNAVAILABLE", "이 서버에서 신규 배포 기능을 사용할 수 없습니다.", fmt.Errorf("endpoint does not advertise Jupiter rollout support"))
	}
	c, e := NewClient(cfg)
	if e != nil {
		return fail("CONNECT_FAILED", "Jupiter 연결을 준비할 수 없습니다.", e)
	}
	if _, e = c.Context(ctx); e != nil {
		c.Close()
		code, message := "AUTH_FAILED", "Jupiter 인증에 실패했습니다. rocontext의 사용자·비밀번호를 확인하세요."
		var timeout net.Error
		if errors.Is(e, errContextPassword) {
			code, message = "CONFIG_FAILED", "rocontext의 비밀번호 설정을 읽을 수 없습니다. 해당 context를 다시 설정하세요."
		} else if status.Code(e) == codes.Unavailable || status.Code(e) == codes.DeadlineExceeded || errors.As(e, &timeout) || errors.Is(e, context.DeadlineExceeded) {
			code, message = "CONNECT_FAILED", "인증 중 Jupiter 연결이 끊겼거나 응답 시간이 초과됐습니다."
		} else if status.Code(e) == codes.PermissionDenied {
			code, message = "ACCESS_DENIED", "Jupiter 접근 권한이 없습니다. 계정 권한을 확인하세요."
		}
		return fail(code, message, e)
	}
	r.client = c
	return r
}
