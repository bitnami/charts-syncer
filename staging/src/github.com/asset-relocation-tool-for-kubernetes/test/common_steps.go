// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package test

import (
	"os/exec"
	"strings"
	"time"

	"github.com/bunniesandbeatings/goerkin"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var (
	ChartMoverBinaryPath string
	CommandSession       *gexec.Session
)

var _ = ginkgo.BeforeSuite(func() {
	var err error
	ChartMoverBinaryPath, err = gexec.Build(
		"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes",
		"-ldflags",
		"-X github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/cmd.Version=1.2.3",
	)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
})

var _ = ginkgo.AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func DefineCommonSteps(define goerkin.Definitions) {
	define.When(`^running relok8s (.*)$`, func(argString string) {
		args := strings.Split(argString, " ")
		command := exec.Command(ChartMoverBinaryPath, args...)
		var err error
		CommandSession, err = gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	define.Then(`^the command exits without error$`, func() {
		gomega.Eventually(CommandSession, time.Minute).Should(gexec.Exit(0))
	})

	define.Then(`^the command exits with an error$`, func() {
		gomega.Eventually(CommandSession, time.Minute).Should(gexec.Exit(1))
	})

	define.Then(`^it prints the usage$`, func() {
		gomega.Expect(CommandSession.Err).To(gbytes.Say("Usage:"))
		gomega.Expect(CommandSession.Err).To(gbytes.Say("relok8s chart move <chart> \\[flags\\]"))
	})
}
