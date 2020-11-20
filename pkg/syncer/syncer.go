package syncer

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/chart"
	"github.com/bitnami-labs/charts-syncer/pkg/client/core"
	"github.com/bitnami-labs/charts-syncer/pkg/helmcli"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
)

// A Syncer can be used to sync a source and target chart repos.
type Syncer struct {
	source *api.SourceRepo
	target *api.TargetRepo

	dryRun        bool
	autoDiscovery bool
	fromDate      string
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

// Sync synchronizes source and target chart repos
func (s *Syncer) Sync(charts ...string) error {
	// TODO(jdrios): The code below is too optimistic and it does not take into
	// account the chart repo backend. For example, index.yaml is specific for
	// helm-like implementations. OCI does not implement an index.yaml file.
	// Adapting this logic will require to refactor the `pkg/chart` package, and
	// probably merging it with this one.

	sourceIndex, err := utils.LoadIndexFromRepo(s.source.GetRepo())
	if err != nil {
		return errors.Trace(fmt.Errorf("error loading index.yaml: %w", err))
	}

	if len(charts) == 0 {
		if !s.autoDiscovery {
			return fmt.Errorf("unable to discover charts to sync")
		}

		// TODO(jdrios): This is only valid for backends supporting an index.yaml file.
		for n := range sourceIndex.Entries {
			charts = append(charts, n)
		}
	}

	// Create basic layout for date and parse flag to time type
	dateThreshold, err := utils.GetDateThreshold(s.fromDate)
	if err != nil {
		return errors.Trace(err)
	}
	targetIndex, err := utils.LoadIndexFromRepo(s.target.GetRepo())
	if err != nil {
		return errors.Trace(fmt.Errorf("error loading index.yaml: %w", err))
	}

	// Add target repo to helm CLI
	helmcli.AddRepoToHelm(s.target.GetRepo().GetUrl(), s.target.GetRepo().GetAuth())

	// Create client for target repo
	tc, err := core.NewClient(s.target.GetRepo())
	if err != nil {
		return errors.Trace(fmt.Errorf("could not create a client for the source repo: %w", err))
	}

	// Iterate over charts in source index
	var errs error
	for _, name := range charts {
		// Iterate over charts versions
		for i := range sourceIndex.Entries[name] {
			version := sourceIndex.Entries[name][i].Metadata.Version
			publishingTime := sourceIndex.Entries[name][i].Created
			if publishingTime.Before(dateThreshold) {
				continue
			}
			if chartExists, _ := tc.ChartExists(name, version); chartExists {
				continue
			}
			if s.dryRun {
				klog.Infof("dry-run: Chart %s-%s pending to be synced", name, version)
				continue
			}
			klog.Infof("Syncing %s-%s", name, version)
			if err := chart.Sync(name, version, s.source.GetRepo(), s.target, sourceIndex, targetIndex, true); err != nil {
				errs = multierror.Append(errs, errors.Trace(err))
			}
		}
	}

	return errs
}
