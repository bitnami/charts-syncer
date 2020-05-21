package utils

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	helmRepo "helm.sh/helm/v3/pkg/repo"
)

var (
	source = &api.SourceRepo{
		Repo: &api.Repo{
			Url: "https://charts.bitnami.com/bitnami",
		},
	}
)

func TestLoadIndexFromRepo(t *testing.T) {
	// Load index.yaml info into index object
	sourceIndex, err := LoadIndexFromRepo(source.Repo)
	if err != nil {
		t.Errorf("Error loading index.yaml: %w", err)
	}
	if sourceIndex.Entries["apache"] == nil {
		t.Errorf("Apache chart not found")
	}
}

func TestChartExistInIndex(t *testing.T) {
	sampleIndexFile := "../../testdata/index.yaml"
	index, err := helmRepo.LoadIndexFile(sampleIndexFile)
	if err != nil {
		t.Errorf("Error loading index.yaml: %v ", err)
	}
	versionExist, err := ChartExistInIndex("etcd", "4.7.4", index)
	versionNotExist, err := ChartExistInIndex("etcd", "0.0.44", index)
	if versionExist != true {
		t.Errorf("Version should exist but is not found")
	}
	if versionNotExist != false {
		t.Errorf("Version should not exist but it is reported as found")
	}
}

func TestDownloadIndex(t *testing.T) {
	indexFile, err := downloadIndex(source.Repo)
	defer os.Remove(indexFile)
	if err != nil {
		t.Errorf("Error downloading index.yaml: %v ", err)
	}

	if _, err := os.Stat(indexFile); err != nil {
		t.Errorf("Index file does not exist.")
	}
}

func TestUntar(t *testing.T) {
	filepath := "../../testdata/apache-7.3.15.tgz"
	// Create temporary working directory
	testTmpDir, err := ioutil.TempDir("", "c3tsyncer-tests")
	defer os.RemoveAll(testTmpDir)
	if err != nil {
		t.Errorf("Error creating temporary: %s", testTmpDir)
	}
	if err := Untar(filepath, testTmpDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path.Join(testTmpDir, "apache/Chart.yaml")); err != nil {
		t.Errorf("Error untaring chart package")
	}
}

func TestGetFileContentType(t *testing.T) {
	filepath := "../../testdata/apache-7.3.15.tgz"
	contentType, err := GetFileContentType(filepath)
	if err != nil {
		t.Errorf("Error checking contentType of %s file", filepath)
	}
	if contentType != "application/x-gzip" {
		t.Errorf("Incorrect content type, got: %s, want: %s.", contentType, "application/x-gzip")
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
		t.Errorf("Incorrect dateThreshold, expected: %v, got %v", date, dateThreshold)
	}
}
