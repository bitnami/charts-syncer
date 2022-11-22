// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package mover

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
)

func TestGetContainersKeychain(t *testing.T) {
	explicitCreds := &OCICredentials{Username: "user", Password: "pass", Server: "server.io"}

	tests := []struct {
		name      string
		auth      *ContainersAuth
		want      authn.Keychain
		wantError bool // if the function should return an error
	}{
		// Valid triplets
		{name: "neither local nor provided credentials", auth: &ContainersAuth{}, wantError: true},
		{name: "both local and provided credentials",
			auth: &ContainersAuth{
				UseDefaultLocalKeychain: true,
				Credentials:             explicitCreds},
			wantError: true},
		{name: "localKeychain", auth: &ContainersAuth{UseDefaultLocalKeychain: true}, want: authn.DefaultKeychain},
		{
			name: "valid explicit creds",
			auth: &ContainersAuth{
				Credentials: explicitCreds,
			},
			want: explicitCreds,
		},
		{name: "invalid provided credentials",
			auth: &ContainersAuth{
				Credentials: &OCICredentials{Username: "user", Password: "pass", Server: "https://server.io"}},
			wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, gotError := getContainersKeychain(test.auth)
			if gotError == nil == test.wantError {
				t.Errorf("expected error %t, got error %q. auth=%v", test.wantError, gotError, test.auth)
			}

			if got != test.want {
				t.Errorf("got=%v, want=%v", got, test.want)
			}
		})
	}
}

func TestValidateOCICredentials(t *testing.T) {
	var tests = []struct {
		username  string
		password  string
		server    string
		wantError bool // if the function should return an error
	}{
		// Valid triplets
		{"username", "password", "server.io", false},
		{"username", "password", "server.io:9999", false},

		// Missing one of the three items
		{"", "password", "server.io", true},
		{"username", "", "server.io", true},
		{"username", "password", "", true},

		// Invalid serverName
		{"username", "password", "http://server.io", true},
		{"username", "password", "//server.io", true},
		{"username", "password", "server.io/baz", true},
	}

	for _, test := range tests {
		_, gotError := validateOCICredentials(&OCICredentials{test.server, test.username, test.password})
		if gotError == nil == test.wantError {
			t.Errorf("expected error %t, got error %q. username=%q, pass=%q, server=%q", test.wantError, gotError, test.username, test.password, test.server)
		}
	}
}
