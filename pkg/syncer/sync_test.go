package syncer_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bitnami-labs/charts-syncer/pkg/syncer"
)

func TestSyncPendingCharts(t *testing.T) {
	testCases := []struct {
		desc    string
		entries []string
		want    []string
	}{
		{
			desc:    "load apache and kafka",
			entries: []string{"apache", "kafka"},
			want:    []string{"apache-7.3.15.tgz", "kafka-10.3.3.tgz", "zookeeper-5.14.3.tgz"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			dstTmp, err := ioutil.TempDir("", "charts-syncer-tests-dst-fake")
			if err != nil {
				t.Fatalf("error creating temporary folder: %v", err)
			}
			defer os.RemoveAll(dstTmp)

			s := syncer.NewFake(t, syncer.WithFakeSyncerDestination(dstTmp))

			if err := s.SyncPendingCharts(tc.entries...); err != nil {
				t.Error(err)
			}

			// We could use the fake client to obtain the list of synced charts.
			// However, as it is a fake implementation, let's rely on the target
			// directory.
			// If we change the implementation to be in-memory, this won't work.
			gotFiles, err := filepath.Glob(fmt.Sprintf("%s/*.tgz", dstTmp))
			if err != nil {
				t.Fatalf("error listing tgz files: %v", err)
			}

			var got []string
			for _, file := range gotFiles {
				got = append(got, filepath.Base(file))
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got: %v, want: %v\n", got, tc.want)
			}
		})
	}
}
