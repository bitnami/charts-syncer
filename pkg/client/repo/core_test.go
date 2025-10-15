package repo

import (
	"fmt"
	"strconv"
	"testing"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/pkg/client/repo/oci"
)

// Creates an HTTP server that knows how to reply to all OCI related requests
func prepareHTTPServer(t *testing.T, ociRepo *apiv1.Repo) {
	t.Helper()

	// Create HTTP server
	tester := oci.NewTester(t)
	ociRepo.Url = tester.GetURL() + "/someproject/charts"
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		repo     *apiv1.Repo
		typeText string
		errText  string
	}{
		{
			&apiv1.Repo{
				Kind: apiv1.Kind_HELM,
				Url:  "https://charts.bitnami.com/bitnami",
			},
			"*helmclassic.Repo",
			"",
		},
		{
			&apiv1.Repo{
				Kind: apiv1.Kind_CHARTMUSEUM,
				// Not a real chartmuseum service. But I just want to reloadIndex() to work
				Url: "https://charts.bitnami.com/bitnami",
			},
			"*chartmuseum.Repo",
			"",
		},
		{
			&apiv1.Repo{
				Kind: apiv1.Kind_HARBOR,
				// Not a real chartmuseum service. But I just want to reloadIndex() to work
				Url: "https://charts.bitnami.com/bitnami",
			},
			"*harbor.Repo",
			"",
		},
		{
			&apiv1.Repo{
				Kind: apiv1.Kind_OCI,
				Url:  "http://localhost:9090/my-project",
				Auth: &apiv1.Auth{
					Username: "user",
					Password: "password",
				},
				DisableChartsIndex: true,
			},
			"*oci.Repo",
			"",
		},
		{
			&apiv1.Repo{
				Kind: apiv1.Kind_LOCAL,
			},
			"*local.Repo",
			"",
		},
		{
			&apiv1.Repo{
				Kind: apiv1.Kind_UNKNOWN,
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
			if test.repo.Kind == apiv1.Kind_OCI {
				prepareHTTPServer(t, test.repo)
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
