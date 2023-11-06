package chart

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/bitnami/charts-syncer/api"
)

var (
	source = &api.Source{
		Spec: &api.Source_Repo{
			Repo: &api.Repo{
				Url:  "https://charts.bitnami.com/bitnami",
				Kind: api.Kind_HELM,
			},
		},
	}
	target = &api.Target{
		Spec: &api.Target_Repo{
			Repo: &api.Repo{
				Url:  "http://fake.target.com",
				Kind: api.Kind_CHARTMUSEUM,
				Auth: &api.Auth{
					Username: "user",
					Password: "password",
				},
			},
		},
		ContainerRegistry:   "test.registry.io",
		ContainerRepository: "test/repo",
	}
)

func TestUpdateValuesFile(t *testing.T) {
	originalValues := `##
## Simplified values yaml file to test registry and repository substitutions
##
global:
  imageRegistry: ""
image:
  registry: new.registry.io
  repository: test/repo/new/zookeeper
  tag: 3.5.7-r7
volumePermissions:
  enabled: false
  image:
    registry: new.registry.io
    repository: repo/new/custom-base-image
    tag: r0`
	want := `##
## Simplified values yaml file to test registry and repository substitutions
##
global:
  imageRegistry: ""
image:
  registry: test.registry.io
  repository: test/repo/zookeeper
  tag: 3.5.7-r7
volumePermissions:
  enabled: false
  image:
    registry: test.registry.io
    repository: test/repo/custom-base-image
    tag: r0`
	// Create temporary working directory
	testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
	if err != nil {
		t.Fatalf("error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)
	destValuesFilePath := path.Join(testTmpDir, ValuesFilename)

	// Write file
	err = ioutil.WriteFile(destValuesFilePath, []byte(originalValues), 0644)
	if err != nil {
		t.Fatalf("error writting destination file")
	}

	updateValuesFile(destValuesFilePath, target)
	valuesFile, err := ioutil.ReadFile(destValuesFilePath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(valuesFile)
	if want != got {
		t.Errorf("incorrect modification, got: \n %s \n, want: \n %s \n", got, want)
	}
}

func TestUpdateReadmeFile(t *testing.T) {
	originalValues := `
# Ghost
[Ghost](https://ghost.org/) is one of the most versatile open source content management systems on the market.

## TL;DR;
$ helm repo add bitnami https://charts.bitnami.com/bitnami
$ helm install my-release bitnami/ghost
...
The above parameters map to the env variables defined in [bitnami/ghost](http://github.com/bitnami/bitnami-docker-ghost).
	`
	want := `
# Ghost
[Ghost](https://ghost.org/) is one of the most versatile open source content management systems on the market.

## TL;DR;
$ helm repo add mytestrepo https://my-new-chart-repo.com
$ helm install my-release mytestrepo/ghost
...
The above parameters map to the env variables defined in [bitnami/ghost](http://github.com/bitnami/bitnami-docker-ghost).
	`
	// Create temporary working directory
	testTmpDir, err := ioutil.TempDir("", "charts-syncer-tests")
	if err != nil {
		t.Fatalf("error creating temporary: %s", testTmpDir)
	}
	defer os.RemoveAll(testTmpDir)
	destValuesFilePath := path.Join(testTmpDir, ReadmeFilename)

	// Write file
	err = ioutil.WriteFile(destValuesFilePath, []byte(originalValues), 0644)
	if err != nil {
		t.Fatalf("error writting destination file")
	}

	updateReadmeFile(destValuesFilePath, "https://charts.bitnami.com/bitnami", "https://my-new-chart-repo.com", "ghost", "mytestrepo")
	readmeFile, err := ioutil.ReadFile(destValuesFilePath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(readmeFile)
	if want != got {
		t.Errorf("incorrect modification, got: \n %s \n, want: \n %s \n", got, want)
	}
}
