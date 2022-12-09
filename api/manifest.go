package api

import (
	"github.com/pkg/errors"

	"github.com/bitnami-labs/charts-syncer/pkg/util/chartutil"
)

// Validate validates the config file is correct
func (m *Manifest) Validate() error {
	// check config chart name and versions
	for _, manifest := range m.Spec.Manifests {
		for _, chart := range manifest.Charts {
			if err := chartutil.ValidateChartName(chart.GetName()); err != nil {
				return errors.Errorf(`"charts name %s is invalid  %s`, chart.Name, err.Error())
			}
			for _, version := range chart.Versions {
				if err := chartutil.ValidateChartVersion(version); err != nil {
					return errors.Errorf(`"charts %s version %s is invalid, %s"`, chart.Name, version, err.Error())
				}
			}
		}
	}
	return nil
}
