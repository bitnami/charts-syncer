package syncer

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/chartmuseum"
)

func TestNewClientV2(t *testing.T) {
	tests := []struct {
		sourceRepo *api.SourceRepo
		targetRepo *api.TargetRepo
		typeText   string
		errText    string
	}{
		{
			&api.SourceRepo{
				Repo: &api.Repo{
					Kind: api.Kind_CHARTMUSEUM,
					Auth: &api.Auth{
						Username: "user",
						Password: "password",
					},
				},
			},
			&api.TargetRepo{
				Repo: &api.Repo{
					Kind: api.Kind_CHARTMUSEUM,
					Auth: &api.Auth{
						Username: "user",
						Password: "password",
					},
				},
			},
			"*syncer.Syncer",
			"",
		},
		// {
		// 	&api.Repo{
		// 		Kind: api.Kind_CHARTMUSEUM,
		// 		// Not a real chartmuseum service. But I just want to reloadIndex() to work
		// 		Url: "https://charts.bitnami.com/bitnami",
		// 	},
		// 	"*chartmuseum.Repo",
		// 	"",
		// },
		// {
		// 	&api.Repo{
		// 		Kind: api.Kind_HARBOR,
		// 		// Not a real chartmuseum service. But I just want to reloadIndex() to work
		// 		Url: "https://charts.bitnami.com/bitnami",
		// 	},
		// 	"*harbor.Repo",
		// 	"",
		// },
		// {
		// 	&api.Repo{
		// 		Kind: api.Kind_OCI,
		// 	},
		// 	"*oci.Repo",
		// 	"",
		// },
		// {
		// 	&api.Repo{
		// 		Kind: api.Kind_LOCAL,
		// 	},
		// 	"*local.Repo",
		// 	"",
		// },
		// {
		// 	&api.Repo{
		// 		Kind: api.Kind_UNKNOWN,
		// 	},
		// 	"<nil>",
		// 	"unsupported repo kind \"UNKNOWN\"",
		// },
	}

	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			chartRepoTest := chartmuseum.ChartMuseumTests[0]
			url, cleanup := chartRepoTest.MakeServer(t, false, "")
			defer cleanup()

			// Update source repo url
			t.Logf("Index url is %q", url)
			tc.sourceRepo.Repo.Url = url
			tc.targetRepo.Repo.Url = url

			// Create new syncer
			s, err := New(tc.sourceRepo, tc.targetRepo)
			errText := ""
			if err != nil {
				errText = err.Error()
			}
			if got, want := errText, tc.errText; got != want {
				t.Errorf("got=%q, want=%q", got, want)
			}
			if got, want := fmt.Sprintf("%T", s), tc.typeText; got != want {
				t.Errorf("got=%q, want=%q", got, want)
			}
		})
	}
}
