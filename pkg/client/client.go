package client

import (
	"github.com/bitnami/charts-syncer/pkg/client/config"
	"github.com/bitnami/charts-syncer/pkg/client/types"
	"helm.sh/helm/v3/pkg/chart"
)

// This package defines the interfaces that clients needs to satisfy in order to work with chart repositories or
// intermediate bundles directories.

// ChartsReader defines the methods that a ReadOnly chart or bundle client should implement.
type ChartsReader interface {
	Fetch(name string, version string) (string, error)
	List() ([]string, error)
	ListChartVersions(name string) ([]string, error)
	Has(name string, version string) (bool, error)
	GetChartDetails(name string, version string) (*types.ChartDetails, error)
	// Reload reloads or refresh the client-side data, in case it needs it
	Reload() error
}

// ChartsWriter defines the methods that a WriteOnly chart or bundle client should implement.
type ChartsWriter interface {
	GetUploadURL() string
	Upload(filepath string, metadata *chart.Metadata) error
}

// ChartsReaderWriter defines the methods that a chart or bundle client should implement
type ChartsReaderWriter interface {
	ChartsReader
	ChartsWriter
}

// ChartsUnwrapper defines the methods required to unwrap a chart
type ChartsUnwrapper interface {
	ChartsReader
	Unwrap(filepath string, metadata *chart.Metadata, opts ...config.Option) error
}

// ChartsWrapper defines the methods required to wrap a chart
type ChartsWrapper interface {
	ChartsReader
	Wrap(source string, destination string, opts ...config.Option) (string, error)
}
