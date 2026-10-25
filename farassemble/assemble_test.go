package farassemble

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

const knowledgeDeployment = `{"build":{"time":"2026-10-25 12:00:00 KST","user":"dave"},"process":"knowledgesvc","process_type":"GENERAL",` +
	`"binaries":{` +
	`"knowledgesvc":{"role":"process"},` +
	`"knowledge-query":{"role":"tool","strip":true},` +
	`"knowledge-backfill":{"role":"tool","strip":true,"install":{"default":false,"profiles":{"local":true}}},` +
	`"knowledge-scope-review":{"role":"tool","strip":true,"install":{"default":false,"profiles":{"local":true}}}}}`

var platforms = []string{"darwin_arm64", "linux_amd64", "linux_arm64"}
var binaries = []string{"knowledge-backfill", "knowledge-query", "knowledge-scope-review", "knowledgesvc"}

// writeGofarFar gofar 와 같은 형태('/' 루트, Deflate)의 FAR 를 만든다
func writeGofarFar(t *testing.T, path, deployment string, rootBinaries bool) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	modified := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	add := func(name string, data []byte, mode os.FileMode) {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: modified}
		if mode != 0 {
			h.SetMode(mode)
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if data != nil {
			if _, err = w.Write(data); err != nil {
				t.Fatal(err)
			}
		}
	}
	add("/", nil, 0)
	add("/application.yaml", []byte("fatima:\n  name: knowledgesvc\n"), 0)
	add("/deployment.json", []byte(deployment), 0)
	if rootBinaries {
		for _, b := range binaries {
			add("/"+b, []byte("bin-"+b), 0755)
		}
	} else {
		add("/platform/", nil, 0755)
		for _, p := range platforms {
			add("/platform/"+p+"/", nil, 0755)
			for _, b := range binaries {
				add("/platform/"+p+"/"+b, []byte(p+"-"+b), 0755)
			}
		}
	}
	if err = zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func entries(t *testing.T, path string) map[string]*zip.File {
	t.Helper()
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { z.Close() })
	m := make(map[string]*zip.File)
	for _, f := range z.File {
		m[f.Name] = f
	}
	return m
}

func content(t *testing.T, f *zip.File) []byte {
	t.Helper()
	r, err := f.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestForProfileRemovesBinaries(t *testing.T) {
	dir := t.TempDir()
	far := filepath.Join(dir, "knowledgesvc.far")
	writeGofarFar(t, far, knowledgeDeployment, false)
	original, _ := os.ReadFile(far)

	result, cleanup, err := ForProfile(far, "qa")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Assembled || !result.HasPolicy || result.Path == far {
		t.Fatalf("unexpected result %+v", result)
	}
	if filepath.Base(result.Path) != "knowledgesvc.far" {
		t.Fatalf("assembled file name = %s", filepath.Base(result.Path))
	}
	if got := strings.Join(result.Removed, ","); got != "knowledge-backfill,knowledge-scope-review" {
		t.Fatalf("removed = %s", got)
	}

	src := entries(t, far)
	dst := entries(t, result.Path)
	for name, f := range src {
		base := filepath.Base(name)
		removed := strings.HasPrefix(name, "/platform/") && (base == "knowledge-backfill" || base == "knowledge-scope-review") && !f.FileInfo().IsDir()
		g, ok := dst[name]
		if removed {
			if ok {
				t.Fatalf("%s must be removed", name)
			}
			continue
		}
		if !ok {
			t.Fatalf("%s is missing", name)
		}
		if name == "/deployment.json" {
			continue
		}
		if !bytes.Equal(content(t, f), content(t, g)) || f.Mode() != g.Mode() || !f.Modified.Equal(g.Modified) {
			t.Fatalf("%s is changed", name)
		}
	}

	var dep map[string]json.RawMessage
	if err = json.Unmarshal(content(t, dst["/deployment.json"]), &dep); err != nil {
		t.Fatal(err)
	}
	var assembly Assembly
	if err = json.Unmarshal(dep["assembly"], &assembly); err != nil {
		t.Fatal(err)
	}
	if assembly.Profile != "qa" || len(assembly.Removed) != 2 || len(assembly.SourceSha256) != 64 {
		t.Fatalf("assembly = %+v", assembly)
	}
	var orig map[string]json.RawMessage
	_ = json.Unmarshal([]byte(knowledgeDeployment), &orig)
	for k, v := range orig {
		if !bytes.Equal(dep[k], v) {
			t.Fatalf("deployment.json %s changed: %s", k, dep[k])
		}
	}
	if !dst["/deployment.json"].Modified.Equal(src["/deployment.json"].Modified) {
		t.Fatal("deployment.json modified time changed")
	}

	if now, _ := os.ReadFile(far); !bytes.Equal(original, now) {
		t.Fatal("original FAR changed")
	}

	cleanup()
	if _, err = os.Stat(filepath.Dir(result.Path)); !os.IsNotExist(err) {
		t.Fatalf("temporary directory remains: %v", err)
	}
}

func TestForProfileIsDeterministic(t *testing.T) {
	far := filepath.Join(t.TempDir(), "knowledgesvc.far")
	writeGofarFar(t, far, knowledgeDeployment, false)
	first, cleanup1, err := ForProfile(far, "qa")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup1()
	second, cleanup2, err := ForProfile(far, "qa")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup2()
	a, _ := os.ReadFile(first.Path)
	b, _ := os.ReadFile(second.Path)
	if first.Path == second.Path || !bytes.Equal(a, b) {
		t.Fatal("assembling the same FAR for the same profile must produce identical bytes")
	}
}

func TestForProfileKeepsOriginal(t *testing.T) {
	dir := t.TempDir()
	withPolicy := filepath.Join(dir, "policy.far")
	writeGofarFar(t, withPolicy, knowledgeDeployment, false)
	noPolicy := filepath.Join(dir, "legacy.far")
	writeGofarFar(t, noPolicy, `{"process":"knowledgesvc","process_type":"GENERAL"}`, false)

	cases := []struct {
		name, far, profile string
		hasPolicy          bool
	}{
		{"local installs everything", withPolicy, "local", true},
		{"far without binaries", noPolicy, "qa", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, cleanup, err := ForProfile(c.far, c.profile)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			if result.Assembled || result.Path != c.far || result.HasPolicy != c.hasPolicy {
				t.Fatalf("unexpected result %+v", result)
			}
		})
	}
}

