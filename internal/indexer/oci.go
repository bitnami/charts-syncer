package indexer

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bitnami-labs/charts-syncer/internal/indexer/api"
	"github.com/bitnami-labs/pbjson"
	"github.com/containerd/containerd/remotes"
	"github.com/juju/errors"
	"io/ioutil"
	"k8s.io/klog"
	"net/url"
	"oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"
	"os"
)

// OciIndexerOpts are the options to configure the OciIndexer
type OciIndexerOpts struct {
	Reference string
	URL       string
	Username  string
	Password  string
	Insecure  bool
}

// type OciIndexerOpt func(opts *OciIndexerOpts)

// OciIndexer is an OCI-based Indexer
type OciIndexer struct {
	reference string
	resolver  remotes.Resolver
}

// DefaultIndexName is the name for the OCI artifact with the index
const DefaultIndexName = "index"

// DefaultIndexTag is the tag for the OCI artifact with the index
const DefaultIndexTag = "latest"

// NewOciIndexer returns a new OCI-based indexer
func NewOciIndexer(opts *OciIndexerOpts) (*OciIndexer, error) {
	u, err := url.Parse(opts.URL)
	if err != nil {
		return nil, err
	}
	resolver := newDockerResolver(u, opts.Username, opts.Password, opts.Insecure)

	ind := &OciIndexer{
		reference: opts.Reference,
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
//nolint:funlen
func (ind *OciIndexer) Get(ctx context.Context) (idx *api.Index, e error) {
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

	// Pull index files from remote
	store := content.NewFileStore(dir)
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
		return nil, errors.Trace(err)
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

	indexFile := store.ResolvePath(indexFilename)
	data, err := ioutil.ReadFile(indexFile)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Populate and return index
	idx = &api.Index{}
	return idx, pbjson.NewDecoder(bytes.NewReader(data), pbjson.AllowUnknownFields(true)).Decode(idx)
}
