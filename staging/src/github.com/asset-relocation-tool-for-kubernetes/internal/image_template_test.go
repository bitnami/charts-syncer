// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package internal_test

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/test"
)

const imageDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

var _ = Describe("internal.NewFromString", func() {
	Context("Empty string", func() {
		It("returns an error", func() {
			_, err := internal.NewFromString("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"\": missing repo or a registry fragment"))
		})
	})
	Context("Invalid template", func() {
		It("returns an error", func() {
			_, err := internal.NewFromString("not a template")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"not a template\": missing repo or a registry fragment"))
		})
	})
	Context("Single template", func() {
		It("parses successfully", func() {
			imageTemplate, err := internal.NewFromString("{{ .image }}")
			Expect(err).ToNot(HaveOccurred())
			Expect(imageTemplate.Raw).To(Equal("{{ .image }}"))
			Expect(imageTemplate.RegistryTemplate).To(BeEmpty())
			Expect(imageTemplate.RepositoryTemplate).To(BeEmpty())
			Expect(imageTemplate.RegistryAndRepositoryTemplate).To(Equal(".image"))
			Expect(imageTemplate.TagTemplate).To(BeEmpty())
			Expect(imageTemplate.DigestTemplate).To(BeEmpty())

			By("translating the template into index format", func() {
				Expect(imageTemplate.Template.Name()).To(Equal("{{ index . \"image\" }}"))
			})
		})
	})
	Context("Image and tag", func() {
		It("parses successfully", func() {
			imageTemplate, err := internal.NewFromString("{{ .image }}:{{ .tag }}")
			Expect(err).ToNot(HaveOccurred())
			Expect(imageTemplate.Raw).To(Equal("{{ .image }}:{{ .tag }}"))
			Expect(imageTemplate.RegistryTemplate).To(BeEmpty())
			Expect(imageTemplate.RepositoryTemplate).To(BeEmpty())
			Expect(imageTemplate.RegistryAndRepositoryTemplate).To(Equal(".image"))
			Expect(imageTemplate.TagTemplate).To(Equal(".tag"))
			Expect(imageTemplate.DigestTemplate).To(BeEmpty())

			By("translating the template into index format", func() {
				Expect(imageTemplate.Template.Name()).To(Equal("{{ index . \"image\" }}:{{ index . \"tag\" }}"))
			})
		})
	})
	Context("Image and multiple tags", func() {
		It("returns an error", func() {
			_, err := internal.NewFromString("{{ .image }}:{{ .tag1 }}:{{ .tag2 }}")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"{{ .image }}:{{ .tag1 }}:{{ .tag2 }}\": too many tag template matches"))
		})
	})
	Context("Image and digest", func() {
		It("parses successfully", func() {
			imageTemplate, err := internal.NewFromString("{{ .image }}@{{ .digest }}")
			Expect(err).ToNot(HaveOccurred())
			Expect(imageTemplate.Raw).To(Equal("{{ .image }}@{{ .digest }}"))
			Expect(imageTemplate.RegistryTemplate).To(BeEmpty())
			Expect(imageTemplate.RepositoryTemplate).To(BeEmpty())
			Expect(imageTemplate.RegistryAndRepositoryTemplate).To(Equal(".image"))
			Expect(imageTemplate.TagTemplate).To(BeEmpty())
			Expect(imageTemplate.DigestTemplate).To(Equal(".digest"))

			By("translating the template into index format", func() {
				Expect(imageTemplate.Template.Name()).To(Equal("{{ index . \"image\" }}@{{ index . \"digest\" }}"))
			})
		})
	})
	Context("Image and multiple digests", func() {
		It("returns an error", func() {
			_, err := internal.NewFromString("{{ .image }}@{{ .digest1 }}@{{ .digest2 }}")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"{{ .image }}@{{ .digest1 }}@{{ .digest2 }}\": too many digest template matches"))
		})
	})
	Context("registry, image, and tag", func() {
		It("parses successfully", func() {
			imageTemplate, err := internal.NewFromString("{{ .registry }}/{{ .image }}:{{ .tag }}")
			Expect(err).ToNot(HaveOccurred())
			Expect(imageTemplate.Raw).To(Equal("{{ .registry }}/{{ .image }}:{{ .tag }}"))
			Expect(imageTemplate.RegistryTemplate).To(Equal(".registry"))
			Expect(imageTemplate.RepositoryTemplate).To(Equal(".image"))
			Expect(imageTemplate.RegistryAndRepositoryTemplate).To(BeEmpty())
			Expect(imageTemplate.TagTemplate).To(Equal(".tag"))
			Expect(imageTemplate.DigestTemplate).To(BeEmpty())

			By("translating the template into index format", func() {
				Expect(imageTemplate.Template.Name()).To(Equal("{{ index . \"registry\" }}/{{ index . \"image\" }}:{{ index . \"tag\" }}"))
			})
		})
	})
	Context("Too many templates", func() {
		It("returns an error", func() {
			_, err := internal.NewFromString("{{ .a }}/{{ .b }}/{{ .c }}/{{ .d }}")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to parse image template \"{{ .a }}/{{ .b }}/{{ .c }}/{{ .d }}\": more fragments than expected"))
		})
	})
	Context("Subchart with dashes", func() {
		It("parses successfully", func() {
			imageTemplate, err := internal.NewFromString("{{ .sub-chart.image.registry }}/{{ .sub-chart.image.repository }}:{{ .sub-chart.image.tag }}")
			Expect(err).ToNot(HaveOccurred())
			Expect(imageTemplate.Raw).To(Equal("{{ .sub-chart.image.registry }}/{{ .sub-chart.image.repository }}:{{ .sub-chart.image.tag }}"))
			Expect(imageTemplate.RegistryTemplate).To(Equal(".sub-chart.image.registry"))
			Expect(imageTemplate.RepositoryTemplate).To(Equal(".sub-chart.image.repository"))
			Expect(imageTemplate.RegistryAndRepositoryTemplate).To(BeEmpty())
			Expect(imageTemplate.TagTemplate).To(Equal(".sub-chart.image.tag"))
			Expect(imageTemplate.DigestTemplate).To(BeEmpty())

			By("translating the template into index format", func() {
				Expect(imageTemplate.Template.Name()).To(Equal("{{ index . \"sub-chart\" \"image\" \"registry\" }}/{{ index . \"sub-chart\" \"image\" \"repository\" }}:{{ index . \"sub-chart\" \"image\" \"tag\" }}"))
			})
		})
	})
})

