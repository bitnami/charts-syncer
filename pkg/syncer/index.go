package syncer

import (
	"fmt"
	"path"
	"sort"

	"github.com/juju/errors"
	"github.com/mkmik/multierror"
	toposort "github.com/philopon/go-toposort"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/pkg/chart"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"
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
		versions, err := s.cli.src.ListChartVersions(name)
		if err != nil {
			errs = multierror.Append(errs, errors.Trace(err))
			continue
		}

		klog.V(5).Infof("Found %d versions for %q chart: %v", len(versions), name, versions)
		klog.V(3).Infof("Indexing %q charts...", name)
		for _, version := range versions {
			details, err := s.cli.src.GetChartDetails(name, version)
			if err != nil {
				errs = multierror.Append(errs, errors.Trace(err))
				continue
			}

			id := fmt.Sprintf("%s-%s", name, version)
			klog.V(5).Infof("Details for %q chart: %+v", id, details)
			if details.PublishedAt.Before(publishingThreshold) {
				klog.V(4).Infof("Skipping %q chart: Published before %q", id, publishingThreshold.String())
				continue
			}

			if ok, err := s.cli.dst.Has(name, version); err != nil {
				klog.Errorf("unable to explore target repo to check %q chart: %v", id, err)
				errs = multierror.Append(errs, errors.Trace(err))
				continue
			} else if ok {
				klog.V(4).Infof("Skipping %q chart: Already synced", id)
				continue
			}

			if ch := s.getIndex().Get(id); ch != nil {
				klog.V(4).Infof("Skipping %q chart: Already indexed", id)
				continue
			}

			if err := s.loadChart(name, version); err != nil {
				klog.Errorf("unable to load %q chart: %v", id, err)
				errs = multierror.Append(errs, errors.Trace(err))
				continue
			}
		}
	}

	return errors.Trace(errs)
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
		klog.V(4).Infof("Skipping %q chart: Already indexed", id)
		return nil
	}
	// In the same way, dependencies may already exist in the target chart
	// repository.
	if ok, err := s.cli.dst.Has(name, version); err != nil {
		return errors.Errorf("unable to explore target repo to check %q chart: %v", id, err)
	} else if ok {
		klog.V(4).Infof("Skipping %q chart: Already synced", id)
		return nil
	}

	tgz := path.Join(s.srcWorkdir, fmt.Sprintf("%s-%s.tgz", name, version))

	// Fetch chart iff it does not exist in the workdir already
	if ok, err := utils.FileExists(tgz); err != nil {
		return errors.Trace(err)
	} else if !ok {
		if err := s.cli.src.Fetch(tgz, name, version); err != nil {
			return errors.Trace(err)
		}
	}

	ch := &Chart{
		Name:    name,
		Version: version,
		TgzPath: tgz,
	}

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
