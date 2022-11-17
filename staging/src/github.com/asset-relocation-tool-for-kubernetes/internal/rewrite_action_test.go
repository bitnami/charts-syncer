// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package internal_test

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"helm.sh/helm/v3/pkg/chart"

	"helm.sh/helm/v3/pkg/chartutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal"
	"helm.sh/helm/v3/pkg/chart/loader"
)

var _ = Describe("RewriteAction", func() {
	Describe("TopLevelKey", func() {
		It("returns the first part of the path", func() {
			action := &internal.RewriteAction{
				Path:  ".alpha.bravo.charlie.delta",
				Value: "needle",
			}
			Expect(action.TopLevelKey()).To(Equal("alpha"))
		})
	})

	Describe("GetPathToMap", func() {
		It("returns the path without the final key", func() {
			action := &internal.RewriteAction{
				Path:  ".alpha.bravo.charlie.delta",
				Value: "needle",
			}
			Expect(action.GetPathToMap()).To(Equal(".alpha.bravo.charlie"))
		})
	})

	Describe("GetSubPathToMap", func() {
		It("returns the path without the final key and the top-level key", func() {
			action := &internal.RewriteAction{
				Path:  ".alpha.bravo.charlie.delta",
				Value: "needle",
			}
			Expect(action.GetSubPathToMap()).To(Equal(".bravo.charlie"))
		})
	})

	Describe("GetKey", func() {
		It("returns the last part of the path", func() {
			action := &internal.RewriteAction{
				Path:  ".alpha.bravo.charlie.delta",
				Value: "needle",
			}
			Expect(action.GetKey()).To(Equal("delta"))
		})
	})

	Describe("ToMap", func() {
		Context("one key", func() {
			It("becomes a flat map", func() {
				action := &internal.RewriteAction{
					Path:  ".alpha",
					Value: "needle",
				}

				haystack := action.ToMap()
				Expect(haystack).To(HaveKeyWithValue("alpha", "needle"))
			})
		})

		Context("two keys", func() {
			It("becomes a nested map", func() {
				action := &internal.RewriteAction{
					Path:  ".alpha.bravo",
					Value: "needle",
				}

				haystack := action.ToMap()
				Expect(haystack).To(HaveKey("alpha"))
				haystackLevelTwo := haystack["alpha"]
				Expect(haystackLevelTwo).To(HaveKeyWithValue("bravo", "needle"))
			})
		})

		Context("multiple keys", func() {
			It("becomes a deeply nested map", func() {
				action := &internal.RewriteAction{
					Path:  ".alpha.bravo.charlie.delta",
					Value: "needle",
				}

				haystack := action.ToMap()
				Expect(haystack).To(HaveKey("alpha"))

				var ok bool
				haystack, ok = haystack["alpha"].(internal.ValuesMap)
				Expect(ok).To(BeTrue())
				Expect(haystack).To(HaveKey("bravo"))

				haystack, ok = haystack["bravo"].(internal.ValuesMap)
				Expect(ok).To(BeTrue())
				Expect(haystack).To(HaveKey("charlie"))

				haystack, ok = haystack["charlie"].(internal.ValuesMap)
				Expect(ok).To(BeTrue())
				Expect(haystack).To(HaveKeyWithValue("delta", "needle"))
			})
		})
	})

	//Describe("FindChartDestination", func() {
	//	Context("action refers to a chart dependency", func() {
	//		It("returns the dependent chart", func() {
	//		})
	//	})
	//})
})

var update = flag.Bool("update-golden", false, "update golden files")

const fixturesRoot = "../test/fixtures/"

func TestApply(t *testing.T) {
	// The chart we are going to modify
	originalChart, err := loader.Load(filepath.Join(fixturesRoot, "3-levels-chart"))
	if err != nil {
		t.Fatal(err)
	}

	rewrites := []*internal.RewriteAction{
		{Path: ".image.repository", Value: "changed-parent"},
		{Path: ".subchart-1.image.repository", Value: "changed-subchart"},
		{Path: ".subchart-1.image.tag", Value: "updated-tag"},
		{Path: ".subchart-1.subchart-3.image.repository", Value: "changed-sub-sub-chart"},
		{Path: ".subchart-2.image.tag", Value: "updated-tag"},
	}

	// Apply changes to the original chart
	for _, r := range rewrites {
		if err := r.Apply(originalChart); err != nil {
			t.Fatal(err)
		}
	}

	// Package the updated chart
	gotChartPath, gotDigest, err := packageChart(originalChart)
	if err != nil {
		t.Fatal(err)
	}

	// The chart we want as result
	wantChartPath := filepath.Join("testdata", "applyoutput")
	wantChart, err := loader.Load(filepath.Join(wantChartPath, "3-levels-chart"))
	if err != nil {
		t.Fatal(err)
	}

	_, wantDigest, err := packageChart(wantChart)
	if err != nil {
		t.Fatal(err)
	}

	// Update fixtures by moving the newly generated chart into the output one for future comparisons
	if *update {
		if err := os.RemoveAll(wantChartPath); err != nil {
			t.Fatal(err)
		}

		if err := os.Rename(gotChartPath, wantChartPath); err != nil {
			t.Fatal(err)
		}
	}

	if gotDigest != wantDigest {
		t.Errorf("the resulting Chart does not match the fixture. got=%s, want=%s", gotDigest, wantDigest)
	}
}

// packageChart will return the chart in directory format as well as its content digest
func packageChart(chart *chart.Chart) (string, string, error) {
	chartDir, err := ioutil.TempDir("", "relok8s-test")
	if err != nil {
		return "", "", err
	}

	if err := chartutil.SaveDir(chart, chartDir); err != nil {
		return "", "", err
	}

	// Calculate the digest from the Chart files
	// the digest from the packaged chart will differ since tar.gz adds timestamps
	hasher := sha256.New()

	if err := filepath.Walk(chartDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := hasher.Write(data); err != nil {
				return err

			}
		}

		return nil
	}); err != nil {
		return "", "", err
	}

	return chartDir, hex.EncodeToString(hasher.Sum(nil)), nil
}
