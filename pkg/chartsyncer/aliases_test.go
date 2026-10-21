package chartsyncer_test

import (
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/bitnami/charts-syncer/pkg/chartsyncer"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/require"
	"github.com/vmware-labs/distribution-tooling-for-helm/pkg/imagelock"
	dtutils "github.com/vmware-labs/distribution-tooling-for-helm/pkg/utils"
	helmregistry "helm.sh/helm/v3/pkg/registry"
)

const aliasRevision = "18.6.0-debian-12-r13"

func aliasRegistry(t *testing.T) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(s.Close)
	return s
}

func aliasIndex(t *testing.T, version string) v1.ImageIndex {
	t.Helper()
	idx := mutate.IndexMediaType(empty.Index, types.OCIImageIndex)
	for _, arch := range []string{"amd64", "arm64"} {
		img, err := mutate.ConfigFile(empty.Image, &v1.ConfigFile{
			OS: "linux", Architecture: arch,
			Config: v1.Config{Labels: map[string]string{"version": version}},
		})
		require.NoError(t, err)
		idx = mutate.AppendManifests(idx, mutate.IndexAddendum{Add: img,
			Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: arch}}})
	}
	return idx
}

func pushAliasIndex(t *testing.T, idx v1.ImageIndex, ref string) {
	t.Helper()
	tag, err := name.NewTag(ref)
	require.NoError(t, err)
	require.NoError(t, remote.WriteIndex(tag, idx))
}

func publishAliasChart(t *testing.T, host string) {
	t.Helper()
	dir := t.TempDir()
	chartYAML := fmt.Sprintf("apiVersion: v2\nname: example\nversion: 1.0.0\nannotations:\n  images: |\n    - name: postgresql\n      image: %s/postgresql:%s\n", host, aliasRevision)
	values := fmt.Sprintf("image:\n  registry: %s\n  repository: postgresql\n  tag: %s\n", host, aliasRevision)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(chartYAML), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(values), 0600))
	tgz := filepath.Join(t.TempDir(), "example-1.0.0.tgz")
	require.NoError(t, dtutils.Tar(dir, tgz, dtutils.TarConfig{Prefix: "example"}))
	data, err := os.ReadFile(tgz)
	require.NoError(t, err)
	cli, err := helmregistry.NewClient(helmregistry.ClientOptPlainHTTP(), helmregistry.ClientOptWriter(io.Discard),
		helmregistry.ClientOptCredentialsFile(filepath.Join(t.TempDir(), "config.json")))
	require.NoError(t, err)
	_, err = cli.Push(data, host+"/charts/example:1.0.0")
	require.NoError(t, err)
}

