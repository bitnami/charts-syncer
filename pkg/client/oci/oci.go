package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/juju/errors"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/cache"
	"github.com/bitnami-labs/charts-syncer/internal/helmcli"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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

// Repo allows to operate a chart repository.
type Repo struct {
	url      *url.URL
	username string
	password string

	cache cache.Cacher
}

// Tags contains the tags for a specific OCI artifact
type Tags struct {
	Name string
	Tags []string
}

// KnownMediaTypes returns a list of layer mediaTypes that the Helm client knows about
func KnownMediaTypes() []string {
	return []string{
		HelmChartConfigMediaType,
		HelmChartContentLayerMediaType,
	}
}

// New creates a Repo object from an api.Repo object.
func New(repo *api.Repo, c cache.Cacher) (*Repo, error) {
	u, err := url.Parse(repo.GetUrl())
	if err != nil {
		return nil, errors.Trace(err)
	}

	return NewRaw(u, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword(), c)
}

// NewRaw creates a Repo object.
func NewRaw(u *url.URL, user string, pass string, c cache.Cacher) (*Repo, error) {
	return &Repo{url: u, username: user, password: pass, cache: c}, nil
}

// List lists all chart names in a repo
func (r *Repo) List() ([]string, error) {
	return nil, errors.Errorf("list method is not supported yet")
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
	resp, err := http.DefaultClient.Do(req)
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

	resp, err := http.DefaultClient.Do(req)
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
	u, err := r.GetDownloadURL(name, version)
	if err != nil {
		return "", errors.Trace(err)
	}

	remoteFilename := fmt.Sprintf("%s-%s.tgz", name, version)
	if r.cache.Has(remoteFilename) {
		return r.cache.Path(remoteFilename), nil
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
	client := &http.Client{}
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
func (r *Repo) Upload(file, name, version string) error {
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

	chartRef := fmt.Sprintf("%s%s/%s:%s", r.url.Host, r.url.Path, name, version)
	if err := helmcli.SaveOciChart(file, chartRef); err != nil {
		return errors.Trace(err)
	}
	if err := helmcli.PushToOCI(chartRef); err != nil {
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
