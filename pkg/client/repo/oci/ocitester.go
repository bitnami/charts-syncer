package oci

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry"
	"oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/internal/cache/cachedisk"
	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/bitnami/charts-syncer/pkg/client/repo/helmclassic"
)

var (
	ociPing             = regexp.MustCompile(`(?m)\/v2\/(.*)`)
	ociIndexRegex       = regexp.MustCompile(`(?m)\/v2\/(.*)\/index\/manifests\/latest`)
	ociTagManifestRegex = regexp.MustCompile(`(?m)\/v2\/(.*)\/manifests\/(.*)`)
	ociBlobsRegex       = regexp.MustCompile(`(?m)\/v2\/(.*)\/blobs\/sha256:(.*)`)
	ociTagsListRegex    = regexp.MustCompile(`(?m)\/v2\/(.*)\/tags\/list`)
	username            = "user"
	password            = "password"
)

// RepoTester allows to unit test each repo implementation
type RepoTester struct {
	url      *url.URL
	username string
	password string
	t        *testing.T
	// Map of chart name to indexed versions, as returned by the charts API.
	index map[string][]*helmclassic.ChartVersion
}

func PushFileToOCI(t *testing.T, filepath string, ref string) {
	ctx := context.Background()
	resolver := docker.NewResolver(docker.ResolverOptions{PlainHTTP: true})
	fileContent, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatal(err)
	}
	filename := path.Base(filepath)
	customMediaType := "my.custom.media.type"
	memoryStore := content.NewMemory()

	blobDesc, err := memoryStore.Add(filename, customMediaType, fileContent)
	if err != nil {
		t.Fatal(err)
	}
	manifest, manifestDesc, config, configDesc, err := content.GenerateManifestAndConfig(nil, nil, blobDesc)
	if err != nil {
		t.Fatal(err)
	}
	memoryStore.Set(configDesc, config)
	if err := memoryStore.StoreManifest(ref, manifestDesc, manifest); err != nil {
		t.Fatal(err)
	}

	if _, err := oras.Copy(ctx, memoryStore, ref, resolver, ref, oras.WithNameValidation(nil)); err != nil {
		t.Fatal(err)
	}
}

func PrepareTest(t *testing.T, ociRepo *api.Repo) *Repo {
	t.Helper()

	// Define cache dir
	cacheDir, err := os.MkdirTemp("", "client")
	if err != nil {
		t.Fatal(err)
	}
	cache, err := cachedisk.New(cacheDir, ociRepo.GetUrl())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(cacheDir) })

	// Create oci client
	client, err := New(ociRepo, cache, false, true)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// Creates an HTTP server that knows how to reply to all OCI related request except PUSH one.
func PrepareHttpServer(t *testing.T, ociRepo *api.Repo) *Repo {
	t.Helper()

	// Create HTTP server
	tester := NewTester(t, ociRepo)
	ociRepo.Url = tester.GetURL() + "/someproject/charts"
	return PrepareTest(t, ociRepo)
}

// Starts an OCI compliant server (docker-registry) so our push command based on oras cli works out-of-the-box.
// This way we don't have to mimic all the low-level HTTP requests made by oras.
func PrepareOciServer(t *testing.T, ociRepo *api.Repo) {
	t.Helper()

	// Create OCI server as docker registry
	config := &configuration.Configuration{}

	addr, err := utils.GetListenAddress()
	if err != nil {
		t.Fatal(err)
	}
	dockerRegistryHost := "http://" + addr
	config.HTTP.Addr = fmt.Sprintf(addr)
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}
	dockerRegistry, err := registry.NewRegistry(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	go dockerRegistry.ListenAndServe()
	ociRepo.Url = dockerRegistryHost + "/someproject/charts"
}

// NewTester creates fake HTTP server to handle requests and return a RepoTester object with useful info for testing
func NewTester(t *testing.T, repo *api.Repo) *RepoTester {
	t.Helper()
	tester := &RepoTester{
		t:        t,
		username: username,
		password: password,
		index:    make(map[string][]*helmclassic.ChartVersion),
	}
	s := httptest.NewServer(tester)
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	tester.url = u
	return tester
}

func testBasicAuth(t *testing.T, r *http.Request) {
	// Check basic auth credentals.
	username, password, ok := r.BasicAuth()
	if got, want := ok, true; got != want {
		t.Fatalf("got: %t, want: %t", got, want)
	}
	if got, want := username, username; got != want {
		t.Fatalf("got: %q, want: %q", got, want)
	}
	if got, want := password, password; got != want {
		t.Fatalf("got: %q, want: %q", got, want)
	}
}

