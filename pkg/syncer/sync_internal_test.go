package syncer

import (
	"reflect"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"

	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/pkg/mover"
)

func TestGetRelok8sMoveRequest(t *testing.T) {
	testCases := []struct {
		desc          string
		source        *api.Source
		target        *api.Target
		chart         *Chart
		wantReq       *mover.ChartMoveRequest
		wantChartPath string
	}{
		{
			desc: "direct sync with relok8s",
			source: &api.Source{
				Spec: &api.Source_Repo{
					Repo: &api.Repo{
						Url:  "https://my-source-chartmuseum.dev",
						Kind: api.Kind_CHARTMUSEUM,
					},
				},
			},
			target: &api.Target{
				Spec: &api.Target_Repo{
					Repo: &api.Repo{
						Url:  "https://my-target-chartmuseum.dev/",
						Kind: api.Kind_CHARTMUSEUM,
					},
				},
				ContainerRegistry:   "test.registry.io",
				ContainerRepository: "test/repo",
			},
			chart: &Chart{
				Name:    "chart-A",
				Version: "1.2.3",
				TgzPath: "/tmp/workdir/chart-A-1.2.3.tgz",
			},
			wantReq: &mover.ChartMoveRequest{
				Source: mover.Source{
					Chart: mover.ChartSpec{
						Local: &mover.LocalChart{
							Path: "/tmp/workdir/chart-A-1.2.3.tgz",
						},
					},
				},
				Target: mover.Target{
					Rules: mover.RewriteRules{
						Registry:         "test.registry.io",
						RepositoryPrefix: "test/repo",
						ForcePush:        true,
					},
					Chart: mover.ChartSpec{
						Local: &mover.LocalChart{
							Path: "/tmp/output-dir/%s-%s.tgz",
						},
					},
				},
			},
			wantChartPath: "/tmp/output-dir/chart-A-1.2.3.tgz",
		}, {
			desc: "save bundles with relok8s",
			source: &api.Source{
				Spec: &api.Source_Repo{
					Repo: &api.Repo{
						Url:  "https://my-source-chartmuseum.dev",
						Kind: api.Kind_CHARTMUSEUM,
					},
				},
			},
			target: &api.Target{
				Spec: &api.Target_IntermediateBundlesPath{
					IntermediateBundlesPath: "/tmp/bundles-dir",
				},
			},
			chart: &Chart{
				Name:    "chart-A",
				Version: "1.2.3",
				TgzPath: "/tmp/workdir/chart-A-1.2.3.tgz",
			},
			wantReq: &mover.ChartMoveRequest{
				Source: mover.Source{
					Chart: mover.ChartSpec{
						Local: &mover.LocalChart{
							Path: "/tmp/workdir/chart-A-1.2.3.tgz",
						},
					},
				},
				Target: mover.Target{
					Chart: mover.ChartSpec{
						IntermediateBundle: &mover.IntermediateBundle{
							Path: "/tmp/output-dir/chart-A-1.2.3.bundle.tar",
						},
					},
				},
			},
			wantChartPath: "/tmp/output-dir/chart-A-1.2.3.bundle.tar",
		}, {
			desc: "load bundles with relok8s",
			source: &api.Source{
				Spec: &api.Source_IntermediateBundlesPath{
					IntermediateBundlesPath: "/tmp/bundles-dir",
				},
			},
			target: &api.Target{
				Spec: &api.Target_Repo{
					Repo: &api.Repo{
						Url:  "https://my-target-chartmuseum.dev/",
						Kind: api.Kind_CHARTMUSEUM,
					},
				},
				ContainerRegistry:   "test.registry.io",
				ContainerRepository: "test/repo",
			},
			chart: &Chart{
				Name:    "chart-A",
				Version: "1.2.3",
				TgzPath: "/tmp/workdir/chart-A-1.2.3.bundle.tar",
			},
			wantReq: &mover.ChartMoveRequest{
				Source: mover.Source{
					Chart: mover.ChartSpec{
						IntermediateBundle: &mover.IntermediateBundle{
							Path: "/tmp/workdir/chart-A-1.2.3.bundle.tar",
						},
					},
				},
				Target: mover.Target{
					Rules: mover.RewriteRules{
						Registry:         "test.registry.io",
						RepositoryPrefix: "test/repo",
						ForcePush:        true,
					},
					Chart: mover.ChartSpec{
						Local: &mover.LocalChart{
							Path: "/tmp/output-dir/%s-%s.relocated.tgz",
						},
					},
				},
			},
			wantChartPath: "/tmp/output-dir/chart-A-1.2.3.relocated.tgz",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			outdir := "/tmp/output-dir/"
			gotReq, gotChartPath := getRelok8sMoveRequest(tc.source, tc.target, tc.chart, outdir)
			if !reflect.DeepEqual(gotReq, tc.wantReq) {
				t.Errorf("got: %v, want: %v\n", gotReq, tc.wantReq)
			}
			if gotChartPath != tc.wantChartPath {
				t.Errorf("got: %q, want: %q\n", gotChartPath, tc.wantChartPath)
			}
		})
	}
}

