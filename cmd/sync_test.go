package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"
	"text/template"

	"github.com/bitnami/charts-syncer/api"
	"github.com/bitnami/charts-syncer/internal/utils"
	"github.com/bitnami/charts-syncer/pkg/client/repo/oci"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
)

var (
	ociSourceRepo = &api.Repo{
		Kind: api.Kind_OCI,
		Auth: &api.Auth{
			Username: "user",
			Password: "password",
		},
		DisableChartsIndex: true,
	}
	ociTargetRepo = &api.Repo{
		Kind: api.Kind_OCI,
		Auth: &api.Auth{
			Username: "foo",
			Password: "password",
		},
		DisableChartsIndex: true,
	}
)

func TestSync(t *testing.T) {
	charts := map[string]string{
		"apache":    "7.3.15",
		"zookeeper": "5.14.3",
	}

	testCases := []struct {
		name         string
		chartsToSync []string
		twice        bool
	}{
		{name: "SyncNoneReturnsNothingToSync", chartsToSync: []string{}, twice: false},
		{name: "SyncAllChartsOnceSyncs", chartsToSync: []string{"apache", "zookeeper"}, twice: false},
		{name: "SyncAllChartsTwiceSyncsAndReturnsNothingToSync", chartsToSync: []string{"apache", "zookeeper"}, twice: true},
		{name: "SyncSelectedChartsOnceSyncsSelectedCharts", chartsToSync: []string{"apache"}, twice: false},
		{name: "SyncSelectedChartsTwiceSyncsSelectedChartsAndReturnsNothingToSync", chartsToSync: []string{"zookeeper"}, twice: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prepareSourceRepo(context.Background(), t)
			oci.PrepareOCIServer(context.Background(), t, ociTargetRepo)
			ct := oci.PrepareTest(t, ociTargetRepo)

			cfg, err := renderConfigFile("../testdata/sync-test.tmpl.yaml", tc.chartsToSync...)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Remove(cfg) })

			args := []string{"sync", "--use-plain-log", "--use-plain-http", "--config", cfg}
			if len(tc.chartsToSync) > 0 {
				chartsyncer(args...).AssertSuccessMatchStderr(t, "Charts synced successfully")
			} else {
				chartsyncer(args...).AssertSuccessMatchStderr(t, "There are no charts out of sync!")
			}

			for k, v := range charts {
				if slices.Contains(tc.chartsToSync, k) {
					assert.NoError(t, verifyChart(ct, k, v))
				} else {
					assert.Error(t, verifyChart(ct, k, v))
				}
			}

			if tc.twice {
				chartsyncer(args...).AssertSuccessMatchStderr(t, "There are no charts out of sync!")
			}
		})
	}
}

func prepareSourceRepo(ctx context.Context, t *testing.T) {
	oci.PrepareOCIServer(context.Background(), t, ociSourceRepo)
	cs := oci.PrepareTest(t, ociSourceRepo)

	charts := []struct {
		Name    string
		Version string
	}{
		{Name: "apache", Version: "7.3.15"},
		{Name: "zookeeper", Version: "5.14.3"},
	}

	for _, c := range charts {
		chartMetadata := &chart.Metadata{
			Name:    c.Name,
			Version: c.Version,
		}
		// Upload chart to source repo
		chartPath := fmt.Sprintf("../testdata/%s-%s.tgz", c.Name, c.Version)
		if err := cs.Upload(chartPath, chartMetadata); err != nil {
			t.Fatal(err)
		}
	}
}

func verifyChart(repo *oci.Repo, name, version string) error {
	chartPath, err := repo.Fetch(name, version)
	if err != nil {
		return err
	}

	if _, err := os.Stat(chartPath); err != nil {
		return fmt.Errorf("chart package does not exist: %w", err)
	}
	defer os.Remove(chartPath)

	contentType, err := utils.GetFileContentType(chartPath)
	if err != nil {
		return fmt.Errorf("error checking contentType of %q file: %w", chartPath, err)
	}

	if contentType != "application/x-gzip" {
		return fmt.Errorf("incorrect content type, got: %q, want %q instead", contentType, "application/x-gzip")
	}

	return nil
}

func renderConfigFile(tmpl string, charts ...string) (string, error) {
	templateData := struct {
		SourceURL      string
		SourceUser     string
		SourcePassword string
		SourceIndex    bool
		TargetURL      string
		TargetUser     string
		TargetPassword string
		TargetIndex    bool
		Charts         []string
	}{
		ociSourceRepo.Url,
		ociSourceRepo.Auth.Username,
		ociSourceRepo.Auth.Password,
		ociSourceRepo.DisableChartsIndex,
		ociTargetRepo.Url,
		ociTargetRepo.Auth.Username,
		ociTargetRepo.Auth.Password,
		ociTargetRepo.DisableChartsIndex,
		charts,
	}

	f, err := os.CreateTemp("", "charts-syncer-*.yaml")
	if err != nil {
		return "", err
	}
	defer f.Close()

	t, err := template.ParseFiles(tmpl)
	if err != nil {
		return "", err
	}

	err = t.Execute(f, templateData)
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}
