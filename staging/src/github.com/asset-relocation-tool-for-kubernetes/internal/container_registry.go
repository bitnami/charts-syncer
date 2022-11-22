// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package internal

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

//go:generate counterfeiter . ContainerRegistryInterface
type ContainerRegistryInterface interface {
	Check(digest string, imageReference name.Reference) (bool, error)
	Pull(imageReference name.Reference) (v1.Image, string, error)
	Push(image v1.Image, dest name.Reference) error
}

type ContainerRegistryClient struct {
	auth authn.Keychain
}

func NewContainerRegistryClient(auth authn.Keychain) *ContainerRegistryClient {
	return &ContainerRegistryClient{auth: auth}
}

func (i *ContainerRegistryClient) Pull(imageReference name.Reference) (v1.Image, string, error) {
	image, err := remote.Image(imageReference, remote.WithAuthFromKeychain(i.auth))
	if err != nil {
		return nil, "", fmt.Errorf("failed to pull image %s: %w", imageReference.Name(), err)
	}

	digest, err := image.Digest()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get image digest for %s: %w", imageReference.Name(), err)
	}

	return image, digest.String(), nil
}

func (i *ContainerRegistryClient) Check(digest string, imageReference name.Reference) (bool, error) {
	_, remoteDigest, err := i.Pull(imageReference)

	if err != nil {
		// Return true if failed to pull the image.
		// We see different errors if the image does not exist, or if the specific tag does not exist
		// It is simpler to attempt to push, which will catch legitimate issues (lack of authorization),
		// than it is to try and handle every error case here.
		return true, nil
	}

	if remoteDigest != digest {
		return false, fmt.Errorf("image %s already exists with a different digest "+
			"(local: %s remote: %s). Will not overwrite", imageReference.Name(), digest, remoteDigest)
	}

	return false, nil
}

func (i *ContainerRegistryClient) Push(image v1.Image, dest name.Reference) error {
	err := remote.Write(dest, image, remote.WithAuthFromKeychain(i.auth))
	if err != nil {
		return fmt.Errorf("failed to push image %s: %w", dest.Name(), err)
	}

	return nil
}
