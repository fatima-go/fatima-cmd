package controlui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/fatima-go/fatima-cmd/cipher"
	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-opm/api"
	"google.golang.org/grpc"
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
	endpoint := registryTestEndpoint(t, backend, &api.Capabilities{Server: "juno", ApiVersion: 2, PackageId: "host:package", Features: []string{"roproc", "rostop"}})
	for _, command := range []string{"roproc", "rostop"} {
		for _, count := range []int{0, 1, 2} {
			for _, explicit := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/explicit=%t", command, count, explicit), func(t *testing.T) {
					inventory := &pickerInventory{catalog: &api.PackageCatalog{}}
					for i := 0; i < count; i++ {
						inventory.catalog.Packages = append(inventory.catalog.Packages, &api.PackageEntry{Target: &api.Target{PackageId: "host:package"}})
					}
					gateway := grpc.NewServer()
					api.RegisterIdentityServer(gateway, &registryIdentity{})
					api.RegisterRoutingServer(gateway, &registryRouting{endpoint: endpoint})
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
	for _, command := range []string{"rostop", "roproc"} {
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
