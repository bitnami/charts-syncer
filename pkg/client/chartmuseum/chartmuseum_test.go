package chartmuseum

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/cache"
	"github.com/bitnami-labs/charts-syncer/internal/chartrepotest"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
	"helm.sh/helm/v3/pkg/time"
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

func prepareTest(t *testing.T) (*Repo, *RepoTester, error) {
	t.Helper()

	// Create temp folder and copy index.yaml
	dstTmp, err := ioutil.TempDir("", "charts-syncer-tests-index-fake")
	if err != nil {
		t.Fatalf("error creating temporary folder: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dstTmp) })
	dstIndex := filepath.Join(dstTmp, "index.yaml")
	if err := utils.CopyFile(dstIndex, "../../../testdata/index.yaml"); err != nil {
		t.Fatal(err)
	}

	// Create tester
	tester, cleanup, err := NewTester(t, cmRepo, false, dstIndex)
	t.Cleanup(func() { cleanup() })
	if err != nil {
		t.Fatal(err)
	}
	cmRepo.Url = tester.GetURL()

	// Replace placeholder
	u := fmt.Sprintf("%s%s", tester.GetURL(), "/charts")
	index, err := ioutil.ReadFile(dstIndex)
	if err != nil {
		t.Fatal(err)
	}
	newContents := strings.Replace(string(index), "TEST_PLACEHOLDER", u, -1)
	if err = ioutil.WriteFile(dstIndex, []byte(newContents), 0); err != nil {
		t.Fatal(err)
	}

	// Define cache dir
	cacheDir, err := ioutil.TempDir("", "client")
	if err != nil {
		t.Fatal(err)
	}
	cache, err := cache.New(cacheDir, cmRepo.GetUrl())
	if err != nil {
		t.Fatal(err)
	}

	// Create chartmuseum client
	client, err := New(cmRepo, cache)
	if err != nil {
		t.Fatal(err)
	}
	return client, tester, nil
}

func TestFetch(t *testing.T) {
	c, _, err := prepareTest(t)
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
	c, _, err := prepareTest(t)
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
	c, _, err := prepareTest(t)
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
	c, _, err := prepareTest(t)
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
	c, _, err := prepareTest(t)
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

func TestReload(t *testing.T) {
	c, _, err := prepareTest(t)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Reload(); err != nil {
		t.Fatal(err)
	}
	want := []string{"common", "etcd", "nginx"}
	indexFile := c.helm.Index
	got := []string{}
	for chart := range indexFile.Entries {
		got = append(got, chart)
	}
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected list of charts. got: %v, want: %v", got, want)
	}
}

func TestGetUploadURL(t *testing.T) {
	c, tester, err := prepareTest(t)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("%s%s", tester.GetURL(), "/api/charts")
	got := c.GetUploadURL()
	if got != want {
		t.Errorf("wrong upload URL. got: %v, want: %v", got, want)
	}
}

func TestUpload(t *testing.T) {
	c, tester, err := prepareTest(t)
	if err != nil {
		t.Fatal(err)
	}
	err = c.Upload("../../../testdata/apache-7.3.15.tgz", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Check the chart really was added to the service's index.
	req, err := http.NewRequest("GET", tester.GetURL()+"/api/charts/apache", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(cmRepo.Auth.Username, cmRepo.Auth.Password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	charts := []*chartrepotest.ChartVersion{}
	if err := json.NewDecoder(resp.Body).Decode(&charts); err != nil {
		t.Fatal(err)
	}
	if got, want := len(charts), 1; got != want {
		t.Fatalf("got: %q, want: %q", got, want)
	}
	if got, want := charts[0].Name, "apache"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
	if got, want := charts[0].Version, "7.3.15"; got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}
