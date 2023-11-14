package oci

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/juju/errors"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/klog"
	"oras.land/oras-go/pkg/content"
	orascontext "oras.land/oras-go/pkg/context"
	"oras.land/oras-go/pkg/oras"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/internal/cache"
	"github.com/bitnami/charts-syncer/internal/indexer"
	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/bitnami/charts-syncer/pkg/client/types"
)

const (
	getTimeout = 1 * time.Minute
	// HelmChartConfigMediaType is the reserved media type for the Helm chart manifest config
	HelmChartConfigMediaType = "application/vnd.cncf.helm.config.v1+json"
	// HelmChartContentLayerMediaType is the reserved media type for Helm chart package content
	HelmChartContentLayerMediaType = "application/vnd.cncf.helm.chart.content.v1.tar+gzip"
	// HelmChartContentLayerMediaTypeDeprecated is the (deprecated) reserved media type for Helm
	// chart package content
	HelmChartContentLayerMediaTypeDeprecated = "application/tar+gzip"
	// ImageManifestMediaType is the reserved media type for OCI manifests
	ImageManifestMediaType = "application/vnd.oci.image.manifest.v1+json"
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
	// Init entries
	entries, err := populateEntries(repo)
	if err != nil {
		return nil, errors.Trace(err)
	}

	u, err := url.Parse(repo.GetUrl())
	if err != nil {
		return nil, errors.Trace(err)
	}
	resolver := newDockerResolver(u, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword(), insecure)

	return NewRaw(u, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword(), c, insecure, entries, resolver)
}

// NewRaw creates a Repo object.
func NewRaw(u *url.URL, user string, pass string, c cache.Cacher, insecure bool, entries map[string][]string, resolver remotes.Resolver) (*Repo, error) {
	return &Repo{url: u, username: user, password: pass, cache: c, insecure: insecure, entries: entries, dockerResolver: resolver}, nil
}

// List lists all chart names in a repo
func (r *Repo) List() ([]string, error) {
	// If entries is not populated, it means we couldn't load any index file, so we need the charts filter in the
	// configuration file. The List() caller will handle this case
	if len(r.entries) == 0 {
		return []string{}, nil
	}

	names := make([]string, 0, len(r.entries))
	for entry := range r.entries {
		names = append(names, entry)
	}
	return names, nil
}

// getTagManifest returns the manifests of a published tag
func (r *Repo) getTagManifest(chartName, version string) (*ocispec.Manifest, error) {
	u := *r.url
	u.Path = path.Join(u.Path, "/", chartName)

	repo, err := name.ParseReference(u.Host + u.Path + ":" + version)
	if err != nil {
		return nil, errors.Errorf("failed parsing OCI reference: %s", err)
	}

	opts := []remote.Option{}
	if r.username != "" && r.password != "" {
		opts = append(opts, remote.WithAuth(&authn.Basic{
			Username: r.username,
			Password: r.password,
		}))
	}
	if r.insecure {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		opts = append(opts, remote.WithTransport(transport))
	}

	image, err := remote.Image(repo, opts...)
	if err != nil {
		return nil, errors.Errorf("failed to fetch %q manifest: %v", repo, err)
	}

	body, err := image.RawManifest()
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
		if isHelmChartContentLayerMediaType(layer.MediaType) {
			return layer.Digest.String(), nil
		}
	}

	return "", errors.NotFoundf("%s:%s digest", name, version)
}

