package chart

import (
	"path"

	"github.com/juju/errors"
	"k8s.io/klog"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/internal/utils"
)

// ChangeReferences changes the references of a chart tgz file from the source
// repo to the target repo
func ChangeReferences(chartPath, name, version string, source *api.Source, target *api.Target) error {
	// Update values*.yaml
	if target.GetContainerRegistry() == "" && target.GetContainerRepository() == "" {
		// Skip modify value.yaml and readme
		return nil
	}

	for _, f := range []string{
		path.Join(chartPath, ValuesFilename),
		path.Join(chartPath, ValuesProductionFilename),
	} {
		if ok, err := utils.FileExists(f); err != nil {
			return errors.Trace(err)
		} else if ok {
			klog.V(5).Infof("Processing %q file...", f)
			if err := updateValuesFile(f, target); err != nil {
				return errors.Trace(err)
			}
		}
	}

	// Update README.md
	readmeFile := path.Join(chartPath, ReadmeFilename)
	if ok, err := utils.FileExists(readmeFile); err != nil {
		return errors.Trace(err)
	} else if ok {
		klog.V(5).Infof("Processing %q file...", readmeFile)
		if err := updateReadmeFile(
			readmeFile,
			source.GetRepo().GetUrl(),
			target.GetRepo().GetUrl(),
			name,
			target.GetRepoName(),
		); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
