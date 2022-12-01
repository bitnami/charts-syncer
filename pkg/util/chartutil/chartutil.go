package chartutil

import (
	"fmt"
	"regexp"

	"github.com/juju/errors"
)

var (
	nameRegexp    = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9+-]*$")
	versionRegexp = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9+-.]*$")
	bundleRegexp  = regexp.MustCompile("(^[a-zA-Z][a-zA-Z0-9+-]*)_([a-zA-Z0-9][a-zA-Z0-9+-.]*)\\.bundle.tar")
)

const maxChartNameLength = 250

func ValidateChartName(name string) error {
	if name == "" || len(name) > maxChartNameLength {
		return fmt.Errorf("chart name must be between 1 and %d characters", maxChartNameLength)
	}
	if !nameRegexp.MatchString(name) {
		return fmt.Errorf("chart name must match the regular expression %q", nameRegexp.String())
	}
	return nil
}

func ValidateChartVersion(version string) error {
	if !versionRegexp.MatchString(version) {
		return errors.Errorf(`"chart version must match the regular expression %s"`, versionRegexp.String())
	}
	return nil
}

func FindStringSubmatch(filename string) []string {
	return bundleRegexp.FindStringSubmatch(filename)
}
