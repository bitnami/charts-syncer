package helmclassic

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/bitnami/charts-syncer/api"
	"gopkg.in/yaml.v2"
)

var (
	helmRegex        = regexp.MustCompile(`(?m)\/charts\/(.*.tgz)`)
	username  string = "user"
	password  string = "password"
)

// Metadata in Chart.yaml files
type Metadata struct {
	AppVersion string `json:"appVersion"`
	Name       string `json:"name"`
	Version    string `json:"version"`
}

// ChartVersion type
type ChartVersion struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	URLs    []string `json:"urls"`
}

type httpError struct {
	status int
	body   string
}

// RepoTester allows to unit test each repo implementation
type RepoTester struct {
	url      *url.URL
	username string
	password string
	t        *testing.T
	// Map of chart name to indexed versions, as returned by the charts API.
	index map[string][]*ChartVersion

	// Whether the repo should load an empty index or not
	emptyIndex bool

	// index.yaml to be loaded for testing purposes
	indexFile string
	// Set to simulate HTTP error responses for specific API calls.
	ChartsPostError *httpError
}

// NewTester creates fake HTTP server to handle requests and return a RepoTester object with useful info for testing
func NewTester(t *testing.T, repo *api.Repo, emptyIndex bool, indexFile string, createServer bool) *RepoTester {
	t.Helper()
	tester := &RepoTester{
		t:          t,
		username:   username,
		password:   password,
		emptyIndex: emptyIndex,
		indexFile:  indexFile,
		index:      make(map[string][]*ChartVersion),
	}
	if createServer {
		s := httptest.NewServer(tester)
		u, err := url.Parse(s.URL)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(s.Close)
		tester.url = u
	}
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
	if r.URL.Path == "/index.yaml" && r.Method == "GET" {
		rt.GetIndex(w, r, rt.emptyIndex, rt.indexFile)
		return
	}
	if helmRegex.Match([]byte(r.URL.Path)) && r.Method == "GET" {
		chartPackage := strings.Split(r.URL.Path, "/")[2]
		rt.GetChartPackage(w, r, chartPackage)
		return
	}

	rt.t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
}

// GetChart returns the chart info from the index
func (rt *RepoTester) GetChart(w http.ResponseWriter, r *http.Request, chart string) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rt.index[chart]); err != nil {
		rt.t.Fatal(err)
	}
}

// GetURL returns the URL of the server
func (rt *RepoTester) GetURL() string {
	return rt.url.String()
}

// GetIndex returns an index file
func (rt *RepoTester) GetIndex(w http.ResponseWriter, r *http.Request, emptyIndex bool, indexFile string) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(200)
	_, filename, _, _ := runtime.Caller(1)
	testdataPath := path.Join(path.Dir(filename), "../../../../testdata")
	// Get index from testdata folder
	if indexFile == "" {
		indexFile = filepath.Join(testdataPath, "index.yaml")
	}
	if emptyIndex {
		indexFile = filepath.Join(testdataPath, "empty-index.yaml")
	}
	index, err := ioutil.ReadFile(indexFile)
	if err != nil {
		rt.t.Fatal(err)
	}
	w.Write(index)
}

// GetChartPackage returns a packaged helm chart
func (rt *RepoTester) GetChartPackage(w http.ResponseWriter, r *http.Request, chartPackageName string) {
	w.WriteHeader(200)
	_, filename, _, _ := runtime.Caller(1)
	testdataPath := path.Join(path.Dir(filename), "../../../../testdata")
	// Get chart from testdata folder
	chartPackageFile := path.Join(testdataPath, "charts", chartPackageName)
	chartPackage, err := ioutil.ReadFile(chartPackageFile)
	if err != nil {
		rt.t.Fatal(err)
	}
	w.Write(chartPackage)
}

// PostChart push a packaged chart
func (rt *RepoTester) PostChart(w http.ResponseWriter, r *http.Request) {
	if rt.ChartsPostError != nil {
		w.WriteHeader(rt.ChartsPostError.status)
		w.Write([]byte(rt.ChartsPostError.body))
		return
	}

	chartFile, _, err := r.FormFile("chart")
	if err != nil {
		rt.t.Fatal(err)
	}

	metadata, err := chartMetadataFromTGZ(chartFile)
	if err != nil {
		rt.t.Fatal(err)
	}

	rt.index[metadata.Name] = append(rt.index[metadata.Name], &ChartVersion{
		Name:    metadata.Name,
		Version: metadata.Version,
		URLs:    []string{fmt.Sprintf("charts/%s-%s.tgz", metadata.Name, metadata.Version)},
	})

	w.WriteHeader(201)
	w.Write([]byte(`{}`))
}

func chartMetadataFromTGZ(r io.Reader) (*Metadata, error) {
	const (
		metadataFile = "Chart.yaml"
	)

	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	t := tar.NewReader(gz)

	// Iterate over tar until the metadata file
	for {
		h, err := t.Next()
		if err != nil {
			return nil, err
		}
		// A tgz may contain multiple Chart.yaml files - the main chart and its
		// subcharts - but assume there are no subcharts for now.
		_, file := filepath.Split(h.Name)
		if file == metadataFile {
			break
		}
	}

	data, err := ioutil.ReadAll(t)
	if err != nil {
		return nil, err
	}
	m := &Metadata{}
	return m, yaml.Unmarshal(data, m)
}
