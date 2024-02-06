package syncer_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/pkg/client/repo"
	"github.com/bitnami/charts-syncer/pkg/client/repo/helmclassic"
	"github.com/bitnami/charts-syncer/pkg/syncer"
)

func getChartIndex(t *testing.T, name string, targetRepo *api.Target, tester repo.ClientTester) []*helmclassic.ChartVersion {
	// Check the chart really was added to the service's index.
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/charts/%s", tester.GetURL(), name), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(targetRepo.GetRepo().GetAuth().GetUsername(), targetRepo.GetRepo().GetAuth().GetPassword())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	charts := []*helmclassic.ChartVersion{}
	if err := json.NewDecoder(resp.Body).Decode(&charts); err != nil {
		t.Fatal(err)
	}
	return charts
}

func TestFakeSyncPendingCharts(t *testing.T) {
	testCases := []struct {
		desc           string
		entries        []string
		skippedEntries []string
		want           []string
	}{
		{
			desc:    "load apache and kafka",
			entries: []string{"apache", "kafka"},
			want:    []string{"apache-7.3.15.wrap.tgz", "kafka-10.3.3.wrap.tgz"},
		},
		{
			desc:           "skip apache",
			entries:        []string{"apache", "kafka"},
			skippedEntries: []string{"apache"},
			want:           []string{"kafka-10.3.3.wrap.tgz"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			dstTmp, err := os.MkdirTemp("", "charts-syncer-tests-dst-fake")
			if err != nil {
				t.Fatalf("error creating temporary folder: %v", err)
			}
			defer os.RemoveAll(dstTmp)

			s := syncer.NewFake(t, syncer.WithFakeSyncerDestination(dstTmp), syncer.WithFakeSkipCharts(tc.skippedEntries))

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