func TestForProfileAlreadyAssembled(t *testing.T) {
	far := filepath.Join(t.TempDir(), "knowledgesvc.far")
	writeGofarFar(t, far, knowledgeDeployment, false)
	qa, cleanup, err := ForProfile(far, "qa")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if _, _, err = ForProfile(qa.Path, "local"); err == nil || !strings.Contains(err.Error(), "already assembled") {
		t.Fatalf("assembled FAR must be refused for another profile: %v", err)
	}
	again, cleanupAgain, err := ForProfile(qa.Path, "qa")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupAgain()
	if again.Assembled || again.Path != qa.Path {
		t.Fatalf("same profile must reuse the assembled FAR: %+v", again)
	}
}

func TestForProfileRootBinaries(t *testing.T) {
	// 플랫폼 축소되어 루트에 바이너리가 있는 FAR
	far := filepath.Join(t.TempDir(), "knowledgesvc.far")
	writeGofarFar(t, far, knowledgeDeployment, true)
	result, cleanup, err := ForProfile(far, "qa")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	names := make([]string, 0)
	for name := range entries(t, result.Path) {
		names = append(names, name)
	}
	sort.Strings(names)
	if got := strings.Join(names, ","); got != "/,/application.yaml,/deployment.json,/knowledge-query,/knowledgesvc" {
		t.Fatalf("entries = %s", got)
	}
}

func TestForProfileRefusesProcessRemoval(t *testing.T) {
	far := filepath.Join(t.TempDir(), "knowledgesvc.far")
	writeGofarFar(t, far, `{"process":"knowledgesvc","binaries":{"knowledgesvc":{"role":"process","install":{"default":false}}}}`, false)
	if _, _, err := ForProfile(far, "qa"); err == nil || !strings.Contains(err.Error(), "process binary") {
		t.Fatalf("process removal must be refused: %v", err)
	}
}

func TestInstalled(t *testing.T) {
	localOnly := BinarySpec{Install: &InstallRule{Default: false, Profiles: map[string]bool{"local": true}}}
	exceptProd := BinarySpec{Install: &InstallRule{Default: true, Profiles: map[string]bool{"prod": false}}}
	cases := []struct {
		spec    BinarySpec
		profile string
		want    bool
	}{
		{BinarySpec{}, "qa", true},
		{BinarySpec{}, "", true},
		{localOnly, "local", true},
		{localOnly, "qa", false},
		{localOnly, "", false},
		{exceptProd, "prod", false},
		{exceptProd, "qa", true},
	}
	for _, c := range cases {
		if got := c.spec.Installed(c.profile); got != c.want {
			t.Fatalf("Installed(%q) = %v, want %v (%+v)", c.profile, got, c.want, c.spec.Install)
		}
	}
}
