package core

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/chartrepotest"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

var (
	sourceHarbor = &api.SourceRepo{
		Repo: &api.Repo{
			Url:  "http://fake.source.com/chartrepo/library",
			Kind: api.Kind_HARBOR,
			Auth: &api.Auth{
				Username: "user",
				Password: "password",
			},
		},
	}
	targetHarbor = &api.TargetRepo{
		Repo: &api.Repo{
			Url:  "http://fake.target.com/chartrepo/library",
			Kind: api.Kind_HARBOR,
			Auth: &api.Auth{
				Username: "user",
				Password: "password",
			},
		},
		ContainerRegistry:   "test.registry.io",
		ContainerRepository: "test/repo",
	}
)

func TestPublishToHarbor(t *testing.T) {
	for _, test := range chartrepotest.HarborTests {
		t.Run(test.Desc, func(t *testing.T) {
			// Check if the test should be skipped or allowed.
			test.Skip(t)

			url, cleanup := test.MakeServer(t)
			defer cleanup()

			// Update target repo url
			newURL := url + "/chartrepo/library"
			targetHarbor.Repo.Url = newURL

			// Create client for target repo
			tc, err := NewClient(targetHarbor.Repo)
			if err != nil {
				t.Fatal("could not create a client for the target repo", err)
			}
			err = tc.Push("../../testdata/apache-7.3.15.tgz")
			if err != nil {
				t.Fatal(err)
			}

			// Check the chart really was added to the service's index.
			req, err := http.NewRequest("GET", targetHarbor.Repo.Url+"/apache", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth(targetHarbor.Repo.Auth.Username, targetHarbor.Repo.Auth.Password)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			charts := []*chartrepotest.ChartVersion{}
			if err := json.NewDecoder(resp.Body).Decode(&charts); err != nil {
				t.Fatal(err)
			}

			if got, want := len(charts), 1; got != want {
				t.Fatalf("got: %q, want: %q", got, want)
			}
			if got, want := charts[0].Name, "apache"; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
			if got, want := charts[0].Version, "7.3.15"; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestDownloadFromHarbor(t *testing.T) {
	for _, test := range chartrepotest.HarborTests {
		t.Run(test.Desc, func(t *testing.T) {
			// Check if the test should be skipped or allowed.
			test.Skip(t)

			url, cleanup := test.MakeServer(t)
			defer cleanup()

			// Update source repo url
			newURL := url + "/chartrepo/library"
			sourceHarbor.Repo.Url = newURL

			// Create client for source repo
			sc, err := NewClient(sourceHarbor.Repo)
			if err != nil {
				t.Fatal("could not create a client for the target repo", err)
			}

			// Create temporary working directory
			testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
			if err != nil {
				t.Fatalf("error creating temporary: %s", testTmpDir)
			}
			defer os.RemoveAll(testTmpDir)

			sourceIndex := repo.NewIndexFile()
			sourceIndex.Add(&chart.Metadata{Name: "apache", Version: "7.3.15"}, "apache-7.3.15.tgz", newURL+"/charts", "sha256:1234567890")

			chartPath := path.Join(testTmpDir, "apache-7.3.15.tgz")
			err = sc.Fetch(chartPath, "apache", "7.3.15")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(chartPath); err != nil {
				t.Errorf("expected %s to exists", chartPath)
			}
		})
	}
}

func TestChartExistsInHarbor(t *testing.T) {
	sourceIndex := repo.NewIndexFile()
	sourceIndex.Add(&chart.Metadata{Name: "grafana", Version: "1.5.2"}, "grafana-1.5.2.tgz", "https://fake-url.com/charts", "sha256:1234567890")
	// Create client for source repo
	sc, err := NewClient(sourceHarbor.Repo)
	if err != nil {
		t.Fatal("could not create a client for the source repo", err)
	}
	chartExists, err := sc.ChartExists("grafana", "1.5.2")
	if err != nil {
		t.Fatal(err)
	}
	if !chartExists {
		t.Errorf("grafana-1.5.2 chart should exists")
	}
}
