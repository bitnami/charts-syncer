package chart

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/yaml"
)

func newChartPath(t *testing.T, file string, name string) string {
	t.Helper()

	testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
	if err != nil {
		t.Fatalf("error creating temporary: %s", testTmpDir)
	}
	t.Cleanup(func() { os.RemoveAll(testTmpDir) })

	if err := utils.Untar(file, testTmpDir); err != nil {
		t.Fatal(err)
	}

	return path.Join(testTmpDir, name)
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

	chartPath := newChartPath(t, "../../testdata/kafka-10.3.3.tgz", "kafka")
	requirementsFile := path.Join(chartPath, RequirementsFilename)

	var ignoreTrusted, syncTrusted []*api.Repo

	if err := updateRequirementsFile(chartPath, lock, source.GetRepo(), target.GetRepo(), syncTrusted, ignoreTrusted); err != nil {
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
	want = "5.x.x"
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

	var ignoreTrusted, syncTrusted []*api.Repo

	if err := updateChartMetadataFile(chartPath, lock, source.GetRepo(), target.GetRepo(), syncTrusted, ignoreTrusted); err != nil {
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
	want := target.GetRepo().GetUrl()
	if got := chartMetadata.Dependencies[0].Repository; got != want {
		t.Errorf("incorrect modification, got: %s, want: %s", got, want)
	}
	want = "5.x.x"
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

func TestGetDependencyRepoURL(t *testing.T) {
	tests := map[string]struct {
		targetRepo *api.Repo
		want       string
	}{
		"not oci repo": {
			&api.Repo{
				Url:  "https://harbor.endpoint.io/chartrepo/library",
				Kind: api.Kind_HARBOR,
			},
			"https://harbor.endpoint.io/chartrepo/library",
		},
		"oci repo": {
			&api.Repo{
				Url:  "https://harbor.endpoint.io/my-project/my-charts-library",
				Kind: api.Kind_OCI,
			},
			"oci://harbor.endpoint.io/my-project/my-charts-library",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := getDependencyRepoURL(tc.targetRepo)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("got: %q, want %q", got, tc.want)
			}
		})
	}
}
