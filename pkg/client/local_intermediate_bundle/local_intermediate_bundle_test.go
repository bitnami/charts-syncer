package intermediate_test

import (
	"os"
	"reflect"
	"sort"
	"testing"

	intermediate "github.com/bitnami-labs/charts-syncer/pkg/client/local_intermediate_bundle"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/time"
)

func TestFetch(t *testing.T) {
	c, err := intermediate.New("../../../testdata/intermediate_bundles")
	if err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	chartPath, err := c.Fetch("etcd", "4.8.0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(chartPath); err != nil {
		t.Errorf("chart package does not exist")
	}
}

func TestHas(t *testing.T) {
	c, err := intermediate.New("../../../testdata/intermediate_bundles")
	if err != nil {
		t.Fatal(err)
	}
	has, err := c.Has("etcd", "4.8.0")
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Errorf("chart not found in index")
	}
}

func TestList(t *testing.T) {
	c, err := intermediate.New("../../../testdata/intermediate_bundles")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"common", "etcd"}
	got, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected list of charts. got: %v, want: %v", got, want)
	}
}

func TestListChartVersions(t *testing.T) {
	c, err := intermediate.New("../../../testdata/intermediate_bundles")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"4.8.0"}
	got, err := c.ListChartVersions("etcd")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected list of charts. got: %v, want: %v", got, want)
	}
}

func TestGetChartDetails(t *testing.T) {
	c, err := intermediate.New("../../../testdata/intermediate_bundles")
	if err != nil {
		t.Fatal(err)
	}
	want := types.ChartDetails{
		PublishedAt: time.Now().Time,
		Digest:      "deadbuff",
	}
	got, err := c.GetChartDetails("etcd", "4.8.0")
	if err != nil {
		t.Fatal(err)
	}
	if want.Digest != got.Digest {
		t.Errorf("unexpected digest in chart. got: %v, want: %v", got, want)
	}
}

func TestUpload(t *testing.T) {
	c, err := intermediate.New("../../../testdata/intermediate_bundles")
	if err != nil {
		t.Fatal(err)
	}
	cMetadata := chart.Metadata{
		Name:    "apache",
		Version: "7.3.15",
	}
	err = c.Upload("../../../testdata/apache-7.3.15.tgz", &cMetadata)
	if err != nil {
		t.Fatal(err)
	}
	expectedChartPath := "../../../testdata/intermediate_bundles/apache-7.3.15.tar"
	if _, err := os.Stat(expectedChartPath); err != nil {
		t.Errorf("chart package does not exist after upload method")
	}
	if err := os.Remove(expectedChartPath); err != nil {
		t.Errorf("error cleaning chart path from %q after successful upload", expectedChartPath)
	}
}
