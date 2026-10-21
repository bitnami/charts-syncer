package chartsyncer

import (
	"testing"

	"github.com/bitnami/charts-syncer/pkg/client"
	"github.com/bitnami/charts-syncer/pkg/client/config"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

type skippedChartSource struct {
	client.ChartsWrapper
	t *testing.T
}

func (s skippedChartSource) WrapChart(string, string, ...config.Option) (string, error) {
	s.t.Fatal("an existing chart must not be wrapped to refresh its image aliases")
	return "", nil
}

func TestAliasSyncDoesNotProcessExistingCharts(t *testing.T) {
	dir := t.TempDir()
	s := NewFake(t, WithFakeSyncerDestination(dir))
	require.NoError(t, s.SyncPendingCharts("apache"))
	s = NewFake(t, WithFakeSyncerDestination(dir))
	s.syncAliasTags = true
	s.cli.src = skippedChartSource{ChartsWrapper: s.cli.src, t: t}
	require.ErrorIs(t, s.SyncPendingCharts("apache"), ErrNoChartsToSync)
	require.Empty(t, s.getIndex())
}

func removeTgzPath(i ChartIndex) {
	for _, c := range i {
		c.TgzPath = ""
	}
}

func TestLoadCharts(t *testing.T) {
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
				"apache-7.3.15": &Chart{Name: "apache", Version: "7.3.15"},
				"kafka-10.3.3":  &Chart{Name: "kafka", Version: "10.3.3"},
			},
		},
		{
			desc:           "skip apache and kafka",
			entries:        []string{"apache", "kafka", "zookeeper"},
			skippedEntries: []string{"apache", "kafka"},
			want: ChartIndex{
				"zookeeper-5.14.3": &Chart{Name: "zookeeper", Version: "5.14.3"},
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

			if diff := cmp.Diff(tc.want, s.getIndex()); diff != "" {
				t.Errorf("want vs got diff:\n %+v", diff)
			}
		})
	}
}
