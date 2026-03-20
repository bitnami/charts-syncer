package oci

import (
	"reflect"
	"sort"
	"testing"

	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
)

func TestLooksLikeDockerImageTag(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		// Standard MAJOR.MINOR.PATCH-DISTRO-DVER-rREV
		{"13.16.0-photon-5-r19", true},
		{"7.4.1-debian-12-r6", true},
		{"1.27.3-ubuntu-22-r0", true},
		// MAJOR.MINOR.PATCH-BUILD-DISTRO-DVER-rREV (extra build/epoch segment)
		{"17.0.16-12-photon-5-r0", true},
		{"11.2.3-45-debian-12-r1", true},
		// MAJOR-DISTRO-DVER-rREV (single integer version)
		{"5-photon-5-r19", true},
		{"3-debian-12-r0", true},
		// Non-matching tags
		{"latest", false},
		{"stable", false},
		{"sha256:abc123", false},
		{"1.0.0", false},
		{"1.0.0-r1", false},
		{"1.0.0-debian", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := looksLikeDockerImageTag(tt.tag)
			if got != tt.want {
				t.Errorf("looksLikeDockerImageTag(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestListWithEntries(t *testing.T) {
	entries := map[string][]string{
		"chartA": {"1.0.1", "1.0.2"},
		"chartB": {"2.0.1", "2.0.2"},
		"chartC": {"0.0.1", "0.0.2"},
	}
	repo := Repo{
		entries: entries,
	}
	want := []string{"chartA", "chartB", "chartC"}
	got, err := repo.List()
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected list of charts names. got: %v, want: %v", got, want)
	}
}
