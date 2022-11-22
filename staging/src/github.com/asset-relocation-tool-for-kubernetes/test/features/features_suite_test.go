// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package features_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFeatures(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Features Suite")
}
