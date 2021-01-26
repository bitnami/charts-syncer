package core

import (
	"github.com/juju/errors"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/chartmuseum"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
)

// ClientV2Tester defines the methods that a fake tester should implement
type ClientV2Tester interface {
	Reader
	Writer
}

// NewClientV2Tester returns a fake repo for testing purposes
//
// The func is exposed as a var to allow tests to temporarily replace its
// implementation, e.g. to return a fake.
var NewClientV2Tester = func(repo *api.Repo, opts ...types.Option) (ClientV2Tester, error) {
	switch repo.Kind {
	case api.Kind_CHARTMUSEUM:
		return chartmuseum.NewTester(repo)
	default:
		return nil, errors.Errorf("unsupported repo kind %q", repo.Kind)
	}
}
