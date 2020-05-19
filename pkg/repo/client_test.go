package repo

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/bitnami-labs/chart-repository-syncer/api"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		repo     *api.Repo
		typeText string
		errText  string
	}{
		{
			&api.Repo{
				Kind: "HELM",
			},
			"*repo.ClassicHelmClient",
			"",
		},
		{
			&api.Repo{
				Kind: "CHARTMUSEUM",
			},
			"*repo.ChartMuseumClient",
			"",
		},
		{
			&api.Repo{
				Kind: "UNKNOWN",
			},
			"<nil>",
			"unsupported repo kind \"UNKNOWN\"",
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
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
