// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package internal

import (
	"encoding/json"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

// CanonicalDockerImage is an image which imageManifest has been modified (indented)
// to follow the same format used by docker during push [1] and hence
// preserve the digest once crafted and pushed from a tarball
// [1] https://github.com/docker/cli/blob/a32cd16160f1b41c1c4ae7bee4dac929d1484e59/cli/command/manifest/push.go#L230
// It reimplements v1.Image https://github.com/google/go-containerregistry/blob/main/pkg/v1/image.go#L22
type CanonicalDockerImage struct {
	v1.Image
}

// NewCanonicalDockerImage returns an instance of the the image which imagemanifest comes indented
func NewCanonicalDockerImage(image v1.Image) *CanonicalDockerImage {
	return &CanonicalDockerImage{Image: image}
}

func (img *CanonicalDockerImage) RawManifest() ([]byte, error) {
	originalManifest, err := img.Image.Manifest()
	if err != nil {
		return nil, err
	}

	// Indent the raw manifest similarly to what Docker push does [1]
	return json.MarshalIndent(originalManifest, "", "   ")
}

func (img *CanonicalDockerImage) Digest() (v1.Hash, error) {
	return partial.Digest(img)
}
