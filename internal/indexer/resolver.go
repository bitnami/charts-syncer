package indexer

import (
	"crypto/tls"
	"net/http"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// newRemoteRepository creates a remote.Repository for an OCI registry with basic auth
func newRemoteRepository(ref, username, password string, insecure bool, usePlainHTTP bool) (*remote.Repository, error) {
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, err
	}

	if usePlainHTTP {
		repo.PlainHTTP = true
	}

	httpClient := retry.DefaultClient
	if insecure {
		httpClient = &http.Client{
			Transport: retry.NewTransport(&http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
			}),
		}
	}

	repo.Client = &auth.Client{
		Client: httpClient,
		Cache:  auth.NewCache(),
		Credential: auth.StaticCredential(repo.Reference.Registry, auth.Credential{
			Username: username,
			Password: password,
		}),
	}

	return repo, nil
}
