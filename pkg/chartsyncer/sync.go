package chartsyncer

import (
	goerrors "errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/pkg/client/config"
	"github.com/juju/errors"
	log "github.com/vmware-labs/distribution-tooling-for-helm/pkg/dtlog"
	helmchart "helm.sh/helm/v3/pkg/chart"
)

// ErrNoChartsToSync is returned when there are no charts to sync
var ErrNoChartsToSync = errors.New("no charts to sync")

// preservedSourceRef returns the bare oci:// reference for chartName in the
// syncer's source repo, or "" if the source isn't an OCI registry.
//
// charts-syncer always fetches ch.TgzPath to a local file before wrapping
// (needed to inspect the chart for indexing/dependency resolution), so
// WrapChart never sees an oci:// inputPath on its own. Passing this
// alongside the local tgz lets the wrap library still capture the pristine
// source manifest for PreserveDigest, instead of only the tgz bytes.
func (s *Syncer) preservedSourceRef(chartName string) string {
	repo := s.source.GetRepo()
	if repo.GetKind() != apiv1.Kind_OCI {
		return ""
	}
	u, err := url.Parse(repo.GetUrl())
	if err != nil {
		return ""
	}
	return fmt.Sprintf("oci://%s%s/%s", u.Host, u.Path, chartName)
}

func (s *Syncer) syncChart(ch *Chart, l log.SectionLogger) error {
	id := fmt.Sprintf("%s-%s", ch.Name, ch.Version)

	outdir, err := os.MkdirTemp("", "charts-syncer")
	if err != nil {
		l.Errorf("unable to create output directory for %q chart: %+v", id, err)
		return errors.Trace(err)
	}
	defer os.RemoveAll(outdir)

	workdir, err := os.MkdirTemp("", "charts-syncer")
	if err != nil {
		l.Errorf("unable to create work directory for %q chart: %+v", id, err)
		return errors.Trace(err)
	}
	defer os.RemoveAll(workdir)

	// Some client Upload() methods needs this info
	metadata := &helmchart.Metadata{
		Name:    ch.Name,
		Version: ch.Version,
	}

	wrapOpts := []config.Option{
		config.WithLogger(l), config.WithWorkDir(workdir),
		config.WithContainerPlatforms(s.containerPlatforms), config.WithSkipArtifacts(s.skipArtifacts),
		config.WithSkipImages(s.skipImages), config.WithPreserveDigest(s.preserveDigest),
	}
	if s.preserveDigest {
		wrapOpts = append(wrapOpts, config.WithPreservedSourceRef(s.preservedSourceRef(ch.Name)))
	}

	wrappedChartPath, err := s.cli.src.WrapChart(ch.TgzPath,
		filepath.Join(workdir, "wraps", fmt.Sprintf("%s-%s.wrap.tgz", ch.Name, ch.Version)),
		wrapOpts...,
	)
	if err != nil {
		return errors.Annotatef(err, "unable to move chart %q with charts-syncer", id)
	}

	if s.dryRun {
		l.Warnf("Dry-run mode is enabled. Upload of chart %q is being skipped.", id)
		return nil
	}

	if err := s.cli.dst.UnwrapChart(
		wrappedChartPath, metadata,
		config.WithLogger(l),
		config.WithWorkDir(workdir),
		config.WithSkipImages(s.skipImages),
		config.WithPreserveDigest(s.preserveDigest),
	); err != nil {
		l.Errorf("unable to upload %q chart: %+v", id, err)
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
		s.logger.Infof("There are no charts out of sync!")
		return ErrNoChartsToSync
	}

	s.logger.Infof(msg)

	for i, ch := range charts {
		id := fmt.Sprintf("%s-%s", ch.Name, ch.Version)
		if err := s.logger.Section(fmt.Sprintf("==> Syncing %q chart (%d/%d)", id, i+1, len(charts)), func(l log.SectionLogger) error {
			return s.syncChart(ch, l)
		}); err != nil {
			s.logger.Warnf("Failed syncing %q chart: %v", id, err)
			errs = goerrors.Join(errs, errors.Trace(err))
		}
	}
	return errors.Trace(errs)
}
