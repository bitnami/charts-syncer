package chartrepotest

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

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
}

// Metadata in Chart.yaml files
type Metadata struct {
	AppVersion string `json:"appVersion"`
	Name       string `json:"name"`
	Version    string `json:"version"`
}

type ChartVersion struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	URLs    []string `json:"urls"`
}

type httpError struct {
	status int
	body   string
}

func newChartMuseumFake(t *testing.T, username, password string) *tChartMuseumFake {
	return &tChartMuseumFake{
		t:        t,
		username: username,
		password: password,
		index:    make(map[string][]*ChartVersion),
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
		cm.indexGet(w, r)
		return
	}
	re := regexp.MustCompile(`(?m)\/charts\/(.*.tgz)`)
	if re.Match([]byte(r.URL.Path)) && r.Method == "GET" {
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
func (cm *tChartMuseumFake) indexGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cm.index); err != nil {
		cm.t.Fatal(err)
	}
}

func (cm *tChartMuseumFake) chartPackageGet(w http.ResponseWriter, r *http.Request, chartPackageName string) {
	w.WriteHeader(200)
	// Get chart from testdata folder
	chartPackageFile := path.Join("../../testdata", chartPackageName)
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
