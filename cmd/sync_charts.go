package main

import (
	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	log "github.com/vmware-labs/distribution-tooling-for-helm/pkg/dtlog"

	"github.com/bitnami/charts-syncer/pkg/chartsyncer"
	"github.com/juju/errors"
)

func runChartsSyncer(parentLog log.SectionLogger, c *apiv1.Config) error {
	l := parentLog.StartSection("Syncing charts")

	syncerOptions := []chartsyncer.Option{
		// TODO(jdrios): Some backends may not support discovery
		chartsyncer.WithAutoDiscovery(true),
		chartsyncer.WithDryRun(rootDryRun),
		chartsyncer.WithFromDate(syncFromDate),
		chartsyncer.WithWorkdir(syncWorkdir),
		chartsyncer.WithContainerPlatforms(c.GetContainerPlatforms()),
		chartsyncer.WithInsecure(rootInsecure),
		chartsyncer.WithLatestVersionOnly(syncLatestVersionOnly),
		chartsyncer.WithSkipArtifacts(c.GetSkipArtifacts()),
		chartsyncer.WithSkipImages(c.GetSkipImages()),
		chartsyncer.WithSkipCharts(c.SkipCharts),
		chartsyncer.WithUsePlainHTTP(usePlainHTTP),

		chartsyncer.WithLogger(l),
	}
	s, err := chartsyncer.New(c.GetSource(), c.GetTarget(), syncerOptions...)
	if err != nil {
		return errors.Trace(err)
	}
	if err := s.SyncPendingCharts(c.GetCharts()...); err != nil {
		if err == chartsyncer.ErrNoChartsToSync {
			parentLog.Successf("There are no charts out of sync!")
			return nil
		}
		return l.Failf("Error syncing charts: %v", err)
	}
	parentLog.Successf("Charts synced successfully")

	return nil
}
