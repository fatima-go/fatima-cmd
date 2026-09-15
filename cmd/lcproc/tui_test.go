package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func fixture(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("FATIMA_HOME", home)
	for _, tag := range []string{"2026_R001", "2026_R002"} {
		dir := filepath.Join(home, "app/revision/sample", tag)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(dir, "sample"), []byte("binary"), 0755)
		os.WriteFile(filepath.Join(dir, "application.yaml"), []byte("config"), 0644)
	}
	if err := os.Symlink("revision/sample/2026_R001", filepath.Join(home, "app/sample")); err != nil {
		t.Fatal(err)
	}
	return home
}
func press(m localModel, key tea.KeyType) localModel {
	next, _ := m.Update(tea.KeyMsg{Type: key})
	return next.(localModel)
}
func TestRevisionConfirmationAndRunningGuard(t *testing.T) {
	home := fixture(t)
	m := newLocalModel(nil)
	m = press(m, tea.KeyEnter)
	m = press(m, tea.KeyEnter)
	if m.stage != "version" || len(m.revisions) != 2 {
		t.Fatal(m)
	}
	m = press(m, tea.KeyEnter)
	if m.stage != "confirm" {
		t.Fatal(m.stage)
	}
	before, _ := os.Readlink(filepath.Join(home, "app/sample"))
	if !strings.Contains(before, "R001") {
		t.Fatal(before)
	}
	canceled := press(m, tea.KeyEsc)
	if canceled.stage != "action" {
		t.Fatal(canceled.stage)
	}
	dir := filepath.Join(home, "app/sample/proc")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "sample.pid"), []byte(fmt.Sprint(os.Getpid())), 0644)
	if err := switchLocalRevision(home, "sample", m.chosen.dir); err == nil {
		t.Fatal("changed running process")
	}
	os.Remove(filepath.Join(dir, "sample.pid"))
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !next.(localModel).busy {
		t.Fatal("missing execution wiring")
	}
	next, _ = next.(localModel).Update(cmd())
	if next.(localModel).err != nil || next.(localModel).stage != "result" {
		t.Fatal(next)
	}
	after, _ := os.Readlink(filepath.Join(home, "app/sample"))
	if !strings.Contains(after, "R002") {
		t.Fatal(after)
	}
}
func TestDuplicateAndFailures(t *testing.T) {
	home := fixture(t)
	m := newLocalModel([]string{"sample", "dup", "copy"})
	m = press(m, tea.KeyEnter)
	if m.stage != "confirm" {
		t.Fatal(m)
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := cmd().(localDone)
	if result.err != nil {
		t.Fatal(result.err)
	}
	b, err := os.ReadFile(filepath.Join(home, "app/copy/copy"))
	if err != nil || string(b) != "binary" {
		t.Fatal(string(b), err)
	}
	if _, err = os.Stat(filepath.Join(home, "app/copy/application.yaml")); err != nil {
		t.Fatal(err)
	}
	if err = duplicateLocal(home, "sample", "copy"); err == nil {
		t.Fatal("overwrote existing target")
	}
	for _, name := range []string{"../escape", "juno", "", "a/b"} {
		if err = duplicateLocal(home, "sample", name); err == nil {
			t.Fatal(name)
		}
	}
	if err = duplicateLocal(home, "missing", "partial"); err == nil {
		t.Fatal("copied absent source")
	}
	if _, err = os.Stat(filepath.Join(home, "app/revision/partial")); !os.IsNotExist(err) {
		t.Fatal("partial destination retained")
	}
}
func TestLocalViewsAndEmptyEnvironment(t *testing.T) {
	fixture(t)
	for _, args := range [][]string{nil, {"sample"}, {"sample", "version"}, {"sample", "version", "R002"}, {"sample", "dup", "copy"}} {
		m := newLocalModel(args)
		for _, size := range [][2]int{{60, 19}, {80, 24}, {132, 42}} {
			m.width, m.height = size[0], size[1]
			view := m.View()
			for _, line := range strings.Split(view, "\n") {
				if ansi.StringWidth(line) > size[0] {
					t.Fatal(line)
				}
			}
			if len(strings.Split(view, "\n")) > size[1] {
				t.Fatal(view)
			}
		}
	}
	t.Setenv("FATIMA_HOME", "")
	m := newLocalModel(nil)
	if m.err == nil || !strings.Contains(m.View(), "FATIMA_HOME") {
		t.Fatal(m)
	}
}
