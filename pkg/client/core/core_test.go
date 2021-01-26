package core

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
)

func TestNewClientV2(t *testing.T) {
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
			c, err := NewClientV2(test.repo)
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
