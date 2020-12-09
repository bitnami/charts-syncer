package cache

import (
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/juju/errors"
	"k8s.io/klog"
)

// Cache implements a Cacher using the local filesystem.
type Cache struct {
	id  string
	dir string
}

// New returns a new Cache object.
//
// The internal cache directory path is computed based on the provided
// workdir. The workdir will contain a hashed subfolder if a unique identifier
// is also provided.
func New(workdir string, id string) (*Cache, error) {
	if workdir == "" {
		return nil, errors.New("workdir was not provided")
	}

	hashID := utils.EncodeSha1(id)
	c := &Cache{id: hashID}

	c.dir = workdir
	if id != "" {
		c.dir = path.Join(workdir, hashID)
	}

	klog.V(4).Infof("Allocating cache dir: %q", c.dir)
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return nil, errors.Trace(err)
	}

	return c, nil
}

// Path returns the path to a cached file named by filename
func (c *Cache) Path(filename string) string {
	return path.Join(c.dir, filename)
}

// Has returns whether the cache contains the file named by filename
func (c *Cache) Has(filename string) bool {
	ok, _ := utils.FileExists(c.Path(filename))
	klog.V(4).Infof("cache hit { op:has, id:%s, filename:%s }", c.id, filename)
	return ok
}

// Read reads a cached file named by filename and writes the contents to the
// provider w writer.
//
// If the file does not exist, Read returns an error that satisfies
// errors.IsNotFound().
func (c *Cache) Read(w io.Writer, filename string) error {
	if !c.Has(filename) {
		return errors.NotFoundf("cache { id:%s, filename:%s }", c.id, filename)
	}
	data, err := ioutil.ReadFile(c.Path(filename))
	if err != nil {
		return errors.Annotatef(err, "reading %q from the cache", filename)
	}
	if _, err := w.Write(data); err != nil {
		return errors.Annotatef(err, "reading %q from the cache", filename)
	}
	klog.V(4).Infof("cache hit { op:read, id:%s, filename:%s }", c.id, filename)
	return nil
}

// Store stores the provided bytes in a cache file named by filename
//
// If the file exists, Store returns an error that satisfies
// errors.IsAlreadyExists(). Otherwise, Store creates it with permissions perm
func (c *Cache) Store(r io.Reader, filename string) error {
	if c.Has(filename) {
		return errors.AlreadyExistsf("cache { id:%s, filename:%s }", c.id, filename)
	}

	f, err := os.Create(c.Path(filename))
	if err != nil {
		return errors.Annotatef(err, "storing %q in the cache", filename)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return errors.Annotatef(err, "storing %q in the cache", filename)
	}

	klog.V(4).Infof("cache hit { op:store, id:%s, filename:%s }", c.id, filename)
	return nil
}

// Writer returns a io.Writer that writes to a cache file named by filename
func (c *Cache) Writer(filename string) (*os.File, error) {
	if c.Has(filename) {
		return nil, errors.AlreadyExistsf("cache { id:%s, filename:%s }", c.id, filename)
	}

	klog.V(4).Infof("cache hit { op:write, id:%s, filename:%s }", c.id, filename)
	return os.Create(c.Path(filename))
}

// Invalidate invalidates a cache file named by filename
func (c *Cache) Invalidate(filename string) error {
	if !c.Has(filename) {
		return nil
	}
	if err := os.Remove(c.Path(filename)); err != nil {
		return errors.Annotatef(err, "invalidating %q in the cache", filename)
	}
	klog.V(4).Infof("cache hit { op:invalidate, id:%s, filename:%s }", c.id, filename)
	return nil
}
