package core

import (
	"fmt"

	"github.com/bitnami-labs/charts-syncer/api"
	helmRepo "helm.sh/helm/v3/pkg/repo"
)

// Client defines the methods that a chart client should implement.
type Client interface {
	Fetch(filepath string, name string, version string, sourceRepo *api.Repo, index *helmRepo.IndexFile) error
	Push(filepath string, targetRepo *api.Repo) error
	ChartExists(name string, version string, index *helmRepo.IndexFile) (bool, error)
}

// NewClient returns a client implementation for the given repo.
//
// The func is exposed as a var to allow tests to temporarily replace its
// implementation, e.g. to return a fake.
var NewClient = func(repo *api.Repo) (Client, error) {
	switch repo.Kind {
	case api.Kind_HELM:
		return NewClassicHelmClient(repo), nil
	case api.Kind_CHARTMUSEUM:
		return NewChartMuseumClient(repo), nil
	case api.Kind_HARBOR:
		return NewHarborClient(repo), nil
	default:
		return nil, fmt.Errorf("unsupported repo kind %q", repo.Kind)
	}
}
