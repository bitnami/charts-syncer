package syncer

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	"github.com/philopon/go-toposort"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/internal/chart"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
)

// Chart describes a chart, including dependencies
type Chart struct {
	Name         string
	Version      string
	Dependencies []string

	TgzPath string
}

// ChartIndex is a map linking a chart reference with its Chart
type ChartIndex map[string]*Chart

// func (i ChartIndex) Equal(ii ChartIndex) bool {
// 	var missing []string
// 	for ni, ich := range i {
// 		if iich, ok := ii[ni]; !ok {
// 			return false
// 		}
// 		missing = append(missing, ni)

// 	}
// }

// getIndex returns the chart index
func (s *Syncer) getIndex() ChartIndex {
	if s.index == nil {
		s.index = make(ChartIndex)
	}
	return s.index
}

// Add adds a chart in the index
func (i ChartIndex) Add(id string, chart *Chart) error {
	if _, ok := i[id]; ok {
		return errors.Errorf("%q is already indexed", id)
	}
	i[id] = chart
	return nil
}

// Get returns an index chart
func (i ChartIndex) Get(id string) *Chart {
	if c, ok := i[id]; ok {
		return c
	}
	return nil
}

// loadCharts loads the charts map into the index from the source repo
func (s *Syncer) loadCharts(charts ...string) error {
	if len(charts) == 0 {
		if !s.autoDiscovery {
			return errors.Errorf("unable to discover charts to sync")
		}
		srcCharts, err := s.cli.src.List()
		if err != nil {
			return errors.Trace(err)
		}
		if len(srcCharts) == 0 {
			return errors.Errorf("not found charts to sync")
		}
		charts = srcCharts
	}
	// Sort chart names
	sort.Strings(charts)

	// Create basic layout for date and parse flag to time type
	publishingThreshold, err := utils.GetDateThreshold(s.fromDate)
	if err != nil {
		return errors.Trace(err)
	}
	klog.V(4).Infof("Publishing threshold set to %q", publishingThreshold.String())

	// Iterate over charts in source index
	var errs error
	for _, name := range charts {
		if shouldSkipChart(name, s.skipCharts) {
			klog.V(3).Infof("Indexing %q charts SKIPPED...", name)
			continue
		}

		versions, err := s.cli.src.ListChartVersions(name)
		if err != nil {
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}

		klog.V(5).Infof("Found %d versions for %q chart: %v", len(versions), name, versions)
		klog.V(3).Infof("Indexing %q charts...", name)
		// TODO 在这里添加版本的正则过滤
		if s.latestVersionOnly {
			vs := make([]*semver.Version, len(versions))
			for i, r := range versions {
				v, err := semver.NewVersion(r)
				if err != nil {
					return errors.Trace(err)
				}
				vs[i] = v
			}
			sort.Sort(semver.Collection(vs))
			// The last element of the array is the latest version
			version := vs[len(vs)-1].String()
			if err := s.processVersion(name, version, publishingThreshold); err != nil {
				klog.Warningf("Failed processing %s:%s chart. The index will remain incomplete.", name, version)
				errs = multierror.Append(errs, errors.Trace(err))
				continue
			}
		} else {
			matchVersionRe := regexp.MustCompile(s.matchVersion)
			for _, version := range versions {
				if matchVersionRe.MatchString(version) {
					if err := s.processVersion(name, version, publishingThreshold); err != nil {
						klog.Warningf("Failed processing %s:%s chart. The index will remain incomplete.", name, version)
						errs = multierror.Append(errs, errors.Trace(err))
						continue
					}
				} else {
					klog.V(3).Infof("Skip the version %s that does not match", version)
				}
			}
		}
	}

	return errors.Trace(errs)
}

// processVersion takes care of loading a specific version of the chart into the index
func (s *Syncer) processVersion(name, version string, publishingThreshold time.Time) error {
	details, err := s.cli.src.GetChartDetails(name, version)
	if err != nil {
		return err
	}

	id := fmt.Sprintf("%s-%s", name, version)
	klog.V(5).Infof("Details for %q chart: %+v", id, details)
	if details.PublishedAt.Before(publishingThreshold) {
		klog.V(5).Infof("Skipping %q chart: Published before %q", id, publishingThreshold.String())
		return nil
	}

	if ok, err := s.cli.dst.Has(name, version); err != nil {
		klog.Errorf("unable to explore target repo to check %q chart: %v", id, err)
		return err
	} else if ok {
		klog.V(5).Infof("Skipping %q chart: Already synced", id)
		return nil
	}

	if ch := s.getIndex().Get(id); ch != nil {
		klog.V(5).Infof("Skipping %q chart: Already indexed", id)
		return nil
	}

	if err := s.loadChart(name, version); err != nil {
		klog.Errorf("unable to load %q chart: %v", id, err)
		return err
	}
	return nil
}

