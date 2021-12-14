package syncer

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/bitnami-labs/charts-syncer/pkg/client/repo/local"

	"github.com/bitnami-labs/charts-syncer/api"
)

// FakeSyncerOpts allows to configure a Fake syncer.
type FakeSyncerOpts struct {
	Destination string
}

// FakeSyncerOption is an option value used to create a new fake syncer instance.
type FakeSyncerOption func(*FakeSyncerOpts)

// WithFakeSyncerDestination configures a destination directory
func WithFakeSyncerDestination(dir string) FakeSyncerOption {
	return func(s *FakeSyncerOpts) {
		s.Destination = dir
	}
}

// NewFake returns a fake Syncer
func NewFake(t *testing.T, opts ...FakeSyncerOption) *Syncer {
	sopts := &FakeSyncerOpts{}
	for _, o := range opts {
		o(sopts)
	}

	srcTmp, err := ioutil.TempDir("", "charts-syncer-tests-src-fake")
	if err != nil {
		t.Fatalf("error creating temporary folder: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(srcTmp) })

	if sopts.Destination == "" {
		dstTmp, err := ioutil.TempDir("", "charts-syncer-tests-dst-fake")
		if err != nil {
			t.Fatalf("error creating temporary folder: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(dstTmp) })
		sopts.Destination = dstTmp
	}

	// Copy all testdata tgz files to the source temporary folder
	// We are not adding charts in the entries only to avoid specifying
	// the dependencies
	matches, err := filepath.Glob("../../testdata/*.tgz")
	if err != nil {
		t.Fatalf("error listing tgz files: %v", err)
	}
	for _, sourceFile := range matches {
		input, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			t.Fatalf("error reading %q chart: %v", sourceFile, err)
		}

		dstFile := path.Join(srcTmp, filepath.Base(sourceFile))
		if err = ioutil.WriteFile(dstFile, input, 0644); err != nil {
			t.Fatalf("error copying chart to %q: %v", dstFile, err)
		}
	}

	srcCli, err := local.New(srcTmp)
	if err != nil {
		t.Fatalf("error creating source client: %v", err)
	}
	dstCli, err := local.New(sopts.Destination)
	if err != nil {
		t.Fatalf("error creating target client: %v", err)
	}

	return &Syncer{
		source: &api.SourceRepo{},
		target: &api.TargetRepo{},
		cli: &Clients{
			src: srcCli,
			dst: dstCli,
		},
	}
}
