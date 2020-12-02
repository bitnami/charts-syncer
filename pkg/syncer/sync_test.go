package syncer_test

import (
	"testing"
)

func TestSyncPendingCharts(t *testing.T) {
	testCases := []struct {
		desc    string
		entries map[string][]string
		charts  []string
	}{
		{
			desc: "load apache and kafka",
			entries: map[string][]string{
				"apache": {"7.3.15"},
				"kafka":  {"10.3.3"},
			},
			charts: []string{"apache", "kafka"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// TODO(jdrios): We can't invoke SyncPendingCharts since this function
			// tries to add the target repo using the helm cli.

			// s := syncer.NewFake(t, tc.entries)
			// var charts []string
			// for name := range tc.entries {
			// 	charts = append(charts, name)
			// }
			// if err := s.SyncPendingCharts(charts...); err != nil {
			// 	t.Error(err)
			// }
		})
	}
}