// ServeHTTP implements the http Handler type
func (rt *RepoTester) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle recognized requests.
	if ociIndexRegex.Match([]byte(r.URL.Path)) && r.Method == "HEAD" {
		rt.HeadManifest404(w)
		return
	}
	if ociBlobsRegex.Match([]byte(r.URL.Path)) && r.Method == "GET" {
		testBasicAuth(rt.t, r)
		name := strings.Split(r.URL.Path, "/")[4]
		fullDigest := strings.Split(r.URL.Path, "/")[6]
		digest := strings.Split(fullDigest, ":")[1]
		rt.GetChartPackage(w, r, name, digest)
		return
	}
	if ociTagManifestRegex.Match([]byte(r.URL.Path)) && r.Method == "HEAD" {
		testBasicAuth(rt.t, r)
		rt.HeadManifest200(w)
		return
	}
	if ociTagManifestRegex.Match([]byte(r.URL.Path)) && r.Method == "GET" {
		testBasicAuth(rt.t, r)
		name := strings.Split(r.URL.Path, "/")[4]
		version := strings.Split(r.URL.Path, "/")[6]
		rt.GetTagManifest(w, r, name, version)
		return
	}
	if ociTagsListRegex.Match([]byte(r.URL.Path)) && r.Method == "GET" {
		testBasicAuth(rt.t, r)
		name := strings.Split(r.URL.Path, "/")[4]
		rt.GetTagsList(w, r, name)
		return
	}
	if ociPing.Match([]byte(r.URL.Path)) && r.Method == "GET" {
		rt.ReplyPing(w)
		return
	}

	rt.t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
}

// GetURL returns the URL of the server
func (rt *RepoTester) GetURL() string {
	return rt.url.String()
}

// GetTagManifest returns the oci manifest of a specific tag
func (rt *RepoTester) GetTagManifest(w http.ResponseWriter, r *http.Request, name, version string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, filename, _, _ := runtime.Caller(1)
	testdataPath := path.Join(path.Dir(filename), "../../../../testdata/oci")
	// Get oci manifest from testdata folder
	manifestFileName := fmt.Sprintf("%s-%s-oci-manifest.json", name, version)
	manifestFile := filepath.Join(testdataPath, manifestFileName)
	manifest, err := os.ReadFile(manifestFile)
	if err != nil {
		rt.t.Fatal(err)
	}
	w.Write(manifest)
}

// GetTagsList returns the list of available tags for the specified asset
func (rt *RepoTester) GetTagsList(w http.ResponseWriter, r *http.Request, name string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, filename, _, _ := runtime.Caller(1)
	testdataPath := path.Join(path.Dir(filename), "../../../../testdata/oci")
	// Get oci manifest from testdata folder
	tagsListFileName := fmt.Sprintf("%s-oci-tags-list.json", name)
	tagsListFile := filepath.Join(testdataPath, tagsListFileName)
	tagsList, err := os.ReadFile(tagsListFile)
	if err != nil {
		rt.t.Fatal(err)
	}
	w.Write(tagsList)
}

// HeadManifest200 return if a manifests exists or not
func (rt *RepoTester) HeadManifest200(w http.ResponseWriter) {
	w.WriteHeader(200)
}

// HeadManifest404 return if a manifests exists or not
func (rt *RepoTester) HeadManifest404(w http.ResponseWriter) {
	w.WriteHeader(404)
}

// ReplyPing reply to a ping operation done by remote.Head to guess some connection parameters
// https://github.com/google/go-containerregistry/blob/main/pkg/v1/remote/transport/transport.go#L51
func (rt *RepoTester) ReplyPing(w http.ResponseWriter) {
	w.WriteHeader(200)
}

// GetChartPackage returns a packaged helm chart
func (rt *RepoTester) GetChartPackage(w http.ResponseWriter, r *http.Request, name, digest string) {
	w.WriteHeader(200)
	_, filename, _, _ := runtime.Caller(1)
	chartPackageName := fmt.Sprintf("%s-%s.tgz", name, digest)
	testdataPath := path.Join(path.Dir(filename), "../../../../testdata/oci")
	// Get chart from testdata folder
	chartPackageFile := path.Join(testdataPath, "charts", chartPackageName)
	chartPackage, err := os.ReadFile(chartPackageFile)
	if err != nil {
		rt.t.Fatal(err)
	}
	w.Write(chartPackage)
}

// GetIndex returns an index file
func (rt *RepoTester) GetIndex(_ http.ResponseWriter, _ *http.Request) {
}

// GetChart returns the chart info from the index
func (rt *RepoTester) GetChart(_ http.ResponseWriter, _ *http.Request, _ string) {
}

// PostChart push a packaged chart
func (rt *RepoTester) PostChart(_ http.ResponseWriter, _ *http.Request) {
}
