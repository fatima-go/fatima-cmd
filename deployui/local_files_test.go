package deployui

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func testFAR(t *testing.T, root, name string, modified time.Time) string {
	t.Helper()
	path := filepath.Join(root, "far", name, name+".far")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	for entry, content := range map[string]string{
		"/":                              "",
		"/deployment.json":               `{"process":"example","process_type":"GENERAL","build":{"user":"builder","git":{"branch":"feature/test","message":"first line\nsecond line"}}}`,
		"/platform/darwin_arm64/example": "test executable",
	} {
		w, err := z.Create(entry)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLocalFARListMatchesGofarAndRebuildOrder(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	now := time.Now().Truncate(time.Second)
	older := testFAR(t, first, "older", now.Add(-time.Hour))
	newer := testFAR(t, first, "newer", now)
	testFAR(t, second, "ignored", now.Add(time.Hour))
	paths := first + string(os.PathListSeparator) + second
	files := scanLocalFARs(paths)
	if files.Root != filepath.Join(first, "far") || len(files.Files) != 2 || files.Files[0].Path != newer {
		t.Fatalf("wrong gofar search root or order: %+v", files)
	}
	// gofar overwrites <process>.far, so sorting by creation time is incorrect.
	if err := os.Chtimes(older, now.Add(time.Minute), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if scanLocalFARs(paths).Files[0].Path != older {
		t.Fatal("rebuilt FAR did not move to the front")
	}
	if got := scanLocalFARs(""); len(got.Files) != 0 || got.Warning == "" {
		t.Fatal("missing GOPATH did not offer manual path entry")
	}
}

func TestPresetAndManualPathRequireSeparateUploadAction(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GOPATH", root)
	path := testFAR(t, root, "file with spaces", time.Now().Add(-time.Hour))
	testFAR(t, root, "newest", time.Now())
	m := newModel(context.Background(), nil, Options{Command: "upload", Value: `"` + path + `"`, RequestID: "preset-request"})
	_, inspect := m.Update(m.localScan()())
	if inspect == nil || m.busy || m.input != `"`+path+`"` {
		t.Fatal("preset path was replaced or upload was started")
	}
	m.Update(inspect())
	if m.preview == nil || m.preview.Err != nil || m.preview.Info.Process != "example" || m.input != path || m.opts.RequestID != "preset-request" {
		t.Fatalf("preset FAR not ready for review: %+v", m.preview)
	}
	m.Update(key("p"))
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m.Update(key(path))
	_, inspect = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if inspect == nil || m.editingPath || m.busy {
		t.Fatal("Enter in the path editor submitted an upload")
	}
	m.Update(inspect())
	if m.opts.RequestID != "" {
		t.Fatal("edited path reused another upload's request ID")
	}
	// Rebuilding after preview must require another review before upload.
	changed := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, changed, changed); err != nil {
		t.Fatal(err)
	}
	_, inspect = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if inspect == nil || m.busy || !m.previewPending {
		t.Fatal("changed FAR was uploaded without refreshing its preview")
	}
	m.Update(inspect())
	m.Update(key("p"))
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m.Update(key(filepath.Join(root, "missing.far")))
	_, inspect = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(inspect())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.busy || m.preview.Err == nil {
		t.Fatal("invalid path initiated an upload")
	}
}

func TestStalePreviewCannotReplaceCurrentSelection(t *testing.T) {
	m := newModel(context.Background(), nil, Options{Command: "upload", Value: "/new.far"})
	m.Update(event{kind: "preview", id: "/old.far", value: farPreview{File: localFAR{Path: "/old.far"}}})
	if m.input != "/new.far" || m.preview != nil {
		t.Fatal("stale preview replaced current selection")
	}
	m.editingPath = true
	m.Update(event{kind: "preview", id: m.input, value: farPreview{File: localFAR{Path: "/normalized.far"}}})
	if m.input != "/new.far" {
		t.Fatal("background preview overwrote active path entry")
	}
}
