package chart

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/juju/errors"

	"github.com/bitnami/charts-syncer/api"
)

var (
	repositoryRegex = regexp.MustCompile(`(?m)(repository:[[:blank:]])(.*)(/)`)
	registryRegex   = regexp.MustCompile(`(?m)(registry:[[:blank:]])(.*)(.*$)`)
)

// updateValuesFile performs some substitutions to a given values.yaml file.
func updateValuesFile(valuesFile string, targetRepo *api.Target) error {
	if err := updateContainerImageRegistry(valuesFile, targetRepo); err != nil {
		return errors.Annotatef(err, "error updating %s file", valuesFile)
	}
	if err := updateContainerImageRepository(valuesFile, targetRepo); err != nil {
		return errors.Annotatef(err, "error updating %s file", valuesFile)
	}
	return nil
}

// updateContainerImageRepository updates the container repository in a values.yaml file.
func updateContainerImageRepository(valuesFile string, target *api.Target) error {
	values, err := os.ReadFile(valuesFile)
	if err != nil {
		return errors.Trace(err)
	}
	submatch := repositoryRegex.FindStringSubmatch(string(values))
	if len(submatch) > 0 {
		replaceLine := fmt.Sprintf("%s%s%s", submatch[1], target.ContainerRepository, submatch[3])
		newContents := repositoryRegex.ReplaceAllString(string(values), replaceLine)
		err = os.WriteFile(valuesFile, []byte(newContents), 0)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return errors.Trace(err)
}

// updateContainerImageRegistry updates the container registry in a values.yaml file.
func updateContainerImageRegistry(valuesFile string, target *api.Target) error {
	values, err := os.ReadFile(valuesFile)
	if err != nil {
		return errors.Trace(err)
	}
	submatch := registryRegex.FindStringSubmatch(string(values))
	if len(submatch) > 0 {
		replaceLine := fmt.Sprintf("%s%s%s", submatch[1], target.ContainerRegistry, submatch[3])
		newContents := registryRegex.ReplaceAllString(string(values), replaceLine)
		err = os.WriteFile(valuesFile, []byte(newContents), 0)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return errors.Trace(err)
}

// updateReadmeFile performs some substitutions to a given README.md file.
func updateReadmeFile(readmeFile, sourceURL, targetURL, chartName, repoName string) error {
	readme, err := os.ReadFile(readmeFile)
	if err != nil {
		return errors.Trace(err)
	}
	// Update helm repo add with string replacement
	addBitnamiRepoLine := fmt.Sprintf("helm repo add bitnami %s", sourceURL)
	addCustomRepoLine := fmt.Sprintf("helm repo add %s %s", repoName, targetURL)
	newContent := strings.ReplaceAll(string(readme), addBitnamiRepoLine, addCustomRepoLine)
	// Update bitnami/chart references with regex
	regexString := fmt.Sprintf(`(?m)(\s)(bitnami/%s)(\s)`, chartName)
	regex, err := regexp.Compile(regexString)
	if err != nil {
		return errors.Trace(err)
	}
	submatch := regex.FindStringSubmatch(string(readme))
	if len(submatch) > 0 {
		replaceText := fmt.Sprintf("%s%s/%s%s", submatch[1], repoName, chartName, submatch[3])
		newContent = regex.ReplaceAllString(newContent, replaceText)
	}
	return errors.Trace(os.WriteFile(readmeFile, []byte(newContent), 0))
}
