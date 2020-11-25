package syncer

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/pkg/chart"
	"github.com/bitnami-labs/charts-syncer/pkg/client/core"
	"github.com/bitnami-labs/charts-syncer/pkg/helmcli"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
)

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

	publishingThreshold := utils.UnixEpoch
	if s.fromDate != "" {
		// Create basic layout for date and parse flag to time type
		t, err := utils.GetDateThreshold(s.fromDate)
		if err != nil {
			return errors.Trace(err)
		}
		publishingThreshold = t
	}

	targetIndex, err := utils.LoadIndexFromRepo(s.target.GetRepo())
	if err != nil {
		return errors.Trace(fmt.Errorf("error loading index.yaml: %w", err))
	}

	// Add target repo to helm CLI
	helmcli.AddRepoToHelm(s.target.GetRepoName(), s.target.GetRepo().GetUrl(), s.target.GetRepo().GetAuth())

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

			if sourceIndex.Entries[name][i].Created.Before(publishingThreshold) {
				continue
			}

			if chartExists, _ := tc.ChartExists(name, version, targetIndex); chartExists {
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

	return errors.Trace(errs)
}

// SyncPendingCharts syncs the charts not found in the target
//
// It uses topological sort to sync dependencies first.
func (s *Syncer) SyncPendingCharts(names ...string) error {
	var errs error

	// There might be problems loading all the charts due to missing dependencies,
	// invalid/wrong charts in the repository, etc. Therefore, let's warn about
	// them instead of blocking the whole sync.
	if err := s.loadCharts(names...); err != nil {
		klog.Warningf("There were some problems loading the information of the requested charts: %v", err)
		errs = multierror.Append(errs, errors.Trace(err))
	}
	// NOTE: We are not checking `errs` in purpose. See the comment above.

	charts, err := s.topologicalSortCharts()
	if err != nil {
		return errors.Trace(err)
	}

	if len(charts) > 1 {
		klog.Infof("There are %d charts out of sync!", len(charts))
	} else if len(charts) == 1 {
		klog.Infof("There is %d chart out of sync!", len(charts))
	} else {
		klog.Info("There are no charts out of sync!")
		return nil
	}

	// Add target repo to helm CLI
	//
	// This is required to use helm CLI for certain operation such us
	// `helm dependency update`.
	//
	// TODO(jdrios): Check if we can remove the helm CLI requirement.
	repoName := fmt.Sprintf("charts-syncer-%s", s.target.GetRepoName())
	cleanup, err := helmcli.AddRepoToHelm(repoName, s.target.GetRepo().GetUrl(), s.target.GetRepo().GetAuth())
	if err != nil {
		return errors.Trace(err)
	}
	defer cleanup()

	for _, ch := range charts {
		id := fmt.Sprintf("%s-%s", ch.Name, ch.Version)
		klog.Infof("Syncing %q chart...", id)

		klog.V(3).Infof("Processing %q chart...", id)
		outDir, err := ioutil.TempDir("", "charts-syncer")
		if err != nil {
			return errors.Trace(err)
		}
		defer os.RemoveAll(outDir)

		hasDeps := len(ch.Dependencies) > 0
		tgz, err := chart.ChangeReferences(outDir, ch.TgzPath, ch.Name, ch.Version, s.source, s.target, hasDeps)
		if err != nil {
			klog.Errorf("unable to process %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}

		if s.dryRun {
			klog.Infof("dry-run: Uploading %q chart", id)
			continue
		}

		klog.V(3).Infof("Uploading %q chart...", id)
		if err := s.cli.dst.Upload(tgz); err != nil {
			klog.Errorf("unable to upload %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}
	}

	return errors.Trace(errs)
}
