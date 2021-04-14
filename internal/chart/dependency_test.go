package chart

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/yaml"
)

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
			APIV1,
			"/tmp/kafka/requirements.lock",
			false,
			nil,
		},
		"api v2 chart": {
			"/tmp/kafka",
			APIV2,
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
				if err.Error() != tc.expectedError.Error() {
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

func TestUpdateRequirementsFile(t *testing.T) {
	lock := &chart.Lock{
		Generated: time.Now(),
		Digest:    "sha256:fe26de7fc873dc8001404168feb920a61ba884a2fe211a7371165ed51bf8cb8b",
		Dependencies: []*chart.Dependency{
			{Name: "zookeeper", Version: "5.5.5", Repository: source.GetRepo().GetUrl()},
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
	requirementsFile := path.Join(chartPath, RequirementsFilename)
	if err := updateRequirementsFile(chartPath, lock, source.GetRepo(), target.GetRepo()); err != nil {
		t.Fatal(err)
	}

	// Test requirements file
	requirements, err := ioutil.ReadFile(requirementsFile)
	if err != nil {
		t.Fatalf("error reading updated %s file", requirementsFile)
	}
	newDeps := &dependencies{}
	err = yaml.Unmarshal(requirements, newDeps)
	if err != nil {
		t.Fatalf("error unmarshaling %s file", requirementsFile)
	}
	want := target.GetRepo().GetUrl()
	if got := newDeps.Dependencies[0].Repository; got != want {
		t.Errorf("incorrect modification, got: %s, want: %s", got, want)
	}
	want = "5.5.5"
	if got := newDeps.Dependencies[0].Version; got != want {
		t.Errorf("incorrect modification, got: %s, want: %s", got, want)
	}

	// Test requirements lock file
	requirementsFileLock := path.Join(chartPath, RequirementsLockFilename)
	requirementsLock, err := ioutil.ReadFile(requirementsFileLock)
	if err != nil {
		t.Fatalf("error reading updated %s file", requirementsFileLock)
	}
	newLock := &chart.Lock{}
	err = yaml.Unmarshal(requirementsLock, newLock)
	if err != nil {
		t.Fatalf("error unmarshaling %s file", requirementsFileLock)
	}
	want = target.GetRepo().GetUrl()
	if got := newLock.Dependencies[0].Repository; got != want {
		t.Errorf("incorrect modification, got: %s, want: %s", got, want)
	}
	want = "5.5.5"
	if got := newLock.Dependencies[0].Version; got != want {
		t.Errorf("incorrect modification, got: %s, want: %s", got, want)
	}
}

func TestUpdateChartMetadataFile(t *testing.T) {
	lock := &chart.Lock{
		Generated: time.Now(),
		Digest:    "sha256:fe26de7fc873dc8001404168feb920a61ba884a2fe211a7371165ed51bf8cb8b",
		Dependencies: []*chart.Dependency{
			{Name: "zookeeper", Version: "5.19.1", Repository: source.GetRepo().GetUrl()},
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
	chartFile := path.Join(chartPath, ChartFilename)
	err = os.MkdirAll(chartPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(chartFile, sourceFile, 0644)
	if err != nil {
		t.Fatal(err)
	}

	if err := updateChartMetadataFile(chartPath, lock, source.GetRepo(), target.GetRepo()); err != nil {
		t.Fatal(err)
	}

	// Test Chart.yaml file
	chartFileContent, err := ioutil.ReadFile(chartFile)
	if err != nil {
		t.Fatalf("error reading updated %s file", chartFile)
	}
	chartMetadata := &chart.Metadata{}
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

	// Test Chart.lock file
	chartFileLock := path.Join(chartPath, ChartLockFilename)
	chartLock, err := ioutil.ReadFile(chartFileLock)
	if err != nil {
		t.Fatalf("error reading updated %s file", chartFileLock)
	}
	newLock := &chart.Lock{}
	err = yaml.Unmarshal(chartLock, newLock)
	if err != nil {
		t.Fatalf("error unmarshaling %s file", chartFileLock)
	}
	want = target.GetRepo().GetUrl()
	if got := newLock.Dependencies[0].Repository; got != want {
		t.Errorf("incorrect modification, got: %s, want: %s", got, want)
	}
	want = "5.19.1"
	if got := newLock.Dependencies[0].Version; got != want {
		t.Errorf("incorrect modification, got: %s, want: %s", got, want)
	}
}

func TestFindDepByName(t *testing.T) {
	deps := &dependencies{
		Dependencies: []*chart.Dependency{
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
