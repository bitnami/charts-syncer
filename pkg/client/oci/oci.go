package oci

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bitnami-labs/pbjson"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/juju/errors"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/klog"
	"oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/cache"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	orascontext "oras.land/oras-go/pkg/context"
)

const (
	getTimeout = 1 * time.Minute
	// HelmChartConfigMediaType is the reserved media type for the Helm chart manifest config
	HelmChartConfigMediaType = "application/vnd.cncf.helm.config.v1+json"
	// HelmChartContentLayerMediaType is the reserved media type for Helm chart package content
	HelmChartContentLayerMediaType = "application/tar+gzip"
	// ImageManifestMediaType is the reserved media type for OCI manifests
	ImageManifestMediaType = "application/vnd.oci.image.manifest.v1+json"
)

var (
	tagRegex = regexp.MustCompile(`^([a-z0-9.:]+/[\w\-_/]+):?([\w0-9.]+)?`)
)

// Repo allows to operate a chart repository.
type Repo struct {
	url      *url.URL
	username string
	password string
	insecure bool

	entries        map[string][]string
	cache          cache.Cacher
	dockerResolver remotes.Resolver
}

// Tags contains the tags for a specific OCI artifact
type Tags struct {
	Name string
	Tags []string
}

// New creates a Repo object from an api.Repo object.
func New(repo *api.Repo, c cache.Cacher, insecure bool) (*Repo, error) {
	u, err := url.Parse(repo.GetUrl())
	if err != nil {
		return nil, errors.Trace(err)
	}
	resolver := newDockerResolver(u, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword(), insecure)

	// First we try to load the index referenced from config file,
	// if it does not exist, then we try to guess it directly from repo url
	chartsIndexRef := repo.GetChartsIndex()
	if chartsIndexRef == "" {
		chartsIndexRef = fmt.Sprintf("%s%s/index:latest", u.Host, u.Path)
		klog.Infof("No Charts OCI index reference specified. Checking default location: %q", chartsIndexRef)
	}

	ociIndexExists, err := ociReferenceExists(u, chartsIndexRef, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword(), insecure)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// Init entries
	entries := make(map[string][]string)
	if ociIndexExists {
		entries, err = populateEntries(chartsIndexRef, resolver)
		if err != nil {
			return nil, errors.Trace(err)
		}
	} else {
		klog.Info("Unable to load OCI index file. Using 'charts' filter from config file...")
	}

	return NewRaw(u, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword(), c, insecure, entries, resolver)
}

// NewRaw creates a Repo object.
func NewRaw(u *url.URL, user string, pass string, c cache.Cacher, insecure bool, entries map[string][]string, resolver remotes.Resolver) (*Repo, error) {
	return &Repo{url: u, username: user, password: pass, cache: c, insecure: insecure, entries: entries, dockerResolver: resolver}, nil
}

// List lists all chart names in a repo
func (r *Repo) List() ([]string, error) {
	var names []string
	// If entries is not populated, it means we couldn't load any index file, so we need the charts filter in the
	// configuration file. The List() caller will handle this case
	if len(r.entries) == 0 {
		return []string{}, nil
	}
	for entry := range r.entries {
		names = append(names, entry)
	}
	return names, nil
}

// getTagManifest returns the manifests of a published tag
func (r *Repo) getTagManifest(name, version string) (*ocispec.Manifest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), getTimeout)
	defer cancel()

	u := *r.url
	// Form API endpoint URL from repo url
	u.Path = path.Join("v2", u.Path, name, "manifests", version)
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	req.Header.Set("Accept", ImageManifestMediaType)

	if err != nil {
		return nil, errors.Trace(err)
	}
	if r.username != "" && r.password != "" {
		req.SetBasicAuth(r.username, r.password)
	}
	client := http.DefaultClient
	if r.insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer resp.Body.Close()

	status := resp.StatusCode
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("unexpected response — %d %q — from %s", status, http.StatusText(status), u.String())
	}
	if status != http.StatusOK {
		return nil, errors.Errorf("unexpected response — %d %q, %s — from %s", status, http.StatusText(status), string(body), u.String())
	}
	tm := &ocispec.Manifest{}
	if err := json.Unmarshal(body, tm); err != nil {
		return nil, err
	}
	return tm, nil
}

// getChartDigest returns the digest of a published chart
func (r *Repo) getChartDigest(name, version string) (string, error) {
	tm, err := r.getTagManifest(name, version)
	if err != nil {
		return "", errors.Trace(err)
	}
	for _, layer := range tm.Layers {
		if layer.MediaType == HelmChartContentLayerMediaType {
			return layer.Digest.String(), nil
		}
	}

	return "", errors.NotFoundf("%s:%s digest", name, version)
}

