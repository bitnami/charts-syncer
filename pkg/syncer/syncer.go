package syncer

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/chart"
	"github.com/bitnami-labs/charts-syncer/pkg/helmcli"
	"github.com/bitnami-labs/charts-syncer/pkg/repo"
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
func (s *Syncer) Sync() error {
	// TODO(jdrios): Remove assertion once we support syncing a list of charts
	if !s.autoDiscovery {
		return errors.Trace(fmt.Errorf("unable to sync repos without charts auto-discovery"))
	}

	// TODO(jdrios): The code below is too optimistic and it does not take into
	// account the chart repo backend. For example, index.yaml is specific for
	// helm-like implementations. OCI does not implement an index.yaml file.
	// Adapting this logic will require to refactor the `pkg/chart` package, and
	// probably merging it with this one.

	// Create basic layout for date and parse flag to time type
	dateThreshold, err := utils.GetDateThreshold(s.fromDate)
	if err != nil {
		return errors.Trace(err)
	}
	sourceIndex, err := utils.LoadIndexFromRepo(s.source.Repo)
	if err != nil {
		return errors.Trace(fmt.Errorf("error loading index.yaml: %w", err))
	}
	targetIndex, err := utils.LoadIndexFromRepo(s.target.Repo)
	if err != nil {
		return errors.Trace(fmt.Errorf("error loading index.yaml: %w", err))
	}

	// Add target repo to helm CLI
	helmcli.AddRepoToHelm(s.target.Repo.Url, s.target.Repo.Auth)

	// Create client for target repo
	tc, err := repo.NewClient(s.target.Repo)
	if err != nil {
		return errors.Trace(fmt.Errorf("could not create a client for the source repo: %w", err))
	}

	// Iterate over charts in source index
	var errs error

	for chartName := range sourceIndex.Entries {
		// Iterate over charts versions
		for i := range sourceIndex.Entries[chartName] {
			chartVersion := sourceIndex.Entries[chartName][i].Metadata.Version
			publishingTime := sourceIndex.Entries[chartName][i].Created
			if publishingTime.Before(dateThreshold) {
				continue
			}
			if chartExists, _ := tc.ChartExists(chartName, chartVersion, targetIndex); chartExists {
				continue
			}
			if s.dryRun {
				klog.Infof("dry-run: Chart %s-%s pending to be synced", chartName, chartVersion)
				continue
			}
			klog.Infof("Syncing %s-%s", chartName, chartVersion)
			if err := chart.Sync(chartName, chartVersion, s.source.Repo, s.target, sourceIndex, targetIndex, true); err != nil {
				errs = multierror.Append(errs, errors.Trace(err))
			}
		}
	}

	return errs
}
