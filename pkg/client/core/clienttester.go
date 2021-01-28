package core

import (
	"net/http"
	"testing"

	"github.com/juju/errors"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/chartmuseum"
)

// ClientTester defines the methods that a fake tester should implement
type ClientTester interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	GetChart(w http.ResponseWriter, r *http.Request, chart string)
	GetIndex(w http.ResponseWriter, r *http.Request, emptyIndex bool, indexFile string)
	GetChartPackage(w http.ResponseWriter, r *http.Request, chartPackageName string)
	PostChart(w http.ResponseWriter, r *http.Request)
	GetURL() string
}

// NewClientTester returns a fake repo for testing purposes
//
// The func is exposed as a var to allow tests to temporarily replace its
// implementation, e.g. to return a fake.
var NewClientTester = func(t *testing.T, repo *api.Repo, emptyIndex bool, indexFile string) (ClientTester, func(), error) {
	switch repo.Kind {
	case api.Kind_CHARTMUSEUM:
		return chartmuseum.NewTester(t, repo, emptyIndex, indexFile)
	default:
		return nil, nil, errors.Errorf("unsupported repo kind %q", repo.Kind)
	}
}