type TableInput struct {
	ParentChart *test.ChartSeed
	Template    string
}
type TableOutput struct {
	Image          string
	RewrittenImage string
	Actions        []*internal.RewriteAction
}

var (
	imageAlone = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"image": "ubuntu:latest",
			},
		},
		Template: "{{ .image }}",
	}
	imageAndTag = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"image": "petewall/amazingapp",
				"tag":   "latest",
			},
		},
		Template: "{{ .image }}:{{ .tag }}",
	}
	registryAndImage = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"registry": "quay.io",
				"image":    "proxy/nginx",
			},
		},
		Template: "{{ .registry }}/{{ .image }}",
	}
	registryImageAndTag = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"registry": "quay.io",
				"image":    "busycontainers/busybox",
				"tag":      "busiest",
			},
		},
		Template: "{{ .registry }}/{{ .image }}:{{ .tag }}",
	}
	imageAndDigest = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"image":  "petewall/platformio",
				"digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
		Template: "{{ .image }}@{{ .digest }}",
	}

	nestedValues = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"image": map[string]interface{}{
					"registry":   "docker.io",
					"repository": "bitnami/wordpress",
					"tag":        "1.2.3",
				},
			},
		},
		Template: "{{ .image.registry }}/{{ .image.repository }}:{{ .image.tag }}",
	}

	dependencyRegistryImageAndTag = &TableInput{
		ParentChart: &test.ChartSeed{
			Values: map[string]interface{}{
				"registry": "quay.io",
				"image":    "busycontainers/busybox",
				"tag":      "busiest",
			},
			Dependencies: []*test.ChartSeed{
				{
					Name: "lazy-chart",
					Values: map[string]interface{}{
						"registry": "index.docker.io",
						"image":    "lazycontainers/lazybox",
						"tag":      "laziest",
					},
				},
			},
		},
		Template: "{{ .lazy-chart.registry }}/{{ .lazy-chart.image }}:{{ .lazy-chart.tag }}",
	}

	parentWithGlobalRegistry = &TableInput{
		ParentChart: &test.ChartSeed{
			Name: "parent",
			Values: map[string]interface{}{
				"registry": "docker.io",
			},
			Dependencies: []*test.ChartSeed{
				{
					Name: "subchart",
					Values: map[string]interface{}{
						"image": "mycompany/coolapp",
						"tag":   "newest",
					},
				},
			},
		},
		Template: "{{ .registry }}/{{ .subchart.image }}:{{ .subchart.tag }}",
	}

	parentWithGlobalRegistryAndPath = &TableInput{
		ParentChart: &test.ChartSeed{
			Name: "parent",
			Values: map[string]interface{}{
				"registry": "tenancy.registry.com/companyone",
			},
			Dependencies: []*test.ChartSeed{
				{
					Name: "subchart",
					Values: map[string]interface{}{
						"image": "super-app",
						"tag":   "1.2.3",
					},
				},
			},
		},
		Template: "{{ .registry }}/{{ .subchart.image }}:{{ .subchart.tag }}",
	}

	registryRule          = &internal.OCIImageLocation{Registry: "registry.vmware.com"}
	repositoryPrefixRule  = &internal.OCIImageLocation{RepositoryPrefix: "my-company"}
	registryAndPrefixRule = &internal.OCIImageLocation{Registry: "registry.vmware.com", RepositoryPrefix: "my-company"}
)

