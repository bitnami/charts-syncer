package indexer

import (
	"bytes"
	"context"
	"net/url"
	"os"

	"github.com/bitnami-labs/pbjson"
	"github.com/bitnami/charts-syncer/internal/indexer/api"
	containerderrs "github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/remotes"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"k8s.io/klog"
	"oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"
)

// ociIndexerOpts are the options to configure the ociIndexer
type ociIndexerOpts struct {
	reference string
	url       string
	username  string
	password  string
	insecure  bool
}

// OciIndexerOpt allows setting configuration options
type OciIndexerOpt func(opts *ociIndexerOpts)

// WithIndexRef configures the charts index OCI reference instead of letting the library
// using the default host/index:latest one.
//
//	opt := WithIndexRef("my.oci.domain/index:prod")
func WithIndexRef(r string) OciIndexerOpt {
	return func(opts *ociIndexerOpts) {
		opts.reference = r
	}
}

// WithBasicAuth configures basic authentication for the OCI host
//
//	opt := WithBasicAuth("user", "pass")
func WithBasicAuth(user, pass string) OciIndexerOpt {
	return func(opts *ociIndexerOpts) {
		opts.username = user
		opts.password = pass
	}
}

// WithInsecure configures insecure connection
//
//	opt := WithInsecure()
func WithInsecure() OciIndexerOpt {
	return func(opts *ociIndexerOpts) {
		opts.insecure = true
	}
}

// WithHost configures the OCI host
//
//	opt := WithHost("my.oci.domain")
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

// NewOciIndexer returns a new OCI-based indexer
func NewOciIndexer(opts ...OciIndexerOpt) (Indexer, error) {
	opt := &ociIndexerOpts{}
	for _, o := range opts {
		o(opt)
	}

	u, err := url.Parse(opt.url)
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidArgument, "invalid OCI host URL: %+v", err)
	}
	resolver := newDockerResolver(u, opt.username, opt.password, opt.insecure)

	ind := &ociIndexer{
		reference: opt.reference,
		resolver:  resolver,
	}

	return ind, nil
}

// chartsIndexLayerMediaType is a media type used to store a JSON containing the index of
// charts in a repository
const chartsIndexLayerMediaType = "application/vnd.vmware.charts.index.layer.v1+json"

// chartsIndexConfigMediaType is a media type used for the configuration of the layer above
const chartsIndexConfigMediaType = "application/vnd.vmware.charts.index.config.v1+json"

// defaultIndexFilename is the default filename used by the library to upload the index
const defaultIndexFilename = "charts-index.json"

// Get implements Indexer
func (ind *ociIndexer) Get(ctx context.Context) (idx *api.Index, e error) {
	// Allocate folder for temporary downloads
	dir, err := os.MkdirTemp("", "indexer")
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create temporary indexer directory")
	}
	defer func() {
		err := os.RemoveAll(dir)
		if e == nil {
			e = err
		}
	}()

	indexFile, err := ind.downloadIndex(ctx, dir)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to download index")
	}

	data, err := os.ReadFile(indexFile)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read index file")
	}

	// Populate and return index
	idx = &api.Index{}
	if err := pbjson.NewDecoder(bytes.NewReader(data), pbjson.AllowUnknownFields(true)).Decode(idx); err != nil {
		return nil, errors.Wrapf(err, "unable to parse index file")
	}
	return idx, nil
}

func (ind *ociIndexer) downloadIndex(ctx context.Context, rootPath string) (f string, e error) {
	// Pull index files from remote
	store := content.NewFile(rootPath)
	defer func() {
		err := store.Close()
		// This library is buggy, and we need to check the error string too
		// https://github.com/oras-project/oras-go/issues/84
		if e == nil && err != nil && err.Error() != "" {
			e = err
		}
	}()

	// Append key prefix for the known media types to the context
	// These prefixes are used for internal purposes in the ORAS library.
	// Otherwise, the library will print warnings.
	ctx = remotes.WithMediaTypeKeyPrefix(ctx, chartsIndexLayerMediaType, "layer-")
	ctx = remotes.WithMediaTypeKeyPrefix(ctx, chartsIndexConfigMediaType, "config-")

	// Infer index filename from layer annotations
	var indexFilename string
	opts := []oras.CopyOpt{
		oras.WithAllowedMediaType(chartsIndexLayerMediaType, chartsIndexConfigMediaType),
		// The index artifact has no title
		oras.WithPullEmptyNameAllowed(),
		oras.WithLayerDescriptors(func(layers []ocispec.Descriptor) {
			for _, layer := range layers {
				switch layer.MediaType {
				case chartsIndexLayerMediaType:
					indexFilename = layer.Annotations["org.opencontainers.image.title"]
				}
			}
		}),
	}
	_, err := oras.Copy(ctx, ind.resolver, ind.reference, store, ind.reference, opts...)
	if err != nil {
		if containerderrs.IsNotFound(err) {
			return "", errors.Wrap(ErrNotFound, err.Error())
		}
		return "", err
	}

	// Fallback to the default index filename if the layers don't specify it
	if indexFilename == "" {
		klog.Infof("Unable to find index filename: using default")
		indexFilename = defaultIndexFilename
	}

	return store.ResolvePath(indexFilename), nil
}
