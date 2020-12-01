package syncer

import (
	"os"
	"path"

	"github.com/juju/errors"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/core"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
)

// Clients holds the source and target chart repo clients
type Clients struct {
	src core.ClientV2
	dst core.ClientV2
}

// A Syncer can be used to sync a source and target chart repos.
type Syncer struct {
	source *api.SourceRepo
	target *api.TargetRepo

	cli *Clients

	dryRun        bool
	autoDiscovery bool
	fromDate      string

	// TODO(jdrios): Cache index in local filesystem to speed
	// up re-runs
	index ChartIndex

	// Storage directory for required artifacts
	workdir    string
	srcWorkdir string
}

// Option is an option value used to create a new syncer instance.
type Option func(*Syncer)

// WithDryRun configures the syncer to run in dry-run mode.
func WithDryRun(enable bool) Option {
	return func(s *Syncer) {
		s.dryRun = enable
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

// WithWorkdir configures the syncer to store artifacts in a specific directory.
func WithWorkdir(dir string) Option {
	return func(s *Syncer) {
		s.workdir = dir
	}
}

// NewSyncer creates a new syncer.
func NewSyncer(source *api.SourceRepo, target *api.TargetRepo, opts ...Option) *Syncer {
	s := &Syncer{
		source: source,
		target: target,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// New creates a new syncer using ClientV2
func New(source *api.SourceRepo, target *api.TargetRepo, opts ...Option) (*Syncer, error) {
	srcCli, err := core.NewClientV2(source.GetRepo())
	if err != nil {
		return nil, errors.Trace(err)
	}

	dstCli, err := core.NewClientV2(target.GetRepo())
	if err != nil {
		return nil, errors.Trace(err)
	}

	cli := &Clients{
		src: srcCli,
		dst: dstCli,
	}

	s := &Syncer{
		source: source,
		target: target,

		cli: cli,
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

	// In order to store charts tgz files in an isolated folder for each chart
	// repo we are computing a workdir using the hash of the source repo URL
	// and the pre-configured workdir.
	s.srcWorkdir = path.Join(s.workdir, utils.EncodeSha1(source.GetRepo().GetUrl()))
	if err := os.MkdirAll(s.srcWorkdir, 0755); err != nil {
		return nil, errors.Trace(err)
	}

	return s, nil
}
