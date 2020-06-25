package chart

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/repo"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/utils"
)

var (
	source = &api.SourceRepo{
		Repo: &api.Repo{
			Url:  "https://charts.bitnami.com/bitnami",
			Kind: api.Kind_HELM,
		},
	}
	target = &api.TargetRepo{
		Repo: &api.Repo{
			Url:  "http://fake.target/com",
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

func TestDownload(t *testing.T) {
	// Create temporary working directory
	testTmpDir, err := ioutil.TempDir("", "c3tsyncer-tests")
	if err != nil {
		t.Errorf("Error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)
	chartPath := path.Join(testTmpDir, "nginx-5.3.1.tgz")
	// Create client for source repo
	sc, err := repo.NewClient(source.Repo)
	if err != nil {
		t.Fatal("could not create a client for the source repo", err)
	}
	sourceIndex, err := utils.LoadIndexFromRepo(source.Repo)
	if err != nil {
		t.Errorf("Error loading index.yaml: %w", err)
	}
	if err := sc.DownloadChart(chartPath, "nginx", "5.3.1", source.Repo, sourceIndex); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(chartPath); err != nil {
		t.Errorf("Chart package does not exist")
	}
}

func TestUpdateValuesFile(t *testing.T) {
	originalValues := `##
## Simplified values yaml file to test registry and repository substitutions
##
global:
  imageRegistry: ""
image:
  registry: new.registry.io
  repository: test/repo/new/zookeeper
  tag: 3.5.7-r7
volumePermissions:
  enabled: false
  image:
    registry: new.registry.io
    repository: repo/new/custom-base-image
    tag: r0`
	want := `##
## Simplified values yaml file to test registry and repository substitutions
##
global:
  imageRegistry: ""
image:
  registry: test.registry.io
  repository: test/repo/zookeeper
  tag: 3.5.7-r7
volumePermissions:
  enabled: false
  image:
    registry: test.registry.io
    repository: test/repo/custom-base-image
    tag: r0`
	// Create temporary working directory
	testTmpDir, err := ioutil.TempDir("", "c3tsyncer-tests")
	if err != nil {
		t.Errorf("Error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)
	destValuesFilePath := path.Join(testTmpDir, "values.yaml")

	// Write file
	err = ioutil.WriteFile(destValuesFilePath, []byte(originalValues), 0644)
	if err != nil {
		t.Errorf("Error writting destination file")
	}

	updateValuesFile(destValuesFilePath, target)
	valuesFile, err := ioutil.ReadFile(destValuesFilePath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(valuesFile)
	if want != got {
		t.Errorf("Incorrect modification, got: \n %s \n, want: \n %s \n", got, want)
	}
}
