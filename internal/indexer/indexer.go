package indexer

import (
	"context"
	"github.com/bitnami-labs/charts-syncer/internal/indexer/api"
)

// Indexer is the interface that an indexer should implement
type Indexer interface {
	// Get retrieves the index
	Get(ctx context.Context) (*api.Index, error)
}