// ListChartVersions lists all versions of a chart
func (r *Repo) ListChartVersions(name string) ([]string, error) {
	// If entries is populated use it to list the chart versions
	// Otherwise, we need the charts list to be defined in the config file, retrieve all the tags for those chart names
	// and verify which tags are real charts by checking its mimeType.
	if _, ok := r.entries[name]; ok {
		return r.entries[name], nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), getTimeout)
	defer cancel()

	u := *r.url
	// Form API endpoint URL from repository URL
	u.Path = path.Join("v2", u.Path, name, "tags", "list")
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if r.username != "" && r.password != "" {
		req.SetBasicAuth(r.username, r.password)
	}
	client := http.DefaultClient
	if r.insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer resp.Body.Close()
	status := resp.StatusCode
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("unexpected response — %d %q — from %s", status, http.StatusText(status), u.String())
	}
	// Valid response codes from OCI registries are listed here:
	// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#endpoints
	switch status {
	case http.StatusNotFound:
		// If status not found 404, it could mean the asset has no release yet
		// TODO (tpizarro): Use the NotFound error instead of just returning nil and handle the case in the caller
		return []string{}, nil
	case http.StatusOK:
		//do nothing, just continue
	default:
		return nil, errors.Errorf("unexpected response — %d %q, %s — from %s", status, http.StatusText(status), string(body), u.String())
	}
	ot := &Tags{}
	if err := json.Unmarshal(body, ot); err != nil {
		return nil, err
	}
	chartTags := []string{}
	for _, tag := range ot.Tags {
		tm, err := r.getTagManifest(name, tag)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if tm.Config.MediaType == HelmChartConfigMediaType {
			chartTags = append(chartTags, tag)
		} else {
			klog.V(5).Infof("Skipping %q tag as it is not chart type", tag)
		}
	}
	return chartTags, nil
}

// GetDownloadURL returns the URL to download a chart
func (r *Repo) GetDownloadURL(name string, version string) (string, error) {
	digest, err := r.getChartDigest(name, version)
	if err != nil {
		return "", errors.Annotatef(err, "obtaining chart digest")
	}
	u := *r.url
	// Form API endpoint URL from repository URL
	u.Path = path.Join("v2", u.Path, name, "blobs", digest)
	return u.String(), nil
}

// Fetch fetches a chart
func (r *Repo) Fetch(name string, version string) (string, error) {
	remoteFilename := fmt.Sprintf("%s-%s.tgz", name, version)
	if r.cache.Has(remoteFilename) {
		return r.cache.Path(remoteFilename), nil
	}

	u, err := r.GetDownloadURL(name, version)
	if err != nil {
		return "", errors.Trace(err)
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", errors.Trace(err)
	}
	if r.username != "" && r.password != "" {
		req.SetBasicAuth(r.username, r.password)
	}

	reqID := utils.EncodeSha1(u + remoteFilename)
	klog.V(4).Infof("[%s] GET %q", reqID, u)
	client := http.DefaultClient
	if r.insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}
	res, err := client.Do(req)
	if err != nil {
		return "", errors.Annotatef(err, "fetching %s:%s chart", name, version)
	}
	defer res.Body.Close()

	status := res.StatusCode
	// Valid response codes from OCI registries are listed here:
	// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#endpoints
	switch status {
	case http.StatusOK:
		//do nothing, just continue
	default:
		bodyStr := utils.HTTPResponseBody(res)
		return "", errors.Errorf("unable to fetch %s:%s chart, got HTTP Status: %s, Resp: %v", name, version, res.Status, bodyStr)
	}
	klog.V(4).Infof("[%s] HTTP Status: %s", reqID, res.Status)

	w, err := r.cache.Writer(remoteFilename)
	if err != nil {
		return "", errors.Trace(err)
	}
	defer w.Close()
	if _, err := io.Copy(w, res.Body); err != nil {
		return "", errors.Trace(err)
	}

	return r.cache.Path(remoteFilename), nil
}

// Has checks if a repo has a specific chart
func (r *Repo) Has(name string, version string) (bool, error) {
	versions, err := r.ListChartVersions(name)
	if err != nil {
		return false, errors.Trace(err)
	}

	for _, v := range versions {
		if v == version {
			return true, nil
		}
	}
	return false, nil
}

// Upload uploads a chart to the repo
func (r *Repo) Upload(file string, metadata *chart.Metadata) error {
	name := metadata.Name
	version := metadata.Version
	// Invalidate cache to avoid inconsistency between an old cache result and
	// the chart repo
	if err := r.cache.Invalidate(filepath.Base(file)); err != nil {
		return errors.Trace(err)
	}

	f, err := os.Open(file)
	if err != nil {
		return errors.Trace(err)
	}
	defer f.Close()

	if err := r.cache.Store(f, filepath.Base(file)); err != nil {
		return errors.Trace(err)
	}

	memoryStore := content.NewMemoryStore()
	resolver := r.dockerResolver

	// Preparing layers
	var layers []ocispec.Descriptor
	fileName := filepath.Base(file)
	fileMediaType := HelmChartContentLayerMediaType
	fileBuffer, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	layers = append(layers, memoryStore.Add(fileName, fileMediaType, fileBuffer))

	// Preparing Oras config
	configBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	orasConfig := memoryStore.Add("", HelmChartConfigMediaType, configBytes)

	// Perform push
	chartRef := fmt.Sprintf("%s%s/%s:%s", r.url.Host, r.url.Path, name, version)
	_, err = oras.Push(orascontext.Background(), resolver, chartRef, memoryStore, layers, oras.WithConfig(orasConfig), oras.WithNameValidation(nil))
	if err != nil {
		return err
	}
	return nil
}

