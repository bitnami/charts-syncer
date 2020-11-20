package core

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
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
		t.Fatalf("error creating temporary: %s", testTmpDir)
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
		t.Fatalf("error loading index.yaml: %v", err)
	}
	if err := sc.Fetch(chartPath, "nginx", "5.3.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(chartPath); err != nil {
		t.Errorf("expected %s to exists", chartPath)
	}
}

func TestChartExistsInHelmClassic(t *testing.T) {
	sourceIndex := repo.NewIndexFile()
	sourceIndex.Add(&chart.Metadata{Name: "nginx", Version: "5.3.1"}, "nginx-5.3.1.tgz", "https://fake-url.com/charts", "sha256:1234567890")
	// Create client for source repo
	sc, err := NewClient(sourceHelm.Repo)
	if err != nil {
		t.Fatal("could not create a client for the source repo", err)
	}
	chartExists, err := sc.ChartExists("nginx", "5.3.1")
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
	err = tc.Push(chartPath)
	expectedErrorMsg := "publishing to a Helm classic repository is not supported yet"
	if err.Error() != expectedErrorMsg {
		t.Errorf("incorrect error, got: \n %s \n, want: \n %s \n", err.Error(), expectedErrorMsg)
	}
}
