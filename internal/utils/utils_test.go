package utils

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/bitnami/charts-syncer/api"
	"helm.sh/helm/v3/pkg/chart"
	helmRepo "helm.sh/helm/v3/pkg/repo"
)

var (
	source = &api.Source{
		Repo: &api.Repo{
			Url: "https://charts.bitnami.com/bitnami",
		},
	}
)

func TestLoadIndexFromRepo(t *testing.T) {
	// Load index.yaml info into index object
	sourceIndex, err := LoadIndexFromRepo(source.GetRepo())
	if err != nil {
		t.Fatalf("error loading index.yaml: %v", err)
	}
	if sourceIndex.Entries["apache"] == nil {
		t.Errorf("apache chart not found")
	}
}

func TestChartExistInIndex(t *testing.T) {
	sampleIndexFile := "../../testdata/index.yaml"
	index, err := helmRepo.LoadIndexFile(sampleIndexFile)
	if err != nil {
		t.Fatalf("error loading index.yaml: %v ", err)
	}
	versionExist := ChartExistInIndex("etcd", "4.7.4", index)
	versionNotExist := ChartExistInIndex("etcd", "0.0.44", index)
	if versionExist != true {
		t.Errorf("version should exist but is not found")
	}
	if versionNotExist != false {
		t.Errorf("version should not exist but it is reported as found")
	}
}

func TestDownloadIndex(t *testing.T) {
	indexFile, err := downloadIndex(source.GetRepo())
	if err != nil {
		t.Fatalf("error downloading index.yaml: %v ", err)
	}
	defer os.Remove(indexFile)

	if _, err := os.Stat(indexFile); err != nil {
		t.Errorf("index file does not exist.")
	}
}