// GetChartDetails returns the details of a chart
func (r *Repo) GetChartDetails(name string, version string) (*types.ChartDetails, error) {
	digest, err := r.getChartDigest(name, version)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &types.ChartDetails{
		// OCI registries does not provide info about the publishing date in any API endpoint.
		// Therefore we cannot use the --from-date and we should publish everything.
		// Setting today's date so they get published.
		PublishedAt: time.Now(),
		Digest:      digest,
	}, nil
}

// Reload reloads the index
func (r *Repo) Reload() error {
	return errors.Errorf("reload method is not supported yet")
}

// SetEntries sets the entries of a repo
func (r *Repo) SetEntries(entries map[string][]string) {
	r.entries = entries
}

// OciReference contains info about an oci reference
type ociReference struct {
	path string
	tag  string
}

// ociReferenceExists checks if a given oci reference exists in the repository
// OCI endpoints: https://github.com/opencontainers/distribution-spec/blob/main/spec.md#endpoints
func ociReferenceExists(repoUrl *url.URL, ociRef, username, password string, insecure bool) (bool, error) {
	ociReference, err := name.ParseReference(ociRef)
	if err != nil {
		return false, errors.Trace(err)
	}
	ociReferenceIdentifier := ociReference.Identifier()
	parts := strings.Split(ociReference.Identifier(), ":")
	if len(parts) == 2 { // Identifier is a digest. We should remove sha256 prefix
		ociReferenceIdentifier = parts[1]
	}
	contextOciRef := fmt.Sprintf("%s://%s", repoUrl.Scheme, ociReference.Context())
	u, err := url.Parse(contextOciRef)
	if err != nil {
		return false, errors.Trace(err)
	}
	u.Path = path.Join("v2", u.Path, "manifests", ociReferenceIdentifier)

	req, err := http.NewRequest("HEAD", u.String(), nil)
	if err != nil {
		return false, errors.Trace(err)
	}
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")
	reqID := utils.EncodeSha1(u.String())
	klog.V(4).Infof("[%s] HEAD %q", reqID, u.String())
	client := http.DefaultClient
	if insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}
	res, err := client.Do(req)
	if err != nil {
		return false, errors.Annotatef(err, "checking existence of %q oci reference", ociRef)
	}
	defer res.Body.Close()

	status := res.StatusCode
	// Valid response codes from OCI registries are listed here:
	// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#endpoints
	klog.V(4).Infof("[%s] HTTP Status: %s", reqID, res.Status)
	switch status {
	case http.StatusOK:
		//do nothing, just continue
	case http.StatusNotFound:
		return false, nil
	default:
		bodyStr := utils.HTTPResponseBody(res)
		return false, errors.Errorf("unable to check if OCI index exists, got HTTP Status: %s, Resp: %v", res.Status, bodyStr)
	}

	return true, nil
}

// populateEntries populates the entries map with the info from the charts index
func populateEntries(chartsIndexRef string, resolver remotes.Resolver) (map[string][]string, error) {
	entries := make(map[string][]string)
	// Pull with ORAS lib
	filename := "asset-index.json"
	defer os.Remove(filename)
	ctx := context.Background()
	klog.Infof("Pulling index from %s...", chartsIndexRef)
	fileStore := content.NewFileStore("")
	defer fileStore.Close()
	_, _, err := oras.Pull(ctx, resolver, chartsIndexRef, fileStore)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Read downloaded file
	ociIndexBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Trace(err)
	}
	r := bytes.NewReader(ociIndexBytes)

	// Unmarshall index and populate entries
	ociIndex := &api.OciIndex{}
	err = pbjson.NewDecoder(r).Decode(ociIndex)
	for _, c := range ociIndex.Charts {
		for _, version := range c.Versions {
			entries[c.Name] = append(entries[c.Name], version.Version)
		}
	}
	return entries, nil
}

func newDockerResolver(u *url.URL, username, password string, insecure bool) remotes.Resolver {
	client := http.DefaultClient
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	opts := docker.ResolverOptions{
		Hosts: func(s string) ([]docker.RegistryHost, error) {
			return []docker.RegistryHost{
				{
					Authorizer: docker.NewDockerAuthorizer(
						docker.WithAuthCreds(func(s string) (string, string, error) {
							return username, password, nil
						})),
					Host:         u.Host,
					Scheme:       u.Scheme,
					Path:         "/v2",
					Capabilities: docker.HostCapabilityPull | docker.HostCapabilityResolve | docker.HostCapabilityPush,
					Client:       client,
				},
			}, nil
		},
	}

	return docker.NewResolver(opts)
}
