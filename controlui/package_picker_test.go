package controlui

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-cmd/cipher"
	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-opm/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type pickerInventory struct {
	api.UnimplementedPackageInventoryServer
	catalog *api.PackageCatalog
}

func (p *pickerInventory) List(context.Context, *api.Empty) (*api.PackageCatalog, error) {
	return p.catalog, nil
}

func TestControlPackagePickerRouting(t *testing.T) {
	registry := &directRegistry{calls: map[string]int{}}
	backend := grpc.NewServer()
	api.RegisterProcessRegistryServer(backend, registry)
	endpoint := registryTestEndpoint(t, backend, &api.Capabilities{Server: "juno", ApiVersion: 2, PackageId: "host:package", Features: []string{"roproc", "rostop", "rostart", "rocron", "rolog", "rohis"}})
	for _, command := range []string{"roproc", "rostop", "rostart", "rocron", "rolog", "rohis"} {
		for _, count := range []int{0, 1, 2} {
			for _, explicit := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/explicit=%t", command, count, explicit), func(t *testing.T) {
					inventory := &pickerInventory{catalog: &api.PackageCatalog{}}
					for i := 0; i < count; i++ {
						inventory.catalog.Packages = append(inventory.catalog.Packages, &api.PackageEntry{Target: &api.Target{PackageId: "host:package"}})
					}
					gateway := grpc.NewServer()
					api.RegisterIdentityServer(gateway, &registryIdentity{})
					api.RegisterRoutingServer(gateway, &pickerRouting{registryRouting: registryRouting{endpoint: endpoint}, emptyCode: codes.NotFound})
					api.RegisterPackageInventoryServer(gateway, inventory)
					url := registryTestEndpoint(t, gateway, &api.Capabilities{Server: "jupiter", ApiVersion: 2, Features: []string{"routing"}})
					password, _ := cipher.Aes256Encode("password")
					opts := Options{Command: command, pickPackage: true}
					if explicit {
						opts.Package = "host:package"
					}
					c, err := Connect(context.Background(), config.JupiterContextRecord{Jupiter: url, Password: password}, "test", opts)
					if err != nil {
						t.Fatal(err)
					}
					defer c.Close()
					picker := !explicit && count != 1
					if (c.backend == nil) != picker {
						t.Fatal("wrong routing mode", c.Target)
					}
					if picker {
						m := model{ctx: context.Background(), opts: opts, client: c}
						result := m.load()().(loaded)
						if result.err != nil || len(result.packages.Packages) != count {
							t.Fatal(result)
						}
						next, cmd := m.Update(result)
						if next.(model).stage != "package" || cmd != nil {
							t.Fatal("command proceeded before selection")
						}
					}
				})
			}
		}
	}
}

