package common

import (
	"testing"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/pkg/client/types"
)

// MockChartsReaderWriter is a mock implementation of ChartsReaderWriter
type MockChartsReaderWriter struct {
	uploadURL string
}

func (m *MockChartsReaderWriter) Fetch(name, version string) (string, error) {
	return "", nil
}

func (m *MockChartsReaderWriter) List() ([]string, error) {
	return []string{}, nil
}

func (m *MockChartsReaderWriter) ListChartVersions(name string) ([]string, error) {
	return []string{}, nil
}

func (m *MockChartsReaderWriter) Has(name, version string) (bool, error) {
	return false, nil
}

func (m *MockChartsReaderWriter) GetChartDetails(name, version string) (*types.ChartDetails, error) {
	return nil, nil
}

func (m *MockChartsReaderWriter) GetUploadURL() string {
	return m.uploadURL
}

func TestGetContainersUploadURL_WithoutExplicitContainersURL(t *testing.T) {
	// Test case: target.containers is nil, should append /containers to chart URL
	target := &apiv1.Target{
		Repo: &apiv1.Repo{
			Url: "http://127.0.0.1:5000/library-replicated/charts/photon-5",
		},
		Containers: nil,
	}

	t.Run("containers_url_not_specified", func(t *testing.T) {
		mockCharts := &MockChartsReaderWriter{
			uploadURL: "http://127.0.0.1:5000/library-replicated/charts/photon-5",
		}
		targetObj, err := New(target, mockCharts, false, false)
		if err != nil {
			t.Fatalf("error creating target: %v", err)
		}

		got := targetObj.getContainersUploadURL()
		want := "127.0.0.1:5000/library-replicated/charts/photon-5/containers"

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestGetContainersUploadURL_WithExplicitContainersURL(t *testing.T) {
	// Test case: target.containers is specified, should use that URL
	target := &apiv1.Target{
		Repo: &apiv1.Repo{
			Url: "http://127.0.0.1:5000/library-replicated/charts/photon-5",
		},
		Containers: &apiv1.Containers{
			Url: "http://127.0.0.1:5000/library-replicated/containers/photon-5",
		},
	}

	t.Run("containers_url_specified", func(t *testing.T) {
		mockCharts := &MockChartsReaderWriter{
			uploadURL: "http://127.0.0.1:5000/library-replicated/charts/photon-5",
		}
		targetObj, err := New(target, mockCharts, false, false)
		if err != nil {
			t.Fatalf("error creating target: %v", err)
		}

		got := targetObj.getContainersUploadURL()
		want := "127.0.0.1:5000/library-replicated/containers/photon-5"

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestGetContainersUploadURL_StripsScheme(t *testing.T) {
	// Test case: verify that scheme is stripped correctly
	target := &apiv1.Target{
		Repo: &apiv1.Repo{
			Url: "oci://127.0.0.1:5000/library-replicated/charts/photon-5",
		},
		Containers: nil,
	}

	t.Run("scheme_stripped_with_containers_suffix", func(t *testing.T) {
		mockCharts := &MockChartsReaderWriter{
			uploadURL: "oci://127.0.0.1:5000/library-replicated/charts/photon-5",
		}
		targetObj, err := New(target, mockCharts, false, false)
		if err != nil {
			t.Fatalf("error creating target: %v", err)
		}

		got := targetObj.getContainersUploadURL()
		// Scheme should be stripped, and /containers should be appended before scheme removal
		// Actually, the scheme removal happens after appending /containers
		want := "127.0.0.1:5000/library-replicated/charts/photon-5/containers"

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