// loadChart loads a chart in the chart index map
func (s *Syncer) loadChart(name string, version string) error {
	id := fmt.Sprintf("%s-%s", name, version)
	// loadChart is a recursive function and it will be invoked again for each
	// dependency.
	//
	// It makes sense that different "tier1" charts use the same "tier2" chart
	// dependencies. This check will make the method to skip already indexed
	// charts.
	//
	// Example:
	// `wordpress` is a "tier1" chart that depends on the "tier2" charts `mariadb`
	// and `common`. `magento` is a "tier1" chart that depends on the "tier2"
	// charts `mariadb` and `elasticsearch`.
	//
	// If we run charts-syncer for `wordpress` and `magento`, this check will
	// avoid re-indexing `mariadb` twice.
	if ch := s.getIndex().Get(id); ch != nil {
		klog.V(5).Infof("Skipping %q chart: Already indexed", id)
		return nil
	}
	// In the same way, dependencies may already exist in the target chart
	// repository.
	if ok, err := s.cli.dst.Has(name, version); err != nil {
		return errors.Errorf("unable to explore target repo to check %q chart: %v", id, err)
	} else if ok {
		klog.V(5).Infof("Skipping %q chart: Already synced", id)
		return nil
	}

	tgz, err := s.cli.src.Fetch(name, version)
	if err != nil {
		return errors.Trace(err)
	}

	if err = modifyChartImageTag(tgz); err != nil {
		return errors.Trace(err)
	}

	ch := &Chart{
		Name:    name,
		Version: version,
		TgzPath: tgz,
	}

	if !s.skipDependencies {
		deps, err := chart.GetChartDependencies(tgz, name)
		if err != nil {
			return errors.Trace(err)
		}

		if len(deps) == 0 {
			klog.V(4).Infof("Indexing %q chart", id)
			return errors.Trace(s.getIndex().Add(id, ch))
		}

		var errs error
		for _, dep := range deps {
			depID := fmt.Sprintf("%s-%s", dep.Name, dep.Version)
			if err := s.loadChart(dep.Name, dep.Version); err != nil {
				errs = multierror.Append(errs, errors.Annotatef(err, "invalid %q chart dependency", depID))
				continue
			}
			ch.Dependencies = append(ch.Dependencies, depID)
		}
		if errs != nil {
			return errors.Trace(errs)
		}
	}

	klog.V(4).Infof("Indexing %q chart", id)
	return errors.Trace(s.getIndex().Add(id, ch))
}

// topologicalSortCharts returns the indexed charts, topologically sorted.
func (s *Syncer) topologicalSortCharts() ([]*Chart, error) {
	graph := toposort.NewGraph(len(s.getIndex()))
	for name := range s.getIndex() {
		graph.AddNode(name)
	}
	for name, ch := range s.getIndex() {
		for _, dep := range ch.Dependencies {
			graph.AddEdge(dep, name)
		}
	}

	result, ok := graph.Toposort()
	if !ok {
		return nil, errors.Errorf("dependency cycle detected in charts")
	}

	charts := make([]*Chart, len(result))
	for i, id := range result {
		charts[i] = s.getIndex().Get(id)
	}
	return charts, nil
}

func shouldSkipChart(chartName string, skippedCharts []string) bool {
	for _, s := range skippedCharts {
		if s == chartName {
			return true
		}
	}
	return false
}

func modifyChartImageTag(chartPath string) error {
	//chartPath := "/Users/vista/project/vista/yihctl/cmd/tools/choerodon-platform-2.2.0.tgz"
	cq, err := loader.Load(chartPath)
	if err != nil {
		return errors.Trace(err)
	}

	// 如果没有设置 image.tag，默认设为 AppVersion
	if cq.Values["image"] != nil && (cq.Values["image"]).(map[string]interface{})["tag"] == nil {
		(cq.Values["image"]).(map[string]interface{})["tag"] = cq.Metadata.AppVersion
		for idx, f := range cq.Raw {
			if f.Name == "values.yaml" {
				cq.Raw[idx].Data, _ = yaml.Marshal(cq.Values)

				// 更新默认的 chart
				err = os.Remove(chartPath)
				if err != nil {
					return errors.Trace(err)
				}
				d := filepath.Dir(chartPath)
				_, err = chartutil.Save(cq, d)
				if err != nil {
					return errors.Trace(err)
				}
				break
			}
		}
	}
	return nil
}
