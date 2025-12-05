package indexer

import (
	"context"
	"io"
	"os"

	"github.com/bitnami/charts-syncer/internal/indexer/api"
	"github.com/containerd/containerd/remotes"
	"github.com/distribution/reference"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/klog"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	oraserr "oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote"
)

// ociIndexerOpts are the options to configure the ociIndexer
type ociIndexerOpts struct {
	reference string
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

// ociIndexer is an OCI-based Indexer
type ociIndexer struct {
	reference  string
	repository *remote.Repository
}

// NewOciIndexer returns a new OCI-based indexer
func NewOciIndexer(opts ...OciIndexerOpt) (Indexer, error) {
	opt := &ociIndexerOpts{}
	for _, o := range opts {
		o(opt)
	}

	named, err := reference.ParseNormalizedNamed(opt.reference)
	if err != nil {
		return nil, err
	}

	repository, err := newRemoteRepository(named.Name(), opt.username, opt.password, opt.insecure)
	if err != nil {
		return nil, err
	}

	ind := &ociIndexer{
		reference:  opt.reference,
		repository: repository,
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
		err = os.RemoveAll(dir)
		if e == nil && err != nil {
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
	u := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err = u.Unmarshal(data, idx); err != nil {
		return nil, errors.Wrapf(err, "unable to parse index file")
	}
	return idx, nil
}

func (ind *ociIndexer) downloadIndex(ctx context.Context, rootPath string) (f string, e error) {
	// Pull index files from remote
	store, err := file.New(rootPath)
	if err != nil {
		return "", errors.Wrapf(err, "unable to create file store")
	}
	defer func() {
		err = store.Close()
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

	// Infer index filename from layer annotations and capture the layer descriptor
	var (
		indexFilename  string
		indexLayerDesc ocispec.Descriptor
	)
	opts := oras.DefaultCopyOptions
	opts.FindSuccessors = func(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		var successors []ocispec.Descriptor
		successors, err = content.Successors(ctx, fetcher, desc)
		if err != nil {
			return nil, err
		}
		var filtered []ocispec.Descriptor
		for _, s := range successors {
			// filter media type
			if s.MediaType == chartsIndexLayerMediaType || s.MediaType == chartsIndexConfigMediaType {
				filtered = append(filtered, s)
			}
			// set indexFilename and capture the layer descriptor
			if s.MediaType == chartsIndexLayerMediaType {
				indexLayerDesc = s
				if title, ok := s.Annotations["org.opencontainers.image.title"]; ok {
					indexFilename = title
				}
			}
		}
		return filtered, nil
	}
	_, err = oras.Copy(ctx, ind.repository, ind.reference, store, ind.reference, opts)
	if err != nil {
		if errors.Is(err, oraserr.ErrNotFound) {
			return "", errors.Wrap(ErrNotFound, err.Error())
		}
		return "", err
	}

	// Fallback to the default index filename if the layers don't specify it
	if indexFilename == "" {
		klog.Infof("Unable to find index filename: using default")
		indexFilename = defaultIndexFilename
	}

	// Read the layer content from the store
	reader, err := store.Fetch(ctx, indexLayerDesc)
	if err != nil {
		return "", errors.Wrapf(err, "unable to fetch index layer")
	}
	defer reader.Close()

	indexFile, err := os.Create(indexFilename)
	if err != nil {
		return "", err
	}
	defer indexFile.Close()

	_, err = io.Copy(indexFile, reader)
	if err != nil {
		return "", err
	}

	return indexFile.Name(), nil
}
