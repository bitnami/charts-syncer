package local_test

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/bitnami/charts-syncer/pkg/client/repo/local"
	"github.com/bitnami/charts-syncer/pkg/client/types"
	"helm.sh/helm/v3/pkg/time"
)

func TestFetch(t *testing.T) {
	c, err := local.New("../../../../testdata/wraps")
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
	c, err := local.New("../../../../testdata/wraps")
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
	c, err := local.New("../../../../testdata/wraps")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"common", "etcd", "kafka", "zookeeper"}
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
	c, err := local.New("../../../../testdata/wraps")
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
	c, err := local.New("../../../../testdata/wraps")
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
