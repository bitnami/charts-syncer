package syncer

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/bitnami-labs/charts-syncer/internal/chart"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/pkg/mover"
	"gopkg.in/yaml.v2"
	helm "helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"k8s.io/klog"
)

// SyncPendingCharts syncs the charts not found in the target
//
// It uses topological sort to sync dependencies first.
func (s *Syncer) SyncPendingCharts(names ...string) error {
	var errs error

	// There might be problems loading all the charts due to missing dependencies,
	// invalid/wrong charts in the repository, etc. Therefore, let's warn about
	// them instead of blocking the whole sync.
	err := s.loadCharts(names...)
	if err != nil {
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

	for _, ch := range charts {
		id := fmt.Sprintf("%s-%s", ch.Name, ch.Version)
		klog.Infof("Syncing %q chart...", id)

		klog.V(3).Infof("Processing %q chart...", id)
		outdir, err := ioutil.TempDir("", "charts-syncer")
		if err != nil {
			klog.Errorf("unable to create output directory for %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}
		defer os.RemoveAll(outdir)

		hasDeps := len(ch.Dependencies) > 0

		workdir, err := ioutil.TempDir("", "charts-syncer")
		if err != nil {
			klog.Errorf("unable to create work directory for %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}
		defer os.RemoveAll(workdir)

		// Some client Upload() methods needs this info
		metadata := &helmchart.Metadata{
			Name:    ch.Name,
			Version: ch.Version,
		}
		var packagedChartPath string

		if s.relocateContainerImages {
			packagedChartPath, err = s.SyncWithRelok8s(ch, outdir)
			if err != nil {
				errs = multierror.Append(errs, errors.Annotatef(err, "unable to move chart %q with relok8s", id))
				continue
			}
		} else {
			packagedChartPath, err = s.SyncWithChartsSyncer(ch, id, workdir, outdir, hasDeps)
			if err != nil {
				errs = multierror.Append(errs, errors.Annotatef(err, "unable to move chart %q with charts-syncer", id))
				continue
			}
		}

		if s.dryRun {
			klog.Infof("dry-run: Uploading %q chart", id)
			continue
		}

		klog.V(3).Infof("Uploading %q chart...", id)
		if err := s.cli.dst.Upload(packagedChartPath, metadata); err != nil {
			klog.Errorf("unable to upload %q chart: %+v", id, err)
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}
	}

	return errors.Trace(errs)
}

// SyncWithRelok8s will take a local packaged chart, a container registry and a container repository and will rewrite the chart
// updating the images in values.yaml. The local chart must include an image hints file so relok8s library knows how to
// update the images
func (s *Syncer) SyncWithRelok8s(chart *Chart, outdir string) (string, error) {
	// Once https://github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/issues/94 is solved, we could
	// specify the name we want for the output file. Until then, we should keep using this template thing
	outputChartPath := filepath.Join(outdir, "%s-%s.relocated.tgz")
	packagedChartPath := filepath.Join(outdir, fmt.Sprintf("%s-%s.relocated.tgz", chart.Name, chart.Version))
	req := &mover.ChartMoveRequest{
		Source: mover.Source{
			Chart: mover.ChartSpec{
				Local: &mover.LocalChart{
					// This chart has a .relok8s-images.yaml file inside so no need to explicitly pass that
					Path: chart.TgzPath,
				},
			},
		},
		Target: mover.Target{
			Rules: mover.RewriteRules{
				Registry:         s.target.GetContainerRegistry(),
				RepositoryPrefix: s.target.GetContainerRepository(),
				ForcePush:        true,
			},
			Chart: mover.ChartSpec{
				Local: &mover.LocalChart{
					// output will be [chart-name]-[chart-version].relocated.tgz
					Path: outputChartPath,
				},
			},
		},
	}
	chartMover, err := mover.NewChartMover(req)
	if err != nil {
		klog.Errorf("unable to create chart mover: %+v", err)
		return "", errors.Trace(err)
	}

	if err = chartMover.Move(); err != nil {
		klog.Errorf("unable to move chart %s:%s: %+v", chart.Name, chart.Version, err)
		return "", errors.Trace(err)
	}

	return packagedChartPath, nil
}

func (s *Syncer) SyncWithChartsSyncer(ch *Chart, id, workdir, outdir string, hasDeps bool) (string, error) {
	if err := utils.Untar(ch.TgzPath, workdir); err != nil {
		klog.Errorf("unable to uncompress %q chart: %+v", id, err)
		return "", errors.Trace(errors.Annotatef(err, "uncompressing %q chart", id))
	}

	chartPath := path.Join(workdir, ch.Name)
	if err := chart.ChangeReferences(chartPath, ch.Name, ch.Version, s.source, s.target); err != nil {
		klog.Errorf("unable to process %q chart: %+v", id, err)
		return "", errors.Trace(err)

	}

	// Update deps
	if hasDeps {
		klog.V(3).Infof("Building %q dependencies", id)
		if err := chart.BuildDependencies(chartPath, s.cli.dst, s.source.GetRepo(), s.target.GetRepo()); err != nil {
			klog.Errorf("unable to build %q chart dependencies: %+v", id, err)
			return "", errors.Trace(err)
		}
	}

	// Read final chart metadata
	configFilePath := fmt.Sprintf("%s/Chart.yaml", chartPath)
	chartConfig, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		klog.Errorf("unable to read %q metadata: %+v", id, err)
		return "", errors.Trace(err)
	}
	metadata := &helmchart.Metadata{}
	if err = yaml.Unmarshal(chartConfig, metadata); err != nil {
		klog.Errorf("unable to decode %q metadata: %+v", id, err)
		return "", errors.Trace(err)
	}

	// Package chart again
	klog.V(3).Infof("Packaging %q", id)
	pkgCli := helm.NewPackage()
	pkgCli.Destination = outdir
	packagedChartPath, err := pkgCli.Run(chartPath, nil)
	if err != nil {
		klog.Errorf("unable to package %q chart: %+v", id, err)
		return "", errors.Trace(err)
	}

	return packagedChartPath, nil
}
