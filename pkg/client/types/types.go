// Package types defines common types used in repository clients
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
	cacheDir  string
	insecure  bool
	plainHTTP bool
}

// Option is an option value used to create a new syncer instance.
type Option func(*ClientOpts)

// WithCache configures a cache directory
func WithCache(dir string) Option {
	return func(s *ClientOpts) {
		s.cacheDir = dir
	}
}

// WithInsecure enables insecure SSL connections
func WithInsecure(enable bool) Option {
	return func(s *ClientOpts) {
		s.insecure = enable
	}
}

// GetCache returns the cache directory
func (o *ClientOpts) GetCache() string {
	if o == nil {
		return ""
	}
	return o.cacheDir
}

// GetInsecure returns if insecure connections are allowed
func (o *ClientOpts) GetInsecure() bool {
	if o == nil {
		return false
	}
	return o.insecure
}

// WithUsePlainHTTP configures the client to use plain HTTP
func WithUsePlainHTTP(enable bool) Option {
	return func(s *ClientOpts) {
		s.plainHTTP = enable
	}
}

// GetUsePlainHTTP returns if the client is configured to use plain HTTP
func (o *ClientOpts) GetUsePlainHTTP() bool {
	if o == nil {
		return false
	}
	return o.plainHTTP
}
