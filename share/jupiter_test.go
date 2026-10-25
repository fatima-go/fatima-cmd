package share

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTokenProfile(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		profile string
		known   bool
	}{
		{"profile", `{"token":"t1","profile":"qa"}`, "qa", true},
		{"empty profile", `{"token":"t1","profile":""}`, "", true},
		{"legacy jupiter", `{"token":"t1"}`, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, v1LoginResourceUrl, r.URL.Path)
				_, _ = w.Write([]byte(c.body))
			}))
			defer server.Close()

			flags := FatimaCmdFlags{Username: "admin", Password: "secret", JupiterUri: server.URL}
			require.NoError(t, GetToken(&flags))
			assert.Equal(t, "t1", flags.Token)
			assert.Equal(t, c.profile, flags.Profile)
			assert.Equal(t, c.known, flags.ProfileKnown)
		})
	}
}
