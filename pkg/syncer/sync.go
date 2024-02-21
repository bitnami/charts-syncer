package syncer

import (
	goerrors "errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitnami/charts-syncer/pkg/client/config"
	"github.com/juju/errors"
	"github.com/vmware-labs/distribution-tooling-for-helm/pkg/log"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"k8s.io/klog"
)

// ErrNoChartsToSync is returned when there are no charts to sync
var ErrNoChartsToSync = errors.New("no charts to sync")

func (s *Syncer) syncChart(ch *Chart, l log.SectionLogger) error {
	id := fmt.Sprintf("%s-%s", ch.Name, ch.Version)
	klog.Infof("Syncing %q chart...", id)

	klog.V(3).Infof("Processing %q chart...", id)
	outdir, err := os.MkdirTemp("", "charts-syncer")
	if err != nil {
		klog.Errorf("unable to create output directory for %q chart: %+v", id, err)
		return errors.Trace(err)
	}
	defer os.RemoveAll(outdir)

	workdir, err := os.MkdirTemp("", "charts-syncer")
	if err != nil {
		klog.Errorf("unable to create work directory for %q chart: %+v", id, err)
		return errors.Trace(err)
	}
	defer os.RemoveAll(workdir)

	// Some client Upload() methods needs this info
	metadata := &helmchart.Metadata{
		Name:    ch.Name,
		Version: ch.Version,
	}

	wrappedChartPath, err := s.cli.src.Wrap(ch.TgzPath,
		filepath.Join(workdir, "wraps", fmt.Sprintf("%s-%s.wrap.tgz", ch.Name, ch.Version)),
		config.WithLogger(l), config.WithWorkDir(workdir), config.WithContainerPlatforms(s.containerPlatforms),
	)
	if err != nil {
		return errors.Annotatef(err, "unable to move chart %q with charts-syncer", id)
	}

	if s.dryRun {
		klog.Infof("dry-run: Uploading %q chart", id)
		return nil
	}

	klog.V(3).Infof("Uploading %q chart...", id)

	if err := s.cli.dst.Unwrap(wrappedChartPath, metadata, config.WithLogger(l), config.WithWorkDir(workdir)); err != nil {
		klog.Errorf("unable to upload %q chart: %+v", id, err)
		return errors.Trace(err)
	}
	return nil
}

// SyncPendingCharts syncs the charts not found in the target
func (s *Syncer) SyncPendingCharts(names ...string) error {
	var errs error

	// There might be problems loading all the charts due to
	// invalid/wrong charts in the repository, etc. Therefore, let's warn about
	// them instead of blocking the whole sync.
	if err := s.logger.ExecuteStep("Loading charts", func() error {
		return s.loadCharts(names...)
	}); err != nil {
		s.logger.Warnf("There were some problems loading the information of the requested charts: %v", err)
		errs = goerrors.Join(errs, errors.Trace(err))
	} else {
		s.logger.Infof("Chart list loaded")
	}

	charts := make([]*Chart, len(s.getIndex()))
	i := 0
	for _, ch := range s.getIndex() {
		charts[i] = ch
		i++
	}

	var msg string
	if len(charts) > 1 {
		msg = fmt.Sprintf("There are %d charts out of sync!", len(charts))
	} else if len(charts) == 1 {
		msg = fmt.Sprintf("There is %d chart out of sync!", len(charts))
	} else {
		klog.Info("There are no charts out of sync!")
		return ErrNoChartsToSync
	}

	klog.Info(msg)

	for i, ch := range charts {
		id := fmt.Sprintf("%s-%s", ch.Name, ch.Version)
		if err := s.logger.Section(fmt.Sprintf("Syncing %q chart (%d/%d)", id, i+1, len(charts)), func(l log.SectionLogger) error {
			return s.syncChart(ch, l)
		}); err != nil {
			s.logger.Warnf("Failed syncing %q chart: %v", id, err)
			errs = goerrors.Join(errs, errors.Trace(err))
		}
	}
	return errors.Trace(errs)
}
