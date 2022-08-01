package helmclassic_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/time"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/cache/cachedisk"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/repo/helmclassic"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
)

var (
	cmRepo = &api.Repo{
		Kind: api.Kind_HELM,
		Auth: &api.Auth{
			Username: "user",
			Password: "password",
		},
	}
)

func prepareTest(t *testing.T, indexFileName string) *helmclassic.Repo {
	t.Helper()

	// Create temp folder and copy index.yaml
	dstTmp, err := ioutil.TempDir("", "charts-syncer-tests-index-fake")
	if err != nil {
		t.Fatalf("error creating temporary folder: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dstTmp) })
	dstIndex := filepath.Join(dstTmp, "index.yaml")
	srcIndex := filepath.Join("../../../../testdata", indexFileName)
	if err := utils.CopyFile(dstIndex, srcIndex); err != nil {
		t.Fatal(err)
	}

	// Create tester
	tester := helmclassic.NewTester(t, cmRepo, false, dstIndex, true)
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
	cache, err := cachedisk.New(cacheDir, cmRepo.GetUrl())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(cacheDir) })

	// Create chartmuseum client
	client, err := helmclassic.New(cmRepo, cache, false)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestFetch(t *testing.T) {
	c := prepareTest(t, "index.yaml")
	chartPath, err := c.Fetch("etcd", "4.8.0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(chartPath); err != nil {
		t.Errorf("chart package does not exist")
	}
}

func TestHas(t *testing.T) {
	c := prepareTest(t, "index.yaml")
	has, err := c.Has("etcd", "4.8.0")
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Errorf("chart not found in index")
	}
}

func TestList(t *testing.T) {
	c := prepareTest(t, "index.yaml")
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
	c := prepareTest(t, "index.yaml")
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
	c := prepareTest(t, "index.yaml")
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
	c := prepareTest(t, "index.yaml")
	if err := c.Reload(); err != nil {
		t.Fatal(err)
	}
	want := []string{"common", "etcd", "nginx"}
	indexFile := c.Index
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

func TestGetDownloadURL(t *testing.T) {
	tests := []struct {
		desc          string
		indexFileName string
	}{
		{
			"full url index",
			"index.yaml",
		},
		{
			"relative url index",
			"index-relative.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			c := prepareTest(t, tc.indexFileName)
			want := fmt.Sprintf("%s%s", cmRepo.Url, "/charts/etcd-4.8.0.tgz")
			got, err := c.GetDownloadURL("etcd", "4.8.0")
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("wrong download URL. got: %v, want: %v", got, want)
			}
		})
	}
}

func TestGetIndexURL(t *testing.T) {
	c := prepareTest(t, "index.yaml")
	want := fmt.Sprintf("%s%s", cmRepo.Url, "/index.yaml")
	got := c.GetIndexURL()
	if got != want {
		t.Errorf("wrong index URL. got: %v, want: %v", got, want)
	}
}

func TestUpload(t *testing.T) {
	c := prepareTest(t, "index.yaml")
	expectedError := "upload method is not supported yet"
	err := c.Upload("../../../testdata/apache-7.3.15.tgz", nil)
	if err.Error() != expectedError {
		t.Errorf("unexpected error message. got: %q, want: %q", err.Error(), expectedError)
	}
}
