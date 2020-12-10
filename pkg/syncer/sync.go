package syncer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	helm "helm.sh/helm/v3/pkg/action"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/chart"
	"github.com/bitnami-labs/charts-syncer/internal/helmcli"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
)

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
	if s.target.GetRepo().GetKind() == api.Kind_OCI {
		cleanup, err := helmcli.OciLogin(s.target.GetRepo().GetUrl(), s.target.GetRepo().GetAuth())
		if err != nil {
			return errors.Trace(err)
		}
		defer cleanup()
	}

	for _, ch := range charts {
		id := fmt.Sprintf("%s-%s", ch.Name, ch.Version)
		klog.Infof("Syncing %q chart...", id)

		klog.V(3).Infof("Processing %q chart...", id)
		outDir, err := ioutil.TempDir("", "charts-syncer")
		if err != nil {
			klog.Errorf("unable to create output directory for %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}
		defer os.RemoveAll(outDir)

		hasDeps := len(ch.Dependencies) > 0

		workDir, err := ioutil.TempDir("", "charts-syncer")
		if err != nil {
			klog.Errorf("unable to create work directory for %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}
		defer os.RemoveAll(workDir)

		if err := utils.Untar(ch.TgzPath, workDir); err != nil {
			klog.Errorf("unable to uncompress %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Annotatef(err, "uncompressing %q chart", id))
			continue
		}

		chartPath := path.Join(workDir, ch.Name)
		if err := chart.ChangeReferences(chartPath, ch.Name, ch.Version, s.source, s.target); err != nil {
			klog.Errorf("unable to process %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}

		// Update deps
		if hasDeps {
			klog.V(3).Infof("Building %q dependencies", id)
			if err := chart.BuildDependencies(chartPath, s.cli.dst); err != nil {
				klog.Errorf("unable to build %q chart dependencies: %+v", id, err)
				errs = multierror.Append(errs, errors.Trace(err))
				continue
			}
		}

		// Package chart again
		klog.V(3).Infof("Packaging %q", id)
		pkgCli := helm.NewPackage()
		pkgCli.Destination = outDir
		tgz, err := pkgCli.Run(chartPath, nil)
		if err != nil {
			klog.Errorf("unable to package %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Trace(err))
			continue
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
