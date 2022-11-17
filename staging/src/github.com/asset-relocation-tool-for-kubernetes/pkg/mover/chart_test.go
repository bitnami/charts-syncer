// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"helm.sh/helm/v3/pkg/chartutil"

	"helm.sh/helm/v3/pkg/chart"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/internal/internalfakes"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/pkg/mover/moverfakes"
	"github.com/vmware-tanzu/asset-relocation-tool-for-kubernetes/test"
	"helm.sh/helm/v3/pkg/chart/loader"
)

type testPrinter struct {
	out *Buffer
}

func (c *testPrinter) print(i ...interface{}) {
	_, _ = fmt.Fprint(c.out, i...)
}

func (c *testPrinter) Printf(format string, i ...interface{}) {
	c.print(fmt.Sprintf(format, i...))
}

func (c *testPrinter) Println(i ...interface{}) {
	c.print(fmt.Sprintln(i...))
}

const testRetries = 3

var testchart = test.MakeChart(&test.ChartSeed{
	Values: map[string]interface{}{
		"image": map[string]interface{}{
			"registry":   "docker.io",
			"repository": "bitnami/wordpress:1.2.3",
		},
		"secondimage": map[string]interface{}{
			"registry":   "docker.io",
			"repository": "bitnami/wordpress",
			"tag":        "1.2.3",
		},
		"observability": map[string]interface{}{
			"image": map[string]interface{}{
				"registry":   "docker.io",
				"repository": "bitnami/wavefront",
				"tag":        "5.6.7",
			},
		},
		"observabilitytoo": map[string]interface{}{
			"image": map[string]interface{}{
				"registry":   "docker.io",
				"repository": "bitnami/wavefront",
				"tag":        "5.6.7",
			},
		},
		"imagewithdigest": map[string]interface{}{
			"repository": "bitnami/wordpress@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	},
})

func newPattern(input string) *internal.ImageTemplate {
	template, err := internal.NewFromString(input)
	Expect(err).ToNot(HaveOccurred())
	return template
}

//go:generate counterfeiter github.com/google/go-containerregistry/pkg/v1.Image

func makeImage(digest string) *moverfakes.FakeImage {
	image := &moverfakes.FakeImage{}
	image.DigestReturns(v1.NewHash(digest))
	return image
}

func testChartMover(registry internal.ContainerRegistryInterface, logger Logger) *ChartMover {
	return &ChartMover{
		chart:                   testchart,
		sourceContainerRegistry: registry,
		targetContainerRegistry: registry,
		logger:                  logger,
		retries:                 testRetries,
	}
}

var _ = Describe("Pull & Push Images", func() {
	var (
		fakeRegistry *internalfakes.FakeContainerRegistryInterface
		printer      *testPrinter
	)
	BeforeEach(func() {
		fakeRegistry = &internalfakes.FakeContainerRegistryInterface{}
		printer = &testPrinter{
			out: NewBuffer(),
		}
	})

	Describe("computeChanges", func() {
		It("checks if the rewritten images are present", func() {
			changes := []*internal.ImageChange{
				{
					Pattern:        newPattern("{{.image.registry}}/{{.image.repository}}"),
					ImageReference: name.MustParseReference("index.docker.io/bitnami/wordpress:1.2.3"),
					Image:          makeImage("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					Digest:         "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
				{
					Pattern:        newPattern("{{.observability.image.registry}}/{{.observability.image.repository}}:{{.observability.image.tag}}"),
					ImageReference: name.MustParseReference("index.docker.io/bitnami/wavefront:5.6.7"),
					Image:          makeImage("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					Digest:         "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			}
			rules := &RewriteRules{
				Registry:         "harbor-repo.vmware.com",
				RepositoryPrefix: "pwall",
			}

			fakeRegistry.CheckReturnsOnCall(0, true, nil)  // Pretend it doesn't exist
			fakeRegistry.CheckReturnsOnCall(1, false, nil) // Pretend it already exists

			cm := testChartMover(fakeRegistry, printer)
			newChanges, actions, err := cm.computeChanges(changes, rules)
			Expect(err).ToNot(HaveOccurred())

			By("checking the existing images on the remote registry", func() {
				Expect(fakeRegistry.CheckCallCount()).To(Equal(2))
				digest, imageReference := fakeRegistry.CheckArgsForCall(0)
				Expect(digest).To(Equal("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				Expect(imageReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wordpress@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				digest, imageReference = fakeRegistry.CheckArgsForCall(1)
				Expect(digest).To(Equal("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				Expect(imageReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wavefront:5.6.7"))
			})

			By("updating the image change list", func() {
				Expect(newChanges).To(HaveLen(2))
				Expect(newChanges[0].Pattern).To(Equal(changes[0].Pattern))
				Expect(newChanges[0].ImageReference).To(Equal(changes[0].ImageReference))
				Expect(newChanges[0].RewrittenReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wordpress@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				Expect(newChanges[0].Digest).To(Equal(changes[0].Digest))
				Expect(newChanges[0].AlreadyPushed).To(BeFalse())

				Expect(newChanges[1].Pattern).To(Equal(changes[1].Pattern))
				Expect(newChanges[1].ImageReference).To(Equal(changes[1].ImageReference))
				Expect(newChanges[1].RewrittenReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wavefront:5.6.7"))
				Expect(newChanges[1].Digest).To(Equal(changes[1].Digest))
				Expect(newChanges[1].AlreadyPushed).To(BeTrue())
			})

			By("returning a list of changes that would need to be applied to the chart", func() {
				Expect(actions).To(HaveLen(4))
				Expect(actions).To(ContainElements([]*internal.RewriteAction{
					{
						Path:  ".image.registry",
						Value: "harbor-repo.vmware.com",
					},
					{
						Path:  ".image.repository",
						Value: "pwall/wordpress@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
					{
						Path:  ".observability.image.registry",
						Value: "harbor-repo.vmware.com",
					},
					{
						Path:  ".observability.image.repository",
						Value: "pwall/wavefront",
					},
				}))
			})
		})

		Context("the target image already exists with a different digest", func() {
			changes := []*internal.ImageChange{
				{
					Pattern:        newPattern("{{.observability.image.registry}}/{{.observability.image.repository}}:{{.observability.image.tag}}"),
					ImageReference: name.MustParseReference("index.docker.io/bitnami/wavefront:5.6.7"),
					Image:          makeImage("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					Digest:         "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
			}
			rules := &RewriteRules{
				Registry: "new-registry.io",
			}

			It("returns an error if no force push is set", func() {
				fakeRegistry.CheckReturns(false, errors.New("Image exists with different digest")) // Pretend it doesn't exist

				cm := testChartMover(fakeRegistry, printer)
				_, _, err := cm.computeChanges(changes, rules)
				Expect(err).To(HaveOccurred())
			})

			It("sets image to be pushed if forcePush is set", func() {
				fakeRegistry.CheckReturns(false, errors.New("Image exists with different digest")) // Pretend it doesn't exist

				cm := testChartMover(fakeRegistry, printer)
				rules.ForcePush = true
				newChanges, _, err := cm.computeChanges(changes, rules)
				Expect(err).ToNot(HaveOccurred())

				By("updating the image change list with the image to be pushed anyways", func() {
					Expect(newChanges).To(HaveLen(1))
					Expect(newChanges[0].Pattern).To(Equal(changes[0].Pattern))
					Expect(newChanges[0].RewrittenReference.Name()).To(Equal("new-registry.io/bitnami/wavefront:5.6.7"))
					Expect(newChanges[0].AlreadyPushed).To(BeFalse())
				})
			})
		})

		Context("two of the same image with different templates", func() {
			It("only checks one image", func() {

				changes := []*internal.ImageChange{
					{
						Pattern:        newPattern("{{.observability.image.registry}}/{{.observability.image.repository}}:{{.observability.image.tag}}"),
						ImageReference: name.MustParseReference("index.docker.io/bitnami/wavefront:5.6.7"),
						Image:          makeImage("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
						Digest:         "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
					{
						Pattern:        newPattern("{{.observabilitytoo.image.registry}}/{{.observabilitytoo.image.repository}}:{{.observabilitytoo.image.tag}}"),
						ImageReference: name.MustParseReference("index.docker.io/bitnami/wavefront:5.6.7"),
						Image:          makeImage("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
						Digest:         "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					},
				}
				rules := &RewriteRules{
					Registry:         "harbor-repo.vmware.com",
					RepositoryPrefix: "pwall",
				}

				fakeRegistry.CheckReturns(true, nil) // Pretend it doesn't exist

				cm := testChartMover(fakeRegistry, printer)
				newChanges, actions, err := cm.computeChanges(changes, rules)
				Expect(err).ToNot(HaveOccurred())

				By("checking the image once", func() {
					Expect(fakeRegistry.CheckCallCount()).To(Equal(1))
					digest, imageReference := fakeRegistry.CheckArgsForCall(0)
					Expect(digest).To(Equal("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
					Expect(imageReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wavefront:5.6.7"))
				})

				By("updating the image change list, but one is marked already pushed", func() {
					Expect(newChanges).To(HaveLen(2))
					Expect(newChanges[0].Pattern).To(Equal(changes[0].Pattern))
					Expect(newChanges[0].ImageReference).To(Equal(changes[0].ImageReference))
					Expect(newChanges[0].RewrittenReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wavefront:5.6.7"))
					Expect(newChanges[0].Digest).To(Equal(changes[0].Digest))
					Expect(newChanges[0].AlreadyPushed).To(BeFalse())

					Expect(newChanges[1].Pattern).To(Equal(changes[1].Pattern))
					Expect(newChanges[1].ImageReference).To(Equal(changes[1].ImageReference))
					Expect(newChanges[1].RewrittenReference.Name()).To(Equal("harbor-repo.vmware.com/pwall/wavefront:5.6.7"))
					Expect(newChanges[1].Digest).To(Equal(changes[1].Digest))
					Expect(newChanges[1].AlreadyPushed).To(BeTrue())
				})

				By("returning a list of changes that would need to be applied to the chart", func() {
					Expect(actions).To(HaveLen(4))
					Expect(actions).To(ContainElements([]*internal.RewriteAction{
						{
							Path:  ".observability.image.registry",
							Value: "harbor-repo.vmware.com",
						},
						{
							Path:  ".observability.image.repository",
							Value: "pwall/wavefront",
						},
						{
							Path:  ".observabilitytoo.image.registry",
							Value: "harbor-repo.vmware.com",
						},
						{
							Path:  ".observabilitytoo.image.repository",
							Value: "pwall/wavefront",
						},
					}))
				})
			})
		})
	})

	Describe("pullOriginalImages", func() {
		It("creates a change list for each image in the pattern list", func() {
			digest1 := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			image1 := makeImage(digest1)
			digest2 := "sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			image2 := makeImage(digest2)
			fakeRegistry.PullReturnsOnCall(0, image1, digest1, nil)
			fakeRegistry.PullReturnsOnCall(1, image2, digest2, nil)

			patterns := []*internal.ImageTemplate{
				newPattern("{{.image.registry}}/{{.image.repository}}"),
				newPattern("{{.observability.image.registry}}/{{.observability.image.repository}}:{{.observability.image.tag}}"),
			}

			cm := testChartMover(fakeRegistry, printer)
			changes, err := cm.loadOriginalImages(patterns)
			Expect(err).ToNot(HaveOccurred())

			By("pulling the images", func() {
				Expect(fakeRegistry.PullCallCount()).To(Equal(2))
				Expect(fakeRegistry.PullArgsForCall(0).Name()).To(Equal("index.docker.io/bitnami/wordpress:1.2.3"))
				Expect(fakeRegistry.PullArgsForCall(1).Name()).To(Equal("index.docker.io/bitnami/wavefront:5.6.7"))
			})

			By("returning a list of images", func() {
				Expect(changes).To(HaveLen(2))
				Expect(changes[0].Pattern).To(Equal(patterns[0]))
				Expect(changes[0].ImageReference.Name()).To(Equal("index.docker.io/bitnami/wordpress:1.2.3"))
				Expect(changes[0].Digest).To(Equal("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				Expect(changes[0].Tag).To(Equal("1.2.3"))
				Expect(changes[1].Pattern).To(Equal(patterns[1]))
				Expect(changes[1].ImageReference.Name()).To(Equal("index.docker.io/bitnami/wavefront:5.6.7"))
				Expect(changes[1].Digest).To(Equal("sha256:1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				Expect(changes[1].Tag).To(Equal("5.6.7"))
			})
		})

		Context("no tag set in chart", func() {
			It("assumes the latest tag", func() {
				digest1 := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				image1 := makeImage(digest1)
				fakeRegistry.PullReturnsOnCall(0, image1, digest1, nil)

				patterns := []*internal.ImageTemplate{
					newPattern("{{.secondimage.registry}}/{{.secondimage.repository}}"),
				}

				cm := testChartMover(fakeRegistry, printer)
				changes, err := cm.loadOriginalImages(patterns)
				Expect(err).ToNot(HaveOccurred())

				By("pulling the image with the latest tag", func() {
					Expect(fakeRegistry.PullCallCount()).To(Equal(1))
					Expect(fakeRegistry.PullArgsForCall(0).Name()).To(Equal("index.docker.io/bitnami/wordpress:latest"))
				})

				By("returning the image", func() {
					Expect(changes).To(HaveLen(1))
					Expect(changes[0].Pattern).To(Equal(patterns[0]))
					Expect(changes[0].ImageReference.Name()).To(Equal("index.docker.io/bitnami/wordpress:latest"))
					Expect(changes[0].Digest).To(Equal("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
					Expect(changes[0].Tag).To(Equal("latest"))
				})
			})
		})

		Context("image has no tag (digest is set)", func() {
			It("does not assumed the latest tag", func() {
				digest1 := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				image1 := makeImage(digest1)
				fakeRegistry.PullReturnsOnCall(0, image1, digest1, nil)

				patterns := []*internal.ImageTemplate{
					newPattern("{{.imagewithdigest.repository}}"),
				}

				cm := testChartMover(fakeRegistry, printer)
				changes, err := cm.loadOriginalImages(patterns)
				Expect(err).ToNot(HaveOccurred())

				By("pulling the image with the latest tag", func() {
					Expect(fakeRegistry.PullCallCount()).To(Equal(1))
					Expect(fakeRegistry.PullArgsForCall(0).Name()).To(Equal("index.docker.io/bitnami/wordpress@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
				})

				By("returning the image", func() {
					Expect(changes).To(HaveLen(1))
					Expect(changes[0].Pattern).To(Equal(patterns[0]))
					Expect(changes[0].ImageReference.Name()).To(Equal("index.docker.io/bitnami/wordpress@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
					Expect(changes[0].Digest).To(Equal("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
					Expect(changes[0].Tag).To(Equal(""))
				})
			})
		})

		Context("duplicated image", func() {
			It("only pulls once", func() {
				digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				image := makeImage(digest)
				fakeRegistry.PullReturns(image, digest, nil)

				patterns := []*internal.ImageTemplate{
					newPattern("{{.image.registry}}/{{.image.repository}}"),
					newPattern("{{.secondimage.registry}}/{{.secondimage.repository}}:{{.secondimage.tag}}"),
				}

				cm := testChartMover(fakeRegistry, printer)
				changes, err := cm.loadOriginalImages(patterns)
				Expect(err).ToNot(HaveOccurred())

				By("pulling the image once", func() {
					Expect(fakeRegistry.PullCallCount()).To(Equal(1))
					Expect(fakeRegistry.PullArgsForCall(0).Name()).To(Equal("index.docker.io/bitnami/wordpress:1.2.3"))
				})

				By("returning a list of images", func() {
					Expect(changes).To(HaveLen(2))
					Expect(changes[0].Pattern).To(Equal(patterns[0]))
					Expect(changes[0].ImageReference.Name()).To(Equal("index.docker.io/bitnami/wordpress:1.2.3"))
					Expect(changes[0].Digest).To(Equal("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
					Expect(changes[0].Tag).To(Equal("1.2.3"))
					Expect(changes[1].Pattern).To(Equal(patterns[1]))
					Expect(changes[1].ImageReference.Name()).To(Equal("index.docker.io/bitnami/wordpress:1.2.3"))
					Expect(changes[1].Digest).To(Equal("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
					Expect(changes[1].Tag).To(Equal("1.2.3"))
				})
			})
		})

		Context("error pulling an image", func() {
			It("returns the error", func() {
				fakeRegistry.PullReturns(nil, "", fmt.Errorf("image pull error"))
				patterns := []*internal.ImageTemplate{
					newPattern("{{.image.registry}}/{{.image.repository}}"),
				}

				cm := testChartMover(fakeRegistry, printer)
				_, err := cm.loadOriginalImages(patterns)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("failed to pull original images: image pull error"))
			})
		})
	})

	Describe("PushRewrittenImages", func() {
		var images []*internal.ImageChange
		BeforeEach(func() {
			images = []*internal.ImageChange{
				{
					ImageReference:     name.MustParseReference("acme/busybox:1.2.3"),
					RewrittenReference: name.MustParseReference("harbor-repo.vmware.com/pwall/busybox@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					Image:              makeImage("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
					Digest:             "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					Tag:                "1.2.3",
				},
			}
		})

		It("pushes the images", func() {
			cm := testChartMover(fakeRegistry, printer)
			err := cm.pushRewrittenImages(images)
			Expect(err).ToNot(HaveOccurred())

			By("pushing the image", func() {
				Expect(fakeRegistry.PushCallCount()).To(Equal(1))
				image, ref := fakeRegistry.PushArgsForCall(0)
				Expect(image).To(Equal(images[0].Image))
				Expect(ref.Name()).To(Equal("harbor-repo.vmware.com/pwall/busybox:1.2.3"))
			})

			By("logging the process", func() {
				Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\nDone"))
			})
		})

		Context("tag is not known", func() {
			It("pushes the digest image", func() {
				images[0].Tag = ""

				cm := testChartMover(fakeRegistry, printer)
				err := cm.pushRewrittenImages(images)
				Expect(err).ToNot(HaveOccurred())

				By("pushing the image", func() {
					Expect(fakeRegistry.PushCallCount()).To(Equal(1))
					image, ref := fakeRegistry.PushArgsForCall(0)
					Expect(image).To(Equal(images[0].Image))
					Expect(ref).To(Equal(images[0].RewrittenReference))
				})

				By("logging the process", func() {
					Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...\nDone"))
				})
			})
		})

		Context("rewritten image is the same", func() {
			It("does not push the image", func() {
				images[0].RewrittenReference = images[0].ImageReference

				cm := testChartMover(fakeRegistry, printer)
				err := cm.pushRewrittenImages(images)
				Expect(err).ToNot(HaveOccurred())

				By("not pushing the image", func() {
					Expect(fakeRegistry.PushCallCount()).To(Equal(0))
				})
			})
		})

		Context("image has already been pushed", func() {
			It("does not push the image", func() {
				images[0].AlreadyPushed = true
				cm := testChartMover(fakeRegistry, printer)
				err := cm.pushRewrittenImages(images)
				Expect(err).ToNot(HaveOccurred())

				By("not pushing the image", func() {
					Expect(fakeRegistry.PushCallCount()).To(Equal(0))
				})
			})
		})

		Context("pushing fails once", func() {
			BeforeEach(func() {
				fakeRegistry.PushReturnsOnCall(0, fmt.Errorf("push failed"))
				fakeRegistry.PushReturnsOnCall(1, nil)
			})

			It("retries and passes", func() {
				cm := testChartMover(fakeRegistry, printer)
				err := cm.pushRewrittenImages(images)
				Expect(err).ToNot(HaveOccurred())

				By("trying to push the image twice", func() {
					Expect(fakeRegistry.PushCallCount()).To(Equal(2))
				})

				By("logging the process", func() {
					Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\n"))
					Expect(printer.out).To(Say("Attempt #1 failed: push failed"))
					Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\nDone"))
				})
			})
		})

		Context("pushing fails every time", func() {
			BeforeEach(func() {
				fakeRegistry.PushReturns(fmt.Errorf("push failed"))
			})

			It("returns an error", func() {
				cm := testChartMover(fakeRegistry, printer)
				err := cm.pushRewrittenImages(images)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("All attempts fail:\n#1: push failed\n#2: push failed\n#3: push failed"))

				By("trying to push the image", func() {
					Expect(fakeRegistry.PushCallCount()).To(Equal(3))
				})

				By("logging the process", func() {
					Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\n"))
					Expect(printer.out).To(Say("Attempt #1 failed: push failed"))
					Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\n"))
					Expect(printer.out).To(Say("Attempt #2 failed: push failed"))
					Expect(printer.out).To(Say("Pushing harbor-repo.vmware.com/pwall/busybox:1.2.3...\n"))
					Expect(printer.out).To(Say("Attempt #3 failed: push failed"))
				})
			})
		})
	})

	Describe("targetOutput", func() {
		It("works with default out flag", func() {
			outFmt := "/path/%s-%s.relocated.tgz"
			target := targetOutput(outFmt, "my-chart", "0.1")
			Expect(target).To(Equal("/path/my-chart-0.1.relocated.tgz"))
		})
		It("builds custom out input as expected", func() {
			target := targetOutput("/path/%s-%s-wildcardhere.tgz", "my-chart", "0.1")
			Expect(target).To(Equal("/path/my-chart-0.1-wildcardhere.tgz"))
		})
	})
})

const (
	fixturesRoot = "../../test/fixtures/"
)

type FakeLogger struct {
	Output *Buffer
}

func (l *FakeLogger) Printf(format string, i ...interface{}) {
	_, _ = fmt.Fprintf(l.Output, format, i...)
}

func (l *FakeLogger) Println(i ...interface{}) {
	_, _ = fmt.Fprintln(l.Output, i...)
}

var _ = Describe("LoadImagePatterns", func() {
	var logger *FakeLogger

	BeforeEach(func() {
		logger = &FakeLogger{
			Output: NewBuffer(),
		}
	})

	It("reads from given file first if present", func() {
		imagefile := filepath.Join(fixturesRoot, "testchart.images.yaml")
		contents, err := loadImageHints(imagefile, nil, logger)
		Expect(err).ToNot(HaveOccurred())

		expected, err := os.ReadFile(imagefile)
		Expect(err).ToNot(HaveOccurred())
		Expect(contents).To(Equal(expected))
	})
	It("reads from chart if file missing", func() {
		chart, err := loader.Load(filepath.Join(fixturesRoot, "self-relok8ing-chart"))
		Expect(err).ToNot(HaveOccurred())

		contents, err := loadImageHints("", chart, logger)
		Expect(err).ToNot(HaveOccurred())

		embeddedPatterns := filepath.Join(fixturesRoot, "self-relok8ing-chart/.relok8s-images.yaml")
		expected, err := os.ReadFile(embeddedPatterns)
		Expect(err).ToNot(HaveOccurred())
		Expect(contents).To(Equal(expected))

		Expect(logger.Output).Should(Say(".relok8s-images.yaml hints file found"))
	})
	It("reads nothing when no file and the chart is not self relok8able", func() {
		chart, err := loader.Load(filepath.Join(fixturesRoot, "testchart"))
		Expect(err).ToNot(HaveOccurred())

		contents, err := loadImageHints("", chart, logger)
		Expect(err).ToNot(HaveOccurred())
		Expect(contents).To(BeEmpty())
	})
})

func stringsContains(strings []string, s string) bool {
	for _, str := range strings {
		if s == str {
			return true
		}
	}
	return false
}

var _ = Describe("loadChartFromPath", func() {
	It("loads chart directory directly", func() {
		cm := &ChartMover{}
		err := cm.loadChartFromPath(filepath.Join(fixturesRoot, "testchart"))
		Expect(err).ToNot(HaveOccurred())
	})
	It("loads chart tarball unpacking it and consumes last of any duplicate", func() {
		cm := &ChartMover{}
		err := cm.loadChartFromPath(filepath.Join(fixturesRoot, "testchart-with-duplicates.tgz"))
		Expect(err).ToNot(HaveOccurred())
		filenames := []string{}
		chartYamlFound := false
		for _, file := range cm.chart.Raw {
			Expect(stringsContains(filenames, file.Name)).ToNot(BeTrue())
			filenames = append(filenames, file.Name)
			if file.Name == "Chart.yaml" {
				chartYamlFound = true
				Expect(string(file.Data)).To(ContainSubstring("A chart for testing relok8s modified"))
			}
		}
		Expect(chartYamlFound).To(BeTrue())
	})
})

func TestNamespacedPath(t *testing.T) {
	tests := []struct {
		inputPath  string
		chartName  string
		outputPath string
	}{
		{".image.registry", "app1", ".image.registry"},
		{".app1.image.registry", "app1", ".image.registry"},
		{".fooapp1.image.registry", "app1", ".fooapp1.image.registry"},
		{".image.app1.registry", "app1", ".image.app1.registry"},
		{".app2.image.registry", "app1", ".app2.image.registry"},
	}

	for _, tc := range tests {
		if got, want := namespacedPath(tc.inputPath, tc.chartName), tc.outputPath; got != want {
			t.Errorf("got=%s; want=%s", got, want)
		}
	}
}

func TestGroupChangesByChart(t *testing.T) {
	rootChart, err := loader.Load(filepath.Join(fixturesRoot, "3-levels-chart"))
	if err != nil {
		t.Fatal(err)
	}

	// Rewrites that affect 3 chart levels, parent -> subchart -> subchart
	r1 := &internal.RewriteAction{Path: ".image"}
	subchart1R1 := &internal.RewriteAction{Path: ".subchart-1.image"}
	subchart1R2 := &internal.RewriteAction{Path: ".subchart-1.image2"}
	subchart2R1 := &internal.RewriteAction{Path: ".subchart-2.image"}
	subchart1Subchart3 := &internal.RewriteAction{Path: ".subchart-1.subchart-3.image"}
	rewrites := []*internal.RewriteAction{r1, subchart1R1, subchart1R2, subchart1Subchart3, subchart2R1}

	// Expected output
	want := make(map[*chart.Chart][]*internal.RewriteAction)
	// parent chart
	want[rootChart] = []*internal.RewriteAction{r1}

	firstLevelDeps := rootChart.Dependencies()
	// Sort dependencies since they come in arbitrary order
	sort.Slice(firstLevelDeps, func(i, j int) bool {
		return firstLevelDeps[i].Name() < firstLevelDeps[j].Name()
	})

	// Subchart1
	want[firstLevelDeps[0]] = []*internal.RewriteAction{subchart1R1, subchart1R2}

	// Subchart2
	want[firstLevelDeps[1]] = []*internal.RewriteAction{subchart2R1}

	// Subchart1.Subchart3
	want[firstLevelDeps[0].Dependencies()[0]] = []*internal.RewriteAction{subchart1Subchart3}

	// Compare output
	if got := groupChangesByChart(rewrites, rootChart); !reflect.DeepEqual(got, want) {
		t.Errorf("got=%v; want=%v", got, want)
	}
}

// Check that a relocated Helm Chart does not contain information about their dependencies
func TestStripDependencyRefs(t *testing.T) {
	testChart, err := loader.Load(filepath.Join(fixturesRoot, "3-levels-chart"))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(testChart.Metadata.Dependencies), 2; got != want {
		t.Errorf("invalid number of dependencies, got=%d, want=%d", got, want)
	}

	if testChart.Lock == nil {
		t.Error("Chart.lock file expected ")
	}

	firstLevelDeps := testChart.Dependencies()
	// Sort dependencies since they come in arbitrary order
	sortCharts(firstLevelDeps)

	// Subchart1
	if got, want := len(firstLevelDeps[0].Metadata.Dependencies), 1; got != want {
		t.Errorf("invalid number of dependencies, got=%d, want=%d", got, want)
	}

	if firstLevelDeps[0].Lock == nil {
		t.Error("Chart.lock file expected ")
	}

	// Strip dependencies and re-package chart
	if err := stripDependencyRefs(testChart); err != nil {
		t.Fatal(err)
	}

	tmpDir, err := os.MkdirTemp("", "external-tests-*")
	if err != nil {
		t.Fatal(err)
	}

	// Repackage chart
	filename, err := chartutil.Save(testChart, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Load from re-packaged version
	modifiedChart, err := loader.Load(filename)
	if err != nil {
		t.Fatal(err)
	}

	firstLevelDeps = modifiedChart.Dependencies()
	sortCharts(firstLevelDeps)

	for _, c := range []*chart.Chart{modifiedChart, firstLevelDeps[0]} {
		if got := len(c.Metadata.Dependencies); got != 0 {
			t.Errorf("expected no dependencies got=%d", got)
		}

		if c.Lock != nil {
			t.Error("Chart.lock file unexpected ")
		}
	}
}

func sortCharts(charts []*chart.Chart) {
	sort.Slice(charts, func(i, j int) bool {
		return charts[i].Name() < charts[j].Name()
	})
}
