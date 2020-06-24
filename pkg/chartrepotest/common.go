package chartrepotest

import (
	"net/http/httptest"
	"os"
	"testing"
)

// Metadata in Chart.yaml files
type Metadata struct {
	AppVersion string `json:"appVersion"`
	Name       string `json:"name"`
	Version    string `json:"version"`
}

// ChartVersion type
type ChartVersion struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	URLs    []string `json:"urls"`
}

type httpError struct {
	status int
	body   string
}

var (
	username string = "user"
	password string = "password"

	// ChartMuseumTests defines two tests, using real & fake ChartMuseum services. This
	// validates the publisher is correct and, at the same time, provides
	// reasonable confidence the fake implementation is good enough.
	ChartMuseumTests = []struct {
		Desc       string
		Skip       func(t *testing.T)
		MakeServer func(t *testing.T) (string, func())
	}{
		{
			"real service",
			func(t *testing.T) {
				key := "TEST_WITH_REAL_CHARTMUSEUM"
				if os.Getenv(key) == "" {
					t.Skipf("skipping because %s env var not set", key)
				}
			},
			func(t *testing.T) (string, func()) {
				return tChartMuseumReal(t, username, password)
			},
		},
		{
			"fake service",
			func(t *testing.T) {},
			func(t *testing.T) (string, func()) {
				s := httptest.NewServer(newChartMuseumFake(t, username, password))
				return s.URL, func() {
					s.Close()
				}
			},
		},
	}

	// HarborTests define a fake server for user with Harbor repositories
	HarborTests = []struct {
		Desc       string
		Skip       func(t *testing.T)
		MakeServer func(t *testing.T) (string, func())
	}{
		{
			"fake service",
			func(t *testing.T) {},
			func(t *testing.T) (string, func()) {
				s := httptest.NewServer(newHarborFake(t, username, password))
				return s.URL, func() {
					s.Close()
				}
			},
		},
	}
)
