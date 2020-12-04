package cache

import (
	"io"
)

// Storer defines the methods that a Cache should implement to write.
type Storer interface {
	Store(r io.Reader, filename string) error
	Writer(filename string) *Writer
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
