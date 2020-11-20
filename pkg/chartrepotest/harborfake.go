package chartrepotest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"regexp"
	"strings"
	"testing"
)

var (
	harborRegex *regexp.Regexp = regexp.MustCompile(`(?m)\/chartrepo/library/charts\/(.*.tgz)`)
)

// A tHarborFake is a fake ChartMuseum implementation useful for (fast)
// unit tests.
//
// An instance implements `http.Handler` so can be used directly or with
// `httptest.NewServer` to make it available over HTTP.
type tHarborFake struct {
	t *testing.T

	// Expected basic auth credentials.
	username string
	password string

	// Set to simulate HTTP error responses for specific API calls.
	ChartsPostError *httpError

	// Map of chart name to indexed versions, as returned by the charts API.
	index map[string][]*ChartVersion
}

func newHarborFake(t *testing.T, username, password string) *tHarborFake {
	return &tHarborFake{
		t:        t,
		username: username,
		password: password,
		index:    make(map[string][]*ChartVersion),
	}
}

func (h *tHarborFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check basic auth credentals.
	username, password, ok := r.BasicAuth()
	if got, want := ok, true; got != want {
		h.t.Errorf("got: %t, want: %t", got, want)
	}
	if got, want := username, h.username; got != want {
		h.t.Errorf("got: %q, want: %q", got, want)
	}
	if got, want := password, h.password; got != want {
		h.t.Errorf("got: %q, want: %q", got, want)
	}

	// Handle recognized requests.
	if base, chart := path.Split(r.URL.Path); base == "/chartrepo/library/" && r.Method == "GET" {
		h.chartGet(w, r, chart)
		return
	}
	if r.URL.Path == "/api/chartrepo/library/charts" && r.Method == "POST" {
		h.chartsPost(w, r)
		return
	}
	if r.URL.Path == "/chartrepo/library/index.yaml" && r.Method == "GET" {
		h.indexGet(w, r)
		return
	}
	if harborRegex.Match([]byte(r.URL.Path)) && r.Method == "GET" {
		chartPackage := strings.Split(r.URL.Path, "/")[4]
		h.chartPackageGet(w, r, chartPackage)
		return
	}

	h.t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
}

func (h *tHarborFake) chartGet(w http.ResponseWriter, r *http.Request, chart string) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.index[chart]); err != nil {
		h.t.Fatal(err)
	}
}
func (h *tHarborFake) indexGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.index); err != nil {
		h.t.Fatal(err)
	}
}

func (h *tHarborFake) chartPackageGet(w http.ResponseWriter, r *http.Request, chartPackageName string) {
	w.WriteHeader(200)
	// Get chart from testdata folder
	chartPackageFile := path.Join("../../../testdata", chartPackageName)
	chartPackage, err := ioutil.ReadFile(chartPackageFile)
	if err != nil {
		h.t.Fatal(err)
	}
	w.Write(chartPackage)
}

func (h *tHarborFake) chartsPost(w http.ResponseWriter, r *http.Request) {
	if h.ChartsPostError != nil {
		w.WriteHeader(h.ChartsPostError.status)
		w.Write([]byte(h.ChartsPostError.body))
		return
	}

	chartFile, _, err := r.FormFile("chart")
	if err != nil {
		h.t.Fatal(err)
	}

	metadata, err := chartMetadataFromTGZ(chartFile)
	if err != nil {
		h.t.Fatal(err)
	}

	h.index[metadata.Name] = append(h.index[metadata.Name], &ChartVersion{
		Name:    metadata.Name,
		Version: metadata.Version,
		URLs:    []string{fmt.Sprintf("charts/%s-%s.tgz", metadata.Name, metadata.Version)},
	})

	w.WriteHeader(201)
	w.Write([]byte(`{}`))
}
