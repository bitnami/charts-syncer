package chartmuseumtest

import (
	"net/http/httptest"
	"os"
	"testing"
)

var (
	username string = "user"
	password string = "password"

	// Execute the test twice, against real & fake ChartMuseum services. This
	// validates the publisher is correct and, at the same time, provides
	// reasonable confidence the fake implementation is good enough.
	Tests = []struct {
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
)
