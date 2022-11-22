// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/google/go-containerregistry/pkg/authn"
)

// Resolve implements an authn.KeyChain
//
// See https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#Keychain
//
// Returns a custom credentials authn.Authenticator if the given resource
// RegistryStr() matches the Repository, otherwise it returns annonymous access
func (repo *OCICredentials) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	if repo.Server == resource.RegistryStr() {
		return repo, nil
	}

	// if no credentials are provided we return annon authentication
	return authn.Anonymous, nil
}

// Authorization implements an authn.Authenticator
//
// See https://pkg.go.dev/github.com/google/go-containerregistry/pkg/authn#Authenticator
//
// Returns an authn.AuthConfig with a user / password pair to be used for authentication
func (repo *OCICredentials) Authorization() (*authn.AuthConfig, error) {
	return &authn.AuthConfig{Username: repo.Username, Password: repo.Password}, nil
}

// Define a container registry keychain based on the settings provided in containers Auth
// If useDefaultKeychain is set, use config/docker.json otherwise it will load the provided credentials (if any)
func getContainersKeychain(c *ContainersAuth) (authn.Keychain, error) {
	// No credentials provided
	if !c.UseDefaultLocalKeychain && c.Credentials == nil {
		return nil, errors.New("either local keychain or explicit credentials are required")
	}

	if c.UseDefaultLocalKeychain && c.Credentials != nil {
		return nil, errors.New("you can use either local keychain or explicit credentials not both")
	}

	if c.UseDefaultLocalKeychain {
		return authn.DefaultKeychain, nil
	}

	return validateOCICredentials(c.Credentials)
}

// validate if the provided OCI credentials are valid
// They include a username, password and a valid (RFC 3986 URI authority) serverName
func validateOCICredentials(c *OCICredentials) (authn.Keychain, error) {
	if c.Username == "" || c.Password == "" || c.Server == "" {
		return nil, errors.New("OCI credentials require an username, password and a server name")
	}

	// See https://github.com/google/go-containerregistry/blob/main/pkg/name/registry.go#L97
	// Valid RFC 3986 URI authority
	if uri, err := url.Parse("//" + c.Server); err != nil || uri.Host != c.Server {
		return nil, fmt.Errorf("credentials server name %q must not contain a scheme (\"http://\") nor a path. Example of valid registry names are \"myregistry.io\" or \"myregistry.io:9999\"", c.Server)
	}

	return c, nil
}
