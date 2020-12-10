package chart

import (
	"errors"
	"testing"
)

func TestLockFilePath(t *testing.T) {
	tests := map[string]struct {
		chartPath     string
		apiVersion    string
		expectedPath  string
		shouldFail    bool
		expectedError error
	}{
		"api v1 chart": {
			"/tmp/kafka",
			APIV1,
			"/tmp/kafka/requirements.lock",
			false,
			nil,
		},
		"api v2 chart": {
			"/tmp/kafka",
			APIV2,
			"/tmp/kafka/Chart.lock",
			false,
			nil,
		},
		"unexisting api chart": {
			"/tmp/kafka",
			"vvv000",
			"",
			true,
			errors.New("unrecognised apiVersion \"vvv000\""),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			want := tc.expectedPath
			got, err := lockFilePath(tc.chartPath, tc.apiVersion)
			if tc.shouldFail {
				if err.Error() != tc.expectedError.Error() {
					t.Errorf("error does not match: [%v:%v]", tc.expectedError, err)
				}
			} else {
				if got != want {
					t.Errorf("got: %q, want %q", got, want)
				}
			}
		})
	}
}
