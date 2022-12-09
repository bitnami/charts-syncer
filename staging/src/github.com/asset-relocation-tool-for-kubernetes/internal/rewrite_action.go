// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package internal

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/divideandconquer/go-merge/merge"
	"github.com/google/go-containerregistry/pkg/name"
	yamlops2 "github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal/yamlops"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

type OCIImageLocation struct {
	Registry         string
	PrefixRegistry   string
	RepositoryPrefix string
}
type RewriteAction struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

func (a *RewriteAction) TopLevelKey() string {
	return strings.Split(a.Path, ".")[1]
}

// removes the first part of the dot delimited string
//.sub-1.foo.bar => .foo.bar
func (a *RewriteAction) stripPrefix() string {
	// Starting in 2 since there is an empty string as first element
	return "." + strings.Join(strings.Split(a.Path, ".")[2:], ".")
}

func (a *RewriteAction) GetPathToMap() string {
	pathParts := strings.Split(a.Path, ".")
	return strings.Join(pathParts[:len(pathParts)-1], ".")
}

func (a *RewriteAction) GetSubPathToMap() string {
	pathParts := strings.Split(a.Path, ".")
	return "." + strings.Join(pathParts[2:len(pathParts)-1], ".")
}

func (a *RewriteAction) GetKey() string {
	pathParts := strings.Split(a.Path, ".")
	return pathParts[len(pathParts)-1]
}

func (a *RewriteAction) ToMap() map[string]interface{} {
	keys := strings.Split(strings.TrimPrefix(a.Path, "."), ".")
	var node ValuesMap
	var value interface{} = a.Value

	for i := len(keys) - 1; i >= 0; i-- {
		key := keys[i]
		node = make(ValuesMap)
		node[key] = value
		value = node
	}

	return node
}

// Apply will try to execute the rewrite action declaration on the given Helm Chart or sub-charts
func (a *RewriteAction) Apply(chart *chart.Chart) error {
	chartToApply, relativeRewriteRule := a.FindChartDestination(chart)
	return applyUpdate(chartToApply, relativeRewriteRule)
}

// Apply the yaml update described in the rewrite action to the provided Helm Chart
func applyUpdate(chart *chart.Chart, a *RewriteAction) error {
	valuesIndex, data := getChartValues(chart)
	value := map[string]string{
		a.GetKey(): a.Value,
	}

	newData, err := yamlops2.UpdateMap(data, a.GetPathToMap(), "", nil, value)
	if err != nil {
		return fmt.Errorf("failed to apply modification to %s: %w", chart.Name(), err)
	}

	chart.Raw[valuesIndex].Data = newData

	return nil
}

// FindChartDestination will recursively find the Helm Chart a rewrite action will apply to
// by starting on a parentChart
// Additionally it will return the rewrite action with path relative to that Helm Chart
func (a *RewriteAction) FindChartDestination(parentChart *chart.Chart) (*chart.Chart, *RewriteAction) {
	for _, subchart := range parentChart.Dependencies() {
		if _, ok := parentChart.Values[a.TopLevelKey()]; ok {
			continue
		}
		if subchart.Name() == a.TopLevelKey() {
			// Recursively perform the check stripping out the rewrite prefix
			// and providing the actual subchart reference
			subChartRewriteAction := &RewriteAction{
				Path:  a.stripPrefix(),
				Value: a.Value,
			}

			return subChartRewriteAction.FindChartDestination(subchart)
		}
	}

	return parentChart, a
}

func getChartValues(chart *chart.Chart) (int, []byte) {
	for fileIndex, file := range chart.Raw {
		if file.Name == chartutil.ValuesfileName {
			return fileIndex, file.Data
		}
	}
	return -1, nil
}

type ValuesMap map[string]interface{}

func buildValuesMap(chart *chart.Chart) map[string]interface{} {
	values := chart.Values
	if values == nil {
		values = map[string]interface{}{}
	}

	// Add values for chart dependencies
	for _, dependency := range chart.Dependencies() {
		// recursively load the dependency values
		values[dependency.Name()] = merge.Merge(buildValuesMap(dependency), values[dependency.Name()])
	}

	return values
}

func applyRewrites(values map[string]interface{}, rewriteActions []*RewriteAction) (map[string]interface{}, error) {
	for _, action := range rewriteActions {
		actionMap := action.ToMap()
		result := merge.Merge(values, actionMap)
		var ok bool
		values, ok = result.(map[string]interface{})
		if !ok {
			return nil, errors.New("can't apply rewrites to Chart values, invalid format")
		}
	}

	return values, nil
}

func (t *ImageTemplate) Render(chart *chart.Chart, insecure bool, rewriteActions ...*RewriteAction) (name.Reference, error) {
	values := buildValuesMap(chart)

	// Apply rewrite actions
	var err error
	values, err = applyRewrites(values, rewriteActions)
	if err != nil {
		return nil, err
	}

	output := bytes.Buffer{}
	err = t.Template.Execute(&output, values)
	if err != nil {
		return nil, fmt.Errorf("failed to render image: %w", err)
	}

	var image name.Reference
	if insecure {
		image, err = name.ParseReference(output.String(), name.Insecure)
	} else {
		image, err = name.ParseReference(output.String())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse image reference: %w", err)
	}

	return image, nil
}

func (t *ImageTemplate) Apply(originalImage name.Repository, imageDigest string, rules *OCIImageLocation) ([]*RewriteAction, error) {
	var rewrites []*RewriteAction

	registry := originalImage.Registry.Name()
	if rules.PrefixRegistry != "" {
		registry = rules.PrefixRegistry
	} else if rules.Registry != "" {
		registry = rules.Registry
	}

	// Repository path should contain the repositoryPrefix + imageName
	repository := originalImage.RepositoryStr()
	if rules.RepositoryPrefix != "" {
		repoParts := strings.Split(originalImage.RepositoryStr(), "/")
		imageName := repoParts[len(repoParts)-1]
		repository = fmt.Sprintf("%s/%s", rules.RepositoryPrefix, imageName)
	}

	if rules.PrefixRegistry != "" {
		if rules.Registry != "" {
			repository = fmt.Sprintf("%s/%s", rules.Registry, repository)
		} else {
			repository = fmt.Sprintf("%s/%s", originalImage.Registry.Name(), repository)
		}
	}

	// Append the image digest unless the tag or digest are explicitly encoded in the template
	// By doing so, we default to immutable references
	if t.TagTemplate == "" && t.DigestTemplate == "" {
		repository = fmt.Sprintf("%s@%s", repository, imageDigest)
	}

	registryChanged := originalImage.Registry.Name() != registry
	repoChanged := originalImage.RepositoryStr() != repository

	// The registry and the repository as encoded in a single template placeholder
	if t.RegistryAndRepositoryTemplate != "" && (registryChanged || repoChanged) {
		rewrites = append(rewrites, &RewriteAction{
			Path:  t.RegistryAndRepositoryTemplate,
			Value: fmt.Sprintf("%s/%s", registry, repository),
		})
	} else {
		// Explicitly override the registry
		if t.RegistryTemplate != "" && registryChanged {
			rewrites = append(rewrites, &RewriteAction{
				Path:  t.RegistryTemplate,
				Value: registry,
			})
		}

		// Explicitly override the repository
		if t.RepositoryTemplate != "" && repoChanged {
			rewrites = append(rewrites, &RewriteAction{
				Path:  t.RepositoryTemplate,
				Value: repository,
			})
		}
	}

	return rewrites, nil
}
