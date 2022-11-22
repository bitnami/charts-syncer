// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package cmd

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Version", func() {
	Context("version is set", func() {
		var (
			stdout          *Buffer
			originalVersion string
		)

		BeforeEach(func() {
			stdout = NewBuffer()

			originalVersion = Version
			Version = "9.9.9"

			versionCmd.SetOut(stdout)
		})
		AfterEach(func() {
			Version = originalVersion
		})

		It("prints the version", func() {
			versionCmd.Run(versionCmd, []string{})
			Expect(stdout).To(Say("relok8s version: 9.9.9"))
		})
	})
})
