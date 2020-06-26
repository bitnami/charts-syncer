package repo

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
)

var (
	sourceHelm = &api.SourceRepo{
		Repo: &api.Repo{
			Url:  "https://charts.bitnami.com/bitnami",
			Kind: api.Kind_HELM,
		},
	}
	targetHelm = &api.SourceRepo{
		Repo: &api.Repo{
			Url:  "https://fake.repo.com",
			Kind: api.Kind_HELM,
		},
	}
)

func TestDownloadFromHelmClassic(t *testing.T) {
	// Create temporary working directory
	testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
	if err != nil {
		t.Errorf("Error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)
	chartPath := path.Join(testTmpDir, "nginx-5.3.1.tgz")
	// Create client for source repo
	sc, err := NewClient(sourceHelm.Repo)
	if err != nil {
		t.Fatal("Could not create a client for the source repo", err)
	}
	sourceIndex, err := utils.LoadIndexFromRepo(sourceHelm.Repo)
	if err != nil {
		t.Errorf("Error loading index.yaml: %w", err)
	}
	if err := sc.DownloadChart(chartPath, "nginx", "5.3.1", sourceHelm.Repo, sourceIndex); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(chartPath); err != nil {
		t.Errorf("Expected %s to exists", chartPath)
	}
}

func TestChartExistsInHelmClassic(t *testing.T) {
	// Create client for source repo
	sc, err := NewClient(sourceHelm.Repo)
	if err != nil {
		t.Fatal("could not create a client for the source repo", err)
	}
	chartExists, err := sc.ChartExists("nginx", "5.3.1", sourceHelm.Repo)
	if err != nil {
		t.Fatal(err)
	}
	if !chartExists {
		t.Errorf("nginx-5.3.1 chart should exists")
	}
}

func TestPublishToHelmClassic(t *testing.T) {
	// Create client for target repo
	tc, err := NewClient(targetHelm.Repo)
	if err != nil {
		t.Fatal("could not create a client for the target repo", err)
	}
	chartPath := "../../testdata/apache-7.3.15.tgz"
	err = tc.PublishChart(chartPath, targetHelm.Repo)
	expectedErrorMsg := "Publishing to a Helm classic repository is not supported yet"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Incorrect error, got: \n %s \n, want: \n %s \n", err.Error(), expectedErrorMsg)
	}
}
