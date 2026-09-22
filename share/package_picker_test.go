package share

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestPackageChoicePolicy(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		choices := make([]PackageChoice, count)
		for i := range choices {
			choices[i].ID = fmt.Sprintf("host%d:default", i)
		}
		id, err := ChoosePackage(choices, false)
		if count == 1 {
			if err != nil || id != choices[0].ID {
				t.Fatal(id, err)
			}
		} else if err == nil {
			t.Fatal("expected a package guidance error")
		}
	}
}

func TestPackagePickerSelectionAndCancel(t *testing.T) {
	m := packagePicker{choices: []PackageChoice{{ID: "a:default"}, {ID: "b:default"}}, width: 60, height: 19}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	next, cmd := next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || next.(packagePicker).selected != "b:default" {
		t.Fatal(next)
	}
	canceled, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil || canceled.(packagePicker).selected != "" {
		t.Fatal(canceled)
	}
	for _, line := range strings.Split(m.View(), "\n") {
		if ansi.StringWidth(line) > 60 {
			t.Fatal(line)
		}
	}
}

func pickerFixture(count int) packagePicker {
	var choices []PackageChoice
	for i := 0; i < count; i++ {
		choices = append(choices, PackageChoice{
			ID:       fmt.Sprintf("be%02d.prod:default", i),
			Group:    "backend",
			Endpoint: fmt.Sprintf("http://10.180.37.%d:9180/XFOuLgmw/", i+1),
			State:    "A",
		})
	}
	choices = append(choices, PackageChoice{ID: "evt01.prod:default", Group: "event", Endpoint: "http://10.180.36.239:9180/DlTsUlcg/", State: "D"})
	return packagePicker{choices: choices, width: 100, height: 20}
}

func typePicker(m tea.Model, keys ...string) tea.Model {
	for _, key := range keys {
		switch key {
		case "enter":
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		case "esc":
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		case "down":
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		case "backspace":
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
		default:
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		}
	}
	return m
}

// The filter must not swallow the picker's own keys, and a key that quits the
// picker outside the filter has to stay typeable inside it.
func TestPackagePickerFilterAndQuickSelect(t *testing.T) {
	m := typePicker(pickerFixture(3), "/", "e", "v", "t")
	if got := m.(packagePicker).filter; got != "evt" {
		t.Fatal(got)
	}
	if rows := m.(packagePicker).visible(); len(rows) != 1 || rows[0].ID != "evt01.prod:default" {
		t.Fatal(rows)
	}
	quitting := typePicker(pickerFixture(3), "/", "q")
	if quitting.(packagePicker).filter != "q" || quitting.(packagePicker).selected != "" {
		t.Fatal("filter input quit the picker")
	}
	applied, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // applies the filter
	if cmd != nil || applied.(packagePicker).filtering {
		t.Fatal("filter was not applied")
	}
	chosen, cmd := applied.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || chosen.(packagePicker).selected != "evt01.prod:default" {
		t.Fatal(chosen)
	}
	// A digit selects by the row number the table prints, within the filter.
	numbered, cmd := applied.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if cmd == nil || numbered.(packagePicker).selected != "evt01.prod:default" {
		t.Fatal(numbered)
	}
	if out, cmd := applied.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")}); cmd != nil || out.(packagePicker).selected != "" {
		t.Fatal("digit beyond the filtered list selected a package")
	}
	full := typePicker(pickerFixture(3), "3")
	if full.(packagePicker).selected != "be02.prod:default" {
		t.Fatal(full)
	}
	// Esc leaves the filter without selecting, restoring every candidate.
	cleared := typePicker(m, "esc")
	if len(cleared.(packagePicker).visible()) != 4 || cleared.(packagePicker).selected != "" {
		t.Fatal(cleared)
	}
	empty := typePicker(pickerFixture(3), "/", "z", "z", "enter")
	if out, cmd := empty.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil || out.(packagePicker).selected != "" {
		t.Fatal("selected from an empty list")
	}
	if !strings.Contains(ansi.Strip(empty.View()), "조건에 맞는 패키지가 없습니다") {
		t.Fatal(empty.View())
	}
}

func TestPackagePickerSheetFitsEverySize(t *testing.T) {
	m := pickerFixture(24)
	for _, size := range [][2]int{{60, 12}, {60, 19}, {80, 24}, {88, 20}, {100, 18}, {132, 42}} {
		m.width, m.height = size[0], size[1]
		for _, cursor := range []int{0, 12, 24} {
			m.cursor = cursor
			for _, filtering := range []bool{false, true} {
				m.filtering, m.filter = filtering, "10.180.37.2"
				view := m.View()
				if rows := strings.Split(view, "\n"); len(rows) > m.height {
					t.Fatalf("height overflow %dx%d: %d rows", m.width, m.height, len(rows))
				}
				for _, row := range strings.Split(view, "\n") {
					if ansi.StringWidth(row) > m.width {
						t.Fatalf("width overflow %dx%d: %q", m.width, m.height, row)
					}
				}
			}
		}
	}
	m.width, m.height, m.filter, m.filtering, m.cursor = 100, 20, "", false, 0
	view := ansi.Strip(m.View())
	// The endpoint's random path token is noise; the host and port are not.
	if !strings.Contains(view, "10.180.37.1:9180") || strings.Contains(view, "XFOuLgmw") {
		t.Fatal(view)
	}
	if !strings.Contains(view, "ALIVE") || !strings.Contains(view, "PACKAGE") || !strings.Contains(view, "HOST") {
		t.Fatal(view)
	}
	m.width = 80
	if narrow := ansi.Strip(m.View()); strings.Contains(narrow, "HOST") || !strings.Contains(narrow, "GROUP") {
		t.Fatal(narrow)
	}
}