var _ = DescribeTable("Rewrite Actions",
	func(input *TableInput, rules *internal.OCIImageLocation, expected *TableOutput) {
		var (
			err           error
			chart         = test.MakeChart(input.ParentChart)
			template      *internal.ImageTemplate
			originalImage name.Reference
			actions       []*internal.RewriteAction
		)

		By("parsing the template string", func() {
			template, err = internal.NewFromString(input.Template)
			Expect(err).ToNot(HaveOccurred())
			Expect(template.Raw).To(Equal(input.Template))
			Expect(template.Template).ToNot(BeNil())
		})

		By("rendering from values", func() {
			originalImage, err = template.Render(chart, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(originalImage).ToNot(BeNil())
			Expect(originalImage.Name()).To(Equal(expected.Image))
		})

		By("generating the rewrite rules", func() {
			actions, err = template.Apply(originalImage.Context(), imageDigest, rules)
			Expect(err).ToNot(HaveOccurred())
			Expect(actions).To(HaveLen(len(expected.Actions)))
			Expect(actions).To(ContainElements(expected.Actions))
		})

		By("rendering the rewritten image", func() {
			rewrittenImage, err := template.Render(chart, false, actions...)
			Expect(err).ToNot(HaveOccurred())
			Expect(rewrittenImage).ToNot(BeNil())
			Expect(rewrittenImage.Name()).To(Equal(expected.RewrittenImage))
		})
	},
	Entry("image alone, registry only", imageAlone, registryRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: fmt.Sprintf("registry.vmware.com/library/ubuntu@%s", imageDigest),
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: fmt.Sprintf("registry.vmware.com/library/ubuntu@%s", imageDigest),
			},
		},
	}),
	Entry("image alone, repository prefix only", imageAlone, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: fmt.Sprintf("index.docker.io/my-company/ubuntu@%s", imageDigest),
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: fmt.Sprintf("index.docker.io/my-company/ubuntu@%s", imageDigest),
			},
		},
	}),
	Entry("image alone, registry and prefix", imageAlone, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/library/ubuntu:latest",
		RewrittenImage: fmt.Sprintf("registry.vmware.com/my-company/ubuntu@%s", imageDigest),
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: fmt.Sprintf("registry.vmware.com/my-company/ubuntu@%s", imageDigest),
			},
		},
	}),
	Entry("image and tag, registry only", imageAndTag, registryRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/petewall/amazingapp:latest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/amazingapp",
			},
		},
	}),
	Entry("image and tag, repository prefix only", imageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "index.docker.io/my-company/amazingapp:latest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/my-company/amazingapp",
			},
		},
	}),
	Entry("image and tag, registry and prefix", imageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/amazingapp:latest",
		RewrittenImage: "registry.vmware.com/my-company/amazingapp:latest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/my-company/amazingapp",
			},
		},
	}),
	Entry("registry and image, registry only", registryAndImage, registryRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: fmt.Sprintf("registry.vmware.com/proxy/nginx@%s", imageDigest),
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image",
				Value: fmt.Sprintf("proxy/nginx@%s", imageDigest),
			},
		},
	}),
	Entry("registry and image, repository prefix only", registryAndImage, repositoryPrefixRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: fmt.Sprintf("quay.io/my-company/nginx@%s", imageDigest),
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: fmt.Sprintf("my-company/nginx@%s", imageDigest),
			},
		},
	}),
	Entry("registry and image, registry and prefix", registryAndImage, registryAndPrefixRule, &TableOutput{
		Image:          "quay.io/proxy/nginx:latest",
		RewrittenImage: fmt.Sprintf("registry.vmware.com/my-company/nginx@%s", imageDigest),
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image",
				Value: fmt.Sprintf("my-company/nginx@%s", imageDigest),
			},
		},
	}),
	Entry("registry, image, and tag, registry only", registryImageAndTag, registryRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/busycontainers/busybox:busiest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("registry, image, and tag, repository prefix only", registryImageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "quay.io/my-company/busybox:busiest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "my-company/busybox",
			},
		},
	}),
	Entry("registry, image, and tag, registry and prefix", registryImageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "quay.io/busycontainers/busybox:busiest",
		RewrittenImage: "registry.vmware.com/my-company/busybox:busiest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image",
				Value: "my-company/busybox",
			},
		},
	}),
	Entry("image and digest, registry only", imageAndDigest, registryRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/petewall/platformio",
			},
		},
	}),
	Entry("image and digest, repository prefix only", imageAndDigest, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "index.docker.io/my-company/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "index.docker.io/my-company/platformio",
			},
		},
	}),
	Entry("image and digest, registry and prefix", imageAndDigest, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/petewall/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RewrittenImage: "registry.vmware.com/my-company/platformio@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image",
				Value: "registry.vmware.com/my-company/platformio",
			},
		},
	}),
	Entry("nested values, registry only", nestedValues, registryRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "registry.vmware.com/bitnami/wordpress:1.2.3",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("nested values, repository prefix only", nestedValues, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "index.docker.io/my-company/wordpress:1.2.3",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image.repository",
				Value: "my-company/wordpress",
			},
		},
	}),
	Entry("nested values, registry and prefix", nestedValues, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/bitnami/wordpress:1.2.3",
		RewrittenImage: "registry.vmware.com/my-company/wordpress:1.2.3",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".image.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".image.repository",
				Value: "my-company/wordpress",
			},
		},
	}),
	Entry("dependency image and digest, registry only", dependencyRegistryImageAndTag, registryRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/lazycontainers/lazybox:laziest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".lazy-chart.registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("dependency image and digest, repository prefix only", dependencyRegistryImageAndTag, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "index.docker.io/my-company/lazybox:laziest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".lazy-chart.image",
				Value: "my-company/lazybox",
			},
		},
	}),
	Entry("dependency image and digest, registry and prefix", dependencyRegistryImageAndTag, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/lazycontainers/lazybox:laziest",
		RewrittenImage: "registry.vmware.com/my-company/lazybox:laziest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".lazy-chart.registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".lazy-chart.image",
				Value: "my-company/lazybox",
			},
		},
	}),

	Entry("global registry, registry only", parentWithGlobalRegistry, registryRule, &TableOutput{
		Image:          "index.docker.io/mycompany/coolapp:newest",
		RewrittenImage: "registry.vmware.com/mycompany/coolapp:newest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("global registry, repository prefix only", parentWithGlobalRegistry, repositoryPrefixRule, &TableOutput{
		Image:          "index.docker.io/mycompany/coolapp:newest",
		RewrittenImage: "index.docker.io/my-company/coolapp:newest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".subchart.image",
				Value: "my-company/coolapp",
			},
		},
	}),
	Entry("global registry, registry and prefix", parentWithGlobalRegistry, registryAndPrefixRule, &TableOutput{
		Image:          "index.docker.io/mycompany/coolapp:newest",
		RewrittenImage: "registry.vmware.com/my-company/coolapp:newest",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".subchart.image",
				Value: "my-company/coolapp",
			},
		},
	}),

	Entry("global registry with path, registry only", parentWithGlobalRegistryAndPath, registryRule, &TableOutput{
		Image:          "tenancy.registry.com/companyone/super-app:1.2.3",
		RewrittenImage: "registry.vmware.com/super-app:1.2.3",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("global registry with path, repository prefix only", parentWithGlobalRegistryAndPath, repositoryPrefixRule, &TableOutput{
		Image:          "tenancy.registry.com/companyone/super-app:1.2.3",
		RewrittenImage: "tenancy.registry.com/companyone/my-company/super-app:1.2.3",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".subchart.image",
				Value: "my-company/super-app",
			},
		},
	}),
	Entry("global registry with path, registry and prefix", parentWithGlobalRegistryAndPath, registryAndPrefixRule, &TableOutput{
		Image:          "tenancy.registry.com/companyone/super-app:1.2.3",
		RewrittenImage: "registry.vmware.com/my-company/super-app:1.2.3",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".subchart.image",
				Value: "my-company/super-app",
			},
		},
	}),

	Entry("global registry with path, registry only", parentWithGlobalRegistryAndPath, registryRule, &TableOutput{
		Image:          "tenancy.registry.com/companyone/super-app:1.2.3",
		RewrittenImage: "registry.vmware.com/super-app:1.2.3", // Is this the expected result?
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("global registry with path, repository prefix only", parentWithGlobalRegistryAndPath, repositoryPrefixRule, &TableOutput{
		Image:          "tenancy.registry.com/companyone/super-app:1.2.3",
		RewrittenImage: "tenancy.registry.com/companyone/my-company/super-app:1.2.3",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".subchart.image",
				Value: "my-company/super-app",
			},
		},
	}),
	Entry("global registry with path, registry and prefix", parentWithGlobalRegistryAndPath, registryAndPrefixRule, &TableOutput{
		Image:          "tenancy.registry.com/companyone/super-app:1.2.3",
		RewrittenImage: "registry.vmware.com/my-company/super-app:1.2.3",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
			{
				Path:  ".subchart.image",
				Value: "my-company/super-app",
			},
		},
	}),
	Entry("global registry with path, registry and same prefix", parentWithGlobalRegistryAndPath, &internal.OCIImageLocation{Registry: "registry.vmware.com", RepositoryPrefix: "companyone"}, &TableOutput{
		Image:          "tenancy.registry.com/companyone/super-app:1.2.3",
		RewrittenImage: "registry.vmware.com/super-app:1.2.3", // Is this the expected result?
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com",
			},
		},
	}),
	Entry("global registry with path, registry with same path", parentWithGlobalRegistryAndPath, &internal.OCIImageLocation{Registry: "registry.vmware.com/companyone"}, &TableOutput{
		Image:          "tenancy.registry.com/companyone/super-app:1.2.3",
		RewrittenImage: "registry.vmware.com/companyone/super-app:1.2.3",
		Actions: []*internal.RewriteAction{
			{
				Path:  ".registry",
				Value: "registry.vmware.com/companyone",
			},
		},
	}),
)
