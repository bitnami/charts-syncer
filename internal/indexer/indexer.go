package indexer

import (
	"context"
	"fmt"

	"github.com/bitnami-labs/charts-syncer/internal/indexer/api"
)

// Indexer is the interface that an indexer should implement
type Indexer interface {
	// Get retrieves the index
	Get(ctx context.Context) (*api.Index, error)
}

// New returns a new Indexer based on the provided options
func New(opts interface{}) (Indexer, error) {
	switch v := opts.(type) {
	case *OciIndexerOpts:
		return NewOciIndexer(v)
	default:
		return nil, fmt.Errorf("%T is unsupported", opts)
	}
}
