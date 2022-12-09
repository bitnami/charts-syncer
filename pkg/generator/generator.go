package generator

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/bitnami-labs/charts-syncer/api"
)

type Generator struct {
	dryRun bool

	manifest *api.Manifest
}

// New a generator
func New(manifest *api.Manifest, opts ...Option) (*Generator, error) {
	g := &Generator{
		manifest: manifest,
	}

	for _, o := range opts {
		o(g)
	}

	return g, nil
}

func getRepoName(repoURL string) string {
	target := strings.Split(repoURL, "/")
	return target[len(target)-1]
}

func convertCharts(charts []*api.Charts) []*Charts {
	c := make([]*Charts, 0, len(charts))
	for _, chart := range charts {
		c = append(c, &Charts{
			Name:     chart.GetName(),
			Versions: chart.GetVersions(),
		})
	}
	return c

}

type Repo struct {
	Kind string `yaml:"kind"`
	URL  string `yaml:"url"`
}

type Source struct {
	Repo Repo `yaml:"repo"`
}

type Target struct {
	IntermediateBundlesPath string `yaml:"intermediateBundlesPath"`
}

type Charts struct {
	Name     string   `yaml:"name"`
	Versions []string `yaml:"versions,omitempty"`
}

type Config struct {
	Source Source `yaml:"source"`
	Target Target `yaml:"target"`
	Charts []*Charts
}

func (g *Generator) Generator() error {
	for _, manifest := range g.manifest.Spec.Manifests {
		c := Config{
			Source: Source{
				Repo: Repo{
					Kind: api.Kind_name[int32(manifest.GetKind())],
					URL:  manifest.GetRepoURL(),
				},
			},
			Target: Target{
				IntermediateBundlesPath: getRepoName(manifest.GetRepoURL()),
			},
			Charts: convertCharts(manifest.GetCharts()),
		}

		data, err := yaml.Marshal(&c)
		if err != nil {
			return err
		}

		err = os.WriteFile(fmt.Sprintf("%s.yaml", getRepoName(manifest.GetRepoURL())), data, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

// Option is an option value used to create a new syncer instance.
type Option func(*Generator)

// WithDryRun configures the syncer to run in dry-run mode.
func WithDryRun(enable bool) Option {
	return func(s *Generator) {
		s.dryRun = enable
	}
}
