// Package syncer implements types to sync charts between repositories
package containersyncer

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
	src client.ContainersWrapper
	dst client.ContainersUnwrapper
}

// A Syncer can be used to sync a source and target chart repos.
type Syncer struct {
	source *apiv1.Source
	target *apiv1.Target

	cli *Clients

	dryRun            bool
	insecure          bool
	usePlainHTTP      bool
	latestVersionOnly bool

	// list of container platforms to sync
	containerPlatforms []string

	// skip syncing artifacts
	skipArtifacts bool

	// copy container images byte-for-byte, preserving their original digest
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
	sourceIsLocal := source.GetRepo() != nil && source.GetRepo().GetKind() == apiv1.Kind_LOCAL
	if source.GetContainers() == nil && !sourceIsLocal {
		return nil, errors.New("missing source.containers config")
	}
	srcCli, err := cs.NewContainerClient(source, types.WithCache(s.workdir), types.WithInsecure(s.insecure), types.WithUsePlainHTTP(s.usePlainHTTP), types.WithTimeout(s.timeout))
	if err != nil {
		return nil, errors.Trace(err)
	}
	s.cli.src = srcCli

	targetIsLocal := target.GetRepo() != nil && target.GetRepo().GetKind() == apiv1.Kind_LOCAL
	if target.GetContainers() == nil && !targetIsLocal {
		return nil, errors.New("missing target.containers config")
	}
	dstCli, err := ct.NewContainerClient(target, types.WithCache(s.workdir), types.WithInsecure(s.insecure), types.WithUsePlainHTTP(s.usePlainHTTP), types.WithTimeout(s.timeout))
	if err != nil {
		return nil, errors.Trace(err)
	}
	s.cli.dst = dstCli

	return s, nil
}

// WithContainerPlatforms configures the syncer to sync chart containers for only
// the specified list of platforms. Leaving a blank list syncs all.
func WithContainerPlatforms(platforms []string) Option {
	return func(s *Syncer) {
		s.containerPlatforms = platforms
	}
}

// WithPreserveDigest configures the syncer to copy container images byte-for-byte,
// preserving their original digest
func WithPreserveDigest(preserve bool) Option {
	return func(s *Syncer) {
		s.preserveDigest = preserve
	}
}