func TestUntar(t *testing.T) {
	filepath := "../../testdata/apache-7.3.15.tgz"
	// Create temporary working directory
	testTmpDir, err := os.MkdirTemp("", "charts-syncer-tests")
	if err != nil {
		t.Fatalf("error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)
	if err := Untar(filepath, testTmpDir); err != nil {
		t.Fatal(err)
	}
	tarFiles := []string{
		"apache/Chart.yaml",
		"apache/values.yaml",
		"apache/templates/NOTES.txt",
		"apache/templates/_helpers.tpl",
		"apache/templates/configmap-vhosts.yaml",
		"apache/templates/configmap.yaml",
		"apache/templates/deployment.yaml",
		"apache/templates/ingress.yaml",
		"apache/templates/svc.yaml",
		"apache/.helmignore",
		"apache/README.md",
		"apache/files/README.md",
		"apache/files/vhosts/README.md",
		"apache/values.schema.json",
	}
	for _, f := range tarFiles {
		if _, err := os.Stat(path.Join(testTmpDir, f)); err != nil {
			t.Errorf("error untaring chart package. %q not found", f)
		}
	}
}

func TestGetFileContentType(t *testing.T) {
	filepath := "../../testdata/apache-7.3.15.tgz"
	contentType, err := GetFileContentType(filepath)
	if err != nil {
		t.Fatalf("error checking contentType of %s file", filepath)
	}
	if contentType != "application/x-gzip" {
		t.Errorf("incorrect content type, got: %s, want: %s.", contentType, "application/x-gzip")
	}
}

func TestGetDateThreshold(t *testing.T) {
	date := time.Date(2020, 05, 15, 0, 0, 0, 0, time.UTC)
	fromDate := "2020-05-15"
	dateThreshold, err := GetDateThreshold(fromDate)
	if err != nil {
		t.Fatal(err)
	}
	if dateThreshold != date {
		t.Errorf("incorrect dateThreshold, expected: %v, got %v", date, dateThreshold)
	}
}

func TestGetDownloadURL(t *testing.T) {
	sourceRepoURL := "https://repo-url.com/charts"
	sourceIndex := helmRepo.NewIndexFile()
	sourceIndex.Add(&chart.Metadata{Name: "apache", Version: "7.3.15"}, "apache-7.3.15.tgz", sourceRepoURL, "sha256:1234567890")
	downloadURL, err := FindChartURL("apache", "7.3.15", sourceIndex, sourceRepoURL)
	if err != nil {
		t.Fatal(err)
	}
	expectedDownloadURL := "https://repo-url.com/charts/apache-7.3.15.tgz"
	if downloadURL != expectedDownloadURL {
		t.Errorf("wrong download URL, got: %s , want: %s", downloadURL, expectedDownloadURL)
	}
	expectedError := "unable to find chart url in index"
	_, err = FindChartURL("apache", "0.0.333", sourceIndex, sourceRepoURL)
	if err.Error() != expectedError {
		t.Errorf("wrong error message, got: %s , want: %s", err.Error(), expectedError)
	}
}

func TestFindChartByVersion(t *testing.T) {
	sourceIndex := helmRepo.NewIndexFile()
	sourceIndex.Add(&chart.Metadata{Name: "apache", Version: "7.3.15"}, "apache-7.3.15.tgz", "https://repo-url.com/charts", "sha256:1234567890")
	chart := findChartByVersion(sourceIndex.Entries["apache"], "7.3.15")
	if chart.Name != "apache" {
		t.Errorf("wrong chart, got: %s , want: %s", chart.Name, "apache")
	}
	if chart.Version != "7.3.15" {
		t.Errorf("wrong chart version, got: %s , want: %s", chart.Version, "7.3.15")
	}
}

func TestIsValidURL(t *testing.T) {
	validURL := "https://chart.repo.com/charts/zookeeper-1.0.0.tgz"
	invalidURL := "charts/zookeeper-1.0.0.tgz"
	if res := isValidURL(validURL); res != true {
		t.Errorf("got: %t , want: %t", res, true)
	}
	if res := isValidURL(invalidURL); res != false {
		t.Errorf("got: %t , want: %t", res, true)
	}
}

func TestNormalizeChartURL(t *testing.T) {
	tests := []struct {
		desc          string
		repoURL       string
		chartURL      string
		want          string
		shouldFail    bool
		expectedError error
	}{
		{
			desc:     "should use repo URL when chart uses relative URL",
			repoURL:  "https://github.com/kubernetes",
			chartURL: "charts/nats-1.2.3.tgz",
			want:     "https://github.com/kubernetes/charts/nats-1.2.3.tgz",
		},
		{
			desc:     "should return chart URL when chart uses absolute URL",
			repoURL:  "https://kubernetes.github.io",
			chartURL: "https://github.com/kubernetes/nats-1.2.3.tgz",
			want:     "https://github.com/kubernetes/nats-1.2.3.tgz",
		},
		{
			desc:          "should return error if the repository URL is empty and the chart URL is relative",
			chartURL:      "charts/nats-1.2.3.tgz",
			shouldFail:    true,
			expectedError: fmt.Errorf("repository URL cannot be empty"),
		},
		{
			desc:          "should return error if the chart URL is empty",
			repoURL:       "https://kubernetes.github.io",
			shouldFail:    true,
			expectedError: fmt.Errorf("chart URL cannot be empty"),
		},
		{
			desc:          "should return error if the repository URL is malformed and chart URL is relative",
			repoURL:       "://kubernetes.github.io",
			chartURL:      "charts/nats-1.2.3.tgz",
			shouldFail:    true,
			expectedError: fmt.Errorf(`parse "://kubernetes.github.io": missing protocol scheme`),
		},
		{
			desc:          "should return error if the chart URL is malformed",
			repoURL:       "https://kubernetes.github.io",
			chartURL:      "://github.com/kubernetes/nats-1.2.3.tgz",
			shouldFail:    true,
			expectedError: fmt.Errorf(`parse "://github.com/kubernetes/nats-1.2.3.tgz": missing protocol scheme`),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := NormalizeChartURL(tc.repoURL, tc.chartURL)
			if tc.shouldFail {
				if err == nil {
					t.Errorf("expected %+v but found %+v", tc.expectedError, err)
					return
				}
				if err.Error() != tc.expectedError.Error() {
					t.Errorf("\n got: %v\nwant: %v", err, tc.expectedError)
					return
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("wrong download URL. got: %v, want: %v", got, tc.want)
			}
		})
	}
}
