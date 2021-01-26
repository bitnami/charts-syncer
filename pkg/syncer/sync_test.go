package syncer_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/chartrepotest"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client/core"
	"github.com/bitnami-labs/charts-syncer/pkg/syncer"
)

func TestFakeSyncPendingCharts(t *testing.T) {
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

func TestSyncPendingChartsChartMuseum(t *testing.T) {
	testCases := []struct {
		desc       string
		sourceRepo *api.SourceRepo
		targetRepo *api.TargetRepo
		entries    []string
		want       []string
	}{
		{
			desc: "sync etcd and common",
			sourceRepo: &api.SourceRepo{
				Repo: &api.Repo{
					Kind: api.Kind_CHARTMUSEUM,
					Auth: &api.Auth{
						Username: "user",
						Password: "password",
					},
				},
			},
			targetRepo: &api.TargetRepo{
				Repo: &api.Repo{
					Kind: api.Kind_CHARTMUSEUM,
					Auth: &api.Auth{
						Username: "user",
						Password: "password",
					},
				},
			},
			entries: []string{"etcd", "common"},
			want:    []string{"common-0.2.1.tgz", "etcd-4.8.0.tgz"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Create temp folder and copy index.yaml
			dstTmp, err := ioutil.TempDir("", "charts-syncer-tests-index-fake")
			if err != nil {
				t.Fatalf("error creating temporary folder: %v", err)
			}
			defer os.RemoveAll(dstTmp)
			dstIndex := filepath.Join(dstTmp, "index.yaml")
			if err := utils.CopyFile(dstIndex, "../../testdata/etcd-index.yaml"); err != nil {
				t.Fatal(err)
			}

			// Create source and target testers
			sTester, sCleanup, err := core.NewClientV2Tester(t, tc.sourceRepo.GetRepo(), false, dstIndex)
			if err != nil {
				t.Fatal(err)
			}
			tTester, tCleanup, err := core.NewClientV2Tester(t, tc.targetRepo.GetRepo(), true, "")
			if err != nil {
				t.Fatal(err)
			}
			defer sCleanup()
			defer tCleanup()

			// Replace URL with source url
			read, err := ioutil.ReadFile(dstIndex)
			if err != nil {
				t.Fatal(err)
			}
			newContents := strings.Replace(string(read), "https://fake.chart.repo.com/testing", fmt.Sprintf("%s%s", sTester.GetURL(), "/charts"), -1)
			if err = ioutil.WriteFile(dstIndex, []byte(newContents), 0); err != nil {
				t.Fatal(err)
			}

			// Update source repo url
			tc.sourceRepo.Repo.Url = sTester.GetURL()
			// Update target repo url
			tc.targetRepo.Repo.Url = tTester.GetURL()

			// Create new syncer
			s, err := syncer.New(tc.sourceRepo, tc.targetRepo)
			if err != nil {
				t.Fatal(err)
			}

			if err := s.SyncPendingCharts(tc.entries...); err != nil {
				t.Error(err)
			}

			// Check the chart really was added to the service's index.
			req, err := http.NewRequest("GET", tTester.GetURL()+"/api/charts/etcd", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth(tc.targetRepo.Repo.Auth.Username, tc.targetRepo.Repo.Auth.Password)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			charts := []*chartrepotest.ChartVersion{}
			if err := json.NewDecoder(resp.Body).Decode(&charts); err != nil {
				t.Fatal(err)
			}
			if got, want := len(charts), 1; got != want {
				t.Fatalf("got: %q, want: %q", got, want)
			}
			if got, want := charts[0].Name, "etcd"; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
			if got, want := charts[0].Version, "4.8.0"; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
