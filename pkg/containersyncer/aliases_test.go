package containersyncer

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	apiv1 "github.com/bitnami/charts-syncer/gen/proto/v1"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/stretchr/testify/require"
)

const testRevision = "18.6.0-debian-12-r13"

func testRegistry(t *testing.T, rejectAlias *atomic.Bool) *httptest.Server {
	t.Helper()
	reg := registry.New(registry.Logger(log.New(io.Discard, "", 0)))
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rejectAlias != nil && rejectAlias.Load() && r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/manifests/18") {
			http.Error(w, "alias update denied", http.StatusForbidden)
			return
		}
		reg.ServeHTTP(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func testImage(t *testing.T, version string) v1.Image {
	t.Helper()
	img, err := mutate.ConfigFile(empty.Image, &v1.ConfigFile{OS: "linux", Architecture: "amd64", Config: v1.Config{Labels: map[string]string{"version": version}}})
	require.NoError(t, err)
	return img
}

func newAliasSyncer(t *testing.T, src *apiv1.Source, dst *apiv1.Target, enabled bool) *Syncer {
	t.Helper()
	s, err := New(src, dst, WithSyncAliasTags(enabled), WithPreserveDigest(true), WithSkipArtifacts(true), WithUsePlainHTTP(true), WithWorkdir(t.TempDir()))
	require.NoError(t, err)
	return s
}

func TestAliasSyncSelectionAndLocalBundles(t *testing.T) {
	for _, scenario := range []string{"enabled", "disabled", "local"} {
		t.Run(scenario, func(t *testing.T) {
			srcReg, dstReg := testRegistry(t, nil), testRegistry(t, nil)
			srcRepo := strings.TrimPrefix(srcReg.URL, "http://") + "/postgresql"
			dstRepo := strings.TrimPrefix(dstReg.URL, "http://") + "/postgresql"
			fresh, old := testImage(t, "new"), testImage(t, "old")
			require.NoError(t, crane.Push(fresh, srcRepo+":"+testRevision))
			require.NoError(t, crane.Push(fresh, srcRepo+":18"))
			require.NoError(t, crane.Push(fresh, srcRepo+":latest"))
			require.NoError(t, crane.Push(old, dstRepo+":18"))
			require.NoError(t, crane.Push(old, dstRepo+":latest"))
			src := &apiv1.Source{Containers: &apiv1.Containers{Url: srcReg.URL}}
			dst := &apiv1.Target{Containers: &apiv1.Containers{Url: dstReg.URL}}
			if scenario == "local" {
				dir := t.TempDir()
				export := newAliasSyncer(t, src, &apiv1.Target{Repo: &apiv1.Repo{Kind: apiv1.Kind_LOCAL, Path: dir}}, true)
				pending, err := export.DiffPendingContainers("postgresql")
				require.NoError(t, err)
				require.NoError(t, export.SyncPendingContainers(pending))
				require.FileExists(t, filepath.Join(dir, "containers", "postgresql-"+testRevision+".container.wrap.tgz"))
				srcReg.Close()
				src = &apiv1.Source{Repo: &apiv1.Repo{Kind: apiv1.Kind_LOCAL, Path: dir}}
			}
			s := newAliasSyncer(t, src, dst, scenario != "disabled")
			pending, err := s.DiffPendingContainers("postgresql")
			require.NoError(t, err)
			require.Len(t, pending, 1)
			require.Equal(t, []string{testRevision}, pending[0].Tags)
			require.NoError(t, s.SyncPendingContainers(pending))
			got, err := crane.Digest(dstRepo + ":18")
			require.NoError(t, err)
			want, err := fresh.Digest()
			require.NoError(t, err)
			if scenario == "disabled" {
				want, err = old.Digest()
				require.NoError(t, err)
			}
			require.Equal(t, want.String(), got)
			latest, err := crane.Digest(dstRepo + ":latest")
			require.NoError(t, err)
			oldDigest, err := old.Digest()
			require.NoError(t, err)
			require.Equal(t, want.String(), latest)

			// Even enabling the feature later must not repair a skipped revision.
			require.NoError(t, crane.Push(old, dstRepo+":18"))
			require.NoError(t, crane.Push(old, dstRepo+":latest"))
			s = newAliasSyncer(t, src, dst, true)
			pending, err = s.DiffPendingContainers("postgresql")
			require.NoError(t, err)
			require.Empty(t, pending)
			for _, alias := range []string{"18", "latest"} {
				got, err = crane.Digest(dstRepo + ":" + alias)
				require.NoError(t, err)
				require.Equal(t, oldDigest.String(), got)
			}
		})
	}
}

func TestAliasFailureCanBeRetried(t *testing.T) {
	var reject atomic.Bool
	reject.Store(true)
	srcReg, dstReg := testRegistry(t, nil), testRegistry(t, &reject)
	srcRepo := strings.TrimPrefix(srcReg.URL, "http://") + "/postgresql"
	img := testImage(t, "new")
	require.NoError(t, crane.Push(img, srcRepo+":"+testRevision))
	require.NoError(t, crane.Push(img, srcRepo+":18"))
	src := &apiv1.Source{Containers: &apiv1.Containers{Url: srcReg.URL}}
	dst := &apiv1.Target{Containers: &apiv1.Containers{Url: dstReg.URL}}
	s := newAliasSyncer(t, src, dst, true)
	pending, err := s.DiffPendingContainers("postgresql")
	require.NoError(t, err)
	require.ErrorContains(t, s.SyncPendingContainers(pending), "publishing alias")
	reject.Store(false)
	s = newAliasSyncer(t, src, dst, true)
	pending, err = s.DiffPendingContainers("postgresql")
	require.NoError(t, err)
	require.Len(t, pending, 1, "failed revision must still be pending")
	require.NoError(t, s.SyncPendingContainers(pending))
}