func TestControlPackagePickerPreservesActionAndProcessGroup(t *testing.T) {
	for _, command := range []string{"rostop", "roproc", "rostart", "rocron", "rolog", "rohis"} {
		m := model{ctx: context.Background(), opts: Options{Command: command, pickPackage: true, Action: "remove", Process: "worker", Group: "svc"}, stage: "select", width: 100, height: 30, client: &Client{Target: &api.Target{PackageId: "패키지 선택"}}}
		catalog := &api.PackageCatalog{Packages: []*api.PackageEntry{
			{Target: &api.Target{PackageId: "a:default", Group: "backend"}},
			{Target: &api.Target{PackageId: "b:default", Group: "backend"}},
		}}
		next, cmd := m.Update(loaded{packages: catalog})
		m = next.(model)
		if cmd != nil || m.stage != "package" || m.count() != 2 {
			t.Fatal("process group incorrectly filtered package list")
		}
		for _, size := range [][2]int{{60, 19}, {80, 24}, {132, 42}} {
			m.width, m.height = size[0], size[1]
			view := ansi.Strip(m.View())
			if !strings.Contains(view, "a:default") || !strings.Contains(view, "b:default") {
				t.Fatal(view)
			}
			for _, line := range strings.Split(view, "\n") {
				if ansi.StringWidth(line) > m.width {
					t.Fatal("view overflow", line)
				}
			}
			if len(strings.Split(view, "\n")) > m.height {
				t.Fatal("height overflow")
			}
		}
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		next, cmd = next.(model).Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = next.(model)
		if cmd == nil || m.opts.Package != "b:default" || m.opts.Process != "worker" || m.opts.Action != "remove" || m.opts.Group != "svc" || !m.busy {
			t.Fatal(m.opts)
		}
		next, _ = m.Update(connected{err: fmt.Errorf("offline")})
		if next.(model).busy {
			t.Fatal("connection error prevents retry")
		}
		next.(model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	}
}

func TestPackagePromptModeParsing(t *testing.T) {
	for _, tc := range []struct {
		command string
		args    []string
		blocked bool
	}{
		{"rocron", []string{"-l"}, false},
		{"rolog", []string{"worker", "info"}, false},
		{"rohis", []string{"worker"}, false},
		{"rohis", []string{"-g", "svc"}, false},
		{"rocron", []string{"--plain", "-l"}, true},
		{"rohis", []string{"--json"}, true},
	} {
		o, err := parse(tc.command, tc.args)
		if err != nil {
			t.Fatal(err)
		}
		if o.noPackagePrompt != tc.blocked {
			t.Fatalf("%s %v: %+v", tc.command, tc.args, o)
		}
	}
}

func TestNonInteractivePackageSelection(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			inventory := &pickerInventory{catalog: &api.PackageCatalog{}}
			for i := 0; i < count; i++ {
				inventory.catalog.Packages = append(inventory.catalog.Packages, &api.PackageEntry{Target: &api.Target{PackageId: "host:package"}})
			}
			backend := grpc.NewServer()
			endpoint := registryTestEndpoint(t, backend, &api.Capabilities{Server: "juno", ApiVersion: 2, PackageId: "host:package", Features: []string{"rocron"}})
			gateway := grpc.NewServer()
			api.RegisterIdentityServer(gateway, &registryIdentity{})
			api.RegisterRoutingServer(gateway, &pickerRouting{registryRouting: registryRouting{endpoint: endpoint}, emptyCode: codes.NotFound})
			api.RegisterPackageInventoryServer(gateway, inventory)
			url := registryTestEndpoint(t, gateway, &api.Capabilities{Server: "jupiter", ApiVersion: 2, Features: []string{"routing"}})
			password, _ := cipher.Aes256Encode("password")
			c, err := Connect(context.Background(), config.JupiterContextRecord{Jupiter: url, Password: password}, "test", Options{Command: "rocron"})
			if count == 1 {
				if err != nil {
					t.Fatal(err)
				}
				c.Close()
			} else if err == nil {
				c.Close()
				t.Fatal("ambiguous package was resolved")
			}
		})
	}
}

func TestReopenedPackagePickerShowsOtherPackages(t *testing.T) {
	for _, command := range []string{"rolog", "rohis", "rostart", "rostop", "roproc", "rocron"} {
		m := model{stage: "package", opts: Options{Command: command, Package: "a:default", Group: "svc"}, packages: &api.PackageCatalog{Packages: []*api.PackageEntry{
			{Target: &api.Target{PackageId: "a:default", Group: "backend"}},
			{Target: &api.Target{PackageId: "b:default", Group: "backend"}},
		}}}
		if len(m.visiblePackages()) != 2 {
			t.Fatal(command, "hid other packages")
		}
	}
}

type pickerRouting struct {
	registryRouting
	emptyCode codes.Code
}

func (r *pickerRouting) Resolve(ctx context.Context, q *api.PackageQuery) (*api.Target, error) {
	if q.PackageId == "" && r.emptyCode != codes.OK {
		return nil, status.Error(r.emptyCode, "routing test")
	}
	return r.registryRouting.Resolve(ctx, q)
}

