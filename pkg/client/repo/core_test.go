package repo

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/repo/oci"
)

// Creates an HTTP server that knows how to reply to all OCI related requests
func prepareHttpServer(t *testing.T, ociRepo *api.Repo) {
	t.Helper()

	// Create HTTP server
	tester := oci.NewTester(t, ociRepo)
	ociRepo.Url = tester.GetURL() + "/someproject/charts"
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		repo     *api.Repo
		typeText string
		errText  string
	}{
		{
			&api.Repo{
				Kind: api.Kind_HELM,
				Url:  "https://charts.bitnami.com/bitnami",
			},
			"*helmclassic.Repo",
			"",
		},
		{
			&api.Repo{
				Kind: api.Kind_CHARTMUSEUM,
				// Not a real chartmuseum service. But I just want to reloadIndex() to work
				Url: "https://charts.bitnami.com/bitnami",
			},
			"*chartmuseum.Repo",
			"",
		},
		{
			&api.Repo{
				Kind: api.Kind_HARBOR,
				// Not a real chartmuseum service. But I just want to reloadIndex() to work
				Url: "https://charts.bitnami.com/bitnami",
			},
			"*harbor.Repo",
			"",
		},
		{
			&api.Repo{
				Kind: api.Kind_OCI,
				Url:  "http://localhost:9090/my-project",
				Auth: &api.Auth{
					Username: "user",
					Password: "password",
				},
			},
			"*oci.Repo",
			"",
		},
		{
			&api.Repo{
				Kind: api.Kind_LOCAL,
			},
			"*local.Repo",
			"",
		},
		{
			&api.Repo{
				Kind: api.Kind_UNKNOWN,
			},
			"<nil>",
			"unsupported repo kind \"UNKNOWN\"",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			// TODO (tpizarro): create simple http server to serve testdata index.yaml file so we don't have to use the index from the public
			// bitnami charts repo.

			// For OCI kind we need first to init an HTTP server to mock responses during client initialization
			if test.repo.Kind == api.Kind_OCI {
				prepareHttpServer(t, test.repo)
			}
			c, err := NewClient(test.repo)
			errText := ""
			if err != nil {
				errText = err.Error()
			}
			if got, want := errText, test.errText; got != want {
				t.Errorf("got=%q, want=%q", got, want)
			}
			if got, want := fmt.Sprintf("%T", c), test.typeText; got != want {
				t.Errorf("got=%q, want=%q", got, want)
			}
		})
	}
}
