package chart

import (
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/juju/errors"

	"github.com/bitnami-labs/chart-repository-syncer/api"
)

// updateValuesFile performs some substitutions to a given values.yaml file
func updateValuesFile(valuesFile string, targetRepo *api.TargetRepo) error {
	if err := updateContainerImageRegistry(valuesFile, targetRepo); err != nil {
		return errors.Annotatef(err, "Error updating %s file", valuesFile)
	}
	if err := updateContainerImageRepository(valuesFile, targetRepo); err != nil {
		return errors.Annotatef(err, "Error updating %s file", valuesFile)
	}
	return nil
}

// updateContainerImageRepository updates the container repository in a values.yaml file
func updateContainerImageRepository(valuesFile string, targetRepo *api.TargetRepo) error {
	regex := regexp.MustCompile(`(?m)(repository:[[:blank:]])(.*)(/)`)
	values, err := ioutil.ReadFile(valuesFile)
	if err != nil {
		return errors.Trace(err)
	}
	submatch := regex.FindStringSubmatch(string(values))
	if len(submatch) > 0 {
		replaceLine := fmt.Sprintf("%s%s%s", submatch[1], targetRepo.ContainerRepository, submatch[3])
		newContents := regex.ReplaceAllString(string(values), replaceLine)
		err = ioutil.WriteFile(valuesFile, []byte(newContents), 0)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return errors.Trace(err)
}

// updateContainerImageRegistry updates the container registry in a values.yaml file
func updateContainerImageRegistry(valuesFile string, targetRepo *api.TargetRepo) error {
	regex := regexp.MustCompile(`(?m)(registry:[[:blank:]])(.*)(.*$)`)
	values, err := ioutil.ReadFile(valuesFile)
	if err != nil {
		return errors.Trace(err)
	}
	submatch := regex.FindStringSubmatch(string(values))
	if len(submatch) > 0 {
		replaceLine := fmt.Sprintf("%s%s%s", submatch[1], targetRepo.ContainerRegistry, submatch[3])
		newContents := regex.ReplaceAllString(string(values), replaceLine)
		err = ioutil.WriteFile(valuesFile, []byte(newContents), 0)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return errors.Trace(err)
}
