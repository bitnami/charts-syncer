package oci

import (
	"reflect"
	"sort"
	"testing"

	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
)

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
