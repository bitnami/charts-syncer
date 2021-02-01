package oci_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/cache"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/oci"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
	"github.com/docker/distribution/configuration"
	"github.com/docker/distribution/registry"
	_ "github.com/docker/distribution/registry/storage/driver/inmemory"
	"helm.sh/helm/v3/pkg/chart"
)

var (
	ociRepo = &api.Repo{
		Kind: api.Kind_OCI,
		Auth: &api.Auth{
			Username: "user",
			Password: "password",
		},
	}
)

func prepareTest(t *testing.T) *oci.Repo {
	t.Helper()

	// Define cache dir
	cacheDir, err := ioutil.TempDir("", "client")
	if err != nil {
		t.Fatal(err)
	}
	cache, err := cache.New(cacheDir, ociRepo.GetUrl())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(cacheDir) })

	// Create oci client
	client, err := oci.New(ociRepo, cache)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// Creates an HTTP server that knows how to reply to all OCI related request except PUSH one.
func prepareHttpServer(t *testing.T) *oci.Repo {
	t.Helper()

	// Create HTTP server
	tester := oci.NewTester(t, ociRepo)
	ociRepo.Url = tester.GetURL() + "/someproject/charts"

	return prepareTest(t)
}

// Starts an OCI compliant server (docker-registry) so our push command based on oras cli works out-of-the-box.
// This way we don't have to mimic all the low-level HTTP requests made by oras.
func prepareOciServer(t *testing.T) *oci.Repo {
	t.Helper()

	// Create OCI server as docker registry
	config := &configuration.Configuration{}

	addr, err := utils.GetListenAddress()
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}
	dockerRegistryHost := "http://" + addr
	config.HTTP.Addr = fmt.Sprintf(":%s", port)
	config.HTTP.DrainTimeout = time.Duration(10) * time.Second
	config.Storage = map[string]configuration.Parameters{"inmemory": map[string]interface{}{}}
	dockerRegistry, err := registry.NewRegistry(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	go dockerRegistry.ListenAndServe()
	//	End OCI part
	ociRepo.Url = dockerRegistryHost + "/someproject/charts"
	return prepareTest(t)
}

func TestFetch(t *testing.T) {
	c := prepareHttpServer(t)
	chartPath, err := c.Fetch("kafka", "12.2.1")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ChartPath is %q", chartPath)
	if _, err := os.Stat(chartPath); err != nil {
		t.Errorf("chart package does not exist")
	}
	contentType, err := utils.GetFileContentType(chartPath)
	if err != nil {
		t.Fatalf("error checking contentType of %s file", chartPath)
	}
	if contentType != "application/x-gzip" {
		t.Errorf("incorrect content type, got: %s, want: %s.", contentType, "application/x-gzip")
	}
}

func TestHas(t *testing.T) {
	c := prepareHttpServer(t)
	has, err := c.Has("kafka", "12.2.1")
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Errorf("chart not found in index")
	}
}

func TestList(t *testing.T) {
	c := prepareHttpServer(t)
	expectedError := "list method is not supported yet"
	_, err := c.List()
	if err.Error() != expectedError {
		t.Errorf("unexpected error message. got: %q, want: %q", err.Error(), expectedError)
	}
}

func TestListChartVersions(t *testing.T) {
	c := prepareHttpServer(t)
	want := []string{"12.2.1"}
	got, err := c.ListChartVersions("kafka")
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
	c := prepareHttpServer(t)
	want := types.ChartDetails{
		PublishedAt: time.Now(),
		Digest:      "sha256:11e974d88391a39e4dd6d7d6c4350b237b1cca1bf32f2074bba41109eaa5f438",
	}
	got, err := c.GetChartDetails("kafka", "12.2.1")
	if err != nil {
		t.Fatal(err)
	}
	if want.Digest != got.Digest {
		t.Errorf("unexpected digest in chart. got: %v, want: %v", got, want)
	}
}

func TestReload(t *testing.T) {
	c := prepareHttpServer(t)
	expectedError := "reload method is not supported yet"
	err := c.Reload()
	if err.Error() != expectedError {
		t.Errorf("unexpected error message. got: %q, want: %q", err.Error(), expectedError)
	}
}

func TestGetDownloadURL(t *testing.T) {
	c := prepareHttpServer(t)
	u, err := url.Parse(ociRepo.Url)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = path.Join("v2", u.Path, "kafka/blobs/sha256:11e974d88391a39e4dd6d7d6c4350b237b1cca1bf32f2074bba41109eaa5f438")
	want := u.String()
	got, err := c.GetDownloadURL("kafka", "12.2.1")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("wrong download URL. got: %v, want: %v", got, want)
	}
}

func TestUpload(t *testing.T) {
	c := prepareOciServer(t)
	chartMetadata := &chart.Metadata{
		Name:    "apache",
		Version: "7.3.15",
	}
	if err := c.Upload("../../../testdata/apache-7.3.15.tgz", chartMetadata); err != nil {
		t.Fatal(err)
	}
	chartPath, err := c.Fetch("apache", "7.3.15")
	if _, err := os.Stat(chartPath); err != nil {
		t.Errorf("chart package does not exist")
	}
	contentType, err := utils.GetFileContentType(chartPath)
	if err != nil {
		t.Fatalf("error checking contentType of %s file", chartPath)
	}
	if contentType != "application/x-gzip" {
		t.Errorf("incorrect content type, got: %s, want: %s.", contentType, "application/x-gzip")
	}
}
