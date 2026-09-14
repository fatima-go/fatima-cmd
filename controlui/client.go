package controlui

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/fatima-go/fatima-cmd/cipher"
	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-core/opm/api"
	"github.com/fatima-go/fatima-core/opm/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sync"
	"time"
)

type Client struct {
	Config           config.JupiterContextRecord
	Name             string
	Target           *api.Target
	Capabilities     *api.Capabilities
	gateway, backend *grpc.ClientConn
	session          *api.Session
	mu               sync.Mutex
}

func password(cfg config.JupiterContextRecord) (value string, err error) {
	defer func() {
		if recover() != nil {
			value = ""
			err = fmt.Errorf("invalid rocontext password; update the saved context")
		}
	}()
	value, err = cipher.Aes256Decode(cfg.Password)
	if err != nil {
		return "", fmt.Errorf("invalid rocontext password; update the saved context")
	}
	return
}
func connect(ctx context.Context, opts Options) (*Client, error) {
	list, err := config.NewJupiterConfigList()
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		if item.Active {
			return Connect(ctx, item.Context, item.Name, opts)
		}
	}
	return nil, fmt.Errorf("no active rocontext")
}
func Connect(ctx context.Context, cfg config.JupiterContextRecord, name string, opts Options) (result *Client, err error) {
	c := &Client{Config: cfg, Name: name}
	defer func() {
		if err != nil {
			c.Close()
		}
	}()
	caps, err := transport.Discover(ctx, cfg.Jupiter)
	if err != nil {
		return nil, err
	}
	if caps.Server != "jupiter" {
		return nil, fmt.Errorf("expected Jupiter endpoint")
	}
	if transport.Supports(caps, "unavailable") {
		return nil, fmt.Errorf("Jupiter APIs are unavailable")
	}
	feature := "routing"
	if opts.Command == "ropack" {
		feature = opts.Command
	}
	if !transport.Supports(caps, feature) {
		return nil, transport.ErrLegacy
	}
	c.gateway, err = transport.Dial(cfg.Jupiter)
	if err != nil {
		return nil, err
	}
	auth, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	if opts.Command == "ropack" {
		c.Target = &api.Target{PackageId: "전체 패키지"}
		c.Capabilities = caps
		return c, nil
	}
	c.Target, err = api.NewRoutingClient(c.gateway).Resolve(auth, &api.PackageQuery{PackageId: opts.Package})
	if code := status.Code(err); opts.pickPackage && opts.Package == "" && (code == codes.FailedPrecondition || code == codes.NotFound) {
		// Several packages (or none) match this client; the rolog screen lists them.
		c.Target = &api.Target{PackageId: "패키지 선택"}
		c.Capabilities = caps
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	caps, err = transport.Discover(ctx, c.Target.Endpoint)
	if err != nil {
		return nil, err
	}
	if caps.Server != "juno" || caps.PackageId != c.Target.PackageId {
		return nil, fmt.Errorf("Juno registration changed; refresh package selection")
	}
	if !transport.Supports(caps, opts.Command) {
		if transport.Supports(caps, "unavailable") || transport.Supports(caps, "control_unavailable") {
			return nil, fmt.Errorf("Juno APIs are unavailable")
		}
		return nil, transport.ErrLegacy
	}
	c.backend, err = transport.Dial(c.Target.Endpoint)
	c.Capabilities = caps
	if err != nil {
		return nil, err
	}
	if opts.Command == "roproc" {
		_, err = c.Registry(ctx)
		if status.Code(err) == codes.Unimplemented {
			return nil, transport.ErrLegacy
		}
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}
func (c *Client) Close() {
	if c.backend != nil {
		c.backend.Close()
	}
	if c.gateway != nil {
		c.gateway.Close()
	}
}
func (c *Client) Context(ctx context.Context) (context.Context, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == nil || c.session.ExpiresAt < time.Now().Add(time.Minute).Unix() {
		plain, err := password(c.Config)
		if err != nil {
			return nil, err
		}
		check, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		c.session, err = api.NewIdentityClient(c.gateway).Login(check, &api.LoginRequest{Username: c.Config.User, Password: "b64:" + base64.StdEncoding.EncodeToString([]byte(plain))})
		if err != nil {
			return nil, err
		}
	}
	return transport.WithToken(ctx, c.session.Token), nil
}
func (c *Client) Cron(ctx context.Context) (*api.CronCatalog, error) {
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	return api.NewCronControlClient(c.backend).List(ctx, &api.Empty{})
}
func (c *Client) RunCron(ctx context.Context, q *api.CronRequest) (*api.ControlOperation, error) {
	if err := c.checkTarget(ctx, "rocron"); err != nil {
		return nil, err
	}
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	return api.NewCronControlClient(c.backend).Rerun(ctx, q)
}
func (c *Client) Get(ctx context.Context, command, id string) (*api.ControlOperation, error) {
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	if command == "roproc" {
		return api.NewProcessRegistryClient(c.backend).Get(ctx, &api.RegistryOperationQuery{PackageId: c.Target.PackageId, Id: id})
	}
	if command != "rocron" {
		return api.NewProcessControlClient(c.backend).Get(ctx, &api.OperationQuery{Id: id})
	}
	return api.NewCronControlClient(c.backend).Get(ctx, &api.OperationQuery{Id: id})
}
func (c *Client) Watch(ctx context.Context, command, id string) (grpc.ServerStreamingClient[api.ControlOperation], error) {
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	if command == "roproc" {
		return api.NewProcessRegistryClient(c.backend).Watch(ctx, &api.RegistryOperationQuery{PackageId: c.Target.PackageId, Id: id})
	}
	if command != "rocron" {
		return api.NewProcessControlClient(c.backend).Watch(ctx, &api.OperationQuery{Id: id})
	}
	return api.NewCronControlClient(c.backend).Watch(ctx, &api.OperationQuery{Id: id})
}
func (c *Client) Processes(ctx context.Context) (*api.ProcessCatalog, error) {
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	return api.NewProcessControlClient(c.backend).Catalog(ctx, &api.ProcessQuery{Timezone: c.Config.Timezone})
}
func (c *Client) Submit(ctx context.Context, o Options) (*api.ControlOperation, error) {
	if o.Command == "roproc" {
		if o.Plan == nil || o.Plan.Request == nil || o.Plan.Request.PackageId != c.Target.PackageId || o.Plan.Request.RequestId != o.RequestID {
			return nil, fmt.Errorf("a matching registration preview is required")
		}
		ctx, err := c.Context(ctx)
		if err != nil {
			return nil, err
		}
		return api.NewProcessRegistryClient(c.backend).Apply(ctx, o.Plan.Request)
	}
	if o.Command == "rocron" {
		return c.RunCron(ctx, &api.CronRequest{RequestId: o.RequestID, Process: o.Process, Job: o.Job, Arguments: o.Arguments})
	}
	if o.Command != "rostart" && o.Command != "rostop" {
		return nil, fmt.Errorf("%s is not a process mutation", o.Command)
	}
	if err := c.checkTarget(ctx, o.Command); err != nil {
		return nil, err
	}
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	q := &api.ProcessRequest{RequestId: o.RequestID, Processes: o.Targets}
	if o.Command == "rostart" {
		return api.NewProcessControlClient(c.backend).Start(ctx, q)
	}
	return api.NewProcessControlClient(c.backend).Stop(ctx, q)
}
func (c *Client) checkTarget(ctx context.Context, feature string) error {
	caps, err := api.NewDiscoveryClient(c.backend).GetCapabilities(ctx, &api.Empty{})
	if err != nil {
		return err
	}
	if caps.Server != "juno" || caps.PackageId != c.Target.PackageId || !transport.Supports(caps, feature) {
		return fmt.Errorf("Juno identity or feature changed; reconnect before issuing a command")
	}
	return nil
}
func legacy(err error) bool { return errors.Is(err, transport.ErrLegacy) }
func (c *Client) WatchProcesses(ctx context.Context) (grpc.ServerStreamingClient[api.ProcessCatalog], error) {
	ctx, err := c.Context(ctx)
	if err != nil {
		return nil, err
	}
	return api.NewProcessControlClient(c.backend).WatchCatalog(ctx, &api.ProcessQuery{Timezone: c.Config.Timezone})
}
