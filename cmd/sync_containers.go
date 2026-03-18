package main

import (
	goerrors "errors"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	log "github.com/vmware-labs/distribution-tooling-for-helm/pkg/dtlog"

	"github.com/bitnami/charts-syncer/pkg/containersyncer"
	"github.com/juju/errors"
)

func runContainersSyncer(parentLog log.SectionLogger, c *apiv1.Config) error {
	l := parentLog.StartSection("Syncing containers")
	syncerOptions := []containersyncer.Option{
		containersyncer.WithDryRun(rootDryRun),
		containersyncer.WithWorkdir(syncWorkdir),
		containersyncer.WithContainerPlatforms(c.GetContainerPlatforms()),
		containersyncer.WithInsecure(rootInsecure),
		containersyncer.WithLatestVersionOnly(syncLatestVersionOnly),
		containersyncer.WithSkipArtifacts(c.GetSkipArtifacts()),
		containersyncer.WithUsePlainHTTP(usePlainHTTP),

		containersyncer.WithLogger(l),
	}
	s, err := containersyncer.New(c.GetSource(), c.GetTarget(), syncerOptions...)
	if err != nil {
		return errors.Trace(err)
	}

	pendingImages, err := s.DiffPendingContainers(c.GetContainers()...)
	if err != nil {
		return errors.Trace(err)
	}
	if len(pendingImages) == 0 {
		parentLog.Successf("There are no containers out of sync!")
		return nil
	}

	if len(pendingImages) > 1 {
		l.Infof("There are %d containers out of sync!", len(pendingImages))
	} else {
		l.Infof("There is %d container out of sync!", len(pendingImages))
	}

	if err := s.SyncPendingContainers(pendingImages); err != nil {
		if goerrors.Is(err, containersyncer.ErrNoContainersToSync) {
			parentLog.Successf("There are no containers out of sync!")
			return nil
		}
		return l.Failf("Error syncing containers: %v", err)
	}
	parentLog.Successf("Containers synced successfully")

	return nil
}
