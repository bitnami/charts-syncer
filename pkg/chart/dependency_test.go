package chart

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/utils"
	"gopkg.in/yaml.v2"
)

var (
	source = &api.SourceRepo{
		Repo: &api.Repo{
			Url:  "https://charts.bitnami.com/bitnami",
			Kind: "HELM",
		},
	}
	target = &api.TargetRepo{
		Repo: &api.Repo{
			Url:  "http://fake.target/com",
			Kind: "CHARTMUSEUM",
			Auth: &api.Auth{
				Username: "user",
				Password: "password",
			},
		},
		ContainerRegistry:   "test.registry.io",
		ContainerRepository: "test/repo",
	}
)

func TestSyncDependencies(t *testing.T) {
	// Create temporary working directory
	testTmpDir, err := ioutil.TempDir("", "c3tsyncer-tests")
	defer os.RemoveAll(testTmpDir)
	if err != nil {
		t.Errorf("Error creating temporary: %s", testTmpDir)
	}
	// Extract test chart to working dir
	sourceChart := "../../testdata/kafka-10.3.3.tgz"
	if err := utils.Untar(sourceChart, testTmpDir); err != nil {
		t.Fatal(err)
	}

	chartPath := path.Join(testTmpDir, "kafka")
	// Call manage depedencies
	err = syncDependencies(chartPath, source.Repo, target, false)
	expectedError := "Please sync zookeeper-5.14.3 dependency first"
	if err != nil && err.Error() != expectedError {
		t.Errorf("Incorrect error, got: \n %s \n, want: \n %s \n", err.Error(), expectedError)
	}
}

func TestUpdateRequirementsFile(t *testing.T) {
	chartDependencies := map[string]string{
		"zookeeper": "5.5.5",
	}
	// Create temporary working directory
	testTmpDir, err := ioutil.TempDir("", "c3tsyncer-tests")
	defer os.RemoveAll(testTmpDir)
	if err != nil {
		t.Errorf("Error creating temporary: %s", testTmpDir)
	}
	// Extract test chart to working dir
	sourceChart := "../../testdata/kafka-10.3.3.tgz"
	if err := utils.Untar(sourceChart, testTmpDir); err != nil {
		t.Fatal(err)
	}

	chartPath := path.Join(testTmpDir, "kafka")
	requirementsFile := path.Join(chartPath, "requirements.yaml")
	// Update file
	if err := updateRequirementsFile(chartPath, chartDependencies, source.Repo, target); err != nil {
		t.Fatal(err)
	}

	// Read new deps file
	requirements, err := ioutil.ReadFile(requirementsFile)
	if err != nil {
		t.Errorf("Error reading updated %s file", requirementsFile)
	}

	// Unmarshall file to new object
	newDeps := &dependencies{}
	err = yaml.Unmarshal(requirements, newDeps)
	if err != nil {
		t.Errorf("Error unmarshaling %s file", requirementsFile)
	}

	// Check properties
	if newDeps.Dependencies[0].Repository != target.Repo.Url {
		t.Errorf("Incorrect modification, got: \n %s \n, want: \n %s \n", newDeps.Dependencies[0].Repository, target.Repo.Url)
	}
	if newDeps.Dependencies[0].Version != "5.5.5" {
		t.Errorf("Incorrect modification, got: \n %s \n, want: \n %s \n", newDeps.Dependencies[0].Version, "5.5.5")
	}
}

func TestWriteRequirementsFile(t *testing.T) {
	target := &api.TargetRepo{
		Repo: &api.Repo{
			Url:  "http://fake.target/com",
			Kind: "CHARTMUSEUM",
			Auth: &api.Auth{
				Username: "user",
				Password: "password",
			},
		},
		ContainerRegistry:   "test.registry.io",
		ContainerRepository: "test/repo",
	}
	// Create temporary working directory
	testTmpDir, err := ioutil.TempDir("", "c3tsyncer-tests")
	defer os.RemoveAll(testTmpDir)
	if err != nil {
		t.Errorf("Error creating temporary: %s", testTmpDir)
	}
	// Extract test chart to working dir
	sourceChart := "../../testdata/kafka-10.3.3.tgz"
	if err := utils.Untar(sourceChart, testTmpDir); err != nil {
		t.Fatal(err)
	}

	chartPath := path.Join(testTmpDir, "kafka")
	// Read current dependencies file
	requirementsFile := path.Join(chartPath, "requirements.yaml")
	requirements, err := ioutil.ReadFile(requirementsFile)
	if err != nil {
		t.Errorf("Error reading %s file", requirementsFile)
	}

	// Unmarshall to struct
	deps := &dependencies{}
	err = yaml.Unmarshal(requirements, deps)
	if err != nil {
		t.Errorf("Error unmarshaling %s file", requirementsFile)
	}

	// Edit dependencies object
	deps.Dependencies[0].Repository = target.Repo.Url

	// Write new requirements file
	if err := writeRequirementsFile(chartPath, deps); err != nil {
		t.Fatal(err)
	}

	// Read new deps file
	requirements, err = ioutil.ReadFile(requirementsFile)
	if err != nil {
		t.Errorf("Error reading updated %s file", requirementsFile)
	}

	// Unmarshall file to new object
	newDeps := &dependencies{}
	err = yaml.Unmarshal(requirements, newDeps)
	if err != nil {
		t.Errorf("Error unmarshaling %s file", requirementsFile)
	}

	// Check properties
	if newDeps.Dependencies[0].Repository != target.Repo.Url {
		t.Errorf("Incorrect modification, got: \n %s \n, want: \n %s \n", newDeps.Dependencies[0].Repository, target.Repo.Url)
	}
	if newDeps.Dependencies[0].Version != "5.x.x" {
		t.Errorf("Incorrect modification, got: \n %s \n, want: \n %s \n", newDeps.Dependencies[0].Version, "5.x.x")
	}
}
