package integration

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fatima-go/fatima-cmd/config"
	"github.com/fatima-go/fatima-cmd/deployui"
	"github.com/fatima-go/fatima-opm/api"
	"github.com/fatima-go/fatima-opm/artifact"
	jupiter "github.com/fatima-go/jupiter/deployment"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const policyDeployment = `{"process":"svc","process_type":"GENERAL","build":{"user":"dave"},"binaries":{` +
	`"svc":{"role":"process"},` +
	`"helper":{"role":"tool","strip":true},` +
	`"local-tool":{"role":"tool","strip":true,"install":{"default":false,"profiles":{"local":true}}}}}`

// policyFar gofar 와 같은 '/' 루트 형태로 두 플랫폼의 바이너리 3개를 담는다
func policyFar(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "svc.far")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	z := zip.NewWriter(f)
	names := []string{"/", "/deployment.json", "/platform/"}
	for _, p := range []string{"linux_amd64", "linux_arm64"} {
		names = append(names, "/platform/"+p+"/")
		for _, b := range []string{"svc", "helper", "local-tool"} {
			names = append(names, "/platform/"+p+"/"+b)
		}
	}
	for _, name := range names {
		w, e := z.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)})
		if e != nil {
			t.Fatal(e)
		}
		body := name
		if name == "/deployment.json" {
			body = policyDeployment
		}
		if !strings.HasSuffix(name, "/") {
			if _, e = w.Write([]byte(body)); e != nil {
				t.Fatal(e)
			}
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

func storedEntries(t *testing.T, path string) (map[string]bool, map[string]json.RawMessage) {
	t.Helper()
	z, e := zip.OpenReader(path)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	names := map[string]bool{}
	var dep map[string]json.RawMessage
	for _, f := range z.File {
		names[f.Name] = true
		if f.Name == "/deployment.json" {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			b, _ := io.ReadAll(r)
			r.Close()
			if e = json.Unmarshal(b, &dep); e != nil {
				t.Fatal(e)
			}
		}
	}
	return names, dep
}

func TestUploadAssemblesFarForJupiterProfile(t *testing.T) {
	cases := []struct {
		name          string
		profile       string
		hideFeature   bool
		removed       bool
		noticeContain string
	}{
		{"qa removes local tool", "qa", false, true, "local-tool"},
		{"local keeps original", "local", false, false, "원본"},
		{"jupiter without session_profile keeps original", "qa", true, false, "profile을 알려주지 않아"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gateway, e := jupiter.New(t.TempDir(), func(string, string) (string, error) { return "OPERATOR", nil }, func() ([]*api.Target, error) { return nil, nil })
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(gateway.Close)
			gateway.Profile = c.profile
			caps := proto.Clone(gateway.Capabilities()).(*api.Capabilities)
			if c.hideFeature {
				features := caps.Features[:0]
				for _, f := range caps.Features {
					if f != "session_profile" {
						features = append(features, f)
					}
				}
				caps.Features = features
			}
			g := grpc.NewServer()
			gateway.Register(g)
			endpoint := serve(t, g, caps)

			client, e := deployui.NewClient(config.NewJupiterContext("integration", endpoint, "operator", "password", "UTC").Context)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(client.Close)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			source := policyFar(t)
			sourceSum, _, _ := artifact.Digest(source)
			uploaded, e := client.Upload(ctx, source, "upload-"+strings.ReplaceAll(c.name, " ", "-"), nil)
			if e != nil {
				t.Fatal(e)
			}
			if uploaded.Filename != "svc.far" || strings.Join(uploaded.Platforms, ",") != "linux_amd64,linux_arm64" {
				t.Fatalf("artifact = %+v", uploaded)
			}
			if notice := client.AssemblyNotice(); !strings.Contains(notice, c.noticeContain) {
				t.Fatalf("notice %q does not contain %q", notice, c.noticeContain)
			}
			if now, _, _ := artifact.Digest(source); now != sourceSum {
				t.Fatal("original FAR changed")
			}

			names, dep := storedEntries(t, filepath.Join(gateway.Store.Dir, "artifacts", uploaded.Id+".far"))
			for _, p := range []string{"linux_amd64", "linux_arm64"} {
				if !names["/platform/"+p+"/svc"] || !names["/platform/"+p+"/helper"] {
					t.Fatalf("%s binaries are missing", p)
				}
				if names["/platform/"+p+"/local-tool"] == c.removed {
					t.Fatalf("%s local-tool presence = %v, removed = %v", p, names["/platform/"+p+"/local-tool"], c.removed)
				}
			}
			_, assembled := dep["assembly"]
			if assembled != c.removed {
				t.Fatalf("assembly recorded = %v, want %v", assembled, c.removed)
			}
			if !c.removed && uploaded.Sha256 != sourceSum {
				t.Fatal("original FAR must be uploaded unchanged")
			}
			if c.removed && !strings.Contains(string(dep["assembly"]), `"profile":"qa"`) {
				t.Fatalf("assembly = %s", dep["assembly"])
			}
		})
	}
}
