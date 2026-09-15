package share

import (
	"encoding/json"
	"fmt"
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
						if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
							t.Error(err)
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
					if err != nil || resolves != 1 {
						t.Fatal(err, resolves)
					}
				} else if err == nil || resolves != 0 {
					t.Fatal("resolved without selection", err, resolves)
				}
				if explicit && lists != 0 {
					t.Fatal("explicit package was ignored")
				}
			})
		}
	}
}
