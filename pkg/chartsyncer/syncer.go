// Package chartsyncer implements types to sync charts between repositories
package chartsyncer

import (
	"os"
	"time"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/pkg/client"
	cs "github.com/bitnami/charts-syncer/pkg/client/source"
	ct "github.com/bitnami/charts-syncer/pkg/client/target"

	"github.com/bitnami/charts-syncer/pkg/client/types"
	"github.com/juju/errors"
	log "github.com/vmware-labs/distribution-tooling-for-helm/pkg/dtlog"
	"github.com/vmware-labs/distribution-tooling-for-helm/pkg/dtlog/silent"

	"k8s.io/klog"
)

// Clients holds the source and target chart repo clients
type Clients struct {
	src client.ChartsWrapper
	dst client.ChartsUnwrapper
}

// A Syncer can be used to sync a source and target chart repos.
type Syncer struct {
	source *apiv1.Source
	target *apiv1.Target

	cli *Clients

	dryRun            bool
	autoDiscovery     bool
	fromDate          string
	insecure          bool
	usePlainHTTP      bool
	latestVersionOnly bool
	// list of charts to skip
	skipCharts []string

	// list of container platforms to sync
	containerPlatforms []string
	// TODO(jdrios): Cache index in local filesystem to speed
	// up re-runs
	index ChartIndex

	// skip syncing artifacts
	skipArtifacts bool

	// skip syncing images
	skipImages bool

	// copy charts and container images byte-for-byte, preserving their original digest
	preserveDigest bool

	// Storage directory for required artifacts
	workdir string

	// Timeout for operations
	timeout time.Duration

	logger log.SectionLogger
}

// Option is an option value used to create a new syncer instance.
type Option func(*Syncer)

// WithDryRun configures the syncer to run in dry-run mode.
func WithDryRun(enable bool) Option {
	return func(s *Syncer) {
		s.dryRun = enable
	}
}

// WithLogger configures the syncer to use a specific logger.
func WithLogger(l log.SectionLogger) Option {
	return func(s *Syncer) {
		s.logger = l
	}
}

// WithAutoDiscovery configures the syncer to discover all the charts to sync
// from the source chart repos.
func WithAutoDiscovery(enable bool) Option {
	return func(s *Syncer) {
		s.autoDiscovery = enable
	}
}

// WithFromDate configures the syncer to synchronize the charts from a specific
// time using YYYY-MM-DD format.
func WithFromDate(date string) Option {
	return func(s *Syncer) {
		s.fromDate = date
	}
}

// WithUsePlainHTTP configures the syncer to use plain HTTP
func WithUsePlainHTTP(enable bool) Option {
	return func(s *Syncer) {
		s.usePlainHTTP = enable
	}
}

// WithSkipArtifacts configures the syncer to skip syncing artifacts
func WithSkipArtifacts(skip bool) Option {
	return func(s *Syncer) {
		s.skipArtifacts = skip
	}
}

// WithSkipImages configures the syncer to skip syncing images
func WithSkipImages(skip bool) Option {
	return func(s *Syncer) {
		s.skipImages = skip
	}
}

// WithPreserveDigest configures the syncer to copy charts and container images
// byte-for-byte, preserving their original digest
func WithPreserveDigest(preserve bool) Option {
	return func(s *Syncer) {
		s.preserveDigest = preserve
	}
}

// WithWorkdir configures the syncer to store artifacts in a specific directory.
func WithWorkdir(dir string) Option {
	return func(s *Syncer) {
		s.workdir = dir
	}
}

// WithTimeout configures the syncer to use a specific timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(s *Syncer) {
		s.timeout = timeout
	}
}

// WithInsecure configures the syncer to allow insecure SSL connections
func WithInsecure(enable bool) Option {
	return func(s *Syncer) {
		s.insecure = enable
	}
}

// WithLatestVersionOnly configures the syncer to sync only the latest version
func WithLatestVersionOnly(latestVersionOnly bool) Option {
	return func(s *Syncer) {
		s.latestVersionOnly = latestVersionOnly
	}
}

// New creates a new syncer using Client
func New(source *apiv1.Source, target *apiv1.Target, opts ...Option) (*Syncer, error) {
	s := &Syncer{
		source: source,
		target: target,
		logger: silent.NewSectionLogger(),
	}

	for _, o := range opts {
		o(s)
	}

	// If a workdir wasn't specified, let's use a directory relative to the
	// current directory
	if s.workdir == "" {
		s.workdir = "./workdir"
	}
	klog.V(3).Infof("Using workdir: %q", s.workdir)

	if err := os.MkdirAll(s.workdir, 0755); err != nil {
		return nil, errors.Trace(err)
	}

	s.cli = &Clients{}
	if source.GetRepo() != nil {
		srcCli, err := cs.NewClient(source, types.WithCache(s.workdir), types.WithInsecure(s.insecure), types.WithUsePlainHTTP(s.usePlainHTTP), types.WithTimeout(s.timeout))
		if err != nil {
			return nil, errors.Trace(err)
		}
		s.cli.src = srcCli
	} else {
		return nil, errors.New("no source info defined in config file")
	}

	if target.GetRepo() != nil {
		dstCli, err := ct.NewClient(target, types.WithCache(s.workdir), types.WithInsecure(s.insecure), types.WithUsePlainHTTP(s.usePlainHTTP), types.WithTimeout(s.timeout))
		if err != nil {
			return nil, errors.Trace(err)
		}
		s.cli.dst = dstCli
	} else {
		return nil, errors.New("no target info defined in config file")
	}

	return s, nil
}

// WithSkipCharts configures the syncer to skip an explicit list of chart names
// from the source chart repos.
func WithSkipCharts(charts []string) Option {
	return func(s *Syncer) {
		s.skipCharts = charts
	}
}

// WithContainerPlatforms configures the syncer to sync chart containers for only
// the specified list of platforms. Leaving a blank list syncs all.
func WithContainerPlatforms(platforms []string) Option {
	return func(s *Syncer) {
		s.containerPlatforms = platforms
	}
}
