package syncer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/chart"
	"github.com/bitnami-labs/charts-syncer/internal/helmcli"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/core"
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
	switch s.target.GetRepo().GetKind() {
	case api.Kind_HELM, api.Kind_CHARTMUSEUM, api.Kind_HARBOR:
		repoName := fmt.Sprintf("charts-syncer-%s", s.target.GetRepoName())
		cleanup, err := helmcli.AddRepoToHelm(repoName, s.target.GetRepo().GetUrl(), s.target.GetRepo().GetAuth())
		if err != nil {
			return errors.Trace(err)
		}
		defer cleanup()
	case api.Kind_OCI:
		cleanup, err := helmcli.OciLogin(s.target.GetRepo().GetUrl(), s.target.GetRepo().GetAuth())
		if err != nil {
			return errors.Trace(err)
		}
		defer cleanup()
	default:
		return errors.Errorf("unsupported repo kind %q", s.target.GetRepo().GetKind())
	}

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

		workDir, err := ioutil.TempDir("", "charts-syncer")
		if err != nil {
			return errors.Trace(err)
		}
		defer os.RemoveAll(workDir)
		chartPath, err := chart.ChangeReferences(workDir, ch.TgzPath, ch.Name, ch.Version, s.source, s.target)
		if err != nil {
			klog.Errorf("unable to process %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}

		// Update deps
		if hasDeps {
			if err := chart.ChangeDependenciesFile(chartPath, ch.Name, s.source, s.target); err != nil {
				return errors.Trace(err)
			}
			switch s.target.GetRepo().GetKind() {
			case api.Kind_OCI:
				if err := s.buildDependenciesFromOci(chartPath, ch.Name); err != nil {
					return errors.Trace(err)
				}
			default:
				if err := helmcli.UpdateDependencies(chartPath); err != nil {
					return errors.Trace(err)
				}
			}
		}

		// Package chart again
		//
		// TODO(jdrios): This relies on the helm client to package the repo. It
		// does not take into account that the target repo could be out of sync yet
		// (for example, if we uploaded a dependency of the chart being packaged a
		// few seconds ago).
		tgz, err := helmcli.Package(chartPath, ch.Name, ch.Version, outDir)
		if err != nil {
			return errors.Trace(err)
		}

		if s.dryRun {
			klog.Infof("dry-run: Uploading %q chart", id)
			continue
		}

		klog.V(3).Infof("Uploading %q chart...", id)
		if err := s.cli.dst.Upload(tgz, ch.Name, ch.Version); err != nil {
			klog.Errorf("unable to upload %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}
	}

	return errors.Trace(errs)
}

func (s *Syncer) buildDependenciesFromOci(chartPath, name string) error {
	// Build deps manually for OCI as helm does not support it yet
	if err := os.RemoveAll(path.Join(chartPath, "charts")); err != nil {
		return errors.Trace(err)
	}
	// Re-create empty charts folder
	err := os.Mkdir(path.Join(chartPath, "charts"), 0755)
	if err != nil {
		return errors.Trace(err)
	}
	lock, err := chart.GetChartLock(chartPath, name)
	if err != nil {
		return errors.Trace(err)
	}

	// Dependencies found
	var errs error
	if lock != nil {
		for _, dep := range lock.Dependencies {
			depID := fmt.Sprintf("%s-%s", dep.Name, dep.Version)

			depTgz, err := s.cli.dst.Fetch(dep.Name, dep.Version)
			if err != nil {
				klog.Errorf("unable to update %q chart dependency: %+v", depID, err)
				errs = multierror.Append(errs, errors.Annotatef(err, "updating %q chart dependency", depID))
				continue
			}
			if err := utils.CopyFile(depTgz, path.Join(chartPath, "charts", fmt.Sprintf("%s.tgz", depID))); err != nil {
				klog.Errorf("unable to update %q chart dependency: %+v", depID, err)
				errs = multierror.Append(errs, errors.Annotatef(err, "updating %q chart dependency", depID))
				continue
			}
		}
	}

	return errors.Trace(err)
}
