package client

import (
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
	"helm.sh/helm/v3/pkg/chart"
)

// Reader defines the methods that a ReadOnly chart client should implement.
type Reader interface {
	Fetch(name string, version string) (string, error)
	List() ([]string, error)
	ListChartVersions(name string) ([]string, error)
	Has(name string, version string) (bool, error)
	GetChartDetails(name string, version string) (*types.ChartDetails, error)

	// Reload reloads or refresh the client-side data, in case it needs it
	Reload() error
}

// Writer defines the methods that a WriteOnly chart client should implement.
type Writer interface {
	Upload(filepath string, metadata *chart.Metadata) error
}

// ReadWriter defines the methods that a chart client should implement.
type ReadWriter interface {
	Reader
	Writer
}