func TestChartAliasSync(t *testing.T) {
	for _, scenario := range []string{"enabled", "disabled", "local", "filtered", "preserved"} {
		t.Run(scenario, func(t *testing.T) {
			srcReg, dstReg := aliasRegistry(t), aliasRegistry(t)
			srcHost := strings.TrimPrefix(srcReg.URL, "http://")
			dstHost := strings.TrimPrefix(dstReg.URL, "http://")
			srcRepo, dstRepo := srcHost+"/postgresql", dstHost+"/mirror/postgresql"
			fresh, old := aliasIndex(t, "new"), aliasIndex(t, "old")
			oldDigest, err := old.Digest()
			require.NoError(t, err)
			otherRevision := "18.6.0-debian-12-r12"
			for _, tag := range []string{aliasRevision, "18", "latest", otherRevision} {
				pushAliasIndex(t, fresh, srcRepo+":"+tag)
				// An existing primary image tag must not prevent a new chart
				// from being synced, and retains DT's overwrite behavior.
				pushAliasIndex(t, old, dstRepo+":"+tag)
			}
			publishAliasChart(t, srcHost)
			src := &apiv1.Source{Repo: &apiv1.Repo{Kind: apiv1.Kind_OCI, Url: srcReg.URL + "/charts", DisableChartsIndex: true}}
			dst := &apiv1.Target{
				Repo:       &apiv1.Repo{Kind: apiv1.Kind_OCI, Url: dstReg.URL + "/charts", DisableChartsIndex: true},
				Containers: &apiv1.Containers{Url: dstReg.URL + "/mirror"},
			}
			opts := []chartsyncer.Option{
				chartsyncer.WithSyncAliasTags(scenario != "disabled"), chartsyncer.WithSkipArtifacts(true),
				chartsyncer.WithUsePlainHTTP(true), chartsyncer.WithPreserveDigest(scenario == "preserved"),
			}
			if scenario == "filtered" {
				opts = append(opts, chartsyncer.WithContainerPlatforms([]string{"linux/amd64"}))
			}
			newSyncer := func(source *apiv1.Source, target *apiv1.Target) *chartsyncer.Syncer {
				s, syncErr := chartsyncer.New(source, target, append(opts, chartsyncer.WithWorkdir(t.TempDir()))...)
				require.NoError(t, syncErr)
				return s
			}
			if scenario == "local" {
				dir := t.TempDir()
				export := newSyncer(src, &apiv1.Target{Repo: &apiv1.Repo{Kind: apiv1.Kind_LOCAL, Path: dir}})
				require.NoError(t, export.SyncPendingCharts("example"))
				extracted := t.TempDir()
				bundle := filepath.Join(dir, "charts", "example-1.0.0.wrap.tgz")
				require.NoError(t, dtutils.Untar(bundle, extracted, dtutils.TarConfig{StripComponents: 1}))
				lock, lockErr := imagelock.FromYAMLFile(filepath.Join(extracted, "chart", "Images.lock"))
				require.NoError(t, lockErr)
				require.Len(t, lock.Images, 1)
				require.Equal(t, []string{"18", "latest"}, lock.Images[0].Aliases)
				srcReg.Close()
				src = &apiv1.Source{Repo: &apiv1.Repo{Kind: apiv1.Kind_LOCAL, Path: dir}}
			}
			require.NoError(t, newSyncer(src, dst).SyncPendingCharts("example"))
			published, err := crane.Digest(dstRepo + ":" + aliasRevision)
			require.NoError(t, err)
			require.NotEqual(t, oldDigest.String(), published)
			wantAlias := published
			if scenario == "disabled" {
				wantAlias = oldDigest.String()
			}
			for _, tag := range []string{"18", "latest"} {
				got, digestErr := crane.Digest(dstRepo + ":" + tag)
				require.NoError(t, digestErr)
				require.Equal(t, wantAlias, got)
			}
			untouched, err := crane.Digest(dstRepo + ":" + otherRevision)
			require.NoError(t, err)
			require.Equal(t, oldDigest.String(), untouched)
			sourceDigest, err := fresh.Digest()
			require.NoError(t, err)
			if scenario == "preserved" {
				require.Equal(t, sourceDigest.String(), published)
			}
			if scenario == "filtered" {
				require.NotEqual(t, sourceDigest.String(), published)
				manifest, manifestErr := crane.Manifest(dstRepo + ":" + aliasRevision)
				require.NoError(t, manifestErr)
				idx, manifestErr := v1.ParseIndexManifest(strings.NewReader(string(manifest)))
				require.NoError(t, manifestErr)
				require.Len(t, idx.Manifests, 1)
				require.Equal(t, "amd64", idx.Manifests[0].Platform.Architecture)
			}
			// A matching chart version skips image processing, even when its
			// aliases have since become stale at the destination.
			pushAliasIndex(t, old, dstRepo+":latest")
			require.ErrorIs(t, newSyncer(src, dst).SyncPendingCharts("example"), chartsyncer.ErrNoChartsToSync)
			latest, err := crane.Digest(dstRepo + ":latest")
			require.NoError(t, err)
			require.Equal(t, oldDigest.String(), latest)
		})
	}
}
