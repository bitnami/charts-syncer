package repo

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/chartmuseumtest"
)

func TestPublishToChartmuseum(t *testing.T) {
	for _, test := range chartmuseumtest.Tests {
		t.Run(test.Desc, func(t *testing.T) {
			// Check if the test should be skipped or allowed.
			test.Skip(t)

			url, cleanup := test.MakeServer(t)
			defer cleanup()

			// Define target repo
			target := &api.TargetRepo{
				Repo: &api.Repo{
					Url:  url,
					Kind: "CHARTMUSEUM",
					Auth: &api.Auth{
						Username: "user",
						Password: "password",
					},
				},
				ContainerRegistry:   "test.registry.io",
				ContainerRepository: "test/repo",
			}

			// Create client for target repo
			tc, err := NewClient(target.Repo)
			if err != nil {
				t.Fatal("could not create a client for the target repo", err)
			}
			err = tc.PublishChart("../../testdata/apache-7.3.15.tgz", target.Repo)
			if err != nil {
				t.Fatal(err)
			}

			// Check the chart really was added to the service's index.
			req, err := http.NewRequest("GET", target.Repo.Url+"/api/charts/apache", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth(target.Repo.Auth.Username, target.Repo.Auth.Password)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			charts := []*chartmuseumtest.ChartVersion{}
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
	for _, test := range chartmuseumtest.Tests {
		t.Run(test.Desc, func(t *testing.T) {
			// Check if the test should be skipped or allowed.
			test.Skip(t)

			url, cleanup := test.MakeServer(t)
			defer cleanup()

			// Define target repo
			source := &api.SourceRepo{
				Repo: &api.Repo{
					Url:  url,
					Kind: "CHARTMUSEUM",
					Auth: &api.Auth{
						Username: "user",
						Password: "password",
					},
				},
			}
			// Create client for source repo
			sc, err := NewClient(source.Repo)
			if err != nil {
				t.Fatal("could not create a client for the target repo", err)
			}

			// If testing real docker chartmuseum, we must push the chart before download it
			if test.Desc == "real service" {
				sc.PublishChart("../../testdata/apache-7.3.15.tgz", source.Repo)
			}

			// Create temporary working directory
			testTmpDir, err := ioutil.TempDir("", "c3tsyncer-tests")
			defer os.RemoveAll(testTmpDir)
			if err != nil {
				t.Errorf("Error creating temporary: %s", testTmpDir)
			}

			chartPath := path.Join(testTmpDir, "apache-7.3.15.tgz")
			err = sc.DownloadChart(chartPath, "apache", "7.3.15", source.Repo)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(chartPath); err != nil {
				t.Errorf("Expected %s to exists", chartPath)
			}
		})
	}
}

func TestChartExistsInChartMuseum(t *testing.T) {
	// Define source repo
	source := &api.SourceRepo{
		Repo: &api.Repo{
			// This repo is not a chartmuseum repo but there are no differences
			// for the ChartExists function.
			Url:  "https://charts.bitnami.com/bitnami",
			Kind: "CHARTMUSEUM",
		},
	}
	// Create client for source repo
	sc, err := NewClient(source.Repo)
	if err != nil {
		t.Fatal("could not create a client for the source repo", err)
	}
	chartExists, err := sc.ChartExists("grafana", "1.5.2", source.Repo)
	if err != nil {
		t.Fatal(err)
	}
	if !chartExists {
		t.Errorf("grafana-1.5.2 chart should exists")
	}
}
