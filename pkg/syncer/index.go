package syncer

import (
	goerrors "errors"
	"fmt"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/juju/errors"
	"k8s.io/klog"

	"github.com/bitnami/charts-syncer/internal/utils"
)

// Chart describes a chart, including dependencies
type Chart struct {
	Name    string
	Version string
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
			errs = goerrors.Join(errs, errors.Trace(err))
			continue
		}
		if len(versions) == 0 {
			klog.V(5).Infof("Indexing chart %q SKIPPED (no versions found)...", name)
			continue
		}
		klog.V(5).Infof("Found %d versions for %q chart", len(versions), name)
		klog.V(3).Infof("Indexing %q charts...", name)
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
				errs = goerrors.Join(errs, errors.Trace(err))
				continue
			}
		} else {
			for _, version := range versions {
				if err := s.processVersion(name, version, publishingThreshold); err != nil {
					klog.Warningf("Failed processing %s:%s chart. The index will remain incomplete.", name, version)
					errs = goerrors.Join(errs, errors.Trace(err))
					continue
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

	tgz, err := s.cli.src.Fetch(name, version)
	if err != nil {
		return errors.Trace(err)
	}

	ch := &Chart{
		Name:    name,
		Version: version,
		TgzPath: tgz,
	}

	klog.V(4).Infof("Indexing %q chart", id)
	return errors.Trace(s.getIndex().Add(id, ch))
}

func shouldSkipChart(chartName string, skippedCharts []string) bool {
	for _, s := range skippedCharts {
		if s == chartName {
			return true
		}
	}
	return false
}