func TestPackageStateLabel(t *testing.T) {
	for state, want := range map[string]string{"A": "ALIVE", "D": "DEAD", "": "UNKNOWN", "alive": "ALIVE", "UNREACHABLE": "UNREACHABLE"} {
		if got := PackageStateLabel(state); got != want {
			t.Fatalf("%q: %q, want %q", state, got, want)
		}
	}
	if got := PackageHost("not a url"); got != "not a url" {
		t.Fatal(got)
	}
}

func TestLegacyEndpointPackageSelection(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/explicit=%t", count, explicit), func(t *testing.T) {
				lists, resolves := 0, 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/auth/login/v1":
						fmt.Fprint(w, `{"token":"test"}`)
					case "/pack/v1":
						lists++
						entries := make([]map[string]string, count)
						for i := range entries {
							entries[i] = map[string]string{"host": fmt.Sprintf("h%d", i), "name": "default"}
						}
						json.NewEncoder(w).Encode(map[string]any{"summary": map[string]any{"deployment": []any{map[string]any{"group_name": "backend", "deploy": entries}}}})
					case "/juno/retrieve/v1":
						resolves++
						var q map[string]string
						if err := json.NewDecoder(r.Body).Decode(&q); err != nil && err != io.EOF {
							t.Error(err)
						}
						if q["package"] == "" {
							fmt.Fprint(w, `{"system":{"code":500,"message":"there are many host(package) exist. you have to specify host:package with option -p"}}`)
							return
						}
						want := "h0:default"
						if explicit {
							want = "chosen:default"
						}
						if q["package"] != want {
							t.Errorf("resolved %q, want %q", q["package"], want)
						}
						fmt.Fprint(w, `{"system":{"code":200},"endpoint":"http://juno"}`)
					default:
						t.Error(r.URL.Path)
						w.WriteHeader(404)
					}
				}))
				defer server.Close()
				flags := FatimaCmdFlags{JupiterUri: server.URL, Plain: true}
				if explicit {
					flags.UserPackage = "chosen:default"
				}
				err := GetJunoEndpoint(&flags)
				if explicit || count == 1 {
					wantResolves := 2
					if explicit {
						wantResolves = 1
					}
					if err != nil || resolves != wantResolves {
						t.Fatal(err, resolves)
					}
				} else if err == nil || resolves != 1 {
					t.Fatal("resolved without selection", err, resolves)
				}
				if explicit && lists != 0 {
					t.Fatal("explicit package was ignored")
				}
			})
		}
	}
}

func TestPreferClientPackageIPs(t *testing.T) {
	choices := []PackageChoice{
		{ID: "local:a", Endpoint: "http://10.1.2.3:9180/a"},
		{ID: "local:b", Endpoint: "http://10.1.2.3:9181/b"},
		{ID: "remote:default", Endpoint: "http://10.1.2.4:9180"},
	}
	local := preferPackageIPs(choices, []net.IP{net.ParseIP("10.1.2.3")})
	if len(local) != 2 || local[0].ID != "local:a" || local[1].ID != "local:b" {
		t.Fatal(local)
	}
	if len(preferPackageIPs(choices, []net.IP{net.ParseIP("10.1.2.99")})) != 3 {
		t.Fatal("remote candidates missing")
	}
	if len(packagesAtEndpointIP(choices, "http://10.1.2.3:9180/a")) != 2 {
		t.Fatal("HTTP first-match ambiguity not detected")
	}
	ipv6 := []PackageChoice{{ID: "v6", Endpoint: "http://[2001:db8::1]:9180"}}
	if len(preferPackageIPs(ipv6, []net.IP{net.ParseIP("2001:db8::1")})) != 1 {
		t.Fatal("IPv6")
	}
}

func TestLegacyIPResolution(t *testing.T) {
	for _, sameIP := range []bool{false, true} {
		t.Run(fmt.Sprint(sameIP), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/auth/login/v1":
					fmt.Fprint(w, `{"token":"test"}`)
				case "/pack/v1":
					other := "http://10.1.2.4:9180"
					if sameIP {
						other = "http://10.1.2.3:9181"
					}
					fmt.Fprintf(w, `{"summary":{"deployment":[{"deploy":[{"host":"local","name":"a","endpoint":"http://10.1.2.3:9180"},{"host":"other","name":"b","endpoint":%q}]}]}}`, other)
				case "/juno/retrieve/v1":
					calls++
					if calls > 1 {
						var q map[string]string
						json.NewDecoder(r.Body).Decode(&q)
						if q["package"] != "local:a" {
							t.Error(q)
						}
					}
					fmt.Fprint(w, `{"system":{"code":200},"endpoint":"http://10.1.2.3:9180"}`)
				}
			}))
			defer server.Close()
			flags := FatimaCmdFlags{JupiterUri: server.URL, Plain: true}
			err := GetJunoEndpoint(&flags)
			if sameIP {
				if err == nil || calls != 1 || flags.Endpoint != "" {
					t.Fatal("multiple local packages silently selected", err, calls)
				}
			} else if err != nil || calls != 2 || flags.UserPackage != "local:a" {
				t.Fatal(err, calls, flags.UserPackage)
			}
		})
	}
}
