// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package features_test

import (
	"regexp"

	. "github.com/bunniesandbeatings/goerkin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/test"
)

var _ = Describe("relok8s chart move input validation", func() {
	steps := NewSteps()

	Scenario("missing helm chart", func() {
		steps.When("running relok8s chart move")
		steps.Then("the command exits with an error")
		steps.And("it says the chart is missing")
		steps.And("it prints the usage")
	})

	Scenario("helm chart does not exist", func() {
		steps.When("running relok8s chart move ../fixtures/does-not-exist --repo-prefix cyberdyne-corp")
		steps.Then("the command exits with an error")
		steps.And("it says the chart does not exist")
		steps.And("it prints the usage")
	})

	Scenario("helm chart is empty directory", func() {
		steps.When("running relok8s chart move ../fixtures/empty-directory --repo-prefix cyberdyne-corp")
		steps.Then("the command exits with an error")
		steps.And("it says the chart is missing a critical file")
		steps.And("it prints the usage")
	})

	Scenario("missing image patterns file", func() {
		steps.When("running relok8s chart move ../fixtures/wordpress-11.0.4.tgz --repo-prefix cyberdyne-corp")
		steps.Then("the command exits with an error")
		steps.And("it says the image patterns file is missing")
		steps.And("it prints the usage")
	})

	Scenario("too many arguments", func() {
		steps.When("running relok8s chart move ../fixtures/wordpress-11.0.4.tgz --repo-prefix cyberdyne-corp")
		steps.Then("the command exits with an error")
		steps.And("it says the image patterns file is missing")
		steps.And("it prints the usage")
	})

	Scenario("no rules are given", func() {
		steps.When("running relok8s chart move ../fixtures/wordpress-11.0.4.tgz --image-patterns ../fixtures/wordpress-11.0.4.images.yaml --registry reg.vmware.com extra arguments")
		steps.Then("the command exits with an error")
		steps.And("it says that there are too many args")
		steps.And("it prints the usage")
	})

	Scenario("no rules are given", func() {
		steps.When("running relok8s chart move ../fixtures/wordpress-11.0.4.tgz --image-patterns ../fixtures/wordpress-11.0.4.images.yaml")
		steps.Then("the command exits with an error")
		steps.And("it says that the rules are missing")
		steps.And("it prints the usage")
	})

	Scenario("invalid registry", func() {
		steps.When("running relok8s chart move ../fixtures/wordpress-11.0.4.tgz --image-patterns ../fixtures/wordpress-11.0.4.images.yaml --registry what:is:this?")
		steps.Then("the command exits with an error")
		steps.And("it says that the registry is invalid")
		steps.And("it prints the usage")
	})

	Scenario("invalid repo prefix", func() {
		steps.When("running relok8s chart move ../fixtures/wordpress-11.0.4.tgz --image-patterns ../fixtures/wordpress-11.0.4.images.yaml --repo-prefix 'What+is$this???'")
		steps.Then("the command exits with an error")
		steps.And("it says that the repository prefix is invalid")
		steps.And("it prints the usage")
	})

	steps.Define(func(define Definitions) {
		test.DefineCommonSteps(define)

		define.Then(`^it says the chart is missing$`, func() {
			Expect(test.CommandSession.Err).To(Say("Error: requires a chart argument"))
		})

		define.Then(`^it says the chart does not exist$`, func() {
			Expect(test.CommandSession.Err).To(Say("Error: failed to load Helm Chart at \"../fixtures/does-not-exist\": stat ../fixtures/does-not-exist: no such file or directory"))
		})

		define.Then(`^it says the chart is missing a critical file$`, func() {
			Expect(test.CommandSession.Err).To(Say("Error: failed to load Helm Chart at \"../fixtures/empty-directory\": Chart.yaml file is missing"))
		})

		define.Then(`^it says the image patterns file is missing$`, func() {
			Expect(test.CommandSession.Err).To(Say("Error: image patterns file is required. Please try again with '--image-patterns <image patterns file>'"))
		})

		define.Then(`^it says that there are too many args$`, func() {
			Expect(test.CommandSession.Err).To(Say(`expected 1 chart argument, received \d args`))
		})

		define.Then(`^it says that the rules are missing$`, func() {
			Expect(test.CommandSession.Err).To(Say("Error: at least one rewrite rule must be given. Please try again with --registry and/or --repo-prefix"))
		})

		define.Then(`^it says that the registry is invalid$`, func() {
			Expect(test.CommandSession.Err).To(Say(regexp.QuoteMeta("Error: registry rule is not valid: registries must be valid RFC 3986 URI authorities: what:is:this?")))
		})

		define.Then(`^it says that the repository prefix is invalid$`, func() {
			Expect(test.CommandSession.Err).To(Say(regexp.QuoteMeta("Error: repository prefix rule is not valid: repository can only contain the runes `abcdefghijklmnopqrstuvwxyz0123456789_-./`: 'What+is$this???'")))
		})
	})
})
