// Package cache defines interfaces to support caching
package cache

import (
	"io"
	"os"
)

// Storer defines the methods that a Cache should implement to write.
type Storer interface {
	Store(r io.Reader, filename string) error
	Writer(filename string) (*os.File, error)
	Invalidate(filename string) error
}

// Fetcher defines the methods that a Cache should implement to read.
type Fetcher interface {
	Read(w io.Writer, filename string) error
	Has(filename string) bool
	Path(filename string) string
}

// Cacher defines all methods that a Cache should implement.
type Cacher interface {
	Storer
	Fetcher
}
