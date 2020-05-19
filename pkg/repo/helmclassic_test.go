package repo

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/bitnami-labs/chart-repository-syncer/api"
)

func TestDownloadFromHelmClassic(t *testing.T) {
	// Define source repo
	source := &api.SourceRepo{
		Repo: &api.Repo{
			Url:  "https://charts.bitnami.com/bitnami",
			Kind: "HELM",
		},
	}
	// Create temporary working directory
	testTmpDir, err := ioutil.TempDir("", "c3tsyncer-tests")
	defer os.RemoveAll(testTmpDir)
	if err != nil {
		t.Errorf("Error creating temporary: %s", testTmpDir)
	}
	chartPath := path.Join(testTmpDir, "nginx-5.3.1.tgz")
	// Create client for source repo
	sc, err := NewClient(source.Repo)
	if err != nil {
		t.Fatal("Could not create a client for the source repo", err)
	}
	if err := sc.DownloadChart(chartPath, "nginx", "5.3.1", source.Repo); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(chartPath); err != nil {
		t.Errorf("Expected %s to exists", chartPath)
	}
}

func TestChartExistsInHelmClassic(t *testing.T) {
	// Define source repo
	source := &api.SourceRepo{
		Repo: &api.Repo{
			Url:  "https://charts.bitnami.com/bitnami",
			Kind: "HELM",
		},
	}
	// Create client for source repo
	sc, err := NewClient(source.Repo)
	if err != nil {
		t.Fatal("could not create a client for the source repo", err)
	}
	chartExists, err := sc.ChartExists("nginx", "5.3.1", source.Repo)
	if err != nil {
		t.Fatal(err)
	}
	if !chartExists {
		t.Errorf("nginx-5.3.1 chart should exists")
	}
}

func TestPublishToHelmClassic(t *testing.T) {
	// Define source repo
	target := &api.SourceRepo{
		Repo: &api.Repo{
			Url:  "https://fake.repo.com",
			Kind: "HELM",
		},
	}
	// Create client for source repo
	tc, err := NewClient(target.Repo)
	if err != nil {
		t.Fatal("could not create a client for the target repo", err)
	}
	chartPath := "../../testdata/apache-7.3.15.tgz"
	err = tc.PublishChart(chartPath, target.Repo)
	expectedErrorMsg := "Publishing to a Helm classic repository is not supported yet"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Incorrect error, got: \n %s \n, want: \n %s \n", err.Error(), expectedErrorMsg)
	}
}
