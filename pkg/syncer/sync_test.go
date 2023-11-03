package syncer_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/internal/utils"
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
			// zookeeper is a dependency
			want: []string{"apache-7.3.15.tgz", "kafka-10.3.3.tgz", "zookeeper-5.14.3.tgz"},
		},
		{
			desc:           "skip apache",
			entries:        []string{"apache", "kafka"},
			skippedEntries: []string{"apache"},
			want:           []string{"kafka-10.3.3.tgz", "zookeeper-5.14.3.tgz"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			dstTmp, err := ioutil.TempDir("", "charts-syncer-tests-dst-fake")
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

func TestSyncPendingChartsChartMuseum(t *testing.T) {
	testCases := []struct {
		desc              string
		sourceRepo        *api.Source
		targetRepo        *api.Target
		skipDependencies  bool
		latestVersionOnly bool
		entries           []string
		requiredCharts    []string
		want              []*helmclassic.ChartVersion
	}{
		{
			desc: "sync etcd and common",
			sourceRepo: &api.Source{
				Spec: &api.Source_Repo{
					Repo: &api.Repo{
						Kind: api.Kind_CHARTMUSEUM,
						Auth: &api.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			targetRepo: &api.Target{
				Spec: &api.Target_Repo{
					Repo: &api.Repo{
						Kind: api.Kind_CHARTMUSEUM,
						Auth: &api.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			skipDependencies: false,
			entries:          []string{"common", "etcd"},
			requiredCharts:   []string{"common", "etcd"},
			want: []*helmclassic.ChartVersion{
				{
					Name:    "common",
					Version: "1.10.1",
					URLs:    []string{"charts/common-1.10.1.tgz"},
				},
				{
					Name:    "common",
					Version: "1.10.0",
					URLs:    []string{"charts/common-1.10.0.tgz"},
				},
				{
					Name:    "etcd",
					Version: "4.8.0",
					URLs:    []string{"charts/etcd-4.8.0.tgz"},
				},
			},
		},
		{
			desc: "sync kafka plus dependencies",
			sourceRepo: &api.Source{
				Spec: &api.Source_Repo{
					Repo: &api.Repo{
						Kind: api.Kind_CHARTMUSEUM,
						Auth: &api.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			targetRepo: &api.Target{
				Spec: &api.Target_Repo{
					Repo: &api.Repo{
						Kind: api.Kind_CHARTMUSEUM,
						Auth: &api.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			skipDependencies: false,
			entries:          []string{"kafka"},
			requiredCharts:   []string{"common", "kafka", "zookeeper"},
			want: []*helmclassic.ChartVersion{
				{
					Name:    "common",
					Version: "1.10.1",
					URLs:    []string{"charts/common-1.10.1.tgz"},
				},
				{
					Name:    "common",
					Version: "1.10.0",
					URLs:    []string{"charts/common-1.10.0.tgz"},
				},
				{
					Name:    "kafka",
					Version: "14.7.0",
					URLs:    []string{"charts/kafka-14.7.0.tgz"},
				},
				{
					Name:    "zookeeper",
					Version: "7.4.11",
					URLs:    []string{"charts/zookeeper-7.4.11.tgz"},
				},
			},
		}, {
			desc: "sync kafka (skipping dependencies)",
			sourceRepo: &api.Source{
				Spec: &api.Source_Repo{
					Repo: &api.Repo{
						Kind: api.Kind_CHARTMUSEUM,
						Auth: &api.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			targetRepo: &api.Target{
				Spec: &api.Target_Repo{
					Repo: &api.Repo{
						Kind: api.Kind_CHARTMUSEUM,
						Auth: &api.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			skipDependencies: true,
			entries:          []string{"kafka"},
			requiredCharts:   []string{"common", "kafka", "zookeeper"},
			want: []*helmclassic.ChartVersion{
				{
					Name:    "kafka",
					Version: "14.7.0",
					URLs:    []string{"charts/kafka-14.7.0.tgz"},
				},
			},
		}, {
			desc: "sync common (all versions)",
			sourceRepo: &api.Source{
				Spec: &api.Source_Repo{
					Repo: &api.Repo{
						Kind: api.Kind_CHARTMUSEUM,
						Auth: &api.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			targetRepo: &api.Target{
				Spec: &api.Target_Repo{
					Repo: &api.Repo{
						Kind: api.Kind_CHARTMUSEUM,
						Auth: &api.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			entries:        []string{"common"},
			requiredCharts: []string{"common"},
			want: []*helmclassic.ChartVersion{
				{
					Name:    "common",
					Version: "1.10.0",
					URLs:    []string{"charts/common-1.10.0.tgz"},
				},
				{
					Name:    "common",
					Version: "1.10.1",
					URLs:    []string{"charts/common-1.10.1.tgz"},
				},
			},
		}, {
			desc: "sync common (latest version only)",
			sourceRepo: &api.Source{
				Spec: &api.Source_Repo{
					Repo: &api.Repo{
						Kind: api.Kind_CHARTMUSEUM,
						Auth: &api.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			targetRepo: &api.Target{
				Spec: &api.Target_Repo{
					Repo: &api.Repo{
						Kind: api.Kind_CHARTMUSEUM,
						Auth: &api.Auth{
							Username: "user",
							Password: "password",
						},
					},
				},
			},
			entries:           []string{"common"},
			requiredCharts:    []string{"common"},
			latestVersionOnly: true,
			want: []*helmclassic.ChartVersion{
				{
					Name:    "common",
					Version: "1.10.1",
					URLs:    []string{"charts/common-1.10.1.tgz"},
				},
			},
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
			if err := utils.CopyFile(dstIndex, "../../testdata/test-index.yaml"); err != nil {
				t.Fatal(err)
			}

			// Create source and target testers
			sTester := repo.NewClientTester(t, tc.sourceRepo.GetRepo(), false, dstIndex)
			tTester := repo.NewClientTester(t, tc.targetRepo.GetRepo(), true, "")

			// Replace placeholder URL with source url
			index, err := ioutil.ReadFile(dstIndex)
			if err != nil {
				t.Fatal(err)
			}
			newContents := strings.Replace(string(index), "TEST_PLACEHOLDER", fmt.Sprintf("%s%s", sTester.GetURL(), "/charts"), -1)
			if err = ioutil.WriteFile(dstIndex, []byte(newContents), 0); err != nil {
				t.Fatal(err)
			}

			// Update source repo url
			tc.sourceRepo.GetRepo().Url = sTester.GetURL()
			// Update target repo url
			tc.targetRepo.GetRepo().Url = tTester.GetURL()

			// Create new syncer
			syncerOptions := []syncer.Option{
				syncer.WithSkipDependencies(tc.skipDependencies),
				syncer.WithLatestVersionOnly(tc.latestVersionOnly),
			}
			s, err := syncer.New(tc.sourceRepo, tc.targetRepo, syncerOptions...)
			if err != nil {
				t.Fatal(err)
			}

			if err := s.SyncPendingCharts(tc.entries...); err != nil {
				t.Error(err)
			}

			// Get charts indexes
			charts := []*helmclassic.ChartVersion{}
			for _, chart := range tc.requiredCharts {
				chartIndex := getChartIndex(t, chart, tc.targetRepo, tTester)
				for _, index := range chartIndex {
					charts = append(charts, index)
				}
			}

			// Sort structs
			sort.SliceStable(charts, func(i, j int) bool {
				return charts[i].Version < charts[j].Version
			})
			sort.SliceStable(tc.want, func(i, j int) bool {
				return tc.want[i].Version < tc.want[j].Version
			})

			if !reflect.DeepEqual(tc.want, charts) {
				t.Errorf("unexpected list of charts. got: %v, want: %v", charts, tc.want)
			}
		})
	}
}