// ListChartVersions lists all versions of a chart
func (r *Repo) ListChartVersions(chartName string) ([]string, error) {
	// If entries is populated use it to list the chart versions
	// Otherwise, we need the charts list to be defined in the config file, retrieve all the tags for those chart names
	// and verify which tags are real charts by checking its mimeType.
	if _, ok := r.entries[chartName]; ok {
		return r.entries[chartName], nil
	}

	u := *r.url
	u.Path = path.Join(u.Path, "/", chartName)

	repo, err := name.NewRepository(u.Host + u.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repo %v", err)
	}

	opts := []remote.Option{}
	if r.username != "" && r.password != "" {
		opts = append(opts, remote.WithAuth(&authn.Basic{
			Username: r.username,
			Password: r.password,
		}))
	}
	if r.insecure {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		opts = append(opts, remote.WithTransport(transport))
	}

	tags, err := remote.List(repo, opts...)
	if err != nil {
		return nil, errors.Errorf("failed to fetch tags for %q: %v", repo, err)
	}

	chartTags := []string{}
	for _, tag := range tags {
		tm, err := r.getTagManifest(chartName, tag)
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
func (r *Repo) Fetch(chartName string, version string) (string, error) {
	id := fmt.Sprintf("%s-%s.tgz", chartName, version)
	if r.cache.Has(id) {
		return r.cache.Path(id), nil
	}

	ref, err := name.ParseReference(
		fmt.Sprintf("%s:%s", path.Join(strings.TrimPrefix(r.url.String(), fmt.Sprintf("%s://", r.url.Scheme)), chartName), version))
	if err != nil {
		return "", errors.Errorf("failed parsing OCI reference: %s", err)
	}

	opts := []remote.Option{}
	if r.username != "" && r.password != "" {
		opts = append(opts, remote.WithAuth(&authn.Basic{
			Username: r.username,
			Password: r.password,
		}))
	}
	if r.insecure {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		opts = append(opts, remote.WithTransport(transport))
	}

	img, err := remote.Image(ref, opts...)
	if err != nil {
		return "", errors.Annotatef(err, "failed checking remote: %s", ref)
	}

	w, err := r.cache.Writer(id)
	if err != nil {
		return "", errors.Annotatef(err, "fetching %q chart", ref)
	}

	layers, err := img.Layers()
	if err != nil {
		return "", errors.Annotatef(err, "fetching %q chart", ref)
	}

	for _, l := range layers {
		t, err := l.MediaType()
		if err != nil {
			// Invalidate the cache
			r.cache.Invalidate(id)
			return "", errors.Annotatef(err, "fetching %q chart", ref)
		}
		// https://helm.sh/docs/topics/registries/#helm-chart-manifest
		if isHelmChartContentLayerMediaType(string(t)) {
			c, err := l.Compressed()
			_, err = io.Copy(w, c)
			if err != nil {
				// Invalidate the cache
				r.cache.Invalidate(id)
				return "", errors.Annotatef(err, "fetching %q chart", ref)
			}
		}
	}

	if err := w.Close(); err != nil {
		// Invalidate the cache
		r.cache.Invalidate(id)
		return "", errors.Annotatef(err, "fetching %q chart", ref)
	}

	return r.cache.Path(id), nil
}

// Has checks if a repo has a specific chart
func (r *Repo) Has(chartName string, version string) (bool, error) {
	ref, err := name.ParseReference(
		fmt.Sprintf("%s:%s", path.Join(strings.TrimPrefix(r.url.String(), fmt.Sprintf("%s://", r.url.Scheme)), chartName), version))
	if err != nil {
		return false, errors.Errorf("failed parsing OCI reference: %s", err)
	}

	opts := []remote.Option{}
	if r.username != "" && r.password != "" {
		opts = append(opts, remote.WithAuth(&authn.Basic{
			Username: r.username,
			Password: r.password,
		}))
	}
	if r.insecure {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		opts = append(opts, remote.WithTransport(transport))
	}

	_, err = remote.Head(ref, opts...)
	if err != nil {
		if terr, ok := err.(*transport.Error); ok {
			if terr.StatusCode == http.StatusNotFound {
				return false, nil
			}
		}
		return false, errors.Errorf("failed checking remote: %s", err)
	}

	return true, nil
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

	memoryStore := content.NewMemory()
	resolver := r.dockerResolver

	// Preparing layers
	fileName := filepath.Base(file)
	fileMediaType := HelmChartContentLayerMediaType
	fileBuffer, err := os.ReadFile(file)
	if err != nil {
		return errors.Trace(err)
	}
	blobDesc, err := memoryStore.Add(fileName, fileMediaType, fileBuffer)
	if err != nil {
		return errors.Trace(err)
	}

	// Preparing Oras config
	configBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	configDesc, err := memoryStore.Add("", HelmChartConfigMediaType, configBytes)
	if err != nil {
		return errors.Trace(err)
	}

	manifest, manifestDesc, err := content.GenerateManifest(&configDesc, nil, blobDesc)
	if err != nil {
		return errors.Trace(err)
	}
	chartRef := fmt.Sprintf("%s%s/%s:%s", r.url.Host, r.url.Path, name, version)
	if err := memoryStore.StoreManifest(chartRef, manifestDesc, manifest); err != nil {
		return errors.Trace(err)
	}

	// Perform push
	copyOpts := []oras.CopyOpt{
		oras.WithAllowedMediaType(HelmChartConfigMediaType, HelmChartContentLayerMediaType),
		oras.WithNameValidation(nil),
	}
	if _, err := oras.Copy(orascontext.Background(), memoryStore, chartRef, resolver, chartRef, copyOpts...); err != nil {
		return errors.Trace(err)
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

func isHelmChartContentLayerMediaType(t string) bool {
	if t == HelmChartContentLayerMediaType {
		return true
	}
	if t == HelmChartContentLayerMediaTypeDeprecated {
		return true
	}
	return false
}

// ociReferenceExists checks if a given oci reference exists in the repository
func ociReferenceExists(ociRef, username, password string) (bool, error) {
	ociReference, err := name.ParseReference(ociRef)
	if err != nil {
		return false, errors.Trace(err)
	}
	authOptions := authn.Basic{Username: username, Password: password}
	opt := []remote.Option{
		remote.WithAuth(&authOptions),
	}
	_, err = remote.Head(ociReference, opt...)
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return false, nil
		} else {
			return false, errors.Trace(err)
		}
	}
	return true, nil
}

// populateEntries populates the entries map with the info from the charts index
func populateEntries(repo *api.Repo) (map[string][]string, error) {
	if repo.GetDisableChartsIndex() {
		return make(map[string][]string), nil
	}

	klog.Infof("Attempting to retrieve remote index...")
	ind, err := indexer.NewOciIndexer(
		indexer.WithHost(repo.GetUrl()),
		indexer.WithBasicAuth(repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword()),
		indexer.WithIndexRef(repo.GetChartsIndex()),
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ociIndex, err := ind.Get(ctx)
	if err != nil {
		// Since we automatically attempt to retrieve the index, let's not fail if not exists
		if indexer.IsNotFound(err) {
			klog.Warningf("The remote index does not exist yet. This process might be slow.")
			return make(map[string][]string), nil
		}

		return nil, errors.Trace(err)
	}

	entries := make(map[string][]string, len(ociIndex.GetEntries()))
	for _, c := range ociIndex.GetEntries() {
		for _, v := range c.GetVersions() {
			entries[v.Name] = append(entries[v.Name], v.Version)
		}
	}
	return entries, nil
}

func newDockerResolver(u *url.URL, username, password string, insecure bool) remotes.Resolver {
	client := utils.DefaultClient
	if insecure {
		client = utils.InsecureClient
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
