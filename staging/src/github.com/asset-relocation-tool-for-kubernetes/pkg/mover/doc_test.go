// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
)

// Move a chart and its dependencies to another registry and or repository
func Example() {
	// i.e ./mariadb-7.5.relocated.tgz
	destinationPath := "%s-%s.relocated.tgz"

	// Initialize the Mover action
	chartMover, err := NewChartMover(
		&ChartMoveRequest{
			Source: Source{
				// The Helm Chart can be provided in either tarball or directory form
				Chart: ChartSpec{Local: &LocalChart{Path: "./helm_chart.tgz"}},
				// path to file containing rules such as // {{.image.registry}}:{{.image.tag}}
				ImageHintsFile: "./image-hints.yaml",
			},
			Target: Target{
				Chart: ChartSpec{Local: &LocalChart{Path: destinationPath}},
				// Where to push and how to rewrite the found images
				// i.e docker.io/bitnami/mariadb => myregistry.com/myteam/mariadb
				Rules: RewriteRules{
					Registry:         "myregistry.com",
					RepositoryPrefix: "/myteam",
				},
			},
		},
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Perform the push, rewrite and repackage of the Helm Chart
	err = chartMover.Move()
	if err != nil {
		fmt.Println(err)
		return
	}
}

// Save a chart and all its dependencies into a intermediate bundle tarball
func Example_save() {
	// Initialize the Mover action
	chartMover, err := NewChartMover(
		&ChartMoveRequest{
			Source: Source{
				// The Helm Chart can be provided in either tarball or directory form
				Chart: ChartSpec{Local: &LocalChart{Path: "./helm_chart.tgz"}},
				// path to file containing rules such as // {{.image.registry}}:{{.image.tag}}
				ImageHintsFile: "./image-hints.yaml",
			},
			Target: Target{
				Chart: ChartSpec{
					// The target intermediate bundle path to place the charts and all its dependencies
					IntermediateBundle: &IntermediateBundle{Path: "helm_chart.intermediate-bundle.tar"},
				},
				// No rewrite rules, as this only saves the chart and its dependencies as is
			},
		},
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Save the chart, hints file and all container images
	// into `helm_chart.intermediate-bundle.tar`
	//
	// So we get something like:
	// $ tar tvf helm_chart.intermediate-bundle.tar
	// -rw-r--r-- 0/0             201 1970-01-01 01:00 hints.yaml
	// -rw-r--r-- 0/0             349 1970-01-01 01:00 original-chart/...
	// -rw-r--r-- 0/0          773120 1970-01-01 01:00 images.tar
	//
	// For how the move completes using an intermediate bundle input see the Load example
	err = chartMover.Move()
	if err != nil {
		fmt.Println(err)
		return
	}
}

// Load a chart and its dependencies from a intermediate bundle tarball
// into a registry repository
func Example_load() {
	// i.e ./mariadb-7.5.relocated.tgz
	destinationPath := "%s-%s.relocated.tgz"

	// Initialize the Mover action
	chartMover, err := NewChartMover(
		&ChartMoveRequest{
			Source: Source{
				Chart: ChartSpec{
					// The intermediate source tarball where the source chart and
					// container images are present. See Save example
					IntermediateBundle: &IntermediateBundle{Path: "helm_chart.intermediate-bundle.tar"},
				},
				// no path to hints file as it is already coming inside the intermediate bundle
			},
			Target: Target{
				Chart: ChartSpec{Local: &LocalChart{Path: destinationPath}},
				// Where to push and how to rewrite the found images
				// i.e docker.io/bitnami/mariadb => myregistry.com/myteam/mariadb
				Rules: RewriteRules{
					Registry:         "myregistry.com",
					RepositoryPrefix: "/myteam",
				},
			},
		},
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Perform the push, rewrite and repackage of the Helm Chart
	// All origin data is taken from within the source intermediate bundle
	err = chartMover.Move()
	if err != nil {
		fmt.Println(err)
		return
	}
}
