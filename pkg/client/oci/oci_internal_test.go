package oci

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/bitnami-labs/charts-syncer/api"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
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
			ociRef := fmt.Sprintf("%s%s/%s", u.Host, u.Path, tc.ociPartialRef)
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
