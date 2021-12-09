package indexer

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"k8s.io/klog"
	"net/url"
	"os"

	"github.com/bitnami-labs/charts-syncer/internal/indexer/api"
	"github.com/bitnami-labs/pbjson"
	"github.com/containerd/containerd/remotes"
	"github.com/juju/errors"
	"oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"
)

// ociIndexerOpts are the options to configure the ociIndexer
type ociIndexerOpts struct {
	reference string
	url      string
	username string
	password string
	insecure bool
}

// OciIndexerOpt allows setting configuration options
type OciIndexerOpt func(opts *ociIndexerOpts)

// WithIndexRef configures the charts index OCI reference instead of letting the library
// using the default host/index:latest one.
//
// 	opt := WithIndexRef("my.oci.domain/index:prod")
//
func WithIndexRef(r string) OciIndexerOpt {
	return func(opts *ociIndexerOpts) {
		opts.reference = r
	}
}

// WithBasicAuth configures basic authentication for the OCI host
//
// 	opt := WithBasicAuth("user", "pass")
//
func WithBasicAuth(user, pass string) OciIndexerOpt {
	return func(opts *ociIndexerOpts) {
		opts.username = user
		opts.password = pass
	}
}

// WithInsecure configures insecure connection
//
// 	opt := WithInsecure()
//
func WithInsecure() OciIndexerOpt {
	return func(opts *ociIndexerOpts) {
		opts.insecure = true
	}
}

// WithHost configures the OCI host
//
// 	opt := WithHost("my.oci.domain")
//
func WithHost(h string) OciIndexerOpt {
	return func(opts *ociIndexerOpts) {
		opts.url = h
	}
}

// ociIndexer is an OCI-based Indexer
type ociIndexer struct {
	reference string
	resolver  remotes.Resolver
}

// DefaultIndexName is the name for the OCI artifact with the index
const DefaultIndexName = "index"

// DefaultIndexTag is the tag for the OCI artifact with the index
const DefaultIndexTag = "latest"

// NewOciIndexer returns a new OCI-based indexer
func NewOciIndexer(opts ...OciIndexerOpt) (Indexer, error) {
	opt := &ociIndexerOpts{}
	for _, o := range opts {
		o(opt)
	}

	u, err := url.Parse(opt.url)
	if err != nil {
		return nil, err
	}
	resolver := newDockerResolver(u, opt.username, opt.password, opt.insecure)

	ind := &ociIndexer{
		reference: opt.reference,
		resolver:  resolver,
	}
	if ind.reference == "" {
		ind.reference = fmt.Sprintf("%s/%s:%s", u.Host, DefaultIndexName, DefaultIndexTag)
	}

	return ind, nil
}

// VacAssetIndexLayerMediaType is a media type used in VAC to store a JSON containing the index of
// charts and containers in a repository
const VacAssetIndexLayerMediaType = "application/vnd.vmware.tac.asset-index.layer.v1.json"
// VacAssetIndexConfigMediaType is a media type used in VAC for the configuration of the layer above
const VacAssetIndexConfigMediaType = "application/vnd.vmware.tac.index.config.v1+json"

// DefaultIndexFilename is the default filename used by the library to upload the index
const DefaultIndexFilename = "asset-index.json"

// Get implements Indexer
func (ind *ociIndexer) Get(ctx context.Context) (idx *api.Index, e error) {
	// Allocate folder for temporary downloads
	dir, err := os.MkdirTemp("", "indexer")
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer func() {
		err := os.RemoveAll(dir)
		if e == nil {
			e = err
		}
	}()

	indexFile, err := ind.downloadIndex(ctx, dir)
	if err != nil {
		return nil, errors.Trace(err)
	}

	data, err := ioutil.ReadFile(indexFile)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Populate and return index
	idx = &api.Index{}
	return idx, pbjson.NewDecoder(bytes.NewReader(data), pbjson.AllowUnknownFields(true)).Decode(idx)
}

func (ind *ociIndexer) downloadIndex(ctx context.Context, rootPath string) (f string, e error) {
	// Pull index files from remote
	store := content.NewFileStore(rootPath)
	defer func() {
		err := store.Close()
		// This library is buggy, and we need to check the error string too
		if e == nil && err != nil && err.Error() != "" {
			e = err
		}
	}()

	// Append key prefix for the known media types to the context
	// These prefixes are used for internal purposes in the ORAS library.
	// Otherwise, the library will print warnings.
	ctx = remotes.WithMediaTypeKeyPrefix(ctx, VacAssetIndexLayerMediaType, "layer-")
	ctx = remotes.WithMediaTypeKeyPrefix(ctx, VacAssetIndexConfigMediaType, "config-")

	opts := []oras.PullOpt{
		oras.WithAllowedMediaType(VacAssetIndexLayerMediaType, VacAssetIndexConfigMediaType),
		// The index artifact has no title
		oras.WithPullEmptyNameAllowed(),
	}
	_, layers, err := oras.Pull(ctx, ind.resolver, ind.reference, store, opts...)
	if err != nil {
		return "", errors.Trace(err)
	}

	// Infer index filename from layer annotations
	var indexFilename string
	for _, layer := range layers {
		//nolint:gocritic
		switch layer.MediaType {
		case VacAssetIndexLayerMediaType:
			indexFilename = layer.Annotations["org.opencontainers.image.title"]
		}
	}
	// Fallback to the default index filename if the layers don't specify it
	if indexFilename == "" {
		klog.Infof("unable to find index filename: using default")
		indexFilename = DefaultIndexFilename
	}

	return store.ResolvePath(indexFilename), nil
}
