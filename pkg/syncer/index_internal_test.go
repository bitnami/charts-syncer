package syncer

import (
	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/google/go-cmp/cmp/cmpopts"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func removeTgzPath(i ChartIndex) {
	for _, c := range i {
		c.TgzPath = ""
	}
}

func TestLoadCharts(t *testing.T) {

	repo := api.Repo{
		Kind: api.Kind_UNKNOWN,
	}

	repoZooKeeper := api.Repo{
		Kind: api.Kind_UNKNOWN,
		Url:  "https://charts.bitnami.com/bitnami",
	}

	testCases := []struct {
		desc           string
		entries        []string
		skippedEntries []string
		want           ChartIndex
	}{
		{
			desc:    "load apache and kafka",
			entries: []string{"apache", "kafka"},
			want: ChartIndex{
				"apache-7.3.15":    &Chart{Name: "apache", Version: "7.3.15", Repo: repo},
				"kafka-10.3.3":     &Chart{Name: "kafka", Version: "10.3.3", Dependencies: []string{"zookeeper-5.14.3"}, Repo: repo},
				"zookeeper-5.14.3": &Chart{Name: "zookeeper", Version: "5.14.3", Repo: repoZooKeeper},
			},
		},
		{
			desc:           "skip apache and kafka",
			entries:        []string{"apache", "kafka", "zookeeper"},
			skippedEntries: []string{"apache", "kafka"},
			want: ChartIndex{
				"zookeeper-5.14.3": &Chart{Name: "zookeeper", Version: "5.14.3", Repo: repo},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			s := NewFake(t, WithFakeSkipCharts(tc.skippedEntries))
			if err := s.loadCharts(tc.entries...); err != nil {
				t.Fatalf("unable to load charts: %v", err)
			}

			// Remove TgzPath values from the computed index
			removeTgzPath(s.getIndex())

			if diff := cmp.Diff(tc.want, s.getIndex(), cmpopts.IgnoreUnexported(api.Repo{})); diff != "" {
				t.Errorf("want vs got diff:\n %+v", diff)
			}
		})
	}
}

func TestTopologicalSortCharts(t *testing.T) {
	testCases := []struct {
		desc  string
		index ChartIndex
		want  []*Chart
	}{
		{
			desc: "sort index to put dependencies first",
			index: ChartIndex{
				"kafka-10.3.3":     &Chart{Name: "kafka", Version: "10.3.3", Dependencies: []string{"zookeeper-5.14.3"}},
				"zookeeper-5.14.3": &Chart{Name: "zookeeper", Version: "5.14.3"},
			},
			want: []*Chart{
				{Name: "zookeeper", Version: "5.14.3"},
				{Name: "kafka", Version: "10.3.3", Dependencies: []string{"zookeeper-5.14.3"}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			s := &Syncer{index: tc.index}

			got, err := s.topologicalSortCharts()
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreUnexported(api.Repo{})); diff != "" {
				t.Errorf("want vs got diff:\n %+v", diff)
			}
		})
	}
}
