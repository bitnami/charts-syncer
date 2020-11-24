package syncer

import (
	"github.com/juju/errors"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/core"
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

	// TODO(jdrios): Cache index (and tgz files) in local filesystem to speed
	// up re-runs
	index ChartIndex
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
	return s, nil
}
