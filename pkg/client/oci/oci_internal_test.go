package oci

import (
	"fmt"
	"github.com/bitnami-labs/charts-syncer/api"
	_ "github.com/docker/distribution/registry/storage/driver/inmemory"
	"net/url"
	"testing"
)

var (
	ociRepo = &api.Repo{
		Kind: api.Kind_OCI,
		Auth: &api.Auth{
			Username: "user",
			Password: "password",
		},
	}
)

func TestSplitOciReference(t *testing.T) {
	tests := []struct {
		desc   string
		ociRef string
		want   ociReference
	}{
		{
			desc:   "should split path and tag",
			ociRef: "demo.goharbor.io/my-project/index:latest",
			want:   ociReference{path: "demo.goharbor.io/my-project/index", tag: "latest"},
		},
		{
			desc:   "should split path and digest",
			ociRef: "demo.goharbor.io/my-project/index@sha256:deadbeef",
			want:   ociReference{path: "demo.goharbor.io/my-project/index", tag: "deadbeef"},
		},
		{
			desc:   "should split path without tag or digest",
			ociRef: "demo.goharbor.io/my-project/index",
			want:   ociReference{path: "demo.goharbor.io/my-project/index"},
		},
		{
			desc:   "should split when host contains port and tag is provided",
			ociRef: "127.0.0.1:56789/my-project/index:latest",
			want:   ociReference{path: "127.0.0.1:56789/my-project/index", tag: "latest"},
		},
		{
			desc:   "should split when host contains port and tag is not provided",
			ociRef: "127.0.0.1:56789/my-project/index",
			want:   ociReference{path: "127.0.0.1:56789/my-project/index"},
		},
		{
			desc:   "should split when host contains port and digest is provided",
			ociRef: "127.0.0.1:56789/my-project/index@sha256:deadbeef",
			want:   ociReference{path: "127.0.0.1:56789/my-project/index", tag: "deadbeef"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			got := splitOciReference(tc.ociRef)
			if got != tc.want {
				t.Errorf("wrong splited OCI reference. got: %v, want: %v", got, tc.want)
			}
		})
	}
}

func TestOciReferenceExists(t *testing.T) {
	tests := []struct {
		desc          string
		ociPartialRef string // to be added to repo url returned by PrepareOciServer
		pushTestAsset bool
		want          bool
	}{
		{
			desc:          "Artifact should exist",
			ociPartialRef: "index:latest",
			pushTestAsset: true,
			want:          true,
		},
		{
			desc:          "Artifact should not exist",
			ociPartialRef: "non-existing-index:latest",
			pushTestAsset: false,
			want:          false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			PrepareOciServer(t, ociRepo)
			u, err := url.Parse(ociRepo.Url)
			if err != nil {
				t.Fatal(err)
			}
			ociRef := fmt.Sprintf("%s%s/%s",u.Host,u.Path, tc.ociPartialRef)
			if tc.pushTestAsset {
				PushFileToOCI(t, "../../../testdata/oci/index.json", ociRef)
			}
			got, err := ociReferenceExists(u, ociRef, ociRepo.GetAuth().GetUsername(), ociRepo.GetAuth().GetPassword(), true)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("wrong result from OCI reference existence check. got: %v, want: %v", got, tc.want)
			}
		})
	}
}
