// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// RewriteRules indicate What kind of target registry overrides we want to apply to the found images
type RewriteRules struct {
	// Registry overrides the registry part of the image FQDN, i.e myregistry.io
	Registry string
	// PrefixRegistry add prefix of the registry
	PrefixRegistry string
	// RepositoryPrefix will override the image path by being prepended before the image name
	RepositoryPrefix string
	// Push the image even if there is already an image with a different digest
	ForcePush bool
}

func (r *RewriteRules) Validate() error {
	if r.Registry != "" {
		if strings.Contains(r.Registry, "/") {
			_, err := name.NewRepository(r.Registry, name.StrictValidation)
			if err != nil {
				return fmt.Errorf("registry rule is not valid: %w", err)
			}
		} else {
			_, err := name.NewRegistry(r.Registry, name.StrictValidation)
			if err != nil {
				return fmt.Errorf("registry rule is not valid: %w", err)
			}
		}
	}

	if r.RepositoryPrefix != "" {
		_, err := name.NewRepository(r.RepositoryPrefix)
		if err != nil {
			return fmt.Errorf("repository prefix rule is not valid: %w", err)
		}
	}

	return nil
}