func TestIPRoutingPrecedesPackagePicker(t *testing.T) {
	for _, command := range []string{"roproc", "rostop", "rostart", "rocron", "rolog", "rohis"} {
		for _, interactive := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/interactive=%t", command, interactive), func(t *testing.T) {
				registry := &directRegistry{calls: map[string]int{}}
				backend := grpc.NewServer()
				api.RegisterProcessRegistryServer(backend, registry)
				endpoint := registryTestEndpoint(t, backend, &api.Capabilities{Server: "juno", ApiVersion: 2, PackageId: "host:package", Features: []string{command}})
				gateway := grpc.NewServer()
				api.RegisterIdentityServer(gateway, &registryIdentity{})
				api.RegisterRoutingServer(gateway, &pickerRouting{registryRouting: registryRouting{endpoint: endpoint}})
				// No inventory service: successful peer-IP resolution must not need it.
				url := registryTestEndpoint(t, gateway, &api.Capabilities{Server: "jupiter", ApiVersion: 2, Features: []string{"routing"}})
				password, _ := cipher.Aes256Encode("password")
				c, err := Connect(context.Background(), config.JupiterContextRecord{Jupiter: url, Password: password}, "test", Options{Command: command, pickPackage: interactive})
				if err != nil {
					t.Fatal(err)
				}
				defer c.Close()
				if c.backend == nil || c.Target.PackageId != "host:package" {
					t.Fatal("IP resolved target was not used")
				}
			})
		}
	}
}

func TestRoutingErrorsDoNotOpenPicker(t *testing.T) {
	for _, code := range []codes.Code{codes.PermissionDenied, codes.Unauthenticated, codes.Unavailable} {
		gateway := grpc.NewServer()
		api.RegisterIdentityServer(gateway, &registryIdentity{})
		api.RegisterRoutingServer(gateway, &pickerRouting{emptyCode: code})
		url := registryTestEndpoint(t, gateway, &api.Capabilities{Server: "jupiter", ApiVersion: 2, Features: []string{"routing"}})
		password, _ := cipher.Aes256Encode("password")
		_, err := Connect(context.Background(), config.JupiterContextRecord{Jupiter: url, Password: password}, "test", Options{Command: "rocron", pickPackage: true})
		if status.Code(err) != code {
			t.Fatal(code, err)
		}
	}
}

func TestInitialPickerKeepsLocalCandidates(t *testing.T) {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	var local net.IP
	for _, a := range addresses {
		ip, _, e := net.ParseCIDR(a.String())
		if e == nil && !ip.IsLoopback() && !ip.IsUnspecified() {
			local = ip
			break
		}
	}
	if local == nil {
		t.Skip("no non-loopback interface")
	}
	gateway := grpc.NewServer()
	api.RegisterIdentityServer(gateway, &registryIdentity{})
	api.RegisterRoutingServer(gateway, &pickerRouting{emptyCode: codes.FailedPrecondition})
	api.RegisterPackageInventoryServer(gateway, &pickerInventory{catalog: &api.PackageCatalog{Packages: []*api.PackageEntry{
		{Target: &api.Target{PackageId: "local:a", Endpoint: "http://" + net.JoinHostPort(local.String(), "9180")}},
		{Target: &api.Target{PackageId: "local:b", Endpoint: "http://" + net.JoinHostPort(local.String(), "9181")}},
		{Target: &api.Target{PackageId: "remote:a", Endpoint: "http://192.0.2.254:9180"}},
	}}})
	url := registryTestEndpoint(t, gateway, &api.Capabilities{Server: "jupiter", ApiVersion: 2, Features: []string{"routing"}})
	password, _ := cipher.Aes256Encode("password")
	opts := Options{Command: "rocron", pickPackage: true}
	c, err := Connect(context.Background(), config.JupiterContextRecord{Jupiter: url, Password: password}, "test", opts)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	m := model{ctx: context.Background(), client: c, opts: opts}
	loaded := m.load()().(loaded)
	if loaded.err != nil || len(loaded.packages.Packages) != 2 {
		t.Fatal(loaded)
	}
	// The one-shot report path uses the same candidate method.
	report, err := c.SelectionPackages(context.Background())
	if err != nil || len(report.Packages) != 2 {
		t.Fatal(report, err)
	}
	// Manually reopening a picker must not permanently hide remote packages.
	all, err := c.Packages(context.Background())
	if err != nil || len(all.Packages) != 3 {
		t.Fatal(all, err)
	}
}
