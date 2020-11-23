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
	sourceCM = &api.SourceRepo{
		Repo: &api.Repo{
			Url:  "http://fake.source.com",
			Kind: api.Kind_CHARTMUSEUM,
			Auth: &api.Auth{
				Username: "user",
				Password: "password",
			},
		},
	}
	targetCM = &api.TargetRepo{
		Repo: &api.Repo{
			Url:  "http://fake.target.com",
			Kind: api.Kind_CHARTMUSEUM,
			Auth: &api.Auth{
				Username: "user",
				Password: "password",
			},
		},
		ContainerRegistry:   "test.registry.io",
		ContainerRepository: "test/repo",
	}
)

func TestPublishToChartmuseum(t *testing.T) {
	for _, test := range chartrepotest.ChartMuseumTests {
		t.Run(test.Desc, func(t *testing.T) {
			// Check if the test should be skipped or allowed.
			test.Skip(t)

			url, cleanup := test.MakeServer(t)
			defer cleanup()

			// Update source repo url
			targetCM.Repo.Url = url

			// Create client for target repo
			tc, err := NewClient(targetCM.Repo)
			if err != nil {
				t.Fatal("could not create a client for the target repo", err)
			}
			err = tc.Push("../../../testdata/apache-7.3.15.tgz", targetCM.Repo)
			if err != nil {
				t.Fatal(err)
			}

			// Check the chart really was added to the service's index.
			req, err := http.NewRequest("GET", targetCM.Repo.Url+"/api/charts/apache", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth(targetCM.Repo.Auth.Username, targetCM.Repo.Auth.Password)

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

func TestDownloadFromChartmuseum(t *testing.T) {
	for _, test := range chartrepotest.ChartMuseumTests {
		t.Run(test.Desc, func(t *testing.T) {
			// Check if the test should be skipped or allowed.
			test.Skip(t)

			url, cleanup := test.MakeServer(t)
			defer cleanup()

			// Update source repo url
			sourceCM.Repo.Url = url

			// Create client for source repo
			sc, err := NewClient(sourceCM.Repo)
			if err != nil {
				t.Fatal("could not create a client for the target repo", err)
			}

			// If testing real docker chartmuseum, we must push the chart before download it
			if test.Desc == "real service" {
				sc.Push("../../../testdata/apache-7.3.15.tgz", sourceCM.Repo)
			}

			// Create temporary working directory
			testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
			if err != nil {
				t.Fatalf("error creating temporary: %s", testTmpDir)
			}
			defer os.RemoveAll(testTmpDir)

			sourceIndex := repo.NewIndexFile()
			sourceIndex.Add(&chart.Metadata{Name: "apache", Version: "7.3.15"}, "apache-7.3.15.tgz", url+"/charts", "sha256:1234567890")

			chartPath := path.Join(testTmpDir, "apache-7.3.15.tgz")
			err = sc.Fetch(chartPath, "apache", "7.3.15", sourceCM.Repo, sourceIndex)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(chartPath); err != nil {
				t.Errorf("expected %s to exists", chartPath)
			}
		})
	}
}

func TestChartExistsInChartMuseum(t *testing.T) {
	sourceIndex := repo.NewIndexFile()
	sourceIndex.Add(&chart.Metadata{Name: "grafana", Version: "1.5.2"}, "grafana-1.5.2.tgz", "https://fake-url.com/charts", "sha256:1234567890")
	// Create client for source repo
	sc, err := NewClient(sourceCM.Repo)
	if err != nil {
		t.Fatal("could not create a client for the source repo", err)
	}
	chartExists, err := sc.ChartExists("grafana", "1.5.2", sourceIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !chartExists {
		t.Errorf("grafana-1.5.2 chart should exists")
	}
}
