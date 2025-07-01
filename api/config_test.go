package api_test

import (
	"testing"

	"github.com/bitnami/charts-syncer/api"
)

func TestValidate(t *testing.T) {
	config := &api.Config{
		Source: &api.Source{
			Repo: &api.Repo{
				Url:  "ht//:fake.source.com",
				Kind: api.Kind_CHARTMUSEUM,
				Auth: &api.Auth{
					Username: "user",
					Password: "password",
				},
			},
		},
		Target: &api.Target{
			Repo: &api.Repo{
				Url:  "http://fake.target.com",
				Kind: api.Kind_CHARTMUSEUM,
				Auth: &api.Auth{
					Username: "user",
					Password: "password",
				},
			},
		},
	}

	if err := config.Validate(); err == nil {
		t.Errorf("expected error but got nothing")
	} else {
		expectedError := `"sources[0].repo.url" should be a valid URL: parse "ht//:fake.source.com": invalid URI for request`
		if err.Error() != expectedError {
			t.Errorf("incorrect error, got: \n %s \n, want: \n %s \n", err.Error(), expectedError)
		}
	}
}

func TestValidateMultipleSources(t *testing.T) {
	t.Run("valid multiple sources", func(t *testing.T) {
		config := &api.Config{
			Sources: []*api.Source{
				{
					Repo: &api.Repo{
						Url:  "http://source1.example.com",
						Kind: api.Kind_HELM,
					},
				},
				{
					Repo: &api.Repo{
						Url:  "http://source2.example.com",
						Kind: api.Kind_CHARTMUSEUM,
					},
				},
			},
			Target: &api.Target{
				Repo: &api.Repo{
					Url:  "http://target.example.com",
					Kind: api.Kind_OCI,
				},
			},
		}

		if err := config.Validate(); err != nil {
			t.Errorf("expected no error but got: %v", err)
		}
	})

	t.Run("invalid URL in second source", func(t *testing.T) {
		config := &api.Config{
			Sources: []*api.Source{
				{
					Repo: &api.Repo{
						Url:  "http://source1.example.com",
						Kind: api.Kind_HELM,
					},
				},
				{
					Repo: &api.Repo{
						Url:  "invalid-url",
						Kind: api.Kind_CHARTMUSEUM,
					},
				},
			},
			Target: &api.Target{
				Repo: &api.Repo{
					Url:  "http://target.example.com",
					Kind: api.Kind_OCI,
				},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Errorf("expected error but got nothing")
		} else if err.Error() != `"sources[1].repo.url" should be a valid URL: parse "invalid-url": invalid URI for request` {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("no sources provided", func(t *testing.T) {
		config := &api.Config{
			Target: &api.Target{
				Repo: &api.Repo{
					Url:  "http://target.example.com",
					Kind: api.Kind_OCI,
				},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Errorf("expected error but got nothing")
		} else if err.Error() != "at least one source must be specified" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("both source and sources specified", func(t *testing.T) {
		config := &api.Config{
			Source: &api.Source{
				Repo: &api.Repo{
					Url:  "http://old-source.example.com",
					Kind: api.Kind_HELM,
				},
			},
			Sources: []*api.Source{
				{
					Repo: &api.Repo{
						Url:  "http://new-source.example.com",
						Kind: api.Kind_HELM,
					},
				},
			},
			Target: &api.Target{
				Repo: &api.Repo{
					Url:  "http://target.example.com",
					Kind: api.Kind_OCI,
				},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Errorf("expected error but got nothing")
		} else if err.Error() != `cannot use both "source" and "sources" fields. Please use only "sources" for multiple source support` {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestBackwardCompatibility(t *testing.T) {
	t.Run("old single source format still works", func(t *testing.T) {
		config := &api.Config{
			Source: &api.Source{
				Repo: &api.Repo{
					Url:  "http://source.example.com",
					Kind: api.Kind_HELM,
				},
			},
			Target: &api.Target{
				Repo: &api.Repo{
					Url:  "http://target.example.com",
					Kind: api.Kind_OCI,
				},
			},
		}

		if err := config.Validate(); err != nil {
			t.Errorf("expected no error but got: %v", err)
		}

		// Test that GetEffectiveSources returns the migrated source
		sources := config.GetEffectiveSources()
		if len(sources) != 1 {
			t.Errorf("expected 1 source but got %d", len(sources))
		}
		if sources[0].GetRepo().GetUrl() != "http://source.example.com" {
			t.Errorf("unexpected source URL: %s", sources[0].GetRepo().GetUrl())
		}
	})
}

func TestGetEffectiveSources(t *testing.T) {
	t.Run("returns sources when using new format", func(t *testing.T) {
		config := &api.Config{
			Sources: []*api.Source{
				{
					Repo: &api.Repo{
						Url:  "http://source1.example.com",
						Kind: api.Kind_HELM,
					},
				},
				{
					Repo: &api.Repo{
						Url:  "http://source2.example.com",
						Kind: api.Kind_CHARTMUSEUM,
					},
				},
			},
		}

		sources := config.GetEffectiveSources()
		if len(sources) != 2 {
			t.Errorf("expected 2 sources but got %d", len(sources))
		}
	})

	t.Run("returns single source when using old format", func(t *testing.T) {
		config := &api.Config{
			Source: &api.Source{
				Repo: &api.Repo{
					Url:  "http://source.example.com",
					Kind: api.Kind_HELM,
				},
			},
		}

		sources := config.GetEffectiveSources()
		if len(sources) != 1 {
			t.Errorf("expected 1 source but got %d", len(sources))
		}
		if sources[0].GetRepo().GetUrl() != "http://source.example.com" {
			t.Errorf("unexpected source URL: %s", sources[0].GetRepo().GetUrl())
		}
	})

	t.Run("returns empty when no sources", func(t *testing.T) {
		config := &api.Config{}

		sources := config.GetEffectiveSources()
		if len(sources) != 0 {
			t.Errorf("expected 0 sources but got %d", len(sources))
		}
	})
}

func TestContainerAuthValidation(t *testing.T) {
	t.Run("validates container auth for multiple sources", func(t *testing.T) {
		config := &api.Config{
			Sources: []*api.Source{
				{
					Repo: &api.Repo{
						Url:  "http://source1.example.com",
						Kind: api.Kind_HELM,
					},
					Containers: &api.Containers{
						Auth: &api.Containers_ContainerAuth{
							Username: "user1",
							Password: "pass1",
							Registry: "registry1.example.com",
						},
					},
				},
				{
					Repo: &api.Repo{
						Url:  "http://source2.example.com",
						Kind: api.Kind_CHARTMUSEUM,
					},
					Containers: &api.Containers{
						Auth: &api.Containers_ContainerAuth{
							Username: "user2",
							Password: "", // Missing password
							Registry: "registry2.example.com",
						},
					},
				},
			},
			Target: &api.Target{
				Repo: &api.Repo{
					Url:  "http://target.example.com",
					Kind: api.Kind_OCI,
				},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Errorf("expected error but got nothing")
		} else if err.Error() != `"sources[1].containers.auth" "registry", "username" and "password" are required"` {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestSourceSpecificCharts(t *testing.T) {
	t.Run("source specific charts validation", func(t *testing.T) {
		config := &api.Config{
			Sources: []*api.Source{
				{
					Repo: &api.Repo{
						Url:  "http://source1.example.com",
						Kind: api.Kind_HELM,
					},
					Charts: []string{"apache", "nginx"},
				},
				{
					Repo: &api.Repo{
						Url:  "http://source2.example.com",
						Kind: api.Kind_CHARTMUSEUM,
					},
					SkipCharts: []string{"deprecated-chart"},
				},
			},
			Target: &api.Target{
				Repo: &api.Repo{
					Url:  "http://target.example.com",
					Kind: api.Kind_OCI,
				},
			},
		}

		if err := config.Validate(); err != nil {
			t.Errorf("expected no error but got: %v", err)
		}
	})

	t.Run("source with both charts and skip_charts should fail", func(t *testing.T) {
		config := &api.Config{
			Sources: []*api.Source{
				{
					Repo: &api.Repo{
						Url:  "http://source1.example.com",
						Kind: api.Kind_HELM,
					},
					Charts:     []string{"apache", "nginx"},
					SkipCharts: []string{"deprecated-chart"}, // Both defined - should fail
				},
			},
			Target: &api.Target{
				Repo: &api.Repo{
					Url:  "http://target.example.com",
					Kind: api.Kind_OCI,
				},
			},
		}

		err := config.Validate()
		if err == nil {
			t.Errorf("expected error but got nothing")
		} else if err.Error() != `"sources[0].charts" and "sources[0].skip_charts" properties cannot be set at the same time` {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestEffectiveChartsForSource(t *testing.T) {
	t.Run("source specific charts take precedence", func(t *testing.T) {
		config := &api.Config{
			Charts: []string{"global-chart1", "global-chart2"},
			Sources: []*api.Source{
				{
					Charts: []string{"source-chart1", "source-chart2"},
				},
			},
		}

		source := config.Sources[0]
		effectiveCharts := config.GetEffectiveChartsForSource(source)

		if len(effectiveCharts) != 2 {
			t.Errorf("expected 2 charts but got %d", len(effectiveCharts))
		}
		if effectiveCharts[0] != "source-chart1" || effectiveCharts[1] != "source-chart2" {
			t.Errorf("expected source-specific charts but got: %v", effectiveCharts)
		}
	})

	t.Run("fallback to global charts when source has none", func(t *testing.T) {
		config := &api.Config{
			Charts: []string{"global-chart1", "global-chart2"},
			Sources: []*api.Source{
				{
					// No charts specified
				},
			},
		}

		source := config.Sources[0]
		effectiveCharts := config.GetEffectiveChartsForSource(source)

		if len(effectiveCharts) != 2 {
			t.Errorf("expected 2 charts but got %d", len(effectiveCharts))
		}
		if effectiveCharts[0] != "global-chart1" || effectiveCharts[1] != "global-chart2" {
			t.Errorf("expected global charts but got: %v", effectiveCharts)
		}
	})
}

func TestEffectiveSkipChartsForSource(t *testing.T) {
	t.Run("source specific skip_charts take precedence", func(t *testing.T) {
		config := &api.Config{
			SkipCharts: []string{"global-skip1", "global-skip2"},
			Sources: []*api.Source{
				{
					SkipCharts: []string{"source-skip1", "source-skip2"},
				},
			},
		}

		source := config.Sources[0]
		effectiveSkipCharts := config.GetEffectiveSkipChartsForSource(source)

		if len(effectiveSkipCharts) != 2 {
			t.Errorf("expected 2 skip_charts but got %d", len(effectiveSkipCharts))
		}
		if effectiveSkipCharts[0] != "source-skip1" || effectiveSkipCharts[1] != "source-skip2" {
			t.Errorf("expected source-specific skip_charts but got: %v", effectiveSkipCharts)
		}
	})

	t.Run("fallback to global skip_charts when source has none", func(t *testing.T) {
		config := &api.Config{
			SkipCharts: []string{"global-skip1", "global-skip2"},
			Sources: []*api.Source{
				{
					// No skip_charts specified
				},
			},
		}

		source := config.Sources[0]
		effectiveSkipCharts := config.GetEffectiveSkipChartsForSource(source)

		if len(effectiveSkipCharts) != 2 {
			t.Errorf("expected 2 skip_charts but got %d", len(effectiveSkipCharts))
		}
		if effectiveSkipCharts[0] != "global-skip1" || effectiveSkipCharts[1] != "global-skip2" {
			t.Errorf("expected global skip_charts but got: %v", effectiveSkipCharts)
		}
	})
}
