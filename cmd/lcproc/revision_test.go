package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// v2Fixture mirrors a host after juno v2 deployments: legacy revisions from
// before, v2 revisions with manifests, and the link on the newest v2 one.
func v2Fixture(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("FATIMA_HOME", home)
	root := filepath.Join(home, "app/revision/knowledgesvc")
	day := func(d int) time.Time { return time.Date(2026, 10, d, 12, 0, 0, 0, time.Local) }
	write := func(name, file, body string) {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("2026.10.01-09.00_R007", "knowledgesvc", "old")
	write("2026.10.01-09.00_R007", deploymentJsonFile, `{"build":{"user":"jin","git":{"branch":"main","commit":"aaa111","message":"legacy"}}}`)
	for i, id := range []string{"o_03109260d321cce125ee4e58e8de1b55", "o_4d48a118375b22f077aef6914b56547e", "o_35e6d8ad1a5330e2e567bcab494729e4"} {
		name := "v2_" + id
		write(name, "knowledgesvc", "binary")
		write(name, deploymentJsonFile, `{"build":{"user":"dave","git":{"branch":"release","commit":"bbb22`+string(rune('0'+i))+`","message":"v2 build\nbody"}}}`)
		write(name, v2ManifestName, `{"OperationID":"`+id+`","ArtifactSHA256":"`+strings.Repeat("c", 64)+`"}`)
		manifest := filepath.Join(root, name, v2ManifestName)
		if err := os.Chtimes(manifest, day(17+i), day(17+i)); err != nil {
			t.Fatal(err)
		}
	}
	write("v2_o_03109260d321cce125ee4e58e8de1b55_previous", "knowledgesvc", "plain")
	os.Chtimes(filepath.Join(root, "v2_o_03109260d321cce125ee4e58e8de1b55_previous"), day(10), day(10))
	write("notes", "readme", "not a revision")
	if err := os.Symlink("revision/knowledgesvc/v2_o_35e6d8ad1a5330e2e567bcab494729e4", filepath.Join(home, "app/knowledgesvc")); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestV2RevisionsAreListedNewestFirst(t *testing.T) {
	home := v2Fixture(t)
	revisions, current, err := localRevisions(home, "knowledgesvc")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"o_35e6d8ad1a", "o_4d48a11837", "o_03109260d3", "o_03109260d3_previous", "R007"}
	got := []string{}
	for _, r := range revisions {
		got = append(got, r.revision)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatal(got)
	}
	if same, _ := sameRevision(current, revisions[0].dir); !same {
		t.Fatal("newest v2 revision should be current")
	}
	r := revisions[0]
	if r.kind != revisionKindV2 || r.operationID != "o_35e6d8ad1a5330e2e567bcab494729e4" || r.deployment.Git.Commit != "bbb222" || !strings.HasPrefix(r.createDtime, "2026-10-19") {
		t.Fatalf("%+v", r)
	}
	if revisions[3].kind != revisionKindPrevious || revisions[4].kind != revisionKindLegacy || revisions[4].number != 7 {
		t.Fatal(revisions[3].kind, revisions[4].kind)
	}
	for _, key := range []string{"O_4D48A11837", "v2_o_4d48a118375b22f077aef6914b56547e"} {
		if found, err := findRevision(revisions, key); err != nil || found.dir != revisions[1].dir {
			t.Fatal(key, err)
		}
	}
	if _, err := findRevision(revisions, "R008"); err == nil {
		t.Fatal("found missing revision")
	}
	if _, err := findRevision(append(revisions, revisions[1]), "o_4d48a11837"); err == nil || !strings.Contains(err.Error(), "여러 개") {
		t.Fatal("ambiguous label accepted", err)
	}
}

func TestSwitchToOlderV2RevisionThroughTUI(t *testing.T) {
	home := v2Fixture(t)
	m := newLocalModel([]string{"knowledgesvc", "version", "o_4d48a11837"})
	if m.stage != "confirm" || m.err != nil {
		t.Fatal(m.stage, m.err)
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "o_35e6d8ad1a") || !strings.Contains(view, "o_4d48a11837") {
		t.Fatal(view)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next, _ = next.Update(cmd())
	if m = next.(localModel); m.stage != "result" || m.err != nil {
		t.Fatal(m.stage, m.err)
	}
	link, _ := os.Readlink(filepath.Join(home, "app/knowledgesvc"))
	if link != "revision/knowledgesvc/v2_o_4d48a118375b22f077aef6914b56547e" {
		t.Fatal(link)
	}
	// Back on the list, the process row shows the new current revision.
	m = press(m, tea.KeyEnter)
	if m.stage != "process" || m.procs[m.procIndex("knowledgesvc")].current.revision != "o_4d48a11837" {
		t.Fatal(m.stage, m.procs)
	}
}

func TestPlainVersionListsV2Revisions(t *testing.T) {
	v2Fixture(t)
	oldArgs, oldProc, oldStdout := os.Args, proc, os.Stdout
	defer func() { os.Args, proc, os.Stdout = oldArgs, oldProc, oldStdout }()
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Args = []string{"lcproc", "knowledgesvc", "version"}
	proc = "knowledgesvc"
	versioning()
	w.Close()
	os.Stdout = oldStdout
	out := make([]byte, 8192)
	n, _ := r.Read(out)
	text := string(out[:n])
	if !strings.Contains(text, "o_35e6d8ad1a") || !strings.Contains(text, "[O]") || !strings.Contains(text, "R007") {
		t.Fatal(text)
	}
}

func TestV2ViewsFitTerminal(t *testing.T) {
	v2Fixture(t)
	for _, args := range [][]string{nil, {"knowledgesvc"}, {"knowledgesvc", "version"}, {"knowledgesvc", "version", "R007"}, {"knowledgesvc", "dup", "copy"}} {
		m := newLocalModel(args)
		for _, size := range [][2]int{{60, 19}, {80, 24}, {100, 30}, {132, 42}} {
			m.width, m.height = size[0], size[1]
			for _, detail := range []bool{false, true} {
				m.detail = detail
				view := m.View()
				lines := strings.Split(view, "\n")
				if len(lines) != size[1] {
					t.Fatalf("%v %v: %d lines", args, size, len(lines))
				}
				for _, l := range lines {
					if ansi.StringWidth(l) > size[0] {
						t.Fatalf("%v %v: %q", args, size, ansi.Strip(l))
					}
				}
			}
		}
	}
}

func TestDetailScrollIsBounded(t *testing.T) {
	v2Fixture(t)
	m := newLocalModel([]string{"knowledgesvc", "version"})
	m.width, m.height = 60, 19
	m = press(m, tea.KeyTab)
	for i := 0; i < 50; i++ {
		m = press(m, tea.KeyDown)
	}
	if m.offset == 0 || m.offset != m.detailScrollLimit() || m.cursor != 0 {
		t.Fatal(m.offset, m.detailScrollLimit(), m.cursor)
	}
	m = press(m, tea.KeyUp)
	if m.offset != m.detailScrollLimit()-1 {
		t.Fatal(m.offset)
	}
}
