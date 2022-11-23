package harbor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/internal/utils"
	"github.com/juju/errors"
)

// Container allows to operate a chart repository.
type Container struct {
	url      *url.URL
	username string
	password string
	insecure bool
}

// New creates a Repo object from an api.Repo object.
func New(registry string, containers *api.Containers, insecure bool) (*Container, error) {
	auth := containers.GetAuth()
	if auth == nil {
		registry = auth.GetRegistry()
	}

	u := url.URL{Host: registry}
	resp, err := http.Get(getPingURL("https", registry))
	if err == nil && resp.StatusCode == http.StatusOK {
		u.Scheme = "https"
	} else {
		resp, err := http.Get(getPingURL("http", registry))
		if err != nil {
			return nil, errors.Trace(err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, errors.Trace(fmt.Errorf("failed to ping harbor %s registry", registry))
		}
		u.Scheme = "http"
	}

	c := Container{url: &u, insecure: insecure}
	if auth != nil {
		c.username = auth.GetUsername()
		c.password = auth.GetPassword()
	}
	return &c, nil
}

// GetPingURL  returns the URL to upload a chart
func getPingURL(scheme, host string) string {
	u := url.URL{}
	u.Scheme = scheme
	u.Host = host
	u.Path = "api/v2.0/ping"
	return u.String()
}

// GetRepositoryURL returns the URL to upload a chart
func (c *Container) GetRepositoryURL() string {
	u := *c.url
	u.Path = "api/v2.0/projects"
	return u.String()
}

func (c *Container) CreateRepository(repository string) error {
	target := strings.Split(repository, "/")
	if len(target) < 2 {
		return nil
	}
	repository = target[1]

	repo := struct {
		ProjectName string `json:"project_name"`
		Public      bool   `json:"public"`
	}{
		ProjectName: repository,
		Public:      true,
	}

	data, err := json.Marshal(&repo)
	if err != nil {
		return err
	}

	body := &bytes.Buffer{}
	body.Write(data)

	u := c.GetRepositoryURL()
	req, err := http.NewRequest("POST", u, body)
	if err != nil {
		return errors.Trace(err)
	}

	req.Header.Add("content-type", "application/json")
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	client := utils.DefaultClient
	if c.insecure {
		client = utils.InsecureClient
	}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotatef(err, "create %q repository", repository)
	}
	defer res.Body.Close()
	if err := (res.StatusCode < 200 || res.StatusCode > 299) && res.StatusCode != 409; err {
		bodyStr := utils.HTTPResponseBody(res)
		return errors.Errorf("unable to create %q repository, got HTTP Status: %s, Resp: %v", repository, res.Status, bodyStr)
	}
	return nil
}
