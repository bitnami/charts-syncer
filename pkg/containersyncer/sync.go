package containersyncer

import (
	goerrors "errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/bitnami/charts-syncer/pkg/client/config"
	"github.com/bitnami/charts-syncer/pkg/client/types"
	"github.com/bitnami/charts-syncer/pkg/httputils"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/juju/errors"
	log "github.com/vmware-labs/distribution-tooling-for-helm/pkg/dtlog"
)

// ErrNoContainersToSync is returned when there are no containers to sync
var ErrNoContainersToSync = errors.New("no containers to sync")

func (s *Syncer) syncContainer(c *types.ContainerImage, tag string, l log.SectionLogger) error {
	id := fmt.Sprintf("%s:%s", c.Reference, tag)

	workdir, err := os.MkdirTemp("", "charts-syncer")
	if err != nil {
		l.Errorf("unable to create work directory for %q container: %+v", id, err)
		return errors.Trace(err)
	}
	defer os.RemoveAll(workdir)

	wrapDest := filepath.Join(workdir, "wraps", fmt.Sprintf("%s-%s.container.wrap.tgz", c.Reference.ImageName, tag))
	wrappedContainerPath, err := s.cli.src.WrapContainer(
		id,
		wrapDest,
		config.WithLogger(l), config.WithWorkDir(workdir),
		config.WithContainerPlatforms(s.containerPlatforms),
		config.WithSkipArtifacts(s.skipArtifacts),
	)
	if err != nil {
		return errors.Annotatef(err, "unable to move container %q with charts-syncer", id)
	}

	l.Infof("Wrapped container path is: %s", wrappedContainerPath)

	if s.dryRun {
		l.Warnf("Dry-run mode is enabled. Upload of container %q is being skipped.", id)
		return nil
	}

	if err := s.cli.dst.UnwrapContainer(wrappedContainerPath, config.WithLogger(l), config.WithWorkDir(workdir)); err != nil {
		l.Errorf("unable to upload %q container: %+v", id, err)
		return errors.Trace(err)
	}

	return nil
}

// DiffPendingContainers calculates the missing containers in the target registry
func (s *Syncer) DiffPendingContainers(names ...string) ([]*types.ContainerImage, error) {
	var containers []*types.ContainerImage

	for _, n := range names {
		sourceTags, err := s.cli.src.ListContainerTags(n)
		if err != nil {
			return nil, err
		}

		// Filter by latest tag
		if s.latestVersionOnly {
			if len(sourceTags) == 0 {
				continue
			}
			latestVersionTag, err := getHighestVersionTag(sourceTags)
			if err != nil {
				return nil, err
			}
			sourceTags = []string{latestVersionTag}
		}

		for _, tag := range sourceTags {
			if ok, hErr := s.cli.dst.HasContainer(n, tag); hErr != nil {
				return nil, hErr
			} else if !ok {
				var ref *types.ImageReference
				if c := s.source.GetContainers(); c != nil {
					imageRef := fmt.Sprintf("%s/%s", httputils.RemoveSchema(c.GetUrl()), n)
					parsedRef, err := name.ParseReference(imageRef)
					if err != nil {
						return nil, errors.Trace(err)
					}
					ref = &types.ImageReference{
						Registry:   parsedRef.Context().RegistryStr(),
						Repository: parsedRef.Context().RepositoryStr(),
						ImageName:  path.Base(parsedRef.Context().RepositoryStr()),
					}
				} else {
					// LOCAL source: build reference directly from the image name without
					// going through name.ParseReference, which would add Docker Hub defaults
					// (e.g. "nginx" → "index.docker.io/library/nginx").
					ref = &types.ImageReference{
						Registry:   "",
						Repository: n,
						ImageName:  path.Base(n),
					}
				}
				containers = append(containers, &types.ContainerImage{
					Reference: ref,
					Tags:      []string{tag},
				})
			}
		}
	}

	return containers, nil
}

// SyncPendingContainers syncs the containers not found in the target
func (s *Syncer) SyncPendingContainers(containers []*types.ContainerImage) error {
	var errs error

	if len(containers) == 0 {
		s.logger.Infof("There are no containers out of sync!")
		return ErrNoContainersToSync
	}

	for i, c := range containers {
		for _, tag := range c.Tags {
			id := fmt.Sprintf("%s:%s", c.Reference, tag)
			if err := s.logger.Section(fmt.Sprintf("==> Syncing %q container (%d/%d)", id, i+1, len(containers)), func(l log.SectionLogger) error {
				return s.syncContainer(c, tag, l)
			}); err != nil {
				s.logger.Warnf("Failed syncing %q container: %v", id, err)
				errs = goerrors.Join(errs, errors.Trace(err))
			}
		}
	}

	return errors.Trace(errs)
}