func TestRelok8sMoveReq(t *testing.T) {
	want := &mover.ChartMoveRequest{
		Source: mover.Source{
			Chart: mover.ChartSpec{
				Local: &mover.LocalChart{
					Path: "/tmp/source-dir",
				},
			},
			ContainersAuth: &mover.ContainersAuth{Credentials: &mover.OCICredentials{Server: "sreg", Username: "suser", Password: "spass"}},
		},
		Target: mover.Target{
			Rules: mover.RewriteRules{
				Registry:         "gcr.io",
				RepositoryPrefix: "my-project/containers",
				ForcePush:        true,
			},
			Chart: mover.ChartSpec{
				Local: &mover.LocalChart{
					Path: "/tmp/target-dir",
				},
			},
			ContainersAuth: &mover.ContainersAuth{Credentials: &mover.OCICredentials{Server: "treg", Username: "tuser", Password: "tpass"}},
		},
	}
	got := relok8sMoveReq("/tmp/source-dir", "/tmp/target-dir", "gcr.io", "my-project/containers",
		&api.Containers_ContainerAuth{Username: "suser", Password: "spass", Registry: "sreg"}, &api.Containers_ContainerAuth{Username: "tuser", Password: "tpass", Registry: "treg"},
	)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected relok8s request. got: %v, want: %v", got, want)
	}
}

func TestRelok8sBundleSaveReq(t *testing.T) {
	want := &mover.ChartMoveRequest{
		Source: mover.Source{
			Chart: mover.ChartSpec{
				Local: &mover.LocalChart{
					Path: "/tmp/source-dir",
				},
			},
			ContainersAuth: &mover.ContainersAuth{Credentials: &mover.OCICredentials{Server: "reg", Username: "user", Password: "pass"}},
		},
		Target: mover.Target{
			Chart: mover.ChartSpec{
				IntermediateBundle: &mover.IntermediateBundle{
					Path: "/tmp/target-dir",
				},
			},
		},
	}
	got := relok8sBundleSaveReq("/tmp/source-dir", "/tmp/target-dir", &api.Containers_ContainerAuth{Username: "user", Password: "pass", Registry: "reg"})
	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected relok8s bundle save request. got: %v, want: %v", got, want)
	}
}

func TestRelok8sBundleLoadReq(t *testing.T) {
	want := &mover.ChartMoveRequest{
		Source: mover.Source{
			Chart: mover.ChartSpec{
				IntermediateBundle: &mover.IntermediateBundle{
					Path: "/tmp/source-dir",
				},
			},
		},
		Target: mover.Target{
			Rules: mover.RewriteRules{
				Registry:         "gcr.io",
				RepositoryPrefix: "my-project/containers",
				ForcePush:        true,
			},
			Chart: mover.ChartSpec{
				Local: &mover.LocalChart{
					Path: "/tmp/target-dir",
				},
			},
			ContainersAuth: &mover.ContainersAuth{Credentials: &mover.OCICredentials{Server: "reg", Username: "user", Password: "pass"}},
		},
	}
	got := relok8sBundleLoadReq("/tmp/source-dir", "/tmp/target-dir", "gcr.io", "my-project/containers", &api.Containers_ContainerAuth{Username: "user", Password: "pass", Registry: "reg"})
	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected relok8s bundle load request. got: %v, want: %v", got, want)
	}
}
