package syncer

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/local"
)

// NewFake returns a fake Syncer
func NewFake(t *testing.T, entries map[string][]string) *Syncer {
	srcTmp, err := ioutil.TempDir("", "charts-syncer-tests-src-fake")
	if err != nil {
		t.Fatalf("error creating temporary folder: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(srcTmp) })

	dstTmp, err := ioutil.TempDir("", "charts-syncer-tests-dst-fake")
	if err != nil {
		t.Fatalf("error creating temporary folder: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dstTmp) })

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
	dstCli, err := local.New(dstTmp)
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
