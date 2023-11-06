package chartmuseum

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/bitnami/charts-syncer/pkg/client/repo/helmclassic"

	"github.com/bitnami/charts-syncer/api"
)

var (
	cmRegex           = regexp.MustCompile(`(?m)\/charts\/(.*.tgz)`)
	username   string = "user"
	password   string = "password"
	repository string = "myrepo"
)

// RepoTester allows to unit test each repo implementation
type RepoTester struct {
	url      *url.URL
	username string
	password string
	t        *testing.T

	// Helmclassic tester with common functions
	helmTester *helmclassic.RepoTester

	// Map of chart name to indexed versions, as returned by the charts API.
	index map[string][]*helmclassic.ChartVersion

	// Whether the repo should load an empty index or not
	emptyIndex bool

	// index.yaml to be loaded for testing purposes
	indexFile string
}

// NewTester creates fake HTTP server to handle requests and return a RepoTester object with useful info for testing
func NewTester(t *testing.T, repo *api.Repo, emptyIndex bool, indexFile string) *RepoTester {
	tester := &RepoTester{
		t:          t,
		username:   username,
		password:   password,
		helmTester: helmclassic.NewTester(t, repo, emptyIndex, indexFile, false),
		index:      make(map[string][]*helmclassic.ChartVersion),
		emptyIndex: emptyIndex,
		indexFile:  indexFile,
	}
	s := httptest.NewServer(tester)
	u, err := url.Parse(fmt.Sprintf("%s/%s", s.URL, repository))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	tester.url = u
	return tester
}

// ServeHTTP implements the the http Handler type
func (rt *RepoTester) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check basic auth credentals.
	username, password, ok := r.BasicAuth()
	if got, want := ok, true; got != want {
		rt.t.Errorf("got: %t, want: %t", got, want)
	}
	if got, want := username, rt.username; got != want {
		rt.t.Errorf("got: %q, want: %q", got, want)
	}
	if got, want := password, rt.password; got != want {
		rt.t.Errorf("got: %q, want: %q", got, want)
	}

	// Handle recognized requests.
	if base, chart := path.Split(r.URL.Path); base == "/myrepo/api/charts/" && r.Method == "GET" {
		rt.GetChart(w, r, chart)
		return
	}
	if r.URL.Path == "/myrepo/api/charts" && r.Method == "POST" {
		rt.PostChart(w, r)
		return
	}
	if r.URL.Path == "/myrepo/index.yaml" && r.Method == "GET" {
		rt.GetIndex(w, r, rt.emptyIndex, rt.indexFile)
		return
	}
	if cmRegex.Match([]byte(r.URL.Path)) && r.Method == "GET" {
		chartPackage := strings.Split(r.URL.Path, "/")[3]
		rt.GetChartPackage(w, r, chartPackage)
		return
	}

	rt.t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
}

// GetChart returns the chart info from the index
func (rt *RepoTester) GetChart(w http.ResponseWriter, r *http.Request, chart string) {
	rt.helmTester.GetChart(w, r, chart)
}

// GetURL returns the URL of the server
func (rt *RepoTester) GetURL() string {
	return rt.url.String()
}

// GetIndex returns an index file
func (rt *RepoTester) GetIndex(w http.ResponseWriter, r *http.Request, emptyIndex bool, indexFile string) {
	rt.helmTester.GetIndex(w, r, emptyIndex, indexFile)
}

// GetChartPackage returns a packaged helm chart
func (rt *RepoTester) GetChartPackage(w http.ResponseWriter, r *http.Request, chartPackageName string) {
	rt.helmTester.GetChartPackage(w, r, chartPackageName)
}

// PostChart push a packaged chart
func (rt *RepoTester) PostChart(w http.ResponseWriter, r *http.Request) {
	rt.helmTester.PostChart(w, r)
}
