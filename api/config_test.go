package api

import (
	"testing"
)

var (
	config = &Config{
		Source: &SourceRepo{
			Repo: &Repo{
				Url:  "ht//:fake.source.com",
				Kind: Kind_CHARTMUSEUM,
				Auth: &Auth{
					Username: "user",
					Password: "password",
				},
			},
		},
		Target: &TargetRepo{
			Repo: &Repo{
				Url:  "http://fake.target.com",
				Kind: Kind_CHARTMUSEUM,
				Auth: &Auth{
					Username: "user",
					Password: "password",
				},
			},
			ContainerRegistry:   "test.registry.io",
			ContainerRepository: "test/repo",
		},
	}
)

func TestValidate(t *testing.T) {
	expectedError := `"source.repo.url" should be a valid URL: parse "ht//:fake.source.com": invalid URI for request`
	err := config.Validate()
	if err != nil && err.Error() != expectedError {
		t.Errorf("Incorrect error, got: \n %s \n, want: \n %s \n", err.Error(), expectedError)
	}
}
