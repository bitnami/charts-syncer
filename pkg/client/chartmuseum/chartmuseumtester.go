package chartmuseum

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
	"strings"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
	"github.com/juju/errors"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
)

// RepoTester allows to unit test each repo implementation
type RepoTester struct {
	url      *url.URL
	username string
	password string
}

// NewTester creates a Repo object from an api.Repo object.
func NewTester(repo *api.Repo) (*RepoTester, error) {
	u, err := url.Parse(repo.GetUrl())
	if err != nil {
		return nil, errors.Trace(err)
	}

	user := repo.GetAuth().GetUsername()
	pass := repo.GetAuth().GetPassword()
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &RepoTester{url: u, username: user, password: pass}, nil
}

// Fetch ...
func (r *RepoTester) Fetch(name string, version string) (string, error) {
	return "", nil
}

// List ...
func (r *RepoTester) List() ([]string, error) {
	return []string{}, nil
}

// ListChartVersions ...
func (r *RepoTester) ListChartVersions(name string) ([]string, error) {
	return []string{}, nil
}

// Has ...
func (r *RepoTester) Has(name string, version string) (bool, error) {
	return false, nil
}

// GetChartDetails ...
func (r *RepoTester) GetChartDetails(name string, version string) (*types.ChartDetails, error) {
	return nil, nil
}

// Reload ...
func (r *RepoTester) Reload() error {
	return nil
}

// Upload ...
func (r *RepoTester) Upload(filepath string, metadata *chart.Metadata) error {
	return nil
}

// ---------------------------------------------------------------

var (
	cmRegex         = regexp.MustCompile(`(?m)\/charts\/(.*.tgz)`)
	username string = "user"
	password string = "password"

	// ChartMuseumTests defines tests for fake ChartMuseum services. This
	// validates the publisher is correct and, at the same time, provides
	// reasonable confidence the fake implementation is good enough.
	ChartMuseumTests = []struct {
		Desc       string
		Skip       func(t *testing.T)
		MakeServer func(t *testing.T, emptyIndex bool, indexFile string) (string, func())
	}{
		{
			"fake service",
			func(t *testing.T) {},
			func(t *testing.T, emptyIndex bool, indexFile string) (string, func()) {
				s := httptest.NewServer(newChartMuseumFake(t, username, password, emptyIndex, indexFile))
				return s.URL, func() {
					s.Close()
				}
			},
		},
	}
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

// A tChartMuseumFake is a fake ChartMuseum implementation useful for (fast)
// unit tests.
//
// An instance implements `http.Handler` so can be used directly or with
// `httptest.NewServer` to make it available over HTTP.
type tChartMuseumFake struct {
	t *testing.T

	// Expected basic auth credentials.
	username string
	password string

	// Set to simulate HTTP error responses for specific API calls.
	ChartsPostError *httpError

	// Map of chart name to indexed versions, as returned by the charts API.
	index map[string][]*ChartVersion

	// Whether the repo should load an empty index or not
	emptyIndex bool

	// index.yaml to be loaded for testing purposes
	indexFile string
}

func newChartMuseumFake(t *testing.T, username, password string, emptyIndex bool, indexFile string) *tChartMuseumFake {
	return &tChartMuseumFake{
		t:          t,
		username:   username,
		password:   password,
		emptyIndex: emptyIndex,
		indexFile:  indexFile,
		index:      make(map[string][]*ChartVersion),
	}
}

func (cm *tChartMuseumFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check basic auth credentals.
	username, password, ok := r.BasicAuth()
	if got, want := ok, true; got != want {
		cm.t.Errorf("got: %t, want: %t", got, want)
	}
	if got, want := username, cm.username; got != want {
		cm.t.Errorf("got: %q, want: %q", got, want)
	}
	if got, want := password, cm.password; got != want {
		cm.t.Errorf("got: %q, want: %q", got, want)
	}

	// Handle recognized requests.
	if base, chart := path.Split(r.URL.Path); base == "/api/charts/" && r.Method == "GET" {
		cm.chartGet(w, r, chart)
		return
	}
	if r.URL.Path == "/api/charts" && r.Method == "POST" {
		cm.chartsPost(w, r)
		return
	}
	if r.URL.Path == "/index.yaml" && r.Method == "GET" {
		cm.getIndex(w, r, cm.emptyIndex, cm.indexFile)
		return
	}
	if cmRegex.Match([]byte(r.URL.Path)) && r.Method == "GET" {
		chartPackage := strings.Split(r.URL.Path, "/")[2]
		cm.chartPackageGet(w, r, chartPackage)
		return
	}

	cm.t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
}

func (cm *tChartMuseumFake) chartGet(w http.ResponseWriter, r *http.Request, chart string) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cm.index[chart]); err != nil {
		cm.t.Fatal(err)
	}
}

func (cm *tChartMuseumFake) getIndex(w http.ResponseWriter, r *http.Request, emptyIndex bool, indexFile string) {
	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(200)
	// Get index from testdata folder
	if indexFile == "" {
		indexFile = "../../testdata/index.yaml"
	}
	if emptyIndex {
		indexFile = "../../testdata/empty-index.yaml"
	}
	index, err := ioutil.ReadFile(indexFile)
	if err != nil {
		cm.t.Fatal(err)
	}
	w.Write(index)
}

func (cm *tChartMuseumFake) chartPackageGet(w http.ResponseWriter, r *http.Request, chartPackageName string) {
	w.WriteHeader(200)
	// Get chart from testdata folder
	chartPackageFile := path.Join("../../testdata/charts", chartPackageName)
	chartPackage, err := ioutil.ReadFile(chartPackageFile)
	if err != nil {
		cm.t.Fatal(err)
	}
	w.Write(chartPackage)
}

func (cm *tChartMuseumFake) chartsPost(w http.ResponseWriter, r *http.Request) {
	if cm.ChartsPostError != nil {
		w.WriteHeader(cm.ChartsPostError.status)
		w.Write([]byte(cm.ChartsPostError.body))
		return
	}

	chartFile, _, err := r.FormFile("chart")
	if err != nil {
		cm.t.Fatal(err)
	}

	metadata, err := chartMetadataFromTGZ(chartFile)
	if err != nil {
		cm.t.Fatal(err)
	}

	cm.index[metadata.Name] = append(cm.index[metadata.Name], &ChartVersion{
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
