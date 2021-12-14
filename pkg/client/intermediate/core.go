package intermediate

import (
	"github.com/bitnami-labs/charts-syncer/pkg/client"
)

// NewIntermediateClient returns a ReadWriter object
var NewIntermediateClient = func(intermediateBundlesPath string) (client.ReadWriter, error) {
	return New(intermediateBundlesPath)
}
