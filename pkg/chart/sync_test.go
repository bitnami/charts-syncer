package chart

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/bitnami-labs/charts-syncer/pkg/chartrepotest"
	"github.com/bitnami-labs/charts-syncer/pkg/repo"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
	helmRepo "helm.sh/helm/v3/pkg/repo"
)

func TestSync(t *testing.T) {
	for _, test := range chartrepotest.ChartMuseumTests {
		t.Run(test.Desc, func(t *testing.T) {
			// Check if the test should be skipped or allowed.
			test.Skip(t)

			url, cleanup := test.MakeServer(t)
			defer cleanup()

			// Update target url
			target.Repo.Url = url

			sourceIndex, err := utils.LoadIndexFromRepo(source.Repo)
			if err != nil {
				t.Errorf("Error loading index.yaml: %w", err)
			}

			name := "zookeeper"
			version := "5.11.0"
			if err := Sync(name, version, source.Repo, target, sourceIndex, false); err != nil {
				t.Fatal(err)
			}

			// Check the chart really was added to the service's index.
			req, err := http.NewRequest("GET", target.Repo.Url+"/api/charts/zookeeper", nil)
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

			charts := []*chartrepotest.ChartVersion{}
			if err := json.NewDecoder(resp.Body).Decode(&charts); err != nil {
				t.Fatal(err)
			}

			if got, want := len(charts), 1; got != want {
				t.Fatalf("got: %q, want: %q", got, want)
			}
			if got, want := charts[0].Name, "zookeeper"; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
			if got, want := charts[0].Version, "5.11.0"; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}

			// If we have pushed the transformed chart into a real service, we can download it
			// and check that all the expected changes have been applied
			if test.Desc == "real service" {
				// Create temporary working directory
				testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
				t.Logf("Working dir is: %s", testTmpDir)
				if err != nil {
					t.Errorf("Error creating temporary: %s", testTmpDir)
				}
				defer os.RemoveAll(testTmpDir)
				// Create client for target repo
				tc, err := repo.NewClient(target.Repo)
				if err != nil {
					t.Fatal("could not create a client for the source repo", err)
				}
				chartPath := path.Join(testTmpDir, "zookeeper-5.11.0.tgz")
				if err := tc.DownloadChart(chartPath, "zookeeper", "5.11.0", target.Repo, sourceIndex); err != nil {
					t.Fatal(err)
				}
				if err := utils.Untar(chartPath, testTmpDir); err != nil {
					t.Fatal(err)
				}
				// Read values.yaml
				values, err := ioutil.ReadFile(path.Join(testTmpDir, "zookeeper", "values.yaml"))
				if err != nil {
					t.Fatal(err)
				}
				valuesText := string(values)
				expectedRegistryText := "registry: test.registry.io"
				expectedRepositoryText := "repository: test/repo/zookeeper"
				if !strings.Contains(valuesText, expectedRegistryText) {
					t.Errorf("Expected values.yaml file to contain %q", expectedRegistryText)
				}
				if !strings.Contains(valuesText, expectedRepositoryText) {
					t.Errorf("Expected values.yaml file to contain %q", expectedRepositoryText)
				}
			}
		})
	}
}

func TestSyncAllVersions(t *testing.T) {
	for _, test := range chartrepotest.ChartMuseumTests {
		t.Run(test.Desc, func(t *testing.T) {
			// Check if the test should be skipped or allowed.
			test.Skip(t)

			url, cleanup := test.MakeServer(t)
			defer cleanup()

			// Update target url
			target.Repo.Url = url

			name := "zookeeper"
			indexFile := "../../testdata/zookeeper-index.yaml"
			// Load index.yaml info into index object
			index, err := helmRepo.LoadIndexFile(indexFile)
			if err != nil {
				t.Fatal(err)
			}
			if err := SyncAllVersions(name, source.Repo, target, false, index, false); err != nil {
				t.Fatal(err)
			}
		})
	}
}
