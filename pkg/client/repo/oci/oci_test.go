package oci_test

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/bitnami/charts-syncer/pkg/client/repo/oci"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	"helm.sh/helm/v3/pkg/chart"
)

var (
	ociRepo = &api.Repo{
		Kind: api.Kind_OCI,
		Auth: &api.Auth{
			Username: "user",
			Password: "password",
		},
		DisableChartsIndex: true,
	}
)

func TestFetch(t *testing.T) {
	oci.PrepareOciServer(t, ociRepo)
	c := oci.PrepareTest(t, ociRepo)

	chartMetadata := &chart.Metadata{
		Name:    "apache",
		Version: "7.3.15",
	}

	if err := c.Upload("../../../../testdata/apache-7.3.15.wrap.tgz", chartMetadata); err != nil {
		t.Fatal(err)
	}

	chartPath, err := c.Fetch("apache", "7.3.15")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(chartPath); err != nil {
		t.Errorf("chart package does not exist")
	}
	contentType, err := utils.GetFileContentType(chartPath)
	if err != nil {
		t.Fatalf("error checking contentType of %s file", chartPath)
	}
	if contentType != "application/x-gzip" {
		t.Errorf("incorrect content type, got: %s, want: %s.", contentType, "application/x-gzip")
	}
}

func TestHas(t *testing.T) {
	oci.PrepareOciServer(t, ociRepo)
	c := oci.PrepareTest(t, ociRepo)

	chartMetadata := &chart.Metadata{
		Name:    "apache",
		Version: "7.3.15",
	}
	if err := c.Upload("../../../../testdata/apache-7.3.15.wrap.tgz", chartMetadata); err != nil {
		t.Fatal(err)
	}

	has, err := c.Has(chartMetadata.Name, chartMetadata.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Errorf("chart not found in index")
	}
}

func TestList(t *testing.T) {
	oci.PrepareOciServer(t, ociRepo)
	c := oci.PrepareTest(t, ociRepo)

	want := []string{}
	got, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected list of charts names. got: %v, want: %v", got, want)
	}
}

func TestListChartVersions(t *testing.T) {
	oci.PrepareOciServer(t, ociRepo)
	c := oci.PrepareTest(t, ociRepo)
	chartMetadata := &chart.Metadata{
		Name:    "apache",
		Version: "7.3.15",
	}
	if err := c.Upload("../../../../testdata/apache-7.3.15.wrap.tgz", chartMetadata); err != nil {
		t.Fatal(err)
	}

	want := []string{"7.3.15"}
	got, err := c.ListChartVersions("apache")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected list of charts. got: %v, want: %v", got, want)
	}
}

func TestReload(t *testing.T) {
	c := oci.PrepareHttpServer(t, ociRepo)
	expectedError := "reload method is not supported yet"
	err := c.Reload()
	if err.Error() != expectedError {
		t.Errorf("unexpected error message. got: %q, want: %q", err.Error(), expectedError)
	}
}

func TestUpload(t *testing.T) {
	oci.PrepareOciServer(t, ociRepo)
	c := oci.PrepareTest(t, ociRepo)
	chartMetadata := &chart.Metadata{
		Name:    "apache",
		Version: "7.3.15",
	}
	if err := c.Upload("../../../../testdata/apache-7.3.15.wrap.tgz", chartMetadata); err != nil {
		t.Fatal(err)
	}
	chartPath, err := c.Fetch("apache", "7.3.15")
	if _, err := os.Stat(chartPath); err != nil {
		t.Errorf("chart package does not exist")
	}
	contentType, err := utils.GetFileContentType(chartPath)
	if err != nil {
		t.Fatalf("error checking contentType of %s file", chartPath)
	}
	if contentType != "application/x-gzip" {
		t.Errorf("incorrect content type, got: %s, want: %s.", contentType, "application/x-gzip")
	}
}
