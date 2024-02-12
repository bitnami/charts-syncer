package syncer

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/bitnami/charts-syncer/api"
	localSource "github.com/bitnami/charts-syncer/pkg/client/source/local"
	localTarget "github.com/bitnami/charts-syncer/pkg/client/target/local"

	"github.com/vmware-labs/distribution-tooling-for-helm/pkg/log/silent"
)

// FakeSyncerOpts allows to configure a Fake syncer.
type FakeSyncerOpts struct {
	Destination string
	skipCharts  []string
}

// FakeSyncerOption is an option value used to create a new fake syncer instance.
type FakeSyncerOption func(*FakeSyncerOpts)

// WithFakeSyncerDestination configures a destination directory
func WithFakeSyncerDestination(dir string) FakeSyncerOption {
	return func(s *FakeSyncerOpts) {
		s.Destination = dir
	}
}

// WithFakeSkipCharts configures the syncer to skip an explicit list of chart names
// from the source chart repos.
func WithFakeSkipCharts(charts []string) FakeSyncerOption {
	return func(s *FakeSyncerOpts) {
		s.skipCharts = charts
	}
}

// NewFake returns a fake Syncer
func NewFake(t *testing.T, opts ...FakeSyncerOption) *Syncer {
	sopts := &FakeSyncerOpts{}
	for _, o := range opts {
		o(sopts)
	}

	srcTmp, err := os.MkdirTemp("", "charts-syncer-tests-src-fake")
	if err != nil {
		t.Fatalf("error creating temporary folder: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(srcTmp) })

	if sopts.Destination == "" {
		dstTmp, err := os.MkdirTemp("", "charts-syncer-tests-dst-fake")
		if err != nil {
			t.Fatalf("error creating temporary folder: %v", err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(dstTmp) })
		sopts.Destination = dstTmp
	}

	// Copy all testdata tgz files to the source temporary folder
	// We are not adding charts in the entries only to avoid specifying
	// the dependencies
	matches, err := filepath.Glob("../../testdata/*.wrap.tgz")
	if err != nil {
		t.Fatalf("error listing tgz files: %v", err)
	}
	for _, sourceFile := range matches {
		input, err := os.ReadFile(sourceFile)
		if err != nil {
			t.Fatalf("error reading %q chart: %v", sourceFile, err)
		}

		dstFile := path.Join(srcTmp, filepath.Base(sourceFile))
		if err = os.WriteFile(dstFile, input, 0644); err != nil {
			t.Fatalf("error copying chart to %q: %v", dstFile, err)
		}
	}

	srcCli, err := localSource.New(srcTmp)
	if err != nil {
		t.Fatalf("error creating source client: %v", err)
	}
	dstCli, err := localTarget.New(sopts.Destination)
	if err != nil {
		t.Fatalf("error creating target client: %v", err)
	}

	return &Syncer{
		source: &api.Source{},
		target: &api.Target{},
		cli: &Clients{
			src: srcCli,
			dst: dstCli,
		},
		skipCharts: sopts.skipCharts,
		logger:     silent.NewSectionLogger(),
	}
}
