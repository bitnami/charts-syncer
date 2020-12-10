package types

import (
	"time"
)

// ChartDetails contains details of a chart
type ChartDetails struct {
	PublishedAt time.Time
	Digest      string
}

// ClientOpts allows to configure a client
type ClientOpts struct {
	cacheDir string
}

// Option is an option value used to create a new syncer instance.
type Option func(*ClientOpts)

// WithCache configures a cache directory
func WithCache(dir string) Option {
	return func(s *ClientOpts) {
		s.cacheDir = dir
	}
}

// GetCache returns the cache directory
func (o *ClientOpts) GetCache() string {
	if o == nil {
		return ""
	}
	return o.cacheDir
}
