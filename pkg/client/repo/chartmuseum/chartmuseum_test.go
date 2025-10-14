package chartmuseum_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/time"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/internal/cache/cachedisk"
	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/bitnami/charts-syncer/pkg/client/repo/chartmuseum"
	"github.com/bitnami/charts-syncer/pkg/client/types"
)

var (
	cmRepo = &api.Repo{
		Kind: api.Kind_CHARTMUSEUM,
		Auth: &api.Auth{
			Username: "user",
			Password: "password",
		},
	}
)

func prepareTest(t *testing.T) (*chartmuseum.Repo, error) {
	t.Helper()

	// Create temp folder and copy index.yaml
	dstTmp, err := os.MkdirTemp("", "charts-syncer-tests-index-fake")
	if err != nil {
		t.Fatalf("error creating temporary folder: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dstTmp) })
	dstIndex := filepath.Join(dstTmp, "index.yaml")
	if err = utils.CopyFile(dstIndex, "../../../../testdata/index.yaml"); err != nil {
		t.Fatal(err)
	}

	// Create tester
	tester := chartmuseum.NewTester(t, false, dstIndex)
	cmRepo.Url = tester.GetURL()

	// Replace placeholder
	u := fmt.Sprintf("%s%s", tester.GetURL(), "/charts")
	index, err := os.ReadFile(dstIndex)
	if err != nil {
		t.Fatal(err)
	}
	newContents := strings.ReplaceAll(string(index), "TEST_PLACEHOLDER", u)
	if err = os.WriteFile(dstIndex, []byte(newContents), 0); err != nil {
		t.Fatal(err)
	}

	// Define cache dir
	cacheDir, err := os.MkdirTemp("", "client")
	if err != nil {
		t.Fatal(err)
	}
	cache, err := cachedisk.New(cacheDir, cmRepo.GetUrl())
	if err != nil {
		t.Fatal(err)
	}

	// Create chartmuseum client
	client, err := chartmuseum.New(cmRepo, cache, false)
	if err != nil {
		t.Fatal(err)
	}
	return client, nil
}

func TestFetch(t *testing.T) {
	c, err := prepareTest(t)
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
	c, err := prepareTest(t)
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
	c, err := prepareTest(t)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"common", "etcd", "nginx"}
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
	c, err := prepareTest(t)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"4.8.0", "4.7.4", "4.7.3", "4.7.2", "4.7.1", "4.7.0"}
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
	c, err := prepareTest(t)
	if err != nil {
		t.Fatal(err)
	}
	want := types.ChartDetails{
		PublishedAt: time.Now().Time,
		Digest:      "d47d94c52aff1fbb92235f0753c691072db1d19ec43fa9a438ab6736dfa7f867",
	}
	got, err := c.GetChartDetails("etcd", "4.8.0")
	if err != nil {
		t.Fatal(err)
	}
	if want.Digest != got.Digest {
		t.Errorf("unexpected digest in chart. got: %v, want: %v", got, want)
	}
}

func TestGetUploadURL(t *testing.T) {
	c, err := prepareTest(t)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("%s%s", cmRepo.Url, "/api/charts")
	got := c.GetUploadURL()
	if got != want {
		t.Errorf("wrong upload URL. got: %v, want: %v", got, want)
	}
}
