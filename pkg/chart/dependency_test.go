package chart

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	helmChart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

func TestSyncDependencies(t *testing.T) {
	testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
	if err != nil {
		t.Fatalf("error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)

	sourceChart := "../../testdata/kafka-10.3.3.tgz"
	if err := utils.Untar(sourceChart, testTmpDir); err != nil {
		t.Fatal(err)
	}

	sourceIndex, err := utils.LoadIndexFromRepo(source.Repo)
	if err != nil {
		t.Fatalf("error loading index.yaml: %v", err)
	}
	targetIndex := repo.NewIndexFile()

	chartPath := path.Join(testTmpDir, "kafka")
	err = syncDependencies(chartPath, source.Repo, target, sourceIndex, targetIndex, "v1", false)
	expectedError := "please sync zookeeper-5.14.3 dependency first"
	if err != nil && err.Error() != expectedError {
		t.Errorf("incorrect error, got: \n %s \n, want: \n %s \n", err.Error(), expectedError)
	}
}

func TestUpdateRequirementsFile(t *testing.T) {
	lock := &helmChart.Lock{
		Generated: time.Now(),
		Digest:    "sha256:fe26de7fc873dc8001404168feb920a61ba884a2fe211a7371165ed51bf8cb8b",
		Dependencies: []*helmChart.Dependency{
			{Name: "zookeeper", Version: "5.5.5"},
		},
	}

	testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
	if err != nil {
		t.Fatalf("error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)

	sourceChart := "../../testdata/kafka-10.3.3.tgz"
	if err := utils.Untar(sourceChart, testTmpDir); err != nil {
		t.Fatal(err)
	}

	chartPath := path.Join(testTmpDir, "kafka")
	requirementsFile := path.Join(chartPath, "requirements.yaml")
	if err := updateRequirementsFile(chartPath, lock, source.Repo, target); err != nil {
		t.Fatal(err)
	}

	requirements, err := ioutil.ReadFile(requirementsFile)
	if err != nil {
		t.Fatalf("error reading updated %s file", requirementsFile)
	}

	newDeps := &dependencies{}
	err = yaml.Unmarshal(requirements, newDeps)
	if err != nil {
		t.Fatalf("error unmarshaling %s file", requirementsFile)
	}
	want := target.Repo.Url
	if got := newDeps.Dependencies[0].Repository; got != want {
		t.Errorf("incorrect modification, got: %s, want: %s", got, want)
	}
	want = "5.5.5"
	if got := newDeps.Dependencies[0].Version; got != want {
		t.Errorf("incorrect modification, got: %s, want: %s", got, want)
	}
}

func TestUpdateChartMetadataFile(t *testing.T) {
	lock := &helmChart.Lock{
		Generated: time.Now(),
		Digest:    "sha256:fe26de7fc873dc8001404168feb920a61ba884a2fe211a7371165ed51bf8cb8b",
		Dependencies: []*helmChart.Dependency{
			{Name: "zookeeper", Version: "5.19.1"},
		},
	}

	testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
	if err != nil {
		t.Fatalf("error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)

	sourceFile, err := ioutil.ReadFile("../../testdata/kafka-chart.yaml")
	if err != nil {
		t.Fatal(err)
	}
	chartPath := path.Join(testTmpDir, "kafka")
	chartFile := path.Join(chartPath, "Chart.yaml")
	err = os.MkdirAll(chartPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(chartFile, sourceFile, 0644)
	if err != nil {
		t.Fatal(err)
	}

	if err := updateChartMetadataFile(chartPath, lock, source.Repo, target); err != nil {
		t.Fatal(err)
	}

	chartFileContent, err := ioutil.ReadFile(chartFile)
	if err != nil {
		t.Fatalf("error reading updated %s file", chartFile)
	}

	chartMetadata := &helmChart.Metadata{}
	err = yaml.Unmarshal(chartFileContent, chartMetadata)
	if err != nil {
		t.Fatalf("error unmarshaling %s file", chartFile)
	}

	want := target.Repo.Url
	if got := chartMetadata.Dependencies[0].Repository; got != want {
		t.Errorf("incorrect modification, got: %s, want: %s", got, want)
	}
	want = "5.19.1"
	if got := chartMetadata.Dependencies[0].Version; got != want {
		t.Errorf("incorrect modification, got: %s, want: %s", got, want)
	}
}

func TestWriteRequirementsFile(t *testing.T) {
	target := &api.TargetRepo{
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

	testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
	if err != nil {
		t.Fatalf("error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)

	sourceChart := "../../testdata/kafka-10.3.3.tgz"
	if err := utils.Untar(sourceChart, testTmpDir); err != nil {
		t.Fatal(err)
	}

	chartPath := path.Join(testTmpDir, "kafka")
	requirementsFile := path.Join(chartPath, "requirements.yaml")
	requirements, err := ioutil.ReadFile(requirementsFile)
	if err != nil {
		t.Fatalf("error reading %s file", requirementsFile)
	}

	deps := &dependencies{}
	err = yaml.Unmarshal(requirements, deps)
	if err != nil {
		t.Fatalf("error unmarshaling %s file", requirementsFile)
	}

	deps.Dependencies[0].Repository = target.Repo.Url

	if err := writeRequirementsFile(chartPath, deps); err != nil {
		t.Fatal(err)
	}

	requirements, err = ioutil.ReadFile(requirementsFile)
	if err != nil {
		t.Fatalf("error reading updated %s file", requirementsFile)
	}

	newDeps := &dependencies{}
	err = yaml.Unmarshal(requirements, newDeps)
	if err != nil {
		t.Fatalf("error unmarshaling %s file", requirementsFile)
	}

	if newDeps.Dependencies[0].Repository != target.Repo.Url {
		t.Errorf("incorrect modification, got: %s, want: %s", newDeps.Dependencies[0].Repository, target.Repo.Url)
	}
	if newDeps.Dependencies[0].Version != "5.x.x" {
		t.Errorf("incorrect modification, got: %s, want: %s", newDeps.Dependencies[0].Version, "5.x.x")
	}
}

func TestFindDepByName(t *testing.T) {
	deps := &dependencies{
		Dependencies: []*helmChart.Dependency{
			{Name: "mariadb", Version: "1.2.3"},
			{Name: "postgresql", Version: "4.5.6"},
		},
	}
	dep := findDepByName(deps.Dependencies, "postgresql")
	if dep.Name != "postgresql" {
		t.Errorf("wrong dependency, got: %s , want: %s", dep.Name, "postgresql")
	}
	if dep.Version != "4.5.6" {
		t.Errorf("wrong dependency, got: %s , want: %s", dep.Version, "4.5.6")
	}
}

func TestLockFilePath(t *testing.T) {
	tests := map[string]struct {
		chartPath     string
		apiVersion    string
		expectedPath  string
		shouldFail    bool
		expectedError error
	}{
		"api v1 chart": {
			"/tmp/kafka",
			"v1",
			"/tmp/kafka/requirements.lock",
			false,
			nil,
		},
		"api v2 chart": {
			"/tmp/kafka",
			"v2",
			"/tmp/kafka/Chart.lock",
			false,
			nil,
		},
		"unexisting api chart": {
			"/tmp/kafka",
			"vvv000",
			"",
			true,
			errors.New("unrecognised apiVersion \"vvv000\""),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			want := tc.expectedPath
			got, err := lockFilePath(tc.chartPath, tc.apiVersion)
			if tc.shouldFail {
				if !assert.EqualError(t, tc.expectedError, err.Error()) {
					t.Errorf("error does not match: [%v:%v]", tc.expectedError, err)
				}
			} else {
				if got != want {
					t.Errorf("got: %q, want %q", got, want)
				}
			}
		})
	}
}
