package syncer

import (
	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/bitnami-labs/charts-syncer/pkg/client"
	"github.com/bitnami-labs/charts-syncer/pkg/client/intermediate"
	"github.com/bitnami-labs/charts-syncer/pkg/client/repo"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
	"github.com/juju/errors"
	"k8s.io/klog"
	"os"
)

// Clients holds the source and target chart repo clients
type Clients struct {
	src client.ChartsReaderWriter
	dst client.ChartsReaderWriter
	//stores clients for ignoreTrustedRepos and syncTrustedRepos repositories
	trusted map[uint32]client.ChartsReaderWriter
}

// A Syncer can be used to sync a source and target chart repos.
type Syncer struct {
	source *api.Source
	target *api.Target

	cli *Clients

	dryRun                  bool
	autoDiscovery           bool
	fromDate                string
	insecure                bool
	relocateContainerImages bool
	skipDependencies        bool
	latestVersionOnly       bool
	// list of charts to skip
	skipCharts []string

	// TODO(jdrios): Cache index in local filesystem to speed
	// up re-runs
	index ChartIndex

	// Storage directory for required artifacts
	workdir string
}

// Option is an option value used to create a new syncer instance.
type Option func(*Syncer)

// WithDryRun configures the syncer to run in dry-run mode.
func WithDryRun(enable bool) Option {
	return func(s *Syncer) {
		s.dryRun = enable
	}
}

// WithAutoDiscovery configures the syncer to discover all the charts to sync
// from the source chart repos.
func WithAutoDiscovery(enable bool) Option {
	return func(s *Syncer) {
		s.autoDiscovery = enable
	}
}

// WithFromDate configures the syncer to synchronize the charts from a specific
// time using YYYY-MM-DD format.
func WithFromDate(date string) Option {
	return func(s *Syncer) {
		s.fromDate = date
	}
}

// WithWorkdir configures the syncer to store artifacts in a specific directory.
func WithWorkdir(dir string) Option {
	return func(s *Syncer) {
		s.workdir = dir
	}
}

// WithInsecure configures the syncer to allow insecure SSL connections
func WithInsecure(enable bool) Option {
	return func(s *Syncer) {
		s.insecure = enable
	}
}

// WithContainerImageRelocation configures the syncer to use relok8s to make the chart transformations and push the
// container images to the target registry
func WithContainerImageRelocation(enable bool) Option {
	return func(s *Syncer) {
		s.relocateContainerImages = enable
	}
}

// WithSkipDependencies configures the syncer to skip dependencies sync
func WithSkipDependencies(skipDependencies bool) Option {
	return func(s *Syncer) {
		s.skipDependencies = skipDependencies
	}
}

// WithLatestVersionOnly configures the syncer to sync only the latest version
func WithLatestVersionOnly(latestVersionOnly bool) Option {
	return func(s *Syncer) {
		s.latestVersionOnly = latestVersionOnly
	}
}

// New creates a new syncer using Client
func New(source *api.Source, target *api.Target, opts ...Option) (*Syncer, error) {
	s := &Syncer{
		source: source,
		target: target,
	}

	for _, o := range opts {
		o(s)
	}

	// If a workdir wasn't specified, let's use a directory relative to the
	// current directory
	if s.workdir == "" {
		s.workdir = "./workdir"
	}
	klog.V(3).Infof("Using workdir: %q", s.workdir)

	if err := os.MkdirAll(s.workdir, 0755); err != nil {
		return nil, errors.Trace(err)
	}

	s.cli = &Clients{}
	//inits trusted map
	s.cli.trusted = make(map[uint32]client.ChartsReaderWriter)

	//adds all ignored but trusted repos to the client map
	for _, tRepo := range source.IgnoreTrustedRepos {
		depClientTmp, err := repo.NewClient(tRepo, types.WithCache(s.workdir), types.WithInsecure(s.insecure))
		if err != nil {
			return nil, errors.Trace(err)
		}
		locationId := utils.GetRepoLocationId(utils.GetRepoLocation(tRepo))
		s.cli.trusted[locationId] = depClientTmp
	}

	//adds all trusted repos to the client map
	for _, tRepo := range target.SyncTrustedRepos {
		depClientTmp, err := repo.NewClient(tRepo, types.WithCache(s.workdir), types.WithInsecure(s.insecure))
		if err != nil {
			return nil, errors.Trace(err)
		}
		locationId := utils.GetRepoLocationId(utils.GetRepoLocation(tRepo))
		s.cli.trusted[locationId] = depClientTmp
	}

	if source.GetRepo() != nil {
		srcCli, err := repo.NewClient(source.GetRepo(), types.WithCache(s.workdir), types.WithInsecure(s.insecure))
		if err != nil {
			return nil, errors.Trace(err)
		}
		s.cli.src = srcCli
	} else if source.GetIntermediateBundlesPath() != "" {
		// Specifically disable dependencies sync for intermediate scenarios
		disableDependencySync(s)
		// Create new intermediate bundles client
		srcCli, err := intermediate.NewIntermediateClient(source.GetIntermediateBundlesPath())
		if err != nil {
			return nil, errors.Trace(err)
		}
		s.cli.src = srcCli
	} else {
		return nil, errors.New("no source info defined in config file")
	}

	if target.GetRepo() != nil {
		dstCli, err := repo.NewClient(target.GetRepo(), types.WithCache(s.workdir), types.WithInsecure(s.insecure))
		if err != nil {
			return nil, errors.Trace(err)
		}
		s.cli.dst = dstCli
	} else if target.GetIntermediateBundlesPath() != "" {
		// Specifically disable dependencies sync for intermediate scenarios
		disableDependencySync(s)
		// Create new intermediate bundles client
		dstCli, err := intermediate.NewIntermediateClient(target.GetIntermediateBundlesPath())
		if err != nil {
			return nil, errors.Trace(err)
		}
		s.cli.dst = dstCli
	} else {
		return nil, errors.New("no target info defined in config file")
	}

	if s.relocateContainerImages {
		// Specifically disable dependencies sync for relok8s scenario
		disableDependencySync(s)
	}
	return s, nil
}

// WithSkipCharts configures the syncer to skip an explicit list of chart names
// from the source chart repos.
func WithSkipCharts(charts []string) Option {
	return func(s *Syncer) {
		s.skipCharts = charts
	}
}

func disableDependencySync(syncer *Syncer) {
	if syncer.skipDependencies == false {
		klog.Warningf("Ignoring skipDependencies option as dependency sync is not supported if container image relocation is true or syncing from/to intermediate directory ")
	}
	syncer.skipDependencies = true
}
