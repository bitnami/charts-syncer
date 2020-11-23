package core

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		repo     *api.Repo
		typeText string
		errText  string
	}{
		{
			&api.Repo{
				Kind: api.Kind_HELM,
			},
			"*core.ClassicHelmClient",
			"",
		},
		{
			&api.Repo{
				Kind: api.Kind_CHARTMUSEUM,
			},
			"*core.ChartMuseumClient",
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
