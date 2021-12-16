package api_test

import (
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
)

func TestValidate(t *testing.T) {
	config := &api.Config{
		Source: &api.Source{
			Spec: &api.Source_Repo{
				Repo: &api.Repo{
					Url:  "ht//:fake.source.com",
					Kind: api.Kind_CHARTMUSEUM,
					Auth: &api.Auth{
						Username: "user",
						Password: "password",
					},
				},
			},
		},
		Target: &api.Target{
			Spec: &api.Target_Repo{
				Repo: &api.Repo{
					Url:  "http://fake.target.com",
					Kind: api.Kind_CHARTMUSEUM,
					Auth: &api.Auth{
						Username: "user",
						Password: "password",
					},
				},
			},
			ContainerRegistry:   "test.registry.io",
			ContainerRepository: "test/repo",
		},
	}

	if err := config.Validate(); err == nil {
		t.Errorf("expected error but got nothing")
	} else {
		expectedError := `"source.repo.url" should be a valid URL: parse "ht//:fake.source.com": invalid URI for request`
		if err.Error() != expectedError {
			t.Errorf("incorrect error, got: \n %s \n, want: \n %s \n", err.Error(), expectedError)
		}
	}
}
